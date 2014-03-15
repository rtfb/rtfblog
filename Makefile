GOFMT=gofmt -s -tabs=false -tabwidth=4

GOFILES=\
	*.go\
	dbtool/*.go

all:
	go build
	./rtfblog

fmt:
	${GOFMT} -w ${GOFILES}
