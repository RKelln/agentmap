package parser

import (
	"reflect"
	"strings"
	"testing"
)

func TestComputeSections_EmptySection(t *testing.T) {
	// §11.3: heading immediately followed by another heading → n=1 (heading line only).
	// Minimum valid n is 1 (the heading itself).
	headings := []Heading{
		{Line: 1, Depth: 1, Text: "Title"},
		{Line: 2, Depth: 2, Text: "Empty Section"},
		{Line: 3, Depth: 2, Text: "Non-Empty Section"},
	}

	sections := ComputeSections(headings, 6)

	if len(sections) != 3 {
		t.Fatalf("len(sections) = %d, want 3", len(sections))
	}

	// "Empty Section" at line 2, followed immediately by another heading at line 3 → End = 2, Len = 1
	emptySection := sections[1]
	if emptySection.Text != "Empty Section" {
		t.Fatalf("sections[1].Text = %q, want %q", emptySection.Text, "Empty Section")
	}
	if emptySection.Len() != 1 {
		t.Errorf("empty section Len() = %d, want 1 (heading line only)", emptySection.Len())
	}
	if emptySection.Start != 2 || emptySection.End != 2 {
		t.Errorf("empty section Start=%d End=%d, want Start=2 End=2", emptySection.Start, emptySection.End)
	}
}

func TestBuildNavEntries_EmptySection(t *testing.T) {
	// §11.3: empty section gets included in nav with empty about field.
	// (This test lives in the parser package for section shape; generate has the nav entry test.)
	headings := []Heading{
		{Line: 1, Depth: 2, Text: "First"},
		{Line: 2, Depth: 2, Text: "Second"},
	}
	sections := ComputeSections(headings, 5)
	if len(sections) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(sections))
	}
	if sections[0].Len() != 1 {
		t.Errorf("First section Len() = %d, want 1 (empty section)", sections[0].Len())
	}
}

func TestComputeSections_DesignDocScenario(t *testing.T) {
	// Simulates the heading structure of agentmap-design.md
	// with h2 "##3. Format Specification" containing h3 children
	headings := []Heading{
		{Line: 1, Depth: 1, Text: "agentmap: Navigation Maps for AI Agents"},
		{Line: 4, Depth: 2, Text: "Design Document v0.1"},
		{Line: 6, Depth: 2, Text: "1. Problem"},
		{Line: 18, Depth: 2, Text: "2. Solution"},
		{Line: 28, Depth: 2, Text: "3. Format Specification"},
		{Line: 30, Depth: 3, Text: "3.1 Nav Block Structure"},
		{Line: 43, Depth: 3, Text: "3.2 Field Definitions"},
		{Line: 71, Depth: 3, Text: "3.3 Heading Depth Convention"},
		{Line: 92, Depth: 3, Text: "3.4 Complete Example"},
		{Line: 127, Depth: 3, Text: "3.5 Placement Rules"},
		{Line: 133, Depth: 3, Text: "3.6 Constraints"},
		{Line: 140, Depth: 3, Text: "3.7 Purpose-Only Block"},
		{Line: 152, Depth: 3, Text: "3.8 Subsection Hints"},
		{Line: 186, Depth: 2, Text: "4. CLI Commands"},
		{Line: 188, Depth: 3, Text: "4.1 agentmap generate [path]"},
		{Line: 219, Depth: 3, Text: "4.2 agentmap update [path]"},
		{Line: 270, Depth: 3, Text: "4.3 agentmap check [path]"},
		{Line: 300, Depth: 2, Text: "5. Description Authoring"},
	}

	sections := ComputeSections(headings, 350)

	// h2 "3. Format Specification" should span from line 28 to 185
	// (ending before h2 "4. CLI Commands" at line 186)
	formatSpec := sections[4]
	if formatSpec.Text != "3. Format Specification" {
		t.Fatalf("expected section 4 to be '3. Format Specification', got %q", formatSpec.Text)
	}
	if formatSpec.End != 185 {
		t.Errorf("##3. Format Specification: End = %d, want 185 (should end before ##4. CLI Commands)", formatSpec.End)
	}

	// h3 "3.1 Nav Block Structure" should span from 30 to 42
	// (ending before h3 "3.2 Field Definitions" at line 43)
	navStructure := sections[5]
	if navStructure.Text != "3.1 Nav Block Structure" {
		t.Fatalf("expected section 5 to be '3.1 Nav Block Structure', got %q", navStructure.Text)
	}
	if navStructure.End != 42 {
		t.Errorf("###3.1 Nav Block Structure: End = %d, want 42 (should end before ###3.2)", navStructure.End)
	}

	// h3 "3.8 Subsection Hints" should span from 152 to 185
	// (ending before h2 "4. CLI Commands" at line 186)
	subsectionHints := sections[12]
	if subsectionHints.Text != "3.8 Subsection Hints" {
		t.Fatalf("expected section 12 to be '3.8 Subsection Hints', got %q", subsectionHints.Text)
	}
	if subsectionHints.End != 185 {
		t.Errorf("###3.8 Subsection Hints: End = %d, want 185 (should end before ##4. CLI Commands)", subsectionHints.End)
	}

	// Verify no section spans overlap incorrectly
	for i := 0; i < len(sections)-1; i++ {
		if sections[i].Start > sections[i].End {
			t.Errorf("section %d (%q): Start %d > End %d", i, sections[i].Text, sections[i].Start, sections[i].End)
		}
	}
}

