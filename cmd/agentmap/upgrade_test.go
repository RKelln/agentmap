package main

import (
	"bytes"
	"testing"
)

const devVersion = "dev"

// TestUpgradeRejectsDevBuild verifies that upgrade refuses to run on dev builds.
func TestUpgradeRejectsDevBuild(t *testing.T) {
	// Save and restore version.
	orig := version
	t.Cleanup(func() { version = orig })

	version = devVersion

	cmd := upgradeCmd
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error for dev build; got nil")
	}
	if err.Error() != "cannot upgrade a dev build; install a release version first" {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestUpgradeCheckFlagExists verifies --check flag is registered.
func TestUpgradeCheckFlagExists(t *testing.T) {
	f := upgradeCmd.Flags().Lookup("check")
	if f == nil {
		t.Error("--check flag not registered on upgradeCmd")
	}
}
