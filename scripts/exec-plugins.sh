#!/usr/bin/env bash

if [[ ${DEBUG} -gt 0 ]]; then set -x; fi

NETCONFPATH="${NETCONFPATH-/etc/cni/net.d}"

function exec_list() {
  plist="$1"
  name="$2"
  cniVersion="$3"
  echo "$plist" | jq -c '.[]' | while read -r conf; do
    plugin_bin="$(echo "$conf" | jq -r '.type')"
    conf="$(echo "$conf" | jq -r ".name = \"$name\" | .cniVersion = \"$cniVersion\"")"
    if [ -n "$res" ]; then
      conf="$(echo "$conf" | jq -r ".prevResult=$res")"
    fi
    if ! res=$(echo "$conf" | $plugin_bin); then
      error "$name" "$res"
    elif [[ ${DEBUG} -gt 0 ]]; then
      echo "${res}" | jq -r .
    fi
  done
}

function error () {
  name="$1"
  res="$2"
  err_msg=$(echo "$res" | jq -r '.msg')
  if [ -z "$errmsg" ]; then
    err_msg=$res
  fi
  echo "${name} : error executing $CNI_COMMAND: $err_msg"
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

	for netconf in $(echo "$NETCONFPATH"/*.conf | sort); do
	  export CNI_IFNAME=$(printf eth%d $i)
	  name=$(jq -r '.name' <"$netconf")
	  cniVersion=$(jq -r '.cniVersion' <"$netconf")
	  plist=$(jq '.plugins | select(.!=null)' <"$netconf")
	  if [ -n "$plist" ]; then
	    exec_list "$plist" "$name" "$cniVersion"
	  else
      plugin=$(jq -r '.type' <"$netconf")

      if ! res=$($plugin <"$netconf"); then
        error "$name" "$res"
      elif [[ ${DEBUG} -gt 0 ]]; then
        echo "${res}" | jq -r .
      fi
    fi

		(( i++ )) || true
	done
}

if [ $# -ne 3 ]; then
	echo "Usage: $0 add|del CONTAINER-ID NETNS-PATH"
	echo "  Adds or deletes the container specified by NETNS-PATH to the networks"
	echo "  specified in \$NETCONFPATH directory"
	exit 1
fi

exec_plugins $1 $2 $3
