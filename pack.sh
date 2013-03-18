#!/bin/sh

package=./package

if [ -d $package ] ; then
    rm -r $package
fi

mkdir -p $package/dbtool

cp rtfblog.go $package
cp rtfblog_test.go $package
cp dbtool/dbtool.go $package/dbtool
cp dbtool/b2e-import.go $package/dbtool
cp Makefile $package
cp sample-server.conf $package
cp -r static $package
cp -r tmpl $package
cp stuff/images/* $package/static/

./dbtool/dbtool -db=./testdata/db.conf -src=./testdata/legacy-db.conf -notest
cp ./testdata/foo.db $package/main.db

tar czvf package.tar.gz $package
