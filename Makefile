GOFMT=gofmt -l -w -s
GO_DEPS_CMD=\
	go list -f '{{ join .Deps  "\n"}}' ./src | grep "github\|code.google.com"

GOFILES=\
	src/*.go

BUILDDIR=build
JSDIR=${BUILDDIR}/static/js
CSSDIR=${BUILDDIR}/static/css

JS_FILES = $(notdir $(wildcard js/*.js))
CSS_FILES = $(notdir $(wildcard static/css/*.css))
PNG_FILES = $(notdir $(wildcard static/*.png))
TMPL_FILES = $(notdir $(wildcard tmpl/*.html))
L10N_FILES = $(notdir $(wildcard l10n/*.json))
TARGETS = $(addprefix $(JSDIR)/, $(JS_FILES)) \
		  $(addprefix $(CSSDIR)/, $(CSS_FILES)) \
		  $(addprefix ${BUILDDIR}/static/, $(PNG_FILES)) \
		  $(addprefix ${BUILDDIR}/tmpl/, $(TMPL_FILES)) \
		  $(addprefix ${BUILDDIR}/l10n/, $(L10N_FILES)) \
		  ${BUILDDIR}/static/robots.txt \
		  ${BUILDDIR}/server.conf \
		  ${JSDIR}/bundle.js \
		  ${JSDIR}/pagedown-bundle.js \
		  ${BUILDDIR}/static/wmd-buttons.png \
		  ${CSSDIR}/pagedown.css \
		  ${CSSDIR}/Ribs.css
GO_DEPS = $(addprefix $(GOPATH)/src/, ${shell ${GO_DEPS_CMD}})

all: vet fmt ${BUILDDIR}/rtfblog

${BUILDDIR}/rtfblog: $(GO_DEPS) $(GOFILES) $(TARGETS) src/version.go
	grunt

$(GO_DEPS):
	go get -t ./...

run: all
	./${BUILDDIR}/rtfblog

vet:
	go vet ${GOFILES}

src/version.go:
	./scripts/genversion.sh > src/version.go

fmt:
	${GOFMT} ${GOFILES}

${JSDIR}/%.js: js/%.js
	@mkdir -p ${JSDIR}
	cp $< $@

${CSSDIR}/%.css: static/css/%.css
	@mkdir -p ${CSSDIR}
	cp $< $@

${BUILDDIR}/static/%.png: static/%.png
	cp $< $@

${BUILDDIR}/tmpl/%.html: tmpl/%.html
	@mkdir -p ${BUILDDIR}/tmpl
	cp $< $@

${BUILDDIR}/l10n/%.json: l10n/%.json
	@mkdir -p ${BUILDDIR}/l10n
	cp $< $@

${BUILDDIR}/static/robots.txt: static/robots.txt
	cp $< $@

${BUILDDIR}/server.conf: server.conf
	cp $< $@

${JSDIR}/bundle.js: js/main.js
	browserify $< -o $@

${JSDIR}/pagedown-bundle.js: js/pgdown-ed.js
	browserify -r pagedown-editor $< -o $@

${BUILDDIR}/static/wmd-buttons.png: node_modules/pagedown-editor/wmd-buttons.png
	cp $< $@

${CSSDIR}/pagedown.css: node_modules/pagedown-editor/pagedown.css
	cp $< $@

${CSSDIR}/Ribs.css: bower_components/ribs/build/css/Ribs.css
	cp $< $@

clean:
	rm -r ${BUILDDIR}

.PHONY: all clean grunt run vet version fmt
