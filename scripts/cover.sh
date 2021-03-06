#!/bin/bash

git archive HEAD -o build/git-arch-for-cover.tar.gz
dest=$GOPATH/src/rtfblog
mkdir -p $dest
tar xzvf build/git-arch-for-cover.tar.gz -C $dest
rm build/git-arch-for-cover.tar.gz
pushd $dest
./scripts/run-pg-tests.sh
go tool cover -html=profile.cov
popd
