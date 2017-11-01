REPO := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
CSS_ROOT = $(REPO)/css
JS_ROOT = $(REPO)/lib
DOC_ROOT = $(REPO)/docroot
CSS = $(DOC_ROOT)/xd.css
JS = $(DOC_ROOT)/xd.min.js

all: build

build:
	cat $(CSS_ROOT)/*.css > $(CSS)
	cat $(JS_ROOT)/*.js > $(JS)

run:
	yarn run server

clean:
	rm -f $(CSS) $(JS)
