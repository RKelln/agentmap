package initcmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ryankelln/agentmap/internal/initcmd"
)

// --- uninit tests ---

// TestUninitNoMarkersFound verifies uninit reports nothing to do when no markers present.
func TestUninitNoMarkersFound(t *testing.T) {
	root := t.TempDir()
	mkFile(t, root, "AGENTS.md", "# Some content\nNo agentmap here.\n")

	opts := initcmd.UninitOptions{
		Root:   root,
		DryRun: false,
		Yes:    true,
	}

	plan, err := initcmd.Uninit(opts)
	if err != nil {
		t.Fatalf("Uninit: %v", err)
	}
	if len(plan.Actions) != 0 {
		t.Errorf("expected no actions when no markers found; got %v", plan.Actions)
	}
}

// TestUninitRemovesBlockFromAgentsMD verifies the init block is removed from AGENTS.md.
func TestUninitRemovesBlockFromAgentsMD(t *testing.T) {
	root := t.TempDir()
	mkFile(t, root, "AGENTS.md", "# Agents\n\nExisting content.\n\n<!-- agentmap:init -->\n## Reading\nstuff\n<!-- /agentmap:init -->\n")

	opts := initcmd.UninitOptions{
		Root:   root,
		DryRun: false,
		Yes:    true,
	}

	_, err := initcmd.Uninit(opts)
	if err != nil {
		t.Fatalf("Uninit: %v", err)
	}

	content := fileContent(t, filepath.Join(root, "AGENTS.md"))
	if strings.Contains(content, "<!-- agentmap:init -->") {
		t.Error("AGENTS.md still contains agentmap:init marker after uninit")
	}
	if !strings.Contains(content, "Existing content.") {
		t.Error("AGENTS.md lost existing content during uninit")
	}
}

