package generate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RKelln/agentmap/internal/config"
	"github.com/RKelln/agentmap/internal/navblock"
)

// writeTempFile creates a file in dir with the given name and content.
func writeTempFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestGenerate_Integration(t *testing.T) {
	dir := t.TempDir()

	// Large file (>50 lines) -- should get full nav block
	largeContent := `# Authentication

Token lifecycle management for the platform.

## Token Exchange

OAuth2 code-for-token flow.

This section describes the main authentication flow.

` + strings.Repeat("Additional details about token exchange.\n", 15) + `
### PKCE

Proof key for code exchange.

### Implicit

Legacy implicit grant flow.

` + strings.Repeat("More exchange details.\n", 15) + `
## Token Refresh

Silent rotation and sliding-window expiry.

More content about token refresh mechanisms.

## Token Revocation

Logout and forced invalidation.

Details about revocation endpoints.

## Migration Guide

Upgrading from v1 tokens.

` + strings.Repeat("More migration details.\n", 40)

	writeTempFile(t, dir, "authentication.md", largeContent)

	// Small file (<50 lines) -- should get purpose-only block
	smallContent := `# Helpers

Some helper utilities for date formatting.

## FormatDate

Formats a date string.

## ParseDate

Parses a date string.
`
	writeTempFile(t, dir, "helpers.md", smallContent)

	// File with frontmatter
	fmContent := `---
title: API Reference
author: team
---
# API Reference

REST API documentation.

## Users

User management endpoints.

## Products

Product catalog endpoints.

` + strings.Repeat("More endpoint details.\n", 45)

	writeTempFile(t, dir, "api-reference.md", fmContent)

	// File with existing nav block
	existingNavContent := `<!-- AGENT:NAV
purpose:old purpose
nav[1]{s,n,name,about}:
1,10,#Old Heading,old description
-->
# New Document

This is the actual content.

## Section One

Content for section one.

## Section Two

Content for section two.

` + strings.Repeat("More content here.\n", 45)

	writeTempFile(t, dir, "existing-nav.md", existingNavContent)

	cfg := config.Defaults()
	cfg.MinLines = 50

	err := Generate(dir, cfg, false, true)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Verify large file got full nav block
	data, err := os.ReadFile(filepath.Join(dir, "authentication.md"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !strings.Contains(got, "<!-- AGENT:NAV") {
		t.Error("large file should have AGENT:NAV block")
	}
	if !strings.Contains(got, "nav[") {
		t.Error("large file should have nav entries")
	}
	if !strings.Contains(got, "#Authentication") {
		t.Error("nav should contain #Authentication")
	}
	if !strings.Contains(got, "##Token Exchange") {
		t.Error("nav should contain ##Token Exchange")
	}
	// Token Exchange is medium-sized, so h3 children appear as > hints
	if !strings.Contains(got, ">PKCE") && !strings.Contains(got, "PKCE") {
		t.Error("nav should have hint for PKCE")
	}
	if !strings.Contains(got, "##Token Refresh") {
		t.Error("nav should contain ##Token Refresh")
	}
	if !strings.Contains(got, "##Migration Guide") {
		t.Error("nav should contain ##Migration Guide")
	}

	// Verify small file got purpose-only block
	data, err = os.ReadFile(filepath.Join(dir, "helpers.md"))
	if err != nil {
		t.Fatal(err)
	}
	got = string(data)
	if !strings.Contains(got, "<!-- AGENT:NAV") {
		t.Error("small file should have AGENT:NAV block")
	}
	if strings.Contains(got, "nav[") {
		t.Error("small file should NOT have nav entries (purpose-only)")
	}

	// Verify file with frontmatter has nav after ---
	data, err = os.ReadFile(filepath.Join(dir, "api-reference.md"))
	if err != nil {
		t.Fatal(err)
	}
	got = string(data)
	fmEnd := strings.Index(got, "---\n")
	if fmEnd < 0 {
		t.Fatal("frontmatter closing --- not found")
	}
	// Find the first --- (opening) and second --- (closing)
	firstDelim := strings.Index(got, "---")
	secondDelim := strings.Index(got[firstDelim+3:], "---")
	if secondDelim < 0 {
		t.Fatal("frontmatter closing --- not found")
	}
	fmEnd = firstDelim + 3 + secondDelim
	navStart := strings.Index(got, "<!-- AGENT:NAV")
	if navStart < fmEnd {
		t.Error("nav block should appear after frontmatter")
	}
	if !strings.Contains(got, "#API Reference") {
		t.Error("nav should contain #API Reference")
	}

	// Verify file with existing nav block gets replaced
	data, err = os.ReadFile(filepath.Join(dir, "existing-nav.md"))
	if err != nil {
		t.Fatal(err)
	}
	got = string(data)
	if strings.Contains(got, "old purpose") {
		t.Error("old purpose should be replaced")
	}
	if strings.Contains(got, "#Old Heading") {
		t.Error("old heading should be replaced")
	}
	if !strings.Contains(got, "#New Document") {
		t.Error("nav should contain #New Document")
	}
}

func TestGenerate_CodeFenceAwareness(t *testing.T) {
	dir := t.TempDir()

	content := `# Documentation

This file documents the API.

## Usage

Here is an example:

` + "```" + `
# This is NOT a heading
## Neither is this
### Or this
code inside fence
` + "```" + `

## Real Section

This is a real section after the code fence.

` + strings.Repeat("More content.\n", 45)

	writeTempFile(t, dir, "code-fence.md", content)

	cfg := config.Defaults()
	cfg.MinLines = 50

	_, err := File(filepath.Join(dir, "code-fence.md"), cfg, false, true)
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "code-fence.md"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)

	// Parse the nav block to verify headings
	parseResult := navblock.ParseNavBlock(got)
	block, found := parseResult.Block, parseResult.Found
	if !found {
		t.Fatal("nav block not found")
	}

	for _, entry := range block.Nav {
		if strings.Contains(entry.Name, "This is NOT a heading") {
			t.Error("code fence heading should NOT be in nav block")
		}
		if strings.Contains(entry.Name, "Neither is this") {
			t.Error("code fence heading should NOT be in nav block")
		}
		if strings.Contains(entry.Name, "Or this") {
			t.Error("code fence heading should NOT be in nav block")
		}
	}

	// Verify real headings ARE present
	if !strings.Contains(got, "#Documentation") {
		t.Error("nav should contain #Documentation")
	}
	if !strings.Contains(got, "##Usage") {
		t.Error("nav should contain ##Usage")
	}
	if !strings.Contains(got, "##Real Section") {
		t.Error("nav should contain ##Real Section")
	}
}

func TestGenerate_DuplicateHeadings(t *testing.T) {
	dir := t.TempDir()

	content := `# Guide

A guide with duplicate headings.

## Examples

First examples section.

Some content here.

## Examples

Second examples section with the same heading.

More content here.

## Conclusion

Final section.

` + strings.Repeat("Conclusion details.\n", 45)

	writeTempFile(t, dir, "duplicates.md", content)

	cfg := config.Defaults()
	cfg.MinLines = 50

	_, err := File(filepath.Join(dir, "duplicates.md"), cfg, false, true)
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "duplicates.md"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)

	parseResult := navblock.ParseNavBlock(got)
	block, found := parseResult.Block, parseResult.Found
	if !found {
		t.Fatal("nav block not found")
	}

	// Count how many ##Examples entries exist
	examplesCount := 0
	for _, entry := range block.Nav {
		if entry.Name == "##Examples" {
			examplesCount++
		}
	}

	if examplesCount != 2 {
		t.Errorf("expected 2 ##Examples entries, got %d", examplesCount)
	}

	// Verify both have different line ranges
	var firstStart, secondStart int
	seenFirst := false
	for _, entry := range block.Nav {
		if entry.Name == "##Examples" {
			if !seenFirst {
				firstStart = entry.Start
				seenFirst = true
			} else {
				secondStart = entry.Start
			}
		}
	}

	if firstStart == 0 || secondStart == 0 {
		t.Error("both ##Examples entries should have valid line ranges")
	}
	if firstStart >= secondStart {
		t.Errorf("first ##Examples (line %d) should come before second (line %d)", firstStart, secondStart)
	}
}

