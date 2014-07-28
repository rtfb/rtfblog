#!/usr/bin/env bash

if [ -d package-tmp ]; then
    rm -r package-tmp
fi

mkdir -p package-tmp
tar xzvf package.tar.gz -C package-tmp

kill $(pidof /home/rtfb/package-staging/rtfblog)

if [ -d package-staging ]; then
    rm -r package-staging
fi

mv package-tmp/package package-staging
rmdir package-tmp
mkdir -p ./package-staging/static
cp ./package/static/* ./package-staging/static/
ln -s /home/rtfb/rtfblogrc-staging ./package-staging/.rtfblogrc
