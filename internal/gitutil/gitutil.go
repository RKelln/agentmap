// Package gitutil provides git diff integration for content change detection.
package gitutil

import (
	"fmt"
	"os"
	"os/exec"
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

func isGitRepo(dir string) bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = dir
	return cmd.Run() == nil
}

var hunkHeaderRE = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)(?:,(\d+))? @@`)

func getChangedRanges(path string) ([]LineRange, error) {
	cmd := exec.Command("git", "diff", "HEAD", "--", path)
	cmd.Dir = getGitRoot(path)
	output, err := cmd.Output()
	if err != nil {
		if isUntracked(path) {
			return nil, nil
		}
		if strings.Contains(string(output), "fatal: bad object") ||
			strings.Contains(string(output), "fatal: ambiguous argument") {
			return nil, nil
		}
		return nil, err
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
