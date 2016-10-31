# macvlan plugin

## Overview

[macvltap](http://backreference.org/2014/03/20/some-notes-on-macvlanmacvtap/) can be used like a tap device.
A host interface gets "enslaved" with the virtual interfaces sharing the physical device but having distinct MAC addresses.

## Example configuration

```
{
  "name": "mynet",
  "type": "macvtap",
  "master": "eth0",
  "mac": "52:54:00:dc:6a:e0",
}
```

## Network configuration reference

* `name` (string, required): the name of the network
* `type` (string, required): "macvlan"
* `master` (string, required): name of the host interface to enslave
* `mac` (string, optional): MAC address to use
* `mode` (string, optional): one of "bridge", "private", "vepa", "passthrough". Defaults to "bridge".
* `mtu` (integer, optional): explicitly set MTU to the specified value. Defaults to the value chosen by the kernel.

## Notes

* TODO: `ipam` support is missing at the moment.
* The container needs cgroup permission to access macvtap devices. Docker for instance allow setting a prepared cgroup via `--cgroup-parent`.
* Since there is no udev running inside a container the device node is not created. This can be done with `mknod`.
