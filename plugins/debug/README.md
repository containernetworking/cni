# debug plugin

## Overview

This plugin aims to help debugging or troubleshooting in CNI plugin development.

## Example Configuration

```
{
        "cniVersion": "0.3.1",
        "name": "mynet",
        "plugins": [
                {
                        "type": "ptp",
                        "ipMasq": true,
                        "ipam": {
                                "type": "host-local",
                                "subnet": "172.16.30.0/24",
                                "routes": [
                                        {
                                                "dst": "0.0.0.0/0"
                                        }
                                ]
                        }
                },
                {
                        "type": "debug",
                        "cniOutput": "/tmp/cni_output.txt",
                        "addHooks": [
                                [ "sh", "-c", "ip link set $CNI_IFNAME promisc on" ]
                        ]
                },
                {
                        "type": "portmap",
                        "capabilities": {"portMappings": true},
                        "externalSetMarkChain": "KUBE-MARK-MASQ"
                }
        ]
}
```

## Config Reference

* `cniOutput` (string, optional): output CNI request into file.
* `addHooks` (string array, optional): commands executed in container network namespace at interface add.
   (note: but just execute it and does not catch command failure)
* `delHooks` (string array, optional): commands executed in container network namespace at interface delete.
   (note: but just execute it and does not catch command failure)
* `checkHooks` (string array, optional): commands executed in container network namespace at interface check.
   (note: but just execute it and does not catch command failure)

### Sample CNI Ouput

```
CmdAdd
ContainerID: cnitool-20c433bb2b1d6ede56d6
Netns: /var/run/netns/cnitest
IfName: eth0
Args: 
Path: /opt/cni/bin
StdinData: {"cniOutput":"/tmp/cni_output.txt","cniVersion":"0.3.1","name":"test","prevResult":{"cniVersion":"0.3.1","interfaces":[{"name":"veth92e295cc","mac":"56:22:7f:b7:5b:75"},{"name":"eth0","mac":"46:b3:f3:77:bf:21","sandbox":"/var/run/netns/cnitest"}],"ips":[{"version":"4","interface":1,"address":"10.1.1.2/24","gateway":"10.1.1.1"}],"dns":{"nameservers":["10.64.255.25","8.8.8.8"]}},"type":"none"}
----------------------
CmdDel
ContainerID: cnitool-20c433bb2b1d6ede56d6
Netns: /var/run/netns/cnitest
IfName: eth0
Args: 
Path: /opt/cni/bin
StdinData: {"cniOutput":"/tmp/cni_output.txt","cniVersion":"0.3.1","name":"test","type":"none"}
----------------------
```
