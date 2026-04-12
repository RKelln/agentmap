package update

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RKelln/agentmap/internal/config"
	"github.com/RKelln/agentmap/internal/navblock"
	"github.com/RKelln/agentmap/internal/parser"
)

func writeTempFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestBuildUpdatedBlock_CommaStripping(t *testing.T) {
	// §11.2: commas must be stripped from heading names (they break CSV parsing).
	// update preserves comma-stripped names (via buildUpdatedBlock).
	oldBlock := navblock.NavBlock{
		Purpose: "test",
		Nav: []navblock.NavEntry{
			{Start: 12, N: 5, Name: "##Setup Configuration", About: "existing desc"},
		},
	}

	sections := []parser.Section{
		{
			Heading: parser.Heading{Line: 14, Depth: 2, Text: "Setup, Configuration"},
			Start:   14,
			End:     18,
		},
	}

	updated := buildUpdatedBlock(oldBlock, sections, nil, nil, config.Defaults(), 0, 0)
	if len(updated.Nav) != 1 {
		t.Fatalf("nav count = %d, want 1", len(updated.Nav))
	}
	if strings.Contains(updated.Nav[0].Name, ",") {
		t.Errorf("Name = %q must not contain a comma", updated.Nav[0].Name)
	}
	want := "##Setup Configuration"
	if updated.Nav[0].Name != want {
		t.Errorf("Name = %q, want %q", updated.Nav[0].Name, want)
	}
	if updated.Nav[0].About != "existing desc" {
		t.Errorf("About = %q, want %q", updated.Nav[0].About, "existing desc")
	}
}

func TestFile_Shifted(t *testing.T) {
	dir := t.TempDir()

	oldContent := `<!-- AGENT:NAV
purpose:authentication documentation
nav[3]{s,n,name,about}:
12,39,#Authentication,token lifecycle management
14,17,##Token Exchange,OAuth2 code-for-token flow
32,19,##Token Refresh,silent rotation and expiry
-->
# Authentication

Token lifecycle management.

## Token Exchange

OAuth2 code-for-token flow.

## Token Refresh

Silent rotation and sliding-window expiry.
`

	path := writeTempFile(t, dir, "auth.md", oldContent)

	cfg := config.Defaults()
	cfg.MaxDepth = 3

	report, err := File(path, cfg, false, false)
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}

	if report == noChanges {
		t.Error("expected changes report, got noChanges")
	}

	if !strings.Contains(report, "shifted: ##Token Exchange") {
		t.Errorf("report should contain shifted: ##Token Exchange, got: %s", report)
	}
	if !strings.Contains(report, "shifted: ##Token Refresh") {
		t.Errorf("report should contain shifted: ##Token Refresh, got: %s", report)
	}

	data, _ := os.ReadFile(path)
	pr := navblock.ParseNavBlock(string(data))
	newBlock, found := pr.Block, pr.Found
	if !found {
		t.Fatal("file should still have nav block after update")
	}

	for _, entry := range newBlock.Nav {
		if entry.Name == "##Token Exchange" {
			if entry.About != "OAuth2 code-for-token flow" {
				t.Errorf("Token Exchange description should be preserved, got %q", entry.About)
			}
		}
		if entry.Name == "##Token Refresh" {
			if entry.About != "silent rotation and expiry" {
				t.Errorf("Token Refresh description should be preserved, got %q", entry.About)
			}
		}
	}
}

func TestFile_NewHeading(t *testing.T) {
	dir := t.TempDir()

	oldContent := `<!-- AGENT:NAV
purpose:authentication documentation
nav[2]{s,n,name,about}:
12,14,#Authentication,token lifecycle management
14,7,##Token Exchange,OAuth2 code-for-token flow
-->
# Authentication

Token lifecycle management.

## Token Exchange

OAuth2 code-for-token flow.

## Token Revocation

Logout and forced invalidation.
`

	path := writeTempFile(t, dir, "auth.md", oldContent)

	cfg := config.Defaults()
	cfg.MaxDepth = 3

	report, err := File(path, cfg, false, false)
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}

	if report == noChanges {
		t.Error("expected changes report, got noChanges")
	}

	if !strings.Contains(report, "new: ##Token Revocation") {
		t.Errorf("report should contain new: ##Token Revocation, got: %s", report)
	}
	if !strings.Contains(report, "no description") {
		t.Errorf("report should indicate no description, got: %s", report)
	}

	data, _ := os.ReadFile(path)
	pr := navblock.ParseNavBlock(string(data))
	newBlock, found := pr.Block, pr.Found
	if !found {
		t.Fatal("file should have nav block after update")
	}

	var revocationEntry navblock.NavEntry
	for _, entry := range newBlock.Nav {
		if entry.Name == "##Token Revocation" {
			revocationEntry = entry
			break
		}
	}

	if revocationEntry.Name == "" {
		t.Error("##Token Revocation should be added to nav block")
	}
	if revocationEntry.About != "" {
		t.Errorf("new heading should have empty description, got %q", revocationEntry.About)
	}
}

