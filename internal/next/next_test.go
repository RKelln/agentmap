package next_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RKelln/agentmap/internal/next"
)

const authMD = "docs/auth.md"

// writeFile creates a file at path with the given content, making parent dirs.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// taskList builds a minimal index-tasks.md with the given entries.
// Each entry is {relPath, checked}.
func makeTaskList(t *testing.T, dir string, entries []struct {
	relPath string
	checked bool
},
) string {
	t.Helper()
	var b strings.Builder
	b.WriteString("# agentmap index tasks\n\nProgress: 0/2 files complete\n\n")
	b.WriteString("## Your job\n\nSome preamble text here.\n\n---\n\n")
	b.WriteString("## Rules (quick ref)\n\n- `purpose`: summary\n\n---\n\n")
	for _, e := range entries {
		lineCount := 50
		b.WriteString("## " + e.relPath + " (" + itoa(lineCount) + " lines)\n\n")
		if e.checked {
			b.WriteString("- [x]\n\n")
		} else {
			b.WriteString("- [ ]\n\n")
		}
	}
	b.WriteString("---\n\n# Appendix\n\nGuide content here.\n")

	taskListPath := filepath.Join(dir, ".agentmap", "index-tasks.md")
	writeFile(t, taskListPath, b.String())
	return taskListPath
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 10)
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	return string(buf)
}

// --- FindTaskList tests ---

func TestFindTaskList_FindsInCurrentDir(t *testing.T) {
	dir := t.TempDir()
	taskListPath := filepath.Join(dir, ".agentmap", "index-tasks.md")
	writeFile(t, taskListPath, "# tasks\n")

	got, err := next.FindTaskList(dir)
	if err != nil {
		t.Fatalf("FindTaskList() error = %v", err)
	}
	if got != taskListPath {
		t.Errorf("got %q, want %q", got, taskListPath)
	}
}

func TestFindTaskList_FindsInParentDir(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "docs", "api")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	taskListPath := filepath.Join(dir, ".agentmap", "index-tasks.md")
	writeFile(t, taskListPath, "# tasks\n")

	got, err := next.FindTaskList(subdir)
	if err != nil {
		t.Fatalf("FindTaskList() error = %v", err)
	}
	if got != taskListPath {
		t.Errorf("got %q, want %q", got, taskListPath)
	}
}

