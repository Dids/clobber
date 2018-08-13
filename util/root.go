package util

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/rhysd/go-github-selfupdate/selfupdate"

	homedir "github.com/mitchellh/go-homedir"
)

// GetCloverPath returns the full path to Clover
func GetCloverPath() string {
	return GetUdkPath() + "/Clover"
}

// GetUdkPath returns the full path to EDK/UDK
func GetUdkPath() string {
	return GetSourcePath() + "/edk2"
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
