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

package main

import (
	"errors"
	"fmt"
	"os"

	"golang.zabbix.com/plugin/ember-plus/plugin"
	"golang.zabbix.com/sdk/plugin/flag"
	"golang.zabbix.com/sdk/zbxerr"
)

const copyrightMessage = //
`Copyright 2001-%d Zabbix SIA
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
  http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.`

//nolint:revive
const (
	PLUGIN_VERSION_MAJOR = 6
	PLUGIN_VERSION_MINOR = 0
	PLUGIN_VERSION_PATCH = 33
	PLUGIN_LICENSE_YEAR  = 2024
	PLUGIN_VERSION_RC    = "rc1"
)

func main() {
	err := flag.HandleFlags(
		plugin.Name,
		os.Args[0],
		fmt.Sprintf(copyrightMessage, PLUGIN_LICENSE_YEAR),
		PLUGIN_VERSION_RC,
		PLUGIN_VERSION_MAJOR,
		PLUGIN_VERSION_MINOR,
		PLUGIN_VERSION_PATCH,
	)
	if err != nil {
		if errors.Is(err, zbxerr.ErrorOSExitZero) {
			return
		}

		panic(err)
	}

	err = plugin.Launch()
	if err != nil {
		panic(err)
	}
}
