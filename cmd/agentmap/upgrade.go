package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	selfupdate "github.com/creativeprojects/go-selfupdate"
	"github.com/spf13/cobra"

	"github.com/RKelln/agentmap/internal/initcmd"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade agentmap to the latest version",
	Long: `Check GitHub Releases for the latest version of agentmap and update the
binary in place. Refuses to operate on dev builds (version == "dev").

If agentmap was installed via Homebrew or Scoop, use your package manager
instead — agentmap upgrade does not support managed installs.

Use --check to only report whether an update is available.`,
	Args: cobra.NoArgs,
	RunE: runUpgrade,
}

func runUpgrade(cmd *cobra.Command, _ []string) error {
	if version == "dev" {
		return fmt.Errorf("cannot upgrade a dev build; install a release version first")
	}

	// Refuse to self-update binaries managed by Homebrew or Scoop — doing so
	// would corrupt their tracking state. Direct the user to their package manager.
	exe, exeErr := selfupdate.ExecutablePath()
	if exeErr == nil {
		if err := detectManagedInstall(exe); err != nil {
			return err
		}
	}

	checkOnly, _ := cmd.Flags().GetBool("check")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	updater, err := selfupdate.NewUpdater(selfupdate.Config{
		Validator:  &selfupdate.ChecksumValidator{UniqueFilename: "checksums.txt"},
		Prerelease: shouldAllowPrerelease(version),
	})
	if err != nil {
		return fmt.Errorf("creating updater: %w", err)
	}

	latest, found, err := updater.DetectLatest(ctx, selfupdate.ParseSlug("RKelln/agentmap"))
	if err != nil {
		return fmt.Errorf("checking for updates: %w", err)
	}
	if !found {
		return fmt.Errorf("no releases found for your platform")
	}

	if latest.LessOrEqual(version) {
		fmt.Printf("Already up to date (%s)\n", version)
		return nil
	}

	if checkOnly {
		fmt.Printf("Update available: %s -> %s\n", version, latest.Version())
		if latest.URL != "" {
			fmt.Printf("Release notes: %s\n", latest.URL)
		}
		return nil
	}

	if exeErr != nil {
		return fmt.Errorf("locating executable: %w", exeErr)
	}

	if latest.URL != "" {
		fmt.Printf("Release notes: %s\n", latest.URL)
	}
	fmt.Printf("Downloading agentmap %s...\n", latest.Version())
	if err := updater.UpdateTo(ctx, latest, exe); err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied updating %s; try: sudo agentmap upgrade", exe)
		}
		return fmt.Errorf("updating: %w", err)
	}

	fmt.Printf("Updated agentmap %s -> %s\n", version, latest.Version())
	return nil
}

// shouldAllowPrerelease returns true when currentVersion is a prerelease tag,
// so upgrade checks can see newer prerelease builds as candidates.
func shouldAllowPrerelease(currentVersion string) bool {
	return strings.Contains(currentVersion, "-")
}

// detectManagedInstall returns an error with upgrade instructions if the binary
// at exePath was installed by a package manager that manages its own upgrades.
// Returns nil for direct installs and go-install installs.
func detectManagedInstall(exePath string) error {
	method := initcmd.DetectInstallMethod(exePath, os.Getenv("GOPATH"), os.Getenv("GOBIN"))
	switch method {
	case initcmd.InstallMethodHomebrew:
		return fmt.Errorf("agentmap was installed via Homebrew; upgrade with:\n\n  brew upgrade agentmap")
	case initcmd.InstallMethodScoop:
		return fmt.Errorf("agentmap was installed via Scoop; upgrade with:\n\n  scoop update agentmap")
	}
	return nil
}
