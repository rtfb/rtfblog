#!/bin/sh

# This script is called on ENTRYPOINT in the container. It fixes up file
# permissions for use in the container, and then execs 'make all', which is
# passed here as param.
sudo chown -R rtfb:rtfb /home/rtfb/dev
exec $@
