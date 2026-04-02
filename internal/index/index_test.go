package index

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ryankelln/agentmap/internal/config"
)

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
	if !strings.Contains(contents, "docs/nav-writing-guide.md") {
		t.Error("task list should reference the nav writing guide")
	}
	if !strings.Contains(contents, "Progress:") {
		t.Error("task list should have Progress counter")
	}
	if !strings.Contains(contents, "docs/auth.md") {
		t.Error("task list should contain the file path")
	}
	if !strings.Contains(contents, "- [ ] purpose:") {
		t.Error("task list should have unchecked purpose checkbox")
	}
	if !strings.Contains(contents, "- [ ] sections") {
		t.Error("task list should have unchecked sections checkbox")
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

	// AGENTS.md should have a pointer line.
	agentsData, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	if !strings.Contains(string(agentsData), "See AGENTMAP.md for the full file index.") {
		t.Error("AGENTS.md should contain pointer to AGENTMAP.md")
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
