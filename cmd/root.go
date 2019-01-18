package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Dids/clobber/patches"
	"github.com/Dids/clobber/util"
	figure "github.com/common-nighthawk/go-figure"
	"github.com/gobuffalo/packr/v2"
	"github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"

	"github.com/briandowns/spinner"
	git "github.com/gogits/git-module"
	"github.com/otiai10/copy"
	"github.com/spf13/cobra"
)

// Version is set in main.go and is overridable when building
var Version = "0.0.0"

// Verbose enables global verbose output
var Verbose bool

// Quiet enables silencing all output
var Quiet bool

// Revision is the default Clover revision to use
var Revision string

// BuildOnly will only build, but not update anything
var BuildOnly bool

// UpdateOnly will only update repositories, but not build anything
var UpdateOnly bool

// InstallerOnly will only build the installer
var InstallerOnly bool

// NoClean skips cleaning of dirty files
var NoClean bool

// Spinner is the CLI spinner/activity indicator
var Spinner = spinner.New(spinner.CharSets[14], 100*time.Millisecond)

// Create a new logger
var log = logrus.New()

// Setup static assets using Packr
var packedPatches = packr.New("patches", "../patches")
var packedAssets = packr.New("assets", "../assets")

// Execute is the entrypoint for the command-line application
func Execute() {
	// Update the version
	rootCmd.Version = Version

	if err := rootCmd.Execute(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1) // Note that only 'go run' prints 'exit status X'
	}
}

// ClobberLogFormatter is a custom log formatter
type ClobberLogFormatter struct {
}

// Format formats logs with the custom log format
func (f *ClobberLogFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	return []byte(entry.Message + "\n"), nil
}

