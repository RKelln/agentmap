package main

import (
	"context"
	"fmt"
	"os"
	"time"

	selfupdate "github.com/creativeprojects/go-selfupdate"
	"github.com/spf13/cobra"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade agentmap to the latest version",
	Long: `Check GitHub Releases for the latest version of agentmap and update the
binary in place. Refuses to operate on dev builds (version == "dev").

Use --check to only report whether an update is available.`,
	Args: cobra.NoArgs,
	RunE: runUpgrade,
}

func runUpgrade(cmd *cobra.Command, _ []string) error {
	if version == "dev" {
		return fmt.Errorf("cannot upgrade a dev build; install a release version first")
	}

	checkOnly, _ := cmd.Flags().GetBool("check")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	updater, err := selfupdate.NewUpdater(selfupdate.Config{
		Validator: &selfupdate.ChecksumValidator{UniqueFilename: "checksums.txt"},
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
		return nil
	}

	exe, err := selfupdate.ExecutablePath()
	if err != nil {
		return fmt.Errorf("locating executable: %w", err)
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
