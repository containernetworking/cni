#!/bin/bash

source $(dirname $0)/functions.sh

# set up host bridge
run brctl addbr br42
run brctl stp br42 off
run ip link set dev br42 up

# set up srv namespace and connect it to host bridge
run ip netns add srv
run ip link add srvint type veth peer name br42srv
run ip link set srvint netns srv
run brctl addif br42 br42srv
run ip link set dev br42srv up
run ip netns exec srv ip link set dev srvint up
run ip netns exec srv ip addr add 2001:db8:42::1/64 dev srvint
run ip netns exec srv ip addr add 172.16.42.1/24 dev srvint

# set up ISC DHCP server and connect it to srv namespace
run chown root:root /var/lib/dhcp/dhcpd.leases
cat << EOF >> /etc/dhcp/dhcpd.conf
shared-network 172-16-42 {
  subnet 172.16.42.0 netmask 255.255.255.0 {
  }
  pool {
    range 172.16.42.100 172.16.42.200;
  }
}
EOF
DEBUGFLAG="-q"
if [ $DEBUG -gt 0 ]; then
	DEBUGFLAG="-d"
fi
run ip netns exec srv /usr/sbin/dhcpd ${DEBUGFLAG} srvint &

# set up radvd and connect it to srv namespace
cat << EOF >> /etc/radvd.conf
interface srvint
{
	AdvSendAdvert on;
	prefix 2001:db8:42::/64
	{
		AdvOnLink on;
		AdvAutonomous on;
	};
};
EOF
run ip netns exec srv /usr/sbin/radvd


# set up cli namespace and connect it to host bridge
run ip netns add cli
run ip link add cliint type veth peer name br42cli
run ip link set cliint netns cli
run brctl addif br42 br42cli
run ip link set dev br42cli up
run ip netns exec cli ip link set dev cliint up

# run ISC DHCP client in cli namespace
run ip netns exec cli /sbin/dhclient cliint

# verify L2 and L3 connectivity
waitfor "ip netns exec cli ip -4 addr show dev cliint" "172.16" "IPv4 Address Configuration (DHCP)" 5
waitfor "ip netns exec cli ip -6 addr show dev cliint" "2001:db8" "IPv6 StateLess Address Autoconfiguration" 5
if [ $DEBUG -gt 0 ]; then
	run ip netns exec cli ping -c 1 172.16.42.1
	run ip netns exec cli ping6 -c 1 2001:db8:42::1
	run ip netns exec cli arp -an
	run ip netns exec cli ip -6 neigh
	run ip netns exec cli ip addr
fi
