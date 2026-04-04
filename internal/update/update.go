// Package update implements the agentmap update command.
package update

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/RKelln/agentmap/internal/config"
	"github.com/RKelln/agentmap/internal/discovery"
	"github.com/RKelln/agentmap/internal/generate"
	"github.com/RKelln/agentmap/internal/gitutil"
	"github.com/RKelln/agentmap/internal/navblock"
	"github.com/RKelln/agentmap/internal/parser"
)

// ReportType indicates the type of update report entry.
type ReportType int

const (
	// ReportOK means heading matched and line numbers unchanged.
	ReportOK ReportType = iota
	// ReportShifted means heading matched but line numbers changed.
	ReportShifted
	// ReportContentChanged means section has content changes from git diff.
	ReportContentChanged
	// ReportNew means new heading with no description.
	ReportNew
	// ReportDeleted means heading removed from document.
	ReportDeleted
)

// noChanges is returned when a file has no nav block or no changes needed.
const noChanges = "no-changes"

// ReportEntry represents a single entry in the update report.
type ReportEntry struct {
	Type          ReportType
	Name          string
	OldStart      int
	OldEnd        int
	NewStart      int
	NewEnd        int
	ModifiedCount int
	CurrentAbout  string
}

func (r ReportEntry) String() string {
	switch r.Type {
	case ReportOK:
		return fmt.Sprintf("  OK: %s (%d-%d)", r.Name, r.NewStart, r.NewEnd)
	case ReportShifted:
		return fmt.Sprintf("  shifted: %s (%d-%d -> %d-%d)", r.Name, r.OldStart, r.OldEnd, r.NewStart, r.NewEnd)
	case ReportContentChanged:
		return fmt.Sprintf("  content-changed: %s (lines %d-%d; %d lines modified)\n    current description: %q", r.Name, r.NewStart, r.NewEnd, r.ModifiedCount, r.CurrentAbout)
	case ReportNew:
		return fmt.Sprintf("  new: %s (%d-%d; no description)", r.Name, r.NewStart, r.NewEnd)
	case ReportDeleted:
		return fmt.Sprintf("  deleted: %s (removed from document)", r.Name)
	default:
		return ""
	}
}

// Update discovers markdown files with nav blocks under root and updates line numbers.
func Update(root string, cfg config.Config, dryRun, quiet bool) error {
	files, err := discovery.DiscoverFiles(root, cfg.Exclude)
	if err != nil {
		return fmt.Errorf("update: discover files: %w", err)
	}

	// §12.4: one git diff call for the whole repo, not one per file.
	repoChanges, err := gitutil.RepoChanges(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: git diff: %v; content-change detection disabled\n", err)
	}

	var anyChanged bool
	for _, f := range files {
		fullPath := filepath.Join(root, f)
		var changedLines []gitutil.LineRange
		if repoChanges != nil {
			changedLines = repoChanges[f]
		}
		report, err := File(fullPath, cfg, dryRun, quiet, changedLines)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s: %v\n", f, err)
			continue
		}
		if report != noChanges {
			anyChanged = true
			if !quiet {
				fmt.Println(report)
			}
		}
	}

	if len(files) > 0 && !anyChanged {
		fmt.Println("No changes")
	}
	return nil
}

