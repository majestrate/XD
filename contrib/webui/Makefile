REPO := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
CSS_ROOT = $(REPO)/css
JS_ROOT = $(REPO)/lib
DOC_ROOT = $(REPO)/docroot
CSS = $(DOC_ROOT)/xd.css
JS = $(DOC_ROOT)/xd.min.js

CAT = cat
RM = rm -f

NODE = $(shell which node || true)

all: build

build: $(CSS) $(JS)

$(CSS):
	$(CAT) $(CSS_ROOT)/*.css > $(CSS)

$(JS):
	$(CAT) $(JS_ROOT)/*.js > $(JS)

run:
	$(NODE) server.js

clean:
	$(RM) $(CSS) $(JS)
