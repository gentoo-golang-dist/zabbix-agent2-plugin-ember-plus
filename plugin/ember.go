package plugin

import (
	"git.zabbix.com/ap/plugin-support/plugin"
)

const (
	Name       = "EmberPlus"
	hkInterval = 10
)

var Impl Plugin

// Plugin -
type Plugin struct {
	plugin.Base
}

func (p *Plugin) Export(key string, rawParams []string, ctx plugin.ContextProvider) (result interface{}, err error) {
	return nil, nil
}