// File processes a single markdown file and updates its nav block.
// changedLines contains pre-computed git diff ranges for this file; if nil, falls back
// to per-file git diff (for direct single-file calls from CLI).
// Returns noChanges if the file has no nav block or no changes needed.
func File(path string, cfg config.Config, dryRun, quiet bool, changedLines ...[]gitutil.LineRange) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("update: read file: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	totalLines := len(lines)

	pr := navblock.ParseNavBlock(string(content))
	oldBlock, hasBlock, corrupted := pr.Block, pr.Found, pr.Corrupted
	if corrupted {
		fmt.Fprintf(os.Stderr, "warning: %s: nav block is corrupted — run 'agentmap generate' to regenerate\n", path)
		return noChanges, nil
	}
	if !hasBlock {
		return noChanges, nil
	}

	// Compute content lines (total newlines minus the nav block itself).
	contentLines := strings.Count(string(content), "\n") - (pr.End - pr.Start + 1)
	if contentLines < 0 {
		contentLines = 0
	}

	headings := parser.ParseHeadings(string(content), cfg.MaxDepth)
	sections := parser.ComputeSections(headings, totalLines)

	// Handle purpose-only files: no headings, but lines:N may still need updating.
	if len(headings) == 0 {
		// Only refresh lines:N if the block already has it (non-zero).
		linesChanged := oldBlock.Lines != 0 && oldBlock.Lines != contentLines
		if !linesChanged {
			return noChanges, nil
		}
		oldLinesCount := oldBlock.Lines
		oldBlock.Lines = contentLines
		blockText := navblock.RenderNavBlock(oldBlock)
		if dryRun {
			return fmt.Sprintf("Updated: %s\n  lines-updated: %d -> %d", path, oldLinesCount, contentLines), nil
		}
		newContent := insertNavBlock(string(content), blockText)
		if err := os.WriteFile(path, []byte(newContent), 0o644); err != nil {
			return "", fmt.Errorf("update: write file: %w", err)
		}
		if quiet {
			return noChanges, nil
		}
		return fmt.Sprintf("Updated: %s\n  lines-updated: %d -> %d", path, oldLinesCount, contentLines), nil
	}

	// Use pre-computed ranges if provided, otherwise fall back to per-file git diff.
	var fileChanges []gitutil.LineRange
	if len(changedLines) > 0 && changedLines[0] != nil {
		fileChanges = changedLines[0]
	} else if len(changedLines) == 0 {
		fileChanges = getChangedLines(path)
	}
	entryReports := buildEntryReports(oldBlock.Nav, sections, fileChanges)

	hasChanges := (oldBlock.Lines != 0 && oldBlock.Lines != contentLines)
	for _, er := range entryReports {
		if er.Type != ReportOK {
			hasChanges = true
			break
		}
	}

	if !hasChanges {
		return noChanges, nil
	}

	block := buildUpdatedBlock(oldBlock, sections, entryReports, lines, cfg, contentLines)
	blockText := navblock.RenderNavBlock(block)

	if dryRun {
		return formatReport(path, entryReports), nil
	}

	newContent := insertNavBlock(string(content), blockText)
	if err := os.WriteFile(path, []byte(newContent), 0o644); err != nil {
		return "", fmt.Errorf("update: write file: %w", err)
	}

	if quiet {
		return noChanges, nil
	}
	return formatReport(path, entryReports), nil
}

func getChangedLines(path string) []gitutil.LineRange {
	ranges, err := gitutil.FileChanges(path)
	if err != nil || ranges == nil {
		return nil
	}
	return ranges
}

func buildEntryReports(oldNav []navblock.NavEntry, sections []parser.Section, changedLines []gitutil.LineRange) []ReportEntry {
	var reports []ReportEntry

	oldByName := make(map[string]navblock.NavEntry)
	for _, e := range oldNav {
		key := navblock.NormalizeHeading(e.Name)
		oldByName[key] = e
	}

	used := make(map[int]bool)

	for _, s := range sections {
		key := navblock.NormalizeHeading(s.Text)
		oldEntry, found := oldByName[key]
		prefix := strings.Repeat("#", s.Depth)
		name := prefix + navblock.NormalizeHeading(s.Text)

		if !found {
			reports = append(reports, ReportEntry{
				Type:     ReportNew,
				Name:     name,
				NewStart: s.Start,
				NewEnd:   s.End,
			})
			continue
		}

		used[oldEntry.Start] = true

		reportType := ReportOK
		oldEnd := oldEntry.Start + oldEntry.N - 1
		if oldEntry.Start != s.Start || oldEnd != s.End {
			reportType = ReportShifted
		}

		modifiedCount := 0
		if len(changedLines) > 0 {
			for _, cl := range changedLines {
				if s.Start <= cl.End && s.End >= cl.Start {
					clLen := cl.End - cl.Start + 1
					if clLen > modifiedCount {
						modifiedCount = clLen
					}
				}
			}
			if modifiedCount > 0 {
				reportType = ReportContentChanged
			}
		}

		reports = append(reports, ReportEntry{
			Type:          reportType,
			Name:          name,
			OldStart:      oldEntry.Start,
			OldEnd:        oldEntry.Start + oldEntry.N - 1,
			NewStart:      s.Start,
			NewEnd:        s.End,
			ModifiedCount: modifiedCount,
			CurrentAbout:  oldEntry.About,
		})
	}

	for _, e := range oldNav {
		key := navblock.NormalizeHeading(e.Name)
		if _, found := oldByName[key]; found && !used[oldByName[key].Start] {
			reports = append(reports, ReportEntry{
				Type: ReportDeleted,
				Name: e.Name,
			})
		}
	}

	return reports
}

