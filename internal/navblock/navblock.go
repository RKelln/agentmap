// Package navblock reads and writes AGENT:NAV blocks in markdown files.
package navblock

import (
	"fmt"
	"os"
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
	Start     int    // start line (1-indexed, inclusive)
	N         int    // line count (length of section)
	Name      string // heading with # prefix (e.g. "##Section")
	About     string // short description; may be empty
	WordCount int    // words in section content (heading line excluded); not serialized
}

// SeeEntry is a single line in the see section.
type SeeEntry struct {
	Path string // relative path to related file
	Why  string // reason to read it
}

// NormalizeHeading converts a raw heading name (with optional leading # characters
// and commas) into a canonical lookup key: strips leading '#' chars, strips
// leading/trailing whitespace, and removes all commas.
//
// Examples:
//
//	"##Setup, Configuration" → "Setup Configuration"
//	"#Authentication"        → "Authentication"
func NormalizeHeading(text string) string {
	// Strip leading # characters
	for strings.HasPrefix(text, "#") {
		text = strings.TrimPrefix(text, "#")
	}
	// Strip leading/trailing whitespace
	text = strings.TrimSpace(text)
	// Strip commas
	text = strings.ReplaceAll(text, ",", "")
	return text
}

// CountWords returns the number of whitespace-separated tokens in s,
// ignoring markdown heading prefixes (leading # chars) and blank lines.
func CountWords(s string) int {
	// Strip leading # characters from each line (heading prefix removal)
	stripped := strings.TrimLeft(s, "#")
	return len(strings.Fields(stripped))
}

// ParseResult holds the result of parsing an AGENT:NAV block.
type ParseResult struct {
	Block     NavBlock
	Start     int // line index of opening delimiter (1-indexed); -1 if not found
	End       int // line index of closing delimiter (1-indexed); -1 if not found
	Found     bool
	Corrupted bool
}

// ParseNavBlock extracts an AGENT:NAV block from file content.
// Returns a ParseResult with the parsed block, its start/end line (1-indexed),
// whether one was found, and whether the block is corrupted (malformed entries).
// Skips nav blocks inside fenced code blocks.
func ParseNavBlock(content string) ParseResult {
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
		return ParseResult{Start: -1, End: -1}
	}

	// Parse lines inside the block
	var parseCorrupted bool
	var block NavBlock
	block.Purpose, block.Nav, block.See, parseCorrupted = parseNavLines(lines[blockStart+1 : blockEnd])

	// Validate N >= 1 for all nav entries; invalid N means corruption.
	for i := range block.Nav {
		if block.Nav[i].N <= 0 {
			fmt.Fprintf(os.Stderr, "warning: nav entry %q has invalid line count %d — treating as no block\n", block.Nav[i].Name, block.Nav[i].N)
			block.Nav[i].N = 0
			parseCorrupted = true
		}
	}

	return ParseResult{
		Block:     block,
		Start:     blockStart + 1, // 1-indexed
		End:       blockEnd + 1,   // 1-indexed
		Found:     true,
		Corrupted: parseCorrupted,
	}
}

func parseNavLines(lines []string) (purpose string, nav []NavEntry, see []SeeEntry, corrupted bool) {
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
			entry, entryCorrupted := parseNavEntry(line)
			if entryCorrupted {
				corrupted = true
			}
			nav = append(nav, entry)
		}

		if parsingSee {
			entry := parseSeeEntry(line)
			see = append(see, entry)
		}
	}

	return purpose, nav, see, corrupted
}

func parseNavEntry(line string) (NavEntry, bool) {
	parts := strings.SplitN(line, ",", 4)
	if len(parts) < 3 {
		return NavEntry{}, true // corrupted: fewer than 3 fields
	}

	start, _ := strconv.Atoi(parts[0])
	n, _ := strconv.Atoi(parts[1])
	name := parts[2]
	about := ""
	if len(parts) >= 4 {
		about = parts[3]
	}

	return NavEntry{Start: start, N: n, Name: name, About: about}, false
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
		fmt.Fprintf(&b, "nav[%d]{s,n,name,about}:\n", len(block.Nav))
		for _, e := range block.Nav {
			fmt.Fprintf(&b, "%d,%d,%s,%s\n", e.Start, e.N, e.Name, e.About)
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

// SectionWordCount returns the word count of a section's content lines,
// excluding the heading line. lines is the full file as 0-indexed []string.
// start is the 1-indexed section start line; n is the line count.
func SectionWordCount(lines []string, start, n int) int {
	// 0-indexed: heading is at lines[start-1], content starts at lines[start]
	// content lines are lines[start : start+n-1] (skip heading, 0-indexed)
	contentEnd := start + n - 1
	if contentEnd > len(lines) {
		contentEnd = len(lines)
	}
	if start >= contentEnd {
		return 0
	}
	var words int
	for _, line := range lines[start:contentEnd] {
		words += CountWords(line)
	}
	return words
}

// RenderPurposeOnly produces a minimal nav block with only a purpose line.
func RenderPurposeOnly(purpose string) string {
	return "<!-- AGENT:NAV\npurpose:" + purpose + "\n-->"
}

// IsAutoGenerated returns true if s starts with the ~ prefix marker
// indicating auto-generated keyword descriptions.
func IsAutoGenerated(s string) bool {
	return strings.HasPrefix(s, "~")
}

// TrimAutoGenerated removes the ~ prefix marker if present.
func TrimAutoGenerated(s string) string {
	return strings.TrimPrefix(s, "~")
}
