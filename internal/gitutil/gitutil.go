// Package gitutil provides git diff integration for content change detection.
package gitutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// LineRange represents a range of lines (1-indexed, inclusive).
type LineRange struct {
	Start int // 1-indexed start line (inclusive)
	End   int // 1-indexed end line (inclusive)
}

// FileChanges returns the line ranges that have changed in the given file.
// Returns nil, nil if the file is untracked, not in a git repo, or has no changes.
func FileChanges(path string) ([]LineRange, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("stat file: %w", err)
	}

	if !isGitRepo(path) {
		return nil, nil
	}

	ranges, err := getChangedRanges(path)
	if err != nil {
		return nil, fmt.Errorf("get changed ranges: %w", err)
	}

	return ranges, nil
}

// RepoChanges runs a single git diff HEAD -U0 in dir and returns a map of relative
// file path → changed line ranges for all modified files in the repository.
// Returns nil, nil if not in a git repo.
func RepoChanges(dir string) (map[string][]LineRange, error) {
	if !isGitRepo(dir) {
		return nil, nil
	}

	cmd := exec.Command("git", "diff", "HEAD", "-U0")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		// If HEAD doesn't exist yet (empty repo) or other issue, return empty map
		if strings.Contains(string(output), "fatal: bad object") ||
			strings.Contains(string(output), "fatal: ambiguous argument") {
			return map[string][]LineRange{}, nil
		}
		return nil, fmt.Errorf("git diff: %w", err)
	}

	return parseRepoDiff(string(output)), nil
}

// parseRepoDiff parses the output of `git diff -U0 HEAD` into a map of
// relative file path → line ranges.
func parseRepoDiff(output string) map[string][]LineRange {
	result := make(map[string][]LineRange)
	currentFile := ""

	for _, line := range strings.Split(output, "\n") {
		// File header: +++ b/<path>
		if strings.HasPrefix(line, "+++ b/") {
			currentFile = strings.TrimPrefix(line, "+++ b/")
			continue
		}
		// Skip dev/null (file deletions)
		if strings.HasPrefix(line, "+++ ") {
			currentFile = ""
			continue
		}

		if currentFile == "" {
			continue
		}

		// Hunk header: @@ -old +new,count @@
		matches := hunkHeaderRE.FindStringSubmatch(line)
		if len(matches) < 2 {
			continue
		}

		start, _ := strconv.Atoi(matches[1])
		var count int
		if len(matches) >= 3 && matches[2] != "" {
			count, _ = strconv.Atoi(matches[2])
		} else {
			count = 1
		}

		if count > 0 {
			result[currentFile] = append(result[currentFile], LineRange{
				Start: start,
				End:   start + count - 1,
			})
		}
	}

	return result
}

func isGitRepo(dir string) bool {
	// If dir is a file, check its parent directory
	if info, err := os.Stat(dir); err == nil && info.Mode().IsRegular() {
		dir = filepath.Dir(dir)
	}
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = dir
	return cmd.Run() == nil
}

var hunkHeaderRE = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)(?:,(\d+))? @@`)

func getChangedRanges(path string) ([]LineRange, error) {
	cmd := exec.Command("git", "diff", "HEAD", "--", path)
	gitRoot := getGitRoot(path)
	cmd.Dir = gitRoot
	output, err := cmd.Output()
	if err != nil {
		if isUntracked(path) {
			return nil, nil
		}
		if strings.Contains(string(output), "fatal: bad object") ||
			strings.Contains(string(output), "fatal: ambiguous argument") {
			return nil, nil
		}
		return nil, fmt.Errorf("git diff: %w", err)
	}

	return parseDiffOutput(string(output)), nil
}

func isUntracked(path string) bool {
	cmd := exec.Command("git", "ls-files", "--others", "--", path)
	cmd.Dir = getGitRoot(path)
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) != ""
}

func getGitRoot(path string) string {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = path
	output, err := cmd.Output()
	if err != nil {
		return "."
	}
	return strings.TrimSpace(string(output))
}

func parseDiffOutput(output string) []LineRange {
	var ranges []LineRange

	for _, line := range strings.Split(output, "\n") {
		matches := hunkHeaderRE.FindStringSubmatch(line)
		if len(matches) < 2 {
			continue
		}

		start, _ := strconv.Atoi(matches[1])
		var count int
		if len(matches) >= 3 && matches[2] != "" {
			count, _ = strconv.Atoi(matches[2])
		} else {
			count = 1
		}

		if count > 0 {
			ranges = append(ranges, LineRange{
				Start: start,
				End:   start + count - 1,
			})
		}
	}

	return ranges
}
