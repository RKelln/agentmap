// Package check validates that nav blocks are in sync with document headings.
package check

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ryankelln/agentmap/internal/config"
	"github.com/ryankelln/agentmap/internal/discovery"
	"github.com/ryankelln/agentmap/internal/generate"
	"github.com/ryankelln/agentmap/internal/navblock"
	"github.com/ryankelln/agentmap/internal/parser"
)

// Check discovers markdown files under root and validates each.
// Returns an error if any file fails validation.
func Check(root string, cfg config.Config, warnUnreviewed bool) error {
	files, err := discovery.DiscoverFiles(root, cfg.Exclude)
	if err != nil {
		return fmt.Errorf("check: discover files: %w", err)
	}

	var failedFiles []string
	var filesWithWarnings int

	for _, f := range files {
		fullPath := filepath.Join(root, f)
		failed, report, warnings, err := CheckFile(fullPath, cfg, warnUnreviewed)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s: %v\n", f, err)
			continue
		}
		if failed {
			failedFiles = append(failedFiles, report)
		}
		if len(warnings) > 0 {
			fmt.Printf("WARN: %s has unreviewed descriptions\n", f)
			for _, w := range warnings {
				fmt.Println(w)
			}
			fmt.Println()
			filesWithWarnings++
		}
	}

	if filesWithWarnings > 0 {
		if filesWithWarnings == 1 {
			fmt.Println("1 file with unreviewed descriptions.")
		} else {
			fmt.Printf("%d files with unreviewed descriptions.\n", filesWithWarnings)
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
// Returns (failed, report, warnings, error) where failed indicates validation errors
// and warnings lists unreviewed descriptions when warnUnreviewed is true.
//
//nolint:revive // keep exported name for CLI/API parity with design spec
func CheckFile(path string, cfg config.Config, warnUnreviewed bool) (bool, string, []string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return false, "", nil, fmt.Errorf("read file: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	totalLines := len(lines)

	pr := navblock.ParseNavBlock(string(content))
	oldBlock, hasBlock, corrupted := pr.Block, pr.Found, pr.Corrupted
	if corrupted {
		fmt.Fprintf(os.Stderr, "warning: %s: nav block is corrupted — run 'agentmap generate' to regenerate\n", path)
		return false, "", nil, nil
	}
	if !hasBlock {
		// No nav block to validate - that's valid
		return false, "", nil, nil
	}

	// Purpose-only blocks are valid (no nav entries to check)
	if len(oldBlock.Nav) == 0 {
		return false, "", nil, nil
	}

	headings := parser.ParseHeadings(string(content), cfg.MaxDepth)
	sections := parser.ComputeSections(headings, totalLines)

	// §W1: Apply the same large-file cap as generate: build lightweight NavEntry slice
	// with WordCount, pass through FilterNavEntries for consistent filtering.
	navEntries := make([]navblock.NavEntry, len(sections))
	for i, s := range sections {
		prefix := strings.Repeat("#", s.Depth)
		navEntries[i] = navblock.NavEntry{
			Start:     s.Start,
			N:         s.End - s.Start + 1,
			Name:      prefix + navblock.NormalizeHeading(s.Text),
			WordCount: navblock.SectionWordCount(lines, s.Start, s.End-s.Start+1),
		}
	}
	filteredEntries := generate.FilterNavEntries(navEntries, cfg.MaxNavEntries, cfg.NavStubWords)

	// Rebuild navSections from filtered entries (match by start line).
	navSections := make([]parser.Section, 0, len(filteredEntries))
	sectionByStart := make(map[int]parser.Section, len(sections))
	for _, s := range sections {
		sectionByStart[s.Start] = s
	}
	for _, e := range filteredEntries {
		if s, ok := sectionByStart[e.Start]; ok {
			navSections = append(navSections, s)
		}
	}

	// Build queues of section indices by heading text so duplicates match in order.
	// §C1: strip commas when building keys to match generate's comma-stripping behavior.
	sectionQueues := make(map[string][]int)
	matched := make([]bool, len(navSections))
	for i, s := range navSections {
		key := navblock.NormalizeHeading(s.Text)
		sectionQueues[key] = append(sectionQueues[key], i)
	}

	var failures []string

	// Check: headings in nav should exist in document with matching line numbers
	for _, e := range oldBlock.Nav {
		key := navblock.NormalizeHeading(e.Name)
		queue := sectionQueues[key]
		if len(queue) == 0 {
			failures = append(failures, fmt.Sprintf("  %s: in nav but not in document", e.Name))
			continue
		}

		idx := queue[0]
		sectionQueues[key] = queue[1:]
		matched[idx] = true
		section := navSections[idx]

		prefix := strings.Repeat("#", section.Depth)
		expectedName := prefix + section.Text

		// Check line number mismatch
		if e.Start != section.Start || e.N != section.End-section.Start+1 {
			failures = append(failures, fmt.Sprintf("  %s: nav says %d-%d, actual %d-%d",
				expectedName, e.Start, e.Start+e.N-1, section.Start, section.End))
		}
	}

	// Check: headings in document should exist in nav
	for i, s := range navSections {
		if matched[i] {
			continue
		}
		prefix := strings.Repeat("#", s.Depth)
		expectedName := prefix + s.Text
		failures = append(failures, fmt.Sprintf("  %s: in document but not in nav", expectedName))
	}

	if len(failures) > 0 {
		report := fmt.Sprintf("FAIL: %s\n%s", path, strings.Join(failures, "\n"))
		return true, report, nil, nil
	}

	var warnings []string
	if warnUnreviewed {
		if navblock.IsAutoGenerated(oldBlock.Purpose) {
			warnings = append(warnings, fmt.Sprintf("  purpose: %s", oldBlock.Purpose))
		}
		for _, e := range oldBlock.Nav {
			if navblock.IsAutoGenerated(e.About) {
				warnings = append(warnings, fmt.Sprintf("  %s: %s", e.Name, e.About))
			}
		}
	}

	return false, "", warnings, nil
}
