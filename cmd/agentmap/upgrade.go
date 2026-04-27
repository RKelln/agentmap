package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
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
		// DetectLatest may fail to find a release if GitHub's releases list
		// index hasn't propagated yet (eventual consistency). Fall back to the
		// /releases/latest endpoint which uses a separate, reliable index.
		if info, err := fetchLatestFromAPI(ctx); err == nil {
			latestVer, perr := semver.NewVersion(info.TagName)
			if perr != nil {
				return fmt.Errorf("no releases found for your platform")
			}
			curVer, cerr := semver.NewVersion(version)
			if cerr != nil {
				return fmt.Errorf("no releases found for your platform")
			}
			if latestVer.GreaterThan(curVer) {
				return fmt.Errorf(
					"update %s is available but GitHub's release index hasn't fully propagated yet; "+
						"try again in a moment\nRelease notes: %s",
					info.TagName, info.HTMLURL)
			}
			fmt.Printf("Already up to date (%s)\n", version)
			return nil
		}
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

// latestReleaseInfo is a minimal subset of the /releases/latest JSON response.
type latestReleaseInfo struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

// fetchLatestFromAPI calls the GitHub API /releases/latest endpoint directly.
// This endpoint uses a separate index from the paginated /releases list and is
// not subject to the same eventual-consistency delays.
func fetchLatestFromAPI(ctx context.Context) (*latestReleaseInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://api.github.com/repos/RKelln/agentmap/releases/latest", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github API returned %d", resp.StatusCode)
	}

	var info latestReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	return &info, nil
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
