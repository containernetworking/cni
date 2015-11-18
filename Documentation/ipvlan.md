# ipvlan plugin

## Overview

ipvlan is a new [addition](https://lwn.net/Articles/620087/) to the Linux kernel.
Like its cousin macvlan, it virtualizes the host interface.
However unlike macvlan which generates a new MAC address for each interface, ipvlan devices all share the same MAC.
The kernel driver inspects the IP address of each packet when making a decision about which virtual interface should process the packet.

Because all ipvlan interfaces share the MAC address with the host interface, DHCP can only be used in conjunction with ClientID (currently not supported by DHCP plugin).

## Example configuration

```
{
	"name": "mynet",
	"type": "ipvlan",
	"master": "eth0",
	"ipam": {
		"type": "host-local",
		"subnet": "10.1.2.0/24",
	}
}
```

## Network configuration reference

* `name` (string, required): the name of the network.
* `type` (string, required): "ipvlan".
* `master` (string, required): name of the host interface to enslave.
* `mode` (string, optional): one of "l2", "l3". Defaults to "l2".
* `mtu` (integer, optional): explicitly set MTU to the specified value. Defaults to the value chosen by the kernel.
* `ipam` (dictionary, required): IPAM configuration to be used for this network.

## Notes

* `ipvlan` does not allow virtual interfaces to communicate with the master interface.
Therefore the container will not be able to reach the host via `ipvlan` interface.
Be sure to also have container join a network that provides connectivity to the host (e.g. `ptp`).
* A single master interface can not be enslaved by both `macvlan` and `ipvlan`.

### Using ipvlan and dhcp
There are two important considerations when using the "ipvlan" plugin together with the "dhcp" IPAM plugin.

First, the kernel ipvlan driver uses the IP packet's destination address when choosing the slave interface to which it should be forwarded.
The driver cannot perform this forwarding if the container interface has not yet been assigned an IP address.
Since the DHCP packets are exchanged to determine which IP should be used, the interface will not yet have an IP address.
DHCP servers typically unicast replies back to the client.
The ipvlan driver will drop these replies.
To work around this, the DHCP client must instruct the DHCP server to use broadcast replies instead.
This can be accomplished by setting the "broadcast" flag in the "ipam" section of the network configuration:

```
{
	"name": "mynet",
	"type": "ipvlan",
	"master": "eth0",
	"ipam": {
		"type": "dhcp",
		"broadcast": true
	}
}
```

Second, because all ipvlan interfaces share the MAC address of the master interface, DHCP servers will have trouble identifying and tracking individual container interfaces.
To mitigate this, it is necessary to include DHCP [option 61](http://tools.ietf.org/html/rfc2132#section-9.14) (Client Identifier) in DHCP messages.
The DHCP plugin will emit such client identifiers if the sendClientID flag is set, as it is by default.
The value of this Client Identifier option will be ContainerID catenated with the network name.
