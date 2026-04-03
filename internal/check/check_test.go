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
12,8,#Test,test section
16,4,##Subtest,sub section
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
8,8,#Guide,guide overview
12,4,##Setup Configuration,setup and configuration steps
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
7,4,#Test,test section
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
8,10,#Test,~OAuth2 PKCE redirect token lifecycle
13,5,##Sub,sub section description
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
9,12,#Test,~auto generated about
13,4,##Sub1,reviewed description
17,4,##Sub2,~another auto generated
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
7,4,#Test,~auto generated about
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
7,4,#Test,reviewed description
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
