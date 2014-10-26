#!/bin/bash

killall rtfblog
rm src/version.go
make all

suffix="-staging"
goose_env="staging"

if [ "$1" == "prod" ]; then
    suffix=""
    goose_env="production"
fi

package=./package
cp -r build $package
rm $package/server.conf
rm $package/server.log
cp $GOPATH/bin/goose $package
cp -r ./db $package
cp ./stuff/images/* $package/static/
cp ./testdata/rtfblog-dump.sql $package/rtfblog-dump.sql
tar czvf package.tar.gz ./package
rm -rf $package

scp -q scripts/unpack.sh rtfb@rtfb.lt:/home/rtfb/unpack.sh
scp -q package.tar.gz rtfb@rtfb.lt:/home/rtfb/package.tar.gz
rm ./package.tar.gz
full_path=/home/rtfb/package$suffix
pid=ssh rtfb@rtfb.lt "pidof $full_path/rtfblog"
ssh rtfb@rtfb.lt "kill $pid"
ssh rtfb@rtfb.lt "/home/rtfb/unpack.sh package$suffix"
ssh rtfb@rtfb.lt "rm $full_path/db/dbconf.yml"
ssh rtfb@rtfb.lt "ln -s /home/rtfb/rtfblog-dbconf.yml $full_path/db/dbconf.yml"
ssh rtfb@rtfb.lt "cd $full_path; ./goose -env=$goose_env up"
ssh rtfb@rtfb.lt "nohup $full_path/rtfblog </dev/null 1>&2&> $full_path/nohup.log &"
