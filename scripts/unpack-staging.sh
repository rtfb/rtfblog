#!/usr/bin/env bash

if [ -d package-staging ]; then
    rm -rf package-staging
fi

mkdir -p package-staging
tar xzvf package.tar.gz -C package-staging
# XXX: can't do that!
#killall rtfblog

mkdir -p ./package-staging/static
cp ./package/static/* ./package-staging/static/