func TestGenerate_EmptyFile(t *testing.T) {
	dir := t.TempDir()

	writeTempFile(t, dir, "empty.md", "")

	cfg := config.Defaults()

	// Should not crash
	_, err := File(filepath.Join(dir, "empty.md"), cfg, false, true)
	if err != nil {
		t.Fatalf("File() on empty file error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "empty.md"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)

	// Empty file should still get a purpose-only block
	if !strings.Contains(got, "<!-- AGENT:NAV") {
		t.Error("empty file should have AGENT:NAV block")
	}
}

func TestGenerate_LargeFile(t *testing.T) {
	dir := t.TempDir()

	// Build a file with 20+ headings
	var b strings.Builder
	b.WriteString("# Project Overview\n\n")
	b.WriteString("This is a large project documentation file.\n\n")

	sections := []string{
		"Getting Started",
		"Installation",
		"Configuration",
		"Authentication",
		"Authorization",
		"API Reference",
		"Endpoints",
		"Models",
		"Error Handling",
		"Retry Policy",
		"Logging",
		"Monitoring",
		"Testing",
		"Deployment",
		"CI/CD",
		"Troubleshooting",
		"FAQ",
		"Changelog",
		"Contributing",
		"License",
	}

	for _, name := range sections {
		b.WriteString("## " + name + "\n\n")
		b.WriteString("Content for " + name + ".\n\n")
		b.WriteString("More details about " + name + ".\n\n")
	}

	// Add some h3 headings under a couple sections
	b.WriteString("### Subsection A\n\nDetails A.\n\n")
	b.WriteString("### Subsection B\n\nDetails B.\n\n")

	content := b.String()
	writeTempFile(t, dir, "large.md", content)

	cfg := config.Defaults()
	cfg.MinLines = 50

	_, err := File(filepath.Join(dir, "large.md"), cfg, false, true)
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "large.md"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)

	parseResult := navblock.ParseNavBlock(got)
	block, found := parseResult.Block, parseResult.Found
	if !found {
		t.Fatal("nav block not found")
	}

	// Should have all 20 h2 sections plus the h1 = 21 entries.
	// The 2 h3 children are over budget and their parent h2 has small N → droppable (no hint).
	if len(block.Nav) != 21 {
		t.Errorf("expected 21 nav entries, got %d", len(block.Nav))
	}

	// Verify all section names are present
	for _, name := range sections {
		found := false
		for _, entry := range block.Nav {
			if entry.Name == "##"+name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("nav block missing ##%s", name)
		}
	}

	// h3 children are dropped (not hinted) because their parent h2 N < sub_threshold
	for _, entry := range block.Nav {
		if entry.Name == "###Subsection A" || entry.Name == "###Subsection B" {
			t.Errorf("h3 %q should have been pruned (over budget)", entry.Name)
		}
	}

	// Verify line ranges are in order (each entry starts after the previous)
	for i := 1; i < len(block.Nav); i++ {
		if block.Nav[i].Start <= block.Nav[i-1].Start {
			t.Errorf("entry %d start (%d) should be after entry %d start (%d)",
				i, block.Nav[i].Start, i-1, block.Nav[i-1].Start)
		}
	}
}

func TestGenerate_KeywordDescriptions(t *testing.T) {
	dir := t.TempDir()

	content := `# Authentication

Token lifecycle management for the platform.

## Token Exchange

OAuth2 code-for-token flow with PKCE proof key.

## Token Refresh

Silent rotation and sliding-window expiry detection.

` + strings.Repeat("More refresh details.\n", 45)

	writeTempFile(t, dir, "auth.md", content)

	cfg := config.Defaults()
	cfg.MinLines = 50

	_, err := File(filepath.Join(dir, "auth.md"), cfg, false, true)
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "auth.md"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)

	parseResult := navblock.ParseNavBlock(got)
	block, found := parseResult.Block, parseResult.Found
	if !found {
		t.Fatal("nav block not found")
	}

	if block.Purpose == "" {
		t.Error("purpose should not be empty")
	}

	for _, entry := range block.Nav {
		if entry.About == "" {
			t.Errorf("entry %q should have keyword description, got empty", entry.Name)
		}
		if strings.Contains(entry.About, ",") {
			t.Errorf("entry %q About should not contain commas: %q", entry.Name, entry.About)
		}
	}
}

