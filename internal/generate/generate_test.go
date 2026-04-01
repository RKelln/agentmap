package generate

import (
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
nav[1]{s,e,name,about}:
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
nav[1]{s,e,name,about}:
1,10,#Old,old desc
-->
# Heading

Content.
`
	block := navblock.NavBlock{
		Purpose: "new purpose",
		Nav: []navblock.NavEntry{
			{Start: 5, End: 20, Name: "#Heading", About: "new desc"},
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
		{Start: 1, End: 12, Name: "#Authentication"},
		{Start: 5, End: 8, Name: "##Token Exchange"},
		{Start: 10, End: 12, Name: "##Token Refresh"},
	}

	for i := range want {
		if got[i].Start != want[i].Start || got[i].End != want[i].End || got[i].Name != want[i].Name {
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
nav[1]{s,e,name,about}:
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
nav[1]{s,e,name,about}:
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
nav[1]{s,e,name,about}:
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
nav[1]{s,e,name,about}:
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
