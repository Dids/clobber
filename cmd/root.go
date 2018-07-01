package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

	"log"

	"github.com/briandowns/spinner"
	git "github.com/gogits/git-module"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/otiai10/copy"
	"github.com/spf13/cobra"
)

// Verbose enables global verbose output
var Verbose bool

// Quiet enables silencing all output
var Quiet bool

// Revision is the default Clover revision to use
var Revision string

// Spinner is the CLI spinner/activity indicator
var Spinner = spinner.New(spinner.CharSets[9], 100*time.Millisecond)

// Execute is the entrypoint for the command-line application
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

type logWriter struct{}

func (writer logWriter) Write(bytes []byte) (int, error) {
	// TODO: This is redundant, but perhaps it's possible to use this to determine if the message is an error or not?
	if Quiet {
		return 0, nil
	}

	// FIXME: Using suffix or prefix causes newlines, while FinalMsg doesn't, but it appears on a separate line with this logic..
	if !Verbose {
		spinnerSuffix := "" + string(bytes)
		if Spinner.FinalMSG != spinnerSuffix {
			Spinner.FinalMSG = spinnerSuffix
			Spinner.Stop()
			Spinner.Start()
		}
		//Spinner.Suffix = "  :" + string(bytes)
		return 0, nil
	}

	return fmt.Print(time.Now().UTC().Format("2006-01-02T15:04:05.999Z") + " " + string(bytes))
}

