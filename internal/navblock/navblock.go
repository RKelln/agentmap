// Package navblock reads and writes AGENT:NAV blocks in markdown files.
package navblock

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// NavBlock represents a complete AGENT:NAV block.
type NavBlock struct {
	Purpose string
	Lines   int // total content lines in the file (excluding the nav block itself); 0 if unknown
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

var (
	attrRe    = regexp.MustCompile(`\s*\{:[^}]*\}$`)
	bracketRe = regexp.MustCompile(`\s*\[[^\]]*\]$`)
)

// stripTrailingParens removes a trailing parenthetical expression from text.
// It handles nested parentheses by finding the matching '(' for the final ')'.
func stripTrailingParens(text string) string {
	if !strings.HasSuffix(text, ")") {
		return text
	}
	depth := 1
	i := len(text) - 2
	for i >= 0 && depth > 0 {
		switch text[i] {
		case ')':
			depth++
		case '(':
			depth--
		}
		i--
	}
	if depth != 0 {
		return text
	}
	parenPos := i + 1
	// Strip whitespace before the parenthetical.
	j := parenPos
	for j > 0 && text[j-1] == ' ' {
		j--
	}
	return text[:j]
}

// NormalizeHeading converts a raw heading name (with optional leading # characters
// and commas) into a canonical lookup key: strips leading '#' chars, strips
// leading/trailing whitespace, removes trailing markdown attributes, bracketed
// text, and parenthetical content, then removes all commas.
//
// Examples:
//
//	"##Setup, Configuration" → "Setup Configuration"
//	"#Authentication"        → "Authentication"
//	"## Title {: .class}"    → "Title"
//	"## Title (deprecated)"  → "Title"
func NormalizeHeading(text string) string {
	// Strip leading # characters
	for strings.HasPrefix(text, "#") {
		text = strings.TrimPrefix(text, "#")
	}
	// Strip leading/trailing whitespace
	text = strings.TrimSpace(text)
	// Strip trailing suffixes: markdown attrs, brackets, parens
	changed := true
	for changed {
		changed = false
		if attrRe.MatchString(text) {
			text = attrRe.ReplaceAllString(text, "")
			changed = true
		}
		if bracketRe.MatchString(text) {
			text = bracketRe.ReplaceAllString(text, "")
			changed = true
		}
		if stripped := stripTrailingParens(text); stripped != text {
			text = stripped
			changed = true
		}
	}
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

	blockStart, blockEnd := LocateNavBlock(lines)

	if blockStart < 0 || blockEnd < 0 {
		return ParseResult{Start: -1, End: -1}
	}

	// Parse lines inside the block
	var parseCorrupted bool
	var block NavBlock
	block.Purpose, block.Lines, block.Nav, block.See, parseCorrupted = parseNavLines(lines[blockStart+1 : blockEnd])

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

func parseNavLines(lines []string) (purpose string, fileLines int, nav []NavEntry, see []SeeEntry, corrupted bool) {
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

		// lines field
		if strings.HasPrefix(line, "lines:") {
			fileLines, _ = strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "lines:")))
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

	return purpose, fileLines, nav, see, corrupted
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
	if block.Lines > 0 {
		fmt.Fprintf(&b, "lines:%d\n", block.Lines)
	}

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

// AutoGeneratedPrefix is the marker prepended to keyword-generated descriptions
// to signal auto-generated content that has not been reviewed.
const AutoGeneratedPrefix = "~"

// FindFrontmatterEnd returns the 0-indexed line of the closing --- of YAML
// frontmatter, or -1 if the file has no frontmatter.
func FindFrontmatterEnd(lines []string) int {
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return -1
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return i
		}
	}
	return -1 // unclosed frontmatter — treat as no frontmatter
}

// LocateNavBlock returns the 0-indexed start and end lines of an AGENT:NAV block.
//
// A nav block is the first non-blank content after optional YAML frontmatter.
// If the first non-blank content after frontmatter is not "<!-- AGENT:NAV", there is
// no nav block. Nav blocks inside fenced code blocks are never matched.
// Single-line examples (<!-- AGENT:NAV ... --> on one line) are not nav blocks.
// Returns (-1, -1) if no nav block is found or if the block is unclosed.
func LocateNavBlock(lines []string) (start, end int) {
	searchFrom := 0
	if fmEnd := FindFrontmatterEnd(lines); fmEnd >= 0 {
		searchFrom = fmEnd + 1
	}

	// Skip leading blank lines after frontmatter
	i := searchFrom
	for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
		i++
	}

	if i >= len(lines) {
		return -1, -1
	}

	trimmed := strings.TrimSpace(lines[i])

	// Single-line example: <!-- AGENT:NAV ... --> on one line — not a nav block
	if strings.HasPrefix(trimmed, "<!-- AGENT:NAV") && strings.Contains(trimmed, "-->") {
		return -1, -1
	}

	// If first non-blank content is inside a code fence, no nav block
	if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
		return -1, -1
	}

	// If first non-blank content is not a nav block opener, no nav block
	if !strings.HasPrefix(trimmed, "<!-- AGENT:NAV") {
		return -1, -1
	}

	// Found opener — scan for closing -->
	for j := i + 1; j < len(lines); j++ {
		if strings.TrimSpace(lines[j]) == "-->" {
			return i, j
		}
	}
	return -1, -1 // unclosed block
}

