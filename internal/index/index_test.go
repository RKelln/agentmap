package index

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RKelln/agentmap/internal/config"
)

// fixtureDir returns the absolute path to testdata/index-fixture relative to
// the module root (two levels up from internal/index/).
func fixtureDir(t *testing.T) string {
	t.Helper()
	// Runtime working directory for tests is the package directory.
	// Walk up to find the module root (contains go.mod).
	dir, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("abs .: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not locate module root (go.mod not found)")
		}
		dir = parent
	}
	return filepath.Join(dir, "testdata", "index-fixture")
}

// copyFixture copies the fixture tree into dst, preserving directory structure.
// Never modifies the original testdata files.
func copyFixture(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(path, target)
	})
	if err != nil {
		t.Fatalf("copyFixture: %v", err)
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()
	_, err = io.Copy(out, in)
	return err
}

// writeFile writes content to path, creating parent directories as needed.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// --- matchExclude tests ---

func TestMatchExclude(t *testing.T) {
	tests := []struct {
		pattern string
		rel     string
		want    bool
	}{
		// Exact filename match.
		{"CHANGELOG.md", "CHANGELOG.md", true},
		{"CHANGELOG.md", "docs/CHANGELOG.md", true}, // base match
		// Single-level glob.
		{"*.md", "README.md", true},
		{"*.md", "docs/README.md", true}, // base match
		// Directory glob with **.
		{"docs/**", "docs/auth.md", true},
		{"docs/**", "docs/sub/auth.md", true},
		{"docs/**", "other/auth.md", false},
		// ** pattern matches the dir itself (rel == prefix) — in practice only
		// files reach matchExclude; directories are pruned by the walk callback.
		{"docs/**", "docs", true},
		// Single-level pattern does not match nested.
		{"docs/*", "docs/auth.md", true},
		{"docs/*", "docs/sub/auth.md", false},
		// No match.
		{"vendor/**", "docs/auth.md", false},
	}

	for _, tt := range tests {
		got := matchExclude(tt.pattern, tt.rel)
		if got != tt.want {
			t.Errorf("matchExclude(%q, %q) = %v, want %v", tt.pattern, tt.rel, got, tt.want)
		}
	}
}

// --- buildTaskEntry tests ---

func TestBuildTaskEntry_LineCount(t *testing.T) {
	// A normal file ends with a trailing newline. strings.Split would produce a
	// spurious empty element and inflate lineCount by 1; strings.Count must be
	// used instead.
	tests := []struct {
		name      string
		content   string
		wantLines int
	}{
		{name: "single line with trailing newline", content: "hello\n", wantLines: 1},
		{name: "three lines with trailing newline", content: "a\nb\nc\n", wantLines: 3},
		{name: "no trailing newline", content: "a\nb\nc", wantLines: 2},
		{name: "empty file", content: "", wantLines: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "test.md"), tt.content)
			cfg := config.Defaults()
			cfg.MinLines = 0
			entry, err := buildTaskEntry(dir, "test.md", cfg)
			if err != nil {
				t.Fatalf("buildTaskEntry() error = %v", err)
			}
			if entry.LineCount != tt.wantLines {
				t.Errorf("LineCount = %d, want %d", entry.LineCount, tt.wantLines)
			}
		})
	}
}

// --- Classify tests ---

