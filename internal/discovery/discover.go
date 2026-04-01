// Package discovery finds markdown files respecting gitignore and exclude patterns.
package discovery

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// DiscoverFiles returns a sorted list of .md file paths relative to root.
// It uses git ls-files when inside a git repo, falling back to filepath.Walk.
// Exclude patterns use path.Match-style globs.
func DiscoverFiles(root string, excludePatterns []string) ([]string, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("discovery: cannot resolve root: %w", err)
	}

	info, err := os.Stat(absRoot)
	if err != nil {
		return nil, fmt.Errorf("discovery: cannot stat root: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("discovery: root is not a directory")
	}

	var files []string
	if isGitRepo(absRoot) {
		files, err = gitLsFiles(absRoot)
		if err != nil {
			return nil, fmt.Errorf("discovery: git ls-files failed: %w", err)
		}
	} else {
		files, err = walkFiles(absRoot)
		if err != nil {
			return nil, fmt.Errorf("discovery: walk failed: %w", err)
		}
	}

	// Filter to .md files and apply exclude patterns
	var result []string
	for _, f := range files {
		if !strings.HasSuffix(f, ".md") {
			continue
		}
		if matchesExclude(f, excludePatterns) {
			continue
		}
		result = append(result, f)
	}

	sort.Strings(result)
	return result, nil
}

// isGitRepo checks if dir is inside a git repository by looking for .git.
func isGitRepo(dir string) bool {
	// Walk up to find .git directory
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return false
		}
		dir = parent
	}
}

// gitLsFiles runs git ls-files in dir and returns the list of tracked and untracked files.
func gitLsFiles(dir string) ([]string, error) {
	cmd := exec.Command("git", "ls-files", "--cached", "--others", "--exclude-standard")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-files: %w", err)
	}

	var files []string
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}

// walkFiles recursively walks dir and returns all file paths relative to dir.
func walkFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return fmt.Errorf("walk: cannot compute relative path: %w", err)
		}
		// Normalize to forward slashes for consistency
		rel = filepath.ToSlash(rel)
		files = append(files, rel)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("discovery: walk: %w", err)
	}
	return files, nil
}

// matchesExclude checks if path matches any of the exclude glob patterns.
func matchesExclude(path string, patterns []string) bool {
	for _, pattern := range patterns {
		matched, err := filepath.Match(pattern, path)
		if err != nil {
			continue
		}
		if matched {
			return true
		}
		// Also try matching just the base name for simple patterns
		base := filepath.Base(path)
		matched, err = filepath.Match(pattern, base)
		if err != nil {
			continue
		}
		if matched {
			return true
		}
	}
	return false
}
