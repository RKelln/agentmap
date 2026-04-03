package initcmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RKelln/agentmap/internal/initcmd"
)

// mkDir creates a directory inside a temp dir.
func mkDir(t *testing.T, root, rel string) {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatalf("mkDir %s: %v", rel, err)
	}
}

// mkFile creates a file with given content inside root.
func mkFile(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkFile dir %s: %v", rel, err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("mkFile %s: %v", rel, err)
	}
}

func fileContent(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

// TestDetectTools verifies detection logic for all supported tool configs.
func TestDetectTools(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(root string)
		wantTools []string
	}{
		{
			name:      "no configs: default AGENTS.md",
			setup:     func(_ string) {},
			wantTools: []string{"agents-md"},
		},
		{
			name: "AGENTS.md exists",
			setup: func(root string) {
				mkFile(t, root, "AGENTS.md", "# Agents\n")
			},
			wantTools: []string{"agents-md"},
		},
		{
			name: "CLAUDE.md exists",
			setup: func(root string) {
				mkFile(t, root, "CLAUDE.md", "# Claude\n")
			},
			wantTools: []string{"claude-md"},
		},
		{
			name: ".cursor/rules/ exists",
			setup: func(root string) {
				mkDir(t, root, ".cursor/rules")
			},
			wantTools: []string{"cursor"},
		},
		{
			name: ".cursorrules exists (legacy cursor)",
			setup: func(root string) {
				mkFile(t, root, ".cursorrules", "# rules\n")
			},
			wantTools: []string{"cursor-legacy"},
		},
		{
			name: ".windsurf/rules/ exists",
			setup: func(root string) {
				mkDir(t, root, ".windsurf/rules")
			},
			wantTools: []string{"windsurf"},
		},
		{
			name: ".continue/rules/ exists",
			setup: func(root string) {
				mkDir(t, root, ".continue/rules")
			},
			wantTools: []string{"continue"},
		},
		{
			name: ".roo/rules/ exists",
			setup: func(root string) {
				mkDir(t, root, ".roo/rules")
			},
			wantTools: []string{"roo"},
		},
		{
			name: ".amazonq/rules/ exists",
			setup: func(root string) {
				mkDir(t, root, ".amazonq/rules")
			},
			wantTools: []string{"amazonq"},
		},
		{
			name: ".opencode/ exists",
			setup: func(root string) {
				mkDir(t, root, ".opencode")
			},
			wantTools: []string{"opencode"},
		},
		{
			name: ".aider.conf.yml exists",
			setup: func(root string) {
				mkFile(t, root, ".aider.conf.yml", "model: gpt-4\n")
			},
			wantTools: []string{"aider"},
		},
		{
			name: "multiple tools detected",
			setup: func(root string) {
				mkFile(t, root, "AGENTS.md", "# Agents\n")
				mkDir(t, root, ".cursor/rules")
				mkFile(t, root, "CLAUDE.md", "# Claude\n")
			},
			wantTools: []string{"agents-md", "claude-md", "cursor"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			tt.setup(root)

			got := initcmd.DetectTools(root)
			gotKeys := make(map[string]bool, len(got))
			for _, d := range got {
				gotKeys[d.ToolID] = true
			}

			for _, want := range tt.wantTools {
				if !gotKeys[want] {
					t.Errorf("expected tool %q to be detected; got %v", want, got)
				}
			}
			if len(got) != len(tt.wantTools) {
				t.Errorf("expected %d tools; got %d: %v", len(tt.wantTools), len(got), got)
			}
		})
	}
}

// TestApplyDryRun verifies dry-run produces a plan but no file writes.
func TestApplyDryRun(t *testing.T) {
	root := t.TempDir()
	// No pre-existing configs → default AGENTS.md creation.
	opts := initcmd.Options{
		Root:   root,
		DryRun: true,
		Yes:    true,
		NoHook: true,
	}

	plan, err := initcmd.Apply(opts)
	if err != nil {
		t.Fatalf("Apply dry-run: %v", err)
	}
	if len(plan.Actions) == 0 {
		t.Error("expected at least one plan action; got none")
	}

	// AGENTS.md must NOT exist after dry-run.
	if _, err := os.Stat(filepath.Join(root, "AGENTS.md")); err == nil {
		t.Error("AGENTS.md should not be created in dry-run mode")
	}
}

