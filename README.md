[![Build Status](https://travis-ci.org/containernetworking/cni.svg?branch=master)](https://travis-ci.org/appc/cni)
[![Coverage Status](https://coveralls.io/repos/github/containernetworking/cni/badge.svg?branch=master)](https://coveralls.io/github/appc/cni?branch=master)

# CNI - the Container Network Interface

## What is CNI?

CNI, the _Container Network Interface_, is a project to help configure network interfaces for Linux application containers.
CNI concerns itself with setting up network connectivity of containers and removing allocated resources when the container is deleted.
Because of this focus CNI has a wide range of support and the specification is simple to implement.

The project repository contains a [simple specification](SPEC.md) for how executable plugins can be used to configure network namespaces, and a number of plugins, libraries and utilities written in Go that can be used to get started with CNI.

The specification and the plugin architecture form a template system with perfectly clear interfaces.
This allows developers to integrate their existing container networking solutions with a thin CNI wrapper plugin.
It also gives a good framework to create new container networking solutions from scratch.

## Why develop CNI?

Application containers on Linux are a rapidly evolving area, and within this space networking is a particularly unsolved problem, as it is highly environment-specific.
We believe that every container runtime will seek to solve the same problem of making the network layer pluggable.

To avoid duplication, we think it is prudent to define a common interface between the network plugins and container execution.
Hence we are proposing this specification, along with an initial set of plugins that can be used by different container runtime systems.

## Who is using CNI?

- [rkt - container engine](https://coreos.com/blog/rkt-cni-networking.html)
- [Kurma - container runtime](http://kurma.io/)
- [Kubernetes - a system to simplify container operations](http://kubernetes.io/docs/admin/network-plugins/)
- [Cloud Foundry - a platform for cloud applications](https://github.com/cloudfoundry-incubator/guardian-cni-adapter)
- [Weave - a multi-host Docker network](https://github.com/weaveworks/weave)
- [Project Calico - a layer 3 virtual network](https://github.com/projectcalico/calico-cni)
- [Contiv Networking - policy networking for various use cases](https://github.com/contiv/netplugin)

## Contributing to CNI

We welcome contributions, including [bug reports](https://github.com/appc/cni/issues), and code and documentation improvements.
If you intend to contribute to code or documentation, please read [CONTRIBUTING.md](CONTRIBUTING.md). Also see the [contact section](#contact) in this README.

## How do I use CNI?

### Requirements
CNI requires Go 1.5+ to build.

Go 1.5 users will need to set GO15VENDOREXPERIMENT=1 to get vendored
dependencies. This flag is set by default in 1.6.

### Included Plugins
This repository includes a number of common plugins in the `plugins/` directory.
Please see the [Documentation/](Documentation/) directory for documentation about particular plugins.

### Running the plugins
The scripts/ directory contains two scripts, `priv-net-run.sh` and `docker-run.sh`, that can be used to exercise the plugins.

**note - priv-net-run.sh depends on `jq`**

Start out by creating a netconf file to describe a network:

```bash
$ mkdir -p /etc/cni/net.d
$ cat >/etc/cni/net.d/10-mynet.conf <<EOF
{
	"name": "mynet",
	"type": "bridge",
	"bridge": "cni0",
	"isGateway": true,
	"ipMasq": true,
	"ipam": {
		"type": "host-local",
		"subnet": "10.22.0.0/16",
		"routes": [
			{ "dst": "0.0.0.0/0" }
		]
	}
}
EOF
$ cat >/etc/cni/net.d/99-loopback.conf <<EOF
{
	"type": "loopback"
}
EOF
```

The directory `/etc/cni/net.d` is the default location in which the scripts will look for net configurations.

Next, build the plugins:

```bash
$ ./build
```

Finally, execute a command (`ifconfig` in this example) in a private network namespace that has joined the `mynet` network:

```bash
$ CNI_PATH=`pwd`/bin
$ cd scripts
$ sudo CNI_PATH=$CNI_PATH ./priv-net-run.sh ifconfig
eth0      Link encap:Ethernet  HWaddr f2:c2:6f:54:b8:2b  
          inet addr:10.22.0.2  Bcast:0.0.0.0  Mask:255.255.0.0
          inet6 addr: fe80::f0c2:6fff:fe54:b82b/64 Scope:Link
          UP BROADCAST MULTICAST  MTU:1500  Metric:1
          RX packets:1 errors:0 dropped:0 overruns:0 frame:0
          TX packets:0 errors:0 dropped:1 overruns:0 carrier:0
          collisions:0 txqueuelen:0 
          RX bytes:90 (90.0 B)  TX bytes:0 (0.0 B)

lo        Link encap:Local Loopback  
          inet addr:127.0.0.1  Mask:255.0.0.0
          inet6 addr: ::1/128 Scope:Host
          UP LOOPBACK RUNNING  MTU:65536  Metric:1
          RX packets:0 errors:0 dropped:0 overruns:0 frame:0
          TX packets:0 errors:0 dropped:0 overruns:0 carrier:0
          collisions:0 txqueuelen:0 
          RX bytes:0 (0.0 B)  TX bytes:0 (0.0 B)
```

The environment variable `CNI_PATH` tells the scripts and library where to look for plugin executables.

## Running a Docker container with network namespace set up by CNI plugins

Use the instructions in the previous section to define a netconf and build the plugins.
Next, docker-run.sh script wraps `docker run`, to execute the plugins prior to entering the container:

```bash
$ CNI_PATH=`pwd`/bin
$ cd scripts
$ sudo CNI_PATH=$CNI_PATH ./docker-run.sh --rm busybox:latest ifconfig
eth0      Link encap:Ethernet  HWaddr fa:60:70:aa:07:d1  
          inet addr:10.22.0.2  Bcast:0.0.0.0  Mask:255.255.0.0
          inet6 addr: fe80::f860:70ff:feaa:7d1/64 Scope:Link
          UP BROADCAST MULTICAST  MTU:1500  Metric:1
          RX packets:1 errors:0 dropped:0 overruns:0 frame:0
          TX packets:0 errors:0 dropped:1 overruns:0 carrier:0
          collisions:0 txqueuelen:0 
          RX bytes:90 (90.0 B)  TX bytes:0 (0.0 B)

lo        Link encap:Local Loopback  
          inet addr:127.0.0.1  Mask:255.0.0.0
          inet6 addr: ::1/128 Scope:Host
          UP LOOPBACK RUNNING  MTU:65536  Metric:1
          RX packets:0 errors:0 dropped:0 overruns:0 frame:0
          TX packets:0 errors:0 dropped:0 overruns:0 carrier:0
          collisions:0 txqueuelen:0 
          RX bytes:0 (0.0 B)  TX bytes:0 (0.0 B)
```

## What might CNI do in the future?

CNI currently covers a wide range of needs for network configuration due to it simple model and API.
However, in the future CNI might want to branch out into other directions:

- Dynamic updates to existing network configuration
- Dynamic policies for network bandwidth and firewall rules

If these topics of are interest please contact the team via the mailing list or IRC and find some like minded people in the community to put a proposal together.

## Contact

For any questions about CNI, please reach out on the mailing list or IRC:
- Email: [cni-dev](https://groups.google.com/forum/#!forum/cni-dev)
- IRC: #[appc](irc://irc.freenode.org:6667/#appc) IRC channel on freenode.org
