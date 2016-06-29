# SR-IOV CNI plugin

## Enable SR-IOV

Given Intel ixgbe NIC on CentOS, Fedora or RHEL:

```
# vi /etc/modprobe.conf
options ixgbe max_vfs=8,8
```

## Usage

Given the following network configuration:

```
# cat > /etc/cni/net.d/10-mynet.conf <<EOF
{
    "name": "mynet",
    "type": "sriov",
    "master": "eth1",
    "vf": 1,
    "mac": "66:d8:02:77:aa:aa",
    "ipam": {
        "type": "host-local",
        "subnet": "10.55.206.0/26",
        "routes": [
            { "dst": "0.0.0.0/0" }
        ],
        "gateway": "10.55.206.1"
    }
}
EOF
```

```
# CNI_PATH=$CNI_PATH CNI_ARGS="IP=10.55.206.46" ./priv-net-run.sh ifconfig
eth0      Link encap:Ethernet  HWaddr A6:9A:3B:00:63:16  
          inet addr:10.55.206.46  Bcast:0.0.0.0  Mask:255.255.255.192
          inet6 addr: fe80::a49a:3bff:fe00:6316/64 Scope:Link
          UP BROADCAST RUNNING MULTICAST  MTU:1500  Metric:1
          RX packets:0 errors:0 dropped:0 overruns:0 frame:0
          TX packets:1 errors:0 dropped:0 overruns:0 carrier:0
          collisions:0 txqueuelen:1000 
          RX bytes:0 (0.0 b)  TX bytes:90 (90.0 b)
lo        Link encap:Local Loopback  
          inet addr:127.0.0.1  Mask:255.0.0.0
          inet6 addr: ::1/128 Scope:Host
          UP LOOPBACK RUNNING  MTU:65536  Metric:1
          RX packets:0 errors:0 dropped:0 overruns:0 frame:0
          TX packets:0 errors:0 dropped:0 overruns:0 carrier:0
          collisions:0 txqueuelen:0 
          RX bytes:0 (0.0 b)  TX bytes:0 (0.0 b)
```