func TestFile_DeletedHeading(t *testing.T) {
	dir := t.TempDir()

	oldContent := `<!-- AGENT:NAV
purpose:authentication documentation
nav[3]{s,n,name,about}:
12,39,#Authentication,token lifecycle management
14,17,##Token Exchange,OAuth2 code-for-token flow
32,19,##Token Refresh,silent rotation and expiry
-->
# Authentication

Token lifecycle management.

## Token Exchange

OAuth2 code-for-token flow.
`

	path := writeTempFile(t, dir, "auth.md", oldContent)

	cfg := config.Defaults()
	cfg.MaxDepth = 3

	report, err := File(path, cfg, false, false)
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}

	if report == noChanges {
		t.Error("expected changes report, got noChanges")
	}

	if !strings.Contains(report, "deleted: ##Token Refresh") {
		t.Errorf("report should contain deleted: ##Token Refresh, got: %s", report)
	}

	data, _ := os.ReadFile(path)
	pr := navblock.ParseNavBlock(string(data))
	newBlock, found := pr.Block, pr.Found
	if !found {
		t.Fatal("file should have nav block after update")
	}

	if len(newBlock.Nav) != 2 {
		t.Errorf("nav should have 2 entries after deletion, got %d", len(newBlock.Nav))
	}

	for _, entry := range newBlock.Nav {
		if entry.Name == "##Token Refresh" {
			t.Error("##Token Refresh should be removed from nav block")
		}
	}
}

func TestFile_NoNavBlock(t *testing.T) {
	dir := t.TempDir()

	content := `# Authentication

Token lifecycle management.

## Token Exchange

OAuth2 flow.
`

	path := writeTempFile(t, dir, "auth.md", content)

	cfg := config.Defaults()
	cfg.MinLines = 5

	report, err := File(path, cfg, false, false)
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}

	// update delegates to generate for files with no nav block.
	if report == noChanges {
		t.Error("update returned noChanges for a nav-less file; expected delegation to generate")
	}
	if !strings.Contains(report, "Generated:") && !strings.Contains(report, "Skipped:") {
		t.Errorf("expected generate report, got: %s", report)
	}

	// File should now have a nav block.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "<!-- AGENT:NAV") {
		t.Error("file has no AGENT:NAV block after update delegated to generate")
	}
}

func TestFile_PurposeOnly_FileUnderMinLines(t *testing.T) {
	dir := t.TempDir()

	content := `# Helpers

Some helper utilities.
`

	path := writeTempFile(t, dir, "helpers.md", content)

	cfg := config.Defaults()

	report, err := File(path, cfg, false, false)
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}

	// update delegates to generate for files with no nav block.
	// For a tiny file, generate writes a purpose-only block.
	if report == noChanges {
		t.Error("update returned noChanges for a nav-less file; expected delegation to generate")
	}
	if !strings.Contains(report, "Skipped:") && !strings.Contains(report, "Generated:") {
		t.Errorf("expected generate report (purpose-only skipped), got: %s", report)
	}
}

func TestFile_DryRun(t *testing.T) {
	dir := t.TempDir()

	oldContent := `<!-- AGENT:NAV
purpose:old purpose
nav[1]{s,n,name,about}:
12,14,#Old Section,old description
-->
# Old Section

Content here.
`

	path := writeTempFile(t, dir, "test.md", oldContent)

	cfg := config.Defaults()
	cfg.MaxDepth = 3

	_, err := File(path, cfg, true, false)
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}

	data, _ := os.ReadFile(path)
	if string(data) != oldContent {
		t.Error("dry-run should not modify file")
	}
}

func TestFile_UpdatePreservesDescriptions(t *testing.T) {
	dir := t.TempDir()

	oldContent := `<!-- AGENT:NAV
purpose:authentication docs
nav[2]{s,n,name,about}:
12,29,#Auth,token lifecycle
14,12,##Exchange,OAuth2 code flow
-->
# Auth

Token lifecycle.

## Exchange

OAuth2 flow.

## Refresh

Silent rotation.
`

	path := writeTempFile(t, dir, "auth.md", oldContent)

	cfg := config.Defaults()
	cfg.MaxDepth = 3

	_, err := File(path, cfg, false, false)
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}

	data, _ := os.ReadFile(path)
	pr2 := navblock.ParseNavBlock(string(data))
	block, found := pr2.Block, pr2.Found
	if !found {
		t.Fatal("nav block should exist after update")
	}

	for _, entry := range block.Nav {
		if entry.Name == "#Auth" && entry.About != "token lifecycle" {
			t.Errorf("Auth description should be preserved, got %q", entry.About)
		}
		if entry.Name == "##Exchange" && entry.About != "OAuth2 code flow" {
			t.Errorf("Exchange description should be preserved, got %q", entry.About)
		}
		if entry.Name == "##Refresh" && entry.About != "" {
			t.Errorf("new Refresh entry should have empty description, got %q", entry.About)
		}
	}
}

func TestFile_NoChanges(t *testing.T) {
	dir := t.TempDir()

	oldContent := `<!-- AGENT:NAV
purpose:docs
nav[4]{s,n,name,about}:
9,15,#Document,main doc
13,4,##Section A,section A content
17,4,##Section B,section B content
21,3,##Section C,section C content
-->
# Document

Main content.

## Section A

Content A.

## Section B

Content B.

## Section C

Content C.
`

	path := writeTempFile(t, dir, "doc.md", oldContent)

	cfg := config.Defaults()
	cfg.MaxDepth = 3

	report, err := File(path, cfg, false, false)
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}

	if report != noChanges {
		t.Errorf("expected noChanges when nav block matches document, got: %s", report)
	}
}