func TestParseHeadings_DesignDocStripped(t *testing.T) {
	// Verify the parser finds the correct headings from a stripped design doc
	// This catches if headings inside code blocks or comments leak through
	input := `# agentmap: Navigation Maps for AI Agents


## Design Document v0.1

### Status: Design complete, ready for implementation

---

## 1. Problem

AI coding agents waste significant tokens.

## 2. Solution

**agentmap** is a CLI tool.

## 3. Format Specification

### 3.1 Nav Block Structure

` + "```markdown" + `
## This is inside a code block
### So is this
` + "```" + `

### 3.2 Field Definitions

**Header line:** ` + "`<!-- AGENT:NAV`" + `

<!--
## This is inside an HTML comment
-->

### 3.3 Heading Depth Convention

Some content here.

## 4. CLI Commands

### 4.1 agentmap generate [path]

Content.
`

	headings := ParseHeadings(input, 3)

	// Should find exactly these headings
	wantTexts := []string{
		"agentmap: Navigation Maps for AI Agents",
		"Design Document v0.1",
		"Status: Design complete, ready for implementation",
		"1. Problem",
		"2. Solution",
		"3. Format Specification",
		"3.1 Nav Block Structure",
		"3.2 Field Definitions",
		"3.3 Heading Depth Convention",
		"4. CLI Commands",
		"4.1 agentmap generate [path]",
	}

	if len(headings) != len(wantTexts) {
		t.Errorf("found %d headings, want %d", len(headings), len(wantTexts))
		for i, h := range headings {
			t.Logf("  heading %d: line %d, depth %d, text %q", i, h.Line, h.Depth, h.Text)
		}
	}

	for i, want := range wantTexts {
		if i >= len(headings) {
			t.Errorf("missing heading %d: %q", i, want)
			continue
		}
		if headings[i].Text != want {
			t.Errorf("heading %d: got %q, want %q", i, headings[i].Text, want)
		}
	}
}

