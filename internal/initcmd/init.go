// Package initcmd implements the agentmap init; uninit; and uninstall commands.
// It detects agent tool configurations in a repository root; appends or creates
// agentmap instruction blocks; and optionally installs pre-commit hooks.
package initcmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/RKelln/agentmap/internal/templates"
)

// ActionStatus describes the outcome of an init action.
type ActionStatus int

const (
	// StatusPending means the action has not been executed yet (dry-run).
	StatusPending ActionStatus = iota
	// StatusDone means the action completed successfully.
	StatusDone
	// StatusSkipped means the target was already configured (idempotent).
	StatusSkipped
	// StatusWarn means a warning was issued (e.g. aider — manual config needed).
	StatusWarn
)

func (s ActionStatus) String() string {
	switch s {
	case StatusPending:
		return "pending"
	case StatusDone:
		return "done"
	case StatusSkipped:
		return "skipped"
	case StatusWarn:
		return "warn"
	default:
		return "unknown"
	}
}

// Action represents one planned or executed init step.
type Action struct {
	ToolID      string
	Description string
	Target      string
	Status      ActionStatus
	Message     string // extra info (warn text, skip reason, etc.)
}

func (a Action) String() string {
	tag := "[" + a.Status.String() + "]"
	switch a.Status {
	case StatusSkipped:
		return fmt.Sprintf("  %s Already configured: %s", tag, a.Target)
	case StatusWarn:
		return fmt.Sprintf("  %s %s: %s", tag, a.ToolID, a.Message)
	case StatusPending:
		return fmt.Sprintf("  %s Will %s: %s", tag, a.Description, a.Target)
	default:
		return fmt.Sprintf("  %s %s: %s", tag, a.Description, a.Target)
	}
}

// Plan is the result of an Apply call.
type Plan struct {
	Actions []Action
}

// String returns a human-readable summary of the plan.
func (p *Plan) String() string {
	if len(p.Actions) == 0 {
		return "Nothing to do.\n"
	}
	var b strings.Builder
	for _, a := range p.Actions {
		b.WriteString(a.String())
		b.WriteString("\n")
	}
	return b.String()
}

// Detection describes a detected tool configuration.
type Detection struct {
	ToolID      string // canonical ID used internally
	DisplayName string
}

// Options controls Apply behavior.
type Options struct {
	Root       string
	DryRun     bool
	Yes        bool   // skip confirmation prompt
	NoHook     bool   // skip hook installation
	ToolFilter string // if non-empty: only process this tool name
}

// toolFilterMatch reports whether a tool matches the optional filter.
// Matching is case-insensitive and checks against ToolID and DisplayName.
func toolFilterMatch(d Detection, filter string) bool {
	if filter == "" {
		return true
	}
	f := strings.ToLower(filter)
	return strings.ToLower(d.ToolID) == f ||
		strings.Contains(strings.ToLower(d.ToolID), f) ||
		strings.Contains(strings.ToLower(d.DisplayName), f)
}

// DetectTools scans root for known agent tool config files/directories
// and returns the set of detected tools. If nothing is detected, it
// returns a single "agents-md" entry (the best cross-tool default).
func DetectTools(root string) []Detection {
	var found []Detection

	add := func(id, name string) {
		found = append(found, Detection{ToolID: id, DisplayName: name})
	}

	dirExists := func(rel string) bool {
		info, err := os.Stat(filepath.Join(root, rel))
		return err == nil && info.IsDir()
	}
	fileExists := func(rel string) bool {
		_, err := os.Stat(filepath.Join(root, rel))
		return err == nil
	}

	if fileExists("AGENTS.md") {
		add("agents-md", "AGENTS.md")
	}
	if fileExists("CLAUDE.md") {
		add("claude-md", "CLAUDE.md")
	}
	if dirExists(".cursor/rules") {
		add("cursor", ".cursor/rules/")
	}
	if fileExists(".cursorrules") {
		add("cursor-legacy", ".cursorrules")
	}
	if dirExists(".windsurf/rules") {
		add("windsurf", ".windsurf/rules/")
	}
	if dirExists(".continue/rules") {
		add("continue", ".continue/rules/")
	}
	if dirExists(".roo/rules") {
		add("roo", ".roo/rules/")
	}
	if dirExists(".amazonq/rules") {
		add("amazonq", ".amazonq/rules/")
	}
	if dirExists(".opencode") {
		add("opencode", ".opencode/")
	}
	if fileExists(".aider.conf.yml") {
		add("aider", ".aider.conf.yml")
	}

	// Fallback: no configs found → default to creating AGENTS.md.
	if len(found) == 0 {
		add("agents-md", "AGENTS.md (create)")
	}

	return found
}

