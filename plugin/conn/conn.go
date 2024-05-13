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
	"strings"
	"sync"
	"time"

	"golang.zabbix.com/plugin/ember-plus/ember"
	"golang.zabbix.com/plugin/ember-plus/ember/asn1"
	"golang.zabbix.com/plugin/ember-plus/ember/s101"
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
	mu               sync.Mutex
	conn             net.Conn
	lastAccessTime   time.Time
	lastAccessTimeMu sync.Mutex
	logr             log.Logger
	expectResponse   *bool
	response         chan ember.ElementCollection
	expectedPath     chan string
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
func (c *ConnCollection) HandleRequest(req []byte, conf ConnConfig, path string) (ember.ElementCollection, error) {
	ch, err := c.get(time.Duration(c.callTimeout)*time.Second, conf)
	if err != nil {
		return nil, errs.Wrap(err, "failed to get conn")
	}

	ch.mu.Lock()
	defer ch.mu.Unlock()

	var expectOn = true
	var expectOff = false
	// turns on response expectation in the listener
	ch.expectResponse = &expectOn
	defer func() { ch.expectResponse = &expectOff }()

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

	select {
	case ch.expectedPath <- path:
		c.logr.Tracef("wrote path %s for request", path)
	case <-time.After((time.Duration(c.callTimeout) * time.Second) / 2):
		return nil, errs.Errorf("failed to send path %s for requested response", path)
	}

	select {
	case el := <-ch.response:
		return el, nil
	case <-time.After((time.Duration(c.callTimeout) * time.Second) / 2):
		return nil, errs.New("element not found")
	}
}

func (c *ConnCollection) Write(req []byte, conf ConnConfig) error {
	ch, err := c.get(time.Duration(c.callTimeout)*time.Second, conf)
	if err != nil {
		return errs.Wrap(err, "failed to get conn")
	}

	err = ch.conn.SetWriteDeadline(time.Now().Add(time.Duration(c.callTimeout) * time.Second))
	if err != nil {
		return errs.Wrap(err, "failed to set write deadline for connection")
	}

	_, err = ch.conn.Write(req)
	if err != nil {
		cerr := c.close(conf)
		if cerr != nil {
			c.logr.Errf("write connection clean-up failed, err: %w", cerr)
		}

		return errs.Wrap(err, "failed to write to connection")
	}

	return nil
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

	ch, err := newConn(timeout, conf, c.logr)
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

func (ch *connHandler) read() ([]byte, error) {
	var out []byte
	var multi bool

read:
	for {
		response := make([]byte, 1290)
		n, err := ch.conn.Read(response)
		if err != nil {
			return nil, errs.Wrap(err, "failed to read from connection")
		}

		glow, pType, err := s101.Decode(response[:n])
		if err != nil {
			ch.logr.Debugf("failed to decode response: %s", err.Error())

			continue
		}

		ch.logr.Tracef("got packet with type %x", pType)
		ch.logr.Tracef("got packet with data %x", response)

		switch pType {
		case s101.FirstMultiPacket, s101.BodyMultiPacket:
			out = append(out, glow...)
			multi = true

			continue
		case s101.LastMultiPacket:
			out = append(out, glow...)
			break read
		default:
			if multi {
				ch.logr.Errf("dropping message in the middle of a multi packet read %x", glow)
				continue
			}

			out = glow
			break read
		}
	}

	return out, nil
}

// updateLastAccessTime updates the last time a connection was accessed.
func (ch *connHandler) updateLastAccessTime() {
	ch.lastAccessTimeMu.Lock()
	defer ch.lastAccessTimeMu.Unlock()

	ch.lastAccessTime = time.Now()
}

// getLastAccessTime returns the last time a connection was accessed.
func (ch *connHandler) getLastAccessTime() time.Time {
	ch.lastAccessTimeMu.Lock()
	defer ch.lastAccessTimeMu.Unlock()

	return ch.lastAccessTime
}

func (ch *connHandler) reader() {
main:
	for {
		glow, err := ch.read()
		if err != nil {
			ch.logr.Debugf("failed to read from handler: %s", err.Error())

			return
		}

		if ch.expectResponse == nil || !*ch.expectResponse {
			sendUnsubscribe()
			ch.logr.Tracef("got spam data, skipping and sent unsubscribe request")

			continue
		}

		path := <-ch.expectedPath

		ch.logr.Tracef("got path for request %s", path)

		el := ember.NewElementConnection()

		err = el.Populate(asn1.NewDecoder(glow))
		if err != nil {
			ch.logr.Debugf("failed to populate glow response: %s", err.Error())

			continue
		}

		if len(el) == 0 {
			ch.logr.Tracef("empty collection, skipping")
			continue
		}

		var gotPath []string

		ch.logr.Tracef("collection, %+v", el)

		for k := range el {
			// we care only about the path from the first element as it's a control value and every other element
			// should have the same path prefix
			gotPath = strings.Split(k.Path, ".")
			ch.logr.Tracef("path from first element %s", gotPath)
			break
		}

		splitExpectedPath, expectedLength := parseExpectedLength(path)
		// gotPath has to be one path element longer
		if len(gotPath) != expectedLength && len(gotPath) != expectedLength+1 {
			ch.logr.Tracef(
				"path %s length %d does not match the expected path %s length %d",
				gotPath, len(gotPath), splitExpectedPath, expectedLength,
			)
			continue
		}

		for i, v := range splitExpectedPath {
			if gotPath[i] != v {
				ch.logr.Tracef("path %s does not match the expected %s", gotPath, splitExpectedPath)
				continue main
			}
		}

		ch.logr.Tracef("found expected response with path %s", path)

		ch.response <- el
	}
}

func parseExpectedLength(path string) ([]string, int) {
	if path == "" {
		return nil, 1
	}

	splitExpectedPath := strings.Split(path, ".")

	return splitExpectedPath, len(splitExpectedPath)
}

func sendUnsubscribe() {}

func newConn(timeout time.Duration, conf ConnConfig, logger log.Logger) (*connHandler, error) {
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

	ch := &connHandler{
		conn:           conn,
		lastAccessTime: time.Now(),
		// callTimeout:    timeout,
		logr:           logger,
		expectResponse: nil,
		response:       make(chan ember.ElementCollection),
		expectedPath:   make(chan string),
	}

	go ch.reader()

	return ch, nil
}
