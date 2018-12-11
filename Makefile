export GO111MODULE=on

BINARY_NAME?=clobber
BINARY_VERSION?=0.0.1

all: deps build
install:
	go install -v ./...
build:
	go build -v -ldflags "-X main.Version=$(BINARY_VERSION)" -o $(BINARY_NAME)
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
