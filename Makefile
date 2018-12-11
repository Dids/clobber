export GO111MODULE=on
export GOBIN := $(GOPATH)/bin
export PATH := $(GOBIN):$(PATH)

BINARY_VERSION?=0.0.1
EXTRA_FLAGS?=-mod=vendor

all: deps build
install:
	go install -v $(EXTRA_FLAGS) -ldflags "-X main.Version=$(BINARY_VERSION)" ./...
build:
	$(shell $(GOBIN)/packr2)
	ls
	go build -v $(EXTRA_FLAGS) -ldflags "-X main.Version=$(BINARY_VERSION)" ./...
test:
	go test -v $(EXTRA_FLAGS) -race -coverprofile=coverage.txt -covermode=atomic ./...
clean:
	go clean
	$(shell $(GOBIN)/packr2 clean)
	rm -f $(BINARY_NAME)
deps:
	go build -v $(EXTRA_FLAGS) ./...
	go get github.com/gobuffalo/packr/v2/packr2
upgrade:
	go get -u
	go get -u github.com/gobuffalo/packr/v2/packr2
version:
	clobber --version
print:
	@echo "PWD: $(shell pwd)"
	@echo "PATH: $(PATH)"
	@echo "GOPATH: $(GOPATH)"