// TestApplyCreatesAgentsMD verifies the fallback creates AGENTS.md from template.
func TestApplyCreatesAgentsMD(t *testing.T) {
	root := t.TempDir()
	opts := initcmd.Options{
		Root:   root,
		DryRun: false,
		Yes:    true,
		NoHook: true,
	}

	plan, err := initcmd.Apply(opts)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if len(plan.Actions) == 0 {
		t.Error("expected at least one action")
	}

	content := fileContent(t, filepath.Join(root, "AGENTS.md"))
	if !strings.Contains(content, "<!-- agentmap:init -->") {
		t.Error("AGENTS.md missing <!-- agentmap:init --> marker")
	}
}

// TestApplyIdempotentAgentsMD verifies re-running skips already-configured files.
func TestApplyIdempotentAgentsMD(t *testing.T) {
	root := t.TempDir()
	// Pre-seed AGENTS.md with the marker.
	mkFile(t, root, "AGENTS.md", "# Agents\n<!-- agentmap:init -->\nstuff\n<!-- /agentmap:init -->\n")

	opts := initcmd.Options{
		Root:   root,
		DryRun: false,
		Yes:    true,
		NoHook: true,
	}

	plan, err := initcmd.Apply(opts)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	for _, a := range plan.Actions {
		if a.ToolID == "agents-md" && a.Status != initcmd.StatusSkipped {
			t.Errorf("expected agents-md action to be skipped; got %v", a.Status)
		}
	}
}

// TestApplyAppendsToCLAUDE verifies appending to CLAUDE.md.
func TestApplyAppendsToCLAUDE(t *testing.T) {
	root := t.TempDir()
	mkFile(t, root, "CLAUDE.md", "# Claude\n\nExisting content.\n")

	opts := initcmd.Options{
		Root:   root,
		DryRun: false,
		Yes:    true,
		NoHook: true,
	}

	_, err := initcmd.Apply(opts)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	content := fileContent(t, filepath.Join(root, "CLAUDE.md"))
	if !strings.Contains(content, "Existing content.") {
		t.Error("CLAUDE.md original content was lost")
	}
	if !strings.Contains(content, "<!-- agentmap:init -->") {
		t.Error("CLAUDE.md missing agentmap:init marker after append")
	}
}

// TestApplyWritesCursorRules verifies cursor creates new file under .cursor/rules/.
func TestApplyWritesCursorRules(t *testing.T) {
	root := t.TempDir()
	mkDir(t, root, ".cursor/rules")

	opts := initcmd.Options{
		Root:   root,
		DryRun: false,
		Yes:    true,
		NoHook: true,
	}

	_, err := initcmd.Apply(opts)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	target := filepath.Join(root, ".cursor", "rules", "agentmap.md")
	content := fileContent(t, target)
	if !strings.Contains(content, "<!-- agentmap:init -->") {
		t.Error(".cursor/rules/agentmap.md missing agentmap:init marker")
	}
	if !strings.Contains(content, "alwaysApply") {
		t.Error(".cursor/rules/agentmap.md missing cursor-specific frontmatter")
	}
}

// TestApplyIdempotentCursorRules verifies cursor file is skipped if marker exists.
func TestApplyIdempotentCursorRules(t *testing.T) {
	root := t.TempDir()
	mkDir(t, root, ".cursor/rules")
	mkFile(t, root, ".cursor/rules/agentmap.md", "<!-- agentmap:init -->\ncontent\n<!-- /agentmap:init -->\n")

	opts := initcmd.Options{
		Root:   root,
		DryRun: false,
		Yes:    true,
		NoHook: true,
	}

	plan, err := initcmd.Apply(opts)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	for _, a := range plan.Actions {
		if a.ToolID == "cursor" && a.Status != initcmd.StatusSkipped {
			t.Errorf("expected cursor action to be skipped; got %v", a.Status)
		}
	}
}

// TestApplyOpenCodeSkill verifies opencode creates skills/agentmap/SKILL.md.
func TestApplyOpenCodeSkill(t *testing.T) {
	root := t.TempDir()
	mkDir(t, root, ".opencode")

	opts := initcmd.Options{
		Root:   root,
		DryRun: false,
		Yes:    true,
		NoHook: true,
	}

	_, err := initcmd.Apply(opts)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	target := filepath.Join(root, ".opencode", "skills", "agentmap", "SKILL.md")
	content := fileContent(t, target)
	if !strings.Contains(content, "<!-- agentmap:init -->") {
		t.Error(".opencode/skills/agentmap/SKILL.md missing agentmap:init marker")
	}
}

// TestApplyAiderWarnOnly verifies aider detection produces a warning action, not a write.
func TestApplyAiderWarnOnly(t *testing.T) {
	root := t.TempDir()
	mkFile(t, root, ".aider.conf.yml", "model: gpt-4\n")

	opts := initcmd.Options{
		Root:   root,
		DryRun: false,
		Yes:    true,
		NoHook: true,
	}

	plan, err := initcmd.Apply(opts)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	var found bool
	for _, a := range plan.Actions {
		if a.ToolID == "aider" {
			found = true
			if a.Status != initcmd.StatusWarn {
				t.Errorf("aider action should be StatusWarn; got %v", a.Status)
			}
		}
	}
	if !found {
		t.Error("expected aider action in plan")
	}
}