func TestFile_DuplicateHeadings(t *testing.T) {
	dir := t.TempDir()

	oldContent := `<!-- AGENT:NAV
purpose:guide
nav[2]{s,n,name,about}:
12,19,#Guide,guide doc
14,2,##Examples,first examples
-->
# Guide

Guide content.

## Examples

First examples section.

## Examples

Second examples section.
`

	path := writeTempFile(t, dir, "guide.md", oldContent)

	cfg := config.Defaults()
	cfg.MaxDepth = 3

	report, err := File(path, cfg, false, false)
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}

	if report == noChanges {
		t.Error("expected changes")
	}

	data, _ := os.ReadFile(path)
	pr2 := navblock.ParseNavBlock(string(data))
	block, found := pr2.Block, pr2.Found
	if !found {
		t.Fatal("nav block should exist")
	}

	examplesCount := 0
	for _, entry := range block.Nav {
		if entry.Name == "##Examples" {
			examplesCount++
		}
	}

	if examplesCount != 2 {
		t.Errorf("expected 2 ##Examples entries, got %d", examplesCount)
	}
}

func TestFile_QuietMode(t *testing.T) {
	dir := t.TempDir()

	oldContent := `<!-- AGENT:NAV
purpose:old
nav[1]{s,n,name,about}:
12,9,#Section,old desc
-->
# Section

Content.
`

	path := writeTempFile(t, dir, "test.md", oldContent)

	cfg := config.Defaults()
	cfg.MaxDepth = 3

	report, err := File(path, cfg, false, true)
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}

	if report != noChanges {
		t.Errorf("quiet mode should return noChanges, got: %s", report)
	}

	data, _ := os.ReadFile(path)
	pr2 := navblock.ParseNavBlock(string(data))
	block, found := pr2.Block, pr2.Found
	if !found {
		t.Fatal("nav block should be updated")
	}

	if len(block.Nav) != 1 || block.Nav[0].Name != "#Section" {
		t.Error("nav block should be updated in quiet mode")
	}
}

func TestFile_EmptyNavEntries(t *testing.T) {
	dir := t.TempDir()

	oldContent := `<!-- AGENT:NAV
purpose:docs
nav[2]{s,n,name,about}:
12,14,#Doc,description
14,12,##Section,
-->
# Doc

Description here.

## Section

Content.
`

	path := writeTempFile(t, dir, "doc.md", oldContent)

	cfg := config.Defaults()
	cfg.MaxDepth = 3

	report, err := File(path, cfg, false, false)
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}

	if report == noChanges {
		t.Error("expected changes")
	}

	data, _ := os.ReadFile(path)
	pr2 := navblock.ParseNavBlock(string(data))
	block, found := pr2.Block, pr2.Found
	if !found {
		t.Fatal("nav block should exist")
	}

	for _, entry := range block.Nav {
		if entry.Name == "##Section" && entry.About != "" {
			t.Errorf("empty description should remain empty, got %q", entry.About)
		}
	}
}

func TestFile_RenameHeading(t *testing.T) {
	dir := t.TempDir()

	oldContent := `<!-- AGENT:NAV
purpose:docs
nav[2]{s,n,name,about}:
7,8,#Doc,description
11,4,##OldName,old description
-->
# Doc

Description here.

## OldName

Old content.

## NewName

New content.
`

	path := writeTempFile(t, dir, "doc.md", oldContent)

	cfg := config.Defaults()
	cfg.MaxDepth = 3

	report, err := File(path, cfg, false, false)
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}

	if report == noChanges {
		t.Error("expected changes after renaming heading")
	}

	if !strings.Contains(report, "new: ##NewName") {
		t.Errorf("report should contain new: ##NewName, got: %s", report)
	}

	data, _ := os.ReadFile(path)
	pr2 := navblock.ParseNavBlock(string(data))
	block, found := pr2.Block, pr2.Found
	if !found {
		t.Fatal("nav block should exist")
	}

	var newEntry navblock.NavEntry
	for _, entry := range block.Nav {
		if entry.Name == "##NewName" {
			newEntry = entry
			break
		}
	}

	if newEntry.Name == "" {
		t.Error("##NewName should be added to nav block")
	}
	if newEntry.About != "" {
		t.Errorf("renamed heading should have empty description, got %q", newEntry.About)
	}
}

func TestFile_PreservesSeeBlock(t *testing.T) {
	dir := t.TempDir()

	oldContent := `<!-- AGENT:NAV
purpose:docs
nav[1]{s,n,name,about}:
12,6,#Doc,description
see[2]{path,why}:
other.md,related file
config.md,configuration
-->
# Doc

Description here.
`

	path := writeTempFile(t, dir, "doc.md", oldContent)

	cfg := config.Defaults()
	cfg.MaxDepth = 3

	_, err := File(path, cfg, false, false)
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}

	data, _ := os.ReadFile(path)
	pr2 := navblock.ParseNavBlock(string(data))
	block, found := pr2.Block, pr2.Found
	if !found {
		t.Fatal("nav block should exist")
	}

	if len(block.Nav) != 1 {
		t.Errorf("nav should have 1 entry, got %d", len(block.Nav))
	}

	if len(block.Nav) != 1 {
		t.Errorf("nav should have 1 entry after update, got %d", len(block.Nav))
	}
	if block.Nav[0].Name != "#Doc" {
		t.Errorf("nav[0].Name should be #Doc, got %q", block.Nav[0].Name)
	}

	content := string(data)
	if !strings.Contains(content, "other.md") {
		t.Error("see block should preserve other.md")
	}
	if !strings.Contains(content, "config.md") {
		t.Error("see block should preserve config.md")
	}
}

