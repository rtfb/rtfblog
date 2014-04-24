GOFMT=gofmt -s -tabs=false -tabwidth=4

GOFILES=\
	*.go\
	dbtool/*.go

all: fmt browserify
	grunt
	./rtfblog

browserify:
	browserify js/main.js -o static/js/bundle.js

fmt:
	${GOFMT} -w ${GOFILES}
