#!/bin/sh

git archive HEAD -o git-arch-for-deploy.tar.gz
rm testdata/foo.db
../goose-rtfb/goose-rtfb up
./dbtool/dbtool -db=./testdata/db.conf -src=./testdata/legacy-db.conf -notest
vagrant up
