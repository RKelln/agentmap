// Package generate implements the agentmap generate command.
package generate

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

const (
	maxPurposeLen    = 80
	frontmatterDelim = "---"
	navBlockEnd      = "-->"
)

// Generate discovers markdown files under root and generates nav blocks for each.
func Generate(root string, cfg config.Config, dryRun bool) error {
	files, err := discovery.DiscoverFiles(root, cfg.Exclude)
	if err != nil {
		return fmt.Errorf("generate: discover files: %w", err)
	}

	for _, f := range files {
		fullPath := filepath.Join(root, f)
		report, err := File(fullPath, cfg, dryRun)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s: %v\n", f, err)
			continue
		}
		fmt.Println(report)
	}

	return nil
}

// File processes a single markdown file and writes a nav block.
func File(path string, cfg config.Config, dryRun bool) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("generate: read file: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	totalLines := len(lines)

	headings := parser.ParseHeadings(string(content), cfg.MaxDepth)
	sections := parser.ComputeSections(headings, totalLines)

	var blockText string
	var report string

	if totalLines < cfg.MinLines || len(headings) == 0 {
		purpose := extractPurpose(string(content))
		blockText = navblock.RenderPurposeOnly(purpose)
		report = fmt.Sprintf("Skipped: %s (purpose-only)", path)
	} else {
		purpose := extractPurpose(string(content))
		entries := buildNavEntries(sections)
		block := navblock.NavBlock{
			Purpose: purpose,
			Nav:     entries,
		}
		_ = block // used for future keyword extraction
		blockText = navblock.RenderNavBlock(block)
		report = fmt.Sprintf("Generated: %s (%d sections)", path, len(entries))
	}

	if dryRun {
		return report, nil
	}

	newContent := insertNavBlock(string(content), blockText)
	if err := os.WriteFile(path, []byte(newContent), 0o644); err != nil {
		return "", fmt.Errorf("generate: write file: %w", err)
	}

	return report, nil
}

// extractPurpose extracts the first non-heading, non-frontmatter paragraph as purpose.
func extractPurpose(content string) string {
	lines := strings.Split(content, "\n")

	inFrontmatter := false
	inNavBlock := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Handle YAML frontmatter
		if i == 0 && trimmed == frontmatterDelim {
			inFrontmatter = true
			continue
		}
		if inFrontmatter {
			if trimmed == frontmatterDelim {
				inFrontmatter = false
			}
			continue
		}

		// Track existing nav block
		if strings.Contains(trimmed, "<!-- AGENT:NAV") {
			inNavBlock = true
			continue
		}
		if inNavBlock {
			if trimmed == navBlockEnd {
				inNavBlock = false
			}
			continue
		}

		// Skip empty lines before first content
		if trimmed == "" {
			continue
		}

		// Skip heading lines
		if len(trimmed) > 0 && trimmed[0] == '#' {
			continue
		}

		// Skip HTML comment lines
		if strings.HasPrefix(trimmed, "<!--") {
			continue
		}

		// This is the first content paragraph
		purpose := trimmed
		if len(purpose) > maxPurposeLen {
			// Trim to maxPurposeLen, cutting at word boundary
			purpose = purpose[:maxPurposeLen]
			if idx := strings.LastIndex(purpose, " "); idx > 0 {
				purpose = purpose[:idx]
			}
		}
		return purpose
	}

	return ""
}

// insertNavBlock inserts or replaces a nav block in file content.
func insertNavBlock(content string, blockText string) string {
	lines := strings.Split(content, "\n")

	// Check for existing nav block
	blockStart := -1
	blockEnd := -1
	for i, line := range lines {
		if strings.Contains(line, "<!-- AGENT:NAV") {
			blockStart = i
		}
		if blockStart >= 0 && strings.TrimSpace(line) == navBlockEnd {
			blockEnd = i
			break
		}
	}

	if blockStart >= 0 && blockEnd >= 0 {
		// Replace existing block
		before := strings.Join(lines[:blockStart], "\n")
		after := ""
		if blockEnd+1 < len(lines) {
			after = strings.Join(lines[blockEnd+1:], "\n")
		}
		result := before + blockText + "\n" + after
		// Clean up extra blank lines
		result = cleanBlankLines(result)
		return result
	}

	// Check for frontmatter
	fmEnd := -1
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == frontmatterDelim {
		for i := 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == frontmatterDelim {
				fmEnd = i
				break
			}
		}
	}

	if fmEnd >= 0 {
		// Insert after frontmatter
		before := strings.Join(lines[:fmEnd+1], "\n")
		after := strings.Join(lines[fmEnd+1:], "\n")
		result := before + "\n" + blockText + "\n" + after
		result = cleanBlankLines(result)
		return result
	}

	// Insert at line 1
	result := blockText + "\n" + content
	result = cleanBlankLines(result)
	return result
}

// cleanBlankLines ensures exactly one blank line between nav block and first heading.
func cleanBlankLines(content string) string {
	lines := strings.Split(content, "\n")

	// Find the nav block end
	navEnd := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == navBlockEnd {
			navEnd = i
			break
		}
	}

	if navEnd < 0 {
		return content
	}

	// Count blank lines after nav block
	blankCount := 0
	for i := navEnd + 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "" {
			blankCount++
		} else {
			break
		}
	}

	// We want exactly one blank line
	if blankCount == 0 {
		// Insert a blank line
		newLines := make([]string, 0, len(lines)+1)
		newLines = append(newLines, lines[:navEnd+1]...)
		newLines = append(newLines, "")
		newLines = append(newLines, lines[navEnd+1:]...)
		return strings.Join(newLines, "\n")
	} else if blankCount > 1 {
		// Remove extra blank lines
		newLines := make([]string, 0, len(lines)-blankCount+1)
		newLines = append(newLines, lines[:navEnd+1]...)
		newLines = append(newLines, "")
		newLines = append(newLines, lines[navEnd+1+blankCount:]...)
		return strings.Join(newLines, "\n")
	}

	return content
}

// buildNavEntries converts parser sections to nav entries.
func buildNavEntries(sections []parser.Section) []navblock.NavEntry {
	entries := make([]navblock.NavEntry, len(sections))
	for i, s := range sections {
		prefix := strings.Repeat("#", s.Depth)
		entries[i] = navblock.NavEntry{
			Start: s.Start,
			End:   s.End,
			Name:  prefix + s.Text,
			About: "",
		}
	}
	return entries
}
