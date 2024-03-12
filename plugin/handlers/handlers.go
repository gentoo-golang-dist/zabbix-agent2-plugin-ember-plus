package handlers

import (
	"encoding/json"
	"net"

	ember "git.zabbix.com/ap/ember-plus/emberPlus"
	"git.zabbix.com/ap/ember-plus/plugin/params"
	"git.zabbix.com/ap/plugin-support/log"
)

// HandlerFunc describes the signature all metric handler functions must have.
type HandlerFunc func(metricParams map[string]string, extraParams ...string) (any, error)

// ConnHandlerFunc describes the signature all connection handler functions
// must have.
type ConnHandlerFunc func(
	conn net.Conn, log log.Logger, metricParams map[string]string, extraParams ...string,
) (any, error)

func GetEmber() ConnHandlerFunc {
	return func(conn net.Conn, log log.Logger, p map[string]string, _ ...string) (any, error) {
		path := p[params.Path.Name()]
		if path == "" {
			return handleRootRequest(conn, log)
		}

		return handleRequest(conn, path)
	}
}

func handleRootRequest(conn net.Conn, log log.Logger) (any, error) {
	el := ember.NewElementConnection()
	err := el.PopulateRootElement(conn, log)
	if err != nil {
		return nil, err
	}

	out, err := json.Marshal(el)
	if err != nil {
		return nil, err
	}

	return string(out), nil
}

func handleRequest(conn net.Conn, path string) (any, error) {
	el := ember.NewElementConnection()
	err := el.PopulateByPath(conn, path)
	if err != nil {
		return nil, err
	}

	out, err := json.Marshal(el)
	if err != nil {
		return nil, err
	}

	return string(out), nil
}
