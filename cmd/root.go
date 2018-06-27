package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"log"

	git "github.com/gogits/git-module"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
)

// Verbose enables global verbose output
var Verbose bool

// Execute is the entrypoint for the command-line application
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

type logWriter struct{}

func (writer logWriter) Write(bytes []byte) (int, error) {
	return fmt.Print(time.Now().UTC().Format("2006-01-02T15:04:05.999Z") + " [VERBOSE] " + string(bytes))
}

var rootCmd = &cobra.Command{
	Use:   "clover-builder",
	Short: "Clover Builder is a command-line application for building Clover",
	Long: `Clover Builder is a command-line application for building Clover.
				 Built by @Dids, with tons and tons of love, sweat and tears.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Measure execution time
		executionStartTime := time.Now()

		if Verbose {
			log.Println("Verbose mode is enabled")
			if args != nil && len(args) > 0 {
				log.Println("Building with arguments:", args)
			}
		}

		// TODO: Do we just blindly run everything here, or split stuff into their own packages and/or functions here?

		// TODO: Do everything in a "chrooted" environment, which we might be able to do just by overriding $HOME?

		// https://github.com/tianocore/edk2 and the branch is UDK2018

		srcRoot := "/tmp/clover-builder_go/src"
		udkRoot := srcRoot + "/edk2"
		cloverRoot := udkRoot + "/clover"

		// Make sure that the correct directory structure exists
		log.Println("Verifying folder structure..")
		mkdirErr := os.MkdirAll(srcRoot, 0755)
		if mkdirErr != nil {
			fmt.Println(mkdirErr)
			os.Exit(1)
		}

		// Download or update UDK2018
		if _, err := os.Stat(udkRoot + "/.git"); os.IsNotExist(err) {
			log.Println("UDK2018 is missing, downloading..")
			git.Clone("https://github.com/tianocore/edk2", udkRoot, git.CloneRepoOptions{Branch: "UDK2018", Bare: false, Quiet: false})
		}
		log.Println("Verifying UDK2018 is up to date..")
		git.Checkout(srcRoot, git.CheckoutOptions{Branch: "UDK2018"})

		// Download or update Clover
		if _, err := os.Stat(cloverRoot + "/.svn"); os.IsNotExist(err) {
			log.Println("Clover is missing, downloading..")
			// TODO: Figure out how to do svn (probably need to use exec)
			//git.Clone("https://github.com/tianocore/edk2", udkRoot, git.CloneRepoOptions{Branch: "UDK2018", Bare: false, Quiet: false})
		}
		log.Println("Verifying Clover is up to date..")
		// TODO: Figure out how to do svn (probably need to use exec)
		//git.Checkout(srcRoot, git.CheckoutOptions{Branch: "UDK2018"})

		executionElapsedTime := time.Since(executionStartTime)
		log.Printf("Finished in %s!", executionElapsedTime)
	},
}

func init() {
	// Custom initialization logic
	cobra.OnInitialize(customInit)

	// Set the version field to add a "--version" flag automatically
	rootCmd.Version = "0.0.1"

	// Add persistent flags that carry over to all commands
	rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "enable verbose output")
}

func customInit() {
	// Initialize logging
	log.SetFlags(0)
	log.SetOutput(new(logWriter))

	// Disable logging if not running in verbose mode
	if Verbose == false {
		log.SetOutput(ioutil.Discard)
	}
}

/* initConfig() {
	// Don't forget to read config either from cfgFile or from home directory!
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".cobra" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".cobra")
	}

	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("Can't read config:", err)
		os.Exit(1)
	}
}*/

// TODO: Add helper functions for getting common paths, such as ~/, ~/src, ~/src/UDK2018 etc.
func getHome() string {
	home, err := homedir.Dir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return home
}
