package generate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RKelln/agentmap/internal/config"
	"github.com/RKelln/agentmap/internal/navblock"
	"github.com/RKelln/agentmap/internal/parser"
)

func TestFile_WithHeadings(t *testing.T) {
	content := `# Authentication

Token lifecycle management.

## Token Exchange

OAuth2 code-for-token flow.

## Token Refresh

Silent rotation and expiry detection.
`
	dir := t.TempDir()
	path := filepath.Join(dir, "auth.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	cfg.MinLines = 10

	report, err := File(path, cfg, false)
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}

	if !strings.Contains(report, "Generated:") {
		t.Errorf("report = %q, want to contain 'Generated:'", report)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	got := string(data)
	if !strings.Contains(got, "<!-- AGENT:NAV") {
		t.Error("file should contain AGENT:NAV block")
	}
	if !strings.Contains(got, "nav[") {
		t.Error("file should contain nav header")
	}
	if !strings.Contains(got, "#Authentication") {
		t.Error("nav should contain #Authentication")
	}
	if !strings.Contains(got, "##Token Exchange") {
		t.Error("nav should contain ##Token Exchange")
	}
	if !strings.Contains(got, "##Token Refresh") {
		t.Error("nav should contain ##Token Refresh")
	}
}

func TestFile_PurposeOnly(t *testing.T) {
	content := `# Tiny File

Some helper utilities.
`
	dir := t.TempDir()
	path := filepath.Join(dir, "tiny.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	cfg.MinLines = 50

	report, err := File(path, cfg, false)
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}

	if !strings.Contains(report, "Skipped:") {
		t.Errorf("report = %q, want to contain 'Skipped:' for purpose-only", report)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	got := string(data)
	if !strings.Contains(got, "<!-- AGENT:NAV") {
		t.Error("file should contain AGENT:NAV block")
	}
	if strings.Contains(got, "nav[") {
		t.Error("purpose-only block should not contain nav header")
	}
}

func TestFile_ReplacesExistingNavBlock(t *testing.T) {
	content := `<!-- AGENT:NAV
purpose:old purpose
nav[1]{s,n,name,about}:
1,10,#Old Heading,old description
-->
# Authentication

New content here.

## Token Exchange

OAuth2 flow.
`
	dir := t.TempDir()
	path := filepath.Join(dir, "auth.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	cfg.MinLines = 10

	_, err := File(path, cfg, false)
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	got := string(data)
	if strings.Contains(got, "old purpose") {
		t.Error("old purpose should be replaced")
	}
	if strings.Contains(got, "#Old Heading") {
		t.Error("old heading should be replaced")
	}
	if !strings.Contains(got, "#Authentication") {
		t.Error("new heading should be in nav")
	}
}

func TestFile_InsertsAfterFrontmatter(t *testing.T) {
	content := `---
title: Authentication
---
# Authentication

Content here.
`
	dir := t.TempDir()
	path := filepath.Join(dir, "auth.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	cfg.MinLines = 10

	_, err := File(path, cfg, false)
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	got := string(data)
	// Nav block should appear after the closing ---
	fmEnd := strings.Index(got, "---\n")
	if fmEnd < 0 {
		t.Fatal("frontmatter closing --- not found")
	}
	navStart := strings.Index(got, "<!-- AGENT:NAV")
	if navStart < 0 {
		t.Fatal("nav block not found")
	}
	if navStart < fmEnd {
		t.Error("nav block should appear after frontmatter")
	}
}

func TestFile_DryRun(t *testing.T) {
	content := `# Authentication

Content here.

## Section

More content.

` + strings.Repeat("Extra line to exceed min_lines.\n", 10) + `
## Another Section

Even more content.
`
	dir := t.TempDir()
	path := filepath.Join(dir, "auth.md")
	originalContent := content
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	cfg.MinLines = 10

	report, err := File(path, cfg, true)
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}

	if !strings.Contains(report, "Generated:") {
		t.Errorf("report = %q, want to contain 'Generated:'", report)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if string(data) != originalContent {
		t.Error("dry-run should not modify the file")
	}
}

func TestInsertNavBlock_ReplacesExisting(t *testing.T) {
	content := `<!-- AGENT:NAV
purpose:old
nav[1]{s,n,name,about}:
1,10,#Old,old desc
-->
# Heading

Content.
`
	block := navblock.NavBlock{
		Purpose: "new purpose",
		Nav: []navblock.NavEntry{
			{Start: 5, N: 16, Name: "#Heading", About: "new desc"},
		},
	}

	got := insertNavBlock(content, navblock.RenderNavBlock(block))
	if strings.Contains(got, "old") {
		t.Error("old content should be replaced")
	}
	if !strings.Contains(got, "new purpose") {
		t.Error("new purpose should be present")
	}
	if !strings.Contains(got, "#Heading") {
		t.Error("heading should be in nav")
	}
}

func TestInsertNavBlock_InsertsAfterFrontmatter(t *testing.T) {
	content := `---
title: Test
---
# Heading

Content.
`

	got := insertNavBlock(content, navblock.RenderPurposeOnly("test purpose", 0))
	if !strings.Contains(got, "---\n") {
		t.Error("frontmatter should be preserved")
	}

	// Nav block should be after frontmatter
	fmEnd := strings.Index(got, "---\n")
	navStart := strings.Index(got, "<!-- AGENT:NAV")
	if navStart < fmEnd {
		t.Error("nav block should appear after frontmatter")
	}
}

func TestInsertNavBlock_InsertsAtLine1(t *testing.T) {
	content := `# Heading

Content here.
`

	got := insertNavBlock(content, navblock.RenderPurposeOnly("test purpose", 0))
	if !strings.HasPrefix(got, "<!-- AGENT:NAV") {
		t.Error("nav block should be at the start of the file")
	}
}

func TestBuildNavEntries_LargeFileCap(t *testing.T) {
	// §11.4: if more than 20 nav entries would be generated,
	// filter to only h1 and h2 entries.
	var sections []parser.Section
	// Build 5 h1 + 5 h2 + 15 h3 = 25 total headings
	line := 1
	for i := 0; i < 5; i++ {
		sections = append(sections, parser.Section{
			Heading: parser.Heading{Line: line, Depth: 1, Text: fmt.Sprintf("Chapter %d", i+1)},
			Start:   line,
			End:     line + 4,
		})
		line++
		sections = append(sections, parser.Section{
			Heading: parser.Heading{Line: line, Depth: 2, Text: fmt.Sprintf("Section %d.1", i+1)},
			Start:   line,
			End:     line + 2,
		})
		line++
		sections = append(sections, parser.Section{
			Heading: parser.Heading{Line: line, Depth: 3, Text: fmt.Sprintf("Sub %d.1.1", i+1)},
			Start:   line,
			End:     line + 1,
		})
		line += 2
		sections = append(sections, parser.Section{
			Heading: parser.Heading{Line: line, Depth: 3, Text: fmt.Sprintf("Sub %d.1.2", i+1)},
			Start:   line,
			End:     line + 1,
		})
		line += 2
		sections = append(sections, parser.Section{
			Heading: parser.Heading{Line: line, Depth: 3, Text: fmt.Sprintf("Sub %d.1.3", i+1)},
			Start:   line,
			End:     line + 1,
		})
		line += 2
	}

	cfg := config.Defaults()
	cfg.SubThreshold = 1
	cfg.ExpandThreshold = 1

	// Build dummy content (just newlines)
	content := strings.Repeat("\n", line+1)

	// §C3: buildNavEntries no longer caps internally; FilterNavEntries applies the cap.
	got := FilterNavEntries(buildNavEntries(sections, content, cfg), 20, 3)

	// With 25 headings (5 h1 + 5 h2 + 15 h3) > MaxNavEntries (20),
	// should filter to h1 and h2 only = 10 entries
	for _, e := range got {
		depth := 0
		for _, ch := range e.Name {
			if ch == '#' {
				depth++
			} else {
				break
			}
		}
		if depth > 2 {
			t.Errorf("entry %q: depth %d > 2 (should be filtered out when >20 entries)", e.Name, depth)
		}
	}

	// Count h1+h2 in input
	wantCount := 0
	for _, s := range sections {
		if s.Depth <= 2 {
			wantCount++
		}
	}
	if len(got) != wantCount {
		t.Errorf("len(entries) = %d, want %d (h1+h2 only)", len(got), wantCount)
	}
}

// TestFilterNavEntries_NoCapNeeded: 15 entries (mix of h1/h2/h3), all n≥3 → all returned
func TestFilterNavEntries_NoCapNeeded(t *testing.T) {
	entries := makeNavEntries([]entrySpec{
		// 5 h1, 5 h2, 5 h3 — all with n=5
		{depth: 1, n: 5},
		{depth: 1, n: 5},
		{depth: 1, n: 5},
		{depth: 1, n: 5},
		{depth: 1, n: 5},
		{depth: 2, n: 5},
		{depth: 2, n: 5},
		{depth: 2, n: 5},
		{depth: 2, n: 5},
		{depth: 2, n: 5},
		{depth: 3, n: 5},
		{depth: 3, n: 5},
		{depth: 3, n: 5},
		{depth: 3, n: 5},
		{depth: 3, n: 5},
	})
	got := FilterNavEntries(entries, 20, 3)
	if len(got) != 15 {
		t.Errorf("len(got) = %d, want 15 (no cap needed)", len(got))
	}
	// Verify document order preserved
	for i := 1; i < len(got); i++ {
		if got[i].Start <= got[i-1].Start {
			t.Errorf("entries not in document order: got[%d].Start=%d <= got[%d].Start=%d",
				i, got[i].Start, i-1, got[i-1].Start)
		}
	}
}

// TestFilterNavEntries_StubPassOnly: 22 entries (8 h1/h2 + 14 h3), 4 h3 are stubs (WordCount=1)
// stub pass removes 4 stubs → 8+10=18 ≤ 20 → no budget pass needed, 18 returned
func TestFilterNavEntries_StubPassOnly(t *testing.T) {
	var specs []entrySpec
	// 4 h1 + 4 h2
	for i := 0; i < 4; i++ {
		specs = append(specs, entrySpec{depth: 1, n: 10})
	}
	for i := 0; i < 4; i++ {
		specs = append(specs, entrySpec{depth: 2, n: 10})
	}
	// 4 h3 stubs (wordCount=1 < stubWords=3)
	for i := 0; i < 4; i++ {
		specs = append(specs, entrySpec{depth: 3, n: 1, wordCount: 1})
	}
	// 10 h3 non-stubs (wordCount=25 ≥ stubWords=3)
	for i := 0; i < 10; i++ {
		specs = append(specs, entrySpec{depth: 3, n: 5, wordCount: 25})
	}
	entries := makeNavEntries(specs)
	got := FilterNavEntries(entries, 20, 3)
	if len(got) != 18 {
		t.Errorf("len(got) = %d, want 18 (stub pass removes 4 stubs; 18 ≤ 20)", len(got))
	}
	// All returned h3 entries should have WordCount >= 3
	for _, e := range got {
		d := entryDepth(e)
		if d > 2 && e.WordCount < 3 {
			t.Errorf("stub entry %q (WordCount=%d) should have been removed", e.Name, e.WordCount)
		}
	}
	// Verify document order
	for i := 1; i < len(got); i++ {
		if got[i].Start <= got[i-1].Start {
			t.Errorf("entries not in document order at index %d", i)
		}
	}
}

// TestFilterNavEntries_BudgetPassOnly: 25 entries (8 h1/h2 + 17 h3), all WordCount≥stubWords
// stub pass removes nothing; budget=12 → keep 12 longest h3, drop 5 shortest; 20 returned
func TestFilterNavEntries_BudgetPassOnly(t *testing.T) {
	var specs []entrySpec
	// 4 h1 + 4 h2 (fixed, n=10)
	for i := 0; i < 4; i++ {
		specs = append(specs, entrySpec{depth: 1, n: 10})
	}
	for i := 0; i < 4; i++ {
		specs = append(specs, entrySpec{depth: 2, n: 10})
	}
	// 17 h3: sizes 3..19 (unique, all ≥ 3, wordCount defaults to n*5 ≥ 15 ≥ stubWords=3)
	for i := 0; i < 17; i++ {
		specs = append(specs, entrySpec{depth: 3, n: 3 + i})
	}
	entries := makeNavEntries(specs)
	got := FilterNavEntries(entries, 20, 3)
	if len(got) != 20 {
		t.Errorf("len(got) = %d, want 20", len(got))
	}
	// Shortest 5 h3 (n=3,4,5,6,7) should be gone (budget pass uses N for sorting)
	for _, e := range got {
		if entryDepth(e) == 3 && e.N < 8 {
			t.Errorf("short h3 entry %q (n=%d) should have been removed by budget pass", e.Name, e.N)
		}
	}
	// Verify document order
	for i := 1; i < len(got); i++ {
		if got[i].Start <= got[i-1].Start {
			t.Errorf("entries not in document order at index %d", i)
		}
	}
}

// TestFilterNavEntries_BothPasses: 25 entries (8 h1/h2 + 17 h3), 3 h3 are stubs (WordCount=1)
// stub pass drops 3 → 8+14=22 > 20; budget=12 → keep 12 longest of 14; 20 returned
func TestFilterNavEntries_BothPasses(t *testing.T) {
	var specs []entrySpec
	// 4 h1 + 4 h2
	for i := 0; i < 4; i++ {
		specs = append(specs, entrySpec{depth: 1, n: 10})
	}
	for i := 0; i < 4; i++ {
		specs = append(specs, entrySpec{depth: 2, n: 10})
	}
	// 3 h3 stubs (wordCount=1 < stubWords=3)
	for i := 0; i < 3; i++ {
		specs = append(specs, entrySpec{depth: 3, n: 2, wordCount: 1})
	}
	// 14 h3 non-stubs (n=5..18, wordCount defaults to n*5 ≥ 25 ≥ 3)
	for i := 0; i < 14; i++ {
		specs = append(specs, entrySpec{depth: 3, n: 5 + i})
	}
	entries := makeNavEntries(specs)
	got := FilterNavEntries(entries, 20, 3)
	if len(got) != 20 {
		t.Errorf("len(got) = %d, want 20 (stub+budget passes)", len(got))
	}
	// No stubs (WordCount<3) should remain among h3s
	for _, e := range got {
		if entryDepth(e) > 2 && e.WordCount < 3 {
			t.Errorf("stub h3 entry %q (WordCount=%d) should be removed", e.Name, e.WordCount)
		}
	}
	// Verify document order
	for i := 1; i < len(got); i++ {
		if got[i].Start <= got[i-1].Start {
			t.Errorf("entries not in document order at index %d", i)
		}
	}
}

// TestFilterNavEntries_H1H2Overrun: 25 entries all h1/h2, no candidates → accept overrun
func TestFilterNavEntries_H1H2Overrun(t *testing.T) {
	var specs []entrySpec
	for i := 0; i < 13; i++ {
		specs = append(specs, entrySpec{depth: 1, n: 10})
	}
	for i := 0; i < 12; i++ {
		specs = append(specs, entrySpec{depth: 2, n: 10})
	}
	entries := makeNavEntries(specs)
	got := FilterNavEntries(entries, 20, 3)
	if len(got) != 25 {
		t.Errorf("len(got) = %d, want 25 (h1/h2 overrun accepted)", len(got))
	}
}

// TestFilterNavEntries_BudgetExhausted: 8 h1/h2 + 15 h3 stubs (all WordCount=1)
// stub pass drops all 15 → 8 remain; 8 ≤ 20 → return 8
func TestFilterNavEntries_BudgetExhausted(t *testing.T) {
	var specs []entrySpec
	for i := 0; i < 4; i++ {
		specs = append(specs, entrySpec{depth: 1, n: 10})
	}
	for i := 0; i < 4; i++ {
		specs = append(specs, entrySpec{depth: 2, n: 10})
	}
	for i := 0; i < 15; i++ {
		specs = append(specs, entrySpec{depth: 3, n: 1, wordCount: 1})
	}
	entries := makeNavEntries(specs)
	got := FilterNavEntries(entries, 20, 3)
	if len(got) != 8 {
		t.Errorf("len(got) = %d, want 8 (all stubs removed)", len(got))
	}
	for _, e := range got {
		if entryDepth(e) > 2 {
			t.Errorf("h3 entry %q should have been removed as stub", e.Name)
		}
	}
}

// entrySpec describes a nav entry for test helpers.
type entrySpec struct {
	depth     int
	n         int
	wordCount int // 0 means use a default based on n
}

// makeNavEntries creates a slice of NavEntry from specs, assigning sequential Start lines.
func makeNavEntries(specs []entrySpec) []navblock.NavEntry {
	entries := make([]navblock.NavEntry, len(specs))
	start := 1
	for i, s := range specs {
		prefix := strings.Repeat("#", s.depth)
		wc := s.wordCount
		if wc == 0 {
			// Default: give each line ~5 words, so substantive sections have WordCount > stubWords(3)
			wc = s.n * 5
		}
		entries[i] = navblock.NavEntry{
			Start:     start,
			N:         s.n,
			Name:      prefix + fmt.Sprintf("Section%d", i+1),
			About:     "desc",
			WordCount: wc,
		}
		start += s.n
	}
	return entries
}

// entryDepth returns the heading depth of a NavEntry by counting leading '#' chars.
func entryDepth(e navblock.NavEntry) int {
	d := 0
	for _, ch := range e.Name {
		if ch == '#' {
			d++
		} else {
			break
		}
	}
	return d
}

// TestFile_LargeFileCapLineNumbers verifies that when the large-file cap fires
// (>20 entries), the kept h1/h2 entries have correct adjusted line numbers — i.e.
// applyAdjustedLines runs on the full list BEFORE the cap, so the offset is applied.
func TestFile_LargeFileCapLineNumbers(t *testing.T) {
	dir := t.TempDir()

	// Build a file that will produce >20 entries (h1 + many h2 + some h3).
	// The h3 headings should cause the cap to fire and be filtered out.
	// After generate, the h1/h2 entries should have s values > nav block size (offset applied).
	var b strings.Builder
	b.WriteString("# Overview\n\n")
	b.WriteString(strings.Repeat("Intro line.\n", 5))
	// Add 20 h2 sections (each with an h3) to force the cap (1 h1 + 20 h2 + 20 h3 = 41 entries)
	for i := 1; i <= 20; i++ {
		fmt.Fprintf(&b, "\n## Section %d\n\n", i)
		b.WriteString(strings.Repeat("Section content.\n", 3))
		fmt.Fprintf(&b, "\n### Sub %d\n\n", i)
		b.WriteString("Subsection content.\n")
	}

	content := b.String()
	path := filepath.Join(dir, "large-cap.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	cfg.MinLines = 10
	// Set expand threshold low so h3s get included before the cap, testing the cap
	cfg.SubThreshold = 1
	cfg.ExpandThreshold = 1

	_, err := File(path, cfg, false)
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	got := string(data)

	// Verify no h3 entries were written (cap fired)
	if strings.Contains(got, "###Sub") {
		t.Error("h3 entries should be filtered by large-file cap")
	}

	// Parse the nav block and verify line numbers are sane
	pr := navblock.ParseNavBlock(got)
	block := pr.Block
	navEndLine := pr.End
	found := pr.Found
	if !found {
		t.Fatal("nav block not found after generate")
	}

	// Verify all entries have s values AFTER the nav block (offset was applied)
	for _, entry := range block.Nav {
		if entry.Start <= navEndLine {
			t.Errorf("entry %q: s=%d should be after nav block end (line %d); offset not applied",
				entry.Name, entry.Start, navEndLine)
		}
	}

	// Verify entries have monotonically increasing s values
	for i := 1; i < len(block.Nav); i++ {
		if block.Nav[i].Start <= block.Nav[i-1].Start {
			t.Errorf("entry[%d] %q: s=%d should be after entry[%d] %q s=%d",
				i, block.Nav[i].Name, block.Nav[i].Start,
				i-1, block.Nav[i-1].Name, block.Nav[i-1].Start)
		}
	}

	// Verify each entry's s value actually points to a heading line in the file
	fileLines := strings.Split(got, "\n")
	for _, entry := range block.Nav {
		if entry.Start < 1 || entry.Start > len(fileLines) {
			t.Errorf("entry %q: s=%d out of range (file has %d lines)", entry.Name, entry.Start, len(fileLines))
		}
	}
}

func TestBuildNavEntries_EmptySection(t *testing.T) {
	// §11.3: empty section (heading immediately followed by another heading) → n=1
	// and is included in nav with empty about field.
	content := `## First

## Second

Some content here.
`
	sections := []parser.Section{
		{Heading: parser.Heading{Line: 1, Depth: 2, Text: "First"}, Start: 1, End: 1},
		{Heading: parser.Heading{Line: 3, Depth: 2, Text: "Second"}, Start: 3, End: 6},
	}
	cfg := config.Defaults()

	got := buildNavEntries(sections, content, cfg)

	if len(got) != 2 {
		t.Fatalf("len(entries) = %d, want 2", len(got))
	}
	// First entry: n=1, about empty
	if got[0].N != 1 {
		t.Errorf("empty section N = %d, want 1", got[0].N)
	}
	if got[0].Name != "##First" {
		t.Errorf("Name = %q, want %q", got[0].Name, "##First")
	}
	// Empty section has no extractable content so about should be empty or minimal
	// The key requirement: it IS included in nav (not skipped)
}

func TestBuildNavEntries_CommaStripping(t *testing.T) {
	// §11.2: commas must be stripped from heading names (they break CSV parsing).
	content := `# Setup, Configuration

Some setup content.
`
	sections := []parser.Section{
		{
			Heading: parser.Heading{Line: 1, Depth: 2, Text: "Setup, Configuration"},
			Start:   1,
			End:     4,
		},
	}

	cfg := config.Defaults()
	got := buildNavEntries(sections, content, cfg)

	if len(got) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(got))
	}
	if strings.Contains(got[0].Name, ",") {
		t.Errorf("entry Name = %q, must not contain a comma", got[0].Name)
	}
	want := "##Setup Configuration"
	if got[0].Name != want {
		t.Errorf("entry Name = %q, want %q", got[0].Name, want)
	}
}

func TestBuildNavEntries(t *testing.T) {
	content := `# Authentication

Token lifecycle management.

## Token Exchange

OAuth2 code-for-token flow.

## Token Refresh

Silent rotation and expiry detection.
`
	sections := []parser.Section{
		{
			Heading: parser.Heading{Line: 1, Depth: 1, Text: "Authentication"},
			Start:   1,
			End:     12,
		},
		{
			Heading: parser.Heading{Line: 5, Depth: 2, Text: "Token Exchange"},
			Start:   5,
			End:     8,
		},
		{
			Heading: parser.Heading{Line: 10, Depth: 2, Text: "Token Refresh"},
			Start:   10,
			End:     12,
		},
	}

	cfg := config.Defaults()
	cfg.MinLines = 10

	got := buildNavEntries(sections, content, cfg)

	if len(got) != 3 {
		t.Fatalf("len(entries) = %d, want 3", len(got))
	}

	// Verify structure (about fields now contain keywords, not empty)
	want := []navblock.NavEntry{
		{Start: 1, N: 12, Name: "#Authentication"},
		{Start: 5, N: 4, Name: "##Token Exchange"},
		{Start: 10, N: 3, Name: "##Token Refresh"},
	}

	for i := range want {
		if got[i].Start != want[i].Start || got[i].N != want[i].N || got[i].Name != want[i].Name {
			t.Errorf("entry[%d] = %+v, want %+v", i, got[i], want[i])
		}
		// About should now contain keywords (not empty)
		if got[i].About == "" {
			t.Errorf("entry[%d] About should contain keywords, got empty", i)
		}
	}
}

func TestFile_NoHeadings(t *testing.T) {
	content := `Just some prose text without any headings.

This is a paragraph.

Another paragraph here.
`
	dir := t.TempDir()
	path := filepath.Join(dir, "prose.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	cfg.MinLines = 10

	_, err := File(path, cfg, false)
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	got := string(data)
	if !strings.Contains(got, "<!-- AGENT:NAV") {
		t.Error("file should contain AGENT:NAV block")
	}
	if strings.Contains(got, "nav[") {
		t.Error("file with no headings should not have nav entries")
	}
}

func TestFile_ThreeLevelHeadings(t *testing.T) {
	content := `# Main

Intro.

## Section A

Content A.

` + strings.Repeat("Detailed content about section A.\n", 20) + `
### Subsection A1

Detail A1.

### Subsection A2

Detail A2.

` + strings.Repeat("More section A content.\n", 20) + `
## Section B

Content B.
`
	dir := t.TempDir()
	path := filepath.Join(dir, "three-level.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	cfg.MinLines = 10
	cfg.SubThreshold = 50
	cfg.ExpandThreshold = 150

	_, err := File(path, cfg, false)
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	got := string(data)
	if !strings.Contains(got, "<!-- AGENT:NAV") {
		t.Error("file should contain AGENT:NAV block")
	}
	if !strings.Contains(got, "#Main") {
		t.Error("nav should contain #Main")
	}
	if !strings.Contains(got, "##Section A") {
		t.Error("nav should contain ##Section A")
	}
	// Section A is medium-sized (between sub_threshold and expand_threshold),
	// so h3 children appear as > hints, not full entries
	if !strings.Contains(got, ">Subsection A1") {
		t.Error("Section A should have > hint for Subsection A1")
	}
	if !strings.Contains(got, "Subsection A2") {
		t.Error("Section A should have > hint for Subsection A2")
	}
	if !strings.Contains(got, "##Section B") {
		t.Error("nav should contain ##Section B")
	}
}

func TestFindNavBlock_IgnoresCodeFences(t *testing.T) {
	content := `# Title

Some content here.

` + "```markdown" + `
<!-- AGENT:NAV
purpose:this is inside a code fence
nav[1]{s,n,name,about}:
1,10,#Fake,fake entry
-->
` + "```" + `

More content.
`
	lines := strings.Split(content, "\n")
	start, end := findNavBlock(lines)
	if start != -1 || end != -1 {
		t.Errorf("findNavBlock() = %d, %d, want -1, -1 (nav block inside code fence should be ignored)", start, end)
	}
}

func TestFindNavBlock_FindsRealBlock(t *testing.T) {
	// Valid nav block comes before any heading
	content := `<!-- AGENT:NAV
purpose:real nav block
nav[1]{s,n,name,about}:
1,10,#Real,real entry
-->

# Title

Content.
`
	lines := strings.Split(content, "\n")
	start, end := findNavBlock(lines)
	if start == -1 {
		t.Fatal("findNavBlock() did not find real nav block")
	}
	if !strings.Contains(lines[start], "<!-- AGENT:NAV") {
		t.Errorf("findNavBlock() start line = %q, want AGENT:NAV", lines[start])
	}
	if strings.TrimSpace(lines[end]) != "-->" {
		t.Errorf("findNavBlock() end line = %q, want -->", lines[end])
	}
}

func TestFindNavBlock_CodeFenceThenRealBlock(t *testing.T) {
	// Nav block at start, code fence example later in file (should find nav block)
	content := `<!-- AGENT:NAV
purpose:real
nav[1]{s,n,name,about}:
10,20,#Real,real
-->

` + "```" + `
<!-- AGENT:NAV
purpose:fake
-->
` + "```" + `

# Title

Content.
`
	lines := strings.Split(content, "\n")
	start, end := findNavBlock(lines)
	if start == -1 {
		t.Fatal("findNavBlock() did not find real nav block")
	}
	if strings.TrimSpace(lines[start]) != "<!-- AGENT:NAV" {
		t.Errorf("findNavBlock() start line = %q, want <!-- AGENT:NAV", lines[start])
	}
	_ = end
}

func TestInsertNavBlock_SkipsCodeFenceNavBlocks(t *testing.T) {
	content := `# Title

` + "```markdown" + `
<!-- AGENT:NAV
purpose:fake
nav[1]{s,n,name,about}:
1,5,#Fake,fake
-->
` + "```" + `

Content goes here.
`
	result := insertNavBlock(content, "<!-- AGENT:NAV\npurpose:test\n-->")

	// Should NOT replace the fake block inside the code fence
	// Should insert at line 1 (no existing real block found)
	if !strings.Contains(result, "```markdown") {
		t.Error("result should still contain the code fence")
	}
	if !strings.Contains(result, "purpose:fake") {
		t.Error("result should preserve the fake block inside code fence")
	}
	// New nav block should be at the very start (line 1)
	if !strings.HasPrefix(result, "<!-- AGENT:NAV\npurpose:test\n-->") {
		t.Error("new nav block should be inserted at line 1")
	}
}

func TestGetH3Children_ImmediateOnly(t *testing.T) {
	tests := []struct {
		name      string
		sections  []parser.Section
		parentIdx int
		wantText  []string
	}{
		{
			name: "h2 with immediate h3 children only",
			sections: []parser.Section{
				{Heading: parser.Heading{Line: 1, Depth: 1, Text: "Main"}, Start: 1, End: 100},
				{Heading: parser.Heading{Line: 10, Depth: 2, Text: "Section A"}, Start: 10, End: 50},
				{Heading: parser.Heading{Line: 15, Depth: 3, Text: "Subsection A1"}, Start: 15, End: 20},
				{Heading: parser.Heading{Line: 25, Depth: 3, Text: "Subsection A2"}, Start: 25, End: 30},
				{Heading: parser.Heading{Line: 60, Depth: 2, Text: "Section B"}, Start: 60, End: 100},
			},
			parentIdx: 1,
			wantText:  []string{"Subsection A1", "Subsection A2"},
		},
		{
			name: "h2 with grandchild h3s separated by another h2",
			sections: []parser.Section{
				{Heading: parser.Heading{Line: 1, Depth: 1, Text: "Main"}, Start: 1, End: 120},
				{Heading: parser.Heading{Line: 10, Depth: 2, Text: "Section A"}, Start: 10, End: 60},
				{Heading: parser.Heading{Line: 15, Depth: 3, Text: "Child A1"}, Start: 15, End: 20},
				{Heading: parser.Heading{Line: 30, Depth: 2, Text: "Section B"}, Start: 30, End: 70},
				{Heading: parser.Heading{Line: 35, Depth: 3, Text: "Child B1"}, Start: 35, End: 40},
				{Heading: parser.Heading{Line: 50, Depth: 3, Text: "Child B2"}, Start: 50, End: 55},
				{Heading: parser.Heading{Line: 80, Depth: 2, Text: "Section C"}, Start: 80, End: 120},
			},
			parentIdx: 1,
			wantText:  []string{"Child A1"},
		},
		{
			name: "h1 parent should not get h3s from later h2 sections",
			sections: []parser.Section{
				{Heading: parser.Heading{Line: 1, Depth: 1, Text: "Main"}, Start: 1, End: 100},
				{Heading: parser.Heading{Line: 10, Depth: 2, Text: "Section A"}, Start: 10, End: 30},
				{Heading: parser.Heading{Line: 15, Depth: 3, Text: "A1"}, Start: 15, End: 20},
				{Heading: parser.Heading{Line: 40, Depth: 2, Text: "Section B"}, Start: 40, End: 100},
				{Heading: parser.Heading{Line: 50, Depth: 3, Text: "B1"}, Start: 50, End: 60},
				{Heading: parser.Heading{Line: 70, Depth: 3, Text: "B2"}, Start: 70, End: 80},
			},
			parentIdx: 0,
			wantText:  nil,
		},
		{
			name: "empty children case - no h3",
			sections: []parser.Section{
				{Heading: parser.Heading{Line: 1, Depth: 1, Text: "Main"}, Start: 1, End: 50},
				{Heading: parser.Heading{Line: 10, Depth: 2, Text: "Section A"}, Start: 10, End: 30},
				{Heading: parser.Heading{Line: 40, Depth: 2, Text: "Section B"}, Start: 40, End: 50},
			},
			parentIdx: 1,
			wantText:  nil,
		},
		{
			name: "multiple h2 siblings - each gets own h3s",
			sections: []parser.Section{
				{Heading: parser.Heading{Line: 1, Depth: 1, Text: "Main"}, Start: 1, End: 200},
				{Heading: parser.Heading{Line: 10, Depth: 2, Text: "CLI Commands"}, Start: 10, End: 80},
				{Heading: parser.Heading{Line: 15, Depth: 3, Text: "generate"}, Start: 15, End: 25},
				{Heading: parser.Heading{Line: 30, Depth: 3, Text: "update"}, Start: 30, End: 40},
				{Heading: parser.Heading{Line: 50, Depth: 2, Text: "Description Authoring"}, Start: 50, End: 120},
				{Heading: parser.Heading{Line: 60, Depth: 3, Text: "Tier 1 Keywords"}, Start: 60, End: 70},
				{Heading: parser.Heading{Line: 80, Depth: 3, Text: "LLM Generated"}, Start: 80, End: 90},
				{Heading: parser.Heading{Line: 130, Depth: 2, Text: "Parser Spec"}, Start: 130, End: 200},
				{Heading: parser.Heading{Line: 140, Depth: 3, Text: "Nav Block Parser"}, Start: 140, End: 160},
				{Heading: parser.Heading{Line: 170, Depth: 3, Text: "Nav Block Writer"}, Start: 170, End: 190},
			},
			parentIdx: 1,
			wantText:  []string{"generate", "update"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getH3Children(tt.sections, tt.parentIdx)
			if tt.wantText == nil {
				if len(got) != 0 {
					t.Errorf("getH3Children() = %v, want nil or empty", got)
				}
				return
			}
			if len(got) != len(tt.wantText) {
				t.Errorf("len(getH3Children()) = %d, want %d. Got: %v, Want: %v",
					len(got), len(tt.wantText), got, tt.wantText)
				return
			}
			for i := range tt.wantText {
				if got[i].Text != tt.wantText[i] {
					t.Errorf("getH3Children()[%d].Text = %q, want %q", i, got[i].Text, tt.wantText[i])
				}
			}
		})
	}
}

func TestBuildNavEntries_HierarchyEdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		sections       []parser.Section
		content        string
		cfg            config.Config
		wantEntryText  []string
		wantEntryAbout []string
	}{
		{
			name: "h2 with immediate h3 children only - Section 4 should not include Section 8 h3s",
			sections: []parser.Section{
				{Heading: parser.Heading{Line: 1, Depth: 1, Text: "Design Doc"}, Start: 1, End: 35},
				{Heading: parser.Heading{Line: 3, Depth: 2, Text: "CLI Commands"}, Start: 3, End: 15},
				{Heading: parser.Heading{Line: 5, Depth: 3, Text: "generate"}, Start: 5, End: 6},
				{Heading: parser.Heading{Line: 8, Depth: 3, Text: "update"}, Start: 8, End: 9},
				{Heading: parser.Heading{Line: 16, Depth: 2, Text: "Description Authoring"}, Start: 16, End: 22},
				{Heading: parser.Heading{Line: 18, Depth: 3, Text: "Keywords"}, Start: 18, End: 19},
				{Heading: parser.Heading{Line: 24, Depth: 2, Text: "Parser Spec"}, Start: 24, End: 35},
				{Heading: parser.Heading{Line: 26, Depth: 3, Text: "Heading Parser"}, Start: 26, End: 27},
			},
			content: `# Design Doc

intro

## CLI Commands

cli content
more content
more content
more content

### generate

generate content

### update

update content

## Description Authoring

desc content
more

### Keywords

keywords content

## Parser Spec

parser content
more

### Heading Parser

heading parser content
`,
			cfg: config.Config{
				SubThreshold:    5,
				ExpandThreshold: 12,
			},
			// CLI Commands (lines 3-15, 13 lines >= expand threshold 12) gets h3 children expanded
			// Description Authoring (lines 16-22, 7 lines >= sub threshold 5 but < expand) gets > hints
			// Parser Spec (lines 24-35, 12 lines >= expand threshold 12) gets h3 children expanded
			wantEntryText: []string{"#Design Doc", "##CLI Commands", "###generate", "###update", "##Description Authoring", "##Parser Spec", "###Heading Parser"},
		},
		{
			name: "small h2 sections - should not expand h3 children, use > hints instead",
			sections: []parser.Section{
				{Heading: parser.Heading{Line: 1, Depth: 1, Text: "Main"}, Start: 1, End: 22},
				{Heading: parser.Heading{Line: 3, Depth: 2, Text: "Section A"}, Start: 3, End: 15},
				{Heading: parser.Heading{Line: 9, Depth: 3, Text: "A1"}, Start: 9, End: 10},
				{Heading: parser.Heading{Line: 13, Depth: 3, Text: "A2"}, Start: 13, End: 14},
				{Heading: parser.Heading{Line: 17, Depth: 2, Text: "Section B"}, Start: 17, End: 22},
			},
			content: `# Main

content

## Section A

section a content
more content
more content
more content
more

### A1

a1 content

### A2

a2 content

## Section B

section b content
`,
			cfg: config.Config{
				SubThreshold:    8,
				ExpandThreshold: 20,
			},
			wantEntryText: []string{"#Main", "##Section A", "##Section B"},
		},
		{
			name: "h2 below subThreshold - no h3 hints",
			sections: []parser.Section{
				{Heading: parser.Heading{Line: 1, Depth: 1, Text: "Main"}, Start: 1, End: 20},
				{Heading: parser.Heading{Line: 3, Depth: 2, Text: "Section A"}, Start: 3, End: 12},
				{Heading: parser.Heading{Line: 9, Depth: 3, Text: "A1"}, Start: 9, End: 10},
			},
			content: `# Main

## Section A

section a content
more content
more

### A1

a1 content
`,
			cfg: config.Config{
				SubThreshold:    50,
				ExpandThreshold: 100,
			},
			wantEntryText: []string{"#Main", "##Section A"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildNavEntries(tt.sections, tt.content, tt.cfg)
			if len(got) != len(tt.wantEntryText) {
				t.Errorf("len(buildNavEntries()) = %d, want %d", len(got), len(tt.wantEntryText))
				for i, e := range got {
					t.Logf("  got[%d]: %s (%d-%d) about=%q", i, e.Name, e.Start, e.Start+e.N-1, e.About)
				}
				return
			}
			for i, want := range tt.wantEntryText {
				if got[i].Name != want {
					t.Errorf("entry[%d].Name = %q, want %q", i, got[i].Name, want)
				}
			}
			// Check About field if provided
			if len(tt.wantEntryAbout) > 0 {
				for i, want := range tt.wantEntryAbout {
					if got[i].About != want {
						t.Errorf("entry[%d].About = %q, want %q", i, got[i].About, want)
					}
				}
			}
		})
	}
}

// TestFilterNavEntries_StubUsesWordCount verifies that the stub pass uses WordCount,
// not N, to decide which h3 entries to drop.
func TestFilterNavEntries_StubUsesWordCount(t *testing.T) {
	// 4 h1/h2 fixed; 2 h3 with high N but low WordCount (stubs); 2 h3 with low N but high WordCount
	entries := []navblock.NavEntry{
		{Start: 1, N: 10, Name: "#Chapter", About: "", WordCount: 100},
		{Start: 11, N: 10, Name: "##Section", About: "", WordCount: 80},
		// h3 stubs: large N but few words
		{Start: 21, N: 50, Name: "###StubA", About: "", WordCount: 5},
		{Start: 71, N: 50, Name: "###StubB", About: "", WordCount: 3},
		// h3 substantive: small N but many words (dense content)
		{Start: 121, N: 3, Name: "###DenseA", About: "", WordCount: 30},
		{Start: 124, N: 3, Name: "###DenseB", About: "", WordCount: 25},
	}
	// 6 entries total ≤ 20, but force stub pass via budget (set maxEntries to 4)
	// With maxEntries=4: fixed=2, candidates=4, budget=2
	// stub pass (stubWords=20): drops StubA(5) and StubB(3), keeps DenseA(30) and DenseB(25)
	// After stub pass: 2 fixed + 2 dense = 4 ≤ 4
	got := FilterNavEntries(entries, 4, 20)
	if len(got) != 4 {
		t.Fatalf("len(got) = %d, want 4", len(got))
	}
	for _, e := range got {
		if e.Name == "###StubA" || e.Name == "###StubB" {
			t.Errorf("stub entry %q (WordCount too low) should have been removed", e.Name)
		}
	}
}

// TestBuildNavEntries_WordCount verifies that buildNavEntries sets WordCount > 0
// reflecting actual content words (not just line count).
func TestBuildNavEntries_WordCount(t *testing.T) {
	// A single-line paragraph with many words — WordCount should reflect words, not N-1.
	content := "## Section\nThis is a long sentence with ten words total here.\n"
	sections := []parser.Section{
		{Heading: parser.Heading{Line: 1, Depth: 2, Text: "Section"}, Start: 1, End: 2},
	}
	cfg := config.Defaults()
	got := buildNavEntries(sections, content, cfg)
	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	if got[0].WordCount <= 0 {
		t.Errorf("WordCount = %d, want > 0", got[0].WordCount)
	}
	// N-1 = 1 (one content line), but WordCount should be > 1 (multiple words)
	if got[0].WordCount <= 1 {
		t.Errorf("WordCount = %d, want > 1 (multi-word content line)", got[0].WordCount)
	}
}

func benchmarkMarkdown(sectionLines, childSections, childLines int) string {
	var b strings.Builder
	b.WriteString("# Benchmark\n\n")
	b.WriteString("Intro text for benchmark.\n\n")
	b.WriteString("## Section\n\n")
	b.WriteString(strings.Repeat("Section line.\n", sectionLines))
	for i := 0; i < childSections; i++ {
		fmt.Fprintf(&b, "### Child %d\n\n", i+1)
		b.WriteString(strings.Repeat("Child line.\n", childLines))
	}
	return b.String()
}

func BenchmarkFileDryRun(b *testing.B) {
	tests := []struct {
		name    string
		content string
	}{
		{name: "small", content: benchmarkMarkdown(12, 0, 0)},
		{name: "medium", content: benchmarkMarkdown(64, 2, 4)},
		{name: "large", content: benchmarkMarkdown(180, 3, 8)},
		{name: "design-clean", content: mustReadBenchmarkFixture(b, filepath.Join("..", "..", "testdata", "design-clean.md"))},
		{name: "authentication", content: mustReadBenchmarkFixture(b, filepath.Join("..", "..", "testdata", "authentication.md"))},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			dir := b.TempDir()
			path := filepath.Join(dir, "bench.md")
			if err := os.WriteFile(path, []byte(tt.content), 0o644); err != nil {
				b.Fatal(err)
			}

			cfg := config.Defaults()
			cfg.MinLines = 1
			cfg.SubThreshold = 40
			cfg.ExpandThreshold = 120

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := File(path, cfg, true); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func mustReadBenchmarkFixture(b *testing.B, path string) string {
	b.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		b.Fatalf("read benchmark fixture %s: %v", path, err)
	}
	return string(data)
}

func TestTildePrefix_PurposeAndAbout(t *testing.T) {
	content := `# Authentication

Token lifecycle management for API access.

## Token Exchange

OAuth2 code-for-token flow implementation.

## Token Refresh

Silent rotation and expiry detection logic.
`
	dir := t.TempDir()
	path := filepath.Join(dir, "auth.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	cfg.MinLines = 10

	_, err := File(path, cfg, false)
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	got := string(data)

	if !strings.Contains(got, "purpose:~") {
		t.Error("purpose line should be prefixed with ~ for keyword-generated content")
	}

	pr := navblock.ParseNavBlock(got)
	if !pr.Found {
		t.Fatal("nav block not found")
	}
	for _, entry := range pr.Block.Nav {
		if entry.About != "" && !strings.HasPrefix(entry.About, "~") {
			t.Errorf("entry %q About = %q, want ~ prefix for keyword-generated", entry.Name, entry.About)
		}
	}
}

func TestTildePrefix_NotAddedWhenEmpty(t *testing.T) {
	// Content where all tokens are stopwords or too short → no keywords extracted
	sections := []parser.Section{
		{Heading: parser.Heading{Line: 1, Depth: 1, Text: "A"}, Start: 1, End: 1},
	}
	content := "# A\n"
	cfg := config.Defaults()

	got := buildNavEntries(sections, content, cfg)

	if len(got) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(got))
	}
	// Section with no extractable keywords should have empty About, not "~"
	if got[0].About != "" {
		t.Errorf("About = %q, want empty for section with no extractable keywords", got[0].About)
	}
}

func TestTildePrefix_HintAppending(t *testing.T) {
	content := `# Main

Intro text here.

## Section A

Section A content with some meaningful words for keyword extraction.
More detailed content about authentication and token management.

### Subsection A1

Detail for A1.

### Subsection A2

Detail for A2.

## Section B

Section B content.
`
	sections := []parser.Section{
		{Heading: parser.Heading{Line: 1, Depth: 1, Text: "Main"}, Start: 1, End: 30},
		{Heading: parser.Heading{Line: 5, Depth: 2, Text: "Section A"}, Start: 5, End: 20},
		{Heading: parser.Heading{Line: 11, Depth: 3, Text: "Subsection A1"}, Start: 11, End: 13},
		{Heading: parser.Heading{Line: 15, Depth: 3, Text: "Subsection A2"}, Start: 15, End: 17},
		{Heading: parser.Heading{Line: 22, Depth: 2, Text: "Section B"}, Start: 22, End: 30},
	}

	cfg := config.Config{
		SubThreshold:    10,
		ExpandThreshold: 100,
	}

	got := buildNavEntries(sections, content, cfg)

	var sectionA *navblock.NavEntry
	for i := range got {
		if got[i].Name == "##Section A" {
			sectionA = &got[i]
			break
		}
	}
	if sectionA == nil {
		t.Fatal("Section A entry not found")
	}
	if sectionA.About == "" {
		t.Fatal("Section A About is empty")
	}
	if !strings.HasPrefix(sectionA.About, "~") {
		t.Errorf("Section A About = %q, want ~ prefix", sectionA.About)
	}
	if !strings.Contains(sectionA.About, ">") {
		t.Errorf("Section A About = %q, should contain > hints for subsections", sectionA.About)
	}
	if strings.Contains(sectionA.About, ">~") {
		t.Errorf("Section A About = %q, ~ should come before > not after", sectionA.About)
	}
}

// TestInsertNavBlock_ReplacesLargeBlock verifies that a nav block with 25 entries
// (where --> falls past line 20) is correctly replaced, not duplicated.
func TestInsertNavBlock_ReplacesLargeBlock(t *testing.T) {
	// Build a nav block with 25 entries so --> falls well past line 20.
	var sb strings.Builder
	sb.WriteString("<!-- AGENT:NAV\n")
	sb.WriteString("purpose:old purpose unique sentinel\n")
	sb.WriteString("nav[25]{s,n,name,about}:\n")
	for i := 1; i <= 25; i++ {
		fmt.Fprintf(&sb, "%d,10,##Section%d,desc\n", i*10, i)
	}
	sb.WriteString("-->\n")
	sb.WriteString("\n# Real Heading\n\nContent here.\n")
	content := sb.String()

	newBlock := navblock.NavBlock{
		Purpose: "new purpose sentinel",
		Nav: []navblock.NavEntry{
			{Start: 30, N: 5, Name: "#RealHeading", About: "~new desc"},
		},
	}
	newBlockText := navblock.RenderNavBlock(newBlock)

	got := insertNavBlock(content, newBlockText)

	// Should have exactly ONE <!-- AGENT:NAV marker
	count := strings.Count(got, "<!-- AGENT:NAV")
	if count != 1 {
		t.Errorf("got %d <!-- AGENT:NAV markers, want 1 (duplicate block inserted)", count)
	}

	// Old purpose should be gone
	if strings.Contains(got, "old purpose unique sentinel") {
		t.Error("old purpose should be replaced, not retained")
	}

	// New purpose should be present
	if !strings.Contains(got, "new purpose sentinel") {
		t.Error("new purpose should be present in result")
	}

	// Trailing content must be preserved after the block
	if !strings.Contains(got, "# Real Heading") {
		t.Error("trailing content after nav block should be preserved")
	}
}

// TestFile_IdempotentLargeNavBlock verifies that running generate.File() twice on a
// file with a large nav block (27+ entries, --> past line 20) does not produce
// duplicate <!-- AGENT:NAV markers.
func TestFile_IdempotentLargeNavBlock(t *testing.T) {
	dir := t.TempDir()

	// Build a file with a large existing nav block (27 entries → --> at line ~30)
	var sb strings.Builder
	sb.WriteString("<!-- AGENT:NAV\n")
	sb.WriteString("purpose:~large nav block test\n")
	sb.WriteString("nav[27]{s,n,name,about}:\n")
	for i := 1; i <= 27; i++ {
		fmt.Fprintf(&sb, "%d,10,##Section%d,~desc\n", 35+i*10, i)
	}
	sb.WriteString("-->\n")
	sb.WriteString("\n# Main Heading\n\n")
	// Add enough content to exceed min_lines and produce real sections
	sb.WriteString("This is the intro paragraph with content.\n\n")
	for i := 1; i <= 5; i++ {
		fmt.Fprintf(&sb, "## Section %d\n\n", i)
		for j := 0; j < 8; j++ {
			sb.WriteString("Content line for section.\n")
		}
		sb.WriteString("\n")
	}

	content := sb.String()
	path := filepath.Join(dir, "large-nav.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	cfg.MinLines = 10

	// First run
	_, err := File(path, cfg, false)
	if err != nil {
		t.Fatalf("File() first run error = %v", err)
	}

	data1, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	count1 := strings.Count(string(data1), "<!-- AGENT:NAV")
	if count1 != 1 {
		t.Errorf("after first run: got %d <!-- AGENT:NAV markers, want 1", count1)
	}

	// Second run — should be idempotent (no new nav block added)
	_, err = File(path, cfg, false)
	if err != nil {
		t.Fatalf("File() second run error = %v", err)
	}

	data2, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	count2 := strings.Count(string(data2), "<!-- AGENT:NAV")
	if count2 != 1 {
		t.Errorf("after second run: got %d <!-- AGENT:NAV markers, want 1 (idempotency broken)", count2)
	}
}

// TestInsertNavBlock_ReplacesLargeBlockWithFrontmatter verifies that when a file has
// frontmatter (before != "") AND a large nav block (blockEnd > 20), the replacement
// path correctly inserts a "\n" separator and does not fuse the frontmatter to the block.
func TestInsertNavBlock_ReplacesLargeBlockWithFrontmatter(t *testing.T) {
	// Build frontmatter + large nav block content
	var sb strings.Builder
	sb.WriteString("---\ntitle: Test File\n---\n")
	sb.WriteString("<!-- AGENT:NAV\n")
	sb.WriteString("purpose:old purpose fm sentinel\n")
	sb.WriteString("nav[25]{s,n,name,about}:\n")
	for i := 1; i <= 25; i++ {
		fmt.Fprintf(&sb, "%d,10,##Section%d,desc\n", i*10, i)
	}
	sb.WriteString("-->\n")
	sb.WriteString("\n# Real Heading\n\nContent here.\n")
	content := sb.String()

	newBlock := navblock.NavBlock{
		Purpose: "new purpose fm sentinel",
		Nav: []navblock.NavEntry{
			{Start: 35, N: 5, Name: "#RealHeading", About: "~new desc"},
		},
	}
	newBlockText := navblock.RenderNavBlock(newBlock)

	got := insertNavBlock(content, newBlockText)

	// Exactly one nav block marker
	count := strings.Count(got, "<!-- AGENT:NAV")
	if count != 1 {
		t.Errorf("got %d <!-- AGENT:NAV markers, want 1", count)
	}

	// Frontmatter still present and well-formed (not fused to nav block)
	if !strings.Contains(got, "---\ntitle: Test File\n---\n") {
		t.Error("frontmatter should be preserved and not fused to the nav block")
	}

	// Old purpose gone, new purpose present
	if strings.Contains(got, "old purpose fm sentinel") {
		t.Error("old purpose should be replaced")
	}
	if !strings.Contains(got, "new purpose fm sentinel") {
		t.Error("new purpose should be present")
	}

	// Trailing content preserved
	if !strings.Contains(got, "# Real Heading") {
		t.Error("trailing content after nav block should be preserved")
	}
}

// TestFile_IdempotentDesignClean verifies that running generate.File() on
// design-clean.md (which has <!-- AGENT:NAV only inside code fences) produces
// exactly one real nav block at the top, and a second run doesn't add another.
func TestFile_IdempotentDesignClean(t *testing.T) {
	// Count <!-- AGENT:NAV occurrences in the original (all inside code fences, no real block).
	// Assert the known baseline so test failures are self-documenting.
	origData, err := os.ReadFile(filepath.Join("..", "..", "testdata", "design-clean.md"))
	if err != nil {
		t.Fatalf("read design-clean.md: %v", err)
	}
	origCount := strings.Count(string(origData), "<!-- AGENT:NAV")
	const wantOrigCount = 11 // all inside code-fence examples; none is a real top-level block
	if origCount != wantOrigCount {
		t.Fatalf("design-clean.md has %d <!-- AGENT:NAV markers, expected %d; update this test if the file changed",
			origCount, wantOrigCount)
	}

	dir := t.TempDir()
	outPath := filepath.Join(dir, "design-clean-gen.md")

	// Copy to temp file
	if err := os.WriteFile(outPath, origData, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	cfg.MinLines = 10

	// First run: generates the real nav block
	_, err = File(outPath, cfg, false)
	if err != nil {
		t.Fatalf("File() first run error = %v", err)
	}

	data1, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}

	// Should be exactly origCount + 1 (the new real block at top)
	count1 := strings.Count(string(data1), "<!-- AGENT:NAV")
	wantCount := origCount + 1
	if count1 != wantCount {
		t.Errorf("after first run: got %d <!-- AGENT:NAV markers, want %d (orig %d + 1 real block)",
			count1, wantCount, origCount)
	}

	// Second run: should be idempotent — count must not increase
	_, err = File(outPath, cfg, false)
	if err != nil {
		t.Fatalf("File() second run error = %v", err)
	}

	data2, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}

	count2 := strings.Count(string(data2), "<!-- AGENT:NAV")
	if count2 != wantCount {
		t.Errorf("after second run: got %d <!-- AGENT:NAV markers, want %d (idempotency broken)",
			count2, wantCount)
	}
}
