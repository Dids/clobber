package cmd

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Dids/clobber/patches"
	"github.com/Dids/clobber/snake"
	"github.com/Dids/clobber/util"
	figure "github.com/common-nighthawk/go-figure"
	"github.com/gobuffalo/packr/v2"
	"github.com/mholt/archiver"
	logrus "github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"

	"github.com/briandowns/spinner"
	git "github.com/gogits/git-module"
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

// Hiss hiss, said the snake
var Hiss bool

// Controls whether to patch buildpkg.sh or not
var patchBuildPkg = true

// Controls whether to patch ebuild.sh or not
// var patchEbuild = true

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

// ErrorWriterHook handles logging errors in quiet/non-verbose mode
type ErrorWriterHook struct {
	Writer    io.Writer
	LogLevels []logrus.Level
}

// Fire formats the error log and writes it to the output
func (hook *ErrorWriterHook) Fire(entry *logrus.Entry) error {
	_, err := hook.Writer.Write([]byte("\n\n" + entry.Message + "\n\nSee the log file for more details:\n" + util.GetLogFilePath() + "\n"))
	return err
}

// Levels define on which log levels this hook would trigger
func (hook *ErrorWriterHook) Levels() []logrus.Level {
	return hook.LogLevels
}

// rootCmd is the Cobra command object
var rootCmd = &cobra.Command{
	Use:   "clobber",
	Short: "Clobber is a command-line application for building Clover",
	Long: `Clobber is a command-line application for building Clover.
				 Built by @Dids with tons of love, sweat and tears.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Setup graceful shutdown support
		c := make(chan os.Signal)
		signal.Notify(c, os.Interrupt, syscall.SIGINT)
		go func() {
			<-c
			log.Fatal("CTRL-C detected, aborting..")
		}()

		// FIXME: Integrate this with the building process
		if Hiss {
			game := snake.NewGame()
			go game.Start()
		}

		// Measure execution time
		executionStartTime := time.Now()

		// Print banner if not playing a game
		if !Hiss {
			logo := figure.NewFigure("CLOBBER", "puffy", true)
			logo.Print()
			fmt.Println("                                  v" + Version + " by @Dids")
			fmt.Println()
		}

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

		// Make sure that the correct directory structure exists
		//log.Info("Verifying folder structure..")
		Spinner.Prefix = formatSpinnerText("Verifying folder structure", false)
		time.Sleep(100 * time.Millisecond)
		mkdirErr := os.MkdirAll(util.GetSourcePath(), 0755)
		if mkdirErr != nil {
			log.Fatal("Error: MkdirAll failed with error: ", mkdirErr)
		}
		Spinner.Prefix = formatSpinnerText("Verifying folder structure", true)

		// Remove any old edk2 installations
		os.RemoveAll(util.GetSourcePath() + "/edk2")

		// Download Clover
		if _, err := os.Stat(util.GetCloverPath() + "/.git"); os.IsNotExist(err) {
			// If Clover is missing and we're only supposed to build, we can't continue any further
			if BuildOnly {
				log.Fatal("Error: Clover is missing and using --build-only, cannot continue")
			}
			if InstallerOnly {
				log.Fatal("Error: Clover is missing and using --installer-only, cannot continue")
			}
			log.Debug("Clover is missing, downloading..")
			Spinner.Prefix = formatSpinnerText("Downloading Clover", false)
			if err := git.Clone("https://github.com/CloverHackyColor/CloverBootloader", util.GetCloverPath(), git.CloneRepoOptions{Branch: "master", Bare: false, Quiet: Verbose}); err != nil {
				log.Fatal("Error: Failure detected, aborting\n", err)
			}
			Spinner.Prefix = formatSpinnerText("Downloading Clover", true)
		}

		// Update Clover
		if !BuildOnly && !InstallerOnly {
			log.Debug("Verifying Clover is up to date..")
			Spinner.Prefix = formatSpinnerText("Verifying Clover is up to date", false)
			// Disable cleaning up of extra files if the NoClean flag is set
			if !NoClean {
				if err := runCommand("git reset --hard", util.GetCloverPath()); err != nil {
					log.Fatal("Error: Failure detected, aborting\n", err)
				}
				if err := runCommand("git clean -fdx", util.GetCloverPath()); err != nil {
					log.Fatal("Error: Failure detected, aborting\n", err)
				}
			}
			if err := git.Checkout(util.GetCloverPath(), git.CheckoutOptions{Branch: Revision}); err != nil {
				log.Fatal("Error: Failure detected, aborting\n", err)
			}
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
			if err := runCommand("make -C BaseTools/Source/C", util.GetCloverPath()); err != nil {
				if err := runCommand("make clean -C BaseTools/Source/C", util.GetCloverPath()); err != nil {
					log.Fatal("Error: Failure detected, aborting\n", err)
				}
				if err := runCommand("make -C BaseTools/Source/C", util.GetCloverPath()); err != nil {
					log.Fatal("Error: Failure detected, aborting\n", err)
				}
			}
			Spinner.Prefix = formatSpinnerText("Building base tools", true)

			// Setup EDK
			log.Debug("Setting up EDK..")
			Spinner.Prefix = formatSpinnerText("Setting up EDK", false)
			if err := runCommand("source ./edksetup.sh BaseTools", util.GetCloverPath()); err != nil {
				log.Fatal("Error: Failure detected, aborting\n", err)
			}
			Spinner.Prefix = formatSpinnerText("Setting up EDK", true)

			// Build gettext, mtoc and nasm (if necessary)
			if _, err := os.Stat(util.GetSourcePath() + "/opt/local/bin/gettext"); os.IsNotExist(err) {
				log.Debug("Linking gettext..")
				Spinner.Prefix = formatSpinnerText("Linking gettext", false)
				if err := runCommand("brew link gettext --force --overwrite", ""); err != nil {
					log.Fatal("Error: Failure detected, aborting\n", err)
				}
				defer runCommand("brew unlink gettext", "")
				if err := runCommand("mkdir -p "+util.GetSourcePath()+"/opt/local/bin", ""); err != nil {
					log.Fatal("Error: Failure detected, aborting\n", err)
				}
				if err := runCommand("ln -sf /usr/local/bin/gettext "+util.GetSourcePath()+"/opt/local/bin/gettext", ""); err != nil {
					log.Fatal("Error: Failure detected, aborting\n", err)
				}
				Spinner.Prefix = formatSpinnerText("Linking gettext", true)
			}
			if _, err := os.Stat(util.GetSourcePath() + "/opt/local/bin/mtoc.NEW"); os.IsNotExist(err) {
				log.Debug("Building mtoc..")
				Spinner.Prefix = formatSpinnerText("Building mtoc", false)
				if err := runCommand(util.GetCloverPath()+"/buildmtoc.sh", ""); err != nil {
					log.Fatal("Error: Failure detected, aborting\n", err)
				}
				Spinner.Prefix = formatSpinnerText("Building mtoc", true)
			}
			if _, err := os.Stat(util.GetSourcePath() + "/opt/local/bin/nasm"); os.IsNotExist(err) {
				log.Debug("Linking nasm..")
				Spinner.Prefix = formatSpinnerText("Linking nasm", false)

				if err := runCommand("brew link nasm --force --overwrite", ""); err != nil {
					log.Fatal("Error: Failure detected, aborting\n", err)
				}
				defer runCommand("brew unlink nasm", "")
				if err := runCommand("mkdir -p "+util.GetSourcePath()+"/opt/local/bin", ""); err != nil {
					log.Fatal("Error: Failure detected, aborting\n", err)
				}
				if err := runCommand("ln -sf /usr/local/bin/nasm "+util.GetSourcePath()+"/opt/local/bin/nasm", ""); err != nil {
					log.Fatal("Error: Failure detected, aborting\n", err)
				}
				Spinner.Prefix = formatSpinnerText("Linking nasm", true)
			}

			// Patch Clover.dsc (eg. skip building ApfsDriverLoader)
			log.Debug("Patching Clover..")
			Spinner.Prefix = formatSpinnerText("Patching Clover build script", false)
			if err := runCommand("sed -i '' -e 's/^[^#]*ApfsDriverLoader/#&/' Clover.dsc", util.GetCloverPath()); err != nil {
				log.Fatal("Error: Failure detected, aborting\n", err)
			}
			if err := runCommand("sed -i '' -e 's/^[^#]*AptioMemoryFix/#&/' Clover.dsc", util.GetCloverPath()); err != nil {
				log.Fatal("Error: Failure detected, aborting\n", err)
			}
			if err := runCommand("sed -i '' -e 's/^[^#]*AptioInputFix/#&/' Clover.dsc", util.GetCloverPath()); err != nil {
				log.Fatal("Error: Failure detected, aborting\n", err)
			}
			// Patch vers.txt (current version is statically embedded for some dumb reason)
			if err := runCommand("git describe --tags | tr -d '\n' > vers.txt", util.GetCloverPath()); err != nil {
				log.Fatal("Error: Failure detected, aborting\n", err)
			}
			// Patch old vers.txt logic back in to ebuild.sh
			// if patchEbuild {
			// 	if err := patches.Patch(packedPatches, "ebuild", util.GetCloverPath()+"/ebuild.sh"); err != nil {
			// 		log.Fatal("Error: Failure detected, aborting\n", err)
			// 	}
			// }
			Spinner.Prefix = formatSpinnerText("Patching", true)

			// Build Clover (clean & build, with extras like ApfsDriverLoader checked out and compiled)
			log.Debug("Building Clover..")
			Spinner.Prefix = formatSpinnerText("Building Clover", false)
			// TODO: Shouldn't this technically be ignored when using --no-clean?
			if err := runCommand("source edksetup.sh BaseTools; ./ebuild.sh -cleanall || true", util.GetCloverPath()); err != nil {
				log.Fatal("Error: Failure detected, aborting\n", err)
			}
			// 64-bit (boot6, default)
			if err := runCommand("source edksetup.sh BaseTools; ./ebuild.sh -fr -D NO_GRUB_DRIVERS_EMBEDDED", util.GetCloverPath()); err != nil {
				log.Fatal("Error: Failure detected, aborting\n", err)
			}
			// 64-bit (boot7, MCP/BiosBlockIO)
			if err := runCommand("source edksetup.sh BaseTools; ./ebuild.sh -fr --x64-mcp --no-usb -D NO_GRUB_DRIVERS_EMBEDDED", util.GetCloverPath()); err != nil {
				log.Fatal("Error: Failure detected, aborting\n", err)
			}
			Spinner.Prefix = formatSpinnerText("Building Clover", true)
		}

		// Handle special cases when using BuildOnly/UpdateOnly
		if !BuildOnly && !InstallerOnly {
			// Download and install extra EFI drivers
			log.Debug("Updating extra EFI drivers..")
			Spinner.Prefix = formatSpinnerText("Updating extra EFI drivers", false)

			// Make sure the driver paths exist (especially important when update only and on a clean install)
			os.MkdirAll(util.GetCloverPath()+"/CloverPackage/CloverV2/EFI/CLOVER/drivers/UEFI", 0700)
			os.MkdirAll(util.GetCloverPath()+"/CloverPackage/CloverV2/EFI/CLOVER/drivers/BIOS", 0700)
			os.MkdirAll(util.GetCloverPath()+"/CloverPackage/CloverV2/EFI/CLOVER/drivers/off/UEFI/FileSystem", 0700)
			os.MkdirAll(util.GetCloverPath()+"/CloverPackage/CloverV2/EFI/CLOVER/drivers/off/BIOS/FileSystem", 0700)

			// Download and copy HFSPlus.efi
			if err := util.DownloadFile("https://github.com/Micky1979/Build_Clover/raw/work/Files/HFSPlus_x64.efi", util.GetCloverPath()+"/CloverPackage/CloverV2/EFI/CLOVER/drivers/UEFI/HFSPlus.efi"); err != nil {
				log.Fatal("Error: Failed to update extra EFI drivers (download HFSPlus): ", err)
			}
			if err := util.DownloadFile("https://github.com/Micky1979/Build_Clover/raw/work/Files/HFSPlus_x64.efi", util.GetCloverPath()+"/CloverPackage/CloverV2/EFI/CLOVER/drivers/BIOS/HFSPlus.efi"); err != nil {
				log.Fatal("Error: Failed to update extra EFI drivers (download HFSPlus): ", err)
			}

			// Download Acidanthera drivers
			os.RemoveAll(os.TempDir() + "AppleSupportPkg.zip")
			if err := util.DownloadFile(getGitHubReleaseLink("https://api.github.com/repos/acidanthera/AppleSupportPkg/releases/latest", "browser_download_url.*RELEASE.zip"), os.TempDir()+"AppleSupportPkg.zip"); err != nil {
				log.Fatal("Error: Failed to update extra EFI drivers (download AppleSupportPkg): ", err)
			}
			defer os.RemoveAll(os.TempDir() + "AppleSupportPkg.zip")

			os.RemoveAll(os.TempDir() + "AptioFixPkg.zip")
			if err := util.DownloadFile(getGitHubReleaseLink("https://api.github.com/repos/acidanthera/AptioFixPkg/releases/latest", "browser_download_url.*RELEASE.zip"), os.TempDir()+"AptioFixPkg.zip"); err != nil {
				log.Fatal("Error: Failed to update extra EFI drivers (download AptioFixPkg): ", err)
			}
			defer os.RemoveAll(os.TempDir() + "AptioFixPkg.zip")
			if err := util.DownloadFile(getGitHubReleaseLink("https://api.github.com/repos/ReddestDream/OcQuirks/releases/latest", "browser_download_url.*.zip"), os.TempDir()+"OcQuirks.zip"); err != nil {
				log.Fatal("Error: Failed to update extra EFI drivers (download OcQuirks): ", err)
			}
			defer os.RemoveAll(os.TempDir() + "OcQuirks.zip")

			// Extract Acidanthera drivers
			os.RemoveAll(os.TempDir() + "AppleSupportPkg")
			if err := archiver.Unarchive(os.TempDir()+"AppleSupportPkg.zip", os.TempDir()+"AppleSupportPkg"); err != nil {
				log.Fatal("Error: Failed to update extra EFI drivers (unzip AppleSupportPkg): ", err)
			}
			defer os.RemoveAll(os.TempDir() + "AppleSupportPkg")
			os.RemoveAll(os.TempDir() + "AptioFixPkg")
			if err := archiver.Unarchive(os.TempDir()+"AptioFixPkg.zip", os.TempDir()+"AptioFixPkg"); err != nil {
				log.Fatal("Error: Failed to update extra EFI drivers (unzip AptioFixPkg): ", err)
			}
			defer os.RemoveAll(os.TempDir() + "AptioFixPkg")
			if err := archiver.Unarchive(os.TempDir()+"OcQuirks.zip", os.TempDir()+"OcQuirks"); err != nil {
				log.Fatal("Error: Failed to update extra EFI drivers (unzip OcQuirks): ", err)
			}
			defer os.RemoveAll(os.TempDir() + "OcQuirks")

			// Copy ApfsDriverLoader.efi
			if err := util.CopyFile(os.TempDir()+"AppleSupportPkg/Drivers/ApfsDriverLoader.efi", util.GetCloverPath()+"/CloverPackage/CloverV2/EFI/CLOVER/drivers/UEFI/ApfsDriverLoader.efi"); err != nil {
				log.Fatal("Error: Failed to update extra EFI drivers (copy ApfsDriverLoader.efi): ", err)
			}
			if err := util.CopyFile(os.TempDir()+"AppleSupportPkg/Drivers/ApfsDriverLoader.efi", util.GetCloverPath()+"/CloverPackage/CloverV2/EFI/CLOVER/drivers/BIOS/ApfsDriverLoader.efi"); err != nil {
				log.Fatal("Error: Failed to update extra EFI drivers (copy ApfsDriverLoader.efi): ", err)
			}

			// Copy UsbKbDxe.efi
			if err := util.CopyFile(os.TempDir()+"AppleSupportPkg/Drivers/UsbKbDxe.efi", util.GetCloverPath()+"/CloverPackage/CloverV2/EFI/CLOVER/drivers/off/UEFI/HID/UsbKbDxe.efi"); err != nil {
				log.Fatal("Error: Failed to update extra EFI drivers (copy UsbKbDxe.efi): ", err)
			}

			// Copy VBoxHfs.efi
			if err := util.CopyFile(os.TempDir()+"AppleSupportPkg/Drivers/VBoxHfs.efi", util.GetCloverPath()+"/CloverPackage/CloverV2/EFI/CLOVER/drivers/off/UEFI/FileSystem/VBoxHfs.efi"); err != nil {
				log.Fatal("Error: Failed to update extra EFI drivers (copy VBoxHfs.efi): ", err)
			}
			if err := util.CopyFile(os.TempDir()+"AppleSupportPkg/Drivers/VBoxHfs.efi", util.GetCloverPath()+"/CloverPackage/CloverV2/EFI/CLOVER/drivers/off/BIOS/FileSystem/VBoxHfs.efi"); err != nil {
				log.Fatal("Error: Failed to update extra EFI drivers (copy VBoxHfs.efi): ", err)
			}

			// Copy AptioInputFix.efi
			if err := util.CopyFile(os.TempDir()+"AptioFixPkg/Drivers/AptioInputFix.efi", util.GetCloverPath()+"/CloverPackage/CloverV2/EFI/CLOVER/drivers/off/UEFI/HID/AptioInputFix.efi"); err != nil {
				log.Fatal("Error: Failed to update extra EFI drivers (copy AptioInputFix.efi): ", err)
			}

			// Copy AptioMemoryFix.efi
			if err := util.CopyFile(os.TempDir()+"AptioFixPkg/Drivers/AptioMemoryFix.efi", util.GetCloverPath()+"/CloverPackage/CloverV2/EFI/CLOVER/drivers/UEFI/AptioMemoryFix.efi"); err != nil {
				log.Fatal("Error: Failed to update extra EFI drivers (copy AptioMemoryFix.efi): ", err)
			}

			// Copy FwRuntimeServices.efi
			if err := util.CopyFile(os.TempDir()+"OcQuirks/OcQuirks/FwRuntimeServices.efi", util.GetCloverPath()+"/CloverPackage/CloverV2/EFI/CLOVER/drivers/UEFI/FwRuntimeServices.efi"); err != nil {
				log.Fatal("Error: Failed to update extra EFI drivers (copy FwRuntimeServices.efi): ", err)
			}

			// Copy OcQuirks.efi
			if err := util.CopyFile(os.TempDir()+"OcQuirks/OcQuirks/OcQuirks.efi", util.GetCloverPath()+"/CloverPackage/CloverV2/EFI/CLOVER/drivers/UEFI/OcQuirks.efi"); err != nil {
				log.Fatal("Error: Failed to update extra EFI drivers (copy OcQuirks.efi): ", err)
			}

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

			// Patch the Clover installer package
			if patchBuildPkg {
				if patchErr := patches.Patch(packedPatches, "buildpkg6", util.GetCloverPath()+"/CloverPackage/package/buildpkg.sh"); patchErr != nil {
					log.Fatal("Error: Failed to patch Clover installer (patch buildpkg.sh): ", patchErr)
				}
			}
			// Load the installer image asset
			backgroundPatch, backgroundPatchErr := packedAssets.Find("background.tiff")
			if backgroundPatchErr != nil {
				log.Fatal("Error: Failed to patch Clover installer (load background.tiff): ", backgroundPatchErr)
			}
			// Replace the Clover installer background image with our own
			if writeErr := ioutil.WriteFile(util.GetCloverPath()+"/CloverPackage/package/Resources/background.tiff", backgroundPatch, 0644); writeErr != nil {
				log.Fatal("Error: Failed to patch Clover installer (replace background.tiff): ", writeErr)
			}
			// Load the compressed Metal theme
			metalTheme, metalThemeErr := packedAssets.Find("metal_theme.tar.gz")
			if metalThemeErr != nil {
				log.Fatal("Error: Failed to patch Clover installer (load metal_theme.tar.gz): ", metalThemeErr)
			}
			// Copy the compressed Metal theme to a temporary directory
			tempDir, tempDirErr := ioutil.TempDir("", "")
			if tempDirErr != nil {
				log.Fatal("Error: Failed to patch Clover installer (get temp dir): ", tempDirErr)
			}
			defer os.Remove(tempDir)
			metalThemeTemp, metalThemeTempErr := ioutil.TempFile(tempDir, "clobber_metal_theme.*.tar.gz")
			if metalThemeTempErr != nil {
				log.Fatal("Error: Failed to patch Clover installer (create metal_theme.tar.gz): ", metalThemeTempErr)
			}
			defer os.Remove(metalThemeTemp.Name())
			if writeErr := ioutil.WriteFile(metalThemeTemp.Name(), metalTheme, 0644); writeErr != nil {
				log.Fatal("Error: Failed to patch Clover installer (write metal_theme.tar.gz): ", writeErr)
			}
			// Extract and install the Metal theme to the Clover installer
			if unarchiveErr := archiver.Unarchive(metalThemeTemp.Name(), util.GetCloverPath()+"/CloverPackage/CloverV2/themespkg"); unarchiveErr != nil {
				log.Fatal("Error: Failed to patch Clover installer (extract metal_theme.tar.gz): ", unarchiveErr)
			}
			Spinner.Prefix = formatSpinnerText("Patching Clover installer", true)

			// Build the Clover installer package
			log.Debug("Building Clover installer..")
			Spinner.Prefix = formatSpinnerText("Building Clover installer", false)
			if err := runCommand("./CloverPackage/makepkg", util.GetCloverPath()); err != nil {
				log.Fatal("Error: Failure detected, aborting\n", err)
			}
			Spinner.Prefix = formatSpinnerText("Building Clover installer", true)

			if !InstallerOnly {
				// Build the Clover ISO image
				log.Debug("Building Clover ISO image..")
				Spinner.Prefix = formatSpinnerText("Building Clover ISO image", false)
				// if err := runCommand("./CloverPackage/makeiso", util.GetCloverPath()); err != nil {
				if err := runCommand("make iso", util.GetCloverPath()+"/CloverPackage"); err != nil {
					log.Fatal("Error: Failure detected, aborting\n", err)
				}
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
	rootCmd.PersistentFlags().StringVarP(&Revision, "revision", "r", "master", "Clover target revision")
	rootCmd.PersistentFlags().BoolVarP(&BuildOnly, "build-only", "b", false, "only build (no update)")
	rootCmd.PersistentFlags().BoolVarP(&UpdateOnly, "update-only", "u", false, "only update (no build)")
	rootCmd.PersistentFlags().BoolVarP(&InstallerOnly, "installer-only", "i", false, "only build the installer")
	rootCmd.PersistentFlags().BoolVarP(&NoClean, "no-clean", "n", false, "skip cleaning of dirty files")
	rootCmd.PersistentFlags().BoolVarP(&Hiss, "hiss", "", false, "that's Sir Hiss to you")
}

func customInit() {
	// Create a new log formatter
	formatter := new(prefixed.TextFormatter)

	// Enable showing a proper timestamp
	formatter.FullTimestamp = true

	// Assign our logger to use the custom formatter
	log.Formatter = formatter

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
		// Disable logging to console if running in quiet/non-verbose mode
		logMultiWriter = io.MultiWriter(ioutil.Discard, lumberjackLogger)
	}
	log.SetOutput(logMultiWriter)

	// Set default log level
	log.Level = logrus.DebugLevel

	// Setup our custom error writer hook, which prints errors in quiet/non-verbose mode
	if Quiet || !Verbose {
		log.AddHook(&ErrorWriterHook{
			Writer: os.Stderr,
			LogLevels: []logrus.Level{
				logrus.PanicLevel,
				logrus.FatalLevel,
				logrus.ErrorLevel,
			},
		})
	}
}

func runCommand(command string, dir string) error {
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

	log.Debug("Running command: '" + cmd + " " + argsString + "'")

	// runCmd := exec.Command(cmd, args...)
	runCmd := exec.Command("bash", "-c", cmd+" "+argsString)
	if len(dir) > 0 {
		runCmd.Dir = dir
	}

	var (
		cmdOut []byte
		err    error
	)
	if cmdOut, err = runCmd.CombinedOutput(); err != nil {
		// log.Fatal("Error: Failed to run '" + cmd + " " + argsString + "':\n" + string(cmdOut))
		customErr := errors.New("Failed to run '" + cmd + " " + argsString + "':\n" + string(cmdOut))
		log.Warn("Warning: " + customErr.Error())
		return customErr
	}
	log.Debug("Command finished with output:\n" + string(cmdOut))
	return nil
}

func getGitHubReleaseLink(url string, filter string) string {
	cmd := "curl"
	if len(os.Getenv("GITHUB_API_TOKEN")) > 0 {
		cmd += " -H \"Authorization: token " + os.Getenv("GITHUB_API_TOKEN") + "\""
	}
	cmd += " -s " + url + " | grep \"" + filter + "\" | cut -d : -f 2,3 | tr -d \\\""
	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return fmt.Sprintf("Failed to execute command: %s", cmd)
	}
	return strings.TrimSpace(string(out))
}

func formatSpinnerText(text string, done bool) string {
	if done {
		fmt.Printf("\r✔ %s  \n", text)
		return fmt.Sprintf("\r✔ %s  \n", text)
	}
	return fmt.Sprintf("\r◌ %s ", text)
}
