/*
** Copyright (C) 2001-2026 Zabbix SIA
**
** This program is free software: you can redistribute it and/or modify it under the terms of
** the GNU Affero General Public License as published by the Free Software Foundation, version 3.
**
** This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY;
** without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.
** See the GNU Affero General Public License for more details.
**
** You should have received a copy of the GNU Affero General Public License along with this program.
** If not, see <https://www.gnu.org/licenses/>.
**/

package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"golang.zabbix.com/plugin/ember-plus/ember"
	"golang.zabbix.com/plugin/ember-plus/plugin/conn"
	"golang.zabbix.com/plugin/ember-plus/plugin/params"
	"golang.zabbix.com/sdk/errs"
	"golang.zabbix.com/sdk/log"
	"golang.zabbix.com/sdk/metric"
	"golang.zabbix.com/sdk/plugin"
	"golang.zabbix.com/sdk/plugin/container"
	"golang.zabbix.com/sdk/zbxerr"
)

const (
	// Name is the name of the plugin.
	Name = "EmberPlus"

	get = emberMetricKey("ember.get")
)

var (
	_ plugin.Configurator = (*EmberPlugin)(nil)
	_ plugin.Exporter     = (*EmberPlugin)(nil)
	_ plugin.Runner       = (*EmberPlugin)(nil)
	_ handlerFunc         = (*EmberPlugin)(nil).GetEmber

	// ErrInvalidPath error when incorrect path is provided.
	ErrInvalidPath = errs.New("invalid path")
)

// EmberPlugin holds plugin parameters.
type EmberPlugin struct {
	plugin.Base
	conns   *conn.ConnCollection
	config  *pluginConfig
	metrics map[emberMetricKey]*emberMetric
}

// HandlerFunc describes the signature all metric handler functions must have.
type handlerFunc func(
	ctx context.Context, connectionTimeout int, metricParams map[string]string, extraParams ...string,
) (any, error)

type emberMetricKey string

type emberMetric struct {
	metric  *metric.Metric
	handler handlerFunc
}

// New creates new plugin implementation.
func New() (*EmberPlugin, error) {
	p := &EmberPlugin{
		conns: &conn.ConnCollection{},
	}

	err := log.Open(log.Console, log.Info, "", 0)
	if err != nil {
		return nil, errs.Wrap(err, "failed to open log")
	}

	p.Logger = log.New(Name)

	err = p.registerMetrics()
	if err != nil {
		return nil, errs.Wrap(err, "failed to register metrics")
	}

	return p, nil
}

// Run starts the plugin.
func (p *EmberPlugin) Run() error {
	h, err := container.NewHandler(Name)
	if err != nil {
		return errs.Wrap(err, "failed to create new handler")
	}

	p.Logger = h

	err = h.Execute()
	if err != nil {
		return errs.Wrap(err, "failed to execute plugin handler")
	}

	return nil
}

// Start initiates the connection handler.
func (p *EmberPlugin) Start() {
	p.conns.Init(p.config.KeepAlive, p)
}

// Stop stops the mssql plugin, closing all the connections.
func (p *EmberPlugin) Stop() {
	p.conns.CloseAll()
}

// Export collects all the metrics.
func (p *EmberPlugin) Export(key string, rawParams []string, ctx plugin.ContextProvider) (any, error) {
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

	// temporary workaround until metric.SetDefaults() supports integers
	connectionTimeout, err := strconv.Atoi(metricParams["ConnectionTimeout"])
	if err != nil {
		connectionTimeout = p.config.Default.ConnectionTimeout // shouldn't happen anyway
	}

	res, err := m.handler(ctx, connectionTimeout, metricParams, extraParams...)
	if err != nil {
		return nil, errs.Wrap(err, "failed to execute handler")
	}

	return res, nil
}

// GetEmber handles ember.get metric, returns collection data based on request metrics, response needs to be handled,
// otherwise it is not possible to json marshal it.
func (p *EmberPlugin) GetEmber(
	ctx context.Context, connectionTimeout int, metricParams map[string]string, _ ...string,
) (any, error) {
	connConf, err := conn.NewConnConfig(metricParams[params.URI.Name()])
	if err != nil {
		return nil, errs.Wrap(err, "failed to create connection config")
	}

	req, err := ember.GetRootRequest()
	if err != nil {
		return nil, errs.Wrap(err, "failed to get root collection request")
	}

	rootCollection, err := p.conns.HandleRequest(ctx, connectionTimeout, req, connConf, "")
	if err != nil {
		return nil, errs.Wrap(err, "failed to retrieve root collection")
	}

	path := metricParams[params.Path.Name()]
	if path == "" {
		return rootCollection, nil
	}

	pathParts, byID, err := parsePathString(path)
	if err != nil {
		return nil, errs.Wrapf(err, "failed to parse input path '%s'", path)
	}

	if byID {
		return p.getCollectionByID(ctx, connectionTimeout, rootCollection, connConf, pathParts)
	}

	return p.getCollectionByPath(ctx, connectionTimeout, rootCollection, connConf, pathParts)
}

func (p *EmberPlugin) registerMetrics() error {
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

func (p *EmberPlugin) getCollectionByPath(
	ctx context.Context,
	connectionTimeout int,
	collection ember.ElementCollection,
	connConf conn.ConnConfig,
	pathPart []string,
) (ember.ElementCollection, error) {
	var fullPath string

	for _, part := range pathPart {
		fullPath = pathJoin(fullPath, part)

		el, err := collection.GetElementByPath(fullPath)
		if err != nil {
			return nil, errs.Wrapf(err, "failed to retrieve element with path %s", fullPath)
		}

		req, err := ember.GetRequestByType(el.ElementType, fullPath)
		if err != nil {
			return nil, errs.Wrap(err, "failed to get request by element type")
		}

		collection, err = p.conns.HandleRequest(ctx, connectionTimeout, req, connConf, fullPath)
		if err != nil {
			return nil, errs.Wrap(err, "failed to handle request")
		}
	}

	return collection, nil
}

func (p *EmberPlugin) getCollectionByID(
	ctx context.Context,
	connectionTimeout int,
	collection ember.ElementCollection,
	connConf conn.ConnConfig,
	ids []string,
) (ember.ElementCollection, error) {
	for _, id := range ids {
		el, fullPath, err := collection.GetElementByID(id)
		if err != nil {
			return nil, errs.Wrapf(
				err,
				"failed to retrieve element with id %s, path to element '%s'", id, el.Path,
			)
		}

		req, err := ember.GetRequestByType(el.ElementType, fullPath)
		if err != nil {
			return nil, errs.Wrap(err, "failed to get request")
		}

		collection, err = p.conns.HandleRequest(ctx, connectionTimeout, req, connConf, fullPath)
		if err != nil {
			return nil, errs.Wrap(err, "failed to handle request")
		}
	}

	return collection, nil
}

func withJSONResponse(handler handlerFunc) handlerFunc {
	return func(
		ctx context.Context, connectionTimeout int, metricParams map[string]string, extraParams ...string,
	) (any, error) {
		res, err := handler(ctx, connectionTimeout, metricParams, extraParams...)
		if err != nil {
			return nil, errs.Wrap(err, "handler failed")
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
			return nil, false, errs.Wrapf(
				ErrInvalidPath,
				"path %q parts can not be a mix of OID and Identifier",
				path,
			)
		}

		if oidPart < 0 {
			return nil, false, errs.Wrapf(
				ErrInvalidPath,
				"path %q OID parts can not be negative",
				path,
			)
		}
	}

	return split, idPath, nil
}
