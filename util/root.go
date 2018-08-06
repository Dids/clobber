package util

import (
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

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
	home, err := homedir.Dir()
	if err != nil {
		log.Fatal("GetHomePath failed with error: ", err)
	}
	return home
}

// StringReplaceFile allows you to replace a string in a file
func StringReplaceFile(path string, find string, replace string) {
	fileContents, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal("StringReplaceFile failed to read the file with error: ", err)
	}
	newFileContents := strings.Replace(string(fileContents), find, replace, -1)
	err = ioutil.WriteFile(path, []byte(newFileContents), 0)
	if err != nil {
		log.Fatal("StringReplaceFile failed to write the file with error: ", err)
	}
}

// DownloadFile will download a url to a local file
func DownloadFile(url string, path string) error {
	//fmt.Println("Downloading file from", url, "to", path)
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
