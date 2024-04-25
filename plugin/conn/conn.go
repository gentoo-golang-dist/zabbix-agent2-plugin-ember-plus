/*
** Zabbix
** Copyright (C) 2001-2024 Zabbix SIA
**
** This program is free software; you can redistribute it and/or modify
** it under the terms of the GNU General Public License as published by
** the Free Software Foundation; either version 2 of the License, or
** (at your option) any later version.
**
** This program is distributed in the hope that it will be useful,
** but WITHOUT ANY WARRANTY; without even the implied warranty of
** MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
** GNU General Public License for more details.
**
** You should have received a copy of the GNU General Public License
** along with this program; if not, write to the Free Software
** Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA  02110-1301, USA.
**/

package conn

import (
	"net"
	"net/url"
	"sync"
	"time"

	"golang.zabbix.com/sdk/errs"
	"golang.zabbix.com/sdk/log"
	"golang.zabbix.com/sdk/uri"
)

const interval = 10

// ConnConfig is a configuration for a connection to the database.
type ConnConfig struct {
	URI string
}

// ConnCollection is a collection of connections to the database.
// Allows managing multiple connections.
type ConnCollection struct {
	mu          sync.Mutex
	conns       map[ConnConfig]*connHandler
	callTimeout int
	keepAlive   time.Duration
	logr        log.Logger
	done        chan bool
}

type connHandler struct {
	conn             net.Conn
	lastAccessTime   time.Time
	lastAccessTimeMu sync.Mutex
}

// Init initializes a pre-allocated connection collection.
func (c *ConnCollection) Init(keepAlive, callTimeout int, logr log.Logger) {
	c.conns = make(map[ConnConfig]*connHandler)
	c.keepAlive = time.Duration(keepAlive) * time.Second
	c.callTimeout = callTimeout
	c.logr = logr
	c.done = make(chan bool)

	go c.housekeeper(interval * time.Second)
}

// HandleRequest sends a request and reads response based on the provided connection parameters.
func (c *ConnCollection) HandleRequest(req []byte, conf ConnConfig) ([]byte, error) {
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

	//nolint:makezero // value taken from ember+ documentation
	response := make([]byte, 1290)

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

// NewConnConfig creates connection configuration with provided uri string.
func NewConnConfig(rawURI string) (ConnConfig, error) {
	parsed, err := uri.New(rawURI, nil)
	if err != nil {
		return ConnConfig{}, errs.Wrap(err, "failed to parse uri")
	}

	return ConnConfig{URI: parsed.Addr()}, nil
}

// close closes the connection with the provided configuration.
func (c *ConnCollection) close(conf ConnConfig) error {
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

func (c *ConnCollection) get(timeout time.Duration, conf ConnConfig) (*connHandler, error) {
	c.logr.Debugf("looking for connection for %s", conf.URI)

	ch := c.getConn(conf)
	if ch != nil {
		c.logr.Debugf("connection found for %s", conf.URI)

		ch.updateLastAccessTime()

		return ch, nil
	}

	c.logr.Debugf("creating new connection for %s", conf.URI)

	ch, err := newConn(timeout, conf)
	if err != nil {
		return nil, errs.Wrap(err, "failed to create conn")
	}

	return c.setConn(conf, ch), nil
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
		if time.Since(conn.getLastAccessTime()) > c.keepAlive {
			err := conn.conn.Close()
			if err != nil {
				c.logr.Errf("failed to close connection: %s", conf.URI)
			}

			delete(c.conns, conf)
			c.logr.Debugf("closed unused connection: %s", conf.URI)
		}
	}
}

// getConn concurrent connections cache getter.
func (c *ConnCollection) getConn(cc ConnConfig) *connHandler {
	c.mu.Lock()
	defer c.mu.Unlock()

	ch, ok := c.conns[cc]
	if !ok {
		return nil
	}

	return ch
}

// setConn concurrent connections cache setter.
//
// Returns the cached connection. If the provider connection is already present
// in cache, it is closed.
func (c *ConnCollection) setConn(cc ConnConfig, ch *connHandler) *connHandler {
	c.mu.Lock()
	defer c.mu.Unlock()

	existingHandler, ok := c.conns[cc]
	if ok {
		defer ch.conn.Close() //nolint:errcheck

		c.logr.Debugf("closed redundant connection: %s", cc.URI)

		return existingHandler
	}

	c.conns[cc] = ch

	return ch
}

// updateLastAccessTime updates the last time a connection was accessed.
func (conn *connHandler) updateLastAccessTime() {
	conn.lastAccessTimeMu.Lock()
	defer conn.lastAccessTimeMu.Unlock()

	conn.lastAccessTime = time.Now()
}

// getLastAccessTime returns the last time a connection was accessed.
func (conn *connHandler) getLastAccessTime() time.Time {
	conn.lastAccessTimeMu.Lock()
	defer conn.lastAccessTimeMu.Unlock()

	return conn.lastAccessTime
}

func newConn(timeout time.Duration, conf ConnConfig) (*connHandler, error) {
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