func TestFindTaskList_ErrorWhenNotFound(t *testing.T) {
	dir := t.TempDir() // no .agentmap/ here
	_, err := next.FindTaskList(dir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- Next tests ---

func TestNext_FirstUnchecked(t *testing.T) {
	dir := t.TempDir()
	taskListPath := makeTaskList(t, dir, []struct {
		relPath string
		checked bool
	}{
		{authMD, false},
		{"docs/errors.md", false},
	})
	// Write a stub markdown file with a nav block.
	writeFile(t, filepath.Join(dir, authMD), `<!-- AGENT:NAV
purpose:~token lifecycle authentication
nav[1]{s,n,name,about}:
10,20,##Token Exchange,~OAuth2 code-for-token flow
-->

# Auth
`)

	task, err := next.Next(taskListPath, 0)
	if err != nil {
		t.Fatalf("Next() error = %v", err)
	}
	if task == nil {
		t.Fatal("Next() returned nil task, expected first entry")
	}
	if task.RelPath != authMD {
		t.Errorf("RelPath = %q, want %s", task.RelPath, authMD)
	}
	if !strings.Contains(task.NavBlockRaw, "purpose:~token lifecycle authentication") {
		t.Errorf("NavBlockRaw does not contain expected content: %q", task.NavBlockRaw)
	}
}

func TestNext_SkipsCheckedEntries(t *testing.T) {
	dir := t.TempDir()
	taskListPath := makeTaskList(t, dir, []struct {
		relPath string
		checked bool
	}{
		{authMD, true},            // already checked
		{"docs/errors.md", false}, // next one
	})
	writeFile(t, filepath.Join(dir, "docs/errors.md"), `<!-- AGENT:NAV
purpose:~error handling policy
-->

# Errors
`)

	task, err := next.Next(taskListPath, 0)
	if err != nil {
		t.Fatalf("Next() error = %v", err)
	}
	if task == nil {
		t.Fatal("Next() returned nil, expected docs/errors.md")
	}
	if task.RelPath != "docs/errors.md" {
		t.Errorf("RelPath = %q, want docs/errors.md", task.RelPath)
	}
}

func TestNext_ReturnsNilWhenAllChecked(t *testing.T) {
	dir := t.TempDir()
	taskListPath := makeTaskList(t, dir, []struct {
		relPath string
		checked bool
	}{
		{authMD, true},
		{"docs/errors.md", true},
	})

	task, err := next.Next(taskListPath, 0)
	if err != nil {
		t.Fatalf("Next() error = %v", err)
	}
	if task != nil {
		t.Errorf("Next() returned %+v, want nil (all done)", task)
	}
}

func TestNext_SkipAdvancesToNextEntry(t *testing.T) {
	dir := t.TempDir()
	taskListPath := makeTaskList(t, dir, []struct {
		relPath string
		checked bool
	}{
		{authMD, false},
		{"docs/errors.md", false},
		{"docs/rate-limiting.md", false},
	})
	for _, f := range []string{authMD, "docs/errors.md", "docs/rate-limiting.md"} {
		writeFile(t, filepath.Join(dir, f), "<!-- AGENT:NAV\npurpose:~stub\n-->\n\n# Title\n")
	}

	task0, _ := next.Next(taskListPath, 0)
	task1, _ := next.Next(taskListPath, 1)
	task2, _ := next.Next(taskListPath, 2)

	if task0 == nil || task0.RelPath != authMD {
		t.Errorf("skip=0: got %v, want %s", task0, authMD)
	}
	if task1 == nil || task1.RelPath != "docs/errors.md" {
		t.Errorf("skip=1: got %v, want docs/errors.md", task1)
	}
	if task2 == nil || task2.RelPath != "docs/rate-limiting.md" {
		t.Errorf("skip=2: got %v, want docs/rate-limiting.md", task2)
	}
}

func TestNext_SkipBeyondListReturnsNil(t *testing.T) {
	dir := t.TempDir()
	taskListPath := makeTaskList(t, dir, []struct {
		relPath string
		checked bool
	}{
		{authMD, false},
	})

	task, err := next.Next(taskListPath, 1) // skip past the only entry
	if err != nil {
		t.Fatalf("Next() error = %v", err)
	}
	if task != nil {
		t.Errorf("expected nil for skip beyond list, got %+v", task)
	}
}

func TestNext_PreambleHeadingsIgnored(t *testing.T) {
	dir := t.TempDir()
	// Task list with preamble headings that look nothing like file entries.
	taskContent := `# agentmap index tasks

Progress: 0/1 files complete

## Your job

Do stuff.

## Rules (quick ref)

- purpose: under 10 words

---

## Example

Before/after.

---

## docs/auth.md (46 lines)

- [ ]

` + "```" + `
<!-- AGENT:NAV
purpose:~token lifecycle
-->
` + "```" + `

---

# Appendix: Nav Writing Guide

Content here.
`
	taskListPath := filepath.Join(dir, ".agentmap", "index-tasks.md")
	writeFile(t, taskListPath, taskContent)
	writeFile(t, filepath.Join(dir, authMD), "<!-- AGENT:NAV\npurpose:~token lifecycle\n-->\n\n# Auth\n")

	task, err := next.Next(taskListPath, 0)
	if err != nil {
		t.Fatalf("Next() error = %v", err)
	}
	if task == nil {
		t.Fatal("Next() returned nil, expected docs/auth.md")
	}
	if task.RelPath != authMD {
		t.Errorf("RelPath = %q, want %s", task.RelPath, authMD)
	}
}

func TestNext_MissingFileStillReturnsTask(t *testing.T) {
	dir := t.TempDir()
	taskListPath := makeTaskList(t, dir, []struct {
		relPath string
		checked bool
	}{
		{"docs/missing.md", false},
	})
	// Don't create the file — nav block should be empty string.

	task, err := next.Next(taskListPath, 0)
	if err != nil {
		t.Fatalf("Next() error = %v", err)
	}
	if task == nil {
		t.Fatal("Next() returned nil, expected task with missing file")
	}
	if task.RelPath != "docs/missing.md" {
		t.Errorf("RelPath = %q, want docs/missing.md", task.RelPath)
	}
	if task.NavBlockRaw != "" {
		t.Errorf("NavBlockRaw = %q, want empty for missing file", task.NavBlockRaw)
	}
}

// --- RenderPrompt tests ---

func TestRenderPrompt_ContainsFilePath(t *testing.T) {
	task := &next.Task{
		RelPath:     authMD,
		AbsPath:     "/repo/docs/auth.md",
		NavBlockRaw: "<!-- AGENT:NAV\npurpose:~token lifecycle\n-->",
		RepoRoot:    "/repo",
	}
	got := next.RenderPrompt(task)
	if !strings.Contains(got, authMD) {
		t.Errorf("prompt does not contain file path: %q", got)
	}
	// update is no longer a manual step — next handles it automatically.
	if strings.Contains(got, "agentmap update") {
		t.Errorf("prompt should not contain 'agentmap update' (handled by next): %q", got)
	}
	if !strings.Contains(got, "agentmap next") {
		t.Errorf("prompt does not contain next command: %q", got)
	}
}

func TestRenderPrompt_IncludesNavBlock(t *testing.T) {
	task := &next.Task{
		RelPath:     authMD,
		NavBlockRaw: "<!-- AGENT:NAV\npurpose:~token lifecycle authentication\n-->",
		RepoRoot:    "/repo",
	}
	got := next.RenderPrompt(task)
	if !strings.Contains(got, "purpose:~token lifecycle authentication") {
		t.Errorf("prompt does not include nav block content: %q", got)
	}
}

func TestRenderPrompt_NoNavBlockOmitsSection(t *testing.T) {
	task := &next.Task{
		RelPath:     "docs/missing.md",
		NavBlockRaw: "",
		RepoRoot:    "/repo",
	}
	got := next.RenderPrompt(task)
	if strings.Contains(got, "Current nav block") {
		t.Errorf("prompt should not include nav block section when empty: %q", got)
	}
}

// --- RenderDone tests ---

func TestRenderDone_ContainsCheckCommand(t *testing.T) {
	got := next.RenderDone("/repo")
	if !strings.Contains(got, "agentmap check") {
		t.Errorf("done message does not contain check command: %q", got)
	}
}

// --- RenderPrompt instruction tests ---

func TestRenderPrompt_NoUpdateStep(t *testing.T) {
	task := &next.Task{
		RelPath:     authMD,
		NavBlockRaw: "<!-- AGENT:NAV\npurpose:~stub\n-->",
		RepoRoot:    "/repo",
	}
	got := next.RenderPrompt(task)
	// Should NOT tell the agent to run agentmap update separately.
	if strings.Contains(got, "agentmap update") {
		t.Errorf("prompt should not contain 'agentmap update': %q", got)
	}
	// Should tell the agent to run agentmap next.
	if !strings.Contains(got, "agentmap next") {
		t.Errorf("prompt should contain 'agentmap next': %q", got)
	}
}

// --- WriteState / FlushState tests ---

func TestWriteState_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	taskListPath := filepath.Join(dir, ".agentmap", "index-tasks.md")
	writeFile(t, taskListPath, "# tasks\n")

	if err := next.WriteState(taskListPath, []string{authMD, "docs/errors.md"}); err != nil {
		t.Fatalf("WriteState() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".agentmap", "next-state"))
	if err != nil {
		t.Fatalf("read next-state: %v", err)
	}
	got := string(data)
	if !strings.Contains(got, authMD) {
		t.Errorf("state file missing %q: %q", authMD, got)
	}
	if !strings.Contains(got, "docs/errors.md") {
		t.Errorf("state file missing docs/errors.md: %q", got)
	}
}

func TestWriteState_NilClearsFile(t *testing.T) {
	dir := t.TempDir()
	taskListPath := filepath.Join(dir, ".agentmap", "index-tasks.md")
	writeFile(t, taskListPath, "# tasks\n")
	// Write state then clear it.
	if err := next.WriteState(taskListPath, []string{authMD}); err != nil {
		t.Fatalf("WriteState() error = %v", err)
	}
	if err := next.WriteState(taskListPath, nil); err != nil {
		t.Fatalf("WriteState(nil) error = %v", err)
	}
	// File should be gone.
	_, err := os.Stat(filepath.Join(dir, ".agentmap", "next-state"))
	if !os.IsNotExist(err) {
		t.Errorf("expected next-state to be removed, err = %v", err)
	}
}

func TestFlushState_NoStateFile_IsNoop(t *testing.T) {
	dir := t.TempDir()
	taskListPath := filepath.Join(dir, ".agentmap", "index-tasks.md")
	writeFile(t, taskListPath, "# tasks\n")

	result, err := next.FlushState(taskListPath, dir)
	if err != nil {
		t.Fatalf("FlushState() error = %v", err)
	}
	if result.Blocked {
		t.Errorf("expected not blocked")
	}
}

func TestFlushState_BlocksWhenTildeRemains(t *testing.T) {
	dir := t.TempDir()
	taskListPath := makeTaskList(t, dir, []struct {
		relPath string
		checked bool
	}{
		{authMD, false},
	})
	// Write file with ~ still present.
	writeFile(t, filepath.Join(dir, authMD), "<!-- AGENT:NAV\npurpose:~token lifecycle authentication\n-->\n\n# Auth\n")

	if err := next.WriteState(taskListPath, []string{authMD}); err != nil {
		t.Fatalf("WriteState() error = %v", err)
	}

	result, err := next.FlushState(taskListPath, dir)
	if err != nil {
		t.Fatalf("FlushState() error = %v", err)
	}
	if !result.Blocked {
		t.Errorf("expected Blocked=true when ~ remains")
	}
	if result.BlockedPath != authMD {
		t.Errorf("BlockedPath = %q, want %s", result.BlockedPath, authMD)
	}
	// Entry should still be unchecked.
	data, _ := os.ReadFile(taskListPath)
	if !strings.Contains(string(data), "- [ ]") {
		t.Errorf("task should still be unchecked when blocked")
	}
}

func TestFlushState_ChecksOffCleanFile(t *testing.T) {
	dir := t.TempDir()
	taskListPath := makeTaskList(t, dir, []struct {
		relPath string
		checked bool
	}{
		{authMD, false},
		{"docs/errors.md", false},
	})
	// Write file with NO ~ remaining.
	writeFile(t, filepath.Join(dir, authMD), "<!-- AGENT:NAV\npurpose:token lifecycle authentication\n-->\n\n# Auth\n")

	if err := next.WriteState(taskListPath, []string{authMD}); err != nil {
		t.Fatalf("WriteState() error = %v", err)
	}

	result, err := next.FlushState(taskListPath, dir)
	if err != nil {
		t.Fatalf("FlushState() error = %v", err)
	}
	if result.Blocked {
		t.Errorf("expected not blocked for clean file")
	}

	// Entry should be checked off.
	data, _ := os.ReadFile(taskListPath)
	if !strings.Contains(string(data), "- [x]") {
		t.Errorf("task should be checked off after flush")
	}
	// Second entry should still be unchecked.
	if strings.Count(string(data), "- [ ]") != 1 {
		t.Errorf("expected exactly 1 unchecked entry remaining")
	}
	// State file should be empty/cleared.
	statePath := filepath.Join(dir, ".agentmap", "next-state")
	stateData, _ := os.ReadFile(statePath)
	if strings.TrimSpace(string(stateData)) != "" {
		t.Errorf("state file should be cleared after flush, got %q", string(stateData))
	}
}

func TestFlushState_MissingFileSkips(t *testing.T) {
	dir := t.TempDir()
	taskListPath := makeTaskList(t, dir, []struct {
		relPath string
		checked bool
	}{
		{"docs/gone.md", false},
	})
	// Don't create the file.

	if err := next.WriteState(taskListPath, []string{"docs/gone.md"}); err != nil {
		t.Fatalf("WriteState() error = %v", err)
	}

	result, err := next.FlushState(taskListPath, dir)
	if err != nil {
		t.Fatalf("FlushState() error = %v", err)
	}
	// Missing file should not block.
	if result.Blocked {
		t.Errorf("missing file should not block flush")
	}
}

func TestFlushState_UnparseableNavBlock_WarnsAndSkips(t *testing.T) {
	dir := t.TempDir()
	taskListPath := makeTaskList(t, dir, []struct {
		relPath string
		checked bool
	}{
		{authMD, false},
	})
	// Write file with unclosed nav block (no -->).
	writeFile(t, filepath.Join(dir, authMD), "<!-- AGENT:NAV\npurpose:test\n\n# Auth\n")

	if err := next.WriteState(taskListPath, []string{authMD}); err != nil {
		t.Fatalf("WriteState() error = %v", err)
	}

	result, err := next.FlushState(taskListPath, dir)
	if err != nil {
		t.Fatalf("FlushState() error = %v", err)
	}
	// Unclosed nav block: should now be treated as corrupted and block progress.
	if !result.Blocked {
		t.Errorf("expected blocked for unparseable nav block")
	}
	if result.BlockedPath != authMD {
		t.Errorf("BlockedPath = %q, want %q", result.BlockedPath, authMD)
	}
	// Task should NOT be checked off.
	data, _ := os.ReadFile(taskListPath)
	if strings.Contains(string(data), "- [x]") {
		t.Errorf("task should NOT be checked off when nav block is unparseable")
	}
	if !strings.Contains(string(data), "- [ ]") {
		t.Errorf("task should still be unchecked")
	}
}
