/*
** Zabbix
** Copyright (C) 2001-2026 Zabbix SIA
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
)