// RenderPurposeOnly produces a minimal nav block with a purpose and line count.
func RenderPurposeOnly(purpose string, lines int) string {
	s := "<!-- AGENT:NAV\npurpose:" + purpose + "\n"
	if lines > 0 {
		s += fmt.Sprintf("lines:%d\n", lines)
	}
	s += "-->"
	return s
}

// IsAutoGenerated returns true if s starts with the ~ prefix marker
// indicating auto-generated keyword descriptions.
func IsAutoGenerated(s string) bool {
	return strings.HasPrefix(s, AutoGeneratedPrefix)
}

// TrimAutoGenerated removes the ~ prefix marker if present.
func TrimAutoGenerated(s string) string {
	return strings.TrimPrefix(s, AutoGeneratedPrefix)
}

// FilesEntry is a single file entry in a files block.
type FilesEntry struct {
	RelPath string // relative to repo root, e.g. "docs/auth.md"
	Lines   int    // content line count (excluding nav block); 0 if unknown
	About   string // short description of the file's purpose
}

// FilesBlock represents a project-level AGENT:NAV files block.
type FilesBlock struct {
	Purpose string
	Lines   int // total line count of the AGENTMAP.md file itself; 0 if unknown
	Entries []FilesEntry
}

// RenderFilesBlock produces the full AGENT:NAV block text for a files block.
// Directory prefix lines (no comma) are emitted before the files in each directory.
// Root-level files appear before any directory prefix.
// Entries are sorted by directory then by filename before rendering so that
// directory prefix lines are never duplicated even when the caller provides
// an unsorted slice.
func RenderFilesBlock(block FilesBlock) string {
	var b strings.Builder

	// Sort a copy so callers that provide a pre-sorted slice pay no penalty
	// and callers that provide an unsorted slice get correct output.
	entries := make([]FilesEntry, len(block.Entries))
	copy(entries, block.Entries)
	sort.Slice(entries, func(i, j int) bool {
		di := dirOf(entries[i].RelPath)
		dj := dirOf(entries[j].RelPath)
		if di != dj {
			// Root-level files (empty dir) sort before any subdirectory.
			if di == "" {
				return true
			}
			if dj == "" {
				return false
			}
			return di < dj
		}
		return entries[i].RelPath < entries[j].RelPath
	})

	b.WriteString("<!-- AGENT:NAV\n")
	b.WriteString("purpose:" + block.Purpose + "\n")
	if block.Lines > 0 {
		fmt.Fprintf(&b, "lines:%d\n", block.Lines)
	}
	fmt.Fprintf(&b, "files[%d]{path,lines,about}:\n", len(entries))

	var lastDir string
	for _, e := range entries {
		dir := dirOf(e.RelPath)
		name := e.RelPath
		if slash := strings.LastIndex(e.RelPath, "/"); slash >= 0 {
			name = e.RelPath[slash+1:]
		}

		if dir != lastDir {
			if dir != "" {
				b.WriteString(dir + "\n")
			}
			lastDir = dir
		}
		fmt.Fprintf(&b, "%s,%d,%s\n", name, e.Lines, e.About)
	}

	b.WriteString("-->")
	return b.String()
}

// dirOf returns the directory portion of relPath with a trailing slash,
// or "" for root-level files.
func dirOf(relPath string) string {
	slash := strings.LastIndex(relPath, "/")
	if slash < 0 {
		return ""
	}
	return relPath[:slash+1]
}

// ParseFilesBlock extracts a files block from content.
// Returns the parsed FilesBlock and whether it was found.
// Supports both the legacy {path,about} format and the current {path,lines,about} format.
func ParseFilesBlock(content string) (FilesBlock, bool) {
	lines := strings.Split(content, "\n")

	blockStart, blockEnd := LocateNavBlock(lines)

	if blockStart < 0 || blockEnd < 0 {
		return FilesBlock{}, false
	}

	inner := lines[blockStart+1 : blockEnd]

	var fb FilesBlock
	var parsingFiles bool
	var hasLinesCol bool
	var currentDir string

	for _, line := range inner {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "purpose:") {
			fb.Purpose = strings.TrimPrefix(line, "purpose:")
			continue
		}
		if strings.HasPrefix(line, "lines:") {
			fb.Lines, _ = strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "lines:")))
			continue
		}
		if strings.HasPrefix(line, "files[") && strings.HasSuffix(line, "}:") {
			parsingFiles = true
			hasLinesCol = strings.Contains(line, "lines")
			currentDir = ""
			continue
		}
		if !parsingFiles {
			continue
		}
		// Directory prefix: ends with "/" and contains no comma
		if strings.HasSuffix(line, "/") && !strings.Contains(line, ",") {
			currentDir = line
			continue
		}
		// File entry: contains comma
		if strings.Contains(line, ",") {
			if hasLinesCol {
				// format: name,lines,about
				parts := strings.SplitN(line, ",", 3)
				if len(parts) >= 2 {
					name := parts[0]
					linesVal, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
					about := ""
					if len(parts) >= 3 {
						about = parts[2]
					}
					relPath := currentDir + name
					fb.Entries = append(fb.Entries, FilesEntry{RelPath: relPath, Lines: linesVal, About: about})
				}
			} else {
				// legacy format: name,about
				parts := strings.SplitN(line, ",", 2)
				if len(parts) < 2 {
					continue
				}
				name := parts[0]
				about := parts[1]
				relPath := currentDir + name
				fb.Entries = append(fb.Entries, FilesEntry{RelPath: relPath, About: about})
			}
		}
	}

	if !parsingFiles {
		return FilesBlock{}, false
	}

	return fb, true
}