// TestApplyHookGit verifies git hook is added when .git/hooks/ exists.
func TestApplyHookGit(t *testing.T) {
	root := t.TempDir()
	mkDir(t, root, ".git/hooks")

	opts := initcmd.Options{
		Root:   root,
		DryRun: false,
		Yes:    true,
		NoHook: false,
	}

	_, err := initcmd.Apply(opts)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	hookPath := filepath.Join(root, ".git", "hooks", "pre-commit")
	content := fileContent(t, hookPath)
	if !strings.Contains(content, "agentmap check") {
		t.Error("pre-commit hook missing 'agentmap check'")
	}
	if !strings.Contains(content, "# agentmap: validate") {
		t.Error("pre-commit hook missing '# agentmap: validate' guard marker")
	}
}

// TestApplyHookGitIdempotent verifies hook not added twice.
func TestApplyHookGitIdempotent(t *testing.T) {
	root := t.TempDir()
	mkDir(t, root, ".git/hooks")
	// Pre-seed with the guard already present.
	mkFile(t, root, ".git/hooks/pre-commit", "#!/bin/sh\n# agentmap: validate\nagentmap check .\n# /agentmap: validate\n")

	opts := initcmd.Options{
		Root:   root,
		DryRun: false,
		Yes:    true,
		NoHook: false,
	}

	_, err := initcmd.Apply(opts)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	content := fileContent(t, filepath.Join(root, ".git/hooks/pre-commit"))
	count := strings.Count(content, "# agentmap: validate")
	if count != 1 {
		t.Errorf("expected 1 occurrence of guard marker; got %d", count)
	}
}

// TestApplyHookPrecommitYAML verifies .pre-commit-config.yaml gets hook appended.
func TestApplyHookPrecommitYAML(t *testing.T) {
	root := t.TempDir()
	mkDir(t, root, ".git/hooks") // Make it a git-like repo too.
	mkFile(t, root, ".pre-commit-config.yaml", "repos:\n  - repo: https://example.com\n    hooks:\n      - id: existing\n")

	opts := initcmd.Options{
		Root:   root,
		DryRun: false,
		Yes:    true,
		NoHook: false,
	}

	_, err := initcmd.Apply(opts)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	content := fileContent(t, filepath.Join(root, ".pre-commit-config.yaml"))
	if !strings.Contains(content, "agentmap-check") {
		t.Error(".pre-commit-config.yaml missing agentmap-check hook entry")
	}
	if !strings.Contains(content, "# agentmap: validate") {
		t.Error(".pre-commit-config.yaml missing guard marker")
	}
}

// TestApplyNoHook verifies --no-hook skips hook installation.
func TestApplyNoHook(t *testing.T) {
	root := t.TempDir()
	mkDir(t, root, ".git/hooks")

	opts := initcmd.Options{
		Root:   root,
		DryRun: false,
		Yes:    true,
		NoHook: true,
	}

	_, err := initcmd.Apply(opts)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	hookPath := filepath.Join(root, ".git", "hooks", "pre-commit")
	if _, err := os.Stat(hookPath); err == nil {
		t.Error("pre-commit hook should not be created with --no-hook")
	}
}

// TestApplyToolFilter verifies --tool flag limits to a single tool.
func TestApplyToolFilter(t *testing.T) {
	root := t.TempDir()
	mkFile(t, root, "AGENTS.md", "# Agents\n")
	mkFile(t, root, "CLAUDE.md", "# Claude\n")

	opts := initcmd.Options{
		Root:       root,
		DryRun:     false,
		Yes:        true,
		NoHook:     true,
		ToolFilter: "claude",
	}

	plan, err := initcmd.Apply(opts)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	for _, a := range plan.Actions {
		if a.ToolID == "agents-md" {
			t.Errorf("expected agents-md to be skipped with --tool=claude; got action %v", a)
		}
	}
}

// TestPlanFormat verifies Plan.String() produces non-empty human-readable output.
func TestPlanFormat(t *testing.T) {
	root := t.TempDir()
	opts := initcmd.Options{
		Root:   root,
		DryRun: true,
		Yes:    true,
		NoHook: true,
	}

	plan, err := initcmd.Apply(opts)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	output := plan.String()
	if output == "" {
		t.Error("Plan.String() returned empty output")
	}
}