func TestBuildIndex_NoNavBlock_GeneratesSkeleton(t *testing.T) {
	dir := t.TempDir()

	// A file with headings but no nav block.
	writeFile(t, filepath.Join(dir, "docs/auth.md"), `# Authentication

Token lifecycle management.

## Token Exchange

OAuth2 code-for-token flow.

## Token Refresh

Silent rotation and expiry detection.
`+strings.Repeat("pad line\n", 40))

	cfg := config.Defaults()
	cfg.MinLines = 5

	result, err := BuildIndex(dir, cfg, false, false)
	if err != nil {
		t.Fatalf("BuildIndex() error = %v", err)
	}

	// File with no nav block → skeleton generated, added to task list.
	if result.Generated != 1 {
		t.Errorf("Generated = %d, want 1", result.Generated)
	}
	if result.TaskFiles != 1 {
		t.Errorf("TaskFiles = %d, want 1", result.TaskFiles)
	}
	if result.Skipped != 0 {
		t.Errorf("Skipped = %d, want 0", result.Skipped)
	}
	if result.TaskPath == "" {
		t.Error("TaskPath should be non-empty when not dry-run")
	}

	// Verify the skeleton was written to the file.
	data, err := os.ReadFile(filepath.Join(dir, "docs/auth.md"))
	if err != nil {
		t.Fatalf("read auth.md: %v", err)
	}
	if !strings.Contains(string(data), "<!-- AGENT:NAV") {
		t.Error("skeleton nav block should have been written to file")
	}
	if !strings.Contains(string(data), "~") {
		t.Error("skeleton nav block should have ~ prefix on descriptions")
	}
}

func TestBuildIndex_NavBlockWithTilde_NoNewSkeleton(t *testing.T) {
	dir := t.TempDir()

	// A file with existing nav block containing ~ descriptions.
	writeFile(t, filepath.Join(dir, "docs/auth.md"), `<!-- AGENT:NAV
purpose:~token lifecycle authentication
nav[2]{s,n,name,about}:
8,20,##Token Exchange,~OAuth2 token flow
28,15,##Token Refresh,~silent rotation
-->

# Authentication

Token lifecycle management.

## Token Exchange

OAuth2 code-for-token flow.

## Token Refresh

Silent rotation and expiry detection.
`)

	cfg := config.Defaults()

	result, err := BuildIndex(dir, cfg, false, false)
	if err != nil {
		t.Fatalf("BuildIndex() error = %v", err)
	}

	// File with ~ in nav block → skip generate; add to task list.
	if result.Generated != 0 {
		t.Errorf("Generated = %d, want 0 (no regeneration for files with existing nav)", result.Generated)
	}
	if result.TaskFiles != 1 {
		t.Errorf("TaskFiles = %d, want 1", result.TaskFiles)
	}
	if result.Skipped != 0 {
		t.Errorf("Skipped = %d, want 0", result.Skipped)
	}
}

func TestBuildIndex_NoTilde_SkippedEntirely(t *testing.T) {
	dir := t.TempDir()

	// A file with existing nav block with no ~ (fully reviewed).
	writeFile(t, filepath.Join(dir, "docs/auth.md"), `<!-- AGENT:NAV
purpose:token lifecycle; OAuth2 exchange
nav[2]{s,n,name,about}:
8,20,##Token Exchange,OAuth2 code-for-token flow
28,15,##Token Refresh,silent rotation and expiry detection
-->

# Authentication

Token lifecycle management.

## Token Exchange

OAuth2 code-for-token flow.

## Token Refresh

Silent rotation and expiry detection.
`)

	cfg := config.Defaults()

	result, err := BuildIndex(dir, cfg, false, false)
	if err != nil {
		t.Fatalf("BuildIndex() error = %v", err)
	}

	// File with no ~ → skip entirely (already indexed).
	if result.Generated != 0 {
		t.Errorf("Generated = %d, want 0", result.Generated)
	}
	if result.TaskFiles != 0 {
		t.Errorf("TaskFiles = %d, want 0", result.TaskFiles)
	}
	if result.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", result.Skipped)
	}
}

func TestBuildIndex_DryRun(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "README.md"), `# Readme

Some content here.
`+strings.Repeat("pad line\n", 50))

	cfg := config.Defaults()
	cfg.MinLines = 5

	result, err := BuildIndex(dir, cfg, true, false)
	if err != nil {
		t.Fatalf("BuildIndex() dry-run error = %v", err)
	}

	// Dry-run: no files written, TaskPath empty.
	if result.TaskPath != "" {
		t.Errorf("TaskPath = %q, want empty for dry-run", result.TaskPath)
	}

	// No skeleton should have been written.
	data, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	if strings.Contains(string(data), "<!-- AGENT:NAV") {
		t.Error("dry-run should not write nav blocks")
	}
}