func TestGenerate_SubsectionHints(t *testing.T) {
	dir := t.TempDir()

	// Build a file that goes over budget so PruneNavEntries must collapse h3 entries.
	// We need: many h2 sections (to push total over MaxNavEntries=20) plus one "medium"
	// h2 (between sub and expand thresholds) that has h3 children.
	// The medium h2 parent will be hintable, its h3s converted to > hints.
	cfg := config.Defaults()
	cfg.MinLines = 50
	cfg.SubThreshold = 50
	cfg.ExpandThreshold = 150
	cfg.MaxNavEntries = 10 // small budget to force pruning

	var b strings.Builder
	b.WriteString("# Guide\n\nOverview of the guide.\n\n")

	// Add a medium-sized Token Exchange section (N between sub=50 and expand=150)
	// with h3 children that should become hints when pruning occurs.
	b.WriteString("## Token Exchange\n\nMain section about token exchange.\n\n")
	b.WriteString(strings.Repeat("Token exchange details and implementation notes about OAuth2 flows.\n", 20))
	b.WriteString("### PKCE\n\nProof key for code exchange details.\n\n")
	b.WriteString("### Implicit\n\nLegacy implicit grant flow.\n\n")
	b.WriteString(strings.Repeat("More exchange content here covering various scenarios.\n", 20))

	// Add many more h2 sections to push the total entry count over budget (10)
	for i := 1; i <= 10; i++ {
		fmt.Fprintf(&b, "## Section%d\n\nContent for section %d.\n\n", i, i)
		b.WriteString(strings.Repeat("Section details.\n", 3))
	}

	writeTempFile(t, dir, "hints.md", b.String())

	_, err := File(filepath.Join(dir, "hints.md"), cfg, false, true)
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "hints.md"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)

	parseResult := navblock.ParseNavBlock(got)
	block, found := parseResult.Block, parseResult.Found
	if !found {
		t.Fatal("nav block not found")
	}

	// Result: h3s were hinted away. The remaining h2 entries may exceed budget
	// since PruneNavEntries accepts overrun when only depth ≤ 2 entries remain.
	// Verify we're at most total-minus-h3s (2 h3s collapsed into hints).
	totalBeforePrune := 14 // 1 h1 + 1 Token Exchange h2 + 2 h3 + 10 small h2
	if len(block.Nav) > totalBeforePrune-2 {
		t.Errorf("nav has %d entries, expected at most %d (2 h3s should be collapsed to hints)",
			len(block.Nav), totalBeforePrune-2)
	}

	// Token Exchange must still be present (it's the hintable parent)
	var exchangeEntry navblock.NavEntry
	for _, entry := range block.Nav {
		if entry.Name == "##Token Exchange" {
			exchangeEntry = entry
			break
		}
	}

	if exchangeEntry.Name == "" {
		t.Fatal("##Token Exchange entry not found")
	}

	// Token Exchange N is between sub and expand → hints should be added
	if !strings.Contains(exchangeEntry.About, ">") {
		t.Errorf("##Token Exchange should have > hints, got: %q", exchangeEntry.About)
	}
	if !strings.Contains(exchangeEntry.About, "PKCE") {
		t.Errorf("hints should mention PKCE, got: %q", exchangeEntry.About)
	}
	if !strings.Contains(exchangeEntry.About, "Implicit") {
		t.Errorf("hints should mention Implicit, got: %q", exchangeEntry.About)
	}

	// PKCE and Implicit must not appear as full entries (they were hinted)
	for _, entry := range block.Nav {
		if entry.Name == "###PKCE" || entry.Name == "###Implicit" {
			t.Errorf("h3 %q should not appear as full entry (should be hinted)", entry.Name)
		}
	}
}

