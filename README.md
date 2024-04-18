# Ember+ plugin for Zabbix agent 2

This plugin provides a native Zabbix solution to monitor devices that support the ember+ protocol.

It can monitor multiple ember+ devices simultaneously, remote or local.


## Requirements

- Zabbix Agent 2 version 6.0.0 or Zabbix Agent 2 version 7.0.0 or newer
- Go programming language version 1.20 or newer (required only to build the
  plugin from source)

## Supported Operating Systems and Architectures

The plugin will work on all operating systems and architectures that the Go
programming language and Zabbix agent 2 supports.

## Installation

Plugin can be compiled using `go build`. However on unix based systems it suggested to use `make` and on Windows based
systems it is suggested to use `mingw32-make`, but it requires `windres.exe`

## Setup

Set `Plugins.EmberPlus.System.Path` setting in Zabbix agent 2 configuration file
with the path to the EmberPlus plugin executable.

We recommend creating a `ember.conf` and placing all plugin related
configurations there. Then import the plugin configuration file in Zabbix agent
2 configuration file - `zabbix_agent2.conf`.

Add the following setting to the EmberPlus plugin configuration file `ember.conf`:

```conf
Plugins.EmberPlus.System.Path=/path/to/executable/ember
```

To import the plugin configuration file in Zabbix agent 2 add the following line
to Zabbix agent 2 configuration file - `zabbix_agent2.conf`

```conf
Include=/path/to/config/ember.conf
```

This is the bare minimum required to get the plugin running. More information
about available configuration settings is available in the section -
Configuration options

## Command line options

The EmberPlus plugin is not intended to be used as a command line utility, however
it does provide the following command line options.

- `-h`, `--help` display a help message
- `-V`, `--version` prints program version and license information

## Ember+ device requirements

Tested with 2.50 version of ember+ protocol

## Connection configuration

To gather monitoring data the plugin needs to establish a connection to a device that supports ember + protocol.
A connection can be configured in two ways. Read more about each
connection configuration option in the following sections.

### In metric key parameters

The metric that the plugin provides has parameters for connection
configuration.

```
ember.get[localhost:9998,1.2.3]
```

## As a named session

Named sessions allow grouping ember connection settings under a name. Define
named session configuration parameters the following way:

```conf
Plugins.EmberPlus.Sessions.StagingEnv.Uri=192.168.1.1
Plugins.EmberPlus.Sessions.TestEnv.Uri=127.0.0.1
```

The example above defines a session named `StagingEnv` and `TestEnv`. The session then can be
used as the first parameter to a metric key `ember.get[StagingEnv]` and `ember.get[TestEnv]` as
opposed to defining each parameter separately
`ember.get[192.168.1.1,1.2.3]` or `ember.get[127.0.0.1,1.2.3]`

## Configuration options

### Plugin settings

Global setting for the Ember plugin. Applied to all connections.

#### `Plugins.EmberPlus.System.Path`

Path to the EmberPlus plugin executable.

Example usage:

```conf
Plugins.EmberPlus.System.Path=/usr/sbin/zabbix-agent2-plugin/zabbix-agent2-plugin-ember
```

#### `Plugins.EmberPlus.Timeout`

Specifies the amount of time to wait for a device to respond when first
connecting and on follow-up operations in the session. Range: 1-30 in seconds.
If not specified, the value defaults to global timeout value defined in agent 2
configuration.

Example usage:

```conf
Plugins.EmberPlus.Timeout=10
```

#### `Plugins.EmberPlus.KeepAlive`

Specifies the time in seconds for waiting before unused connections will be
closed. Range: 60-900 in seconds. The default value is 300 (seconds).

Example usage:

```conf
Plugins.EmberPlus.KeepAlive=600
```

### Session settings

For following session config options, the `*` symbol in the field name implies a
session name. Replace `*` with the actual (like `production` or `stage`) session
name.

#### `Plugins.EmberPlus.Sessions.*.Uri`

Specifies the URI to connect, for session `*`. The only supported schema is
`tcp`. Embedded credentials will be ignored.

Default: `localhost:9998`

Example usage:

```conf
Plugins.EmberPlus.Sessions.exampleSession.Uri=localhost:9998
```

### Default settings

`Plugins.EmberPlus.Default.*` fields define the default values, that will be used if
no other value is specified. (The `*` symbol implies a specific config field, currently only `Uri` is available)

#### `Plugins.EmberPlus.Default.Uri`

Specifies the default URI to connect. The only supported schema is `tcp`.
Embedded credentials will be ignored.

Default: `localhost:9998`

Example usage:

```conf
Plugins.EmberPlus.Default.Uri=localhost:9998
```

## Metric keys

### `ember.get[<uri>,<path>]`

Returns the result of the required device.

`<uri>` - Ember+ device URI. Default: 127.0.0.1:9998
`<path>` - oid path to desired device. Default <empty>, if left empty returns root collection data.

## Troubleshooting

The plugin sends all of its logs to Zabbix agent 2, that further logs them where
ever agent 2 log location is configured to.

For debugging Zabbix Agent 2 log level setting can be increased either in config
by field `DebugLevel` or by runtime control by running

```sh
zabbix_agent2 -R log_level_increase
```

For more information about Zabbix agent 2 view
[Zabbix documentation](https://www.zabbix.com/documentation/current/en/manual/concepts/agent2).

## Contributing

Noticed a bug or have an idea for improvement? Feel free to open an issue or a
feature request in
[Zabbix support system](https://support.zabbix.com/secure/Dashboard.jspa)

Want to contribute? Pull requests are welcome!