var rootCmd = &cobra.Command{
	Use:   "clobber",
	Short: "Clobber is a command-line application for building Clover",
	Long: `Clobber is a command-line application for building Clover.
				 Built by @Dids, with tons and tons of love, sweat and tears.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Measure execution time
		executionStartTime := time.Now()

		// Start the spinner
		if !Verbose {
			Spinner.Start()
		}

		if Verbose {
			log.Println("Verbose mode is enabled")
			log.Println("Target Clover revision:", Revision)
			if args != nil && len(args) > 0 {
				log.Println("Building with arguments:", args)
			}
		}

		// FIXME: Ditch the "git" package and just use our custom exec-based logic, so it's more consistent across the app?

		// TODO: Do we just blindly run everything here, or split stuff into their own packages and/or functions here?

		// TODO: Do everything in a "chrooted" environment, which we might be able to do just by overriding $HOME?

		// Make sure that the correct directory structure exists
		log.Println("Verifying folder structure..")
		mkdirErr := os.MkdirAll(getSourcePath(), 0755)
		if mkdirErr != nil {
			log.Fatal("MkdirAll failed with error: ", mkdirErr)
		}

		// Download or update UDK2018
		if _, err := os.Stat(getUdkPath() + "/.git"); os.IsNotExist(err) {
			log.Println("UDK2018 is missing, downloading..")
			git.Clone("https://github.com/tianocore/edk2", getUdkPath(), git.CloneRepoOptions{Branch: "UDK2018", Bare: false, Quiet: Verbose})
		}
		log.Println("Verifying UDK2018 is up to date..")
		git.Checkout(getSourcePath(), git.CheckoutOptions{Branch: "UDK2018"})
		runCommand("git clean -fdx --exclude=\"Clover/\"")

		// Download or update Clover
		if _, err := os.Stat(getCloverPath() + "/.svn"); os.IsNotExist(err) {
			log.Println("Clover is missing, downloading..")
			runCommand("svn co " + "https://svn.code.sf.net/p/cloverefiboot/code" + " " + getCloverPath())
		}
		log.Println("Verifying Clover is up to date..")
		runCommand("svn up -r" + Revision + " " + getCloverPath())
		runCommand("svn revert -R" + " " + getCloverPath())
		runCommand("svn cleanup --remove-unversioned " + getCloverPath())

		// Override HOME environment variable (use chroot-like logic for the build process)
		log.Println("Overriding HOME..")
		os.Setenv("HOME", getClobberPath())

		// Override TOOLCHAIR_DIR environment variable
		log.Println("Overriding TOOLCHAIN_DIR..")
		os.Setenv("TOOLCHAIN_DIR", getSourcePath()+"/opt/local")

		// Build base tools
		log.Println("Building base tools..")
		runCommand("make -C" + " " + getUdkPath() + "/BaseTools/Source/C")

		// Setup UDK
		log.Println("Setting up UDK..")
		// source edksetup.sh
		runCommand("cd " + getUdkPath() + " && " + "source edksetup.sh") // TODO: Why does this work, because I thought "cd" didn't work with exec?

		// Build gettext, mtoc and nasm (if necessary)
		if _, err := os.Stat(getSourcePath() + "/opt/local/bin/gettext"); os.IsNotExist(err) {
			log.Println("Building gettext..")
			runCommand(getCloverPath() + "/buildgettext.sh")
		}
		if _, err := os.Stat(getSourcePath() + "/opt/local/bin/mtoc.NEW"); os.IsNotExist(err) {
			log.Println("Building mtoc..")
			runCommand(getCloverPath() + "/buildmtoc.sh")
		}
		if _, err := os.Stat(getSourcePath() + "/opt/local/bin/nasm"); os.IsNotExist(err) {
			log.Println("Building nasm..")
			runCommand(getCloverPath() + "/buildnasm.sh")
		}

		// Apply UDK patches
		log.Println("Applying patches for UDK..")
		copyErr := copy.Copy(getCloverPath()+"/Patches_for_UDK2018", getUdkPath())
		if copyErr != nil {
			log.Fatal("Failed to copy UDK patches: ", copyErr)
		}

		// Build Clover (clean & build)
		log.Println("Building Clover..")
		runCommand(getCloverPath() + "/ebuild.sh -cleanall")
		runCommand(getCloverPath() + "/ebuild.sh -fr")

		// TODO: Modify credits

		// TODO: Add custom drivers (apfs.efi, apfs_patched.efi, ntfs.efi, hfsplus.efi, AptioFixPkg, ApfsSupportPkg)

		// TODO: Update template resource descriptions

		// Build the Clover installer package
		log.Println("Building Clover installer..")
		runCommand(getCloverPath() + "/CloverPackage/makepkg")

		// TODO: Would be nice to have a better formatting for the time string (eg. 1 minute and 20 seconds, instead of 1m20s)
		// Stop the execution timer
		executionElapsedTime := time.Since(executionStartTime)
		executionResult := fmt.Sprintf("Finished in %s!\n", executionElapsedTime)

		// Stop the spinner
		if !Verbose {
			Spinner.FinalMSG = executionResult
			Spinner.Stop()
		} else {
			log.Printf(executionResult)
		}
	},
}

func init() {
	// Custom initialization logic
	cobra.OnInitialize(customInit)

	// Set the version field to add a "--version" flag automatically
	rootCmd.Version = "0.0.1"

	// Add persistent flags that carry over to all commands
	rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "enable verbose output")
	rootCmd.PersistentFlags().BoolVarP(&Quiet, "quiet", "q", false, "silence all output")
	rootCmd.PersistentFlags().StringVarP(&Revision, "revision", "r", "HEAD", "Clover target revision")
}

func customInit() {
	// Initialize logging
	log.SetFlags(0)
	log.SetOutput(new(logWriter))

	// FIXME: Ideally log.Fatal should still work when this is set, but not sure if the log package supports that?
	// Disable logging if running in quiet mode
	if Quiet == true {
		log.SetOutput(ioutil.Discard)
	}
}

// TODO: Implement some sort of persistent config (if we're planning on allowing customizable builds?)
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

func runCommand(command string) {
	// If there are no args (or no spaces), we need to deal with those situations too
	var (
		cmd        string
		args       []string
		argsString string
	)
	if strings.Contains(command, " ") {
		splitArgs := strings.Split(command, " ")
		cmd = splitArgs[0]
		args = strings.Split(command[len(cmd)+1:len(command)], " ")
		argsString = strings.Join(args, " ")
	} else {
		cmd = command
		argsString = ""
	}

	if Verbose {
		log.Println("Running command: '" + cmd + " " + argsString + "'")
	}

	var (
		cmdOut []byte
		err    error
	)
	if cmdOut, err = exec.Command(cmd, args...).CombinedOutput(); err != nil {
		//log.Fatal("Failed to run '" + cmd + strings.Join(args, " ") + "':\n" + string(cmdOut) + " (" + err.Error() + ")")
		log.Fatal("Failed to run '" + cmd + " " + argsString + "':\n" + string(cmdOut))
	}
	if Verbose {
		log.Println("Command finished with output:\n" + string(cmdOut))
	}
}

func getCloverPath() string {
	return getUdkPath() + "/Clover"
}

func getUdkPath() string {
	return getSourcePath() + "/edk2"
}

func getSourcePath() string {
	return getClobberPath() + "/src"
}

func getClobberPath() string {
	return getHomePath() + "/.clobber"
}

func getHomePath() string {
	home, err := homedir.Dir()
	if err != nil {
		log.Fatal("getHomePath failed with error: ", err)
	}
	return home
}

// FIXME: d.Round doesn't exist in go versions <= 1.8
/*func fmtDuration(d time.Duration) string {
	d = d.Round(time.Minute)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	return fmt.Sprintf("%02d:%02d", h, m)
}*/
