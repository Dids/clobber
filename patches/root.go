package patches

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/gobuffalo/packr"
)

// Patch function for patching files
func Patch(packedPatches packr.Box, patchName string, fileToPatch string) error {
	// Parse the necessary patch information
	tempFilePath := "/tmp/" + patchName + ".patch"

	// Create a temporary file for the patch
	file, fileErr := os.Create(tempFilePath)
	if fileErr != nil {
		return fileErr
	}

	// Write the patch contents to the temporary file
	// if _, writeErr := file.WriteString(packedPatches.String("buildpkg.patch")); writeErr != nil {
	if _, writeErr := file.WriteString(packedPatches.String(patchName + ".patch")); writeErr != nil {
		os.Remove(tempFilePath)
		return writeErr
	}

	// Synchronize the file contents and close the file
	file.Sync()
	file.Close()

	// Run the patch command
	// patchCommandString := "if ! /usr/bin/patch -Rsf --dry-run " + util.GetCloverPath() + "/CloverPackage/package/buildpkg.sh /tmp/buildpkg.patch 2>/dev/null ; then /usr/bin/patch --backup " + util.GetCloverPath() + "/CloverPackage/package/buildpkg.sh /tmp/buildpkg.patch; fi"
	patchCommandString := fmt.Sprintf("if ! /usr/bin/patch -Rsf --dry-run %s %s 2>/dev/null ; then /usr/bin/patch --backup %s %s; fi", fileToPatch, tempFilePath, fileToPatch, tempFilePath)
	if _, patchCommandErr := exec.Command("/usr/bin/env", "bash", "-c", patchCommandString).CombinedOutput(); patchCommandErr != nil {
		os.Remove(tempFilePath)
		return patchCommandErr
	}

	// Remove the temporary file
	if deleteErr := os.Remove(tempFilePath); deleteErr != nil {
		return deleteErr
	}

	// Return a null if successful
	return nil
}
