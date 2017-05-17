#!/usr/bin/env bash

if [[ ${DEBUG} -gt 0 ]]; then set -x; fi

NETCONFPATH=${NETCONFPATH-/etc/cni/net.d}

function process_error() {
	errmsg=$(echo $1 | jq -r '.msg')
	if [ -z "$errmsg" ]; then
		errmsg=$res
	fi

	echo "${name} : error executing $CNI_COMMAND: $errmsg"
	exit 1
}

function exec_plugins() {
	i=0
	contid=$2
	netns=$3
	export CNI_COMMAND=$(echo $1 | tr '[:lower:]' '[:upper:]')
	export PATH=$CNI_PATH:$PATH
	export CNI_CONTAINERID=$contid
	export CNI_NETNS=$netns

	for netconf in $(echo $NETCONFPATH/*.conf | sort); do
		cniVersion=$(jq -r '.cniVersion' <$netconf)
		name=$(jq -r '.name' <$netconf)
		export CNI_IFNAME=$(printf eth%d $i)
		numPlugins=$(jq '.plugins | length' <$netconf)


		if [ $numPlugins -eq 0 ]; then
			plugin=$(jq -r '.type' <$netconf)
			res=$($plugin <$netconf)

			if [ $? -ne 0 ]; then
				process_error "$res"
			elif [[ ${DEBUG} -gt 0 ]]; then
				echo ${res} | jq -r .
			fi
		else
			for ((j=0; j<numPlugins; j++)); do
				partConfig=$(jq -r --arg index "$j" '.plugins[$index | tonumber]' < $netconf)
				plugin=$(jq -r --arg index "$j" '.plugins[$index | tonumber].type' < $netconf)
				pluginConfig=$(echo $partConfig | jq -r --arg version "$cniVersion" --arg name "$name" \
					'{"cniVersion": $version} + {"name": $name} + .')

				res=$(echo $pluginConfig | $plugin)
				if [ $? -ne 0 ]; then
					process_error "$res"
				elif [[ ${DEBUG} -gt 0 ]]; then
					echo ${res} | jq -r .
				fi
			done
		fi

		let "i=i+1"
		done
}

if [ $# -ne 3 ]; then
	echo "Usage: $0 add|del CONTAINER-ID NETNS-PATH"
	echo "  Adds or deletes the container specified by NETNS-PATH to the networks"
	echo "  specified in \$NETCONFPATH directory"
	exit 1
fi

exec_plugins $1 $2 $3
