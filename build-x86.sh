#!/usr/bin/env bash

if ! [ $(id -un) = vagrant ]; then
    echo "You're supposed to run this inside a vagrant box!" 1>&2
    exit 1
fi

# Needed to go get stuff from various sources
apt-get install --yes git mercurial

if ! hash go 2>/dev/null; then
    wget -q https://godeb.s3.amazonaws.com/godeb-386.tar.gz
    tar xzvf godeb-386.tar.gz
    ./godeb install 1.1.2
fi

go=/usr/bin/go
gopkgs=/home/vagrant/gopkgs
mkdir -p $gopkgs
export GOPATH=$gopkgs

builddir=rtfblog
rm -rf /home/vagrant/$builddir
package=/home/vagrant/$builddir/package
mkdir -p $package

cp /vagrant/git-arch-for-deploy.tar.gz /home/vagrant/$builddir/
cd /home/vagrant/$builddir
tar xzvf git-arch-for-deploy.tar.gz
$go get
$go build

cd /home/vagrant
if ! [ -d goose ]; then
    git clone https://bitbucket.org/liamstask/goose.git
fi
cd goose/cmd/goose
$go get
$go build

cd /home/vagrant/$builddir/
cp /home/vagrant/goose/cmd/goose/goose $package
cp -r /vagrant/db $package
cp /home/vagrant/$builddir/rtfblog $package
cp -r /vagrant/static $package
cp -r /vagrant/tmpl $package
cp /vagrant/stuff/images/* $package/static/
cp /vagrant/testdata/rtfblog-dump.sql $package/rtfblog-dump.sql
tar czvf package.tar.gz ./package
rm -rf $package

/vagrant/ssh_key_setup.sh
scp -q package.tar.gz rtfb@rtfb.lt:/home/rtfb/package.tar.gz
scp -q /vagrant/unpack.sh rtfb@rtfb.lt:/home/rtfb/unpack.sh
ssh rtfb@rtfb.lt /home/rtfb/unpack.sh
ssh rtfb@rtfb.lt "cd /home/rtfb/package; ./goose -env=production up"
ssh rtfb@rtfb.lt "./rtfblog &"