const initMarker = "<!-- agentmap:init -->"

// containsMarker reports whether data contains the idempotency marker.
func containsMarker(data []byte) bool {
	return bytes.Contains(data, []byte(initMarker))
}

// fileContainsMarker reads a file and checks for the marker.
// Returns false if the file doesn't exist.
func fileContainsMarker(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return containsMarker(data)
}

// appendToFile appends content to path, creating it if it doesn't exist.
func appendToFile(path string, content []byte) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	// Ensure we start on a new line.
	info, _ := f.Stat()
	if info.Size() > 0 {
		// Read the last byte to check if file ends with newline.
		existing, err := os.ReadFile(path)
		if err == nil && len(existing) > 0 && existing[len(existing)-1] != '\n' {
			if _, err := f.WriteString("\n"); err != nil {
				return err
			}
		}
	}

	_, err = f.Write(content)
	return err
}

// writeNewFile writes content to path, creating parent directories as needed.
func writeNewFile(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}
	return os.WriteFile(path, content, 0o644)
}

// Apply runs the init command with the given options. It returns the plan
// describing what was (or would be) done.
func Apply(opts Options) (*Plan, error) {
	root := opts.Root
	if root == "" {
		root = "."
	}

	detections := DetectTools(root)

	// Apply tool filter.
	if opts.ToolFilter != "" {
		var filtered []Detection
		for _, d := range detections {
			if toolFilterMatch(d, opts.ToolFilter) {
				filtered = append(filtered, d)
			}
		}
		detections = filtered
	}

	plan := &Plan{}

	for _, d := range detections {
		actions, err := planToolAction(root, d, opts.DryRun)
		if err != nil {
			return nil, fmt.Errorf("plan %s: %w", d.ToolID, err)
		}
		plan.Actions = append(plan.Actions, actions...)
	}

	// Hook setup (unless --no-hook or dry-run-only).
	if !opts.NoHook {
		hookActions, err := planHookActions(root, opts.DryRun)
		if err != nil {
			return nil, fmt.Errorf("plan hooks: %w", err)
		}
		plan.Actions = append(plan.Actions, hookActions...)
	}

	return plan, nil
}

// planToolAction returns the Action(s) for a single detected tool.
func planToolAction(root string, d Detection, dryRun bool) ([]Action, error) {
	switch d.ToolID {
	case "agents-md":
		return planAppendOrCreate(root, "AGENTS.md", "agents.md.tmpl", d.ToolID, "append/create AGENTS.md", dryRun)

	case "claude-md":
		return planAppendOrCreate(root, "CLAUDE.md", "agents.md.tmpl", d.ToolID, "append to CLAUDE.md", dryRun)

	case "cursor":
		target := filepath.Join(".cursor", "rules", "agentmap.md")
		return planWriteNew(root, target, "cursor.md.tmpl", d.ToolID, "create "+target, dryRun)

	case "cursor-legacy":
		return planAppendOrCreate(root, ".cursorrules", "agents.md.tmpl", d.ToolID, "append to .cursorrules", dryRun)

	case "windsurf":
		target := filepath.Join(".windsurf", "rules", "agentmap.md")
		return planWriteNew(root, target, "windsurf.md.tmpl", d.ToolID, "create "+target, dryRun)

	case "continue":
		target := filepath.Join(".continue", "rules", "agentmap.md")
		return planWriteNew(root, target, "continue.md.tmpl", d.ToolID, "create "+target, dryRun)

	case "roo":
		target := filepath.Join(".roo", "rules", "agentmap.md")
		return planWriteNew(root, target, "roo.md.tmpl", d.ToolID, "create "+target, dryRun)

	case "amazonq":
		target := filepath.Join(".amazonq", "rules", "agentmap.md")
		return planWriteNew(root, target, "amazonq.md.tmpl", d.ToolID, "create "+target, dryRun)

	case "opencode":
		target := filepath.Join(".opencode", "skills", "agentmap", "SKILL.md")
		return planWriteNew(root, target, "opencode-skill.md.tmpl", d.ToolID, "create "+target, dryRun)

	case "aider":
		return []Action{{
			ToolID:      d.ToolID,
			Description: "warn",
			Target:      ".aider.conf.yml",
			Status:      StatusWarn,
			Message:     "manual config needed; add 'read: [AGENTS.md]' to .aider.conf.yml",
		}}, nil

	default:
		return nil, fmt.Errorf("unknown tool ID: %s", d.ToolID)
	}
}

