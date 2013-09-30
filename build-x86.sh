#!/usr/bin/env bash

# Needed to go get stuff from github
apt-get install --yes git

# Now, I'd love to `apt-get install --yes golang`, but its installer has a
# stupid TUI dialog with a silly question that I couldn't find a way to dismiss
# and it fucks everything up. So install prerequisites to build Go from source
# instead.
apt-get install --yes gcc libc6-dev mercurial

if ! [ -d /home/vagrant/go ]; then
    hg clone -u release-branch.go1.1 https://code.google.com/p/go
    cd go/src
    ./all.bash
fi

go=/home/vagrant/go/bin/go
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
git clone https://bitbucket.org/liamstask/goose.git
cd goose
$go get
$go build

cd /home/vagrant/$builddir/
cp /home/vagrant/goose/goose $package
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