func TestFile_PreservesPurpose(t *testing.T) {
	dir := t.TempDir()

	oldContent := `<!-- AGENT:NAV
purpose:important docs
nav[1]{s,n,name,about}:
12,19,#Doc,description
-->
# Doc

Description here.
`

	path := writeTempFile(t, dir, "doc.md", oldContent)

	cfg := config.Defaults()
	cfg.MaxDepth = 3

	_, err := File(path, cfg, false, false)
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}

	data, _ := os.ReadFile(path)
	pr2 := navblock.ParseNavBlock(string(data))
	block, found := pr2.Block, pr2.Found
	if !found {
		t.Fatal("nav block should exist")
	}

	if block.Purpose != "important docs" {
		t.Errorf("purpose should be preserved, got %q", block.Purpose)
	}
}

func TestFile_ShiftedWithDescription(t *testing.T) {
	dir := t.TempDir()

	oldContent := `<!-- AGENT:NAV
purpose:docs
nav[2]{s,n,name,about}:
12,9,#Doc,description
14,7,##Section,section content
-->
# Doc

Description.

## Section

Content.
`

	path := writeTempFile(t, dir, "doc.md", oldContent)

	cfg := config.Defaults()
	cfg.MaxDepth = 3

	report, err := File(path, cfg, false, false)
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}

	if report == noChanges {
		t.Error("expected changes")
	}

	if !strings.Contains(report, "shifted: ##Section") {
		t.Errorf("report should contain shifted, got: %s", report)
	}

	data, _ := os.ReadFile(path)
	pr2 := navblock.ParseNavBlock(string(data))
	block, found := pr2.Block, pr2.Found
	if !found {
		t.Fatal("nav block should exist")
	}

	for _, entry := range block.Nav {
		if entry.Name == "##Section" {
			if entry.About != "section content" {
				t.Errorf("description should be preserved, got %q", entry.About)
			}
		}
	}
}

func TestUpdate_PreservesTildePrefix(t *testing.T) {
	dir := t.TempDir()

	oldContent := `<!-- AGENT:NAV
purpose:~token OAuth2 authentication flow
nav[2]{s,n,name,about}:
12,39,#Authentication,~OAuth2 PKCE redirect token lifecycle
14,17,##Token Exchange,~code exchange redirect
-->
# Authentication

Token lifecycle management.

## Token Exchange

OAuth2 code-for-token flow.
`

	path := writeTempFile(t, dir, "auth.md", oldContent)

	cfg := config.Defaults()
	cfg.MaxDepth = 3

	report, err := File(path, cfg, false, false)
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}

	if report == noChanges {
		t.Error("expected changes report, got noChanges")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	pr := navblock.ParseNavBlock(string(data))
	block, found := pr.Block, pr.Found
	if !found {
		t.Fatal("file should still have nav block after update")
	}

	if !strings.HasPrefix(block.Purpose, "~") {
		t.Errorf("Purpose should preserve ~ prefix, got %q", block.Purpose)
	}

	for _, entry := range block.Nav {
		if !strings.HasPrefix(entry.About, "~") {
			t.Errorf("About for %q should preserve ~ prefix, got %q", entry.Name, entry.About)
		}
	}
}

// TestFile_TotalLinesOffByOne verifies that update uses strings.Count(content,"\n") for
// totalLines, not len(strings.Split(...)) which overcounts by 1 for POSIX files ending
// with \n. Regression for Bug 4 in update (parallel to the same fix in generate).
//
// The last section's End should equal totalLines (= wc -l). Previously End was
// totalLines+1 (= wc -l + 1), causing a spurious "shifted" report on first update.
func TestFile_TotalLinesOffByOne(t *testing.T) {
	dir := t.TempDir()

	// Nav block (5 lines) + blank separator (1 line) + body (3 lines) = 9 total \n chars.
	// strings.Count("\n") = 9 = totalLines. #Doc at line 7, End=9, N=3.
	// With len(strings.Split) bug: totalLines=10, End=10, N=4 → shifted.
	oldContent := "" +
		"<!-- AGENT:NAV\n" + // line 1
		"purpose:test\n" + // line 2
		"nav[1]{s,n,name,about}:\n" + // line 3
		"7,3,#Doc,doc\n" + // line 4 — #Doc at line 7, End=9, N=3 ✓
		"-->\n" + // line 5
		"\n" + // line 6 blank separator
		"# Doc\n" + // line 7
		"\n" + // line 8
		"Content.\n" // line 9 (trailing \n → strings.Count=9, totalLines=9)

	path := writeTempFile(t, dir, "doc.md", oldContent)

	cfg := config.Defaults()
	cfg.MaxDepth = 3

	report, err := File(path, cfg, false, false)
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}

	// The nav block accurately reflects totalLines=9 (strings.Count), so no changes expected.
	// If totalLines were overcounted as 10 (len(strings.Split)), #Doc.End would be 10 ≠ 9 → shifted.
	if report != noChanges {
		t.Errorf("expected noChanges (correct totalLines), got: %s", report)
	}
}

