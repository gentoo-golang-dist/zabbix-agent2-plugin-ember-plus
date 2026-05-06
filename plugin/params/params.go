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

package params

import (
	"golang.zabbix.com/sdk/metric"
)

//nolint:gochecknoglobals // global constants.
var (

	// URI is a metric param that specifies database connection URI.
	URI = metric.NewConnParam(
		"URI", "URL connection string to connect to the database.",
	).
		WithDefault("localhost:9998").
		WithSession()

	// Path to require ember collection, node or parameter.
	Path = metric.NewParam("Path", "Path to require ember collection, node or parameter").WithDefault("")

	// ConnTimeout the amount of time for Ember+ connection to be created.
	ConnTimeout = metric.NewSessionOnlyParam("ConnectionTimeout", "Connection timeout for Ember+ connections.")
)
