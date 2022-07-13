# Container Network Interface (CNI) Specification
- [Container Network Interface (CNI) Specification](#container-network-interface-cni-specification)
  - [Version](#version)
      - [Released versions](#released-versions)
  - [Overview](#overview)
  - [Summary](#summary)
  - [Section 1: Network configuration format](#section-1-network-configuration-format)
    - [Configuration format](#configuration-format)
      - [Plugin configuration objects:](#plugin-configuration-objects)
      - [Example configuration](#example-configuration)
  - [Section 2: Execution Protocol](#section-2-execution-protocol)
    - [Overview](#overview-1)
    - [Parameters](#parameters)
    - [Errors](#errors)
    - [CNI operations](#cni-operations)
      - [`ADD`: Add container to network, or apply modifications](#add-add-container-to-network-or-apply-modifications)
      - [`DEL`: Remove container from network, or un-apply modifications](#del-remove-container-from-network-or-un-apply-modifications)
      - [`CHECK`: Check container's networking is as expected](#check-check-containers-networking-is-as-expected)
      - [`VERSION`: probe plugin version support](#version-probe-plugin-version-support)
  - [Section 3: Execution of Network Configurations](#section-3-execution-of-network-configurations)
    - [Lifecycle & Ordering](#lifecycle--ordering)
    - [Attachment Parameters](#attachment-parameters)
    - [Adding an attachment](#adding-an-attachment)
    - [Deleting an attachment](#deleting-an-attachment)
    - [Checking an attachment](#checking-an-attachment)
    - [Deriving execution configuration from plugin configuration](#deriving-execution-configuration-from-plugin-configuration)
      - [Deriving `runtimeConfig`](#deriving-runtimeconfig)
  - [Section 4: Plugin Delegation](#section-4-plugin-delegation)
    - [Delegated Plugin protocol](#delegated-plugin-protocol)
    - [Delegated plugin execution procedure](#delegated-plugin-execution-procedure)
  - [Section 5: Result Types](#section-5-result-types)
    - [Success](#success)
      - [Delegated plugins (IPAM)](#delegated-plugins-ipam)
    - [Error](#error)
    - [Version](#version-1)
  - [Appendix: Examples](#appendix-examples)
    - [Add example](#add-example)
    - [Check example](#check-example)
    - [Delete example](#delete-example)

## Version

This is CNI **spec** version **1.0.0**.

Note that this is **independent from the version of the CNI library and plugins** in this repository (e.g. the versions of [releases](https://github.com/containernetworking/cni/releases)).

#### Released versions

Released versions of the spec are available as Git tags.

| tag                                                                                  | spec permalink                                                                        | major changes                     |
| ------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------- | --------------------------------- |
| [`spec-v1.0.0`](https://github.com/containernetworking/cni/releases/tag/spec-v1.0.0) | [spec at v1.0.0](https://github.com/containernetworking/cni/blob/spec-v1.0.0/SPEC.md) | Removed non-list configurations; removed `version` field of `interfaces` array |
| [`spec-v0.4.0`](https://github.com/containernetworking/cni/releases/tag/spec-v0.4.0) | [spec at v0.4.0](https://github.com/containernetworking/cni/blob/spec-v0.4.0/SPEC.md) | Introduce the CHECK command and passing prevResult on DEL |
| [`spec-v0.3.1`](https://github.com/containernetworking/cni/releases/tag/spec-v0.3.1) | [spec at v0.3.1](https://github.com/containernetworking/cni/blob/spec-v0.3.1/SPEC.md) | none (typo fix only)              |
| [`spec-v0.3.0`](https://github.com/containernetworking/cni/releases/tag/spec-v0.3.0) | [spec at v0.3.0](https://github.com/containernetworking/cni/blob/spec-v0.3.0/SPEC.md) | rich result type, plugin chaining |
| [`spec-v0.2.0`](https://github.com/containernetworking/cni/releases/tag/spec-v0.2.0) | [spec at v0.2.0](https://github.com/containernetworking/cni/blob/spec-v0.2.0/SPEC.md) | VERSION command                   |
| [`spec-v0.1.0`](https://github.com/containernetworking/cni/releases/tag/spec-v0.1.0) | [spec at v0.1.0](https://github.com/containernetworking/cni/blob/spec-v0.1.0/SPEC.md) | initial version                   |

*Do not rely on these tags being stable.  In the future, we may change our mind about which particular commit is the right marker for a given historical spec version.*


## Overview

This document proposes a generic plugin-based networking solution for application containers on Linux, the _Container Networking Interface_, or _CNI_.

For the purposes of this proposal, we define three terms very specifically:
- _container_ is a network isolation domain, though the actual isolation technology is not defined by the specification. This could be a [network namespace][namespaces] or a virtual machine, for example.
- _network_ refers to a group of endpoints that are uniquely addressable that can communicate amongst each other. This could be either an individual container (as specified above), a machine, or some other network device (e.g. a router). Containers can be conceptually _added to_ or _removed from_ one or more networks.
- _runtime_ is the program responsible for executing CNI plugins.
- _plugin_ is a program that applies a specified network configuration.

This document aims to specify the interface between "runtimes" and "plugins". The key words "must", "must not", "required", "shall", "shall not", "should", "should not", "recommended", "may" and "optional" are used as specified in [RFC 2119][rfc-2119].

[namespaces]: http://man7.org/linux/man-pages/man7/namespaces.7.html
[rfc-2119]: https://www.ietf.org/rfc/rfc2119.txt



## Summary

The CNI specification defines:

1. A format for administrators to define network configuration.
2. A protocol for container runtimes to make requests to network plugins.
3. A procedure for executing plugins based on a supplied configuration.
4. A procedure for plugins to delegate functionality to other plugins.
5. Data types for plugins to return their results to the runtime.

## Section 1: Network configuration format

CNI defines a network configuration format for administrators. It contains
directives for both the container runtime as well as the plugins to consume. At
plugin execution time, this configuration format is interpreted by the runtime and
transformed in to a form to be passed to the plugins.

In general, the network configuration is intended to be static. It can conceptually
be thought of as being "on disk", though the CNI specification does not actually
require this.

### Configuration format

A network configuration consists of a JSON object with the following keys:

- `cniVersion` (string): [Semantic Version 2.0](https://semver.org) of CNI specification to which this configuration list and all the individual configurations conform. Currently "1.0.0"
- `name` (string): Network name. This should be unique across all network configurations on a host (or other administrative domain).  Must start with an alphanumeric character, optionally followed by any combination of one or more alphanumeric characters, underscore, dot (.) or hyphen (-).
- `disableCheck` (boolean): Either `true` or `false`.  If `disableCheck` is `true`, runtimes must not call `CHECK` for this network configuration list.  This allows an administrator to prevent `CHECK`ing where a combination of plugins is known to return spurious errors.
- `plugins` (list): A list of CNI plugins and their configuration, which is a list of plugin configuration objects.

#### Plugin configuration objects:
Plugin configuration objects may contain additional fields than the ones defined here.
The runtime MUST pass through these fields, unchanged, to the plugin, as defined in section 3.

**Required keys:**
- `type` (string): Matches the name of the CNI plugin binary on disk. Must not contain characters disallowed in file paths for the system (e.g. / or \\).

**Optional keys, used by the protocol:**
- `capabilities` (dictionary): Defined in [section 3](#Deriving-runtimeConfig)

**Reserved keys, used by the protocol:**
These keys are generated by the runtime at execution time, and thus should not be used in configuration.
- `runtimeConfig`
- `args`
- Any keys starting with `cni.dev/`

**Optional keys, well-known:**
These keys are not used by the protocol, but have a standard meaning to plugins.
Plugins that consume any of these configuration keys should respect their intended semantics.

- `ipMasq` (boolean): If supported by the plugin, sets up an IP masquerade on the host for this network. This is necessary if the host will act as a gateway to subnets that are not able to route to the IP assigned to the container.
- `ipam` (dictionary): Dictionary with IPAM (IP Address Management) specific values:
    - `type` (string): Refers to the filename of the IPAM plugin executable. Must not contain characters disallowed in file paths for the system (e.g. / or \\).
- `dns` (dictionary, optional): Dictionary with DNS specific values:
    - `nameservers` (list of strings, optional): list of a priority-ordered list of DNS nameservers that this network is aware of. Each entry in the list is a string containing either an IPv4 or an IPv6 address.
    - `domain` (string, optional): the local domain used for short hostname lookups.
    - `search` (list of strings, optional): list of priority ordered search domains for short hostname lookups. Will be preferred over `domain` by most resolvers.
    - `options` (list of strings, optional): list of options that can be passed to the resolver

**Other keys:**
Plugins may define additional fields that they accept and may generate an error if called with unknown fields. Runtimes must preserve unknown fields in plugin configuration objects when transforming for execution.

#### Example configuration
```jsonc
{
  "cniVersion": "1.0.0",
  "name": "dbnet",
  "plugins": [
    {
      "type": "bridge",
      // plugin specific parameters
      "bridge": "cni0",
      "keyA": ["some more", "plugin specific", "configuration"],
      
      "ipam": {
        "type": "host-local",
        // ipam specific
        "subnet": "10.1.0.0/16",
        "gateway": "10.1.0.1",
        "routes": [
            {"dst": "0.0.0.0/0"}
        ]
      },
      "dns": {
        "nameservers": [ "10.1.0.1" ]
      }
    },
    {
      "type": "tuning",
      "capabilities": {
        "mac": true
      },
      "sysctl": {
        "net.core.somaxconn": "500"
      }
    },
    {
        "type": "portmap",
        "capabilities": {"portMappings": true}
    }
  ]
}
```

## Section 2: Execution Protocol

### Overview

The CNI protocol is based on execution of binaries invoked by the container runtime. CNI defines the protocol between the plugin binary and the runtime.

A CNI plugin is responsible for configuring a container's network interface in some manner. Plugins fall in to two broad categories:
* "Interface" plugins, which create a network interface inside the container and ensure it has connectivity.
* "Chained" plugins, which adjust the configuration of an already-created interface (but may need to create more interfaces to do so).

The runtime passes parameters to the plugin via environment variables and configuration. It supplies configuration via stdin. The plugin returns
a [result](#Section-5-Result-Types) on stdout on success, or an error on stderr if the operation fails. Configuration and results are encoded in JSON.

Parameters define invocation-specific settings, whereas configuration is, with some exceptions, the same for any given network.

The runtime must execute the plugin in the runtime's networking domain. (For most cases, this means the root network namespace / `dom0`).

### Parameters

Protocol parameters are passed to the plugins via OS environment variables.

- `CNI_COMMAND`: indicates the desired operation; `ADD`, `DEL`, `CHECK`, or `VERSION`.
- `CNI_CONTAINERID`: Container ID. A unique plaintext identifier for a container, allocated by the runtime. Must not be empty.  Must start with an alphanumeric character, optionally followed by any combination of one or more alphanumeric characters, underscore (), dot (.) or hyphen (-).
- `CNI_NETNS`: A reference to the container's "isolation domain". If using network namespaces, then a path to the network namespace (e.g. `/run/netns/[nsname]`)
- `CNI_IFNAME`: Name of the interface to create inside the container; if the plugin is unable to use this interface name it must return an error.
- `CNI_ARGS`: Extra arguments passed in by the user at invocation time. Alphanumeric key-value pairs separated by semicolons; for example, "FOO=BAR;ABC=123"
- `CNI_PATH`: List of paths to search for CNI plugin executables. Paths are separated by an OS-specific list separator; for example ':' on Linux and ';' on Windows

### Errors
A plugin must exit with a return code of 0 on success, and non-zero on failure. If the plugin encounters an error, it should output an ["error" result structure](#Error) (see below).

### CNI operations

CNI defines 4 operations: `ADD`, `DEL`, `CHECK`, and `VERSION`. These are passed to the plugin via the `CNI_COMMAND` environment variable.

#### `ADD`: Add container to network, or apply modifications

A CNI plugin, upon receiving an `ADD` command, should either
- create the interface defined by `CNI_IFNAME` inside the container at `CNI_NETNS`, or
- adjust the configuration of the interface defined by `CNI_IFNAME` inside the container at `CNI_NETNS`.

If the CNI plugin is successful, it must output a [result structure](#Success) (see below) on standard out. If the plugin was supplied a `prevResult` as part of its input configuration, it MUST handle `prevResult` by either passing it through, or modifying it appropriately.

If an interface of the requested name already exists in the container, the CNI plugin MUST return with an error.

A runtime should not call `ADD` twice (without an intervening DEL) for the same `(CNI_CONTAINERID, CNI_IFNAME)` tuple. This implies that a given container ID may be added to a specific network more than once only if each addition is done with a different interface name.

**Input:**

The runtime will provide a JSON-serialized plugin configuration object (defined below) on standard in.

Required environment parameters:
- `CNI_COMMAND`
- `CNI_CONTAINERID`
- `CNI_NETNS`
- `CNI_IFNAME`

Optional environment parameters:
- `CNI_ARGS`
- `CNI_PATH`

#### `DEL`: Remove container from network, or un-apply modifications

A CNI plugin, upon receiving a `DEL` command, should either
- delete the interface defined by `CNI_IFNAME` inside the container at `CNI_NETNS`, or
- undo any modifications applied in the plugin's `ADD` functionality

Plugins should generally complete a `DEL` action without error even if some resources are missing.  For example, an IPAM plugin should generally release an IP allocation and return success even if the container network namespace no longer exists, unless that network namespace is critical for IPAM management. While DHCP may usually send a 'release' message on the container network interface, since DHCP leases have a lifetime this release action would not be considered critical and no error should be returned if this action fails. For another example, the `bridge` plugin should delegate the DEL action to the IPAM plugin and clean up its own resources even if the container network namespace and/or container network interface no longer exist.

Plugins MUST accept multiple `DEL` calls for the same (`CNI_CONTAINERID`, `CNI_IFNAME`) pair, and return success if the interface in question, or any modifications added, are missing.

**Input:**

The runtime will provide a JSON-serialized plugin configuration object (defined below) on standard in.

Required environment parameters:
- `CNI_COMMAND`
- `CNI_CONTAINERID`
- `CNI_IFNAME`

Optional environment parameters:
- `CNI_NETNS`
- `CNI_ARGS`
- `CNI_PATH`


#### `CHECK`: Check container's networking is as expected
`CHECK` is a way for a runtime to probe the status of an existing container.

Plugin considerations:
- The plugin must consult the `prevResult` to determine the expected interfaces and addresses.
- The plugin must allow for a later chained plugin to have modified networking resources, e.g. routes, on `ADD`.
- The plugin should return an error if a resource included in the CNI Result type (interface, address or route) was created by the plugin, and is listed in `prevResult`, but is missing or in an invalid state.
- The plugin should return an error if other resources not tracked in the Result type such as the following are missing or are in an invalid state:
  - Firewall rules
  - Traffic shaping controls
  - IP reservations
  - External dependencies such as a daemon required for connectivity
  - etc.
- The plugin should return an error if it is aware of a condition where the container is generally unreachable.
- The plugin must handle `CHECK` being called immediately after an `ADD`, and therefore should allow a reasonable convergence delay for any asynchronous resources.
- The plugin should call `CHECK` on any delegated (e.g. IPAM) plugins and pass any errors on to its caller.


Runtime considerations:
- A runtime must not call `CHECK` for a container that has not been `ADD`ed, or has been `DEL`eted after its last `ADD`.
- A runtime must not call `CHECK` if `disableCheck` is set to `true` in the [configuration](#configuration-format).
- A runtime must include a `prevResult` field in the network configuration containing the `Result` of the immediately preceding `ADD` for the container. The runtime may wish to use libcni's support for caching `Result`s.
- A runtime may choose to stop executing `CHECK` for a chain when a plugin returns an error.
- A runtime may execute `CHECK` from immediately after a successful `ADD`, up until the container is `DEL`eted from the network.
- A runtime may assume that a failed `CHECK` means the container is permanently in a misconfigured state.


**Input:**

The runtime will provide a json-serialized plugin configuration object (defined below) on standard in.

Required environment parameters:
- `CNI_COMMAND`
- `CNI_CONTAINERID`
- `CNI_NETNS`
- `CNI_IFNAME`

Optional environment parameters:
- `CNI_ARGS`
- `CNI_PATH`

All parameters, with the exception of `CNI_PATH`, must be the same as the corresponding `ADD` for this container.

#### `VERSION`: probe plugin version support
The plugin should output via standard-out a json-serialized version result object (see below).

**Input:**

A json-serialized object, with the following key:
- `cniVersion`: The version of the protocol in use.

Required environment parameters:
- `CNI_COMMAND`


## Section 3: Execution of Network Configurations

This section describes how a container runtime interprets a network configuration (as defined in section 1) and executes plugins accordingly. A runtime may wish to _add_, _delete_, or _check_ a network configuration in a container. This results in a series of plugin `ADD`, `DELETE`, or `CHECK` executions, correspondingly. This section also defines how a network configuration is transformed and provided to the plugin.

The operation of a network configuration on a container is called an _attachment_. An attachment may be uniquely identified by the `(CNI_CONTAINERID, CNI_IFNAME)` tuple.

### Lifecycle & Ordering

- The container runtime must create a new network namespace for the container before invoking any plugins.
- The container runtime must not invoke parallel operations for the same container, but is allowed to invoke parallel operations for different containers. This includes across multiple attachments.
- Plugins must handle being executed concurrently across different containers. If necessary, they must implement locking on shared resources (e.g. IPAM databases).
- The container runtime must ensure that _add_ is eventually followed by a corresponding _delete_. The only exception is in the event of catastrophic failure, such as node loss. A _delete_ must still be executed even if the _add_ fails.
- _delete_ may be followed by additional _deletes_.
- The network configuration should not change between _add_ and _delete_.
- The network configuration should not change between _attachments_.
- The container runtime is responsible for cleanup of the container's network namespace.

### Attachment Parameters
While a network configuration should not change between _attachments_, there are certain parameters supplied by the container runtime that are per-attachment. They are:

- **Container ID:** A unique plaintext identifier for a container, allocated by the runtime. Must not be empty.  Must start with an alphanumeric character, optionally followed by any combination of one or more alphanumeric characters, underscore (), dot (.) or hyphen (-). During execution, always set as the  `CNI_CONTAINERID` parameter.
- **Namespace**: A reference to the container's "isolation domain". If using network namespaces, then a path to the network namespace (e.g. `/run/netns/[nsname]`). During execution, always set as the `CNI_NETNS` parameter.
- **Container interface name**: Name of the interface to create inside the container. During execution, always set as the `CNI_IFNAME` parameter.
- **Generic Arguments**: Extra arguments, in the form of key-value string pairs, that are relevant to a specific attachment.  During execution, always set as the `CNI_ARGS` parameter.
- **Capability Arguments**: These are also key-value pairs. The key is a string, whereas the value is any JSON-serializable type. The keys and values are defined by [convention](CONVENTIONS.md).

Furthermore, the runtime must be provided a list of paths to search for CNI plugins. This must also be provided to plugins during execution via the `CNI_PATH` environment variable.

### Adding an attachment
For every configuration defined in the `plugins` key of the network configuration,
1. Look up the executable specified in the `type` field. If this does not exist, then this is an error.
2. Derive request configuration from the plugin configuration, with the following parameters:
    - If this is the first plugin in the list, no previous result is provided,
    - For all additional plugins, the previous result is the result of the previous plugins.
3. Execute the plugin binary, with `CNI_COMMAND=ADD`. Provide parameters defined above as environment variables. Supply the derived configuration via standard in.
4. If the plugin returns an error, halt execution and return the error to the caller.

The runtime must store the result returned by the final plugin persistently, as it is required for _check_ and _delete_ operations.

### Deleting an attachment
Deleting a network attachment is much the same as adding, with a few key differences:
- The list of plugins is executed in **reverse order**
- The previous result provided is always the final result of the _add_ operation.

For every plugin defined in the `plugins` key of the network configuration, *in reverse order*,
1. Look up the executable specified in the `type` field. If this does not exist, then this is an error.
2. Derive request configuration from the plugin configuration, with the previous result from the initial _add_ operation.
3. Execute the plugin binary, with `CNI_COMMAND=DEL`. Provide parameters defined above as environment variables. Supply the derived configuration via standard in.
4. If the plugin returns an error, halt execution and return the error to the caller.

If all plugins return success, return success to the caller.

### Checking an attachment
The runtime may also ask every plugin to confirm that a given attachment is still functional. The runtime must use the same attachment parameters as it did for the _add_ operation.

Checking is similar to add with two exceptions:
- the previous result provided is always the final result of the _add_ operation.
- If the network configuration defines `disableCheck`, then always return success to the caller.

For every plugin defined in the `plugins` key of the network configuration,
1. Look up the executable specified in the `type` field. If this does not exist, then this is an error.
2. Derive request configuration from the plugin configuration, with the previous result from the initial _add_ operation.
3. Execute the plugin binary, with `CNI_COMMAND=CHECK`. Provide parameters defined above as environment variables. Supply the derived configuration via standard in.
4. If the plugin returns an error, halt execution and return the error to the caller.

If all plugins return success, return success to the caller.

### Deriving execution configuration from plugin configuration
The network configuration format (which is a list of plugin configurations to execute) must be transformed to a format understood by the plugin (which is a single plugin configuration). This section describes that transformation.

The execution configuration for a single plugin invocation is also JSON. It consists of the plugin configuration, primarily unchanged except for the specified additions and removals.

The following fields must be inserted into the execution configuration by the runtime:
- `cniVersion`: taken from the `cniVersion` field of the network configuration
- `name`: taken from the `name` field of the network configuration
- `runtimeConfig`: A JSON object, consisting of the union of capabilities provided by the plugin and requested by the runtime (more details below)
- `prevResult`: A JSON object, consisting of the result type returned by the "previous" plugin. The meaning of "previous" is defined by the specific operation (_add_, _delete_, or _check_).

The following fields must be **removed** by the runtime:
- `capabilities`

All other fields should be passed through unaltered.

#### Deriving `runtimeConfig`

Whereas CNI_ARGS are provided to all plugins, with no indication if they are going to be consumed, _Capability arguments_ need to be declared explicitly in configuration. The runtime, thus, can determine if a given network configuration supports a specific _capability_. Capabilities are not defined by the specification - rather, they are documented [conventions](CONVENTIONS.md).

As defined in section 1, the plugin configuration includes an optional key, `capabilities`. This example shows a plugin that supports the `portMapping` capability:

```json
{
  "type": "myPlugin",
  "capabilities": {
    "portMappings": true
  }
}
```

The `runtimeConfig` parameter is derived from the `capabilities` in the network configuration and the _capability arguments_ generated by the runtime. Specifically, any capability supported by the plugin configuration and provided by the runtime should be inserted in the `runtimeConfig`.

Thus, the above example could result in the following being passed to the plugin as part of the execution configuration:
```json
{
  "type": "myPlugin",
  "runtimeConfig": {
    "portMappings": [ { "hostPort": 8080, "containerPort": 80, "protocol": "tcp" } ]
  }
  ...
}
```

## Section 4: Plugin Delegation

There are some operations that, for whatever reason, cannot reasonably be implemented as a discrete chained plugin. Rather, a CNI plugin may wish to delegate some functionality to another plugin. One common example of this is IP address management.

As part of its operation, a CNI plugin is expected to assign (and maintain) an IP address to the interface and install any necessary routes relevant for that interface. This gives the CNI plugin great flexibility but also places a large burden on it. Many CNI plugins would need to have the same code to support several IP management schemes that users may desire (e.g. dhcp, host-local). A CNI plugin may choose to delegate IP management to another plugin.

To lessen the burden and make IP management strategy be orthogonal to the type of CNI plugin, we define a third type of plugin -- IP Address Management Plugin (IPAM plugin), as well as a protocol for plugins to delegate functionality to other plugins.

It is however the responsibility of the CNI plugin, rather than the runtime, to invoke the IPAM plugin at the proper moment in its execution. The IPAM plugin must determine the interface IP/subnet, Gateway and Routes and return this information to the "main" plugin to apply. The IPAM plugin may obtain the information via a protocol (e.g. dhcp), data stored on a local filesystem, the "ipam" section of the Network Configuration file, etc.


### Delegated Plugin protocol

Like CNI plugins, delegated plugins are invoked by running an executable. The executable is searched for in a predefined list of paths, indicated to the CNI plugin via `CNI_PATH`. The delegated plugin must receive all the same environment variables that were passed in to the CNI plugin. Just like the CNI plugin, delegated plugins receive the network configuration via stdin and output results via stdout.

Delegated plugins are provided the *complete network configuration* passed to the "upper" plugin. In other words, in the IPAM case, not just the `ipam` section of the configuration.

Success is indicated by a zero return code and a _Success_ result type output to stdout.

### Delegated plugin execution procedure

When a plugin executes a delegated plugin, it should:
- Look up the plugin binary by searching the directories provided in `CNI_PATH` environment variable.
- Execute that plugin with the same environment and configuration that it received.
- Ensure that the delegated plugin's stderr is output to the calling plugin's stderr.

If a plugin is executed with `CNI_COMMAND=CHECK` or `DEL`, it must also execute any delegated plugins. If any of the delegated plugins return error, error should be returned by the upper plugin.

If, on `ADD`, a delegated plugin fails, the "upper" plugin should execute again with `DEL` before returning failure.

## Section 5: Result Types

Plugins can return one of three result types:

- _Success_ (or _Abbreviated Success_)
- _Error_
- _Version_

### Success

Plugins provided a `prevResult` key as part of their request configuration must output it as their result, with any possible modifications made by that plugin included. If a plugin makes no changes that would be reflected in the _Success result_ type, then it must output a result equivalent to the provided `prevResult`.

Plugins must output a JSON object with the following keys upon a successful `ADD` operation:

- `cniVersion`: The same version supplied on input - the string "1.0.0"
- `interfaces`: An array of all interfaces created by the attachment, including any host-level interfaces:
    - `name`: The name of the interface.
    - `mac`: The hardware address of the interface (if applicable).
    - `sandbox`: The isolation domain reference (e.g. path to network namespace) for the interface, or empty if on the host. For interfaces created inside the container, this should be the value passed via `CNI_NETNS`.
- `ips`: IPs assigned by this attachment. Plugins may include IPs assigned external to the container.
    - `address` (string): an IP address in CIDR notation (eg "192.168.1.3/24").
    - `gateway` (string): the default gateway for this subnet, if one exists.
    - `interface` (uint): the index into the `interfaces` list for a [CNI Plugin Result](#result) indicating which interface this IP configuration should be applied to.
- `routes`: Routes created by this attachment:
    - `dst`: The destination of the route, in CIDR notation
    - `gw`: The next hop address. If unset, a value in `gateway` in the `ips` array may be used.
- `dns`: a dictionary consisting of DNS configuration information
    - `nameservers` (list of strings): list of a priority-ordered list of DNS nameservers that this network is aware of. Each entry in the list is a string containing either an IPv4 or an IPv6 address.
    - `domain` (string): the local domain used for short hostname lookups.
    - `search` (list of strings): list of priority ordered search domains for short hostname lookups. Will be preferred over `domain` by most resolvers.
    - `options` (list of strings): list of options that can be passed to the resolver.

#### Delegated plugins (IPAM)
Delegated plugins may omit irrelevant sections.

Delegated IPAM plugins must return an abbreviated _Success_ object. Specifically, it is missing the `interfaces` array, as well as the `interface` entry in `ips`.


### Error

Plugins should output a JSON object with the following keys if they encounter an error:

- `cniVersion`: The same value as provided by the configuration
- `code`: A numeric error code, see below for reserved codes.
- `msg`: A short message characterizing the error.
- `details`: A longer message describing the error.

Example:

```json
{
  "cniVersion": "1.0.0",
  "code": 7,
  "msg": "Invalid Configuration",
  "details": "Network 192.168.0.0/31 too small to allocate from."
}
```

Error codes 0-99 are reserved for well-known errors. Values of 100+ can be freely used for plugin specific errors.


Error Code|Error Description
---|---
 `1`|Incompatible CNI version
 `2`|Unsupported field in network configuration. The error message must contain the key and value of the unsupported field.
 `3`|Container unknown or does not exist. This error implies the runtime does not need to perform any container network cleanup (for example, calling the `DEL` action on the container).
 `4`|Invalid necessary environment variables, like CNI_COMMAND, CNI_CONTAINERID, etc. The error message must contain the names of invalid variables.
 `5`|I/O failure. For example, failed to read network config bytes from stdin.
 `6`|Failed to decode content. For example, failed to unmarshal network config from bytes or failed to decode version info from string.
 `7`|Invalid network config. If some validations on network configs do not pass, this error will be raised.
 `11`|Try again later. If the plugin detects some transient condition that should clear up, it can use this code to notify the runtime it should re-try the operation later.

In addition, stderr can be used for unstructured output such as logs.

### Version

Plugins must output a JSON object with the following keys upon a `VERSION` operation:

- `cniVersion`: The value of `cniVersion` specified on input
- `supportedVersions`: A list of supported specification versions

Example:
```json
{
    "cniVersion": "1.0.0",
    "supportedVersions": [ "0.1.0", "0.2.0", "0.3.0", "0.3.1", "0.4.0", "1.0.0" ]
}
```


## Appendix: Examples
We assume the network configuration [shown above](#Example-configuration) in section 1. For this attachment, the runtime produces `portmap` and `mac` capability args, along with the generic argument "argA=foo".
The examples uses `CNI_IFNAME=eth0`.

### Add example

The container runtime would perform the following steps for the `add` operation.


1) Call the `bridge` plugin with the following JSON, `CNI_COMMAND=ADD`:

```json
{
    "cniVersion": "1.0.0",
    "name": "dbnet",
    "type": "bridge",
    "bridge": "cni0",
    "keyA": ["some more", "plugin specific", "configuration"],
    "ipam": {
        "type": "host-local",
        "subnet": "10.1.0.0/16",
        "gateway": "10.1.0.1"
    },
    "dns": {
        "nameservers": [ "10.1.0.1" ]
    }
}
```

The bridge plugin, as it delegates IPAM to the `host-local` plugin, would execute the `host-local` binary with the exact same input, `CNI_COMMAND=ADD`.

The `host-local` plugin returns the following result:

```json
{
    "ips": [
        {
          "address": "10.1.0.5/16",
          "gateway": "10.1.0.1"
        }
    ],
    "routes": [
      {
        "dst": "0.0.0.0/0"
      }
    ],
    "dns": {
      "nameservers": [ "10.1.0.1" ]
    }
}
```

The bridge plugin returns the following result, configuring the interface according to the delegated IPAM configuration:

```json
{
    "ips": [
        {
          "address": "10.1.0.5/16",
          "gateway": "10.1.0.1",
          "interface": 2
        }
    ],
    "routes": [
      {
        "dst": "0.0.0.0/0"
      }
    ],
    "interfaces": [
        {
            "name": "cni0",
            "mac": "00:11:22:33:44:55"
        },
        {
            "name": "veth3243",
            "mac": "55:44:33:22:11:11"
        },
        {
            "name": "eth0",
            "mac": "99:88:77:66:55:44",
            "sandbox": "/var/run/netns/blue"
        }
    ],
    "dns": {
      "nameservers": [ "10.1.0.1" ]
    }
}
```

2) Next, call the `tuning` plugin, with `CNI_COMMAND=ADD`. Note that `prevResult` is supplied, along with the `mac` capability argument. The request configuration passed is:

```json
{
  "cniVersion": "1.0.0",
  "name": "dbnet",
  "type": "tuning",
  "sysctl": {
    "net.core.somaxconn": "500"
  },
  "runtimeConfig": {
    "mac": "00:11:22:33:44:66"
  },
  "prevResult": {
    "ips": [
        {
          "address": "10.1.0.5/16",
          "gateway": "10.1.0.1",
          "interface": 2
        }
    ],
    "routes": [
      {
        "dst": "0.0.0.0/0"
      }
    ],
    "interfaces": [
        {
            "name": "cni0",
            "mac": "00:11:22:33:44:55"
        },
        {
            "name": "veth3243",
            "mac": "55:44:33:22:11:11"
        },
        {
            "name": "eth0",
            "mac": "99:88:77:66:55:44",
            "sandbox": "/var/run/netns/blue"
        }
    ],
    "dns": {
      "nameservers": [ "10.1.0.1" ]
    }
  }
}
```

The plugin returns the following result. Note that the **mac** has changed.

```json
{
    "ips": [
        {
          "address": "10.1.0.5/16",
          "gateway": "10.1.0.1",
          "interface": 2
        }
    ],
    "routes": [
      {
        "dst": "0.0.0.0/0"
      }
    ],
    "interfaces": [
        {
            "name": "cni0",
            "mac": "00:11:22:33:44:55"
        },
        {
            "name": "veth3243",
            "mac": "55:44:33:22:11:11"
        },
        {
            "name": "eth0",
            "mac": "00:11:22:33:44:66",
            "sandbox": "/var/run/netns/blue"
        }
    ],
    "dns": {
      "nameservers": [ "10.1.0.1" ]
    }
}
```

3) Finally, call the `portmap` plugin, with `CNI_COMMAND=ADD`. Note that `prevResult` matches that returned by `tuning`:

```json
{
  "cniVersion": "1.0.0",
  "name": "dbnet",
  "type": "portmap",
  "runtimeConfig": {
    "portMappings" : [
      { "hostPort": 8080, "containerPort": 80, "protocol": "tcp" }
    ]
  },
  "prevResult": {
    "ips": [
        {
          "address": "10.1.0.5/16",
          "gateway": "10.1.0.1",
          "interface": 2
        }
    ],
    "routes": [
      {
        "dst": "0.0.0.0/0"
      }
    ],
    "interfaces": [
        {
            "name": "cni0",
            "mac": "00:11:22:33:44:55"
        },
        {
            "name": "veth3243",
            "mac": "55:44:33:22:11:11"
        },
        {
            "name": "eth0",
            "mac": "00:11:22:33:44:66",
            "sandbox": "/var/run/netns/blue"
        }
    ],
    "dns": {
      "nameservers": [ "10.1.0.1" ]
    }
  }
}
```

The `portmap` plugin outputs the exact same result as that returned by `bridge`, as the plugin has not modified anything that would change the result (i.e. it only created iptables rules).


### Check example

Given the previous _Add_, the container runtime would perform the following steps for the _Check_ action:

1) First call the `bridge` plugin with the following request configuration, including the `prevResult` field containing the final JSON response from the _Add_ operation, **including the changed mac**. `CNI_COMMAND=CHECK`

```json
{
  "cniVersion": "1.0.0",
  "name": "dbnet",
  "type": "bridge",
  "bridge": "cni0",
  "keyA": ["some more", "plugin specific", "configuration"],
  "ipam": {
    "type": "host-local",
    "subnet": "10.1.0.0/16",
    "gateway": "10.1.0.1"
  },
  "dns": {
    "nameservers": [ "10.1.0.1" ]
  },
  "prevResult": {
    "ips": [
        {
          "address": "10.1.0.5/16",
          "gateway": "10.1.0.1",
          "interface": 2
        }
    ],
    "routes": [
      {
        "dst": "0.0.0.0/0"
      }
    ],
    "interfaces": [
        {
            "name": "cni0",
            "mac": "00:11:22:33:44:55"
        },
        {
            "name": "veth3243",
            "mac": "55:44:33:22:11:11"
        },
        {
            "name": "eth0",
            "mac": "00:11:22:33:44:66",
            "sandbox": "/var/run/netns/blue"
        }
    ],
    "dns": {
      "nameservers": [ "10.1.0.1" ]
    }
  }
}
```

The `bridge` plugin, as it delegates IPAM, calls `host-local`, `CNI_COMMAND=CHECK`. It returns no error.

Assuming the `bridge` plugin is satisfied, it produces no output on standard out and exits with a 0 return code.

2) Next call the `tuning` plugin with the following request configuration:

```json
{
  "cniVersion": "1.0.0",
  "name": "dbnet",
  "type": "tuning",
  "sysctl": {
    "net.core.somaxconn": "500"
  },
  "runtimeConfig": {
    "mac": "00:11:22:33:44:66"
  },
  "prevResult": {
    "ips": [
        {
          "address": "10.1.0.5/16",
          "gateway": "10.1.0.1",
          "interface": 2
        }
    ],
    "routes": [
      {
        "dst": "0.0.0.0/0"
      }
    ],
    "interfaces": [
        {
            "name": "cni0",
            "mac": "00:11:22:33:44:55"
        },
        {
            "name": "veth3243",
            "mac": "55:44:33:22:11:11"
        },
        {
            "name": "eth0",
            "mac": "00:11:22:33:44:66",
            "sandbox": "/var/run/netns/blue"
        }
    ],
    "dns": {
      "nameservers": [ "10.1.0.1" ]
    }
  }
}
```

Likewise, the `tuning` plugin exits indicating success.

3) Finally, call `portmap` with the following request configuration:

```json
{
  "cniVersion": "1.0.0",
  "name": "dbnet",
  "type": "portmap",
  "runtimeConfig": {
    "portMappings" : [
      { "hostPort": 8080, "containerPort": 80, "protocol": "tcp" }
    ]
  },
  "prevResult": {
    "ips": [
        {
          "address": "10.1.0.5/16",
          "gateway": "10.1.0.1",
          "interface": 2
        }
    ],
    "routes": [
      {
        "dst": "0.0.0.0/0"
      }
    ],
    "interfaces": [
        {
            "name": "cni0",
            "mac": "00:11:22:33:44:55"
        },
        {
            "name": "veth3243",
            "mac": "55:44:33:22:11:11"
        },
        {
            "name": "eth0",
            "mac": "00:11:22:33:44:66",
            "sandbox": "/var/run/netns/blue"
        }
    ],
    "dns": {
      "nameservers": [ "10.1.0.1" ]
    }
  }
}
```


### Delete example

Given the same network configuration JSON list, the container runtime would perform the following steps for the _Delete_ action.
Note that plugins are executed in reverse order from the _Add_ and _Check_ actions.

1) First, call `portmap` with the following request configuration, `CNI_COMMAND=DEL`:

```json
{
  "cniVersion": "1.0.0",
  "name": "dbnet",
  "type": "portmap",
  "runtimeConfig": {
    "portMappings" : [
      { "hostPort": 8080, "containerPort": 80, "protocol": "tcp" }
    ]
  },
  "prevResult": {
    "ips": [
        {
          "address": "10.1.0.5/16",
          "gateway": "10.1.0.1",
          "interface": 2
        }
    ],
    "routes": [
      {
        "dst": "0.0.0.0/0"
      }
    ],
    "interfaces": [
        {
            "name": "cni0",
            "mac": "00:11:22:33:44:55"
        },
        {
            "name": "veth3243",
            "mac": "55:44:33:22:11:11"
        },
        {
            "name": "eth0",
            "mac": "00:11:22:33:44:66",
            "sandbox": "/var/run/netns/blue"
        }
    ],
    "dns": {
      "nameservers": [ "10.1.0.1" ]
    }
  }
}
```


2) Next, call the `tuning` plugin with the following request configuration, `CNI_COMMAND=DEL`:

```json
{
  "cniVersion": "1.0.0",
  "name": "dbnet",
  "type": "tuning",
  "sysctl": {
    "net.core.somaxconn": "500"
  },
  "runtimeConfig": {
    "mac": "00:11:22:33:44:66"
  },
  "prevResult": {
    "ips": [
        {
          "address": "10.1.0.5/16",
          "gateway": "10.1.0.1",
          "interface": 2
        }
    ],
    "routes": [
      {
        "dst": "0.0.0.0/0"
      }
    ],
    "interfaces": [
        {
            "name": "cni0",
            "mac": "00:11:22:33:44:55"
        },
        {
            "name": "veth3243",
            "mac": "55:44:33:22:11:11"
        },
        {
            "name": "eth0",
            "mac": "00:11:22:33:44:66",
            "sandbox": "/var/run/netns/blue"
        }
    ],
    "dns": {
      "nameservers": [ "10.1.0.1" ]
    }
  }
}
```

3) Finally, call `bridge`:

```json
{
  "cniVersion": "1.0.0",
  "name": "dbnet",
  "type": "bridge",
  "bridge": "cni0",
  "keyA": ["some more", "plugin specific", "configuration"],
  "ipam": {
    "type": "host-local",
    "subnet": "10.1.0.0/16",
    "gateway": "10.1.0.1"
  },
  "dns": {
    "nameservers": [ "10.1.0.1" ]
  },
  "prevResult": {
    "ips": [
        {
          "address": "10.1.0.5/16",
          "gateway": "10.1.0.1",
          "interface": 2
        }
    ],
    "routes": [
      {
        "dst": "0.0.0.0/0"
      }
    ],
    "interfaces": [
        {
            "name": "cni0",
            "mac": "00:11:22:33:44:55"
        },
        {
            "name": "veth3243",
            "mac": "55:44:33:22:11:11"
        },
        {
            "name": "eth0",
            "mac": "00:11:22:33:44:66",
            "sandbox": "/var/run/netns/blue"
        }
    ],
    "dns": {
      "nameservers": [ "10.1.0.1" ]
    }
  }
}
```

The bridge plugin executes the `host-local` delegated plugin with `CNI_COMMAND=DEL` before returning.
