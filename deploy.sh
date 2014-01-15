#!/bin/sh

cd dbtool
go build
cd ..
go build
killall rtfblog
go test

package=./package
mkdir -p $package
cp ../goose/upstream/cmd/goose/goose $package
cp -r ./db $package
cp ./rtfblog $package
cp -r ./static $package
cp -r ./tmpl $package
cp ./stuff/images/* $package/static/
cp ./testdata/rtfblog-dump.sql $package/rtfblog-dump.sql
tar czvf package.tar.gz ./package
rm -rf $package

scp -q ./unpack.sh rtfb@rtfb.lt:/home/rtfb/unpack.sh
scp -q package.tar.gz rtfb@rtfb.lt:/home/rtfb/package.tar.gz
rm ./package.tar.gz
ssh rtfb@rtfb.lt /home/rtfb/unpack.sh
#ssh rtfb@rtfb.lt "cd /home/rtfb/package; ./goose -env=production up"
ssh rtfb@rtfb.lt "cd /home/rtfb/package; nohup ./rtfblog </dev/null 1>&2&> nohup.log &"
