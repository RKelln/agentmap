// Package parser extracts headings from markdown files to build nav blocks.
package parser

import (
	"bufio"
	"fmt"
	"strings"
)

// Heading represents a markdown heading (h1-h3) with its line number and text.
type Heading struct {
	Line  int    // 1-indexed line number
	Depth int    // 1 for #, 2 for ##, 3 for ###
	Text  string // heading text without # prefix
}

// isFrontmatterClose reports whether a trimmed line closes YAML frontmatter.
// It mirrors the logic in navblock.FindFrontmatterEnd.
func isFrontmatterClose(trimmed string) (isClose, dirty bool) {
	if !strings.HasPrefix(trimmed, "---") {
		return false, false
	}
	if len(trimmed) == 3 {
		return true, false
	}
	if trimmed[3] == '-' {
		return false, false
	}
	return true, true
}

// ParseHeadings extracts h1-h3 headings from markdown content.
// Headings inside fenced code blocks and HTML comments are skipped.
// warnings contains a message for each structural anomaly detected (e.g. unclosed fence at EOF).
func ParseHeadings(content string, maxDepth int) ([]Heading, []string) {
	scanner := bufio.NewScanner(strings.NewReader(content))

	var headings []Heading
	inFence := false
	fenceChar := byte(0) // '`' or '~' of the opening fence marker
	fenceLen := 0        // number of fence chars in the opening marker
	inComment := false
	lineNum := 0

	var warnings []string

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Track YAML frontmatter: skip lines between opening/closing ---
		if lineNum == 1 && strings.HasPrefix(line, "---") {
			// Skip until closing ---
			for scanner.Scan() {
				lineNum++
				text := scanner.Text()
				if isClose, dirty := isFrontmatterClose(strings.TrimSpace(text)); isClose {
					if dirty {
						warnings = append(warnings, fmt.Sprintf("frontmatter close delimiter has trailing content: %q", text))
					}
					break
				}
			}
			continue
		}

		// Track fenced code blocks (``` or ~~~).
		// Per CommonMark: only a bare closing fence (same char, >= opener length,
		// no info string) can close an open fence. A line like ```bash can only
		// ever open a fence, never close one.
		trimmed := strings.TrimSpace(line)
		if !inComment {
			if inFence {
				// Check for a valid closing fence: same char, >= fenceLen, no trailing content.
				if len(trimmed) >= fenceLen && trimmed[0] == fenceChar {
					n := 0
					for n < len(trimmed) && trimmed[n] == fenceChar {
						n++
					}
					if n >= fenceLen && strings.TrimSpace(trimmed[n:]) == "" {
						inFence = false
						fenceChar = 0
						fenceLen = 0
					}
				}
				continue
			}
			// Not in a fence: detect an opening fence marker (3+ ` or ~).
			if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
				ch := trimmed[0]
				n := 0
				for n < len(trimmed) && trimmed[n] == ch {
					n++
				}
				inFence = true
				fenceChar = ch
				fenceLen = n
				continue
			}
		}
		if inFence {
			continue
		}

		// Track HTML block comments (multi-line).
		// Only treat <!-- as a comment opener when it starts the trimmed line —
		// this avoids false positives from prose that mentions <!-- in inline
		// code spans (e.g. "Find files with `<!-- AGENT:NAV` block.").
		if strings.HasPrefix(trimmed, "<!--") && !strings.Contains(line, "-->") {
			inComment = true
			continue
		}
		if inComment {
			if strings.Contains(line, "-->") {
				inComment = false
			}
			continue
		}
		if strings.HasPrefix(trimmed, "<!--") && strings.Contains(line, "-->") {
			// Single-line block comment, skip
			continue
		}

		// Check for heading
		if len(line) > 0 && line[0] == '#' {
			depth := 0
			for _, ch := range line {
				if ch == '#' {
					depth++
				} else {
					break
				}
			}
			// Must have 1-# depth, followed by a space, and within maxDepth
			if depth > 0 && depth <= maxDepth && depth < len(line) && line[depth] == ' ' {
				text := strings.TrimSpace(line[depth+1:])
				headings = append(headings, Heading{
					Line:  lineNum,
					Depth: depth,
					Text:  text,
				})
			}
		}
	}

	if inFence {
		warnings = append(warnings, "unclosed code fence at end of file (malformed markdown)")
	}

	return headings, warnings
}
