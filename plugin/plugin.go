package plugin

import (
	"git.zabbix.com/ap/ember-plus/plugin/dbconn"
	"git.zabbix.com/ap/ember-plus/plugin/handlers"
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
)

type emberMetricKey string

type emberMetric struct {
	metric  *metric.Metric
	handler handlers.HandlerFunc
}

type emberPlugin struct {
	plugin.Base
	conns   *dbconn.ConnCollection
	config  *pluginConfig
	metrics map[emberMetricKey]*emberMetric
}

func Launch() error {
	p := &emberPlugin{
		conns: &dbconn.ConnCollection{},
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

func (p *emberPlugin) Start() {
	p.Infof("timeout, %d", p.config.Timeout)
	p.Infof("timeout, %d", p.config.KeepAlive)
	p.conns.Init(p.config.KeepAlive, p.config.Timeout, p)
}

// Stop stops the mssql plugin, closing all the connections.
func (p *emberPlugin) Stop() {
	p.conns.Close()
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
			handler: p.conns.WithConnHandlerFunc(handlers.GetEmber()),
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
