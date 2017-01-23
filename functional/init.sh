#!/bin/bash

# load common functions
source lib/functions.sh

# set up network
sudo lib/setup-network.sh

# run tests
for script in `ls -1R test-*.sh`; do
	run ./$script
done
