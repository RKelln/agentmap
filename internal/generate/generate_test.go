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

const (
	testAutoAbout = "~keywords"
	testSection2  = "##Section2"
	testParent    = "##Parent"
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

	report, err := File(path, cfg, false, true)
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

	report, err := File(path, cfg, false, true)
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
	cfg.MinLines = 5 // content is 7 lines; keep well below to test nav replacement, not threshold

	_, err := File(path, cfg, false, true)
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

	_, err := File(path, cfg, false, true)
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

func TestFile_FrontmatterHeadingLineNumbers(t *testing.T) {
	// Regression: when a file has YAML frontmatter followed by a blank line,
	// the first generate produced heading line numbers that were off by +1.
	// cleanBlankLines merges the existing blank (after ---) with the separator,
	// so the separator doesn't add a net new line.
	content := "---\ntitle: Test\n---\n\n# Heading One\n\nSome content here.\n\n## Section Two\n\nMore content.\n" +
		strings.Repeat("Extra line to exceed min_lines.\n", 45)

	dir := t.TempDir()
	path := filepath.Join(dir, "fm.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	cfg.MinLines = 10

	_, err := File(path, cfg, false, true)
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)

	// Parse the generated nav block and verify line numbers match actual positions.
	pr := navblock.ParseNavBlock(got)
	if !pr.Found {
		t.Fatal("nav block not found in output")
	}

	// Find actual heading positions in the final content.
	headings, _ := parser.ParseHeadings(got, cfg.MaxDepth)
	if len(headings) == 0 {
		t.Fatal("no headings found in output")
	}

	// Build a map of heading name -> actual line from the final content.
	actualLines := make(map[string]int)
	for _, h := range headings {
		actualLines[navblock.NormalizeHeading(h.Text)] = h.Line
	}

	// Check each nav entry's Start matches the actual heading position.
	for _, entry := range pr.Block.Nav {
		name := navblock.NormalizeHeading(entry.Name)
		actual, ok := actualLines[name]
		if !ok {
			t.Errorf("nav entry %q not found in headings", entry.Name)
			continue
		}
		if entry.Start != actual {
			t.Errorf("nav entry %q: Start=%d, but heading is actually at line %d (off by %d)",
				entry.Name, entry.Start, actual, entry.Start-actual)
		}
	}
}

func TestFile_NoFrontmatterLeadingBlankLineNumbers(t *testing.T) {
	// Regression: files starting with a blank line before the first heading
	// also had the separator absorbed by the existing blank.
	content := "\n# Heading\n\nContent here.\n\n## Section\n\nMore content.\n" +
		strings.Repeat("Extra line.\n", 45)

	dir := t.TempDir()
	path := filepath.Join(dir, "leading-blank.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	cfg.MinLines = 10

	_, err := File(path, cfg, false, true)
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)

	pr := navblock.ParseNavBlock(got)
	if !pr.Found {
		t.Fatal("nav block not found in output")
	}

	headings, _ := parser.ParseHeadings(got, cfg.MaxDepth)
	actualLines := make(map[string]int)
	for _, h := range headings {
		actualLines[navblock.NormalizeHeading(h.Text)] = h.Line
	}

	for _, entry := range pr.Block.Nav {
		name := navblock.NormalizeHeading(entry.Name)
		actual, ok := actualLines[name]
		if !ok {
			t.Errorf("nav entry %q not found in headings", entry.Name)
			continue
		}
		if entry.Start != actual {
			t.Errorf("nav entry %q: Start=%d, but heading is actually at line %d (off by %d)",
				entry.Name, entry.Start, actual, entry.Start-actual)
		}
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

	report, err := File(path, cfg, true, true)
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
	// PruneNavEntries prunes the deepest entries by parent size.
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
	cfg.ExpandThreshold = 1000 // force droppable (parentN < subThreshold=1 → false; all are droppable at N=2)

	// Build dummy content (just newlines)
	content := strings.Repeat("\n", line+1)

	// buildNavEntries returns all 25 entries; PruneNavEntries applies the cap.
	got := PruneNavEntries(buildNavEntries(sections, content), cfg.SubThreshold, cfg.ExpandThreshold, 20)

	// With 25 headings (5 h1 + 5 h2 + 15 h3) > MaxNavEntries (20),
	// h3s should be pruned until within budget (10 h1+h2 remain or 20 if some h3s survive).
	// With sub=1 and expand=1000, parentN=2 is >= sub(1) but < expand(1000) → hintable.
	// Pruning should occur until len(got) <= 20.
	if len(got) > 20 {
		t.Errorf("len(entries) = %d, want ≤ 20 after pruning", len(got))
	}
}

// TestPruneNavEntries_UnderBudget: 18 entries under cap → all returned unchanged.
func TestPruneNavEntries_UnderBudget(t *testing.T) {
	// 1 h1 + 2 h2 (N=100 each) + 15 h3, cap=20
	var specs []entrySpec
	specs = append(specs, entrySpec{depth: 1, n: 200})
	specs = append(specs, entrySpec{depth: 2, n: 100})
	specs = append(specs, entrySpec{depth: 2, n: 100})
	for i := 0; i < 15; i++ {
		specs = append(specs, entrySpec{depth: 3, n: 5})
	}
	entries := makeNavEntries(specs)
	// Add about fields to h2 entries to ensure they're not modified.
	entries[1].About = testAutoAbout
	entries[2].About = testAutoAbout

	got := PruneNavEntries(entries, 50, 150, 20)
	if len(got) != 18 {
		t.Errorf("len(got) = %d, want 18 (under budget, no pruning)", len(got))
	}
	// h2 About fields should be unchanged (no hints appended)
	for _, e := range got {
		if e.Name == testSection2 || e.Name == "##Section3" {
			if strings.Contains(e.About, ">") {
				t.Errorf("entry %q About = %q, should not have > hints (under budget)", e.Name, e.About)
			}
		}
	}
}

// TestPruneNavEntries_DropsSmallParent: h3 with parent N < sub_threshold → dropped, no hint.
func TestPruneNavEntries_DropsSmallParent(t *testing.T) {
	// 1 h1 + 1 h2 (N=30 < sub=50) + 10 h3, cap=5
	specs := []entrySpec{
		{depth: 1, n: 200},
		{depth: 2, n: 30}, // small parent
	}
	for i := 0; i < 10; i++ {
		specs = append(specs, entrySpec{depth: 3, n: 5})
	}
	entries := makeNavEntries(specs)
	// h2 About to confirm no hint is added.
	entries[1].About = "~original"

	got := PruneNavEntries(entries, 50, 150, 5)
	if len(got) > 5 {
		t.Errorf("len(got) = %d, want ≤ 5 after pruning", len(got))
	}
	// h2 About should remain unchanged (droppable, no hint)
	for _, e := range got {
		if e.Name == testSection2 {
			if e.About != "~original" {
				t.Errorf("h2 About = %q, want %q (no hint for droppable h3)", e.About, "~original")
			}
		}
	}
}

// TestPruneNavEntries_HintsForMediumParent: h3 with medium parent → hint appended to parent About.
func TestPruneNavEntries_HintsForMediumParent(t *testing.T) {
	// 1 h1 + 1 h2 (N=80, medium) + 10 h3, cap=5
	specs := []entrySpec{
		{depth: 1, n: 200},
		{depth: 2, n: 80}, // medium: sub(50) ≤ 80 < expand(150)
	}
	for i := 0; i < 10; i++ {
		specs = append(specs, entrySpec{depth: 3, n: 5})
	}
	entries := makeNavEntries(specs)
	entries[1].About = testAutoAbout

	got := PruneNavEntries(entries, 50, 150, 5)
	if len(got) > 5 {
		t.Errorf("len(got) = %d, want ≤ 5 after pruning", len(got))
	}
	// h2 About should contain > hints
	for _, e := range got {
		if e.Name == testSection2 {
			if !strings.Contains(e.About, ">") {
				t.Errorf("medium parent About = %q, should contain > hints", e.About)
			}
		}
	}
}

// TestPruneNavEntries_MultipleHintsAccumulate: 3 h3s under same medium h2, all pruned → hints accumulate.
func TestPruneNavEntries_MultipleHintsAccumulate(t *testing.T) {
	entries := []navblock.NavEntry{
		{Start: 1, N: 200, Name: "#Top"},
		{Start: 10, N: 80, Name: testParent, About: testAutoAbout},
		{Start: 20, N: 5, Name: "###h3a"},
		{Start: 30, N: 5, Name: "###h3b"},
		{Start: 40, N: 5, Name: "###h3c"},
	}
	// cap=2 forces all 3 h3s to be pruned
	got := PruneNavEntries(entries, 50, 150, 2)
	if len(got) != 2 {
		t.Errorf("len(got) = %d, want 2", len(got))
	}
	var parent *navblock.NavEntry
	for i := range got {
		if got[i].Name == testParent {
			parent = &got[i]
		}
	}
	if parent == nil {
		t.Fatal(testParent + " entry not found after pruning")
	}
	// All 3 h3 hints should be present
	if !strings.Contains(parent.About, "h3a") {
		t.Errorf("parent About = %q, want h3a hint", parent.About)
	}
	if !strings.Contains(parent.About, "h3b") {
		t.Errorf("parent About = %q, want h3b hint", parent.About)
	}
	if !strings.Contains(parent.About, "h3c") {
		t.Errorf("parent About = %q, want h3c hint", parent.About)
	}
	// Multiple hints separated by ;
	if !strings.Contains(parent.About, ";") {
		t.Errorf("parent About = %q, want ; between accumulated hints", parent.About)
	}
}

// TestPruneNavEntries_KeepsLargeParent: h3 with parent N >= expand_threshold → kept; overrun accepted.
func TestPruneNavEntries_KeepsLargeParent(t *testing.T) {
	entries := []navblock.NavEntry{
		{Start: 1, N: 200, Name: "#Top"},
		{Start: 10, N: 200, Name: "##BigParent", About: "~big"}, // N=200 >= expand=150
		{Start: 20, N: 5, Name: "###Child1"},
		{Start: 30, N: 5, Name: "###Child2"},
		{Start: 40, N: 5, Name: "###Child3"},
	}
	// cap=2 would normally force pruning, but h3s are unkillable
	got := PruneNavEntries(entries, 50, 150, 2)
	// Overrun accepted: all 5 entries remain
	if len(got) != 5 {
		t.Errorf("len(got) = %d, want 5 (unkillable h3s cause overrun acceptance)", len(got))
	}
}

// TestPruneNavEntries_NoParentDrops: orphan h3 (no preceding h2) → dropped, no panic.
func TestPruneNavEntries_NoParentDrops(t *testing.T) {
	entries := []navblock.NavEntry{
		{Start: 1, N: 5, Name: "###OrphanH3"}, // no parent (depth 2) above it
		{Start: 10, N: 5, Name: "###OrphanH3b"},
		{Start: 20, N: 5, Name: "###OrphanH3c"},
		{Start: 30, N: 5, Name: "###OrphanH3d"},
		{Start: 40, N: 5, Name: "###OrphanH3e"},
	}
	// Should not panic; orphans have parentN=0 (droppable)
	got := PruneNavEntries(entries, 50, 150, 2)
	if len(got) > 2 {
		t.Errorf("len(got) = %d, want ≤ 2 (orphans dropped)", len(got))
	}
}

// TestPruneNavEntries_ShallowOverrunAccepted: 25 h2 entries (no h3), cap=20 → all 25 returned.
func TestPruneNavEntries_ShallowOverrunAccepted(t *testing.T) {
	var specs []entrySpec
	for i := 0; i < 25; i++ {
		specs = append(specs, entrySpec{depth: 2, n: 10})
	}
	entries := makeNavEntries(specs)
	got := PruneNavEntries(entries, 50, 150, 20)
	if len(got) != 25 {
		t.Errorf("len(got) = %d, want 25 (shallow overrun accepted)", len(got))
	}
}

// TestPruneNavEntries_MixedOutcomes: mix of large/medium/small h2 parents with h3s, over budget.
func TestPruneNavEntries_MixedOutcomes(t *testing.T) {
	// 1 h1 + large h2 (N=200) + 5 h3 + medium h2 (N=80) + 5 h3 + small h2 (N=30) + 5 h3 = 18 total
	// cap=10: 5 h3s under large parent are unkillable; medium → hints; small → dropped
	entries := []navblock.NavEntry{
		{Start: 1, N: 250, Name: "#Top"},
		// Large parent section: h3s unkillable
		{Start: 10, N: 200, Name: "##LargeParent"},
		{Start: 20, N: 5, Name: "###LargeChild1"},
		{Start: 30, N: 5, Name: "###LargeChild2"},
		{Start: 40, N: 5, Name: "###LargeChild3"},
		{Start: 50, N: 5, Name: "###LargeChild4"},
		{Start: 60, N: 5, Name: "###LargeChild5"},
		// Medium parent: h3s become hints
		{Start: 220, N: 80, Name: "##MediumParent", About: "~medium"},
		{Start: 230, N: 5, Name: "###MediumChild1"},
		{Start: 240, N: 5, Name: "###MediumChild2"},
		{Start: 250, N: 5, Name: "###MediumChild3"},
		{Start: 260, N: 5, Name: "###MediumChild4"},
		{Start: 270, N: 5, Name: "###MediumChild5"},
		// Small parent: h3s dropped
		{Start: 310, N: 30, Name: "##SmallParent", About: "~small"},
		{Start: 320, N: 5, Name: "###SmallChild1"},
		{Start: 330, N: 5, Name: "###SmallChild2"},
		{Start: 340, N: 5, Name: "###SmallChild3"},
		{Start: 350, N: 5, Name: "###SmallChild4"},
	}
	// cap=10: forces pruning of h3s.
	// Large h3s: unkillable (parent N=200 >= expand=150).
	// Medium h3s: hintable (50 <= parent N=80 < 150).
	// Small h3s: droppable (parent N=30 < sub=50).
	// Start: 18 entries, need to reduce to 10 → remove 8.
	// First round: maxD=3. Small h3s (parentN=30) droppable, medium h3s (parentN=80) hintable, large h3s unkillable.
	// Sort by N asc (all N=5): droppable first, then hintable.
	// Need to remove 8: 4 small (droppable) + 4 medium (hintable from 5 available), leaving 10 = 1+2+5+1+1 = 10
	got := PruneNavEntries(entries, 50, 150, 10)
	if len(got) > 10 {
		t.Errorf("len(got) = %d, want ≤ 10 after pruning", len(got))
	}
	// Large parent h3s should all be kept (unkillable)
	largeKept := 0
	for _, e := range got {
		if strings.HasPrefix(e.Name, "###LargeChild") {
			largeKept++
		}
	}
	if largeKept != 5 {
		t.Errorf("LargeChild h3s kept = %d, want 5 (unkillable)", largeKept)
	}
	// Small parent h3s should all be dropped
	for _, e := range got {
		if strings.HasPrefix(e.Name, "###SmallChild") {
			t.Errorf("SmallChild h3 %q should have been dropped", e.Name)
		}
	}
	// Medium parent should have > hints
	for _, e := range got {
		if e.Name == "##MediumParent" {
			if !strings.Contains(e.About, ">") {
				t.Errorf("MediumParent About = %q, should have > hints", e.About)
			}
		}
	}
}

// TestBuildNavEntries_AllSectionsIncluded: all sections appear as entries, no threshold filtering.
func TestBuildNavEntries_AllSectionsIncluded(t *testing.T) {
	content := `# Doc
intro

## SmallSection

brief

### SmallChild

child content

## MediumSection

` + strings.Repeat("medium content line\n", 10) + `
### MediumChild

child content here

## LargeSection

` + strings.Repeat("large content line\n", 50) + `
### LargeChild

child content here
`
	sections := []parser.Section{
		{Heading: parser.Heading{Line: 1, Depth: 1, Text: "Doc"}, Start: 1, End: 100},
		{Heading: parser.Heading{Line: 3, Depth: 2, Text: "SmallSection"}, Start: 3, End: 10},
		{Heading: parser.Heading{Line: 7, Depth: 3, Text: "SmallChild"}, Start: 7, End: 9},
		{Heading: parser.Heading{Line: 12, Depth: 2, Text: "MediumSection"}, Start: 12, End: 30},
		{Heading: parser.Heading{Line: 25, Depth: 3, Text: "MediumChild"}, Start: 25, End: 28},
		{Heading: parser.Heading{Line: 32, Depth: 2, Text: "LargeSection"}, Start: 32, End: 90},
		{Heading: parser.Heading{Line: 85, Depth: 3, Text: "LargeChild"}, Start: 85, End: 88},
	}

	got := buildNavEntries(sections, content)

	// ALL 7 sections should appear as full nav entries with no threshold branching.
	if len(got) != 7 {
		t.Errorf("len(got) = %d, want 7 (all sections included)", len(got))
		for i, e := range got {
			t.Logf("  got[%d]: %s", i, e.Name)
		}
	}

	// No hints should appear in About fields (hints only from PruneNavEntries, not buildNavEntries)
	for _, e := range got {
		if strings.Contains(e.About, ">") {
			t.Errorf("entry %q About = %q: should not have > hints from buildNavEntries", e.Name, e.About)
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
	// Each h2 parent has N≈7 lines; set SubThreshold=20 so all h2 parents are droppable
	// (parentN < subThreshold → droppable → h3 children dropped with no hints).
	cfg.SubThreshold = 20
	cfg.ExpandThreshold = 200

	_, err := File(path, cfg, false, true)
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

	got := buildNavEntries(sections, content)

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

	got := buildNavEntries(sections, content)

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

	got := buildNavEntries(sections, content)

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

	_, err := File(path, cfg, false, true)
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

	_, err := File(path, cfg, false, true)
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
	// With 5 entries total (well under MaxNavEntries=20), no pruning occurs —
	// all h3 sections appear as full nav entries.
	if !strings.Contains(got, "###Subsection A1") {
		t.Error("nav should contain ###Subsection A1 as full entry (under budget)")
	}
	if !strings.Contains(got, "###Subsection A2") {
		t.Error("nav should contain ###Subsection A2 as full entry (under budget)")
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

func TestBuildNavEntries_HierarchyEdgeCases(t *testing.T) {
	// New behavior: buildNavEntries always includes every section; no threshold branching.
	// All sections become full entries; no hints are added at build time.
	tests := []struct {
		name          string
		sections      []parser.Section
		content       string
		cfg           config.Config
		wantEntryText []string
	}{
		{
			name: "all sections included regardless of size",
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
			cfg: config.Config{SubThreshold: 5, ExpandThreshold: 12},
			// ALL 8 sections — including h3s — are full entries; no threshold filtering in buildNavEntries.
			wantEntryText: []string{"#Design Doc", "##CLI Commands", "###generate", "###update", "##Description Authoring", "###Keywords", "##Parser Spec", "###Heading Parser"},
		},
		{
			name: "small h2 sections still include h3 children",
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
			cfg: config.Config{SubThreshold: 8, ExpandThreshold: 20},
			// All 5 sections are full entries now; pruning only via PruneNavEntries.
			wantEntryText: []string{"#Main", "##Section A", "###A1", "###A2", "##Section B"},
		},
		{
			name: "h2 below subThreshold still includes h3",
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
			cfg: config.Config{SubThreshold: 50, ExpandThreshold: 100},
			// All 3 sections are full entries; no threshold-based skipping.
			wantEntryText: []string{"#Main", "##Section A", "###A1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildNavEntries(tt.sections, tt.content)
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
			// No About field should contain > hints (hints come from PruneNavEntries, not buildNavEntries)
			for _, e := range got {
				if strings.Contains(e.About, ">") {
					t.Errorf("entry %q About = %q: buildNavEntries must not add > hints", e.Name, e.About)
				}
			}
		})
	}
}

// TestBuildNavEntries_LargeH2NoH3Children verifies that a large h2 section with no
// h3 children does not panic (regression for index-out-of-range [-1] bug).
func TestBuildNavEntries_LargeH2NoH3Children(t *testing.T) {
	// Build a content string where the h2 section is larger than ExpandThreshold
	// but contains zero h3 children.
	var sb strings.Builder
	sb.WriteString("# Title\n\n")
	sb.WriteString("## Big Section\n\n")
	for i := 0; i < 200; i++ {
		sb.WriteString("paragraph line here with some words to exceed the expand threshold\n")
	}
	content := sb.String()

	parsedHeadings, _ := parser.ParseHeadings(content, 3)
	sections := parser.ComputeSections(parsedHeadings, len(strings.Split(content, "\n")))

	// Must not panic.
	entries := buildNavEntries(sections, content)

	// Sanity: we should get at least 2 entries (#Title and ##Big-Section).
	if len(entries) < 2 {
		t.Errorf("expected at least 2 entries, got %d", len(entries))
	}
}

// TestPruneNavEntries_DropsShortestFirst verifies that PruneNavEntries prunes
// by N ascending (shortest sections first) when over budget.
func TestPruneNavEntries_DropsShortestFirst(t *testing.T) {
	// h2 parent N=40 < subThreshold=50 → droppable. 3 h3 children with varying N.
	// With 5 entries and maxEntries=3, need to drop 2 → shortest h3s first.
	entries := []navblock.NavEntry{
		{Start: 1, N: 200, Name: "#Chapter", About: "", WordCount: 100},
		{Start: 11, N: 40, Name: "##Section", About: "", WordCount: 50}, // N=40 < sub=50 → droppable parent
		{Start: 51, N: 10, Name: "###Short1", About: "", WordCount: 80},
		{Start: 61, N: 20, Name: "###Medium1", About: "", WordCount: 80},
		{Start: 81, N: 50, Name: "###Large1", About: "", WordCount: 80},
	}
	// 5 entries; cap to 3 → drop 2 h3s (Short1 N=10, Medium1 N=20 are cheapest)
	got := PruneNavEntries(entries, 50, 150, 3)
	if len(got) != 3 {
		t.Fatalf("len(got) = %d, want 3", len(got))
	}
	for _, e := range got {
		if e.Name == "###Short1" || e.Name == "###Medium1" {
			t.Errorf("shortest entry %q should have been removed first", e.Name)
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
	got := buildNavEntries(sections, content)
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
				if _, err := File(path, cfg, true, true); err != nil {
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

	_, err := File(path, cfg, false, true)
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

	got := buildNavEntries(sections, content)

	if len(got) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(got))
	}
	// Section with no extractable keywords should have empty About, not "~"
	if got[0].About != "" {
		t.Errorf("About = %q, want empty for section with no extractable keywords", got[0].About)
	}
}

func TestTildePrefix_HintAppending(t *testing.T) {
	// When PruneNavEntries collapses hintable h3 entries into the parent's About,
	// the hint is appended to whatever About already contains. If the parent About
	// starts with ~ (auto-generated prefix), the > hints must come after ~ — not
	// interspersed or prefixed.
	entries := []navblock.NavEntry{
		{Start: 1, N: 30, Name: "#Main", About: "~intro text", WordCount: 10},
		// Section A: N=16 is between sub(10) and expand(100) → hintable parent
		{Start: 5, N: 16, Name: "##Section A", About: "~authentication token management", WordCount: 20},
		{Start: 11, N: 2, Name: "###Subsection A1", About: "~detail a1", WordCount: 5},
		{Start: 15, N: 2, Name: "###Subsection A2", About: "~detail a2", WordCount: 5},
		{Start: 22, N: 8, Name: "##Section B", About: "~section b content", WordCount: 5},
	}

	// 5 entries, cap to 3 → must drop 2 h3 entries; Section A N=16 ≥ subThreshold(10) → hintable
	got := PruneNavEntries(entries, 10, 100, 3)

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
	_, err := File(path, cfg, false, true)
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
	_, err = File(path, cfg, false, true)
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
	// Count <!-- AGENT:NAV occurrences in the original.
	// 1 real nav block at top + code fence examples = 12 total
	// Assert the known baseline so test failures are self-documenting.
	origData, err := os.ReadFile(filepath.Join("..", "..", "testdata", "design-clean.md"))
	if err != nil {
		t.Fatalf("read design-clean.md: %v", err)
	}
	origCount := strings.Count(string(origData), "<!-- AGENT:NAV")
	const wantOrigCount = 12 // 1 real nav block + code fence examples
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
	_, err = File(outPath, cfg, false, true)
	if err != nil {
		t.Fatalf("File() first run error = %v", err)
	}

	data1, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}

	// Should be exactly origCount (generate replaces existing nav block, doesn't add)
	count1 := strings.Count(string(data1), "<!-- AGENT:NAV")
	wantCount := origCount // generate replaces existing block, count stays same
	if count1 != wantCount {
		t.Errorf("after first run: got %d <!-- AGENT:NAV markers, want %d (orig %d)",
			count1, wantCount, origCount)
	}

	// Second run: should be idempotent — count must be unchanged
	_, err = File(outPath, cfg, false, true)
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

// TestFile_LineNumbersCorrectOnFirstGenerate verifies that on first generate (no
// existing nav block) every nav entry's s value matches the actual line number of
// that heading in the resulting file. This catches the off-by-1 bug caused by the
// blank separator line that cleanBlankLines inserts after the nav block.
func TestFile_LineNumbersCorrectOnFirstGenerate(t *testing.T) {
	// Build a file where h1 is at line 1 with enough content for a full nav block.
	var b strings.Builder
	b.WriteString("# Database Guide\n\n")
	b.WriteString(strings.Repeat("Database content line.\n", 10))
	for i := 1; i <= 15; i++ {
		fmt.Fprintf(&b, "\n## Section %d\n\n", i)
		b.WriteString(strings.Repeat("Section content here.\n", 5))
	}

	content := b.String()
	dir := t.TempDir()
	path := filepath.Join(dir, "database.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	cfg.MinLines = 10

	_, err := File(path, cfg, false, true)
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)

	pr := navblock.ParseNavBlock(got)
	if !pr.Found {
		t.Fatal("nav block not found after generate")
	}

	fileLines := strings.Split(got, "\n")

	// Verify each nav entry's s value matches the actual line number of that heading.
	for _, entry := range pr.Block.Nav {
		headingText := strings.TrimLeft(entry.Name, "#")
		depth := len(entry.Name) - len(headingText)
		prefix := strings.Repeat("#", depth)
		needle := prefix + " " + headingText

		actualLine := -1
		for i, line := range fileLines {
			if line == needle {
				actualLine = i + 1 // 1-indexed
				break
			}
		}

		if actualLine < 0 {
			t.Errorf("entry %q: heading %q not found in file", entry.Name, needle)
			continue
		}

		if entry.Start != actualLine {
			t.Errorf("entry %q: s=%d but heading is at line %d (off by %d)",
				entry.Name, entry.Start, actualLine, actualLine-entry.Start)
		}
	}
}

// TestFile_LineNumbersCorrectWithH3Children verifies that nav entry s values are
// correct even with multi-level headings (h1/h2/h3). buildNavEntries now includes
// all sections; PruneNavEntries may prune h3s when over budget. Regardless,
// every retained entry's s must point to the correct heading line.
func TestFile_LineNumbersCorrectWithH3Children(t *testing.T) {
	// Build a file with h1 and several h2 sections, each with h3 subsections.
	// 1 h1 + 5 h2 + 10 h3 = 16 entries, under MaxNavEntries=20, so all are kept.
	var b strings.Builder
	b.WriteString("# Top Level\n\n")
	b.WriteString(strings.Repeat("Intro content.\n", 5))
	for i := 1; i <= 5; i++ {
		fmt.Fprintf(&b, "\n## Chapter %d\n\n", i)
		b.WriteString(strings.Repeat("Chapter body.\n", 4))
		fmt.Fprintf(&b, "\n### Section %d.1\n\n", i)
		b.WriteString(strings.Repeat("Sub content.\n", 3))
		fmt.Fprintf(&b, "\n### Section %d.2\n\n", i)
		b.WriteString(strings.Repeat("Sub content.\n", 3))
	}

	content := b.String()
	dir := t.TempDir()
	path := filepath.Join(dir, "chapters.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	cfg.MinLines = 10

	_, err := File(path, cfg, false, true)
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)

	pr := navblock.ParseNavBlock(got)
	if !pr.Found {
		t.Fatal("nav block not found after generate")
	}

	fileLines := strings.Split(got, "\n")

	// Every nav entry's s must point to the actual heading line in the file.
	for _, entry := range pr.Block.Nav {
		headingText := strings.TrimLeft(entry.Name, "#")
		depth := len(entry.Name) - len(headingText)
		needle := strings.Repeat("#", depth) + " " + headingText

		actualLine := -1
		for i, line := range fileLines {
			if line == needle {
				actualLine = i + 1
				break
			}
		}
		if actualLine < 0 {
			t.Errorf("entry %q: heading %q not found in file", entry.Name, needle)
			continue
		}
		if entry.Start != actualLine {
			t.Errorf("entry %q: s=%d but heading is at line %d (off by %d)",
				entry.Name, entry.Start, actualLine, actualLine-entry.Start)
		}
	}
}

// TestFile_MinLinesBoundary verifies the MinLines threshold is evaluated against
// the correct line count. The bug: len(strings.Split(content, "\n")) overcounts by 1
// for files ending with \n, so a file with exactly MinLines-1 content lines was
// incorrectly treated as meeting the threshold and got a full nav block instead of
// purpose-only. With strings.Count the boundary is exact.
func TestFile_MinLinesBoundary(t *testing.T) {
	cfg := config.Defaults()
	// MinLines default is 50. Build a file with exactly MinLines-1 newlines (49 lines,
	// trailing \n) that has a heading — it must get purpose-only treatment.
	threshold := cfg.MinLines
	targetNewlines := threshold - 1 // 49 newlines → 49 "lines" per wc -l

	var b strings.Builder
	b.WriteString("# Boundary Heading\n")
	// Fill remaining lines so the file has exactly targetNewlines newlines total.
	// We already wrote 1 newline above, so write targetNewlines-1 more lines.
	for i := 1; i < targetNewlines; i++ {
		b.WriteString("filler line content here\n")
	}
	content := b.String()

	if got := strings.Count(content, "\n"); got != targetNewlines {
		t.Fatalf("setup error: content has %d newlines, want %d", got, targetNewlines)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "boundary.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	report, err := File(path, cfg, false, true)
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}

	// File has fewer lines than MinLines → must be purpose-only, not a full nav block.
	if !strings.Contains(report, "Skipped:") {
		t.Errorf("report = %q; file with %d lines (threshold %d) should be purpose-only (Skipped:)",
			report, targetNewlines, threshold)
	}

	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), "nav[") {
		t.Errorf("file with %d lines should not have nav[] entries (threshold %d); off-by-1 in totalLines",
			targetNewlines, threshold)
	}
}

// TestPruneNavEntries_Idempotent verifies that calling PruneNavEntries twice on the
// same base entries (with Start values shifted on the second call to simulate an
// offset adjustment) produces structurally identical pruning decisions: the same
// entries survive, the same become hints, and the hint text on parent entries is
// identical. This also confirms PruneNavEntries does not mutate the original slice.
func TestPruneNavEntries_Idempotent(t *testing.T) {
	// Base entries: 1 h1 + 1 h2 (medium parent, N=80) + 5 h3.
	// With cap=2, all 5 h3s must be pruned to reach the budget; medium parent gets hints.
	base := []navblock.NavEntry{
		{Start: 1, N: 200, Name: "#Top", About: "~top"},
		{Start: 10, N: 80, Name: testParent, About: "~parent desc"},
		{Start: 20, N: 5, Name: "###Child1", About: "~c1"},
		{Start: 30, N: 5, Name: "###Child2", About: "~c2"},
		{Start: 40, N: 5, Name: "###Child3", About: "~c3"},
		{Start: 50, N: 5, Name: "###Child4", About: "~c4"},
		{Start: 55, N: 5, Name: "###Child5", About: "~c5"},
	}

	// First call — baseline result.
	got1 := PruneNavEntries(base, 50, 150, 2)

	// Verify the original slice was not mutated.
	if base[1].About != "~parent desc" {
		t.Errorf("original base[1].About = %q after first PruneNavEntries call; should be unchanged (no mutation)", base[1].About)
	}

	// Build shifted copy: same entries with Start += 5 to simulate offset adjustment.
	shifted := make([]navblock.NavEntry, len(base))
	copy(shifted, base)
	for i := range shifted {
		shifted[i].Start += 5
	}

	// Second call on shifted entries.
	got2 := PruneNavEntries(shifted, 50, 150, 2)

	// Structural check: both results must have the same length.
	if len(got1) != len(got2) {
		t.Fatalf("first call len=%d, second call len=%d; pruning decisions should be structurally identical", len(got1), len(got2))
	}

	// Both must have exactly 2 entries (h1 + h2; all 5 h3s pruned to reach cap=2).
	if len(got1) != 2 {
		t.Errorf("len(got1) = %d, want 2 (h1 + h2 after all h3s pruned to cap=2)", len(got1))
	}

	// Locate the parent entry in each result and compare hint text.
	var parent1, parent2 *navblock.NavEntry
	for i := range got1 {
		if got1[i].Name == testParent {
			parent1 = &got1[i]
		}
	}
	for i := range got2 {
		if got2[i].Name == testParent {
			parent2 = &got2[i]
		}
	}

	if parent1 == nil {
		t.Fatal(testParent + " not found in first result")
	}
	if parent2 == nil {
		t.Fatal(testParent + " not found in second result")
	}

	// The hint text (everything after the > separator) should be identical across both calls.
	if parent1.About != parent2.About {
		t.Errorf("hint text diverged between calls:\n  call1 About = %q\n  call2 About = %q", parent1.About, parent2.About)
	}

	// Both should contain > hints.
	if !strings.Contains(parent1.About, ">") {
		t.Errorf("parent1.About = %q, should contain > hints", parent1.About)
	}
}

// TestFile_LinesFieldIsTotalLines verifies that generate stores the total file
// line count in lines:N — matching what an editor displays — not content lines
// (which would exclude the nav block itself).
func TestFile_SkipsExistingNavBlockByDefault(t *testing.T) {
	content := `# Auth Guide

Token lifecycle management.

## Token Exchange

OAuth2 code-for-token flow.
`
	dir := t.TempDir()
	path := filepath.Join(dir, "auth.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	cfg.MinLines = 5

	// First generate: no existing block, should succeed.
	report, err := File(path, cfg, false, false)
	if err != nil {
		t.Fatalf("first File() error = %v", err)
	}
	if !strings.Contains(report, "Generated:") {
		t.Errorf("first call report = %q, want 'Generated:'", report)
	}

	// Overwrite the purpose with a hand-written value.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	original := string(data)
	modified := strings.Replace(original, "~", "hand-written", 1)
	if err := os.WriteFile(path, []byte(modified), 0o644); err != nil {
		t.Fatal(err)
	}

	// Second generate without --force: should skip.
	report, err = File(path, cfg, false, false)
	if err != nil {
		t.Fatalf("second File() error = %v", err)
	}
	if report != SkippedExisting {
		t.Errorf("second call report = %q, want %q", report, SkippedExisting)
	}

	// Confirm hand-written value was not overwritten.
	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "hand-written") {
		t.Error("hand-written description was overwritten by generate without --force")
	}
}

func TestFile_ForceOverwritesExistingNavBlock(t *testing.T) {
	content := `# Auth Guide

Token lifecycle management.

## Token Exchange

OAuth2 code-for-token flow.
`
	dir := t.TempDir()
	path := filepath.Join(dir, "auth.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	cfg.MinLines = 5

	// First generate.
	if _, err := File(path, cfg, false, false); err != nil {
		t.Fatalf("first File() error = %v", err)
	}

	// Second generate with force=true: should overwrite.
	report, err := File(path, cfg, false, true)
	if err != nil {
		t.Fatalf("force File() error = %v", err)
	}
	if !strings.Contains(report, "Generated:") {
		t.Errorf("force call report = %q, want 'Generated:'", report)
	}
}

func TestFile_LinesFieldIsTotalLines(t *testing.T) {
	content := `# Auth Guide

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
	cfg.MinLines = 5

	if _, err := File(path, cfg, false, true); err != nil {
		t.Fatalf("File() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	pr := navblock.ParseNavBlock(string(data))
	if !pr.Found {
		t.Fatal("nav block not found after generate")
	}

	totalLines := strings.Count(string(data), "\n")
	if pr.Block.Lines != totalLines {
		t.Errorf("lines:N = %d, want total file lines %d (nav block has %d lines)",
			pr.Block.Lines, totalLines, pr.End-pr.Start+1)
	}
}
