// Package check provides validation of nav blocks against document headings.
package check

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RKelln/agentmap/internal/config"
)

func TestCheckFile_MatchingNav(t *testing.T) {
	tmpDir := t.TempDir()
	content := `---
title: Test
---

<!-- AGENT:NAV
purpose:test file
nav[2]{s,n,name,about}:
12,7,#Test,test section
16,3,##Subtest,sub section
-->

# Test

Some content here.

## Subtest

More content.
`
	path := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	failed, report, warnings, err := CheckFile(path, cfg, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if failed {
		t.Errorf("expected pass but got failure: %s", report)
	}
	if len(warnings) > 0 {
		t.Errorf("expected no warnings, got %d", len(warnings))
	}
}

func TestCheckFile_LineNumberMismatch(t *testing.T) {
	tmpDir := t.TempDir()
	content := `---
title: Test
---

<!-- AGENT:NAV
purpose:test file
nav[1]{s,n,name,about}:
7,14,#Test,test section
-->

# Test

Some content here.
`
	path := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	failed, report, warnings, err := CheckFile(path, cfg, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !failed {
		t.Error("expected failure but passed")
	}
	if !strings.Contains(report, "nav says") || !strings.Contains(report, "actual") {
		t.Errorf("expected line mismatch report, got: %s", report)
	}
	if len(warnings) > 0 {
		t.Errorf("expected no warnings, got %d", len(warnings))
	}
}

func TestCheckFile_MissingHeadingInNav(t *testing.T) {
	tmpDir := t.TempDir()
	content := `---
title: Test
---

<!-- AGENT:NAV
purpose:test file
nav[1]{s,n,name,about}:
11,6,#Test,test section
-->

# Test

## Missing

Content here.
`
	path := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	failed, report, warnings, err := CheckFile(path, cfg, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !failed {
		t.Error("expected failure but passed")
	}
	if !strings.Contains(report, "in document but not in nav") {
		t.Errorf("expected missing heading report, got: %s", report)
	}
	if len(warnings) > 0 {
		t.Errorf("expected no warnings, got %d", len(warnings))
	}
}

func TestCheckFile_ExtraHeadingInNav(t *testing.T) {
	tmpDir := t.TempDir()
	content := `---
title: Test
---

<!-- AGENT:NAV
purpose:test file
nav[2]{s,n,name,about}:
12,4,#Test,test section
16,3,##Extra,extra section
-->

# Test

Some content.
`
	path := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	failed, report, warnings, err := CheckFile(path, cfg, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !failed {
		t.Error("expected failure but passed")
	}
	if !strings.Contains(report, "in nav but not in document") {
		t.Errorf("expected extra heading report, got: %s", report)
	}
	if len(warnings) > 0 {
		t.Errorf("expected no warnings, got %d", len(warnings))
	}
}

func TestCheckFile_PurposeOnlyBlock(t *testing.T) {
	tmpDir := t.TempDir()
	content := `---
title: Test
---

<!-- AGENT:NAV
purpose:small file
-->

# Test

Short content.
`
	path := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	failed, report, warnings, err := CheckFile(path, cfg, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if failed {
		t.Errorf("expected pass but got failure: %s", report)
	}
	if len(warnings) > 0 {
		t.Errorf("expected no warnings, got %d", len(warnings))
	}
}

func TestCheckFile_NoNavBlock(t *testing.T) {
	tmpDir := t.TempDir()
	content := `# Test

Some content.
`
	path := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	failed, report, warnings, err := CheckFile(path, cfg, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if failed {
		t.Errorf("expected pass but got failure: %s", report)
	}
	if len(warnings) > 0 {
		t.Errorf("expected no warnings, got %d", len(warnings))
	}
}

func TestCheckFile_MultipleMismatches(t *testing.T) {
	tmpDir := t.TempDir()
	content := `---
title: Test
---

<!-- AGENT:NAV
purpose:test file
nav[3]{s,n,name,about}:
7,3,#Test,test section
9,7,##OldOne,old section
20,6,##Missing,missing section
-->

# Test

## OldOne

Content.

## Changed

More content.
`
	path := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	failed, report, warnings, err := CheckFile(path, cfg, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !failed {
		t.Error("expected failure but passed")
	}
	if !strings.Contains(report, "in nav but not in document") {
		t.Errorf("expected extra heading report, got: %s", report)
	}
	if !strings.Contains(report, "in document but not in nav") {
		t.Errorf("expected missing heading report, got: %s", report)
	}
	if len(warnings) > 0 {
		t.Errorf("expected no warnings, got %d", len(warnings))
	}
}

func TestCheckFile_CommaHeadingRoundtrip(t *testing.T) {
	tmpDir := t.TempDir()
	content := `<!-- AGENT:NAV
purpose:test comma roundtrip
nav[2]{s,n,name,about}:
8,7,#Guide,guide overview
12,3,##Setup Configuration,setup and configuration steps
-->

# Guide

Overview text here.

## Setup, Configuration

Configuration steps here.
`
	path := filepath.Join(tmpDir, "comma.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	failed, report, warnings, err := CheckFile(path, cfg, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if failed {
		t.Errorf("expected pass for comma heading roundtrip, got failure: %s", report)
	}
	if len(warnings) > 0 {
		t.Errorf("expected no warnings, got %d", len(warnings))
	}
}

func TestCheckFile_WarnUnreviewed_AutoGeneratedPurpose(t *testing.T) {
	tmpDir := t.TempDir()
	content := `<!-- AGENT:NAV
purpose:~token OAuth2 authentication flow
nav[1]{s,n,name,about}:
7,3,#Test,test section
-->

# Test

Some content.
`
	path := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	failed, _, warnings, err := CheckFile(path, cfg, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if failed {
		t.Error("expected pass but got failure")
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
	if !strings.Contains(warnings[0], "purpose:") || !strings.Contains(warnings[0], "~token OAuth2") {
		t.Errorf("unexpected warning: %s", warnings[0])
	}
}

func TestCheckFile_WarnUnreviewed_AutoGeneratedAbout(t *testing.T) {
	tmpDir := t.TempDir()
	content := `<!-- AGENT:NAV
purpose:test file
nav[2]{s,n,name,about}:
8,9,#Test,~OAuth2 PKCE redirect token lifecycle
13,4,##Sub,sub section description
-->

# Test

Some content here for testing purposes.
More words to make this section bigger.

## Sub

More content for the subsection here.
Additional words to avoid stub filtering.
`
	path := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	failed, report, warnings, err := CheckFile(path, cfg, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if failed {
		t.Errorf("expected pass but got failure: %s", report)
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
	if !strings.Contains(warnings[0], "#Test") || !strings.Contains(warnings[0], "~OAuth2") {
		t.Errorf("unexpected warning: %s", warnings[0])
	}
}

func TestCheckFile_WarnUnreviewed_MultipleAutoGenerated(t *testing.T) {
	tmpDir := t.TempDir()
	content := `<!-- AGENT:NAV
purpose:~auto generated purpose
nav[3]{s,n,name,about}:
9,11,#Test,~auto generated about
13,4,##Sub1,reviewed description
17,3,##Sub2,~another auto generated
-->

# Test

Some content.

## Sub1

Reviewed content.

## Sub2

Auto content.
`
	path := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	failed, _, warnings, err := CheckFile(path, cfg, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if failed {
		t.Error("expected pass but got failure")
	}
	if len(warnings) != 3 {
		t.Fatalf("expected 3 warnings, got %d: %v", len(warnings), warnings)
	}
}

func TestCheckFile_WarnUnreviewed_FlagDisabled(t *testing.T) {
	tmpDir := t.TempDir()
	content := `<!-- AGENT:NAV
purpose:~auto generated purpose
nav[1]{s,n,name,about}:
7,3,#Test,~auto generated about
-->

# Test

Some content.
`
	path := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	failed, _, warnings, err := CheckFile(path, cfg, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if failed {
		t.Error("expected pass but got failure")
	}
	if len(warnings) != 0 {
		t.Errorf("expected no warnings when flag disabled, got %d", len(warnings))
	}
}

func TestCheckFile_WarnUnreviewed_NoAutoGenerated(t *testing.T) {
	tmpDir := t.TempDir()
	content := `<!-- AGENT:NAV
purpose:reviewed purpose
nav[1]{s,n,name,about}:
7,3,#Test,reviewed description
-->

# Test

Some content.
`
	path := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	failed, _, warnings, err := CheckFile(path, cfg, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if failed {
		t.Error("expected pass but got failure")
	}
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %d", len(warnings))
	}
}

// TestCheckFile_TotalLinesOffByOne verifies that check does not report a false
// mismatch for the last section in a POSIX file (trailing newline).
// strings.Split("...\n", "\n") produces a spurious empty trailing element, so
// len(lines) over-counts by 1; strings.Count must be used instead.
func TestCheckFile_TotalLinesOffByOne(t *testing.T) {
	tmpDir := t.TempDir()
	// File ends with exactly one trailing newline (standard POSIX).
	// The nav block records the last section ending at line 11 (wc -l == 11).
	// With the bug, check would compute totalLines=12 and report "actual 7-12".
	content := "<!-- AGENT:NAV\npurpose:test\nnav[1]{s,n,name,about}:\n7,5,#Heading,about text\n-->\n\n# Heading\n\nSome content.\n\nEnd line.\n"
	// Confirm the file has the expected line count (strings.Count == wc -l).
	if got := strings.Count(content, "\n"); got != 11 {
		t.Fatalf("test setup: expected 11 newlines, got %d", got)
	}
	path := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	failed, report, _, err := CheckFile(path, cfg, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if failed {
		t.Errorf("expected pass (no off-by-one), got failures:\n%s", report)
	}
}

func TestCheckFile_MissingHintWarning(t *testing.T) {
	// File with h2 + 2 h3s; max_nav_entries=2 forces pruning of h3s into > hints.
	content := `<!-- AGENT:NAV
purpose:test
nav[2]{s,n,name,about}:
8,14,#Title,
10,12,##Parent,parent description
-->

# Title

## Parent

Content line one.
Content line two.

### Child One

Content.

### Child Two

Content.
`

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	cfg.MaxNavEntries = 2
	cfg.SubThreshold = 1      // make all sections hintable
	cfg.ExpandThreshold = 999 // prevent unkillable

	failed, report, warnings, err := CheckFile(path, cfg, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if failed {
		t.Errorf("expected pass but got failure: %s", report)
	}
	if len(warnings) == 0 {
		t.Errorf("expected hint warning, got no warnings")
	}
	foundHintWarning := false
	for _, w := range warnings {
		if strings.Contains(w, "missing subsection hint") {
			foundHintWarning = true
			break
		}
	}
	if !foundHintWarning {
		t.Errorf("expected missing subsection hint warning, got: %v", warnings)
	}
}
