GOFMT=gofmt -s -tabs=false -tabwidth=4

GOFILES=\
	rtfblog.go\
	data.go\
	db.go\
	rtfblog_test.go\
	util/util.go\
	dbtool/dbtool.go\
	dbtool/b2e-import.go\

all:
	go build
	./rtfblog

fmt:
	${GOFMT} -w ${GOFILES}
