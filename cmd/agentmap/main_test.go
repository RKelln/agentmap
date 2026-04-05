package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// executeCommand runs the root command with given args and returns captured output.
func executeCommand(args ...string) (output string, err error) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(args)

	_, err = rootCmd.ExecuteC()
	return buf.String(), err
}

// simpleMarkdown is a minimal markdown file used across multiple check tests.
const simpleMarkdown = "# Test\n\nSome content.\n"

func TestHookCommand_DefaultOutput(t *testing.T) {
	output, err := executeCommand("hook")
	if err != nil {
		t.Fatalf("ExecuteC() error = %v", err)
	}

	wantFragments := []string{
		"#!/bin/sh",
		"agentmap check .",
		"AGENT:NAV blocks are out of sync",
		"agentmap update .",
		"exit 1",
	}
	for _, want := range wantFragments {
		if !strings.Contains(output, want) {
			t.Errorf("hook output missing %q\nfull output: %s", want, output)
		}
	}
}

func TestHookCommand_YAMLFlag(t *testing.T) {
	output, err := executeCommand("hook", "--yaml")
	if err != nil {
		t.Fatalf("ExecuteC() error = %v", err)
	}

	wantFragments := []string{
		"repos:",
		"agentmap-check",
		"Validate AGENT:NAV blocks",
		"agentmap check .",
		"markdown",
		"pass_filenames: false",
	}
	for _, want := range wantFragments {
		if !strings.Contains(output, want) {
			t.Errorf("hook --yaml output missing %q\nfull output: %s", want, output)
		}
	}
}

func TestRootCommand_ListsSubcommands(t *testing.T) {
	output, err := executeCommand("--help")
	if err != nil {
		t.Fatalf("ExecuteC() error = %v", err)
	}

	for _, cmd := range []string{"generate", "update", "check", "version"} {
		if !strings.Contains(output, cmd) {
			t.Errorf("help output missing %q command", cmd)
		}
	}
}

func TestVersionCommand_PrintsVersion(t *testing.T) {
	output, err := executeCommand("version")
	if err != nil {
		t.Fatalf("ExecuteC() error = %v", err)
	}

	if strings.TrimSpace(output) == "" {
		t.Error("version command produced no output")
	}
}

func TestGenerateCommand_FlagParsing(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"no args", []string{"generate"}, false},
		{"dry-run", []string{"generate", "--dry-run"}, false},
		{"min-lines override", []string{"generate", "--min-lines", "10"}, false},
		{"sub-threshold", []string{"generate", "--sub-threshold", "30"}, false},
		{"expand-threshold", []string{"generate", "--expand-threshold", "100"}, false},
		{"all flags", []string{"generate", "--dry-run", "--min-lines", "20", "--sub-threshold", "30", "--expand-threshold", "100"}, false},
		{"invalid min-lines", []string{"generate", "--min-lines", "abc"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := executeCommand(tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteC(%v) error = %v, wantErr %v", tt.args, err, tt.wantErr)
			}
		})
	}
}

func TestGenerateCommand_TooManyArgs(t *testing.T) {
	_, err := executeCommand("generate", ".", "extra")
	if err == nil {
		t.Error("expected error for too many args, got nil")
	}
}

func TestUpdateCommand_FlagParsing(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"no args", []string{"update"}, false},
		{"quiet", []string{"update", "--quiet"}, false},
		{"dry-run", []string{"update", "--dry-run"}, false},
		{"invalid flag", []string{"update", "--bogus"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := executeCommand(tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteC(%v) error = %v, wantErr %v", tt.args, err, tt.wantErr)
			}
		})
	}
}

func TestCheckCommand_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "test.md"), []byte(simpleMarkdown), 0o644); err != nil {
		t.Fatal(err)
	}

	output, err := executeCommand("check", tmpDir)
	if err != nil {
		t.Fatalf("ExecuteC() error = %v", err)
	}

	if !strings.Contains(output, "in sync") {
		t.Errorf("check output = %q, want success message containing 'in sync'", output)
	}
}