func TestBuildIndex_Force_RegeneratesExisting(t *testing.T) {
	dir := t.TempDir()

	// A file with a fully-reviewed nav block (no ~).
	writeFile(t, filepath.Join(dir, "docs/auth.md"), `<!-- AGENT:NAV
purpose:token lifecycle; OAuth2 exchange
nav[1]{s,n,name,about}:
8,20,#Authentication,token lifecycle management
-->

# Authentication

Token lifecycle management.

## Token Exchange

OAuth2 code-for-token flow.

## Token Refresh

Silent rotation and expiry detection.
`+strings.Repeat("pad\n", 40))

	cfg := config.Defaults()
	cfg.MinLines = 5

	// With --force, even fully-indexed files get regenerated.
	result, err := BuildIndex(dir, cfg, false, true)
	if err != nil {
		t.Fatalf("BuildIndex() force error = %v", err)
	}

	if result.Generated != 1 {
		t.Errorf("Generated = %d, want 1 (force regenerates)", result.Generated)
	}
}

func TestBuildIndex_UnparsableNavMarker_DoesNotOverwriteWithoutForce(t *testing.T) {
	dir := t.TempDir()

	original := `<!-- AGENT:NAV
purpose:human curated description
nav[1]{s,n,name,about}:
12,20,##Section,custom text

# Guide

## Section

Details here.
`
	writeFile(t, filepath.Join(dir, "docs/guide.md"), original)

	cfg := config.Defaults()
	result, err := BuildIndex(dir, cfg, false, false)
	if err != nil {
		t.Fatalf("BuildIndex() error = %v", err)
	}

	if result.Generated != 0 {
		t.Errorf("Generated = %d, want 0 (should not overwrite unparsable marker file)", result.Generated)
	}
	if result.TaskFiles != 1 {
		t.Errorf("TaskFiles = %d, want 1", result.TaskFiles)
	}

	data, err := os.ReadFile(filepath.Join(dir, "docs/guide.md"))
	if err != nil {
		t.Fatalf("read docs/guide.md: %v", err)
	}
	if string(data) != original {
		t.Error("file content changed; expected unparsable marker file to be preserved")
	}
}

func TestBuildIndex_TaskListContents(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "docs/auth.md"), `# Authentication

Token lifecycle management.

## Token Exchange

OAuth2 code-for-token flow.

## Token Refresh

Silent rotation and expiry detection.
`+strings.Repeat("pad line\n", 40))

	cfg := config.Defaults()
	cfg.MinLines = 5

	result, err := BuildIndex(dir, cfg, false, false)
	if err != nil {
		t.Fatalf("BuildIndex() error = %v", err)
	}

	if result.TaskPath == "" {
		t.Fatal("TaskPath should be set")
	}

	data, err := os.ReadFile(result.TaskPath)
	if err != nil {
		t.Fatalf("read task list: %v", err)
	}
	contents := string(data)

	// Check required sections of the task list format.
	if !strings.Contains(contents, "# agentmap index tasks") {
		t.Error("task list should have title '# agentmap index tasks'")
	}
	if !strings.Contains(contents, "Nav Writing Guide") {
		t.Error("task list should embed a nav writing guide")
	}
	if strings.Contains(contents, "docs/nav-writing-guide.md") {
		t.Error("task list should not reference repo-local nav writing guide path")
	}
	if !strings.Contains(contents, "Progress:") {
		t.Error("task list should have Progress counter")
	}
	if !strings.Contains(contents, "docs/auth.md") {
		t.Error("task list should contain the file path")
	}
	if !strings.Contains(contents, "- [ ]") {
		t.Error("task list should have an unchecked checkbox")
	}
	if !strings.Contains(contents, "<!-- AGENT:NAV") {
		t.Error("task list should embed the nav block for each file")
	}
}

