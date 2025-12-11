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

package ember

import (
	"strconv"
	"strings"

	"golang.zabbix.com/sdk/errs"
)

const (
	TypeInt = iota + 1
	TypeReal
	TypeString
	TypeBool
	TypeTrigger
	TypeEnum
	TypeOctets
)

// ErrElementNotFound error when element is not found.
var ErrElementNotFound = errs.New("element not found")

// Element contains all the values a glow element might contain.
type Element struct {
	Path              string
	ElementType       string
	Identifier        string
	Description       string
	SchemaIdentifiers string
	Children          []*Element
	IsOnline          bool
	IsRoot            bool
	Maximum           any
	Minimum           any
	Value             any
	Access            int
	Format            string
	Enumeration       string
	Factor            int
	Default           any
	StreamValue       any
	StreamIdentifier  int
	ValueType         int
	Number            int
}

// node hold information about node and qualified node parameter fields.
type node struct {
	Path              string `json:"path"`
	ElementType       string `json:"element_type"`
	Identifier        string `json:"identifier"`
	Description       string `json:"description"`
	SchemaIdentifiers string `json:"schema_identifiers"`
	IsOnline          bool   `json:"is_online"`
	IsRoot            bool   `json:"is_root"`
}

// function hold information about function parameter fields.
type function struct {
	Path        string     `json:"path"`
	ElementType string     `json:"element_type"`
	Children    []*Element `json:"children"`
	Identifier  string     `json:"identifier"`
	Description string     `json:"description"`
}

type matrix struct {
	Path        string `json:"path"`
	ElementType string `json:"element_type"`
	Identifier  string `json:"identifier"`
	Description string `json:"description"`
}

type command struct {
	Path        string `json:"path"`
	ElementType string `json:"element_type"`
	Number      int    `json:"number"`
}

type stream struct {
	Path             string `json:"path"`
	ElementType      string `json:"element_type"`
	Identifier int    `json:"stream_identifier"`
	Value      any    `json:"stream_value"`
	ValueType        int    `json:"type,omitempty"`
}

// parameter hold information about parameter and qualified parameter fields.
type parameter struct {
	Path        string `json:"path"`
	ElementType string `json:"element_type"`
	Identifier  string `json:"identifier,omitempty"`
	Description string `json:"description,omitempty"`
	Value       any    `json:"value,omitempty"`
	Minimum     any    `json:"minimum,omitempty"`
	Maximum     any    `json:"maximum,omitempty"`
	Access      int    `json:"access,omitempty"`
	Format      string `json:"format,omitempty"`
	Enumeration string `json:"enumeration,omitempty"`
	Factor      int    `json:"factor,omitempty"`
	IsOnline    bool   `json:"is_online,omitempty"`
	Default     any    `json:"default,omitempty"`
	ValueType   int    `json:"type,omitempty"`
}

// parsePath returns string oid path as integer array.
func parsePath(path string) ([]int, error) {
	if path == "" {
		return nil, nil
	}

	paths := strings.Split(path, ".")
	out := make([]int, 0, len(paths))

	for _, p := range paths {
		i, err := strconv.Atoi(p)
		if err != nil {
			return nil, errs.Wrap(err, "failed to parse path component")
		}

		out = append(out, i)
	}

	return out, nil
}
