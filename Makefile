export GO111MODULE=on

BINARY_VERSION?=0.0.1

all: deps build
install:
	go install -v -ldflags "-X main.Version=$(BINARY_VERSION)" ./...
build:
	go build -v -ldflags "-X main.Version=$(BINARY_VERSION)"
test:
	go test -v -mod=vendor -race -coverprofile=coverage.txt -covermode=atomic ./...
test-no-vendor:
	go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...
clean:
	go clean
	rm -f $(BINARY_NAME)
deps:
	go build -v ./...
upgrade:
	go get -u
version:
	clobber --version
