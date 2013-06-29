#!/usr/bin/env bash

if [ -d package-tmp ]; then
    rm -rf package-tmp
fi

mkdir -p package-tmp
tar xzvf package.tar.gz -C package-tmp
killall rtfblog

if [ -d package-last ]; then
    rm -rf package-last
fi

mv package package-last
mv package-tmp/package ./package
rmdir package-tmp
