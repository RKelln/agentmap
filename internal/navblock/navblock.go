// Package navblock reads and writes AGENT:NAV blocks in markdown files.
package navblock

import (
	"fmt"
	"strconv"
	"strings"
)

// NavBlock represents a complete AGENT:NAV block.
type NavBlock struct {
	Purpose string
	Nav     []NavEntry
	See     []SeeEntry
}

// NavEntry is a single line in the nav section.
type NavEntry struct {
	Start int    // start line (1-indexed, inclusive)
	End   int    // end line (1-indexed, inclusive)
	Name  string // heading with # prefix (e.g. "##Section")
	About string // short description; may be empty
}

// SeeEntry is a single line in the see section.
type SeeEntry struct {
	Path string // relative path to related file
	Why  string // reason to read it
}

// ParseNavBlock extracts an AGENT:NAV block from file content.
// Returns the block, its start line, end line (1-indexed), and whether one was found.
// Skips nav blocks inside fenced code blocks.
func ParseNavBlock(content string) (block NavBlock, startLine, endLine int, found bool) {
	lines := strings.Split(content, "\n")
	blockStart := -1
	blockEnd := -1

	// Only search for nav block start in first 20 lines of the file
	// (after frontmatter if present). Nav blocks inside code examples
	// or in body content are ignored.
	const maxNavBlockLine = 20

	inFence := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Track fenced code blocks (but still process nav block ends)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}

		// Only look for nav block START in first 20 lines
		// But allow finding the END even if beyond line 20
		if i <= maxNavBlockLine {
			// Check for nav block start (skip single-line examples like `<!-- AGENT:NAV ... -->`)
			if strings.Contains(line, "<!-- AGENT:NAV") && !strings.Contains(line, "-->") {
				if blockStart < 0 {
					blockStart = i
				}
			}
		}

		// Check for nav block end (can be beyond line 20)
		if blockStart >= 0 && trimmed == "-->" {
			blockEnd = i
			break
		}
	}

	if blockStart < 0 || blockEnd < 0 {
		return NavBlock{}, 0, 0, false
	}

	// Parse lines inside the block
	block.Purpose, block.Nav, block.See = parseNavLines(lines[blockStart+1 : blockEnd])
	return block, blockStart + 1, blockEnd + 1, true // 1-indexed
}

func parseNavLines(lines []string) (purpose string, nav []NavEntry, see []SeeEntry) {
	parsingNav := false
	parsingSee := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// purpose line
		if strings.HasPrefix(line, "purpose:") {
			purpose = strings.TrimPrefix(line, "purpose:")
			continue
		}

		// nav header
		if strings.HasPrefix(line, "nav[") && strings.HasSuffix(line, "}:") {
			parsingNav = true
			parsingSee = false
			continue
		}

		// see header
		if strings.HasPrefix(line, "see[") && strings.HasSuffix(line, "}:") {
			parsingNav = false
			parsingSee = true
			continue
		}

		if parsingNav {
			entry := parseNavEntry(line)
			nav = append(nav, entry)
		}

		if parsingSee {
			entry := parseSeeEntry(line)
			see = append(see, entry)
		}
	}

	return purpose, nav, see
}

func parseNavEntry(line string) NavEntry {
	parts := strings.SplitN(line, ",", 4)
	if len(parts) < 3 {
		return NavEntry{}
	}

	start, _ := strconv.Atoi(parts[0])
	end, _ := strconv.Atoi(parts[1])
	name := parts[2]
	about := ""
	if len(parts) >= 4 {
		about = parts[3]
	}

	return NavEntry{Start: start, End: end, Name: name, About: about}
}

func parseSeeEntry(line string) SeeEntry {
	parts := strings.SplitN(line, ",", 2)
	if len(parts) < 2 {
		return SeeEntry{}
	}
	return SeeEntry{Path: parts[0], Why: parts[1]}
}

// RenderNavBlock produces the full AGENT:NAV block text.
func RenderNavBlock(block NavBlock) string {
	var b strings.Builder

	b.WriteString("<!-- AGENT:NAV\n")
	b.WriteString("purpose:" + block.Purpose + "\n")

	if len(block.Nav) > 0 {
		fmt.Fprintf(&b, "nav[%d]{s,e,name,about}:\n", len(block.Nav))
		for _, e := range block.Nav {
			fmt.Fprintf(&b, "%d,%d,%s,%s\n", e.Start, e.End, e.Name, e.About)
		}
	}

	if len(block.See) > 0 {
		fmt.Fprintf(&b, "see[%d]{path,why}:\n", len(block.See))
		for _, e := range block.See {
			fmt.Fprintf(&b, "%s,%s\n", e.Path, e.Why)
		}
	}

	b.WriteString("-->")
	return b.String()
}

// RenderPurposeOnly produces a minimal nav block with only a purpose line.
func RenderPurposeOnly(purpose string) string {
	return "<!-- AGENT:NAV\npurpose:" + purpose + "\n-->"
}
