#!/bin/bash

DEBUG=0

run () {
	$*
	RET=$?
	if [ $RET -gt 0 ]; then
		echo "ERR: $1 returned $RET (commandline: '$*')" 1>&2
		exit 1
	fi
}

waitfor () {
	cmd="$1"         # command to run
	matchstr="$2"    # string to match on in output for success
	description="$3" # description of job
	timeoutsec="$4"  # timeout

	timeout=0
	istimeout=0

	if [ $timeoutsec -gt 0 ]; then
		timestamp=`date +%s`
		timeout=`expr $timestamp + $timeoutsec`
	fi

	echo "waitfor: $description" 1>&2
	RET=1
	while [ $RET -gt 0 ]; do
		if [ $timeout -gt 0 ]; then
			timestamp=`date +%s`
			if [ $timestamp -gt $timeout ]; then
				istimeout=1
				break
			fi
		fi
		$1 | grep $2
		RET=$?
	done
	if [ $istimeout -gt 0 ]; then
		echo "ERR: waitfor timed out!" 1>&2
		exit 1
	fi
}
