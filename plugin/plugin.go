package plugin

import (
	"encoding/json"
	"fmt"
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

func (p *emberPlugin) GetEmber(metricParams map[string]string, _ ...string) (any, error) {
	rootColl, err := p.handleRequest("", "", metricParams, ember.GetRootRequest)
	if err != nil {
		return nil, errs.Wrap(err, "failed to retrieve root element collection")
	}

	path := metricParams[params.Path.Name()]
	if path == "" {
		return rootColl, nil
	}

	parsedPath := parsePathString(path)

	out := rootColl

	var currentPath string

	for _, pp := range parsedPath {
		currentPath = pathJoin(currentPath, pp)

		el, ok := out[currentPath]
		if !ok {
			return nil, errs.Errorf("failed to retrieve element with path %s", currentPath)
		}

		out, err = p.handleRequest(currentPath, el.ElementType, metricParams, ember.GetRequestByType)
		if err != nil {
			return nil, errs.Wrapf(err, "failed to retrieve element collection with path %s", currentPath)
		}
	}

	return out, nil
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

func pathJoin(currentPath, pathPart string) string {
	if currentPath == "" {
		return pathPart
	}

	return fmt.Sprintf("%s.%s", currentPath, pathPart)
}

func parsePathString(path string) []string {
	if len(path) == 0 {
		return nil
	}

	return strings.Split(path, ".")
}
