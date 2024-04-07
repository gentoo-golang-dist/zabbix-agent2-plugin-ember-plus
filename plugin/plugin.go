package plugin

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"git.zabbix.com/ap/ember-plus/ember"
	"git.zabbix.com/ap/ember-plus/ember/asn1"
	"git.zabbix.com/ap/ember-plus/ember/s101"
	"git.zabbix.com/ap/ember-plus/plugin/conn"
	"git.zabbix.com/ap/ember-plus/plugin/params"
	"git.zabbix.com/ap/plugin-support/errs"
	"git.zabbix.com/ap/plugin-support/metric"
	"git.zabbix.com/ap/plugin-support/plugin"
	"git.zabbix.com/ap/plugin-support/plugin/container"
	"git.zabbix.com/ap/plugin-support/zbxerr"
)

const (
	// Name is the name of the plugin.
	Name = "EmberPlus"

	get = emberMetricKey("ember.get")
)

var (
	_ plugin.Configurator = (*emberPlugin)(nil)
	_ plugin.Exporter     = (*emberPlugin)(nil)
	_ plugin.Runner       = (*emberPlugin)(nil)
	_ handlerFunc         = (*emberPlugin)(nil).GetEmber
)

// HandlerFunc describes the signature all metric handler functions must have.
type handlerFunc func(metricParams map[string]string, extraParams ...string) (any, error)

type emberMetricKey string

type emberMetric struct {
	metric  *metric.Metric
	handler handlerFunc
}

type emberPlugin struct {
	plugin.Base
	conns   *conn.ConnCollection
	config  *pluginConfig
	metrics map[emberMetricKey]*emberMetric
}

// Plugin holds require plugin data.
type Plugin struct {
	plugin.Base
}

// Launch starts the plugin.
func Launch() error {
	p := &emberPlugin{
		conns: &conn.ConnCollection{},
	}

	err := p.registerMetrics()
	if err != nil {
		return err
	}

	h, err := container.NewHandler(Name)
	if err != nil {
		return errs.Wrap(err, "failed to create new handler")
	}

	p.Logger = &h

	err = h.Execute()
	if err != nil {
		return errs.Wrap(err, "failed to execute plugin handler")
	}

	return nil
}

// Start initiates the connection handler.
func (p *emberPlugin) Start() {
	p.conns.Init(p.config.KeepAlive, p.config.Timeout, p)
}

// Stop stops the mssql plugin, closing all the connections.
func (p *emberPlugin) Stop() {
	p.conns.CloseAll()
}

// Export collects all the metrics.
func (p *emberPlugin) Export(key string, rawParams []string, _ plugin.ContextProvider) (any, error) {
	m, ok := p.metrics[emberMetricKey(key)]
	if !ok {
		return nil, errs.Wrapf(zbxerr.ErrorUnsupportedMetric, "unknown metric %q", key)
	}

	metricParams, extraParams, hardcodedParams, err := m.metric.EvalParams(rawParams, p.config.Sessions)
	if err != nil {
		return nil, errs.Wrap(err, "failed to evaluate metric parameters")
	}

	err = metric.SetDefaults(metricParams, hardcodedParams, p.config.Default)
	if err != nil {
		return nil, errs.Wrap(err, "failed to set default params")
	}

	res, err := m.handler(metricParams, extraParams...)
	if err != nil {
		return nil, errs.Wrap(err, "failed to execute handler")
	}

	return res, nil
}

// GetEmber handles ember.get metric, returns collection data based on request metrics, response needs to be handled,
// otherwise it is not possible to json marshal it.
func (p *emberPlugin) GetEmber(metricParams map[string]string, _ ...string) (any, error) {
	connConf, err := conn.NewConnConfig(metricParams[params.URI.Name()])
	if err != nil {
		return nil, errs.Wrap(err, "failed to create connection config")
	}

	rootCollection, err := p.handleRequest("", "", connConf, ember.GetRootRequest)
	if err != nil {
		return nil, errs.Wrap(err, "failed to retrieve root collection")
	}

	path := metricParams[params.Path.Name()]
	if path == "" {
		return rootCollection, nil
	}

	pathPart, byID, err := parsePathString(path)
	if err != nil {
		return nil, errs.Wrapf(err, "failed to parse input path '%s'", path)
	}

	if byID {
		return p.getCollectionByID(rootCollection, connConf, pathPart)
	}

	return p.getCollectionByPath(rootCollection, connConf, pathPart)
}

