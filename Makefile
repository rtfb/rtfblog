GOFMT=gofmt -s

GOFILES=\
	src/*.go\

BUILDDIR=build
JSDIR=${BUILDDIR}/static/js
CSSDIR=${BUILDDIR}/static/css

all: version vet fmt copy_static browserify grunt

grunt:
	grunt

run: all
	./${BUILDDIR}/rtfblog

copy_static:
	mkdir -p ${JSDIR}
	mkdir -p ${CSSDIR}
	cp js/*.js ${JSDIR}
	cp -r static/* ${BUILDDIR}/static/
	cp -r tmpl ${BUILDDIR}
	cp -r l10n ${BUILDDIR}
	cp server.conf ${BUILDDIR}

browserify:
	browserify js/main.js -o ${JSDIR}/bundle.js
	browserify -r pagedown-editor js/pgdown-ed.js -o ${JSDIR}/pagedown-bundle.js
	cp ./node_modules/pagedown-editor/wmd-buttons.png ${BUILDDIR}/static/
	cp ./node_modules/pagedown-editor/pagedown.css ${CSSDIR}
	cp ./bower_components/ribs/build/css/Ribs.css ${CSSDIR}

vet:
	go vet ${GOFILES}

version:
	@./scripts/genversion.sh > src/version.go

fmt:
	${GOFMT} -w ${GOFILES}

clean:
	rm -r ${BUILDDIR}
