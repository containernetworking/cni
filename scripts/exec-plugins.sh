#!/bin/bash -e

NETCONFPATH=${NETCONFPATH-/etc/cni/net.d}

function exec_plugins() {
	i=0
	netns=$2
	export CNI_COMMAND=$(echo $1 | tr '[:lower:]' '[:upper:]')
	export PATH=$CNI_PATH:$PATH
	export CNI_NETNS=$netns

	for netconf in $(echo $NETCONFPATH/*.conf | sort); do
		plugin=$(jq -r '.type' <$netconf)
		export CNI_IFNAME=$(printf eth%d $i)

		$plugin <$netconf >/dev/null

		let "i=i+1"
	done
}

if [ $# -ne 2 ]; then
	echo "Usage: $0 add|del NETNS-PATH"
	echo "  Adds or deletes the container specified by NETNS-PATH to the networks"
	echo "  specified in \$NETCONFPATH directory"
	exit 1
fi

exec_plugins $1 $2
