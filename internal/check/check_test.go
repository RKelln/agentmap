// Package check provides validation of nav blocks against document headings.
package check

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ryankelln/agentmap/internal/config"
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
	failed, report, err := CheckFile(path, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if failed {
		t.Errorf("expected pass but got failure: %s", report)
	}
}

func TestCheckFile_LineNumberMismatch(t *testing.T) {
	tmpDir := t.TempDir()
	content := `---
title: Test
---

<!-- AGENT:NAV
purpose:test file
nav[1]{s,e,name,about}:
7,20,#Test,test section
-->

# Test

Some content here.
`
	path := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	failed, report, err := CheckFile(path, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !failed {
		t.Error("expected failure but passed")
	}
	if !strings.Contains(report, "nav says") || !strings.Contains(report, "actual") {
		t.Errorf("expected line mismatch report, got: %s", report)
	}
}

func TestCheckFile_MissingHeadingInNav(t *testing.T) {
	tmpDir := t.TempDir()
	content := `---
title: Test
---

<!-- AGENT:NAV
purpose:test file
nav[1]{s,e,name,about}:
11,16,#Test,test section
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
	failed, report, err := CheckFile(path, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !failed {
		t.Error("expected failure but passed")
	}
	if !strings.Contains(report, "in document but not in nav") {
		t.Errorf("expected missing heading report, got: %s", report)
	}
}

func TestCheckFile_ExtraHeadingInNav(t *testing.T) {
	tmpDir := t.TempDir()
	content := `---
title: Test
---

<!-- AGENT:NAV
purpose:test file
nav[2]{s,e,name,about}:
12,15,#Test,test section
16,18,##Extra,extra section
-->

# Test

Some content.
`
	path := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	failed, report, err := CheckFile(path, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !failed {
		t.Error("expected failure but passed")
	}
	if !strings.Contains(report, "in nav but not in document") {
		t.Errorf("expected extra heading report, got: %s", report)
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
	failed, report, err := CheckFile(path, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if failed {
		t.Errorf("expected pass but got failure: %s", report)
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
	failed, report, err := CheckFile(path, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if failed {
		t.Errorf("expected pass but got failure: %s", report)
	}
}

func TestCheckFile_MultipleMismatches(t *testing.T) {
	tmpDir := t.TempDir()
	content := `---
title: Test
---

<!-- AGENT:NAV
purpose:test file
nav[3]{s,e,name,about}:
7,25,#Test,test section
9,15,##OldOne,old section
20,25,##Missing,missing section
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
	failed, report, err := CheckFile(path, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !failed {
		t.Error("expected failure but passed")
	}
	// Should have multiple failures
	if !strings.Contains(report, "in nav but not in document") {
		t.Errorf("expected extra heading report, got: %s", report)
	}
	if !strings.Contains(report, "in document but not in nav") {
		t.Errorf("expected missing heading report, got: %s", report)
	}
}