// rootCmd is the Cobra command object
var rootCmd = &cobra.Command{
	Use:   "clobber",
	Short: "Clobber is a command-line application for building Clover",
	Long: `Clobber is a command-line application for building Clover.
				 Built by @Dids with tons of love, sweat and tears.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Measure execution time
		executionStartTime := time.Now()

		logo := figure.NewFigure("CLOBBER", "puffy", true)
		logo.Print()
		//fmt.Println()
		fmt.Println("                                  v" + Version + " by @Dids")
		fmt.Println()

		// Don't allow a mixture of --build-only, --update-only and --installer-only to be used simultaneously
		if BuildOnly && UpdateOnly {
			log.Fatal("Error: Cannot use --build-only and --update-only simultaneously")
		}
		if BuildOnly && InstallerOnly {
			log.Fatal("Error: Cannot use --build-only and --installer-only simultaneously")
		}
		if UpdateOnly && InstallerOnly {
			log.Fatal("Error: Cannot use --update-only and --installer-only simultaneously")
		}
		if BuildOnly && UpdateOnly && InstallerOnly {
			log.Fatal("Error: Cannot use --build-only, --update-only and --installer-only simultaneously")
		}

		// Start the spinner
		if !Verbose && !Quiet {
			Spinner.Start()
		}

		log.Debug("Target Clover revision:", Revision)
		if args != nil && len(args) > 0 {
			log.Debug("Building with arguments:", args)
		}

		// FIXME: Ditch the "git" package and just use our custom exec-based logic, so it's more consistent across the app?

		// TODO: Do we just blindly run everything here, or split stuff into their own packages and/or functions here?

		// Make sure that the correct directory structure exists
		//log.Info("Verifying folder structure..")
		Spinner.Prefix = formatSpinnerText("Verifying folder structure", false)
		time.Sleep(100 * time.Millisecond)
		mkdirErr := os.MkdirAll(util.GetSourcePath(), 0755)
		if mkdirErr != nil {
			log.Fatal("Error: MkdirAll failed with error: ", mkdirErr)
		}
		Spinner.Prefix = formatSpinnerText("Verifying folder structure", true)

		// Download UDK2018
		if _, err := os.Stat(util.GetUdkPath() + "/.git"); os.IsNotExist(err) {
			// If UDK2018 is missing and we're only supposed to build, we can't continue any further
			if BuildOnly {
				log.Fatal("Error: UDK2018 is missing and using --build-only, cannot continue")
			}
			if InstallerOnly {
				log.Fatal("Error: UDK2018 is missing and using --installer-only, cannot continue")
			}
			log.Debug("UDK2018 is missing, downloading..")
			Spinner.Prefix = formatSpinnerText("Downloading UDK2018", false)
			git.Clone("https://github.com/tianocore/edk2", util.GetUdkPath(), git.CloneRepoOptions{Branch: "UDK2018", Bare: false, Quiet: Verbose})
			Spinner.Prefix = formatSpinnerText("Downloading UDK2018", true)
		}

		// Update UDK2018
		if !BuildOnly && !InstallerOnly {
			log.Debug("Verifying UDK2018 is up to date..")
			Spinner.Prefix = formatSpinnerText("Verifying UDK2018 is up to date", false)
			// Disable cleaning up of extra files if the NoClean flag is set
			if !NoClean {
				runCommand("cd " + util.GetUdkPath() + " && git clean -fdx --exclude=\"Clover/\"")
			}
			git.Checkout(util.GetSourcePath(), git.CheckoutOptions{Branch: "UDK2018"})
			Spinner.Prefix = formatSpinnerText("Verifying UDK2018 is up to date", true)
		}

		// Download Clover
		if _, err := os.Stat(util.GetCloverPath() + "/.svn"); os.IsNotExist(err) {
			// If Clover is missing and we're only supposed to build, we can't continue any further
			if BuildOnly {
				log.Fatal("Error: Clover is missing and using --build-only, cannot continue")
			}
			if InstallerOnly {
				log.Fatal("Error: Clover is missing and using --installer-only, cannot continue")
			}
			log.Debug("Clover is missing, downloading..")
			Spinner.Prefix = formatSpinnerText("Downloading Clover", false)
			runCommand("svn co " + "https://svn.code.sf.net/p/cloverefiboot/code" + " " + util.GetCloverPath())
			Spinner.Prefix = formatSpinnerText("Downloading Clover", true)
		}

		// Update Clover
		if !BuildOnly && !InstallerOnly {
			log.Debug("Verifying Clover is up to date..")
			Spinner.Prefix = formatSpinnerText("Verifying Clover is up to date", false)
			// Disable cleaning up of extra files if the NoClean flag is set
			if !NoClean {
				runCommand("svn cleanup " + util.GetCloverPath())
				runCommand("svn cleanup --remove-unversioned " + util.GetCloverPath())
				runCommand("svn revert -R" + " " + util.GetCloverPath())
			}
			runCommand("svn up -r" + Revision + " " + util.GetCloverPath())
			Spinner.Prefix = formatSpinnerText("Verifying Clover is up to date", true)
		}

		if !UpdateOnly && !InstallerOnly {
			// Override HOME environment variable (use chroot-like logic for the build process)
			log.Debug("Overriding HOME..")
			os.Setenv("HOME", util.GetClobberPath())

			// Override TOOLCHAIR_DIR environment variable
			log.Debug("Overriding TOOLCHAIN_DIR..")
			os.Setenv("TOOLCHAIN_DIR", util.GetSourcePath()+"/opt/local")

			// Build base tools
			log.Debug("Building base tools..")
			Spinner.Prefix = formatSpinnerText("Building base tools", false)
			runCommand("make -C" + " " + util.GetUdkPath() + "/BaseTools/Source/C")
			Spinner.Prefix = formatSpinnerText("Building base tools", true)

			// Setup UDK
			log.Debug("Setting up UDK..")
			Spinner.Prefix = formatSpinnerText("Setting up UDK", false)
			runCommand("cd " + util.GetUdkPath() + " && " + "source edksetup.sh")
			Spinner.Prefix = formatSpinnerText("Setting up UDK", true)

			// Build gettext, mtoc and nasm (if necessary)
			if _, err := os.Stat(util.GetSourcePath() + "/opt/local/bin/gettext"); os.IsNotExist(err) {
				log.Debug("Building gettext..")
				Spinner.Prefix = formatSpinnerText("Building gettext", false)
				runCommand(util.GetCloverPath() + "/buildgettext.sh")
				Spinner.Prefix = formatSpinnerText("Building gettext", true)
			}
			if _, err := os.Stat(util.GetSourcePath() + "/opt/local/bin/mtoc.NEW"); os.IsNotExist(err) {
				log.Debug("Building mtoc..")
				Spinner.Prefix = formatSpinnerText("Building mtoc", false)
				runCommand(util.GetCloverPath() + "/buildmtoc.sh")
				Spinner.Prefix = formatSpinnerText("Building mtoc", true)
			}
			if _, err := os.Stat(util.GetSourcePath() + "/opt/local/bin/nasm"); os.IsNotExist(err) {
				log.Debug("Building nasm..")
				Spinner.Prefix = formatSpinnerText("Building nasm", false)
				runCommand(util.GetCloverPath() + "/buildnasm.sh")
				Spinner.Prefix = formatSpinnerText("Building nasm", true)
			}

			// Apply UDK patches
			log.Debug("Applying patches for UDK..")
			Spinner.Prefix = formatSpinnerText("Applying UDK patches", false)
			copyErr := copy.Copy(util.GetCloverPath()+"/Patches_for_UDK2018", util.GetUdkPath())
			Spinner.Prefix = formatSpinnerText("Applying UDK patches", true)
			if copyErr != nil {
				log.Fatal("Error: Failed to copy UDK patches: ", copyErr)
			}

			// NOTE: Disabled because ebuild.sh was officially "fixed" again..
			/* // Patch the Clover build script
			log.Debug("Patching Clover build script..")
			Spinner.Prefix = formatSpinnerText("Patching Clover build script", false)
			if err := patches.Patch(packedPatches, "ebuild", util.GetCloverPath()+"/ebuild.sh"); err != nil {
				log.Fatal("Error: Failed to patch Clover build script: ", err)
			}
			Spinner.Prefix = formatSpinnerText("Patching Clover build script", true) */

			// Build Clover (clean & build, with extras like ApfsDriverLoader checked out and compiled)
			log.Debug("Building Clover..")
			Spinner.Prefix = formatSpinnerText("Building Clover", false)
			runCommand(util.GetCloverPath() + "/ebuild.sh -cleanall") // TODO: Should this technically be ignored when using --no-clean?
			runCommand(util.GetCloverPath() + "/ebuild.sh -fr --x64 --ext-co -D NO_GRUB_DRIVERS_EMBEDDED")
			runCommand(util.GetCloverPath() + "/ebuild.sh -fr --x64-mcp --no-usb --ext-co -D NO_GRUB_DRIVERS_EMBEDDED")
			Spinner.Prefix = formatSpinnerText("Building Clover", true)
		}

		// Handle special cases when using BuildOnly/UpdateOnly
		if !BuildOnly && !InstallerOnly {
			// TODO: Add error handling for when HFSPlus.efi doesn't exist but running in BuildOnly mode?
			// Download and install extra EFI drivers
			log.Debug("Updating extra EFI drivers..")
			Spinner.Prefix = formatSpinnerText("Updating extra EFI drivers", false)
			util.DownloadFile("https://github.com/Micky1979/Build_Clover/raw/work/Files/HFSPlus_x64.efi", util.GetCloverPath()+"/CloverPackage/CloverV2/drivers-Off/drivers64UEFI/HFSPlus.efi")
			Spinner.Prefix = formatSpinnerText("Updating extra EFI drivers", true)
		}

		if !UpdateOnly {
			// Update the status, since this is a multi-step process anyway (and because our spinner freaks out otherwise)
			log.Debug("Patching Clover installer..")
			Spinner.Prefix = formatSpinnerText("Patching Clover installer", false)

			// Log important version information
			log.Debug("Listing environment version information:\n" + util.GetVersionDump())

			// Modify credits to differentiate between "official" and custom builds
			log.Debug("Updating package credits..")
			additionalCredits := "Custom package by Dids."
			creditsFilePath := util.GetCloverPath() + "/CloverPackage/CREDITS"
			fileBuffer, fileReadErr := ioutil.ReadFile(creditsFilePath)
			if fileReadErr != nil {
				log.Fatal("Error: Failed to update package credits: ", fileReadErr)
			}
			creditsString := string(fileBuffer)
			if !strings.Contains(creditsString, additionalCredits) {
				strReplaceErr := util.StringReplaceFile(creditsFilePath, "Chameleon team, crazybirdy, JrCs.", "Chameleon team, crazybirdy, JrCs. "+additionalCredits)
				if strReplaceErr != nil {
					log.Fatal("Error: Failed to update package credits: ", strReplaceErr)
				}
			}

			// Modify the installer package description to contain all important environment information
			log.Debug("Updating package description..")
			additionalDescription := "<p><b>Dids's build details:</b></p>\n"
			additionalDescription += "<ul>\n"
			versionDump := util.GetVersionDump()
			for _, line := range strings.Split(strings.TrimSuffix(versionDump, "\n"), "\n") {
				additionalDescription += "<li>" + line + "</li>\n"
			}
			additionalDescription += "</ul>\n"
			descriptionFilePath := util.GetCloverPath() + "/CloverPackage/package/Resources/templates/Description.html"
			descriptionFileBuffer, descriptionFileReadErr := ioutil.ReadFile(descriptionFilePath)
			if descriptionFileReadErr != nil {
				log.Fatal("Error: Failed to update package description: ", descriptionFileReadErr)
			}
			descriptionString := string(descriptionFileBuffer)
			if !strings.Contains(descriptionString, additionalDescription) && !strings.Contains(descriptionString, "Dids's build details:") {
				strReplaceErr := util.StringReplaceFile(descriptionFilePath, "</body>\n</html>\n", additionalDescription+"</body>\n</html>\n")
				if strReplaceErr != nil {
					log.Fatal("Error: Failed to update package description: ", strReplaceErr)
				}
			}

			// FIXME: When using "log.Fatal(...)" and running non-verbosely, the errors aren't shown to the user,
			//        but they are written to the log file, but we really need the user to know what's wrong too..

			// Patch the Clover installer package
			if err := patches.Patch(packedPatches, "buildpkg4", util.GetCloverPath()+"/CloverPackage/package/buildpkg.sh"); err != nil {
				log.Fatal("Error: Failed to patch Clover installer: ", err)
			}
			// Load the patch
			backgroundPatch, backgroundPatchErr := packedAssets.Find("background.tiff")
			if backgroundPatchErr != nil {
				log.Fatal("Error: Failed to patch Clover installer: ", backgroundPatchErr)
			}
			// Replace the Clover installer background image with our own
			if err := ioutil.WriteFile(util.GetCloverPath()+"/CloverPackage/package/Resources/background.tiff", backgroundPatch, 0644); err != nil {
				log.Fatal("Error: Failed to patch Clover installer: ", err)
			}
			Spinner.Prefix = formatSpinnerText("Patching Clover installer", true)

			// Build the Clover installer package
			log.Debug("Building Clover installer..")
			Spinner.Prefix = formatSpinnerText("Building Clover installer", false)
			runCommand(util.GetCloverPath() + "/CloverPackage/makepkg")
			Spinner.Prefix = formatSpinnerText("Building Clover installer", true)

			if !InstallerOnly {
				// Build the Clover ISO image
				log.Debug("Building Clover ISO image..")
				Spinner.Prefix = formatSpinnerText("Building Clover ISO image", false)
				runCommand(util.GetCloverPath() + "/CloverPackage/makeiso")
				Spinner.Prefix = formatSpinnerText("Building Clover ISO image", true)
			}
		}

		// Stop the execution timer
		executionElapsedTime := util.GenerateTimeString(time.Since(executionStartTime))
		executionResult := fmt.Sprintf("\n🎉  Finished in %s 🎉\n", executionElapsedTime)

		// Stop the spinner
		if !Verbose && !Quiet {
			log.Debug(executionResult)
			Spinner.FinalMSG = executionResult
			Spinner.Stop()
		} else {
			log.Info(executionResult)
		}
		fmt.Println()
	},
}

func init() {
	// Custom initialization logic
	cobra.OnInitialize(customInit)

	// Add persistent flags that carry over to all commands
	rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "enable verbose output")
	rootCmd.PersistentFlags().BoolVarP(&Quiet, "quiet", "q", false, "silence all output")
	rootCmd.PersistentFlags().StringVarP(&Revision, "revision", "r", "HEAD", "Clover target revision")
	rootCmd.PersistentFlags().BoolVarP(&BuildOnly, "build-only", "b", false, "only build (no update)")
	rootCmd.PersistentFlags().BoolVarP(&UpdateOnly, "update-only", "u", false, "only update (no build)")
	rootCmd.PersistentFlags().BoolVarP(&InstallerOnly, "installer-only", "i", false, "only build the installer")
	rootCmd.PersistentFlags().BoolVarP(&NoClean, "no-clean", "n", false, "skip cleaning of dirty files")
}

func customInit() {
	// Create a new log formatter
	formatter := new(prefixed.TextFormatter)

	// Enable showing a proper timestamp
	formatter.FullTimestamp = true

	/*// Change the log format based on the current verbosity
	if Verbose {
		// Enable showing a proper timestamp
		formatter.FullTimestamp = true
	} else {
		// Hide timestamp when running in non-verbose mode
		formatter.DisableTimestamp = true
	}*/

	// Assign our logger to use the custom formatter
	log.Formatter = formatter
	/*if Verbose && !Quiet {
		log.Formatter = formatter
	} else if !Verbose && !Quiet {
		log.Formatter = new(ClobberLogFormatter)
	}*/

	// Ensure the log file folder exists
	mkdirErr := os.MkdirAll(util.GetLogsPath(), 0755)
	if mkdirErr != nil {
		log.Fatal("Error: MkdirAll failed with error: ", mkdirErr)
	}

	// Setup logging with lumberjack support (log to stdout + log file)
	lumberjackLogger := &lumberjack.Logger{
		Filename:   util.GetLogFilePath(),
		MaxSize:    15,    // Log file size in megabytes
		MaxBackups: 5,     // Maximum amount of files to keep
		MaxAge:     90,    // Days to keep files
		Compress:   false, // Compress log files (disabled by default)
	}
	lumberjackLogger.Rotate()
	logMultiWriter := io.MultiWriter(os.Stdout, lumberjackLogger)
	if Quiet || !Verbose {
		// Disable logging to console if running in quiet mode
		logMultiWriter = io.MultiWriter(ioutil.Discard, lumberjackLogger)
	}
	log.SetOutput(logMultiWriter)
	//log.WriterLevel(logrus.DebugLevel)

	// Set log level
	log.Level = logrus.DebugLevel
	/*if Verbose == true {
		log.Level = logrus.DebugLevel
	} else {
		log.Level = logrus.InfoLevel
	}*/
}

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

	//if Verbose {
	log.Debug("Running command: '" + cmd + " " + argsString + "'")
	//}

	var (
		cmdOut []byte
		err    error
	)
	if cmdOut, err = exec.Command(cmd, args...).CombinedOutput(); err != nil {
		log.Fatal("Error: Failed to run '" + cmd + " " + argsString + "':\n" + string(cmdOut))
	}
	//if Verbose {
	log.Debug("Command finished with output:\n" + string(cmdOut))
	//}
}

func formatSpinnerText(text string, done bool) string {
	if done {
		fmt.Printf("\r✔ %s  \n", text)
		return fmt.Sprintf("\r✔ %s  \n", text)
	}
	return fmt.Sprintf("\r◌ %s ", text)
}
