# rtfblog

![GHA build status](../../workflows/test/badge.svg)
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

Building is done from within a docker container, as I don't want to have any npm
present on my dev box. So first, build the building container, then run it,
which calls make in its entry point.

* `make dbuild`
* `make drun`

Alternatively, you can shell into the container and call make manually:

* `     host $ make dbuild`
* `     host $ make dshell`
* `container $ make all`

## Installing

You will need [migrate][migrate-url] for DB migration. Read
[Dockerfile](./Dockerfile) to get an overview of how to install it. Get it at:

    go install -tags 'postgres,sqlite3' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.15.2

You will need [go-bindata][go-bindata-url] to build embedded resources. Get it
at:

    go install github.com/go-bindata/go-bindata/go-bindata@latest

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
[baby-gopher-image]: https://raw.github.com/drnic/babygopher-site/gh-pages/images/babygopher-badge.png
[postgres-config]: http://stackoverflow.com/questions/1471571/how-to-configure-postgresql-for-the-first-time
[migrate-url]: https://github.com/golang-migrate/migrate
[go-bindata-url]: https://github.com/go-bindata/go-bindata