// TestFile_NewH3IncludedAndPruned verifies that update now includes new h3 entries
// for all h2 sections (including those below ExpandThreshold), then delegates to
// PruneNavEntries to enforce the max_nav_entries budget. This matches generate's
// budget-first behavior: buildNavEntries includes everything; PruneNavEntries prunes.
func TestFile_NewH3IncludedAndPruned(t *testing.T) {
	dir := t.TempDir()

	// The h2 "##Small Section" is ~38 lines — below ExpandThreshold (150).
	// The old nav block only has the h2 entry (no h3 children).
	// Under the new model, update should include the new h3 entries and call
	// PruneNavEntries; since total entries is well under MaxNavEntries(20),
	// all entries survive and the report lists them as "new".
	var sb strings.Builder
	sb.WriteString("<!-- AGENT:NAV\npurpose:test\nnav[2]{s,n,name,about}:\n")
	sb.WriteString("9,42,#Doc,doc\n")
	sb.WriteString("13,38,##Small Section,small\n")
	sb.WriteString("-->\n")
	sb.WriteString("# Doc\n\nContent.\n\n")
	sb.WriteString("## Small Section\n\n")
	// Add h3 children inside the h2 (total h2 section = 38 lines < 150 ExpandThreshold)
	for i := 0; i < 5; i++ {
		sb.WriteString("### Child " + string(rune('A'+i)) + "\n\nChild content.\n\n")
	}
	// Pad to reach 50 lines for the h2 section end
	sb.WriteString("End of small section.\n")
	// Add a closing h1-level marker that ends the section
	sb.WriteString("# AnotherDoc\n\nMore content.\n")
	content := sb.String()
	path := writeTempFile(t, dir, "doc.md", content)

	cfg := config.Defaults()
	cfg.MaxDepth = 3
	cfg.ExpandThreshold = 150
	cfg.MaxNavEntries = 20 // well above 7 total sections → no pruning expected

	report, err := File(path, cfg, false, false)
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}

	// New h3 entries should now be included and reported (under budget, all survive).
	// The report should contain "new: ###Child" entries — that is the correct behavior.
	if !strings.Contains(report, "new: ###Child") {
		t.Errorf("update should now include new h3 entries (budget-first model); report:\n%s", report)
	}

	// Verify the written nav block actually contains the h3 entries.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	pr := navblock.ParseNavBlock(string(data))
	if !pr.Found {
		t.Fatal("nav block not found after update")
	}
	h3Count := 0
	for _, e := range pr.Block.Nav {
		if strings.HasPrefix(e.Name, "###") {
			h3Count++
		}
	}
	if h3Count == 0 {
		t.Errorf("nav block should contain h3 entries after update (budget-first model); entries: %v", pr.Block.Nav)
	}
}

// TestFile_PurposeOnlyBelowMinLines verifies that update does not add nav entries
// to a file whose existing nav block is purpose-only (no entries) and whose
// content line count is below MinLines. Generate intentionally writes purpose-only
// blocks for such files (purpose + optional see, no sections list); update must
// not contradict that by adding entries — the file is small enough to read whole.
func TestFile_PurposeOnlyBelowMinLines(t *testing.T) {
	dir := t.TempDir()

	cfg := config.Defaults() // MinLines=50 by default

	// Sub-case 1: lines:N is already correct → noChanges (nothing to do).
	// Content has 10 total lines (strings.Count("\n") on the full file).
	const content = "<!-- AGENT:NAV\npurpose:test\nlines:10\n-->\n\n# Admin UI Guide\n\nThis document has been split.\n\nSee sub-docs.\n"
	path := writeTempFile(t, dir, "short.md", content)

	report, err := File(path, cfg, true /* dry-run */, false)
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}
	if report != noChanges {
		t.Errorf("sub-case 1: report = %q, want noChanges", report)
	}

	// Sub-case 2: lines:N is stale → reports lines-updated and writes correct total lines.
	const contentStale = "<!-- AGENT:NAV\npurpose:test\nlines:99\n-->\n\n# Admin UI Guide\n\nThis document has been split.\n\nSee sub-docs.\n"
	path2 := writeTempFile(t, dir, "short2.md", contentStale)

	report2, err := File(path2, cfg, false /* not dry-run */, false)
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}
	if strings.Contains(report2, "new: #Admin UI Guide") {
		t.Errorf("sub-case 2: update added nav entry to purpose-only below-MinLines file; report:\n%s", report2)
	}
	if !strings.Contains(report2, "lines-updated") {
		t.Errorf("sub-case 2: expected lines-updated in report; got:\n%s", report2)
	}

	// Verify the written value equals total file lines, not content lines.
	data2, err := os.ReadFile(path2)
	if err != nil {
		t.Fatal(err)
	}
	pr2 := navblock.ParseNavBlock(string(data2))
	if !pr2.Found {
		t.Fatal("sub-case 2: nav block not found after update")
	}
	wantLines := strings.Count(string(data2), "\n")
	if pr2.Block.Lines != wantLines {
		t.Errorf("sub-case 2: lines:N = %d, want total file lines %d", pr2.Block.Lines, wantLines)
	}
}

