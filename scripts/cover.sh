#!/bin/bash

git archive HEAD -o git-arch-for-deploy.tar.gz
dest=$GOPATH/src/rtfblog
mkdir -p $dest
tar xzvf git-arch-for-deploy.tar.gz -C $dest
pushd $dest
go test -coverprofile=coverage.out ./src/...
go tool cover -html=coverage.out
popd