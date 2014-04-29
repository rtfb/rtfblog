# rtfblog

[![Build Status](https://travis-ci.org/rtfb/rtfblog.svg?branch=master)](https://travis-ci.org/rtfb/rtfblog)

## What is this?

Rtfblog is a blog software that powers [my blog](http://blog.rtfb.lt), written
in [Go](http://golang.org). Its primary purpose is to serve my own needs, but I
was writing it with making it available to others in mind.

This is my first sizeable piece of Go code, so I'm a [baby
gopher](http://www.babygopher.org/):

[![baby-gopher](https://raw2.github.com/drnic/babygopher-site/gh-pages/images/babygopher-badge.png)](http://www.babygopher.org)

## Can I use it?

Yes. But I can't promise it would be easy at this point. You should be able to
build without problems, but installing and running will require reading some
deployment code. You will need a physical server or a VPS to run it. I have only
tried running it on Linux, but in theory it should run everywhere where Go runs.

## Building

`go get`, `go build`.

## Installing

You will need [goose](https://bitbucket.org/liamstask/goose/) for DB migration.
Read [deploy.sh](deploy.sh) to get an overview of how I install it on my server,
you would need to do essentially the same.

## Configuration

### DB (postgres)

Currently rtfblog only supports [PostgreSQL](http://www.postgresql.org/).
[Here](http://stackoverflow.com/questions/1471571/how-to-configure-postgresql-for-the-first-time)
is a useful quick start primer on how to configure postgres for the first time.

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