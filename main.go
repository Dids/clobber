package main

import (
	"github.com/Dids/clobber/cmd"
)

// NOTE: This can be overridden when compiling with "go build"
var version = "0.0.1"

func main() {
	cmd.RootCmd.Version = version
	cmd.Execute()
}
