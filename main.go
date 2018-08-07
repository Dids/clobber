package main

import (
	"github.com/Dids/clobber/cmd"
	"github.com/Dids/clobber/util"
)

// Version is set dynamically when building
var Version = "0.0.1"

func main() {
	util.CheckForUpdates(Version)
	cmd.RootCmd.Version = Version
	cmd.Execute()
}