// planAppendOrCreate handles append-to-existing-file or create-if-missing actions.
func planAppendOrCreate(root, relPath, tmplName, toolID, description string, dryRun bool) ([]Action, error) {
	absPath := filepath.Join(root, relPath)

	// Idempotency: check if marker already present.
	if fileContainsMarker(absPath) {
		return []Action{{
			ToolID:      toolID,
			Description: description,
			Target:      relPath,
			Status:      StatusSkipped,
			Message:     "already configured",
		}}, nil
	}

	// Load template.
	content, err := templates.Get(tmplName)
	if err != nil {
		return nil, fmt.Errorf("get template %s: %w", tmplName, err)
	}

	status := StatusPending
	if !dryRun {
		if err := appendToFile(absPath, content); err != nil {
			return nil, err
		}
		status = StatusDone
	}

	return []Action{{
		ToolID:      toolID,
		Description: description,
		Target:      relPath,
		Status:      status,
	}}, nil
}

// planWriteNew handles creating a new file in a tool-specific directory.
func planWriteNew(root, relPath, tmplName, toolID, description string, dryRun bool) ([]Action, error) {
	absPath := filepath.Join(root, relPath)

	// Idempotency: file exists and contains marker.
	if fileContainsMarker(absPath) {
		return []Action{{
			ToolID:      toolID,
			Description: description,
			Target:      relPath,
			Status:      StatusSkipped,
			Message:     "already configured",
		}}, nil
	}

	content, err := templates.Get(tmplName)
	if err != nil {
		return nil, fmt.Errorf("get template %s: %w", tmplName, err)
	}

	status := StatusPending
	if !dryRun {
		if err := writeNewFile(absPath, content); err != nil {
			return nil, err
		}
		status = StatusDone
	}

	return []Action{{
		ToolID:      toolID,
		Description: description,
		Target:      relPath,
		Status:      status,
	}}, nil
}

// planHookActions detects hook infrastructure and returns relevant hook actions.
func planHookActions(root string, dryRun bool) ([]Action, error) {
	var actions []Action

	preCommitYAML := filepath.Join(root, ".pre-commit-config.yaml")
	huskyDir := filepath.Join(root, ".husky")
	gitHooksDir := filepath.Join(root, ".git", "hooks")
	lefthook := filepath.Join(root, ".lefthook.yml")

	// .pre-commit-config.yaml takes priority.
	if _, err := os.Stat(preCommitYAML); err == nil {
		a, err := planPreCommitYAML(preCommitYAML, dryRun)
		if err != nil {
			return nil, err
		}
		actions = append(actions, a)
	}

	// .husky/ directory.
	if info, err := os.Stat(huskyDir); err == nil && info.IsDir() {
		a, err := planHuskyHook(huskyDir, dryRun)
		if err != nil {
			return nil, err
		}
		actions = append(actions, a)
	}

	// .git/hooks/ (plain git hooks) — only if no husky and no pre-commit.
	if _, err := os.Stat(preCommitYAML); os.IsNotExist(err) {
		if _, herr := os.Stat(huskyDir); os.IsNotExist(herr) {
			if info, err := os.Stat(gitHooksDir); err == nil && info.IsDir() {
				a, err := planGitHook(gitHooksDir, dryRun)
				if err != nil {
					return nil, err
				}
				actions = append(actions, a)
			}
		}
	}

	// Lefthook warning.
	if _, err := os.Stat(lefthook); err == nil {
		actions = append(actions, Action{
			ToolID:      "hook-lefthook",
			Description: "warn",
			Target:      ".lefthook.yml",
			Status:      StatusWarn,
			Message:     "add agentmap check to your lefthook config manually",
		})
	}

	return actions, nil
}