func TestBuildIndex_AgentmapDirExcluded(t *testing.T) {
	dir := t.TempDir()

	// Create a file inside .agentmap/ that should be excluded.
	writeFile(t, filepath.Join(dir, ".agentmap/index-tasks.md"), `# task list`)

	// Create a real file to be indexed.
	writeFile(t, filepath.Join(dir, "README.md"), `# Readme

Some content.
`+strings.Repeat("pad\n", 50))

	cfg := config.Defaults()
	cfg.MinLines = 5

	result, err := BuildIndex(dir, cfg, false, false)
	if err != nil {
		t.Fatalf("BuildIndex() error = %v", err)
	}

	// Only README.md should be discovered, not the .agentmap file.
	// Generated and TaskFiles both count the same file (one generates skeleton + adds to list),
	// so we check TaskFiles directly.
	if result.TaskFiles != 1 {
		t.Errorf("TaskFiles = %d, want 1 (agentmap dir should be excluded)", result.TaskFiles)
	}
	if result.Skipped != 0 {
		t.Errorf("Skipped = %d, want 0", result.Skipped)
	}

	// Verify the task list doesn't mention the .agentmap file.
	if result.TaskPath != "" {
		taskData, err := os.ReadFile(result.TaskPath)
		if err == nil && strings.Contains(string(taskData), ".agentmap") {
			t.Error("task list should not reference files in .agentmap directory")
		}
	}
}

// --- BuildFilesBlock tests ---

func TestBuildFilesBlock_GroupsByDir(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "README.md"), `<!-- AGENT:NAV
purpose:project overview and quickstart
-->
# Readme
`)
	writeFile(t, filepath.Join(dir, "docs/authentication.md"), `<!-- AGENT:NAV
purpose:~token lifecycle authentication
-->
# Authentication
`)
	writeFile(t, filepath.Join(dir, "docs/api/endpoints.md"), `<!-- AGENT:NAV
purpose:REST endpoint catalog
-->
# Endpoints
`)

	cfg := config.Defaults()

	entries, err := BuildFilesBlock(dir, cfg)
	if err != nil {
		t.Fatalf("BuildFilesBlock() error = %v", err)
	}

	if len(entries) != 3 {
		t.Fatalf("len(entries) = %d, want 3", len(entries))
	}

	// Root-level files before dirs.
	if entries[0].RelPath != "README.md" {
		t.Errorf("entries[0].RelPath = %q, want README.md", entries[0].RelPath)
	}
	if entries[1].RelPath != "docs/authentication.md" {
		t.Errorf("entries[1].RelPath = %q, want docs/authentication.md", entries[1].RelPath)
	}
	if entries[2].RelPath != "docs/api/endpoints.md" {
		t.Errorf("entries[2].RelPath = %q, want docs/api/endpoints.md", entries[2].RelPath)
	}
}

func TestBuildFilesBlock_SortsAlphabetically(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "z.md"), `<!-- AGENT:NAV
purpose:z file
-->
# Z
`)
	writeFile(t, filepath.Join(dir, "a.md"), `<!-- AGENT:NAV
purpose:a file
-->
# A
`)
	writeFile(t, filepath.Join(dir, "m.md"), `<!-- AGENT:NAV
purpose:m file
-->
# M
`)

	cfg := config.Defaults()

	entries, err := BuildFilesBlock(dir, cfg)
	if err != nil {
		t.Fatalf("BuildFilesBlock() error = %v", err)
	}

	if len(entries) != 3 {
		t.Fatalf("len(entries) = %d, want 3", len(entries))
	}

	if entries[0].RelPath != "a.md" {
		t.Errorf("entries[0].RelPath = %q, want a.md", entries[0].RelPath)
	}
	if entries[1].RelPath != "m.md" {
		t.Errorf("entries[1].RelPath = %q, want m.md", entries[1].RelPath)
	}
	if entries[2].RelPath != "z.md" {
		t.Errorf("entries[2].RelPath = %q, want z.md", entries[2].RelPath)
	}
}

