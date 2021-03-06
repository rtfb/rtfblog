GOFMT=gofmt -l -w -s

NODE_DEPS_CMD=\
	cat package.json | jq '.devDependencies | keys[]' | xargs

BOWER_DEPS_CMD=\
	cat bower.json | jq '.dependencies | keys[]' | xargs

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
TARGETS = \
		  $(addprefix $(CSSDIR)/, $(CSS_FILES)) \
		  $(addprefix ${BUILDDIR}/static/, $(PNG_FILES)) \
		  $(addprefix ${BUILDDIR}/tmpl/, $(TMPL_FILES)) \
		  $(addprefix ${BUILDDIR}/l10n/, $(L10N_FILES)) \
		  ${BUILDDIR}/static/robots.txt \
		  ${BUILDDIR}/default.db \
		  ${JSDIR}/bundle.js \
		  ${JSDIR}/pagedown-bundle.js \
		  ${JSDIR}/tag-it.min.js \
		  ${JSDIR}/jquery.min.js \
		  ${JSDIR}/jquery-ui.min.js \
		  ${BUILDDIR}/static/wmd-buttons.png \
		  ${CSSDIR}/pagedown.css \
		  ${CSSDIR}/jquery.tagit.css \
		  ${CSSDIR}/tagit.ui-zendesk.css \
		  ${CSSDIR}/Ribs.css

ifneq ($(wildcard server.conf),)
	TARGETS += ${BUILDDIR}/server.conf
endif

GOPATH_HEAD = $(firstword $(subst :, ,$(GOPATH)))
NODE_DEPS = $(addprefix node_modules/, ${shell ${NODE_DEPS_CMD}})
BOWER_DEPS = $(addprefix bower_components/, ${shell ${BOWER_DEPS_CMD}})
ASSETS_PKG = src/rtfblog_resources

all: ${BUILDDIR}/rtfblog

${BUILDDIR}/rtfblog: $(NODE_DEPS) $(BOWER_DEPS) \
                     $(GOFILES) $(ASSETS_PKG)
	go vet ./src/...
	${GOFMT} ${GOFILES}
	grunt
	go build -i -o ${BUILDDIR} \
		-ldflags "-X main.genVer=$(shell scripts/version.sh)" ./cmd/rtfblog/...
	go test ./src/...

$(NODE_DEPS):
	npm install

$(BOWER_DEPS):
	bower install --config.interactive=false

$(ASSETS_PKG): $(TARGETS)
	go-bindata -pkg rtfblog_resources -o $@/res.go -prefix ${BUILDDIR} \
		${BUILDDIR}/l10n \
		${BUILDDIR}/default.db \
		${BUILDDIR}/static/... \
		${BUILDDIR}/tmpl

run: all
	./${BUILDDIR}/rtfblog

vet:
	go vet ${GOFILES}

fmt:
	${GOFMT} ${GOFILES}

${CSSDIR}/%.css: static/css/%.css
	@mkdir -p ${CSSDIR}
	cp $< $@

${BUILDDIR}/default.db: db/sqlite/dbconf.yml db/sqlite/migrations/*.sql
	RTFBLOG_DB_URL=$@ RTFBLOG_DB_DRIVER=sqlite3 goose -path=db/sqlite/ -env=production up

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
	@mkdir -p ${JSDIR}
	browserify $< -o $@

${JSDIR}/pagedown-bundle.js: js/pgdown-ed.js
	@mkdir -p ${JSDIR}
	browserify -r pagedown-editor $< -o $@

${JSDIR}/tag-it.min.js: bower_components/tag-it/js/tag-it.min.js
	cp $< $@

${JSDIR}/jquery.min.js: bower_components/jquery/dist/jquery.min.js
	cp $< $@

${JSDIR}/jquery-ui.min.js: bower_components/jquery-ui/jquery-ui.min.js
	cp $< $@

${BUILDDIR}/static/wmd-buttons.png: node_modules/pagedown-editor/wmd-buttons.png
	cp $< $@

${CSSDIR}/pagedown.css: node_modules/pagedown-editor/pagedown.css
	cp $< $@

${CSSDIR}/Ribs.css: bower_components/ribs/build/css/Ribs.css
	cp $< $@

${CSSDIR}/jquery.tagit.css: bower_components/tag-it/css/jquery.tagit.css
	cp $< $@

${CSSDIR}/tagit.ui-zendesk.css: bower_components/tag-it/css/tagit.ui-zendesk.css
	cp $< $@

clean:
	rm -r $(ASSETS_PKG)
	rm -r ${BUILDDIR}

.PHONY: all clean run vet fmt