// planPreCommitYAML appends the agentmap hook entry to .pre-commit-config.yaml.
func planPreCommitYAML(path string, dryRun bool) (Action, error) {
	existing, err := os.ReadFile(path)
	if err != nil {
		return Action{}, fmt.Errorf("read %s: %w", path, err)
	}

	if bytes.Contains(existing, []byte("agentmap check")) {
		return Action{
			ToolID:  "hook-precommit",
			Target:  ".pre-commit-config.yaml",
			Status:  StatusSkipped,
			Message: "already configured",
		}, nil
	}

	hookContent, err := templates.Get("hook-precommit.yml.tmpl")
	if err != nil {
		return Action{}, err
	}

	status := StatusPending
	if !dryRun {
		if err := appendToFile(path, append([]byte("\n"), hookContent...)); err != nil {
			return Action{}, err
		}
		status = StatusDone
	}

	return Action{
		ToolID:      "hook-precommit",
		Description: "append hook to .pre-commit-config.yaml",
		Target:      ".pre-commit-config.yaml",
		Status:      status,
	}, nil
}

// planHuskyHook appends the agentmap guard block to .husky/pre-commit.
func planHuskyHook(huskyDir string, dryRun bool) (Action, error) {
	hookPath := filepath.Join(huskyDir, "pre-commit")

	if fileContainsString(hookPath, "agentmap check") {
		return Action{
			ToolID:  "hook-husky",
			Target:  ".husky/pre-commit",
			Status:  StatusSkipped,
			Message: "already configured",
		}, nil
	}

	hookContent, err := templates.Get("hook-git.sh.tmpl")
	if err != nil {
		return Action{}, err
	}

	status := StatusPending
	if !dryRun {
		if err := appendToFile(hookPath, hookContent); err != nil {
			return Action{}, err
		}
		status = StatusDone
	}

	return Action{
		ToolID:      "hook-husky",
		Description: "append hook to .husky/pre-commit",
		Target:      ".husky/pre-commit",
		Status:      status,
	}, nil
}

// planGitHook installs or appends to .git/hooks/pre-commit.
func planGitHook(hooksDir string, dryRun bool) (Action, error) {
	hookPath := filepath.Join(hooksDir, "pre-commit")

	if fileContainsString(hookPath, "agentmap check") {
		return Action{
			ToolID:  "hook-git",
			Target:  ".git/hooks/pre-commit",
			Status:  StatusSkipped,
			Message: "already configured",
		}, nil
	}

	hookContent, err := templates.Get("hook-git.sh.tmpl")
	if err != nil {
		return Action{}, err
	}

	status := StatusPending
	if !dryRun {
		if err := appendToFile(hookPath, hookContent); err != nil {
			return Action{}, err
		}
		// Ensure executable.
		if err := os.Chmod(hookPath, 0o755); err != nil {
			return Action{}, fmt.Errorf("chmod %s: %w", hookPath, err)
		}
		status = StatusDone
	}

	return Action{
		ToolID:      "hook-git",
		Description: "create/append .git/hooks/pre-commit",
		Target:      ".git/hooks/pre-commit",
		Status:      status,
	}, nil
}

// fileContainsString reports whether path exists and contains substr.
func fileContainsString(path, substr string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return bytes.Contains(data, []byte(substr))
}
