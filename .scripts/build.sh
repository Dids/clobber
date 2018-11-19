#!/usr/bin/env bash

# Setup error handling
set -o errexit   # Exit when a command fails (set -e)
set -o nounset   # Exit when using undeclared variables (set -u)
set -o pipefail  # Exit when piping fails
set -o xtrace    # Enable debugging (set -x)

# Setup overrideable arguments
VERSION=${1:-0.0.1}
OUTPUT=${2:-clobber}

# Define necessary paths
GOBIN=$GOPATH/bin
#REAL_CWD=$(pwd)
#CWD=${TRAVIS_BUILD_DIR:-$REAL_CWD}

# Remove intermediate files
#rm -fr clobber \
#       packrd
#find . -type f -name '*-packr.go' -delete

# Make sure Packr is installed
go get -u github.com/gobuffalo/packr/v2/packr2

# Prepare the Packr build command (this is a workaround for Homebrew)
#BUILD_CMD="go run $GOPATH/src/github.com/gobuffalo/packr/v2/packr2/main.go"

# Switch to the correct directory
#cd $GOPATH/src/github.com/Dids/clobber

# Clean the intermediate files
$GOBIN/packr2 clean

## TODO: Why would we need to do this?
rm -fr vendor

# Build the application
#$BUILD_CMD build -ldflags "-X main.Version=${VERSION}" -o ${OUTPUT} .
$GOBIN/packr2 build -ldflags "-X main.Version=${VERSION}" -o ${OUTPUT} .
#$GOBIN/packr2 build -ldflags "-X main.Version=${VERSION}" -o ${OUTPUT} ${CWD}

# Clean the intermediate files
$GOBIN/packr2 clean
