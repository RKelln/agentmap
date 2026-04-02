package main

import (
	"bytes"
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