func TestComputeSections_H3ContainedWithinH2(t *testing.T) {
	// The core bug: h3 sections were extending past their h2 parent
	// because the algorithm only looks for headings at same or higher level,
	// but h3 (depth 3) should stop at the next h3 OR the next h2 (whichever comes first).
	// The current algorithm IS correct for this, so this test verifies it.
	headings := []Heading{
		{Line: 1, Depth: 1, Text: "Root"},
		{Line: 5, Depth: 2, Text: "Parent A"},
		{Line: 10, Depth: 3, Text: "Child A1"},
		{Line: 20, Depth: 3, Text: "Child A2"},
		{Line: 30, Depth: 2, Text: "Parent B"},
		{Line: 35, Depth: 3, Text: "Child B1"},
		{Line: 45, Depth: 3, Text: "Child B2"},
		{Line: 55, Depth: 1, Text: "Another Root"},
	}

	sections := ComputeSections(headings, 60)

	want := []Section{
		{Heading: Heading{Line: 1, Depth: 1, Text: "Root"}, Start: 1, End: 54},
		{Heading: Heading{Line: 5, Depth: 2, Text: "Parent A"}, Start: 5, End: 29},
		{Heading: Heading{Line: 10, Depth: 3, Text: "Child A1"}, Start: 10, End: 19},
		{Heading: Heading{Line: 20, Depth: 3, Text: "Child A2"}, Start: 20, End: 29},
		{Heading: Heading{Line: 30, Depth: 2, Text: "Parent B"}, Start: 30, End: 54},
		{Heading: Heading{Line: 35, Depth: 3, Text: "Child B1"}, Start: 35, End: 44},
		{Heading: Heading{Line: 45, Depth: 3, Text: "Child B2"}, Start: 45, End: 54},
		{Heading: Heading{Line: 55, Depth: 1, Text: "Another Root"}, Start: 55, End: 60},
	}

	if !reflect.DeepEqual(sections, want) {
		t.Errorf("ComputeSections() mismatch:")
		for i := range sections {
			if i < len(want) {
				if !reflect.DeepEqual(sections[i], want[i]) {
					t.Errorf("  [%d] got %+v, want %+v", i, sections[i], want[i])
				}
			} else {
				t.Errorf("  [%d] extra: %+v", i, sections[i])
			}
		}
	}
}

func TestParseAndCompute_DesignDocEndToEnd(t *testing.T) {
	// Full pipeline: parse headings from a realistic design doc snippet,
	// compute sections, verify no duplicates and no missing sections.
	input := `# Title

## 1. Overview

Content for section 1.

## 2. Specification

### 2.1 Structure

Details about structure.

### 2.2 Fields

Details about fields.

### 2.3 Depth

Details about depth.

## 3. CLI Commands

### 3.1 generate

Generate command details.

### 3.2 update

Update command details.

## 4. Edge Cases

### 4.1 Duplicates

How to handle duplicates.

## 5. Future Work

### 5.1 Project Index

Future project index feature.

### 5.2 Cross-File Links

Future cross-file feature.
`

	headings := ParseHeadings(input, 3)
	sections := ComputeSections(headings, 50)

	// Verify heading count matches expected (1 h1 + 5 h2 + 8 h3 = 14)
	if len(sections) != 14 {
		for _, s := range sections {
			t.Logf("  %d-%d %s%s", s.Start, s.End, strings.Repeat("#", s.Depth), s.Text)
		}
		t.Fatalf("expected 13 sections, got %d", len(sections))
	}

	// Verify no duplicate h3 entries
	seen := make(map[string]int)
	for _, s := range sections {
		key := strings.Repeat("#", s.Depth) + s.Text
		seen[key]++
	}
	for key, count := range seen {
		if count > 1 {
			t.Errorf("duplicate section: %q appears %d times", key, count)
		}
	}

	// Verify h2 sections contain their h3 children
	// "2. Specification" (h2, sections[2]) should contain "2.1", "2.2", "2.3" (h3)
	specSection := sections[2] // ## 2. Specification
	if specSection.End < sections[5].End {
		t.Errorf("##2. Specification ends at %d but ###2.3 Depth ends at %d",
			specSection.End, sections[5].End)
	}

	// Verify sections are in document order
	for i := 1; i < len(sections); i++ {
		if sections[i].Start < sections[i-1].Start {
			t.Errorf("sections out of order: section %d starts at %d but section %d starts at %d",
				i, sections[i].Start, i-1, sections[i-1].Start)
		}
	}
}