func TestBuildFilesBlock_StripsAutoGenPrefix(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "docs/auth.md"), `<!-- AGENT:NAV
purpose:~token lifecycle authentication
-->
# Authentication
`)

	cfg := config.Defaults()

	entries, err := BuildFilesBlock(dir, cfg)
	if err != nil {
		t.Fatalf("BuildFilesBlock() error = %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(entries))
	}

	// ~ should be stripped from purpose in files block.
	if strings.HasPrefix(entries[0].Purpose, "~") {
		t.Errorf("entries[0].Purpose = %q, ~ should be stripped", entries[0].Purpose)
	}
	if entries[0].Purpose != "token lifecycle authentication" {
		t.Errorf("entries[0].Purpose = %q, want %q", entries[0].Purpose, "token lifecycle authentication")
	}
}

func TestBuildFilesBlock_ExcludesFilesWithoutNavBlock(t *testing.T) {
	dir := t.TempDir()

	// File with nav block.
	writeFile(t, filepath.Join(dir, "indexed.md"), `<!-- AGENT:NAV
purpose:indexed file
-->
# Indexed
`)
	// File without nav block.
	writeFile(t, filepath.Join(dir, "unindexed.md"), `# Unindexed

No nav block here.
`)

	cfg := config.Defaults()

	entries, err := BuildFilesBlock(dir, cfg)
	if err != nil {
		t.Fatalf("BuildFilesBlock() error = %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1 (unindexed file excluded)", len(entries))
	}
	if entries[0].RelPath != "indexed.md" {
		t.Errorf("entries[0].RelPath = %q, want indexed.md", entries[0].RelPath)
	}
}

func TestBuildFilesBlock_RootFilesBeforeDirs(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "docs/guide.md"), `<!-- AGENT:NAV
purpose:usage guide
-->
# Guide
`)
	writeFile(t, filepath.Join(dir, "README.md"), `<!-- AGENT:NAV
purpose:project readme
-->
# Readme
`)

	cfg := config.Defaults()

	entries, err := BuildFilesBlock(dir, cfg)
	if err != nil {
		t.Fatalf("BuildFilesBlock() error = %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("len(entries) = %d, want 2", len(entries))
	}
	// Root-level files before dirs.
	if entries[0].RelPath != "README.md" {
		t.Errorf("entries[0].RelPath = %q, want README.md (root before dirs)", entries[0].RelPath)
	}
	if entries[1].RelPath != "docs/guide.md" {
		t.Errorf("entries[1].RelPath = %q, want docs/guide.md", entries[1].RelPath)
	}
}

// --- WriteFilesBlock tests ---

func TestWriteFilesBlock_InlineSmall(t *testing.T) {
	dir := t.TempDir()

	entries := make([]FileEntry, 5) // 5 ≤ 20 → inline
	for i := range entries {
		entries[i] = FileEntry{RelPath: "file.md", Dir: "", Name: "file.md", Purpose: "a file"}
	}

	cfg := config.Defaults()
	cfg.IndexInlineMax = 20

	dest, err := WriteFilesBlock(dir, entries, cfg, false)
	if err != nil {
		t.Fatalf("WriteFilesBlock() error = %v", err)
	}

	if dest != filepath.Join(dir, "AGENTS.md") {
		t.Errorf("dest = %q, want AGENTS.md (inline for small projects)", dest)
	}

	data, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	contents := string(data)
	if !strings.Contains(contents, "<!-- agentmap:index -->") {
		t.Error("AGENTS.md should contain opening agentmap:index marker")
	}
	if !strings.Contains(contents, "<!-- /agentmap:index -->") {
		t.Error("AGENTS.md should contain closing /agentmap:index marker")
	}
	if !strings.Contains(contents, "<!-- AGENT:NAV") {
		t.Error("AGENTS.md should contain files block")
	}
}

