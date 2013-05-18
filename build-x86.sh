#!/usr/bin/env bash

# Needed to go get stuff from github
apt-get install --yes git

# Need this to build go-sqlite3
apt-get install --yes pkg-config

# Need these to build sqlite. Need to build sqlite with --enable-threadsafe
# because the version on the server was built without it and go-sqlite3 assumes
# it.
apt-get install --yes libsqlite3-dev make

# Now, I'd love to `apt-get install --yes golang`, but its installer has a
# stupid TUI dialog with a silly question that I couldn't find a way to dismiss
# and it fucks everything up. So install prerequisites to build Go from source
# instead.
apt-get install --yes gcc libc6-dev mercurial

if ! [ -d /home/vagrant/go ]; then
    hg clone -u release https://code.google.com/p/go
    cd go/src
    ./all.bash
fi

builddir=rtfblog
rm -rf /home/vagrant/$builddir
package=/home/vagrant/$builddir/package
mkdir /home/vagrant/$builddir
mkdir -p $package/sqlite-fix

sqlite="sqlite-autoconf-3071602"
if ! [ -d /home/vagrant/$sqlite ]; then
    wget -q http://www.sqlite.org/2013/$sqlite.tar.gz
    tar xzvf $sqlite.tar.gz
    cd $sqlite
    ./configure --enable-threadsafe
    make
fi

cp /vagrant/git-arch-for-deploy.tar.gz /home/vagrant/$builddir/
cd /home/vagrant/$builddir
tar xzvf git-arch-for-deploy.tar.gz
cp /vagrant/testdata/foo.db /home/vagrant/$builddir/
/home/vagrant/go/bin/go get
/home/vagrant/go/bin/go build
cd dbtool
/home/vagrant/go/bin/go build

cd /home/vagrant/$builddir/
cp /vagrant/server.conf $package
cp /vagrant/run $package
cp /home/vagrant/$builddir/rtfblog $package
cp -d /home/vagrant/$sqlite/.libs/libsqlite3.so* $package/sqlite-fix
cp -r /vagrant/static $package
cp -r /vagrant/tmpl $package
cp /vagrant/stuff/images/* $package/static/
cp /vagrant/testdata/foo.db $package/main.db
tar czvf package.tar.gz ./package
rm -rf $package

/vagrant/ssh_key_setup.sh
scp -q package.tar.gz rtfb@rtfb.lt:/home/rtfb/package.tar.gz
scp -q /vagrant/unpack.sh rtfb@rtfb.lt:/home/rtfb/unpack.sh
ssh rtfb@rtfb.lt /home/rtfb/unpack.sh
ssh rtfb@rtfb.lt "cd /home/rtfb/package; ./run &"
