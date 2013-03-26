GOFMT=gofmt -s -tabs=false -tabwidth=4

GOFILES=\
	rtfblog.go\
	rtfblog_test.go\
	util/util.go\
	dbtool/dbtool.go\
	dbtool/b2e-import.go\

all:
	go build
	./rtfblog

fmt:
	${GOFMT} -w ${GOFILES}

package=./package

pack:
	-rm -rf $(package)
	-mkdir -p $(package)/dbtool
	-mkdir -p $(package)/util
	-cp *.go $(package)
	-cp util/util.go $(package)/util
	-cp dbtool/*.go $(package)/dbtool
	-cp Makefile $(package)
	-cp sample-server.conf $(package)
	-cp -r static $(package)
	-cp -r tmpl $(package)
	-cp stuff/images/* $(package)/static/
	./dbtool/dbtool -db=./testdata/db.conf -src=./testdata/legacy-db.conf -notest
	-cp ./testdata/foo.db $(package)/main.db
	-tar czvf package.tar.gz $(package)
	-rm -rf $(package)
