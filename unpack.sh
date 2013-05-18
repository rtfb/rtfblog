#!/usr/bin/env bash

if [ -d package-tmp ]; then
    rm -rf package-tmp
fi

mkdir -p package-tmp
tar xzvf package.tar.gz -C package-tmp
killall rtfblog
mv package package-last
mv package-tmp/package ./package
rmdir package-tmp
