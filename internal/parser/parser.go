// Package parser extracts headings from markdown files to build nav blocks.
package parser

import (
	"bufio"
	"strings"
)

// Heading represents a markdown heading (h1-h3) with its line number and text.
type Heading struct {
	Line  int    // 1-indexed line number
	Depth int    // 1 for #, 2 for ##, 3 for ###
	Text  string // heading text without # prefix
}

// ParseHeadings extracts h1-h3 headings from markdown content.
// Headings inside fenced code blocks and HTML comments are skipped.
func ParseHeadings(content string, maxDepth int) []Heading {
	scanner := bufio.NewScanner(strings.NewReader(content))

	var headings []Heading
	inFence := false
	inComment := false
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Track YAML frontmatter: skip lines between opening/closing ---
		if lineNum == 1 && strings.HasPrefix(line, "---") {
			// Skip until closing ---
			for scanner.Scan() {
				lineNum++
				if strings.TrimSpace(scanner.Text()) == "---" {
					break
				}
			}
			continue
		}

		// Track fenced code blocks (``` or ~~~)
		trimmed := strings.TrimSpace(line)
		if !inComment && (strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~")) {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}

		// Track HTML comments (multi-line)
		if strings.Contains(line, "<!--") && !strings.Contains(line, "-->") {
			inComment = true
			continue
		}
		if inComment {
			if strings.Contains(line, "-->") {
				inComment = false
			}
			continue
		}
		if strings.Contains(line, "<!--") && strings.Contains(line, "-->") {
			// Single-line comment, skip
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

	return headings
}
