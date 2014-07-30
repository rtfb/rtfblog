#!/usr/bin/env bash

if [ -d package-tmp ]; then
    rm -r package-tmp
fi

mkdir -p package-tmp
tar xzvf package.tar.gz -C package-tmp

if [ -d $1-last ]; then
    rm -r $1-last
fi

mv $1 $1-last

mv package-tmp/package $1
mkdir -p ./$1/static
if [[ $1 == *-staging ]]; then
    cp ./package/static/* ./$1/static/
    ln -s /home/rtfb/rtfblogrc-staging ./$1/.rtfblogrc
else
    cp ./package-last/static/* ./$1/static/
fi
rmdir package-tmp
