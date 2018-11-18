#!/usr/bin/env bash

# Setup error handling
set -o errexit   # Exit when a command fails (set -e)
set -o nounset   # Exit when using undeclared variables (set -u)
set -o pipefail  # Exit when piping fails
set -o xtrace    # Enable debugging (set -x)

# Setup overrideable arguments
VERSION=${1:-0.0.1}
OUTPUT=${2:-clobber}

# Make sure Packr is installed
go get -u github.com/gobuffalo/packr/packr

# Prepare the Packr build command (this is a workaround for Homebrew)
BUILD_CMD="go run $GOPATH/src/github.com/gobuffalo/packr/packr/main.go"
#BUILD_CMD="go run vendor/github.com/gobuffalo/packr/packr/main.go"

# Build the application
#go build -ldflags "-X main.Version=${VERSION}" -o ${OUTPUT}
#packr build -ldflags "-X main.Version=${VERSION}" -o ${OUTPUT}
$BUILD_CMD build -ldflags "-X main.Version=${VERSION}" -o ${OUTPUT}
