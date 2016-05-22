#!/bin/bash

source lib/functions.sh

# Verify the operation of CNI IPAM DHCP plugin with ipvlan transport
#
# Problem/Test description:
# - With ipvlan multiple clients will have the same MAC address
# - DHCP servers typically identify their clients using MAC address
# - DHCP servers SHOULD use DHCP Client Identifier whenever present
# - To make sure DHCP Client Identifier is used packets are sent with
#   zero CHAddr.
# - Verify that this works with this functional test

# create configuration
run sudo mkdir -p /etc/cni/net.d
run cat <<EOF > /tmp/10-mynet.conf
{
    "name": "mynet",
    "type": "ipvlan",
    "master": "br42",
    "ipam": {
        "type": "dhcp",
        "broadcast": true,
        "sendClientID": true
    }
}
EOF
run sudo mv /tmp/10-mynet.conf /etc/cni/net.d/

# run CNI IPAM DHCP Daemon
run sudo ../bin/dhcp daemon &
waitfor "sudo ls -l /run/cni/dhcp.sock" "dhcp.sock" "CNI DHCP Daemon UNIX socket" 5

# use CNI to create namespace
cd ../scripts
CNI_PATH=`pwd`/../bin
run sudo CNI_PATH=$CNI_PATH CNI_IFNAME=eth0 ./priv-net-run.sh ip addr show dev eth0 | tee /tmp/cni-ip-addr-output-1
run sudo CNI_PATH=$CNI_PATH CNI_IFNAME=eth0 ./priv-net-run.sh ip addr show dev eth0 | tee /tmp/cni-ip-addr-output-2

# verify that we get a different address in each attempt
IP1=`grep "inet " /tmp/cni-ip-addr-output-1 | cut -f6 -d\ `
IP2=`grep "inet " /tmp/cni-ip-addr-output-2 | cut -f6 -d\ `

if [ "${IP1}" != "${IP2}" ]; then
	echo "Successfully acquired two different IP leases with DHCP using ipvlan"
	exit 0
else
	echo "ERROR: No lease acquired or the same lease acquired twice"
	exit 255
fi
