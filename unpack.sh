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
mkdir -p ./package/static
cp ./package-last/static/* ./package/static/
mv package-tmp/package/static/* ./package/static/
rmdir package-tmp/package/static/
mv package-tmp/package/* ./package/
rm -r package-tmp
