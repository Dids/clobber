package util

import (
	"io/ioutil"
	"log"
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
