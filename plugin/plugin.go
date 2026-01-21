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

import "C"

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"golang.zabbix.com/plugin/ember-plus/ember"
	"golang.zabbix.com/plugin/ember-plus/plugin/conn"
	"golang.zabbix.com/plugin/ember-plus/plugin/emberlib"
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

// EmberPlugin global ember plugin structure.
type EmberPlugin struct {
	plugin.Base
	conns   *conn.ConnCollection
	config  *pluginConfig
	metrics map[emberMetricKey]*emberMetric
}

// HandlerFunc describes the signature all metric handler functions must have.
type handlerFunc func(
	timeout time.Duration,
	handler *emberlib.Handler,
	metricParams map[string]string,
	extraParams ...string,
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
	emberlib.Init(p.Logger)

	p.conns.Init(p.config.KeepAlive, p)
}

// Stop stops the mssql plugin, closing all the connections.
func (p *EmberPlugin) Stop() {
	p.conns.CloseAll()
}

// Export collects all the metrics.
func (p *EmberPlugin) Export(key string, rawParams []string, pluginCtx plugin.ContextProvider) (any, error) {
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

	timeout := time.Second * time.Duration(p.config.Timeout)

	if pluginCtx != nil && timeout < time.Second*time.Duration(pluginCtx.Timeout()) {
		timeout = time.Second * time.Duration(pluginCtx.Timeout())
	}

	emberLibHandler, err := emberlib.NewHandler()
	if err != nil {
		return nil, errs.Wrap(err, "failed to init ember handler")
	}

	defer emberLibHandler.CleanUp()

	res, err := m.handler(timeout, emberLibHandler, metricParams, extraParams...)
	if err != nil {
		return nil, errs.Wrap(err, "failed to get response")
	}

	return res, nil
}

// GetEmber handles ember.get metric, returns collection data based on request metrics, response needs to be handled,
// otherwise it is not possible to json marshal it.
func (p *EmberPlugin) GetEmber(
	timeout time.Duration,
	handler *emberlib.Handler,
	metricParams map[string]string,
	_ ...string,
) (any, error) {
	connConf, err := conn.NewConnConfig(metricParams[params.URI.Name()])
	if err != nil {
		return nil, errs.Wrap(err, "failed to create connection config")
	}

	rootCollection, err := p.conns.HandleRequest(ember.NodeType, connConf, "", timeout, handler)
	if err != nil {
		return nil, errs.Wrap(err, "failed to handle request")
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
		return p.getCollectionByID(handler, rootCollection, connConf, pathParts, timeout)
	}

	return p.getCollectionByPath(handler, rootCollection, connConf, pathParts, timeout)
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
	handler *emberlib.Handler,
	collection ember.ElementCollection,
	connConf conn.ConnConfig,
	pathPart []string,
	timeout time.Duration,
) (ember.ElementCollection, error) {
	var fullPath string

	for _, part := range pathPart {
		fullPath = pathJoin(fullPath, part)
		el, err := collection.GetElementByPath(fullPath)
		if err != nil {
			return nil, errs.Wrapf(err, "failed to get element with path %s", fullPath)
		}

		collection, err = p.conns.HandleRequest(el.ElementType, connConf, fullPath, timeout, handler)
		if err != nil {
			return nil, errs.Wrap(err, "failed to handle request")
		}
	}

	return collection, nil
}

func (p *EmberPlugin) getCollectionByID(
	handler *emberlib.Handler,
	collection ember.ElementCollection,
	connConf conn.ConnConfig,
	ids []string,
	timeout time.Duration,
) (ember.ElementCollection, error) {
	for _, id := range ids {
		el, fullPath, err := collection.GetElementByID(id)
		if err != nil {
			return nil, errs.Wrapf(
				err,
				"failed to retrieve element with id %s", id,
			)
		}

		collection, err = p.conns.HandleRequest(el.ElementType, connConf, fullPath, timeout, handler)
		if err != nil {
			return nil, errs.Wrap(err, "failed to handle request")
		}
	}

	return collection, nil
}

func withJSONResponse(handler handlerFunc) handlerFunc {
	return func(
		timeout time.Duration, h *emberlib.Handler, metricParams map[string]string, extraParams ...string,
	) (any, error) {
		res, err := handler(timeout, h, metricParams, extraParams...)
		if err != nil {
			return nil, errs.Wrap(err, "handler function failed")
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
