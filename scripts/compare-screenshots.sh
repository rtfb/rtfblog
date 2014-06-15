#!/bin/bash

# These need to be installed for this to work:
#
# $ sudo apt-get install imagemagick

for file in stuff/reference-screenshots/*.png
do
    bn=$(basename $file)
    difffile="${bn%.*}".diff.png
    echo $difffile
    composite $file shots/$bn -compose difference $difffile
done
