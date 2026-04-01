// Package check validates that nav blocks are in sync with document headings.
package check

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ryankelln/agentmap/internal/config"
	"github.com/ryankelln/agentmap/internal/discovery"
	"github.com/ryankelln/agentmap/internal/navblock"
	"github.com/ryankelln/agentmap/internal/parser"
)

// Check discovers markdown files under root and validates each.
// Returns an error if any file fails validation.
func Check(root string, cfg config.Config) error {
	files, err := discovery.DiscoverFiles(root, cfg.Exclude)
	if err != nil {
		return fmt.Errorf("check: discover files: %w", err)
	}

	var failedFiles []string

	for _, f := range files {
		fullPath := filepath.Join(root, f)
		failed, report, err := CheckFile(fullPath, cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s: %v\n", f, err)
			continue
		}
		if failed {
			failedFiles = append(failedFiles, report)
		}
	}

	if len(failedFiles) > 0 {
		for _, report := range failedFiles {
			fmt.Println(report)
		}
		if len(failedFiles) == 1 {
			fmt.Println("1 file failed validation.")
		} else {
			fmt.Printf("%d files failed validation.\n", len(failedFiles))
		}
		if len(failedFiles) == 1 {
			return fmt.Errorf("1 file failed validation")
		}
		return fmt.Errorf("%d files failed validation", len(failedFiles))
	}

	return nil
}

// CheckFile validates a single markdown file's nav block against its headings.
// Returns (failed, report, error) where failed indicates validation errors.
//
//nolint:revive // keep exported name for CLI/API parity with design spec
func CheckFile(path string, cfg config.Config) (bool, string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return false, "", fmt.Errorf("read file: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	totalLines := len(lines)

	oldBlock, _, _, hasBlock := navblock.ParseNavBlock(string(content))
	if !hasBlock {
		// No nav block to validate - that's valid
		return false, "", nil
	}

	// Purpose-only blocks are valid (no nav entries to check)
	if len(oldBlock.Nav) == 0 {
		return false, "", nil
	}

	headings := parser.ParseHeadings(string(content), cfg.MaxDepth)
	sections := parser.ComputeSections(headings, totalLines)

	// Build queues of section indices by heading text so duplicates match in order.
	sectionQueues := make(map[string][]int)
	matched := make([]bool, len(sections))
	for i, s := range sections {
		key := stripHash(s.Text)
		sectionQueues[key] = append(sectionQueues[key], i)
	}

	var failures []string

	// Check: headings in nav should exist in document with matching line numbers
	for _, e := range oldBlock.Nav {
		key := stripHash(e.Name)
		queue := sectionQueues[key]
		if len(queue) == 0 {
			failures = append(failures, fmt.Sprintf("  %s: in nav but not in document", e.Name))
			continue
		}

		idx := queue[0]
		sectionQueues[key] = queue[1:]
		matched[idx] = true
		section := sections[idx]

		prefix := strings.Repeat("#", section.Depth)
		expectedName := prefix + section.Text

		// Check line number mismatch
		if e.Start != section.Start || e.N != section.End-section.Start+1 {
			failures = append(failures, fmt.Sprintf("  %s: nav says %d-%d, actual %d-%d",
				expectedName, e.Start, e.Start+e.N-1, section.Start, section.End))
		}
	}

	// Check: headings in document should exist in nav
	for i, s := range sections {
		if matched[i] {
			continue
		}
		prefix := strings.Repeat("#", s.Depth)
		expectedName := prefix + s.Text
		failures = append(failures, fmt.Sprintf("  %s: in document but not in nav", expectedName))
	}

	if len(failures) > 0 {
		report := fmt.Sprintf("FAIL: %s\n%s", path, strings.Join(failures, "\n"))
		return true, report, nil
	}

	return false, "", nil
}

func stripHash(name string) string {
	name = strings.TrimSpace(name)
	for strings.HasPrefix(name, "#") {
		name = strings.TrimPrefix(name, "#")
	}
	return strings.TrimSpace(name)
}
