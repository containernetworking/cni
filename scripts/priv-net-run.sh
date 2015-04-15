#!/bin/bash -e

# Run a command in a private network namespace
# set up by CNI plugins

netnsname=$(printf '%x%x' $RANDOM $RANDOM)
netnspath=/var/run/netns/$netnsname

ip netns add $netnsname
ip netns exec $netnsname ip link set lo up
./exec-plugins.sh add $netnspath


function cleanup() {
	./exec-plugins.sh del $netnspath
	ip netns delete $netnsname
}
trap cleanup EXIT

ip netns exec $netnsname $@