func (p *emberPlugin) registerMetrics() error {
	p.metrics = map[emberMetricKey]*emberMetric{
		get: {
			metric: metric.New(
				"Returns the ember data based on path.",
				[]*metric.Param{params.URI, params.Path},
				false,
			),
			handler: withJSONResponse(p.GetEmber),
		},
	}

	metricSet := metric.MetricSet{}

	for k, m := range p.metrics {
		metricSet[string(k)] = m.metric
	}

	err := plugin.RegisterMetrics(p, Name, metricSet.List()...)
	if err != nil {
		return errs.Wrap(err, "failed to register metrics")
	}

	return nil
}

func (p *emberPlugin) getCollectionByPath(
	collection ember.ElementCollection, connConf conn.ConnConfig, pathPart []string,
) (map[string]*ember.Element, error) {
	var fullPath string

	for _, part := range pathPart {
		fullPath = pathJoin(fullPath, part)

		el, err := collection.GetElementByPath(fullPath)
		if err != nil {
			return nil, errs.Errorf("failed to retrieve element with path %s", fullPath)
		}

		collection, err = p.handleRequest(fullPath, el.ElementType, connConf, ember.GetRequestByType)
		if err != nil {
			return nil, errs.Wrapf(err, "failed to retrieve element collection with path %s", fullPath)
		}
	}

	return collection.ToJSONCompatible(), nil
}

func (p *emberPlugin) getCollectionByID(
	collection ember.ElementCollection, connConf conn.ConnConfig, ids []string,
) (map[string]*ember.Element, error) {
	var fullPath string

	for _, id := range ids {
		el, err := collection.GetElementByID(id)
		if err != nil {
			return nil, errs.Errorf(
				"failed to retrieve element with id %s, path to element '%s'", id, fullPath,
			)
		}

		fullPath = el.Path

		collection, err = p.handleRequest(fullPath, el.ElementType, connConf, ember.GetRequestByType)
		if err != nil {
			return nil, errs.Wrapf(err, "failed to retrieve element collection with path '%s'", fullPath)
		}
	}

	return collection.ToJSONCompatible(), nil
}

func (p *emberPlugin) handleRequest(
	path string, elType ember.ElementType, connConf conn.ConnConfig, reqFunc ember.RequestFunction,
) (ember.ElementCollection, error) {
	req, err := reqFunc(elType, path)
	if err != nil {
		return nil, errs.Wrap(err, "failed to get request")
	}

	resp, err := p.conns.HandleRequest(req, connConf)
	if err != nil {
		return nil, errs.Wrap(err, "failed to handle request")
	}

	glow, err := s101.Decode(resp)
	if err != nil {
		return nil, errs.Wrap(err, "failed to decode response")
	}

	el := ember.NewElementConnection()

	err = el.Populate(asn1.NewDecoder(glow))
	if err != nil {
		return nil, errs.Wrap(err, "failed to populate glow response")
	}

	return el, nil
}

func withJSONResponse(handler handlerFunc) handlerFunc {
	return func(
		metricParams map[string]string, extraParams ...string,
	) (any, error) {
		res, err := handler(metricParams, extraParams...)
		if err != nil {
			return nil, errs.Wrap(err, "failed to execute handler")
		}

		jsonRes, err := json.Marshal(res)
		if err != nil {
			return nil, errs.Wrap(err, "failed to marshal result to JSON")
		}

		return string(jsonRes), nil
	}
}

func pathJoin(currentPath, pathPart string) string {
	if currentPath == "" {
		return pathPart
	}

	return fmt.Sprintf("%s.%s", currentPath, pathPart)
}

func parsePathString(path string) ([]string, bool, error) {
	var idPath bool

	split := strings.Split(path, ".")

	for _, s := range split {
		if strings.Trim(s, " ") == "" {
			return nil, false, errs.New("path part can not be empty")
		}

		oidPart, err := strconv.Atoi(s)
		if err != nil {
			idPath = true

			continue
		}

		if idPath {
			return nil, false, errs.New("path parts can not be a mix of OID and Identifier")
		}

		if oidPart < 0 {
			return nil, false, errs.New("path oid parts can not be negative")
		}
	}

	return split, idPath, nil
}