func TestCheckCommand_SuccessMessage(t *testing.T) {
	tests := []struct {
		name    string
		files   int
		wantMsg string
	}{
		{
			name:    "single file",
			files:   1,
			wantMsg: "All nav blocks in sync (1 file checked)",
		},
		{
			name:    "multiple files",
			files:   3,
			wantMsg: "All nav blocks in sync (3 files checked)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			for i := 0; i < tt.files; i++ {
				fname := filepath.Join(tmpDir, strings.Repeat("a", i+1)+".md")
				if err := os.WriteFile(fname, []byte(simpleMarkdown), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			output, err := executeCommand("check", tmpDir)
			if err != nil {
				t.Fatalf("ExecuteC() error = %v", err)
			}

			if !strings.Contains(output, tt.wantMsg) {
				t.Errorf("check output = %q, want %q", output, tt.wantMsg)
			}
		})
	}
}

func TestCheckCommand_SingleFileSingular(t *testing.T) {
	tmpDir := t.TempDir()
	fname := filepath.Join(tmpDir, "only.md")
	if err := os.WriteFile(fname, []byte(simpleMarkdown), 0o644); err != nil {
		t.Fatal(err)
	}

	output, err := executeCommand("check", fname)
	if err != nil {
		t.Fatalf("ExecuteC() error = %v", err)
	}

	if !strings.Contains(output, "All nav blocks in sync (1 file checked)") {
		t.Errorf("check single file output = %q, want singular success message", output)
	}
}

func TestVersionCmd_NoCommit(t *testing.T) {
	orig := commit
	commit = ""
	defer func() { commit = orig }()

	output, err := executeCommand("version")
	if err != nil {
		t.Fatalf("ExecuteC() error = %v", err)
	}

	want := version + "\n"
	if output != want {
		t.Errorf("version output = %q, want %q", output, want)
	}
}

func TestVersionCmd_WithCommit(t *testing.T) {
	origCommit := commit
	origVersion := version
	commit = "abc1234"
	version = "v1.2.3"
	defer func() {
		commit = origCommit
		version = origVersion
	}()

	output, err := executeCommand("version")
	if err != nil {
		t.Fatalf("ExecuteC() error = %v", err)
	}

	want := "v1.2.3 (abc1234)\n"
	if output != want {
		t.Errorf("version output = %q, want %q", output, want)
	}
}

func TestVersionCmd_DevDefault(t *testing.T) {
	if version != "dev" {
		t.Errorf("default version = %q, want %q", version, "dev")
	}
}

// captureStdout runs f() and returns everything written to os.Stdout during the call.
func captureStdout(f func()) string {
	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w
	f()
	w.Close() //nolint:errcheck
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r) //nolint:errcheck
	return buf.String()
}

// TestGenerateDebug_LineCount verifies that "generate -D" reports the correct
// line count for a file. The bug: len(strings.Split(content, "\n")) is 1 too
// high for any file ending with a newline (the trailing \n produces an extra
// empty element). The correct count is strings.Count(content, "\n").
func TestGenerateDebug_LineCount(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.md")

	// File with exactly 7 lines, each ending with \n (standard POSIX text file).
	content := "# Heading\n\nParagraph one.\n\n## Section\n\nParagraph two.\n"
	wantLines := strings.Count(content, "\n") // 7 — the correct count

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	// -D writes via fmt.Printf to os.Stdout, not cobra's output buffer, so
	// we must capture os.Stdout directly.
	captured := captureStdout(func() {
		executeCommand("generate", "-D", path) //nolint:errcheck
	})

	// The first output line is: "File: <path> (<N> lines)"
	want := fmt.Sprintf("(%d lines)", wantLines)
	if !strings.Contains(captured, want) {
		t.Errorf("generate -D output:\n%s\nwant line count %q (off-by-1: strings.Split adds spurious empty element for trailing newline)", captured, want)
	}
}