func buildUpdatedBlock(oldBlock navblock.NavBlock, sections []parser.Section, _ []ReportEntry, lines []string, cfg config.Config, contentLines int) navblock.NavBlock {
	oldByName := make(map[string]navblock.NavEntry)
	for _, e := range oldBlock.Nav {
		key := navblock.NormalizeHeading(e.Name)
		oldByName[key] = e
	}

	var newNav []navblock.NavEntry

	for _, s := range sections {
		key := navblock.NormalizeHeading(s.Text)
		oldEntry, found := oldByName[key]

		prefix := strings.Repeat("#", s.Depth)
		wc := navblock.SectionWordCount(lines, s.Start, s.Len())

		if found {
			newNav = append(newNav, navblock.NavEntry{
				Start:     s.Start,
				N:         s.Len(),
				Name:      prefix + navblock.NormalizeHeading(s.Text),
				About:     oldEntry.About,
				WordCount: wc,
			})
		} else {
			newNav = append(newNav, navblock.NavEntry{
				Start:     s.Start,
				N:         s.Len(),
				Name:      prefix + navblock.NormalizeHeading(s.Text),
				About:     "",
				WordCount: wc,
			})
		}
	}

	// Only carry forward lines:N if the original block had it.
	outLines := 0
	if oldBlock.Lines != 0 {
		outLines = contentLines
	}

	return navblock.NavBlock{
		Purpose: oldBlock.Purpose,
		Lines:   outLines,
		Nav:     generate.FilterNavEntries(newNav, cfg.MaxNavEntries, cfg.NavStubWords),
		See:     oldBlock.See,
	}
}

func formatReport(path string, reports []ReportEntry) string {
	var b strings.Builder
	b.WriteString("Updated: ")
	b.WriteString(path)
	b.WriteString("\n")

	for _, r := range reports {
		b.WriteString(r.String())
		b.WriteString("\n")
	}

	return strings.TrimSpace(b.String())
}

const (
	frontmatterDelim = "---"
	navBlockEnd      = "-->"
)

func insertNavBlock(content string, blockText string) string {
	lines := strings.Split(content, "\n")

	blockStart := -1
	blockEnd := -1
	inFence := false
	for i, line := range lines {
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
		before := strings.Join(lines[:blockStart], "\n")
		after := ""
		if blockEnd+1 < len(lines) {
			after = strings.Join(lines[blockEnd+1:], "\n")
		}
		result := before + blockText + "\n" + after
		return cleanBlankLines(result)
	}

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
		before := strings.Join(lines[:fmEnd+1], "\n")
		after := strings.Join(lines[fmEnd+1:], "\n")
		result := before + "\n" + blockText + "\n" + after
		return cleanBlankLines(result)
	}

	result := blockText + "\n" + content
	return cleanBlankLines(result)
}

func cleanBlankLines(content string) string {
	lines := strings.Split(content, "\n")

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

	blankCount := 0
	for i := navEnd + 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "" {
			blankCount++
		} else {
			break
		}
	}

	if blankCount == 0 {
		newLines := make([]string, 0, len(lines)+1)
		newLines = append(newLines, lines[:navEnd+1]...)
		newLines = append(newLines, "")
		newLines = append(newLines, lines[navEnd+1:]...)
		return strings.Join(newLines, "\n")
	} else if blankCount > 1 {
		newLines := make([]string, 0, len(lines)-blankCount+1)
		newLines = append(newLines, lines[:navEnd+1]...)
		newLines = append(newLines, "")
		newLines = append(newLines, lines[navEnd+1+blankCount:]...)
		return strings.Join(newLines, "\n")
	}

	return content
}
