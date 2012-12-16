GOFMT=gofmt -s -tabs=false -tabwidth=4

GOFILES=\
	rtfblog.go\
	rtfblog_test.go\

all:
	go build
	./rtfblog

format:
	${GOFMT} -w ${GOFILES}
