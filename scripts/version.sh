#!/bin/sh

if git describe --exact-match HEAD > /dev/null 2>&1 ; then
    echo $(git describe --exact-match HEAD)
else
    echo $(git rev-parse --short HEAD)
fi
