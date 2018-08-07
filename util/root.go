package util

import (
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

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

// CheckForUpdates checks GitHub for any version updates
func CheckForUpdates(version string) (bool, error) {
	// TODO: Comment the code
	semverVersion, err := semver.Make(version)
	if err != nil {
		log.Println("Invalid or missing semver version:", err)
		return false, err
	}
	log.Println("Current version:", semverVersion)
	selfupdate.EnableLog()
	latest, found, err := selfupdate.DetectLatest("Dids/clobber")
	if err != nil {
		log.Println("Error occurred while detecting version:", err)
		return false, err
	}
	if !found || latest == nil {
		log.Println("No latest version found, assuming latest")
		return false, nil
	}
	log.Println("Latest version:", latest.Version)
	if !found || latest.Version.Equals(semverVersion) {
		log.Println("Current version is the latest")
		return false, nil
	}
	log.Println("New version is available", latest.Version)
	log.Println("Release notes:\n", latest.ReleaseNotes)
	return true, nil
}
