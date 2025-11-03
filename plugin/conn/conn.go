/*
** Copyright (C) 2001-2025 Zabbix SIA
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

package conn

import (
	"golang.zabbix.com/plugin/ember-plus/ember/s101"
	"golang.zabbix.com/plugin/ember-plus/plugin/emberlib"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.zabbix.com/plugin/ember-plus/ember"
	"golang.zabbix.com/plugin/ember-plus/ember/asn1"
	"golang.zabbix.com/sdk/errs"
	"golang.zabbix.com/sdk/log"
	"golang.zabbix.com/sdk/uri"
)

// ErrConnectionSet error when trying to cache a connection handler that already exists.
var ErrConnectionSet = errs.New("connection handler already exists")

// ConnConfig is a configuration for a connection to the database.
type ConnConfig struct {
	URI string
}

// ConnCollection is a collection of connections to the database.
// Allows managing multiple connections.
type ConnCollection struct {
	mu        sync.Mutex
	conns     map[ConnConfig]*connHandler
	keepAlive time.Duration
	logr      log.Logger
	done      chan bool
}

type connHandler struct {
	mu               sync.Mutex
	conn             net.Conn
	conf             ConnConfig
	lastAccessTime   time.Time
	lastAccessTimeMu sync.Mutex
	logr             log.Logger
	readData         chan readResponse
	parsedData       chan parsedResponse
	expectedPath     chan string
}

type parsedResponse struct {
	element ember.ElementCollection
	err     error
}

type readResponse struct {
	collection ember.ElementCollection
	err        error
}

// Init initializes a pre-allocated connection collection.
func (c *ConnCollection) Init(keepAlive int, logr log.Logger) {
	c.conns = make(map[ConnConfig]*connHandler)
	c.keepAlive = time.Duration(keepAlive) * time.Second
	c.logr = logr
	c.done = make(chan bool)

	go c.housekeeper(10 * time.Second)
}

// HandleRequest sends a request and reads response based on the provided connection parameters.
func (c *ConnCollection) HandleRequestNew(
	req string,
	conf ConnConfig,
	path string,
	reqTimeout time.Duration,
	el *emberlib.EmberLib,
) (ember.ElementCollection, error) {
	ch, err := c.get(reqTimeout, conf, el)
	if err != nil {
		return nil, errs.Wrap(err, "failed to get conn")
	}

	ch.mu.Lock()
	defer ch.mu.Unlock()

	// turns on response expectation in the listener

	ch.expectedPath <- path

	err = ch.conn.SetWriteDeadline(time.Now().Add(reqTimeout))
	if err != nil {
		return nil, errs.Wrap(err, "failed to set write deadline for connection")
	}

	err = emberlib.SendGetDirectory(ch.conn, path, req)
	if err != nil {
		cerr := c.close(conf)
		if cerr != nil {
			c.logr.Errf("connection failed %w, write connection clean-up failed, err: %w", err, cerr)
		}

		return nil, errs.Wrap(err, "failed to write to connection")
	}

	data := <-ch.parsedData
	if data.err != nil {
		return nil, errs.Wrapf(ember.ErrElementNotFound, "failed to find element, err %s", data.err.Error())
	}

	return data.element, nil
}

//
//// HandleRequest sends a request and reads response based on the provided connection parameters.
//func (c *ConnCollection) HandleRequest(
//	req []byte,
//	conf ConnConfig,
//	path string,
//	reqTimeout time.Duration,
//) (ember.ElementCollection, error) {
//	ch, err := c.get(reqTimeout, conf)
//	if err != nil {
//		return nil, errs.Wrap(err, "failed to get conn")
//	}
//
//	ch.mu.Lock()
//	defer ch.mu.Unlock()
//
//	// turns on response expectation in the listener
//
//	ch.expectedPath <- path
//
//	err = ch.conn.SetWriteDeadline(time.Now().Add(reqTimeout))
//	if err != nil {
//		return nil, errs.Wrap(err, "failed to set write deadline for connection")
//	}
//
//	_, err = ch.conn.Write(req)
//	if err != nil {
//		cerr := c.close(conf)
//		if cerr != nil {
//			c.logr.Errf("write connection clean-up failed, err: %w", cerr)
//		}
//
//		return nil, errs.Wrap(err, "failed to write to connection")
//	}
//
//	data := <-ch.parsedData
//	if data.err != nil {
//		return nil, errs.Wrapf(ember.ErrElementNotFound, "failed to find element, err %s", data.err.Error())
//	}
//
//	return data.element, nil
//}

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

func (c *ConnCollection) Get(timeout time.Duration, conf ConnConfig, el *emberlib.EmberLib) (net.Conn, error) {
	ch, err := c.get(timeout, conf, el)
	if err != nil {
		return nil, errs.Wrap(err, "failed to get conn")
	}

	return ch.conn, nil
}

func (c *ConnCollection) get(timeout time.Duration, conf ConnConfig, el *emberlib.EmberLib) (*connHandler, error) {
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

	err = c.setConn(conf, ch)
	if err != nil {
		defer ch.conn.Close() //nolint:errcheck

		c.logr.Debugf("closed redundant connection %s, %s", conf.URI, err.Error())

		existing := c.getConn(conf)
		if existing == nil {
			return nil, errs.New("failed to get existing connection handler")
		}

		return existing, nil
	}

	go ch.pathReader(c, timeout, el)

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
// Caches the connections, and returns an error if it already exists.
func (c *ConnCollection) setConn(cc ConnConfig, ch *connHandler) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, ok := c.conns[cc]
	if ok {
		return ErrConnectionSet
	}

	c.conns[cc] = ch

	return nil
}

//nolint:cyclop
func (ch *connHandler) read() ([]byte, error) {
	var (
		s101s          [][]byte
		incompleteS101 []byte
		out            []byte
		multi          bool
	)

	for {
		//nolint:makezero
		// length taken from Ember+ documentation
		response := make([]byte, 1290)

		n, err := ch.conn.Read(response)
		if err != nil {
			return nil, errs.Wrap(err, "failed to read from connection")
		}

		if len(incompleteS101) > 0 {
			response = append(incompleteS101, response[:n]...)
		}

		s101s, incompleteS101, err = s101.GetS101s(response)
		if err != nil {
			return nil, errs.Wrap(err, "failed to get s101 data from read")
		}

		if len(incompleteS101) > 0 {
			continue
		}

		glow, lastPacketType, err := s101.Decode(s101s)
		if err != nil {
			ch.logr.Debugf("failed to decode response: %s", err.Error())

			continue
		}

		ch.logr.Tracef("got packet with last packet type %x and data %x", lastPacketType, response)

		switch lastPacketType {
		case s101.FirstMultiPacket, s101.BodyMultiPacket:
			out = append(out, glow...)
			multi = true

			continue
		case s101.LastMultiPacket:
			out = append(out, glow...)

			return out, nil
		default:
			if multi {
				ch.logr.Errf("dropping message in the middle of a multi packet read %x", glow)

				continue
			}

			return glow, nil
		}
	}
}

//nolint:cyclop
func (ch *connHandler) readNew(lib *emberlib.EmberLib) (ember.ElementCollection, error) {

	for {
		//nolint:makezero
		// length taken from Ember+ documentation
		response := make([]byte, 1290)

		n, err := ch.conn.Read(response)
		if err != nil {
			return nil, errs.Wrap(err, "failed to read from connection")
		}

		stop := lib.EmberRead(response, n)
		if stop {
			break
		}
	}

	el := lib.GetFromState()

	return el, nil
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

func (ch *connHandler) pathReader(c *ConnCollection, timeout time.Duration, el *emberlib.EmberLib) {
	go ch.reader(c, el)

	for {
		select {
		case path := <-ch.expectedPath:
			ch.logr.Tracef("got path for request %s", path)
			ch.parsedData <- ch.readExpected(path, timeout)
		case resp, ok := <-ch.readData:
			if !ok {
				// incase we get an error in readExpected, then we will exit this function here. As ch.reader will be
				// stopped and ch.readData chan will be closed.
				return
			}

			if resp.err != nil {
				ch.logr.Debugf("stopping pathReader for connection %s, err: %s", ch.conf.URI, resp.err.Error())

				return
			}

			ch.logr.Tracef("got not requested ember+ plus data, skipping")
		}
	}
}

func (ch *connHandler) reader(c *ConnCollection, el *emberlib.EmberLib) {
	for {

		emberCollection, err := ch.readNew(el)

		ch.readData <- readResponse{emberCollection, err}

		if err != nil {
			ch.logr.Debugf("stopping reader for connection %s, err: %s", ch.conf.URI, err.Error())

			cerr := c.close(ch.conf)
			if cerr != nil {
				ch.logr.Errf("reader connection clean-up failed, err: %w", cerr)
			}

			close(ch.readData)

			return
		}
	}
}

func (ch *connHandler) readExpected(path string, timeout time.Duration) parsedResponse {
	t := time.NewTimer(timeout)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			ch.logr.Debugf("failed to find Ember+ response in time for request with path %s", path)

			return parsedResponse{nil, errs.New("failed to find Ember+ response in time")}
		case resp := <-ch.readData:
			if resp.err != nil {
				ch.logr.Debugf("stopping reader for connection %s, err: %s", ch.conf.URI, resp.err.Error())

				return parsedResponse{nil, errs.Wrapf(resp.err, "failed to read Ember+ response")}
			}

			parsed := filter(resp.collection, path)

			gotPath, err := ch.getPath(parsed)
			if err != nil {
				ch.logr.Debugf("failed to read glow response: %s", err.Error())

				continue
			}

			if !ch.expectedData(path, gotPath) {
				continue
			}

			ch.logr.Tracef("found expected response with path %s", path)

			return parsedResponse{parsed, nil}
		}
	}
}

func (ch *connHandler) getPath(el ember.ElementCollection) ([]string, error) {
	if len(el) == 0 {
		return nil, errs.New("empty collection")
	}

	var gotPath []string

	for k := range el {
		// we care only about the path from the one element as it's a control value and every other element
		// should have the same path prefix
		gotPath = strings.Split(k.Path, ".")
		ch.logr.Tracef("path from first element %s", gotPath)

		break
	}

	return gotPath, nil
}

func (ch *connHandler) getCollection(glow []byte) (ember.ElementCollection, []string, error) {
	el := ember.NewElementCollection()

	err := el.Populate(asn1.NewDecoder(glow))
	if err != nil {
		return ember.ElementCollection{}, nil, errs.Errorf("failed to populate glow response: %s", err.Error())
	}

	if len(el) == 0 {
		return ember.ElementCollection{}, nil, errs.New("empty collection")
	}

	var gotPath []string

	ch.logr.Tracef("got collection, %+v", el)

	for k := range el {
		// we care only about the path from the one element as it's a control value and every other element
		// should have the same path prefix
		gotPath = strings.Split(k.Path, ".")
		ch.logr.Tracef("path from first element %s", gotPath)

		break
	}

	return el, gotPath, nil
}

func (ch *connHandler) expectedData(expectedPath string, incomingPath []string) bool {
	splitExpectedPath, expectedLength := parseExpectedLength(expectedPath)
	// gotPath has to be one path element longer or the same length in case data has children and not values
	if len(incomingPath) != expectedLength && len(incomingPath) != expectedLength+1 {
		ch.logr.Tracef(
			"path %s length %d does not match the expected path %s length %d",
			incomingPath, len(incomingPath), splitExpectedPath, expectedLength,
		)

		return false
	}

	for i, v := range splitExpectedPath {
		if incomingPath[i] != v {
			ch.logr.Tracef("path %s does not match the expected %s", incomingPath, splitExpectedPath)

			return false
		}
	}

	return true
}

func parseExpectedLength(path string) ([]string, int) {
	if path == "" {
		return nil, 1
	}

	splitExpectedPath := strings.Split(path, ".")

	return splitExpectedPath, len(splitExpectedPath)
}

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

	return &connHandler{
		conn:           conn,
		lastAccessTime: time.Now(),
		logr:           logger,
		conf:           conf,
		readData:       make(chan readResponse),
		parsedData:     make(chan parsedResponse),
		expectedPath:   make(chan string),
	}, nil
}

func filter(el ember.ElementCollection, path string) ember.ElementCollection {
	if path == "" {
		return el
	}

	out := make(ember.ElementCollection)
	for k, v := range el {
		if strings.HasPrefix(k.Path, path+".") || k.Path == path {
			out[k] = v
		}
	}

	return out
}
