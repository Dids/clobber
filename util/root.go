package util

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/rhysd/go-github-selfupdate/selfupdate"

	homedir "github.com/mitchellh/go-homedir"
)

// TODO: Should we create a "GetLogPath" function that returns a STATIC log filename/path,
//       with the current date/time appended to it, so it's the same file per each run?

// GetLogFilePath returns the full path to the current log file
func GetLogFilePath() string {
	return GetLogsPath() + "/clobber.log"
}

// GetLogsPath returns the full path to the logs directory
func GetLogsPath() string {
	return GetClobberPath() + "/logs"
}

// GetCloverPath returns the full path to Clover
func GetCloverPath() string {
	return GetEdkPath() + "/Clover"
}

// GetEdkPath returns the full path to EDK
func GetEdkPath() string {
	return GetSourcePath() + "/edk2"
}

// GetExtPath returns the full path to external packages
func GetExtPath() string {
	return GetSourcePath() + "/EXT_PACKAGES"
}

// GetSourcePath returns the full path to the source/work directory
func GetSourcePath() string {
	return GetClobberPath() + "/src"
}

// GetClobberPath returns the full path to the Clobber directory
func GetClobberPath() string {
	return GetHomePath() + "/.clobber"
}

// GetHomePath returns the full path to the user's home directory
func GetHomePath() string {
	// TODO: Comment the code
	home, err := homedir.Dir()
	if err != nil {
		log.Fatal("GetHomePath failed with error: ", err)
	}
	return home
}

// GetVersionDump returns a multi-line string containing the versions/commits
// for important dependencies and environments, like OS and LLVM versions
func GetVersionDump() string {
	// Start with an empty result string
	var result = ""

	// Get macOS version
	macosVersionOutput, macosVersionErr := exec.Command("sw_vers", "-productVersion").CombinedOutput()
	if macosVersionErr != nil {
		log.Fatal("Failed to get macOS version:", macosVersionErr, string(macosVersionOutput))
	}
	result += "macOS " + string(macosVersionOutput) + "\n"

	// Get Xcode version
	xcodeVersionOutput, xcodeVersionErr := exec.Command("xcodebuild", "-version").CombinedOutput()
	if xcodeVersionErr != nil {
		log.Fatal("Failed to get Xcode version:", xcodeVersionErr, string(xcodeVersionOutput))
	}
	xcodeVersionSplit := strings.Split(string(xcodeVersionOutput), "\n")
	xcodeVersion := xcodeVersionSplit[0]
	result += string(xcodeVersion) + "\n"

	// Get clang version
	clangVersionOutput, clangVersionErr := exec.Command("clang", "-v").CombinedOutput()
	if clangVersionErr != nil {
		log.Fatal("Failed to get clang version:", clangVersionErr, string(clangVersionOutput))
	}
	clangVersionSplit := strings.Split(string(clangVersionOutput), "\n")
	clangVersion := clangVersionSplit[0]
	result += string(clangVersion) + "\n"

	// Get EDK
	getEdkVersionCommand := exec.Command("git", "rev-parse", "HEAD")
	getEdkVersionCommand.Dir = GetEdkPath()
	edkVersionOutput, edkVersionErr := getEdkVersionCommand.CombinedOutput()
	if edkVersionErr != nil {
		log.Fatal("Failed to get edk2 version:", edkVersionErr, string(edkVersionOutput))
	}
	edkVersionSplit := strings.Split(string(edkVersionOutput), "\n")
	edkPackageVersion := edkVersionSplit[0]
	result += "EDK2 (" + string(edkPackageVersion) + ")\n"

	// Get Clover version
	getCloverVersionCommand := exec.Command("svn", "info", "--show-item", "revision")
	getCloverVersionCommand.Dir = GetCloverPath()
	cloverVersionOutput, cloverVersionErr := getCloverVersionCommand.CombinedOutput()
	if cloverVersionErr != nil {
		log.Fatal("Failed to get Clover version:", cloverVersionErr, string(cloverVersionOutput))
	}
	cloverVersion := strings.Replace(string(cloverVersionOutput), "\n", "", -1)
	result += "Clover (" + string(cloverVersion) + ")\n"

	// Check if the external packages directory exists
	if _, extPackagePathExistsErr := os.Stat(GetExtPath()); !os.IsNotExist(extPackagePathExistsErr) {
		if extPackagePathExistsErr != nil {
			log.Fatal("Failed to check if external packages directory exists:", extPackagePathExistsErr)
		} else {
			// Loop through each external package and get their versions/commits,
			// finally appending them to the result string
			extPackagePaths, listExtPackagesErr := ioutil.ReadDir(GetExtPath())
			if listExtPackagesErr != nil {
				log.Fatal("Failed to list external packages:", listExtPackagesErr)
			}
			for _, extPackage := range extPackagePaths {
				getExtPackageVersionCommand := exec.Command("git", "rev-parse", "HEAD")
				getExtPackageVersionCommand.Dir = GetExtPath() + "/" + extPackage.Name()
				extPackageVersionOutput, getVersionErr := getExtPackageVersionCommand.CombinedOutput()
				if getVersionErr != nil {
					log.Fatal("Failed to get version for external package:", getVersionErr, string(extPackageVersionOutput))
				}

				// Format the package version
				extPackageVersionSplit := strings.Split(string(extPackageVersionOutput), "\n")
				extPackageVersion := extPackageVersionSplit[0]

				// Append the package name and version to the result, ending with a newline
				result += extPackage.Name() + " (" + string(extPackageVersion) + ")\n"
			}
		}
	}

	// Remove empty lines from the result (if any)
	removeEmptyLinesRegex, removeEmptyLinesErr := regexp.Compile("\n\n")
	if removeEmptyLinesErr != nil {
		log.Fatal("Failed to remove empty lines:", removeEmptyLinesErr)
	}
	result = removeEmptyLinesRegex.ReplaceAllString(result, "\n")

	// Return the constructed result string
	return result
}