func TestGenerate_H3Expansion(t *testing.T) {
	dir := t.TempDir()

	content := `# Guide

Overview.

## Token Lifecycle

This is a very large section about token lifecycle management.
It covers many topics including rotation, expiry, revocation,
introspection, and more.

` + strings.Repeat("Lots of detailed content about token lifecycle.\n", 100) + `
### Refresh

Silent rotation and sliding-window expiry.

` + strings.Repeat("Detailed refresh content.\n", 10) + `
### Revocation

Logout and forced invalidation.

` + strings.Repeat("Detailed revocation content.\n", 10) + `
### Introspection

Token validation endpoint.

` + strings.Repeat("Detailed introspection content.\n", 10) + `
## Other Section

Brief other section.

` + strings.Repeat("Other section details.\n", 20)

	writeTempFile(t, dir, "expand.md", content)

	cfg := config.Defaults()
	cfg.MinLines = 50
	cfg.SubThreshold = 50
	cfg.ExpandThreshold = 150

	_, err := File(filepath.Join(dir, "expand.md"), cfg, false, true)
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "expand.md"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)

	parseResult := navblock.ParseNavBlock(got)
	block, found := parseResult.Block, parseResult.Found
	if !found {
		t.Fatal("nav block not found")
	}

	// 1 h1 + 2 h2 + 3 h3 = 6 entries, well under MaxNavEntries=20 → no pruning.
	// All h3s appear as full entries regardless of parent size.
	h3Names := []string{"###Refresh", "###Revocation", "###Introspection"}
	for _, h3 := range h3Names {
		found := false
		for _, entry := range block.Nav {
			if entry.Name == h3 {
				found = true
				if entry.About == "" {
					t.Errorf("%s should have keyword description", h3)
				}
				break
			}
		}
		if !found {
			t.Errorf("nav block missing %s as full entry", h3)
		}
	}
}

