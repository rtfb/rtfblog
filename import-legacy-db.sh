#!/bin/sh

cd dbtool
go build
cd ..
dropdb tstdb
createdb tstdb
../goose/rtfb/goose up
./dbtool/dbtool -db=./testdata/db.conf -src=./testdata/legacy-db.conf -notest
pg_dump tstdb > ./testdata/rtfblog-dump.sql
