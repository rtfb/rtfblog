#!/bin/bash

TEMP_GOPATH=`mktemp -d /tmp/GOPATH-XXXXX`
make clean
GOPATH=$TEMP_GOPATH make
rm -fr ${TEMP_GOPATH}
