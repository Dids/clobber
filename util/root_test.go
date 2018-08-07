package util

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestGetCloverPath(t *testing.T) {
	path := GetCloverPath()
	if !isValidPath(path) {
		t.Errorf("Invalid path: %s", path)
	}
}

func TestGetUdkPath(t *testing.T) {
	path := GetUdkPath()
	if !isValidPath(path) {
		t.Errorf("Invalid path: %s", path)
	}
}

func TestGetSourcePath(t *testing.T) {
	path := GetSourcePath()
	if !isValidPath(path) {
		t.Errorf("Invalid path: %s", path)
	}
}

func TestGetClobberPath(t *testing.T) {
	path := GetClobberPath()
	if !isValidPath(path) {
		t.Errorf("Invalid path: %s", path)
	}
}

func TestGetHomePath(t *testing.T) {
	path := GetHomePath()
	if !isValidPath(path) {
		t.Errorf("Invalid path: %s", path)
	}
}

func TestStringReplaceFile(t *testing.T) {
	file, err := ioutil.TempFile("", "clobber-test")
	if err != nil {
		t.Errorf("Failed to create temporary file: %s", err)
	}

	_, err = file.WriteString("Clobber Test\n")
	if err != nil {
		file.Close()
		os.Remove(file.Name())
		t.Errorf("Failed to write to file: %s", err)
	}

	err = StringReplaceFile(file.Name(), "Clobber", "Clubber")
	if err != nil {
		file.Close()
		os.Remove(file.Name())
		t.Errorf("Failed to string replace file: %s", err)
	}

	data, err := ioutil.ReadFile(file.Name())
	if err != nil {
		file.Close()
		os.Remove(file.Name())
		t.Errorf("Failed to read from file: %s", err)
	}
	if data == nil {
		file.Close()
		os.Remove(file.Name())
		t.Errorf("Failed to read from file: data was null")
	}
	dataString := string(data)
	if dataString != "Clubber Test\n" {
		file.Close()
		os.Remove(file.Name())
		t.Errorf("Failed to read from file: data string did not match: %s", dataString)
	}

	file.Close()
	os.Remove(file.Name())
}

func TestDownloadFile(t *testing.T) {
	url := "https://www.w3.org/TR/PNG/iso_8859-1.txt"
	file, err := ioutil.TempFile("", "clobber-test")
	if err != nil {
		t.Errorf("Failed to create temporary file: %s", err)
	}

	path := file.Name()
	file.Close()

	err = DownloadFile(url, path)
	if err != nil {
		os.Remove(file.Name())
		t.Errorf("Failed to download file: %s", err)
	}

	err = DownloadFile("", path)
	if err == nil {
		os.Remove(file.Name())
		t.Errorf("Failed to test download file when missing url")
	}

	err = DownloadFile(url, "")
	if err == nil {
		os.Remove(file.Name())
		t.Errorf("Failed to test download file when missing path")
	}

	err = DownloadFile("", "")
	if err == nil {
		os.Remove(file.Name())
		t.Errorf("Failed to test download file when missing url and path")
	}

	os.Remove(file.Name())
}

func TestCheckForUpdates(t *testing.T) {
	_, err := CheckForUpdates("0.0.1")
	if err != nil {
		t.Errorf("Failed to check for updates: %s", err)
	}

	_, err = CheckForUpdates("")
	if err == nil {
		t.Errorf("Failed to handle invalid version strings")
	}
}

func isValidPath(filePath string) bool {
	return len(filePath) > 0
	// TODO: Figure out how to do this when no paths exist yet
	/*if _, err := os.Stat(filePath); err == nil {
		return true
	}

	var d []byte
	if err := ioutil.WriteFile(filePath, d, 0644); err == nil {
		os.Remove(filePath)
		return true
	}

	return false*/
}
