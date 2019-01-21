package patches

import (
	"testing"

	"github.com/gobuffalo/packr/v2"
)

var packedPatches = packr.New("patches", "../patches")
var packedAssets = packr.New("assets", "../assets")

func TestPackedPatches(t *testing.T) {
	if _, err := packedPatches.FindString("buildpkg5.patch"); err != nil {
		t.Errorf("Failed to load packr asset: %s", err)
	}
}

func TestPackedAsssets(t *testing.T) {
	if _, err := packedAssets.Find("background.tiff"); err != nil {
		t.Errorf("Failed to load packr asset: %s", err)
	}
}