func TestWriteFilesBlock_DedicatedLarge(t *testing.T) {
	dir := t.TempDir()

	entries := make([]FileEntry, 25) // 25 > 20 → dedicated file
	for i := range entries {
		entries[i] = FileEntry{RelPath: "file.md", Dir: "", Name: "file.md", Purpose: "a file"}
	}

	cfg := config.Defaults()
	cfg.IndexInlineMax = 20

	dest, err := WriteFilesBlock(dir, entries, cfg, false)
	if err != nil {
		t.Fatalf("WriteFilesBlock() error = %v", err)
	}

	if dest != filepath.Join(dir, "AGENTMAP.md") {
		t.Errorf("dest = %q, want AGENTMAP.md (dedicated for large projects)", dest)
	}

	// AGENTMAP.md should have the files block.
	data, err := os.ReadFile(filepath.Join(dir, "AGENTMAP.md"))
	if err != nil {
		t.Fatalf("read AGENTMAP.md: %v", err)
	}
	if !strings.Contains(string(data), "<!-- AGENT:NAV") {
		t.Error("AGENTMAP.md should contain files block")
	}

	// AGENTS.md should not be touched in dedicated mode.
	if _, err := os.Stat(filepath.Join(dir, "AGENTS.md")); !os.IsNotExist(err) {
		t.Error("AGENTS.md should not be created or modified in dedicated mode")
	}
}