// TestFile_DuplicateHeadingMatching verifies that when a document has two
// sections with the same heading name, each nav entry is matched to the section
// at the same absolute line (exact-match first), not greedily to the first
// occurrence by proximity alone.
//
// Regression for Bug 8: navIndex used a single-value map, causing the second
// entry to overwrite the first. matchSectionsToNav fixes this with a two-pass
// approach (exact line match first, then proximity).
func TestFile_DuplicateHeadingMatching(t *testing.T) {
	dir := t.TempDir()

	// Build a file with two sections named "## Step-by-Step Setup".
	// Nav block records the second occurrence at line ~30 (after the nav block).
	// The two-pass matcher must pair the nav entry at s=30 with the section at
	// line 30, not displace it to the first occurrence at line 10.
	//
	// Layout (after prepended nav block of 6 lines + 1 blank = 7 offset):
	//   line 1-6: nav block, line 7: blank
	//   line  8: # Doc
	//   line  9: (blank)
	//   line 10: ## Step-by-Step Setup  (first occurrence, NOT in nav)
	//   lines 11-28: filler (18 lines)
	//   line 29: ## Other Section
	//   lines 30-38: filler (9 lines)
	//   line 39: ## Step-by-Step Setup  (second occurrence, IN nav at s=39)
	//   lines 40-55: filler (16 lines)

	// Build nav block with only the second occurrence recorded.
	navText := "<!-- AGENT:NAV\npurpose:test\nnav[1]{s,n,name,about}:\n39,17,##Step-by-Step Setup,existing desc\n-->\n"

	var sb strings.Builder
	sb.WriteString(navText)
	sb.WriteString("\n")                      // blank sep after nav block (line 7)
	sb.WriteString("# Doc\n")                 // line 8
	sb.WriteString("\n")                      // line 9
	sb.WriteString("## Step-by-Step Setup\n") // line 10 — first occurrence
	for i := 0; i < 18; i++ {
		sb.WriteString("filler line\n") // lines 11-28
	}
	sb.WriteString("## Other Section\n") // line 29
	for i := 0; i < 9; i++ {
		sb.WriteString("filler line\n") // lines 30-38
	}
	sb.WriteString("## Step-by-Step Setup\n") // line 39 — second occurrence
	for i := 0; i < 16; i++ {
		sb.WriteString("filler line\n") // lines 40-55
	}

	path := writeTempFile(t, dir, "dup.md", sb.String())

	cfg := config.Defaults()
	cfg.MaxDepth = 2

	report, err := File(path, cfg, true /* dry-run */, false)
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}

	// The nav entry at s=39 must NOT be reported as deleted and re-added as new.
	// It should either be OK or shifted (if lines moved slightly), but the
	// existing description "existing desc" must survive.
	if strings.Contains(report, "deleted: ##Step-by-Step Setup") {
		t.Errorf("nav entry for second occurrence was incorrectly deleted; report:\n%s", report)
	}

	// Re-read file (dry-run doesn't write) and check descriptions are preserved.
	// We test via buildUpdatedBlock directly for precision.
	sections := []parser.Section{
		{Heading: parser.Heading{Line: 10, Depth: 2, Text: "Step-by-Step Setup"}, Start: 10, End: 28},
		{Heading: parser.Heading{Line: 29, Depth: 2, Text: "Other Section"}, Start: 29, End: 38},
		{Heading: parser.Heading{Line: 39, Depth: 2, Text: "Step-by-Step Setup"}, Start: 39, End: 55},
	}
	oldBlock := navblock.NavBlock{
		Purpose: "test",
		Nav: []navblock.NavEntry{
			{Start: 39, N: 17, Name: "##Step-by-Step Setup", About: "existing desc"},
		},
	}
	updated := buildUpdatedBlock(oldBlock, sections, nil, nil, cfg, 50, 50)

	// Find the entry matched to line 39 (third section).
	var matchedAbout string
	for _, e := range updated.Nav {
		if e.Start == 39 {
			matchedAbout = e.About
			break
		}
	}
	if matchedAbout != "existing desc" {
		t.Errorf("second occurrence (line 39) about = %q, want %q; full nav: %+v",
			matchedAbout, "existing desc", updated.Nav)
	}

	// The first occurrence (line 10) must have an empty About (new, no description).
	for _, e := range updated.Nav {
		if e.Start == 10 && e.About != "" {
			t.Errorf("first occurrence (line 10) incorrectly inherited about %q; should be empty", e.About)
		}
	}
}

// TestFile_BothDuplicatesInNav verifies that when both occurrences of a
// duplicate heading are tracked in the nav block, each is matched to its own
// section after content shifts. This is the normal post-generate state.
func TestFile_BothDuplicatesInNav(t *testing.T) {
	// Nav block records both occurrences with distinct descriptions.
	// After the nav block is prepended, the sections will be at the same lines
	// (exact match), so both should match and carry their descriptions forward.
	sections := []parser.Section{
		{Heading: parser.Heading{Line: 10, Depth: 2, Text: "Step-by-Step Setup"}, Start: 10, End: 28},
		{Heading: parser.Heading{Line: 29, Depth: 2, Text: "Other Section"}, Start: 29, End: 38},
		{Heading: parser.Heading{Line: 39, Depth: 2, Text: "Step-by-Step Setup"}, Start: 39, End: 55},
	}
	oldBlock := navblock.NavBlock{
		Purpose: "test",
		Nav: []navblock.NavEntry{
			{Start: 10, N: 19, Name: "##Step-by-Step Setup", About: "first desc"},
			{Start: 29, N: 10, Name: "##Other Section", About: "other desc"},
			{Start: 39, N: 17, Name: "##Step-by-Step Setup", About: "second desc"},
		},
	}

	cfg := config.Defaults()
	cfg.MaxDepth = 2
	updated := buildUpdatedBlock(oldBlock, sections, nil, nil, cfg, 50, 50)

	want := map[int]string{
		10: "first desc",
		29: "other desc",
		39: "second desc",
	}
	got := map[int]string{}
	for _, e := range updated.Nav {
		got[e.Start] = e.About
	}
	for line, wantAbout := range want {
		if got[line] != wantAbout {
			t.Errorf("section at line %d: about = %q, want %q; full nav: %+v", line, got[line], wantAbout, updated.Nav)
		}
	}
}

