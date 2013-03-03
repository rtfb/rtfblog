GOFMT=gofmt -s -tabs=false -tabwidth=4

GOFILES=\
	rtfblog.go\
	rtfblog_test.go\
	dbtool/dbtool.go\
	dbtool/b2e-import.go\

all:
	go build
	./rtfblog

format:
	${GOFMT} -w ${GOFILES}
