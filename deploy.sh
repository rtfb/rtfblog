#!/bin/sh

cd dbtool
go build
cd ..
go build
git archive HEAD -o git-arch-for-deploy.tar.gz
dropdb tstdb
createdb tstdb
../goose/rtfb/goose up
./dbtool/dbtool -db=./testdata/db.conf -src=./testdata/legacy-db.conf -notest
pg_dump tstdb > ./testdata/rtfblog-dump.sql
vagrant up
