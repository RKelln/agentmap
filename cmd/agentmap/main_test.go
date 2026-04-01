package main

import (
	"bytes"
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

func TestCheckCommand_NotYetImplemented(t *testing.T) {
	output, err := executeCommand("check", ".")
	if err != nil {
		t.Fatalf("ExecuteC() error = %v", err)
	}

	if !strings.Contains(output, "not yet implemented") {
		t.Errorf("check output = %q, want 'not yet implemented'", output)
	}
}

func TestUpdateCommand_NotYetImplemented(t *testing.T) {
	output, err := executeCommand("update", ".")
	if err != nil {
		t.Fatalf("ExecuteC() error = %v", err)
	}

	if !strings.Contains(output, "not yet implemented") {
		t.Errorf("update output = %q, want 'not yet implemented'", output)
	}
}
