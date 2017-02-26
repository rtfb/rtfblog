# rtfblog

[![Build status][travis-image]][travis-url]
[![Test coverage][coveralls-image]][coveralls-url]

## What is this?

Rtfblog is a blog software that powers [my blog](http://blog.rtfb.lt), written
in [Go](http://golang.org). Its primary purpose is to serve my own needs, but I
was writing it with making it available to others in mind.

When I started it, this was my first sizeable piece of Go code (you can
certainly tell that if you look at early commits!), so I was a [baby
gopher](http://www.babygopher.org/):

[![baby-gopher][baby-gopher-image]](http://www.babygopher.org)

## Can I use it?

Yes. But I can't promise it would be easy at this point. You should be able to
build without problems, but installing and running will require reading some
deployment code. You will need a physical server or a VPS to run it. I have only
tried running it on Linux, but in theory it should run everywhere where Go runs.

## Building

Build instructions on Ubuntu 14.04 (older versions might differ a bit):

* `$ cat scripts/dev-packages.txt | xargs sudo apt-get install -y`
  * Note: make sure your $GOPATH and $GOBIN are set up (see the
    [docs](https://golang.org/cmd/go/#hdr-GOPATH_environment_variable))
* Node JS (you will need to use http://nodejs.org/download/ or follow
  [these](https://github.com/joyent/node/wiki/installing-node.js-via-package-manager)
  instructions, it doesn't work with the one provided by the package manager)
  * Note: make sure `node/bin/` dir is in `PATH`, build scripts assume that
* `$ npm install -g grunt-cli bower browserify json`
* `$ make`
  * Note: don't get surprised when make will take a lot of time and network
    activity on the first run, it's downloading dependencies.

## Installing

You will need [goose](https://github.com/steinbacher/goose) for DB migration.
Read [deploy.sh][deploy-sh-url] to get an overview of how I install it on my
server, you would need to do essentially the same. Get it at:

    go get github.com/steinbacher/goose/cmd/goose

You will need [go-bindata](https://github.com/jteeuwen/go-bindata) to build
embedded resources. Get it at:

    go get -u github.com/jteeuwen/go-bindata/...

## Configuration

### DB

Currently rtfblog supports [PostgreSQL](http://www.postgresql.org/) and
[SQLite](https://www.sqlite.org/). It ships with an empty `default.db` SQLite
database for immediate use.

[Here][postgres-config] is a useful quick start primer on how to configure
postgres for the first time.

### Config file

The server is meant to start and run without any configuration whatsoever.
Currently that is not fully achieved, since the server requires at least a DB
connection being configured. But it should at least start and serve the static
content.

The [sample-server.conf](sample-server.conf) should suit your needs with few
simple modifications. The server is looking for config in these locations, in
this order:

* `/etc/rtfblogrc`
* `$HOME/.rtfblogrc`
* `./.rtfblogrc`
* `./server.conf`

All the files found will be read, in the order specified above, with options
in more specific locations overriding the more generic ones. So if you happen to
run a few instances, you can have global stuff configured in one place, leaving
local tweaks for each instance.

## License

BSD Simplified, see [LICENSE.md](LICENSE.md).

[travis-image]: https://travis-ci.org/rtfb/rtfblog.svg?branch=master
[travis-url]: https://travis-ci.org/rtfb/rtfblog
[coveralls-image]: https://coveralls.io/repos/rtfb/rtfblog/badge.png
[coveralls-url]: https://coveralls.io/r/rtfb/rtfblog
[deploy-sh-url]: https://github.com/rtfb/blog-rtfb-lt/blob/master/scripts/deploy.sh
[baby-gopher-image]: https://raw.github.com/drnic/babygopher-site/gh-pages/images/babygopher-badge.png
[postgres-config]: http://stackoverflow.com/questions/1471571/how-to-configure-postgresql-for-the-first-time