// StringReplaceFile allows you to replace a string in a file
func StringReplaceFile(path string, find string, replace string) error {
	// TODO: Comment the code
	fileContents, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	newFileContents := strings.Replace(string(fileContents), find, replace, -1)
	err = ioutil.WriteFile(path, []byte(newFileContents), 0)
	if err != nil {
		return err
	}
	return nil
}

// DownloadFile will download a url to a local file
func DownloadFile(url string, path string) error {
	// TODO: Comment the code
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}
	return nil
}

// GenerateTimeString generates a human readable time string (eg. "1 hour, 2 minutes and 12 seconds")
func GenerateTimeString(duration time.Duration) string {
	// Create an empty time string
	timeString := ""

	// Convenience variables
	inputSeconds := int(duration.Seconds())
	secondsInAMinute := 60
	secondsInAnHour := 60 * secondsInAMinute
	secondsInADay := 24 * secondsInAnHour

	// Parse the days
	days := int(inputSeconds / secondsInADay)
	if days > 0 {
		timeString = timeString + fmt.Sprintf("%v", days)

		// Suffix based on length
		if days > 1 {
			timeString = timeString + " days"
		} else {
			timeString = timeString + " day"
		}
	}

	// Parse the hours
	hourSeconds := inputSeconds % secondsInADay
	hours := int(hourSeconds / secondsInAnHour)
	if hours > 0 {
		// Add separator if necessary
		if len(timeString) > 0 {
			timeString = timeString + ", "
		}
		timeString = timeString + fmt.Sprintf("%v", hours)

		// Suffix based on length
		if hours > 1 {
			timeString = timeString + " hours"
		} else {
			timeString = timeString + " hour"
		}
	}

	// Parse the minutes
	minuteSeconds := hourSeconds % secondsInAnHour
	minutes := int(minuteSeconds / secondsInAMinute)
	if minutes > 0 {
		// Add separator if necessary
		if len(timeString) > 0 {
			timeString = timeString + ", "
		}
		timeString = timeString + fmt.Sprintf("%v", minutes)

		// Suffix based on length
		if minutes > 1 {
			timeString = timeString + " minutes"
		} else {
			timeString = timeString + " minute"
		}
	}

	// Parse the seconds
	seconds := int(minuteSeconds % secondsInAMinute)
	if seconds > 0 {
		// Add separator if necessary
		if len(timeString) > 0 {
			timeString = timeString + " and "
		}
		timeString = timeString + fmt.Sprintf("%v", seconds)

		// Suffix based on length
		if seconds > 1 {
			timeString = timeString + " seconds"
		} else {
			timeString = timeString + " second"
		}
	}

	return timeString
}

// FIXME: Using GitHub API to check for updates might not be plausible,
//        as we need a token, but we're using brew to compile, so we can't expose the token..

// CheckForUpdates checks GitHub for any version updates
func CheckForUpdates(version string) (bool, error) {
	// TODO: Comment the code
	semverVersion, err := semver.Make(version)
	if err != nil {
		log.Println("Invalid or missing semver version:", err)
		return false, err
	}
	//log.Println("Current version:", semverVersion)
	//selfupdate.EnableLog()
	latest, found, err := selfupdate.DetectLatest("Dids/clobber")
	if err != nil {
		log.Println("Error occurred while detecting version:", err)
		return false, err
	}
	if !found || latest == nil {
		//log.Println("No latest version found, assuming latest")
		return false, nil
	}
	log.Println("Latest version:", latest.Version)
	if !found || latest.Version.Equals(semverVersion) {
		//log.Println("Current version is the latest")
		return false, nil
	}
	//log.Println("New version is available", latest.Version)
	//log.Println("Release notes:\n", latest.ReleaseNotes)
	return true, nil
}
