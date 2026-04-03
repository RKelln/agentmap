package main

import (
	"bytes"
	"strings"
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

// TestDetectManagedInstall_Homebrew verifies that a Homebrew-managed binary
// path returns an error pointing to "brew upgrade".
func TestDetectManagedInstall_Homebrew(t *testing.T) {
	paths := []string{
		"/opt/homebrew/Cellar/agentmap/0.1.0/bin/agentmap",
		"/usr/local/Cellar/agentmap/0.1.0/bin/agentmap",
		"/home/linuxbrew/.linuxbrew/Cellar/agentmap/0.1.0/bin/agentmap",
	}
	for _, p := range paths {
		t.Run(p, func(t *testing.T) {
			err := detectManagedInstall(p)
			if err == nil {
				t.Fatal("expected error for Homebrew path; got nil")
			}
			if !strings.Contains(err.Error(), "brew upgrade agentmap") {
				t.Errorf("error should mention 'brew upgrade agentmap', got: %v", err)
			}
		})
	}
}

// TestDetectManagedInstall_Scoop verifies that a Scoop-managed binary path
// returns an error pointing to "scoop update".
func TestDetectManagedInstall_Scoop(t *testing.T) {
	paths := []string{
		`C:\Users\user\scoop\apps\agentmap\current\agentmap.exe`,
		"/home/user/scoop/apps/agentmap/current/agentmap",
	}
	for _, p := range paths {
		t.Run(p, func(t *testing.T) {
			err := detectManagedInstall(p)
			if err == nil {
				t.Fatal("expected error for Scoop path; got nil")
			}
			if !strings.Contains(err.Error(), "scoop update agentmap") {
				t.Errorf("error should mention 'scoop update agentmap', got: %v", err)
			}
		})
	}
}

// TestDetectManagedInstall_Direct verifies that direct-install paths return nil.
func TestDetectManagedInstall_Direct(t *testing.T) {
	paths := []string{
		"/usr/local/bin/agentmap",
		"/home/user/.local/bin/agentmap",
		`C:\Program Files\agentmap\agentmap.exe`,
	}
	for _, p := range paths {
		t.Run(p, func(t *testing.T) {
			if err := detectManagedInstall(p); err != nil {
				t.Errorf("expected nil for direct install path %q, got: %v", p, err)
			}
		})
	}
}