// TestMatchSectionsToNav_ExactPass verifies pass 1: sections whose Start
// exactly matches a nav entry's Start are consumed before proximity kicks in.
func TestMatchSectionsToNav_ExactPass(t *testing.T) {
	// Two sections with the same name. Nav has one entry at s=39 (the second).
	// Pass 1 should bind the entry to section index 2 (line 39), leaving
	// section index 0 (line 10) unmatched.
	sections := []parser.Section{
		{Heading: parser.Heading{Line: 10, Depth: 2, Text: "Dup"}, Start: 10, End: 28},
		{Heading: parser.Heading{Line: 39, Depth: 2, Text: "Dup"}, Start: 39, End: 55},
	}
	byName := navIndex([]navblock.NavEntry{
		{Start: 39, N: 17, Name: "##Dup", About: "desc"},
	})

	matched, usedStart := matchSectionsToNav(sections, byName)

	// Section 0 (line 10) must be unmatched.
	if _, ok := matched[0]; ok {
		t.Errorf("section at line 10 should be unmatched, but was matched to %+v", matched[0])
	}
	// Section 1 (line 39) must be matched.
	if e, ok := matched[1]; !ok {
		t.Errorf("section at line 39 should be matched, but was not")
	} else if e.About != "desc" {
		t.Errorf("section at line 39 matched entry About = %q, want %q", e.About, "desc")
	}
	// usedStart must record the matched entry's Start.
	if !usedStart[39] {
		t.Errorf("usedStart[39] = false, want true")
	}
	if usedStart[10] {
		t.Errorf("usedStart[10] = true, want false")
	}
}

// TestMatchSectionsToNav_ProximityPass verifies pass 2: when a nav entry's
// Start no longer matches any section exactly (section shifted), proximity
// picks the closest remaining candidate rather than the first occurrence.
func TestMatchSectionsToNav_ProximityPass(t *testing.T) {
	// Nav entry was at s=39. Section shifted to s=42 (3 lines down).
	// A separate first occurrence is at s=10 (distance 29 from 39).
	// Proximity must choose s=42 (distance 3) over s=10 (distance 29).
	sections := []parser.Section{
		{Heading: parser.Heading{Line: 10, Depth: 2, Text: "Dup"}, Start: 10, End: 28},
		{Heading: parser.Heading{Line: 42, Depth: 2, Text: "Dup"}, Start: 42, End: 58},
	}
	byName := navIndex([]navblock.NavEntry{
		{Start: 39, N: 17, Name: "##Dup", About: "desc"},
	})

	matched, _ := matchSectionsToNav(sections, byName)

	// Section 0 (line 10) must be unmatched (proximity chose line 42).
	if _, ok := matched[0]; ok {
		t.Errorf("section at line 10 should be unmatched; matched to %+v", matched[0])
	}
	// Section 1 (line 42) must be matched.
	if e, ok := matched[1]; !ok {
		t.Errorf("section at line 42 should be matched by proximity")
	} else if e.About != "desc" {
		t.Errorf("section at line 42 About = %q, want %q", e.About, "desc")
	}
}

// TestMatchSectionsToNav_MoreOccurrencesThanEntries verifies that when the
// document has more occurrences of a heading name than the nav block has
// entries, the stored entry is matched exactly and the extra occurrences are
// left unmatched.
func TestMatchSectionsToNav_MoreOccurrencesThanEntries(t *testing.T) {
	// Three occurrences in document; nav tracks only the middle one (s=30).
	sections := []parser.Section{
		{Heading: parser.Heading{Line: 10, Depth: 2, Text: "Dup"}, Start: 10, End: 20},
		{Heading: parser.Heading{Line: 30, Depth: 2, Text: "Dup"}, Start: 30, End: 40},
		{Heading: parser.Heading{Line: 50, Depth: 2, Text: "Dup"}, Start: 50, End: 60},
	}
	byName := navIndex([]navblock.NavEntry{
		{Start: 30, N: 11, Name: "##Dup", About: "middle"},
	})

	matched, _ := matchSectionsToNav(sections, byName)

	// Only section 1 (line 30) should be matched.
	if _, ok := matched[0]; ok {
		t.Errorf("section at line 10 should not be matched")
	}
	if e, ok := matched[1]; !ok {
		t.Errorf("section at line 30 should be matched")
	} else if e.About != "middle" {
		t.Errorf("section at line 30 About = %q, want %q", e.About, "middle")
	}
	if _, ok := matched[2]; ok {
		t.Errorf("section at line 50 should not be matched")
	}
}

