// Package generate implements the agentmap generate command.
package generate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ryankelln/agentmap/internal/config"
	"github.com/ryankelln/agentmap/internal/discovery"
	"github.com/ryankelln/agentmap/internal/keywords"
	"github.com/ryankelln/agentmap/internal/navblock"
	"github.com/ryankelln/agentmap/internal/parser"
)

const (
	frontmatterDelim = "---"
	navBlockEnd      = "-->"
)

// Generate discovers markdown files under root and generates nav blocks for each.
func Generate(root string, cfg config.Config, dryRun bool) error {
	files, err := discovery.DiscoverFiles(root, cfg.Exclude)
	if err != nil {
		return fmt.Errorf("generate: discover files: %w", err)
	}

	var anySuccess bool
	for _, f := range files {
		fullPath := filepath.Join(root, f)
		report, err := File(fullPath, cfg, dryRun)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s: %v\n", f, err)
			continue
		}
		anySuccess = true
		fmt.Println(report)
	}

	if len(files) > 0 && !anySuccess {
		return fmt.Errorf("generate: no files processed successfully")
	}
	return nil
}

// File processes a single markdown file and writes a nav block.
// If outputPath is non-empty, the result is written there instead of modifying the source.
func File(path string, cfg config.Config, dryRun bool, outputPath ...string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("generate: read file: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	totalLines := len(lines)

	// Check for existing nav block to compute line offset
	blockStart, blockEnd := findNavBlock(lines)
	oldBlockLines := 0
	if blockStart >= 0 {
		oldBlockLines = blockEnd - blockStart + 1
	}

	headings := parser.ParseHeadings(string(content), cfg.MaxDepth)
	sections := parser.ComputeSections(headings, totalLines)

	var blockText string
	var report string

	if totalLines < cfg.MinLines || len(headings) == 0 {
		purpose := keywords.ExtractPurpose(string(content))
		blockText = navblock.RenderPurposeOnly(purpose)
		report = fmt.Sprintf("Skipped: %s (purpose-only)", path)
	} else {
		purpose := keywords.ExtractPurpose(string(content))
		// Build nav entries with keyword descriptions from original content
		originalEntries := buildNavEntries(sections, string(content), cfg)
		// Compute line offset: new block will shift headings down
		placeholder := navblock.NavBlock{Purpose: purpose, Nav: originalEntries}
		placeholderText := navblock.RenderNavBlock(placeholder)
		newBlockLines := len(strings.Split(placeholderText, "\n"))
		offset := newBlockLines - oldBlockLines

		adjusted := adjustSections(sections, offset)
		// Apply adjusted line numbers to entries (preserve keyword descriptions)
		entries := applyAdjustedLines(originalEntries, adjusted)
		block := navblock.NavBlock{
			Purpose: purpose,
			Nav:     entries,
		}
		blockText = navblock.RenderNavBlock(block)
		report = fmt.Sprintf("Generated: %s (%d sections)", path, len(entries))
	}

	if dryRun {
		return report, nil
	}

	newContent := insertNavBlock(string(content), blockText)

	// Write to output path if specified, otherwise modify source in place
	dest := path
	if len(outputPath) > 0 && outputPath[0] != "" {
		dest = outputPath[0]
	}

	if err := os.WriteFile(dest, []byte(newContent), 0o644); err != nil {
		return "", fmt.Errorf("generate: write file: %w", err)
	}

	return report, nil
}

// applyAdjustedLines copies Start/End line numbers from adjusted sections to entries.
// Preserves the About field (keyword descriptions) from the original entries.
func applyAdjustedLines(entries []navblock.NavEntry, adjusted []parser.Section) []navblock.NavEntry {
	if len(entries) != len(adjusted) {
		return entries
	}
	result := make([]navblock.NavEntry, len(entries))
	for i := range entries {
		result[i] = navblock.NavEntry{
			Start: adjusted[i].Start,
			End:   adjusted[i].End,
			Name:  entries[i].Name,
			About: entries[i].About,
		}
	}
	return result
}

// adjustSections shifts all Start/End line numbers by the given offset.
func adjustSections(sections []parser.Section, offset int) []parser.Section {
	if offset == 0 {
		return sections
	}
	adjusted := make([]parser.Section, len(sections))
	for i, s := range sections {
		adjusted[i] = parser.Section{
			Heading: s.Heading,
			Start:   s.Start + offset,
			End:     s.End + offset,
		}
	}
	return adjusted
}

// findNavBlock returns the 0-indexed start and end lines of an existing nav block.
// Skips nav blocks inside fenced code blocks. Only searches after frontmatter.
// Returns -1,-1 if any non-whitespace, non-nav-block content appears before nav block.
func findNavBlock(lines []string) (start, end int) {
	// Find frontmatter end first
	fmEnd := -1
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == frontmatterDelim {
		for i := 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == frontmatterDelim {
				fmEnd = i
				break
			}
		}
	}

	searchStart := 0
	if fmEnd >= 0 {
		searchStart = fmEnd + 1
	}

	inFence := false
	searchEnd := searchStart + 20
	if searchEnd > len(lines) {
		searchEnd = len(lines)
	}
	for i := searchStart; i < searchEnd; i++ {
		trimmed := strings.TrimSpace(lines[i])

		// Track fenced code blocks
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}

		// Empty line - skip (nav block can have leading blank lines)
		if trimmed == "" {
			continue
		}

		// Found nav block start
		if strings.HasPrefix(trimmed, "<!-- AGENT:NAV") {
			start = i
			for j := i + 1; j < len(lines); j++ {
				if strings.TrimSpace(lines[j]) == "-->" {
					return start, j
				}
			}
			return -1, -1 // incomplete block
		}

		// Any other non-empty content - continue searching but no longer in valid zone
		// Move searchEnd to current position - can't find valid nav block after this
		searchEnd = i
	}

	return -1, -1
}

