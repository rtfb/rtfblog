GOFMT=gofmt -l -w -s

GOFILES= \
	embed.go \
	src/*.go \
	src/assets/*.go \
	src/htmltest/*.go

BUILDDIR=build
JSDIR=${BUILDDIR}/static/js
CSSDIR=${BUILDDIR}/static/css

CSS_FILES = $(notdir $(wildcard static/css/*.css))
PNG_FILES = $(notdir $(wildcard static/*.png))
TMPL_FILES = $(notdir $(wildcard tmpl/*.html))
L10N_FILES = $(notdir $(wildcard l10n/*.json))
JS_TARGETS = \
          ${JSDIR}/bundle.js \
          ${JSDIR}/pagedown-bundle.js \
          ${JSDIR}/tag-it.min.js \
          ${JSDIR}/jquery.min.js \
          ${JSDIR}/jquery-ui.min.js

JS_STATIC = \
          ${BUILDDIR}/static/wmd-buttons.png \
          ${CSSDIR}/pagedown.css \
          ${CSSDIR}/Ribs.css \
          ${CSSDIR}/jquery.tagit.css \
          ${CSSDIR}/tagit.ui-zendesk.css

TARGETS = \
		  $(addprefix $(CSSDIR)/, $(CSS_FILES)) \
		  $(addprefix ${BUILDDIR}/static/, $(PNG_FILES)) \
		  $(addprefix ${BUILDDIR}/tmpl/, $(TMPL_FILES)) \
		  $(addprefix ${BUILDDIR}/l10n/, $(L10N_FILES)) \
		  ${BUILDDIR}/static/robots.txt \
		  ${BUILDDIR}/default.db

ifneq ($(wildcard server.conf),)
	TARGETS += ${BUILDDIR}/server.conf
endif

GOPATH_HEAD = $(firstword $(subst :, ,$(GOPATH)))

all: ${BUILDDIR}/rtfblog

${BUILDDIR}/rtfblog: $(TARGETS) $(GOFILES)
	./scripts/version.sh > ${BUILDDIR}/version
	${GOFMT} ${GOFILES}
	go build -o ${BUILDDIR} ./cmd/rtfblog/...
	go test -v ./src/... -covermode=count -coverprofile=coverage.out
	go vet ./src/...
	cp -r ../jsbuild/* build/

jsbundles: ${JS_TARGETS} ${JS_STATIC}
	@echo "Done"

run: all
	./${BUILDDIR}/rtfblog

vet:
	go vet ${GOFILES}

fmt:
	${GOFMT} ${GOFILES}

${CSSDIR}/%.css: static/css/%.css
	@mkdir -p ${CSSDIR}
	cp $< $@

${BUILDDIR}/default.db: db/sqlite/migrations/*.sql
	$(GOPATH)/bin/migrate -path=db/sqlite/migrations -database="sqlite3://$@" up

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
	rm -r ${BUILDDIR}

.PHONY: all clean run vet fmt

APPNAME := rtfblog-dev

# builds the docker image. Add '--target=dev-env' or similar to build
# intermediate image
.PHONY: dbuild
dbuild:
	docker build -t ${APPNAME} .

# Run with '-it' locally, but without it on GH Actions:
ifeq ($(GITHUB_ACTIONS), true)
    DASH_IT=
else
    DASH_IT=-it
endif

# runs the container
.PHONY: drun
drun:
	docker run $(DASH_IT) --name ${APPNAME} --rm \
    --mount type=bind,source="$(shell pwd)",target=/home/rtfb/dev \
    --net=host ${APPNAME}:latest

# override entrypoint to gain interactive shell
.PHONY: dshell
dshell:
	docker run --entrypoint /bin/bash -it --name ${APPNAME} --rm \
    --mount type=bind,source="$(shell pwd)",target=/home/rtfb/dev \
    --net=host ${APPNAME}:latest
