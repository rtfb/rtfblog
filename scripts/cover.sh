#!/bin/bash

git archive HEAD -o build/git-arch-for-cover.tar.gz
dest=$GOPATH/src/rtfblog
mkdir -p $dest
tar xzvf build/git-arch-for-cover.tar.gz -C $dest
rm build/git-arch-for-cover.tar.gz
make src/version.go
cp src/version.go $dest/src
pushd $dest
./scripts/run-db-tests.sh
go tool cover -html=profile.cov
popd
