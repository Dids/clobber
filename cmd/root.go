package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Dids/clobber/util"
	"github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"

	"github.com/briandowns/spinner"
	git "github.com/gogits/git-module"
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

var log = logrus.New()

// Execute is the entrypoint for the command-line application
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

/*type logWriter struct{}

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
}*/

// ClobberLogFormatter is a custom log formatter
type ClobberLogFormatter struct {
}

// Format formats logs with the custom log format
func (f *ClobberLogFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	return []byte(entry.Message + "\n"), nil
}

var rootCmd = &cobra.Command{
	Use:   "clobber",
	Short: "Clobber is a command-line application for building Clover",
	Long: `Clobber is a command-line application for building Clover.
				 Built by @Dids with tons of love, sweat and tears.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Measure execution time
		executionStartTime := time.Now()

		// Start the spinner
		if !Verbose && !Quiet {
			Spinner.Start()
		}

		if Verbose {
			log.Debug("Verbose mode is enabled")
			log.Debug("Target Clover revision:", Revision)
			if args != nil && len(args) > 0 {
				log.Debug("Building with arguments:", args)
			}
		}

		// FIXME: Ditch the "git" package and just use our custom exec-based logic, so it's more consistent across the app?

		// TODO: Do we just blindly run everything here, or split stuff into their own packages and/or functions here?

		// TODO: Do everything in a "chrooted" environment, which we might be able to do just by overriding $HOME?

		// Make sure that the correct directory structure exists
		log.Info("Verifying folder structure..")
		mkdirErr := os.MkdirAll(util.GetSourcePath(), 0755)
		if mkdirErr != nil {
			log.Fatal("MkdirAll failed with error: ", mkdirErr)
		}

		// Download or update UDK2018
		if _, err := os.Stat(util.GetUdkPath() + "/.git"); os.IsNotExist(err) {
			log.Warning("UDK2018 is missing, downloading..")
			git.Clone("https://github.com/tianocore/edk2", util.GetUdkPath(), git.CloneRepoOptions{Branch: "UDK2018", Bare: false, Quiet: Verbose})
		}
		log.Info("Verifying UDK2018 is up to date..")
		git.Checkout(util.GetSourcePath(), git.CheckoutOptions{Branch: "UDK2018"})
		runCommand("cd " + util.GetUdkPath() + " && git clean -fdx --exclude=\"Clover/\"")

		// Download or update Clover
		if _, err := os.Stat(util.GetCloverPath() + "/.svn"); os.IsNotExist(err) {
			log.Warning("Clover is missing, downloading..")
			runCommand("svn co " + "https://svn.code.sf.net/p/cloverefiboot/code" + " " + util.GetCloverPath())
		}
		log.Info("Verifying Clover is up to date..")
		runCommand("svn up -r" + Revision + " " + util.GetCloverPath())
		runCommand("svn revert -R" + " " + util.GetCloverPath())
		runCommand("svn cleanup --remove-unversioned " + util.GetCloverPath())

		// Override HOME environment variable (use chroot-like logic for the build process)
		log.Info("Overriding HOME..")
		os.Setenv("HOME", util.GetClobberPath())

		// Override TOOLCHAIR_DIR environment variable
		log.Info("Overriding TOOLCHAIN_DIR..")
		os.Setenv("TOOLCHAIN_DIR", util.GetSourcePath()+"/opt/local")

		// Build base tools
		log.Info("Building base tools..")
		runCommand("make -C" + " " + util.GetUdkPath() + "/BaseTools/Source/C")

		// Setup UDK
		log.Info("Setting up UDK..")
		// source edksetup.sh
		runCommand("cd " + util.GetUdkPath() + " && " + "source edksetup.sh") // TODO: Why does this work, because I thought "cd" didn't work with exec?

		// Build gettext, mtoc and nasm (if necessary)
		if _, err := os.Stat(util.GetSourcePath() + "/opt/local/bin/gettext"); os.IsNotExist(err) {
			log.Warning("Building gettext..")
			runCommand(util.GetCloverPath() + "/buildgettext.sh")
		}
		if _, err := os.Stat(util.GetSourcePath() + "/opt/local/bin/mtoc.NEW"); os.IsNotExist(err) {
			log.Warning("Building mtoc..")
			runCommand(util.GetCloverPath() + "/buildmtoc.sh")
		}
		if _, err := os.Stat(util.GetSourcePath() + "/opt/local/bin/nasm"); os.IsNotExist(err) {
			log.Warning("Building nasm..")
			runCommand(util.GetCloverPath() + "/buildnasm.sh")
		}

		// Apply UDK patches
		log.Info("Applying patches for UDK..")
		copyErr := copy.Copy(util.GetCloverPath()+"/Patches_for_UDK2018", util.GetUdkPath())
		if copyErr != nil {
			log.Fatal("Failed to copy UDK patches: ", copyErr)
		}

		// Build Clover (clean & build, with extras like ApfsDriverLoader checked out and compiled)
		log.Info("Building Clover..")
		runCommand(util.GetCloverPath() + "/ebuild.sh -cleanall")
		runCommand(util.GetCloverPath() + "/ebuild.sh -fr --x64-mcp --ext-co")

		// Modify credits to differentiate between "official" and custom builds
		log.Info("Updating package credits..")
		util.StringReplaceFile(util.GetCloverPath()+"/CloverPackage/CREDITS", "Chameleon team, crazybirdy, JrCs.", "Chameleon team, crazybirdy, JrCs. Custom package by Dids.")

		// Build the Clover installer package
		log.Info("Building Clover installer..")
		runCommand(util.GetCloverPath() + "/CloverPackage/makepkg")

		// Build the Clover ISO image
		log.Info("Building Clover ISO image..")
		runCommand(util.GetCloverPath() + "/CloverPackage/makeiso")

		// TODO: Would be nice to have a better formatting for the time string (eg. 1 minute and 20 seconds, instead of 1m20s)
		// Stop the execution timer
		executionElapsedTime := time.Since(executionStartTime)
		executionResult := fmt.Sprintf("Finished in %s!\n", executionElapsedTime)

		// Stop the spinner
		if !Verbose && !Quiet {
			Spinner.FinalMSG = executionResult
			Spinner.Stop()
		} else {
			log.Info(executionResult)
		}
	},
}

func init() {
	// Custom initialization logic
	cobra.OnInitialize(customInit)

	// Set the version field to add a "--version" flag automatically
	//rootCmd.Version = "0.0.1"
	var version = "0.0.1" // NOTE: This can be overridden when compiling with "go build"
	rootCmd.Version = version

	// Add persistent flags that carry over to all commands
	rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "enable verbose output")
	rootCmd.PersistentFlags().BoolVarP(&Quiet, "quiet", "q", false, "silence all output")
	rootCmd.PersistentFlags().StringVarP(&Revision, "revision", "r", "HEAD", "Clover target revision")
}

func customInit() {
	// Create a new log formatter
	formatter := new(prefixed.TextFormatter)

	// Change the log format based on the current verbosity
	if Verbose {
		// Enable showing a proper timestamp
		formatter.FullTimestamp = true
	} else {
		// TODO: Customize for non-verbose running, so remove the timestamp for instance?
		formatter.DisableTimestamp = true
	}

	// FIXME: The spinner isn't compatible with the current logging style, so we'll probably need some
	//        custom logwriter magic when running in non-verbose and non-quiet mode (eg. standard mode)
	/*if !Verbose && !Quiet {
		log.SetOutput(new(logWriter))
	}*/

	// Set specific colors for prefix and timestamp
	/*formatter.SetColorScheme(&prefixed.ColorScheme{
		PrefixStyle:    "blue+b",
		TimestampStyle: "white+h",
	})*/

	// TODO: Perhaps we just need a custom formatter to deal with the spinner integration?
	// Assign our logger to use the custom formatter
	if Verbose && !Quiet {
		log.Formatter = formatter
	} else if !Verbose && !Quiet {
		log.Formatter = new(ClobberLogFormatter)
	}

	// FIXME: Ideally log.Fatal should still work when this is set, but not sure if the log package supports that?
	// Disable logging if running in quiet mode
	if Quiet == true {
		log.SetOutput(ioutil.Discard)
	} else if Verbose == true {
		//log.SetLevel(log.DebugLevel)
		log.Level = logrus.DebugLevel
	} else {
		//log.SetLevel(log.InfoLevel)
		log.Level = logrus.InfoLevel
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
		log.Debug("Running command: '" + cmd + " " + argsString + "'")
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
		log.Debug("Command finished with output:\n" + string(cmdOut))
	}
}

// FIXME: d.Round doesn't exist in go versions <= 1.8
/*func fmtDuration(d time.Duration) string {
	d = d.Round(time.Minute)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	return fmt.Sprintf("%02d:%02d", h, m)
}*/
