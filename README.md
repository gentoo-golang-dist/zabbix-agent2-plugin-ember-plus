# Ember+ plugin for Zabbix agent 2

This plugin provides a native Zabbix solution to monitor devices that support the Ember+ protocol.

The plugin can monitor multiple Ember+ devices simultaneously, remote or local.

## Requirements

- Zabbix Agent 2 version 6.0.0, Zabbix Agent 2 version 7.0.0, or newer
- Go programming language version 1.20 or newer (required only to build the plugin from the source)

## Supported operating systems and architectures

The plugin works on all operating systems and architectures that the Go programming language and Zabbix agent 2 supports.

## Installation

The plugin can be compiled using `go build`. However, on Unix-based systems, it is suggested to use `make`;
on Windows-based systems, while it is suggested to use `mingw32-make`, it is required to use `windres.exe`.

## Setup

Set the `Plugins.EmberPlus.System.Path` setting in the Zabbix agent 2 configuration file with the path to the Ember+ plugin executable.

We recommend creating an `ember.conf` file and placing all plugin-related configurations there.
Then, import the plugin configuration file in the Zabbix agent 2 configuration file - `zabbix_agent2.conf`.

Add the following setting to the Ember+ plugin configuration file `ember.conf`:

```conf
Plugins.EmberPlus.System.Path=/path/to/executable/ember
```

To import the plugin configuration file in Zabbix agent 2, add the following line to `zabbix_agent2.conf`:

```conf
Include=/path/to/config/ember.conf
```

This is the bare minimum required to get the plugin running.
More information about available configuration settings is available below, under *Configuration options*.

## Command line options

The Ember+ plugin is not intended to be used as a command line utility; however, it does provide the following command line options.

- `-h`, `--help` - displays a help message
- `-V`, `--version` - prints the program version and license information

## Ember+ device requirements

Tested with version 2.50 of the Ember+ protocol

## Connection configuration

To gather monitoring data, the plugin needs to establish a connection to a device that supports the Ember+ protocol.
A connection can be configured in two ways - read more about each in the following sections.

### In metric key parameters

The metric below, provided by the plugin, has parameters for connection configuration.

```
ember.get[localhost:9998,1.2.3]
```

### As a named session

Named sessions allow grouping Ember+ connection settings under a name. Define the named session configuration parameters like so:

```conf
Plugins.EmberPlus.Sessions.StagingEnv.Uri=192.168.1.1
Plugins.EmberPlus.Sessions.TestEnv.Uri=127.0.0.1
```

The example above defines sessions named `StagingEnv` and `TestEnv`.
The sessions can then be used as the first parameter to a metric key (`ember.get[StagingEnv]` and `ember.get[TestEnv]`)
as opposed to defining each parameter separately (`ember.get[192.168.1.1,1.2.3]` and `ember.get[127.0.0.1,1.2.3]`).

## Configuration options

### Plugin settings

Below are the global settings for the Ember+ plugin. Applicable to all connections.

#### `Plugins.EmberPlus.System.Path`

Path to the Ember+ plugin executable.

Example:

```conf
Plugins.EmberPlus.System.Path=/usr/sbin/zabbix-agent2-plugin/zabbix-agent2-plugin-ember
```

#### `Plugins.EmberPlus.Timeout`

Specifies the wait time (in seconds) for a device to respond when first connecting and on follow-up operations in the session.
Global item-type timeout (or individual item timeout) will override this value if it is smaller.
Range: 1-30 seconds. If not specified, the value defaults to a global timeout value defined in Zabbix agent 2 configuration file.

Example:

```conf
Plugins.EmberPlus.Timeout=10
```

#### `Plugins.EmberPlus.KeepAlive`

Specifies the wait time (in seconds) before unused connections are closed. Range: 60-900 seconds. The default value is 300.

Example:

```conf
Plugins.EmberPlus.KeepAlive=600
```

### Session settings

For the following session configuration option, the `*` symbol in the field name implies a session name.
Replace `*` with the actual session name (for example, `StagingEnv` or `TestEnv`).

#### `Plugins.EmberPlus.Sessions.*.Uri`

Specifies the URI to connect for session `*`. The only supported schema is `tcp`. Embedded credentials will be ignored.

Default: `localhost:9998`

Example:

```conf
Plugins.EmberPlus.Sessions.exampleSession.Uri=localhost:9998
```

### Default settings

The `Plugins.EmberPlus.Default.*` fields define the default values that will be used if no other value is specified.
The `*` symbol implies a specific configuration field, currently only `Uri` is available.

#### `Plugins.EmberPlus.Default.Uri`

Specifies the default URI to connect. The only supported schema is `tcp`. Embedded credentials will be ignored.

Default: `localhost:9998`

Example:

```conf
Plugins.EmberPlus.Default.Uri=localhost:9998
```

## Metric keys

### `ember.get[<uri>,<path>]`

Returns the result of the required device.

`<uri>` - Ember+ device URI. Default: 127.0.0.1:9998

`<path>` - OID path to the desired device. The default is empty; if left empty, root collection data is returned.

## Troubleshooting

The plugin sends all of its logs to Zabbix agent 2, which further logs them wherever the agent 2 log location is configured to.

For debugging Zabbix Agent 2, the log level setting can be increased either in the configuration file field `DebugLevel` or by runtime control, running:

```sh
zabbix_agent2 -R log_level_increase
```

For more information about Zabbix agent 2, see [Zabbix documentation](https://www.zabbix.com/documentation/current/en/manual/concepts/agent2).

## Contributing

Noticed a bug or have an idea for improvement?
Feel free to open an issue or a feature request in the [Zabbix support system](https://support.zabbix.com/secure/Dashboard.jspa).

Want to contribute? Pull requests are welcome!