// TestFile_FrontmatterNavBlockSeparation verifies that update preserves the
// newline between the YAML frontmatter closing "---" and the "<!-- AGENT:NAV"
// line. Previously, insertNavBlock would concatenate them as "---<!-- AGENT:NAV"
// because strings.Join(lines[:blockStart], "\n") produces no trailing newline.
func TestFile_FrontmatterNavBlockSeparation(t *testing.T) {
	dir := t.TempDir()

	content := `---
marp: true
theme: default
---
<!-- AGENT:NAV
purpose:Test presentation file
lines:18
nav[1]{s,n,name,about}:
8,3,#First Slide,First slide content
-->

# First Slide

Some content here.

More content.

End.
`
	path := writeTempFile(t, dir, "test.md", content)
	cfg := config.Defaults()

	_, err := File(path, cfg, false, false)
	if err != nil {
		t.Fatalf("File() error: %v", err)
	}

	result, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(result)

	// The frontmatter close "---" must be on its own line, not merged with <!-- AGENT:NAV.
	if strings.Contains(got, "---<!-- AGENT:NAV") {
		t.Errorf("frontmatter close merged with nav block opener: got %q", got[:100])
	}
	// The nav block opener must appear on its own line.
	lines := strings.Split(got, "\n")
	foundSep := false
	for _, line := range lines {
		if strings.TrimSpace(line) == "<!-- AGENT:NAV" {
			foundSep = true
			break
		}
	}
	if !foundSep {
		t.Errorf("nav block opener not found on its own line in:\n%s", got)
	}
}

func TestFile_GeneratesNavBlockWhenMissing(t *testing.T) {
	// update on a file with no nav block should delegate to generate
	// and produce a nav block (not silently skip).
	content := `# Setup Guide

Installation and configuration steps.

## Prerequisites

Required tools and versions.

## Installation

Step-by-step install procedure.

## Configuration

Environment variables and config file.
`
	dir := t.TempDir()
	path := writeTempFile(t, dir, "setup.md", content)

	cfg := config.Defaults()
	cfg.MinLines = 5

	report, err := File(path, cfg, false, false)
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}

	// update should report generation, not noChanges.
	if report == noChanges {
		t.Error("update returned noChanges for a file with no nav block; expected delegation to generate")
	}
	if !strings.Contains(report, "Generated:") {
		t.Errorf("report = %q, want to contain 'Generated:'", report)
	}

	// The file should now have a nav block.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "<!-- AGENT:NAV") {
		t.Error("file has no AGENT:NAV block after update delegated to generate")
	}
}

func TestUpdate_SubdirDoesNotCreateAgentsMDInSubdir(t *testing.T) {
	// When update is called with a subdir, AGENTS.md (or AGENTMAP.md) must be
	// written to repoRoot, not to the subdir.
	repoRoot := t.TempDir()
	subdir := filepath.Join(repoRoot, "docs", "api")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}

	content := `<!-- AGENT:NAV
purpose:API reference
lines:10
nav[1]{s,n,name,about}:
8,3,#API,api endpoints
-->

# API

Some api content here.

More content for the file.
`
	writeTempFile(t, subdir, "api.md", content)

	cfg := config.Defaults()
	cfg.MinLines = 5
	cfg.IndexInlineMax = 20 // small project → inline AGENTS.md

	// Call Update with the subdir as root but repoRoot as the repo root.
	if err := Update(subdir, repoRoot, cfg, false, true); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// AGENTS.md must NOT be created in the subdir.
	subdirAgents := filepath.Join(subdir, "AGENTS.md")
	if _, err := os.Stat(subdirAgents); err == nil {
		t.Errorf("AGENTS.md was created in subdir %s — should only exist at repo root", subdir)
	}

	// AGENTS.md (or AGENTMAP.md) should be at the repo root.
	rootAgents := filepath.Join(repoRoot, "AGENTS.md")
	rootAgentmap := filepath.Join(repoRoot, "AGENTMAP.md")
	_, agentsErr := os.Stat(rootAgents)
	_, agentmapErr := os.Stat(rootAgentmap)
	if agentsErr != nil && agentmapErr != nil {
		t.Errorf("neither AGENTS.md nor AGENTMAP.md was created at repo root %s", repoRoot)
	}
}

func TestUpdate_SkipsAndGeneratesMixedDir(t *testing.T) {
	// update on a directory containing both indexed and unindexed files
	// should refresh the indexed file and generate for the unindexed one.
	dir := t.TempDir()

	// File 1: already has a nav block.
	existing := `<!-- AGENT:NAV
purpose:Existing indexed file
lines:10
nav[1]{s,n,name,about}:
7,3,#Existing,existing section
-->

# Existing

Some content here to make it long enough.

More content for the file.
`
	writeTempFile(t, dir, "existing.md", existing)

	// File 2: no nav block.
	newFile := `# New File

Brand new content that needs indexing.

## Section One

First section content.

## Section Two

Second section content.
`
	writeTempFile(t, dir, "new.md", newFile)

	cfg := config.Defaults()
	cfg.MinLines = 5

	if err := Update(dir, "", cfg, false, true); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// new.md should now have a nav block.
	newPath := filepath.Join(dir, "new.md")
	newData, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(newData), "<!-- AGENT:NAV") {
		t.Errorf("new.md has no AGENT:NAV block after Update(); update did not delegate to generate")
	}
}
