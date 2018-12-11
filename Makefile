export GO111MODULE=on

export PATH := $(GOPATH)/bin:$(PATH)

BINARY_VERSION?=0.0.1
BINARY_OUTPUT?=clobber
EXTRA_FLAGS?=-mod=vendor

.PHONY: all install build test clean deps upgrade version print

all: deps build

install:
	go install -v $(EXTRA_FLAGS) -ldflags "-X main.Version=$(BINARY_VERSION)"

build:
	packr2 clean
	packr2
	ls
	go build -v $(EXTRA_FLAGS) -ldflags "-X main.Version=$(BINARY_VERSION)" -o $(BINARY_OUTPUT)
	ls
	packr2 clean

test:
	go test -v $(EXTRA_FLAGS) -race -coverprofile=coverage.txt -covermode=atomic ./...

clean:
	go clean
	rm -f $(BINARY_NAME)

deps:
	go build -v $(EXTRA_FLAGS) ./...
	## FIXME: This reinstalls every time when running on Go >= v1.11
	go get github.com/gobuffalo/packr/v2/packr2

upgrade:
	go get -u ./...
	go get -u github.com/gobuffalo/packr/v2/packr2
	go mod vendor

version:
	clobber --version

print:
	@echo "PWD: $(shell pwd)"
	@echo "PATH: $(PATH)"
	@echo "GOPATH: $(GOPATH)"
	@echo "GOBIN: $(GOBIN)"
	@ls $(GOPATH)/bin
