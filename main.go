package main

import (
	"fmt"

	"github.com/Dids/clobber/cmd"
	figure "github.com/common-nighthawk/go-figure"
)

// Version is set dynamically when building
var Version = "0.0.1"

func main() {
	// TODO: Updates disabled for now
	/*updateAvailable, err := util.CheckForUpdates(Version)
	if err != nil {
		// TODO: Should we just silently fail to check for updates?
		panic(err)
	}
	if updateAvailable {
		fmt.Println()
		fmt.Println("NOTICE: A new version of Clobber is available. Please run 'brew upgrade clobber' to update.")
		fmt.Println()
	}*/

	logo := figure.NewFigure("CLOBBER", "puffy", true)
	logo.Print()
	//fmt.Println()
	fmt.Println("                                  v" + Version + " by @Dids")
	fmt.Println()

	cmd.RootCmd.Version = Version
	cmd.Execute()
}
