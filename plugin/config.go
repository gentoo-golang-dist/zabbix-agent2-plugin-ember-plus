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

package plugin

import (
	"golang.zabbix.com/sdk/conf"
	"golang.zabbix.com/sdk/errs"
	"golang.zabbix.com/sdk/log"
	"golang.zabbix.com/sdk/plugin"
)

type session struct {
	URI string `conf:"name=Uri,optional"`

	// json tag is a temporary workaround until metric.SetDefaults() supports integers
	ConnectionTimeout int `conf:"optional,range=1:30" json:"ConnectionTimeout,string"` //nolint:tagalign,tagliatelle
}

type pluginConfig struct {
	//nolint:staticcheck
	System plugin.SystemOptions `conf:"optional"`
	// LegacyTimeout timeout used for items.
	//
	// Deprecated: LegacyTimeout old timeout value kept for compatibility.
	LegacyTimeout int `conf:"name=Timeout,optional,range=1:30"`
	// KeepAlive is a time to wait before unused connections will be closed.
	KeepAlive int `conf:"optional,range=60:900,default=300"`
	// Sessions stores pre-defined named sets of connections settings.
	Sessions map[string]session `conf:"optional"`
	// Default stores default connection parameter values from configuration
	// file.
	Default session `conf:"optional"`
}

// Configure implements the Configurator interface.
// Initializes configuration structures.
func (p *EmberPlugin) Configure(global *plugin.GlobalOptions, options any) {
	pConfig := &pluginConfig{}

	err := conf.UnmarshalStrict(options, pConfig)
	if err != nil {
		p.Errf("cannot unmarshal configuration options: %s", err.Error())

		return
	}

	p.config = pConfig

	if p.config.LegacyTimeout != 0 {
		log.Debugf("config value 'Plugins.EmberPlus.Timeout' is deprecated." +
			"Use 'Plugins.EmberPlus.Default.ConnectionTimeout' instead")

		if p.config.Default.ConnectionTimeout == 0 {
			p.config.Default.ConnectionTimeout = p.config.LegacyTimeout
		}
	}

	if p.config.Default.ConnectionTimeout == 0 {
		p.config.Default.ConnectionTimeout = global.Timeout
	}

	if p.config.LegacyTimeout == 0 {
		p.config.LegacyTimeout = global.Timeout
	}
}

// Validate implements the Configurator interface.
// Returns an error if validation of a plugin's configuration is failed.
func (*EmberPlugin) Validate(options any) error {
	var opts pluginConfig

	err := conf.UnmarshalStrict(options, &opts)
	if err != nil {
		return errs.Wrap(err, "failed to unmarshal configuration options")
	}

	return nil
}
