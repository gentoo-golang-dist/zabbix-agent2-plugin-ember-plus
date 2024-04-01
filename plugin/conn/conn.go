//nolint:gci,gofmt
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

package conn

import (
	"net"
	"net/url"
	"sync"
	"time"

	"git.zabbix.com/ap/ember-plus/plugin/params"
	"git.zabbix.com/ap/plugin-support/errs"
	"git.zabbix.com/ap/plugin-support/log"
	"git.zabbix.com/ap/plugin-support/uri"
)

const interval = 10

// connConfig is a configuration for a connection to the database.
type connConfig struct {
	URI string
}

// ConnCollection is a collection of connections to the database.
// Allows managing multiple connections.
type ConnCollection struct {
	mu          sync.Mutex
	conns       map[connConfig]*connHandler
	callTimeout int
	keepAlive   time.Duration
	logr        log.Logger
	done        chan bool
}

type connHandler struct {
	conn           net.Conn
	lastAccessTime time.Time
}

// Init initializes a pre-allocated connection collection.
func (c *ConnCollection) Init(keepAlive, callTimeout int, logr log.Logger) {
	c.conns = make(map[connConfig]*connHandler)
	c.keepAlive = time.Duration(keepAlive) * time.Second
	c.callTimeout = callTimeout
	c.logr = logr

	go c.housekeeper(interval * time.Second)
}

// HandleRequest sends a request and reads response based on the provided connection parameters.
func (c *ConnCollection) HandleRequest(req []byte, metricParams map[string]string) ([]byte, error) {
	conf := newConnConfig(metricParams)

	ch, err := c.get(time.Duration(c.callTimeout)*time.Second, conf)
	if err != nil {
		return nil, errs.Wrap(err, "failed to get conn")
	}

	err = ch.conn.SetWriteDeadline(time.Now().Add(time.Duration(c.callTimeout) * time.Second))
	if err != nil {
		return nil, errs.Wrap(err, "failed to set write deadline for connection")
	}

	_, err = ch.conn.Write(req)
	if err != nil {
		cerr := c.close(conf)
		if cerr != nil {
			c.logr.Errf("write connection clean-up failed, err: %w", cerr)
		}

		return nil, errs.Wrap(err, "failed to write to connection")
	}

	err = ch.conn.SetReadDeadline(time.Now().Add(time.Duration(c.callTimeout) * time.Second))
	if err != nil {
		return nil, errs.Wrap(err, "failed to set read deadline for connection")
	}

	//nolint:makezero
	response := make([]byte, 1024)

	_, err = ch.conn.Read(response)
	if err != nil {
		cerr := c.close(conf)
		if cerr != nil {
			c.logr.Errf("read connection clean-up failed, err: %w", cerr)
		}

		return nil, errs.Wrap(err, "failed to read from connection")
	}

	return response, nil
}

// CloseAll closes all connections in the collection.
func (c *ConnCollection) CloseAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	close(c.done)

	for conf, ch := range c.conns {
		err := ch.conn.Close()
		if err != nil {
			c.logr.Errf("failed to close connection: %s", err.Error())
		}

		delete(c.conns, conf)
	}
}

// close closes the connection with the provided configuration.
func (c *ConnCollection) close(conf connConfig) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	ch, ok := c.conns[conf]
	if !ok {
		return nil
	}

	err := ch.conn.Close()
	if err != nil {
		return errs.Wrap(err, "failed to close connection")
	}

	delete(c.conns, conf)

	return nil
}

func (c *ConnCollection) get(timeout time.Duration, conf connConfig) (*connHandler, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.logr.Debugf("looking for connection for %s", conf.URI)

	ch, ok := c.conns[conf]
	if ok {
		ch.lastAccessTime = time.Now()

		return ch, nil
	}

	ch, err := newConn(timeout, &conf)
	if err != nil {
		return nil, errs.Wrap(err, "failed to create conn")
	}

	c.conns[conf] = ch

	return ch, nil
}

// housekeeper repeatedly checks for unused connections and closes them.
func (c *ConnCollection) housekeeper(interval time.Duration) {
	ticker := time.NewTicker(interval)

	c.logr.Debugf("starting housekeeper")

	for {
		select {
		case <-c.done:
			c.logr.Debugf("housekeeper done")

			return
		case <-ticker.C:
			c.logr.Debugf("house keeper tick")

			c.closeUnused()
		}
	}
}

func (c *ConnCollection) closeUnused() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for conf, conn := range c.conns {
		if time.Since(conn.lastAccessTime) > c.keepAlive {
			err := conn.conn.Close()
			if err != nil {
				c.logr.Errf("failed to close connection: %s", conf.URI)
			}

			delete(c.conns, conf)
			c.logr.Debugf("closed unused connection: %s", conf.URI)
		}
	}
}

func newConn(timeout time.Duration, conf *connConfig) (*connHandler, error) {
	connURI, err := uri.New(conf.URI, nil)
	if err != nil {
		return nil, errs.Wrap(err, "failed to set URI defaults")
	}

	u, err := url.Parse(connURI.String())
	if err != nil {
		return nil, errs.Wrap(err, "failed to parse URI")
	}

	d := &net.Dialer{Timeout: timeout, KeepAlive: 4 * time.Second}

	conn, err := d.Dial("tcp", u.Host)
	if err != nil {
		return nil, errs.Wrap(err, "failed to create connection")
	}

	return &connHandler{conn: conn, lastAccessTime: time.Now()}, nil
}

func newConnConfig(metricParams map[string]string) connConfig {
	return connConfig{URI: metricParams[params.URI.Name()]}
}
