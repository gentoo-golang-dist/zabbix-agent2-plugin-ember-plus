package plugin

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	ember "git.zabbix.com/ap/ember-plus/emberPlus"
	"git.zabbix.com/ap/ember-plus/plugin/conn"
	"git.zabbix.com/ap/ember-plus/plugin/params"
	"git.zabbix.com/ap/plugin-support/errs"
	"git.zabbix.com/ap/plugin-support/metric"
	"git.zabbix.com/ap/plugin-support/plugin"
	"git.zabbix.com/ap/plugin-support/plugin/container"
	"git.zabbix.com/ap/plugin-support/zbxerr"
)

const (
	Name = "EmberPlus"

	get = emberMetricKey("ember.get")
)

var Impl Plugin

// Plugin -
type Plugin struct {
	plugin.Base
}

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

func (p *emberPlugin) registerMetrics() error {
	p.metrics = map[emberMetricKey]*emberMetric{
		get: {
			metric: metric.New(
				"Returns the ember data based on path.",
				params.Join(params.BaseParams, params.EmberGetParams),
				false,
			),
			handler: withJSONResponse(withRootCollectionResponse(p.GetEmber)),
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

func (p *emberPlugin) GetEmber(metricParams map[string]string, _ ...string) (any, error) {
	rootCollection, err := p.handleRequest("", "", metricParams, ember.GetRootRequest)
	if err != nil {
		return nil, errs.Wrap(err, "failed to retrieve root collection")
	}

	path := metricParams[params.Path.Name()]
	if path == "" {
		return rootCollection, nil
	}

	parsedPath, byID, err := parsePathString(path)
	if err != nil {
		return nil, errs.Errorf("failed to parse input path '%s', err: %s", path, err.Error())
	}

	collection, err := p.getCollection(rootCollection, parsedPath, metricParams, byID)
	if err != nil {
		return nil, errs.Errorf("failed to retrieve collection for path '%s', err: %s", path, err.Error())
	}

	return collection, nil
}

func (p *emberPlugin) getCollection(
	collection ember.ElementCollection, paths []string, params map[string]string, byId bool,
) (ember.ElementCollection, error) {
	if byId {
		return p.getCollectionByID(collection, params, paths)
	}

	return p.getCollectionByPath(collection, params, paths)
}

func (p *emberPlugin) getCollectionByPath(
	collection ember.ElementCollection, params map[string]string, paths []string,
) (ember.ElementCollection, error) {
	var fullPath string

	for _, path := range paths {
		fullPath = pathJoin(fullPath, path)

		el, err := getElementByPath(collection, fullPath)
		if err != nil {
			return nil, errs.Errorf("failed to retrieve element with path %s", fullPath)
		}

		collection, err = p.handleRequest(fullPath, el.ElementType, params, ember.GetRequestByType)
		if err != nil {
			return nil, errs.Wrapf(err, "failed to retrieve element collection with path %s", fullPath)
		}
	}

	return collection, nil
}

func (p *emberPlugin) getCollectionByID(
	collection ember.ElementCollection, params map[string]string, ids []string,
) (ember.ElementCollection, error) {
	var fullPath string

	for _, id := range ids {
		el, err := getElementByID(collection, id)
		if err != nil {
			return nil, errs.Errorf(
				"failed to retrieve element with id %s, path to element '%s'", id, fullPath,
			)
		}

		fullPath = el.Path

		collection, err = p.handleRequest(fullPath, el.ElementType, params, ember.GetRequestByType)
		if err != nil {
			return nil, errs.Wrapf(err, "failed to retrieve element collection with path '%s'", fullPath)
		}
	}

	return collection, nil
}

func getElementByPath(collection ember.ElementCollection, currentPath string) (*ember.Element, error) {
	for key, value := range collection {
		if key.Path == currentPath {
			return value, nil
		}
	}

	return nil, errs.Errorf("element not found")
}

func getElementByID(collection ember.ElementCollection, id string) (*ember.Element, error) {
	for key, value := range collection {
		if key.Id == id {
			return value, nil
		}
	}

	return nil, errs.Errorf("element not found")
}

func (p *emberPlugin) handleRequest(
	path string, elType ember.ElementType, metricParams map[string]string, reqFunc ember.RequestFunction,
) (ember.ElementCollection, error) {
	req, err := reqFunc(elType, path)
	if err != nil {
		return nil, err
	}

	resp, err := p.conns.HandleRequest(req, metricParams)
	if err != nil {
		return nil, err
	}

	codec := ember.NewCodec()

	glow, err := codec.Decode(resp)
	if err != nil {
		return nil, err
	}

	el := ember.NewElementConnection()

	err = el.Populate(ember.NewASN1Decoder(glow))
	if err != nil {
		return nil, err
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

func withRootCollectionResponse(handler handlerFunc) handlerFunc {
	return func(
		metricParams map[string]string, extraParams ...string,
	) (any, error) {
		res, err := handler(metricParams, extraParams...)
		if err != nil {
			return nil, errs.Wrap(err, "failed to execute handler")
		}

		collection, ok := res.(ember.ElementCollection)
		if !ok {
			return nil, errs.Wrapf(err, "unknown response type %T", res)
		}

		out := make(map[string]*ember.Element)

		for k, v := range collection {
			out[k.Path] = v
		}

		return out, nil
	}
}

func pathJoin(currentPath, pathPart string) string {
	if currentPath == "" {
		return pathPart
	}

	return fmt.Sprintf("%s.%s", currentPath, pathPart)
}

func parsePathString(path string) ([]string, bool, error) {
	if len(path) == 0 {
		return nil, false, nil
	}

	var idPath bool

	split := strings.Split(path, ".")

	for _, s := range split {
		if strings.Trim(s, " ") == "" {
			return nil, false, errs.New("path part can not be empty")
		}

		_, err := strconv.Atoi(s)
		if err != nil {
			idPath = true
		}
	}

	return split, idPath, nil
}
