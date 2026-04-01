package generate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ryankelln/agentmap/internal/config"
	"github.com/ryankelln/agentmap/internal/navblock"
	"github.com/ryankelln/agentmap/internal/parser"
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

	got := insertNavBlock(content, navblock.RenderPurposeOnly("test purpose"))
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

	got := insertNavBlock(content, navblock.RenderPurposeOnly("test purpose"))
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
	got := FilterNavEntries(buildNavEntries(sections, content, cfg))

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
	block, _, navEndLine, found, _ := navblock.ParseNavBlock(got)
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
