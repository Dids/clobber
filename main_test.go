package main

import (
	"testing"

	"github.com/blang/semver"
)

func TestVersion(t *testing.T) {
	_, err := semver.Make(Version)
	if err != nil {
		t.Errorf("Version failed to validate with error: %s", err)
	}
}
