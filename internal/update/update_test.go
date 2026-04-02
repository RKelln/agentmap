package update

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ryankelln/agentmap/internal/config"
	"github.com/ryankelln/agentmap/internal/navblock"
	"github.com/ryankelln/agentmap/internal/parser"
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

	updated := buildUpdatedBlock(oldBlock, sections, nil, nil, config.Defaults())
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

	report, err := File(path, cfg, false, false)
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}

	if report != noChanges {
		t.Errorf("expected noChanges for file without nav block, got: %s", report)
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

	if report != noChanges {
		t.Errorf("expected noChanges for file without nav block, got: %s", report)
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
9,16,#Document,main doc
13,4,##Section A,section A content
17,4,##Section B,section B content
21,4,##Section C,section C content
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
