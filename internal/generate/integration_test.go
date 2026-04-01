package generate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ryankelln/agentmap/internal/config"
	"github.com/ryankelln/agentmap/internal/navblock"
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

### PKCE

Proof key for code exchange.

### Implicit

Legacy implicit grant flow.

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
nav[1]{s,e,name,about}:
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

	err := Generate(dir, cfg, false)
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
	if !strings.Contains(got, "###PKCE") {
		t.Error("nav should contain ###PKCE")
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

	_, err := File(filepath.Join(dir, "code-fence.md"), cfg, false)
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "code-fence.md"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)

	// Parse the nav block to verify headings
	block, _, _, found := navblock.ParseNavBlock(got)
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

	_, err := File(filepath.Join(dir, "duplicates.md"), cfg, false)
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "duplicates.md"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)

	block, _, _, found := navblock.ParseNavBlock(got)
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
	_, err := File(filepath.Join(dir, "empty.md"), cfg, false)
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

	_, err := File(filepath.Join(dir, "large.md"), cfg, false)
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "large.md"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)

	block, _, _, found := navblock.ParseNavBlock(got)
	if !found {
		t.Fatal("nav block not found")
	}

	// Should have all 20 h2 sections plus the h1 plus the 2 h3s = 23 entries
	if len(block.Nav) != 23 {
		t.Errorf("expected 23 nav entries, got %d", len(block.Nav))
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

	// Verify h3 entries are present
	if !strings.Contains(got, "###Subsection A") {
		t.Error("nav should contain ###Subsection A")
	}
	if !strings.Contains(got, "###Subsection B") {
		t.Error("nav should contain ###Subsection B")
	}

	// Verify line ranges are in order (each entry starts after the previous)
	for i := 1; i < len(block.Nav); i++ {
		if block.Nav[i].Start <= block.Nav[i-1].Start {
			t.Errorf("entry %d start (%d) should be after entry %d start (%d)",
				i, block.Nav[i].Start, i-1, block.Nav[i-1].Start)
		}
	}
}
