package discovery

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"testing"
)

func TestDiscoverFilesFindsAllMdFiles(t *testing.T) {
	dir := t.TempDir()

	// Create a directory structure with .md files
	writeFile(t, dir, "README.md", "# Root")
	writeFile(t, dir, "docs/guide.md", "# Guide")
	writeFile(t, dir, "docs/api/reference.md", "# API Reference")
	writeFile(t, dir, "src/main.go", "package main")

	files, err := DiscoverFiles(dir, nil)
	if err != nil {
		t.Fatalf("DiscoverFiles() error = %v", err)
	}

	sort.Strings(files)
	expected := []string{"README.md", "docs/api/reference.md", "docs/guide.md"}
	if len(files) != len(expected) {
		t.Fatalf("got %d files, want %d: %v", len(files), len(expected), files)
	}
	for i, f := range expected {
		if files[i] != f {
			t.Errorf("files[%d] = %q, want %q", i, files[i], f)
		}
	}
}

func TestDiscoverFilesExcludesNonMdFiles(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "readme.md", "# Readme")
	writeFile(t, dir, "notes.txt", "some notes")
	writeFile(t, dir, "config.json", "{}")
	writeFile(t, dir, "script.py", "print('hi')")

	files, err := DiscoverFiles(dir, nil)
	if err != nil {
		t.Fatalf("DiscoverFiles() error = %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("got %d files, want 1: %v", len(files), files)
	}
	if files[0] != "readme.md" {
		t.Errorf("files[0] = %q, want %q", files[0], "readme.md")
	}
}

func TestDiscoverFilesWithExcludePatterns(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "README.md", "# Root")
	writeFile(t, dir, "docs/guide.md", "# Guide")
	writeFile(t, dir, "docs/internal/secret.md", "# Secret")
	writeFile(t, dir, "CHANGELOG.md", "# Changelog")

	patterns := []string{"docs/internal/*", "CHANGELOG.md"}
	files, err := DiscoverFiles(dir, patterns)
	if err != nil {
		t.Fatalf("DiscoverFiles() error = %v", err)
	}

	sort.Strings(files)
	expected := []string{"README.md", "docs/guide.md"}
	if len(files) != len(expected) {
		t.Fatalf("got %d files, want %d: %v", len(files), len(expected), files)
	}
	for i, f := range expected {
		if files[i] != f {
			t.Errorf("files[%d] = %q, want %q", i, files[i], f)
		}
	}
}

func TestDiscoverFilesExcludesHiddenDirectoriesByDefault(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "README.md", "# Root")
	writeFile(t, dir, ".opencode/commands/release.md", "# Hidden tool docs")
	writeFile(t, dir, "docs/.draft/notes.md", "# Hidden draft")
	writeFile(t, dir, "docs/guide.md", "# Guide")

	files, err := DiscoverFiles(dir, nil)
	if err != nil {
		t.Fatalf("DiscoverFiles() error = %v", err)
	}

	sort.Strings(files)
	expected := []string{"README.md", "docs/guide.md"}
	if len(files) != len(expected) {
		t.Fatalf("got %d files, want %d: %v", len(files), len(expected), files)
	}
	for i, f := range expected {
		if files[i] != f {
			t.Errorf("files[%d] = %q, want %q", i, files[i], f)
		}
	}
}

func TestDiscoverFilesEmptyDirectory(t *testing.T) {
	dir := t.TempDir()

	files, err := DiscoverFiles(dir, nil)
	if err != nil {
		t.Fatalf("DiscoverFiles() error = %v", err)
	}

	if len(files) != 0 {
		t.Errorf("got %d files, want 0", len(files))
	}
}

func TestDiscoverFilesNonExistentRoot(t *testing.T) {
	_, err := DiscoverFiles("/nonexistent/path/that/does/not/exist", nil)
	if err == nil {
		t.Error("DiscoverFiles() error = nil, want error for nonexistent root")
	}
}

func TestIsGitRepo(t *testing.T) {
	// This project's root is a git repo
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}

	// Navigate up to find the project root (where .git is)
	root := findProjectRoot(cwd)
	if root == "" {
		t.Skip("could not find project root")
	}

	if !isGitRepo(root) {
		t.Error("isGitRepo(project root) = false, want true")
	}

	// A temp directory is not a git repo
	tempDir := t.TempDir()
	if isGitRepo(tempDir) {
		t.Error("isGitRepo(temp dir) = true, want false")
	}
}

func TestGitLsFiles(t *testing.T) {
	// Create a temp directory and initialize a git repo
	dir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init error = %v", err)
	}

	// Create and stage some files
	writeFile(t, dir, "README.md", "# Readme")
	writeFile(t, dir, "docs/guide.md", "# Guide")
	writeFile(t, dir, "src/main.go", "package main")

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git add error = %v", err)
	}

	files, err := gitLsFiles(dir)
	if err != nil {
		t.Fatalf("gitLsFiles() error = %v", err)
	}

	sort.Strings(files)
	expected := []string{"README.md", "docs/guide.md", "src/main.go"}
	if len(files) != len(expected) {
		t.Fatalf("got %d files, want %d: %v", len(files), len(expected), files)
	}
	for i, f := range expected {
		if files[i] != f {
			t.Errorf("files[%d] = %q, want %q", i, files[i], f)
		}
	}
}

func TestMatchesExclude(t *testing.T) {
	tests := []struct {
		path     string
		patterns []string
		want     bool
	}{
		{"CHANGELOG.md", []string{"CHANGELOG.md"}, true},
		{"docs/guide.md", []string{"CHANGELOG.md"}, false},
		{"docs/internal/secret.md", []string{"docs/internal/*"}, true},
		{"docs/public/guide.md", []string{"docs/internal/*"}, false},
		{"README.md", []string{}, false},
		{"dist/build.md", []string{"dist/**"}, true},
		{"agents/commands/release.md", []string{"agents/**"}, true},
		{"agents.md", []string{"agents/**"}, false},
		{"src/test.md", []string{"*.md"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := matchesExclude(tt.path, tt.patterns)
			if got != tt.want {
				t.Errorf("matchesExclude(%q, %v) = %v, want %v", tt.path, tt.patterns, got, tt.want)
			}
		})
	}
}

func TestHasHiddenDir(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"README.md", false},
		{"docs/guide.md", false},
		{".opencode/commands/release.md", true},
		{"docs/.draft/notes.md", true},
		{"docs/.hidden.md", false}, // hidden filename only is allowed
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := hasHiddenDir(tt.path)
			if got != tt.want {
				t.Errorf("hasHiddenDir(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

// Helper functions

func writeFile(t *testing.T, dir, path, content string) {
	t.Helper()
	fullPath := filepath.Join(dir, path)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(fullPath), err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", fullPath, err)
	}
}

func findProjectRoot(dir string) string {
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}
