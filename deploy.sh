#!/bin/sh

cd dbtool
go build
cd ..
go build
killall rtfblog
go test
git archive HEAD -o git-arch-for-deploy.tar.gz
vagrant up