func TestWriteFilesBlock_IdempotentInline(t *testing.T) {
	dir := t.TempDir()

	entries := []FileEntry{
		{RelPath: "README.md", Dir: "", Name: "README.md", Purpose: "project readme"},
	}

	cfg := config.Defaults()
	cfg.IndexInlineMax = 20

	// First write.
	if _, err := WriteFilesBlock(dir, entries, cfg, false); err != nil {
		t.Fatalf("first WriteFilesBlock() error = %v", err)
	}

	// Second write with updated entries.
	entries[0].Purpose = "updated readme purpose"
	if _, err := WriteFilesBlock(dir, entries, cfg, false); err != nil {
		t.Fatalf("second WriteFilesBlock() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	contents := string(data)

	// Should not duplicate markers.
	if count := strings.Count(contents, "<!-- agentmap:index -->"); count != 1 {
		t.Errorf("found %d opening markers, want 1 (idempotent)", count)
	}
	if count := strings.Count(contents, "<!-- /agentmap:index -->"); count != 1 {
		t.Errorf("found %d closing markers, want 1 (idempotent)", count)
	}
	// Updated purpose should be present.
	if !strings.Contains(contents, "updated readme purpose") {
		t.Error("updated purpose should be in file after second write")
	}
}

func TestWriteFilesBlock_DryRun(t *testing.T) {
	dir := t.TempDir()

	entries := []FileEntry{
		{RelPath: "README.md", Dir: "", Name: "README.md", Purpose: "project readme"},
	}

	cfg := config.Defaults()

	dest, err := WriteFilesBlock(dir, entries, cfg, true)
	if err != nil {
		t.Fatalf("WriteFilesBlock() dry-run error = %v", err)
	}

	// Dry-run: no files should be written.
	if _, err := os.Stat(filepath.Join(dir, "AGENTS.md")); !os.IsNotExist(err) {
		t.Error("dry-run should not create AGENTS.md")
	}
	if _, err := os.Stat(filepath.Join(dir, "AGENTMAP.md")); !os.IsNotExist(err) {
		t.Error("dry-run should not create AGENTMAP.md")
	}
	// But dest should still be returned to indicate what would be written.
	if dest == "" {
		t.Error("dry-run should still return the destination path")
	}
}

func TestWriteFilesBlock_BoundaryExactlyInlineMax(t *testing.T) {
	// When len(entries) == IndexInlineMax, the condition (<=) must route to inline
	// (AGENTS.md), not to the dedicated AGENTMAP.md path.
	dir := t.TempDir()

	const inlineMax = 20
	entries := make([]FileEntry, inlineMax) // exactly at the boundary
	for i := range entries {
		entries[i] = FileEntry{RelPath: "file.md", Dir: "", Name: "file.md", Purpose: "a file"}
	}

	cfg := config.Defaults()
	cfg.IndexInlineMax = inlineMax

	dest, err := WriteFilesBlock(dir, entries, cfg, false)
	if err != nil {
		t.Fatalf("WriteFilesBlock() error = %v", err)
	}

	wantDest := filepath.Join(dir, "AGENTS.md")
	if dest != wantDest {
		t.Errorf("dest = %q, want %q (exactly IndexInlineMax entries should go inline)", dest, wantDest)
	}

	// AGENTMAP.md must NOT be created at the boundary.
	if _, err := os.Stat(filepath.Join(dir, "AGENTMAP.md")); !os.IsNotExist(err) {
		t.Error("AGENTMAP.md should not exist when len(entries) == IndexInlineMax")
	}
}

// --- Integration tests using testdata/index-fixture ---

// TestFixture_BuildIndex_Classification runs BuildIndex against the fixture
// tree and verifies the expected classification counts:
//
//	Generated=2  (README.md, docs/api/endpoints.md — no nav block)
//	TaskFiles=4  (the two generated + docs/authentication.md + docs/api/rate-limiting.md — have ~)
//	Skipped=2    (CONTRIBUTING.md, docs/error-policy.md — fully reviewed)
func TestFixture_BuildIndex_Classification(t *testing.T) {
	src := fixtureDir(t)
	dir := t.TempDir()
	copyFixture(t, src, dir)

	cfg := config.Defaults()

	result, err := BuildIndex(dir, cfg, false, false)
	if err != nil {
		t.Fatalf("BuildIndex() error = %v", err)
	}

	if result.Generated != 2 {
		t.Errorf("Generated = %d, want 2 (README.md + docs/api/endpoints.md)", result.Generated)
	}
	if result.TaskFiles != 4 {
		t.Errorf("TaskFiles = %d, want 4", result.TaskFiles)
	}
	if result.Skipped != 2 {
		t.Errorf("Skipped = %d, want 2 (CONTRIBUTING.md + docs/error-policy.md)", result.Skipped)
	}
	if result.TaskPath == "" {
		t.Error("TaskPath should be set (tasks were found)")
	}
}

// TestFixture_BuildIndex_DryRun verifies no files are modified when --dry-run.
func TestFixture_BuildIndex_DryRun(t *testing.T) {
	src := fixtureDir(t)
	dir := t.TempDir()
	copyFixture(t, src, dir)

	cfg := config.Defaults()

	result, err := BuildIndex(dir, cfg, true, false)
	if err != nil {
		t.Fatalf("BuildIndex() dry-run error = %v", err)
	}

	if result.TaskPath != "" {
		t.Errorf("TaskPath = %q, want empty for dry-run", result.TaskPath)
	}

	// The two unindexed files must not have been modified.
	for _, f := range []string{"README.md", "docs/api/endpoints.md"} {
		data, err := os.ReadFile(filepath.Join(dir, f))
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		if strings.Contains(string(data), "AGENT:NAV") {
			t.Errorf("dry-run must not write nav block to %s", f)
		}
	}
}

// TestFixture_BuildIndex_SkeletonWritten verifies skeletons are actually
// written to the two unindexed files and have ~ prefixes.
func TestFixture_BuildIndex_SkeletonWritten(t *testing.T) {
	src := fixtureDir(t)
	dir := t.TempDir()
	copyFixture(t, src, dir)

	cfg := config.Defaults()

	if _, err := BuildIndex(dir, cfg, false, false); err != nil {
		t.Fatalf("BuildIndex() error = %v", err)
	}

	for _, f := range []string{"README.md", "docs/api/endpoints.md"} {
		data, err := os.ReadFile(filepath.Join(dir, f))
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		content := string(data)
		if !strings.Contains(content, "<!-- AGENT:NAV") {
			t.Errorf("%s: skeleton nav block not written", f)
		}
		if !strings.Contains(content, "~") {
			t.Errorf("%s: skeleton should have ~ prefix on auto-generated descriptions", f)
		}
	}
}

// TestFixture_BuildIndex_ReviewedFilesUnchanged verifies that fully-reviewed
// files are not modified even when BuildIndex runs.
func TestFixture_BuildIndex_ReviewedFilesUnchanged(t *testing.T) {
	src := fixtureDir(t)
	dir := t.TempDir()
	copyFixture(t, src, dir)

	cfg := config.Defaults()

	if _, err := BuildIndex(dir, cfg, false, false); err != nil {
		t.Fatalf("BuildIndex() error = %v", err)
	}

	// Read original content from source fixture.
	for _, f := range []string{"CONTRIBUTING.md", "docs/error-policy.md"} {
		orig, err := os.ReadFile(filepath.Join(src, f))
		if err != nil {
			t.Fatalf("read original %s: %v", f, err)
		}
		got, err := os.ReadFile(filepath.Join(dir, f))
		if err != nil {
			t.Fatalf("read result %s: %v", f, err)
		}
		if string(orig) != string(got) {
			t.Errorf("%s: fully-reviewed file was modified", f)
		}
	}
}

// TestFixture_BuildFilesBlock_Entries verifies that BuildFilesBlock collects
// the correct entries from the (pre-indexed) fixture tree.
// The two files without nav blocks are excluded; the four with nav blocks appear.
func TestFixture_BuildFilesBlock_Entries(t *testing.T) {
	src := fixtureDir(t)
	dir := t.TempDir()
	copyFixture(t, src, dir)

	cfg := config.Defaults()

	// First index so all files get nav blocks.
	if _, err := BuildIndex(dir, cfg, false, false); err != nil {
		t.Fatalf("BuildIndex() error = %v", err)
	}

	entries, err := BuildFilesBlock(dir, cfg)
	if err != nil {
		t.Fatalf("BuildFilesBlock() error = %v", err)
	}

	// All 6 files should now have nav blocks.
	if len(entries) != 6 {
		paths := make([]string, len(entries))
		for i, e := range entries {
			paths[i] = e.RelPath
		}
		t.Fatalf("len(entries) = %d, want 6; got: %v", len(entries), paths)
	}

	// Root-level files must come first.
	rootCount := 0
	for _, e := range entries {
		if e.Dir == "" {
			rootCount++
		}
	}
	if rootCount != 2 {
		t.Errorf("root-level entries = %d, want 2 (README.md + CONTRIBUTING.md)", rootCount)
	}

	// Verify no entry has a ~ prefix in its Purpose (BuildFilesBlock strips it).
	for _, e := range entries {
		if strings.HasPrefix(e.Purpose, "~") {
			t.Errorf("entry %s has ~ prefix in Purpose: %q", e.RelPath, e.Purpose)
		}
	}
}

// TestFixture_BuildFilesBlock_PreIndexed runs BuildFilesBlock on the fixture
// without running BuildIndex first. Only the 4 files that already have nav
// blocks should appear.
func TestFixture_BuildFilesBlock_PreIndexed(t *testing.T) {
	src := fixtureDir(t)
	dir := t.TempDir()
	copyFixture(t, src, dir)

	cfg := config.Defaults()

	entries, err := BuildFilesBlock(dir, cfg)
	if err != nil {
		t.Fatalf("BuildFilesBlock() error = %v", err)
	}

	// 4 files have nav blocks in the fixture (CONTRIBUTING.md, docs/authentication.md,
	// docs/error-policy.md, docs/api/rate-limiting.md).
	if len(entries) != 4 {
		paths := make([]string, len(entries))
		for i, e := range entries {
			paths[i] = e.RelPath
		}
		t.Fatalf("len(entries) = %d, want 4; got: %v", len(entries), paths)
	}

	// Check sort order: root first, then alphabetically by dir.
	wantOrder := []string{
		"CONTRIBUTING.md",
		"docs/authentication.md",
		"docs/error-policy.md",
		"docs/api/rate-limiting.md",
	}
	got := make([]string, len(entries))
	for i, e := range entries {
		got[i] = e.RelPath
	}
	// docs/ entries before docs/api/ entries — check prefix ordering.
	docsIdx := -1
	docsAPIIdx := -1
	for i, p := range got {
		if p == "docs/authentication.md" || p == "docs/error-policy.md" {
			if docsIdx < 0 {
				docsIdx = i
			}
		}
		if p == "docs/api/rate-limiting.md" {
			docsAPIIdx = i
		}
	}
	if docsIdx >= 0 && docsAPIIdx >= 0 && docsIdx > docsAPIIdx {
		t.Errorf("expected docs/ entries before docs/api/ entries; got order: %v", got)
	}
	_ = wantOrder // reference for human readability
}