// TestUninitDeletesEmptyFileAfterBlockRemoval verifies that a file containing
// only the agentmap block (created by init when no prior content) is deleted.
func TestUninitDeletesEmptyFileAfterBlockRemoval(t *testing.T) {
	root := t.TempDir()
	// File contains only the agentmap block (init created it from scratch).
	mkFile(t, root, "AGENTS.md", "<!-- agentmap:init -->\n## Reading\nstuff\n<!-- /agentmap:init -->\n")

	opts := initcmd.UninitOptions{
		Root:   root,
		DryRun: false,
		Yes:    true,
	}

	_, err := initcmd.Uninit(opts)
	if err != nil {
		t.Fatalf("Uninit: %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, "AGENTS.md")); err == nil {
		t.Error("AGENTS.md should be deleted when it only contained the agentmap block")
	}
}

// TestUninitDeletesCreatedRulesFile verifies .cursor/rules/agentmap.md is deleted.
func TestUninitDeletesCreatedRulesFile(t *testing.T) {
	root := t.TempDir()
	mkFile(t, root, ".cursor/rules/agentmap.md", "---\nalwaysApply: false\n---\n<!-- agentmap:init -->\ncontent\n<!-- /agentmap:init -->\n")

	opts := initcmd.UninitOptions{
		Root:   root,
		DryRun: false,
		Yes:    true,
	}

	_, err := initcmd.Uninit(opts)
	if err != nil {
		t.Fatalf("Uninit: %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, ".cursor", "rules", "agentmap.md")); err == nil {
		t.Error(".cursor/rules/agentmap.md should be deleted by uninit")
	}
}

// TestUninitDeletesOpenCodeSkillDir verifies .opencode/skills/agentmap/ is cleaned up.
func TestUninitDeletesOpenCodeSkillDir(t *testing.T) {
	root := t.TempDir()
	mkFile(t, root, ".opencode/skills/agentmap/SKILL.md", "---\nname: agentmap\n---\n<!-- agentmap:init -->\ncontent\n<!-- /agentmap:init -->\n")

	opts := initcmd.UninitOptions{
		Root:   root,
		DryRun: false,
		Yes:    true,
	}

	_, err := initcmd.Uninit(opts)
	if err != nil {
		t.Fatalf("Uninit: %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, ".opencode", "skills", "agentmap")); err == nil {
		t.Error(".opencode/skills/agentmap/ dir should be removed after uninit")
	}
}

// TestUninitRemovesGitHook verifies agentmap block is removed from pre-commit hook.
func TestUninitRemovesGitHook(t *testing.T) {
	root := t.TempDir()
	mkDir(t, root, ".git/hooks")
	mkFile(t, root, ".git/hooks/pre-commit",
		"#!/bin/sh\necho hello\n# agentmap: validate\nagentmap check .\n# /agentmap: validate\n")

	opts := initcmd.UninitOptions{
		Root:   root,
		DryRun: false,
		Yes:    true,
	}

	_, err := initcmd.Uninit(opts)
	if err != nil {
		t.Fatalf("Uninit: %v", err)
	}

	content := fileContent(t, filepath.Join(root, ".git", "hooks", "pre-commit"))
	if strings.Contains(content, "agentmap check") {
		t.Error("pre-commit hook still contains agentmap check after uninit")
	}
	if !strings.Contains(content, "echo hello") {
		t.Error("pre-commit hook lost existing content after uninit")
	}
}

// TestUninitDeletesHookWhenOnlyAgentmapContent verifies hook file is deleted when
// it only contains the agentmap guard block (and optional shebang).
func TestUninitDeletesHookWhenOnlyAgentmapContent(t *testing.T) {
	root := t.TempDir()
	mkDir(t, root, ".git/hooks")
	mkFile(t, root, ".git/hooks/pre-commit",
		"#!/bin/sh\n# agentmap: validate\nagentmap check .\n# /agentmap: validate\n")

	opts := initcmd.UninitOptions{
		Root:   root,
		DryRun: false,
		Yes:    true,
	}

	_, err := initcmd.Uninit(opts)
	if err != nil {
		t.Fatalf("Uninit: %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, ".git", "hooks", "pre-commit")); err == nil {
		t.Error("pre-commit hook should be deleted when it only contained agentmap block")
	}
}

// TestUninitRemovesPreCommitYAMLEntry verifies agentmap-check is removed from .pre-commit-config.yaml.
func TestUninitRemovesPreCommitYAMLEntry(t *testing.T) {
	root := t.TempDir()
	mkFile(t, root, ".pre-commit-config.yaml",
		"repos:\n  - repo: https://example.com\n    hooks:\n      - id: existing\n\n# agentmap: validate\n- repo: local\n  hooks:\n    - id: agentmap-check\n      name: Validate AGENT:NAV blocks\n      entry: agentmap check\n      language: system\n      types: [markdown]\n# /agentmap: validate\n")

	opts := initcmd.UninitOptions{
		Root:   root,
		DryRun: false,
		Yes:    true,
	}

	_, err := initcmd.Uninit(opts)
	if err != nil {
		t.Fatalf("Uninit: %v", err)
	}

	content := fileContent(t, filepath.Join(root, ".pre-commit-config.yaml"))
	if strings.Contains(content, "agentmap-check") {
		t.Error(".pre-commit-config.yaml still contains agentmap-check after uninit")
	}
	if !strings.Contains(content, "existing") {
		t.Error(".pre-commit-config.yaml lost existing content after uninit")
	}
}

// TestUninitDryRun verifies dry-run does not modify files.
func TestUninitDryRun(t *testing.T) {
	root := t.TempDir()
	original := "# Agents\n\nContent.\n\n<!-- agentmap:init -->\nstuff\n<!-- /agentmap:init -->\n"
	mkFile(t, root, "AGENTS.md", original)

	opts := initcmd.UninitOptions{
		Root:   root,
		DryRun: true,
		Yes:    true,
	}

	plan, err := initcmd.Uninit(opts)
	if err != nil {
		t.Fatalf("Uninit dry-run: %v", err)
	}
	if len(plan.Actions) == 0 {
		t.Error("expected actions in dry-run plan; got none")
	}

	// File must be unchanged.
	content := fileContent(t, filepath.Join(root, "AGENTS.md"))
	if content != original {
		t.Errorf("dry-run modified file; got:\n%s", content)
	}
}

// TestUninitTrimsDoubleBlankLines verifies no double blank lines left after block removal.
func TestUninitTrimsDoubleBlankLines(t *testing.T) {
	root := t.TempDir()
	mkFile(t, root, "AGENTS.md", "# Agents\n\nContent.\n\n\n<!-- agentmap:init -->\nstuff\n<!-- /agentmap:init -->\n")

	opts := initcmd.UninitOptions{
		Root:   root,
		DryRun: false,
		Yes:    true,
	}

	_, err := initcmd.Uninit(opts)
	if err != nil {
		t.Fatalf("Uninit: %v", err)
	}

	content := fileContent(t, filepath.Join(root, "AGENTS.md"))
	if strings.Contains(content, "\n\n\n") {
		t.Errorf("uninit left triple blank lines:\n%q", content)
	}
}

// --- uninstall tests ---

// TestDetectInstallMethod_GoPath verifies go install detection based on GOPATH.
func TestDetectInstallMethod_GoPath(t *testing.T) {
	gopath := t.TempDir()
	fakeBin := filepath.Join(gopath, "bin", "agentmap")
	mkFile(t, gopath, "bin/agentmap", "fake binary")

	method := initcmd.DetectInstallMethod(fakeBin, gopath, "")
	if method != initcmd.InstallMethodGo {
		t.Errorf("expected InstallMethodGo; got %v", method)
	}
}

// TestDetectInstallMethod_Homebrew verifies Homebrew detection via /Cellar/ in path.
func TestDetectInstallMethod_Homebrew(t *testing.T) {
	fakePath := "/usr/local/Cellar/agentmap/0.1.0/bin/agentmap"
	method := initcmd.DetectInstallMethod(fakePath, "/home/user/go", "")
	if method != initcmd.InstallMethodHomebrew {
		t.Errorf("expected InstallMethodHomebrew; got %v", method)
	}
}

// TestDetectInstallMethod_Scoop verifies Scoop detection via \scoop\ in path.
func TestDetectInstallMethod_Scoop(t *testing.T) {
	fakePath := `C:\Users\user\scoop\apps\agentmap\current\agentmap.exe`
	method := initcmd.DetectInstallMethod(fakePath, `C:\Users\user\go`, "")
	if method != initcmd.InstallMethodScoop {
		t.Errorf("expected InstallMethodScoop; got %v", method)
	}
}

// TestDetectInstallMethod_Direct verifies direct install for unknown paths.
func TestDetectInstallMethod_Direct(t *testing.T) {
	fakePath := "/usr/local/bin/agentmap"
	method := initcmd.DetectInstallMethod(fakePath, "/home/user/go", "")
	if method != initcmd.InstallMethodDirect {
		t.Errorf("expected InstallMethodDirect; got %v", method)
	}
}
