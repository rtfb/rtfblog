#!/bin/bash

git archive HEAD -o build/git-arch-for-cover.tar.gz
dest=$GOPATH/src/rtfblog
mkdir -p $dest
tar xzvf build/git-arch-for-cover.tar.gz -C $dest
rm build/git-arch-for-cover.tar.gz
make version
cp src/version.go $dest/src
pushd $dest
go test -coverprofile=coverage.out ./src/...
go tool cover -html=coverage.out
popd