// insertNavBlock inserts or replaces a nav block in file content.
// Skips nav blocks inside fenced code blocks.
func insertNavBlock(content string, blockText string) string {
	lines := strings.Split(content, "\n")

	// Check for existing nav block (only in first 20 lines after frontmatter)
	const maxExistingBlockLine = 20
	blockStart := -1
	blockEnd := -1
	inFence := false
	for i, line := range lines {
		if i > maxExistingBlockLine {
			break
		}
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if strings.Contains(line, "<!-- AGENT:NAV") {
			blockStart = i
		}
		if blockStart >= 0 && trimmed == navBlockEnd {
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

// buildNavEntries converts parser sections to nav entries with keyword descriptions.
// Applies three-tier subsection logic:
//   - Under subThreshold: no subsection info (h3 children skipped)
//   - Between subThreshold and expandThreshold: > hints for h3 children (h3 skipped)
//   - Over expandThreshold: full h3 entries as separate nav entries
func buildNavEntries(sections []parser.Section, content string, cfg config.Config) []navblock.NavEntry {
	lines := strings.Split(content, "\n")
	var entries []navblock.NavEntry
	skipUntil := 0 // track sections to skip (h3 children of parents not expanding)

	for i, s := range sections {
		if s.Start < skipUntil {
			continue
		}

		sectionContent := getSectionContent(lines, s.Start, s.End)
		sectionSize := s.End - s.Start + 1

		prefix := strings.Repeat("#", s.Depth)
		about := keywords.ExtractKeywords(sectionContent, 5)

		// Apply subsection logic for h2 entries (depth 2) with h3 children
		if s.Depth == 2 {
			h3Children := getH3Children(sections, i)

			if sectionSize >= cfg.ExpandThreshold {
				// Full expansion: add h3 children as separate entries
				entries = append(entries, navblock.NavEntry{
					Start: s.Start,
					End:   s.End,
					Name:  prefix + s.Text,
					About: about,
				})
				for _, child := range h3Children {
					childContent := getSectionContent(lines, child.Start, child.End)
					childAbout := keywords.ExtractKeywords(childContent, 5)
					childPrefix := strings.Repeat("#", child.Depth)
					entries = append(entries, navblock.NavEntry{
						Start: child.Start,
						End:   child.End,
						Name:  childPrefix + child.Text,
						About: childAbout,
					})
				}
				continue
			} else if len(h3Children) > 0 {
				// Small or medium section: skip h3 children
				skipUntil = h3Children[len(h3Children)-1].Start + 1
				// Only add > hints for medium sections (>= sub_threshold)
				if sectionSize >= cfg.SubThreshold {
					hints := make([]string, len(h3Children))
					for j, child := range h3Children {
						hints[j] = strings.ReplaceAll(child.Text, ",", ";")
					}
					if about != "" {
						about += ">" + strings.Join(hints, ";")
					} else {
						about = ">" + strings.Join(hints, ";")
					}
				}
			}
		}

		entries = append(entries, navblock.NavEntry{
			Start: s.Start,
			End:   s.End,
			Name:  prefix + s.Text,
			About: about,
		})
	}

	return entries
}

// getSectionContent extracts the text content between start and end line numbers (1-indexed).
func getSectionContent(lines []string, start, end int) string {
	if start < 1 || end < start {
		return ""
	}
	if end > len(lines) {
		end = len(lines)
	}
	return strings.Join(lines[start-1:end], "\n")
}

// getH3Children returns h3 sections that are immediate children of the section at index i.
func getH3Children(sections []parser.Section, parentIdx int) []parser.Section {
	if parentIdx+1 >= len(sections) {
		return nil
	}

	parent := sections[parentIdx]
	if parent.Depth != 2 {
		return nil
	}

	var children []parser.Section
	for j := parentIdx + 1; j < len(sections); j++ {
		s := sections[j]
		if s.Depth == 3 && s.Start <= parent.End {
			children = append(children, s)
		} else if s.Depth <= 2 {
			break
		}
	}

	return children
}