func TestGenerate_NoSubsectionInfoForSmallSections(t *testing.T) {
	dir := t.TempDir()

	// When a small parent section (N < sub_threshold) has h3 children and we're
	// over budget, the h3s are dropped with NO hint (not hintable, only droppable).
	// We force over-budget by using a tiny MaxNavEntries.
	cfg := config.Defaults()
	cfg.MinLines = 50
	cfg.SubThreshold = 50
	cfg.ExpandThreshold = 150
	cfg.MaxNavEntries = 3 // force pruning of both h3s (5 entries → need to drop 2)

	// File must exceed min_lines (50) to get full nav block.
	// Small Section N < subThreshold(50) → droppable (no hints for its h3s).
	content := `# Guide

Overview of the guide document.

## Small Section

Brief content here.

### SubA

Detail A.

### SubB

Detail B.

## Another Section

` + strings.Repeat("Content for another section to push file over min_lines.\n", 45)

	writeTempFile(t, dir, "small.md", content)

	_, err := File(filepath.Join(dir, "small.md"), cfg, false, true)
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "small.md"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)

	parseResult := navblock.ParseNavBlock(got)
	block, found := parseResult.Block, parseResult.Found
	if !found {
		t.Fatal("nav block not found")
	}

	// Result must be within budget
	if len(block.Nav) > cfg.MaxNavEntries {
		t.Errorf("nav has %d entries, want ≤ %d", len(block.Nav), cfg.MaxNavEntries)
	}

	var entry navblock.NavEntry
	for _, e := range block.Nav {
		if e.Name == "##Small Section" {
			entry = e
			break
		}
	}

	if entry.Name == "" {
		t.Fatal("##Small Section entry not found")
	}

	// Small section N < sub_threshold → droppable, not hintable → no > hints
	if strings.Contains(entry.About, ">") {
		t.Errorf("small section should not have > hints, got: %q", entry.About)
	}

	// h3 children should be dropped (not hinted) for a droppable parent
	for _, e := range block.Nav {
		if e.Name == "###SubA" || e.Name == "###SubB" {
			t.Errorf("h3 %q should not appear: small parent is droppable (N < sub_threshold)", e.Name)
		}
	}
}
