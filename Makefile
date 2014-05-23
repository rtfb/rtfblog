GOFMT=gofmt -s -tabs=false -tabwidth=4

GOFILES=\
	src/*.go\
	dbtool/*.go

all: vet fmt browserify grunt

grunt:
	grunt

run: all
	./rtfblog

browserify:
	mkdir -p static/js
	browserify js/main.js -o static/js/bundle.js
	browserify -r pagedown-editor js/pgdown-ed.js -o static/js/pagedown-bundle.js
	cp ./node_modules/pagedown-editor/wmd-buttons.png static/

vet:
	go vet src/*.go

fmt:
	${GOFMT} -w ${GOFILES}
