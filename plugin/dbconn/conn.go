/*
** Zabbix
** Copyright 2001-2024 Zabbix SIA
**
** Licensed under the Apache License, Version 2.0 (the "License");
** you may not use this file except in compliance with the License.
** You may obtain a copy of the License at
**
**     http://www.apache.org/licenses/LICENSE-2.0
**
** Unless required by applicable law or agreed to in writing, software
** distributed under the License is distributed on an "AS IS" BASIS,
** WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
** See the License for the specific language governing permissions and
** limitations under the License.
**/

package dbconn

import (
	"net"
	"net/url"
	"sync"
	"time"

	"git.zabbix.com/ap/ember-plus/plugin/handlers"
	"git.zabbix.com/ap/ember-plus/plugin/params"
	"git.zabbix.com/ap/plugin-support/errs"
	"git.zabbix.com/ap/plugin-support/log"
	"git.zabbix.com/ap/plugin-support/uri"
)

var (
	_ handlers.HandlerFunc = (*ConnCollection)(nil).WithConnHandlerFunc(nil)
)

// connConfig is a configuration for a connection to the database.
type connConfig struct {
	URI string
}

// ConnCollection is a collection of connections to the database.
// Allows managing multiple connections.
type ConnCollection struct {
	mu          sync.Mutex
	conns       map[connConfig]net.Conn
	keepAlive   int
	callTimeout int
	logr        log.Logger
}

// Init initializes a pre-allocated connection collection.
func (c *ConnCollection) Init(keepAlive, callTimeout int, logr log.Logger) {
	c.conns = make(map[connConfig]net.Conn)
	c.keepAlive = keepAlive
	c.callTimeout = callTimeout
	c.logr = logr
}

// WithConnHandlerFunc creates a new function that creates or gets cached DB
// connection for the given metric parameters and calls the given handler
// function with the connection.
func (c *ConnCollection) WithConnHandlerFunc(handler handlers.ConnHandlerFunc) handlers.HandlerFunc {
	return func(
		metricParams map[string]string, extraParams ...string,
	) (any, error) {
		conn, err := c.get(time.Duration(c.callTimeout)*time.Second, newConnConfig(metricParams))
		if err != nil {
			return nil, errs.Wrap(err, "failed to get conn")
		}

		return handler(conn, metricParams, extraParams...)
	}
}

// Close closes all connections in the collection.
func (c *ConnCollection) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for conf, conn := range c.conns {
		err := conn.Close()
		if err != nil {
			c.logr.Errf("failed to close connection: %s", err.Error())
		}

		delete(c.conns, conf)
	}
}

func (c *ConnCollection) get(timeout time.Duration, conf connConfig) (net.Conn, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	conn, ok := c.conns[conf]
	if ok {
		return conn, nil
	}

	conn, err := c.newConn(timeout, &conf)
	if err != nil {
		return nil, errs.Wrap(err, "failed to create conn")
	}

	c.conns[conf] = conn

	return conn, nil
}

func (c *ConnCollection) newConn(timeout time.Duration, conf *connConfig) (net.Conn, error) {
	c.logr.Infof("Creating new connection to %q, with user %q to database %q", conf.URI)

	connURI, err := uri.New(conf.URI, nil)
	if err != nil {
		return nil, errs.Wrap(err, "failed to set URI defaults")
	}

	u, err := url.Parse(connURI.String())
	if err != nil {
		return nil, errs.Wrap(err, "failed to parse URI")
	}

	d := &net.Dialer{Timeout: timeout, KeepAlive: time.Duration(c.keepAlive)}
	conn, err := d.Dial("TCP", u.String())
	if err != nil {
		return nil, errs.Wrap(err, "failed to create connection")
	}

	return conn, nil
}

func newConnConfig(metricParams map[string]string) connConfig {
	return connConfig{
		URI: metricParams[params.URI.Name()],
	}
}
