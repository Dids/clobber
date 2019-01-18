SHELL := /bin/bash

export GO111MODULE=on
export GOPROXY=

export PATH := $(GOPATH)/bin:$(PATH)

BINARY_VERSION?=0.0.1
BINARY_OUTPUT?=clobber
EXTRA_FLAGS?=-mod=vendor

define timed_function
	@d=$$(date +%s); \
	$(shell echo $1); \
	echo "=> Ran $1 in $$(($$(date +%s)-d)) seconds"
endef

.PHONY: all install uninstall build test clean deps upgrade tidy version

all: deps build

install: deps build
	$(call timed_function,'go install -v $(EXTRA_FLAGS) -ldflags "-X main.Version=$(BINARY_VERSION)"')

uninstall:
	$(call timed_function,'rm -f $(GOPATH)/bin/$(BINARY_OUTPUT)')

build:
	$(call timed_function,'$(GOPATH)/bin/packr2 clean')
	$(call timed_function,'$(GOPATH)/bin/packr2')
	$(call timed_function,'go build -v $(EXTRA_FLAGS) -ldflags "-X main.Version=$(BINARY_VERSION)" -o $(BINARY_OUTPUT)')
	$(call timed_function,'$(GOPATH)/bin/packr2 clean')

test:
	$(call timed_function,'go test -v $(EXTRA_FLAGS) -race -coverprofile=coverage.txt -covermode=atomic ./...')

clean:
	$(call timed_function,'go clean')
	$(call timed_function,'rm -f $(BINARY_OUTPUT)')

deps:
	$(call timed_function,'go build -v $(EXTRA_FLAGS) ./...')
	$(call timed_function,'go get github.com/gobuffalo/packr/v2/packr2')

upgrade:
	$(call timed_function,'go get -u ./...')
	$(call timed_function,'go get -u github.com/gobuffalo/packr/v2/packr2')
	$(call timed_function,'go mod vendor')

tidy:
	$(call timed_function,'go mod tidy')

version:
	$(call timed_function,'clobber --version')
