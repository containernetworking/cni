# cnitool

`cnitool` is a simple program that executes a CNI configuration. It will
add, check, remove, gc, or get status of an interface in an already-created network namespace.

## Environment Variables

* `NETCONFPATH`: This environment variable needs to be set to a
  directory. It defaults to `/etc/cni/net.d`. The `cnitool` searches
  for CNI configuration files in this directory according to the following priorities:
  1. Search files with the extension `*.conflist`, representing a list of plugin configurations.
  2. If there are no `*.conflist` files in the directory, search files with the extension `*.conf` or `*.json`,
  representing a single plugin configuration.

  It loads all the CNI configuration files in
  this directory and if it finds a CNI configuration with the `network
  name` given to the cnitool it returns the corresponding CNI
  configuration, else it returns `nil`.
* `CNI_PATH`: For a given CNI configuration `cnitool` will search for
  the corresponding CNI plugin in this path.
* `CNI_ARGS`: Optional arguments in the format "KEY1=VALUE1;KEY2=VALUE2;..." passed to the CNI plugin.
* `CAP_ARGS`: Optional capability arguments in JSON format passed to the CNI plugin.
* `CNI_IFNAME`: Interface name to be configured. Defaults to "eth0" if not specified.

## Command Line Usage

```bash
cnitool: CNI Tool for managing network interfaces in a network namespace

Usage:
  cnitool [command]

Available Commands:
  add         Add network interface to a network namespace
  check       Check network interface in a network namespace
  completion  Generate the autocompletion script for the specified shell
  del         Delete network interface from a network namespace
  gc          Garbage collect network interfaces
  help        Help about any command
  status      Get status of network interfaces

Flags:
  -h, --help            help for cnitool
  -i, --ifname string   Interface name (defaults to env var CNI_IFNAME or 'eth0')

Use "cnitool [command] --help" for more information about a command.
```

## Example invocation

First, install cnitool:

```bash
go install github.com/containernetworking/cni/cnitool@latest
```

Then, check out and build the plugins. All commands should be run from this directory.

```bash
git clone https://github.com/containernetworking/plugins.git
cd plugins
./build_linux.sh
# or
./build_windows.sh
```

Create a network configuration

```bash
echo '{"cniVersion":"0.4.0","name":"myptp","type":"ptp","ipMasq":true,"ipam":{"type":"host-local","subnet":"172.16.29.0/24","routes":[{"dst":"0.0.0.0/0"}]}}' | sudo tee /etc/cni/net.d/10-myptp.conf
```

Create a network namespace. This will be called `testing`:

```bash
sudo ip netns add testing
```

Add the container to the network:

```bash
sudo CNI_PATH=./bin cnitool add myptp /var/run/netns/testing
```

Check whether the container's networking is as expected (ONLY for spec v0.4.0+):

```bash
sudo CNI_PATH=./bin cnitool check myptp /var/run/netns/testing
```

Test that it works:

```bash
sudo ip -n testing addr
sudo ip netns exec testing ping -c 1 4.2.2.2
```

And clean up:

```bash
sudo CNI_PATH=./bin cnitool del myptp /var/run/netns/testing
sudo ip netns del testing
```
