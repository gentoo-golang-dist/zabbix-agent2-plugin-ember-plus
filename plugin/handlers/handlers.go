package handlers

import (
	"net"
)

// HandlerFunc describes the signature all metric handler functions must have.
type HandlerFunc func(metricParams map[string]string, extraParams ...string) (any, error)

// ConnHandlerFunc describes the signature all connection handler functions
// must have.
type ConnHandlerFunc func(conn net.Conn, metricParams map[string]string, extraParams ...string) (any, error)
