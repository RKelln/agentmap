package initcmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// UninitOptions controls Uninit behavior.
type UninitOptions struct {
	Root   string
	DryRun bool
	Yes    bool
}

// UninitPlan holds the actions that uninit will perform.
type UninitPlan struct {
	Actions []UninitAction
}

// UninitActionKind describes what uninit will do to a target.
type UninitActionKind int

const (
	// KindRemoveBlock removes the agentmap:init block from an existing file.
	KindRemoveBlock UninitActionKind = iota
	// KindDeleteFile deletes a file created entirely by agentmap init.
	KindDeleteFile
	// KindRemoveHookBlock removes the agentmap guard block from a hook file.
	KindRemoveHookBlock
	// KindDeleteHookFile deletes a hook file that only contains the agentmap block.
	KindDeleteHookFile
	// KindRemovePreCommitEntry removes the agentmap-check entry from .pre-commit-config.yaml.
	KindRemovePreCommitEntry
)

// UninitAction describes a single uninit step.
type UninitAction struct {
	Kind    UninitActionKind
	Target  string // relative path from root
	Status  ActionStatus
	Message string
}

func (a UninitAction) String() string {
	tag := "[" + a.Status.String() + "]"
	switch a.Kind {
	case KindDeleteFile, KindDeleteHookFile:
		return fmt.Sprintf("  %s Delete: %s", tag, a.Target)
	case KindRemoveBlock, KindRemoveHookBlock:
		return fmt.Sprintf("  %s Remove agentmap block from: %s", tag, a.Target)
	case KindRemovePreCommitEntry:
		return fmt.Sprintf("  %s Remove agentmap-check entry from: %s", tag, a.Target)
	default:
		return fmt.Sprintf("  %s %s", tag, a.Target)
	}
}

// String returns a human-readable summary of the uninit plan.
func (p *UninitPlan) String() string {
	if len(p.Actions) == 0 {
		return "agentmap not initialized in this project.\n"
	}
	var b strings.Builder
	for _, a := range p.Actions {
		b.WriteString(a.String())
		b.WriteString("\n")
	}
	return b.String()
}

const (
	initStartMarker = "<!-- agentmap:init -->"
	initEndMarker   = "<!-- /agentmap:init -->"
	hookStartMarker = "# agentmap: validate"
	hookEndMarker   = "# /agentmap: validate"
)

// Uninit scans root for agentmap-injected content and removes it.
func Uninit(opts UninitOptions) (*UninitPlan, error) {
	root := opts.Root
	if root == "" {
		root = "."
	}

	plan := &UninitPlan{}

	// Scan for markdown files containing the init block.
	markdownTargets := []string{
		"AGENTS.md",
		"CLAUDE.md",
		".cursorrules",
	}
	for _, rel := range markdownTargets {
		abs := filepath.Join(root, rel)
		data, err := os.ReadFile(abs)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", rel, err)
		}
		if !bytes.Contains(data, []byte(initStartMarker)) {
			continue
		}
		actions, err := planMarkdownBlockRemoval(root, rel, data, opts.DryRun)
		if err != nil {
			return nil, err
		}
		plan.Actions = append(plan.Actions, actions...)
	}

	// Scan for tool-specific created files.
	createdFiles := []struct {
		rel     string
		cleanup string // optional parent dir to remove if empty
	}{
		{filepath.Join(".cursor", "rules", "agentmap.md"), ""},
		{filepath.Join(".windsurf", "rules", "agentmap.md"), ""},
		{filepath.Join(".continue", "rules", "agentmap.md"), ""},
		{filepath.Join(".roo", "rules", "agentmap.md"), ""},
		{filepath.Join(".amazonq", "rules", "agentmap.md"), ""},
		{filepath.Join(".opencode", "skills", "agentmap", "SKILL.md"), filepath.Join(".opencode", "skills", "agentmap")},
	}
	for _, cf := range createdFiles {
		abs := filepath.Join(root, cf.rel)
		data, err := os.ReadFile(abs)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", cf.rel, err)
		}
		if !bytes.Contains(data, []byte(initStartMarker)) {
			// Safety: don't delete files we didn't create.
			continue
		}
		action := UninitAction{
			Kind:   KindDeleteFile,
			Target: cf.rel,
			Status: StatusPending,
		}
		if !opts.DryRun {
			if err := os.Remove(abs); err != nil && !os.IsNotExist(err) {
				return nil, fmt.Errorf("remove %s: %w", cf.rel, err)
			}
			// Remove empty parent dirs if applicable.
			if cf.cleanup != "" {
				_ = removeEmptyDir(filepath.Join(root, cf.cleanup))
			}
			action.Status = StatusDone
		}
		plan.Actions = append(plan.Actions, action)
	}

	// Hook removal: .pre-commit-config.yaml
	preCommitYAML := filepath.Join(root, ".pre-commit-config.yaml")
	if data, err := os.ReadFile(preCommitYAML); err == nil {
		if bytes.Contains(data, []byte("agentmap check")) {
			a, err := planPreCommitYAMLRemoval(preCommitYAML, data, opts.DryRun)
			if err != nil {
				return nil, err
			}
			plan.Actions = append(plan.Actions, a)
		}
	}

	// Hook removal: .git/hooks/pre-commit
	gitHook := filepath.Join(root, ".git", "hooks", "pre-commit")
	if data, err := os.ReadFile(gitHook); err == nil {
		if bytes.Contains(data, []byte(hookStartMarker)) {
			a, err := planHookRemoval(gitHook, ".git/hooks/pre-commit", data, opts.DryRun)
			if err != nil {
				return nil, err
			}
			plan.Actions = append(plan.Actions, a)
		}
	}

	// Hook removal: .husky/pre-commit
	huskyHook := filepath.Join(root, ".husky", "pre-commit")
	if data, err := os.ReadFile(huskyHook); err == nil {
		if bytes.Contains(data, []byte(hookStartMarker)) {
			a, err := planHookRemoval(huskyHook, ".husky/pre-commit", data, opts.DryRun)
			if err != nil {
				return nil, err
			}
			plan.Actions = append(plan.Actions, a)
		}
	}

	return plan, nil
}

// planMarkdownBlockRemoval handles removing the init block from a markdown file.
func planMarkdownBlockRemoval(root, rel string, data []byte, dryRun bool) ([]UninitAction, error) {
	cleaned, err := removeInitBlock(data)
	if err != nil {
		return nil, fmt.Errorf("remove block from %s: %w", rel, err)
	}

	// Determine action kind: if nothing meaningful left, delete the file.
	if isEffectivelyEmpty(cleaned) {
		action := UninitAction{
			Kind:   KindDeleteFile,
			Target: rel,
			Status: StatusPending,
		}
		if !dryRun {
			abs := filepath.Join(root, rel)
			if err := os.Remove(abs); err != nil && !os.IsNotExist(err) {
				return nil, fmt.Errorf("remove %s: %w", rel, err)
			}
			action.Status = StatusDone
		}
		return []UninitAction{action}, nil
	}

	action := UninitAction{
		Kind:   KindRemoveBlock,
		Target: rel,
		Status: StatusPending,
	}
	if !dryRun {
		abs := filepath.Join(root, rel)
		if err := os.WriteFile(abs, cleaned, 0o644); err != nil {
			return nil, fmt.Errorf("write %s: %w", rel, err)
		}
		action.Status = StatusDone
	}
	return []UninitAction{action}, nil
}

// removeInitBlock removes the <!-- agentmap:init --> ... <!-- /agentmap:init --> block
// from data and trims resulting double blank lines.
func removeInitBlock(data []byte) ([]byte, error) {
	content := string(data)

	startIdx := strings.Index(content, initStartMarker)
	if startIdx < 0 {
		return data, nil
	}
	endIdx := strings.Index(content, initEndMarker)
	if endIdx < 0 {
		return nil, fmt.Errorf("found %s but no closing %s", initStartMarker, initEndMarker)
	}
	endIdx += len(initEndMarker)

	// Include the trailing newline if present.
	if endIdx < len(content) && content[endIdx] == '\n' {
		endIdx++
	}

	before := content[:startIdx]
	after := content[endIdx:]

	result := before + after

	// Collapse triple+ blank lines to double.
	for strings.Contains(result, "\n\n\n") {
		result = strings.ReplaceAll(result, "\n\n\n", "\n\n")
	}

	return []byte(result), nil
}

// isEffectivelyEmpty reports whether data is empty or contains only whitespace/shebang.
func isEffectivelyEmpty(data []byte) bool {
	s := strings.TrimSpace(string(data))
	if s == "" {
		return true
	}
	// A file with only a shebang line is effectively empty.
	lines := strings.Split(s, "\n")
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed == "" || strings.HasPrefix(trimmed, "#!") {
			continue
		}
		return false
	}
	return true
}

// planPreCommitYAMLRemoval removes the agentmap guard block from .pre-commit-config.yaml.
// The guard block is delimited by "# agentmap: validate" and "# /agentmap: validate".
func planPreCommitYAMLRemoval(path string, data []byte, dryRun bool) (UninitAction, error) {
	rel, _ := filepath.Rel(filepath.Dir(filepath.Dir(path)), path)
	action := UninitAction{
		Kind:   KindRemovePreCommitEntry,
		Target: ".pre-commit-config.yaml",
		Status: StatusPending,
	}

	cleaned := removeGuardBlock(string(data), hookStartMarker, hookEndMarker)

	if !dryRun {
		if err := os.WriteFile(path, []byte(cleaned), 0o644); err != nil {
			return action, fmt.Errorf("write %s: %w", rel, err)
		}
		action.Status = StatusDone
	}
	return action, nil
}

// planHookRemoval removes the agentmap guard block from a hook file.
func planHookRemoval(abs, rel string, data []byte, dryRun bool) (UninitAction, error) {
	cleaned := removeGuardBlock(string(data), hookStartMarker, hookEndMarker)

	if isEffectivelyEmpty([]byte(cleaned)) {
		action := UninitAction{
			Kind:   KindDeleteHookFile,
			Target: rel,
			Status: StatusPending,
		}
		if !dryRun {
			if err := os.Remove(abs); err != nil && !os.IsNotExist(err) {
				return action, fmt.Errorf("remove %s: %w", rel, err)
			}
			action.Status = StatusDone
		}
		return action, nil
	}

	action := UninitAction{
		Kind:   KindRemoveHookBlock,
		Target: rel,
		Status: StatusPending,
	}
	if !dryRun {
		if err := os.WriteFile(abs, []byte(cleaned), 0o755); err != nil {
			return action, fmt.Errorf("write %s: %w", rel, err)
		}
		action.Status = StatusDone
	}
	return action, nil
}

// removeGuardBlock removes a delimited block (start marker to end marker inclusive)
// from content, collapsing resulting blank lines.
func removeGuardBlock(content, start, end string) string {
	startIdx := strings.Index(content, start)
	if startIdx < 0 {
		return content
	}
	endIdx := strings.Index(content, end)
	if endIdx < 0 {
		return content
	}
	endIdx += len(end)
	if endIdx < len(content) && content[endIdx] == '\n' {
		endIdx++
	}

	// Trim preceding blank line if one exists right before the block.
	trimStart := startIdx
	if trimStart > 0 && content[trimStart-1] == '\n' {
		j := trimStart - 1
		for j > 0 && content[j-1] == '\n' {
			j--
		}
		// Only trim if the preceding chars are newlines (blank lines).
		if j < trimStart-1 {
			trimStart--
		}
	}

	result := content[:trimStart] + content[endIdx:]
	for strings.Contains(result, "\n\n\n") {
		result = strings.ReplaceAll(result, "\n\n\n", "\n\n")
	}
	return result
}

// removeEmptyDir removes dir if it is empty.
func removeEmptyDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return os.Remove(dir)
	}
	return nil
}

// --- uninstall: install method detection ---

// InstallMethod describes how agentmap was installed.
type InstallMethod int

const (
	// InstallMethodDirect means the binary was placed directly (curl/wget/cp).
	InstallMethodDirect InstallMethod = iota
	// InstallMethodGo means installed via `go install`.
	InstallMethodGo
	// InstallMethodHomebrew means installed via Homebrew.
	InstallMethodHomebrew
	// InstallMethodScoop means installed via Scoop (Windows).
	InstallMethodScoop
)

func (m InstallMethod) String() string {
	switch m {
	case InstallMethodDirect:
		return "direct"
	case InstallMethodGo:
		return "go"
	case InstallMethodHomebrew:
		return "homebrew"
	case InstallMethodScoop:
		return "scoop"
	default:
		return "unknown"
	}
}

// DetectInstallMethod determines how agentmap was installed based on the binary path.
// exePath is the resolved executable path.
// gopath is $GOPATH (or "" if unset).
// gobin is $GOBIN (or "" if unset).
func DetectInstallMethod(exePath, gopath, gobin string) InstallMethod {
	// Normalize to forward slashes for cross-platform matching.
	normalized := strings.ReplaceAll(filepath.ToSlash(exePath), "\\", "/")

	if strings.Contains(normalized, "/Cellar/") || strings.Contains(normalized, "/homebrew/") {
		return InstallMethodHomebrew
	}
	if strings.Contains(strings.ToLower(normalized), "/scoop/") {
		return InstallMethodScoop
	}

	// Check GOBIN first (more specific).
	if gobin != "" {
		gobinNorm := filepath.ToSlash(gobin)
		if strings.HasPrefix(normalized, gobinNorm) {
			return InstallMethodGo
		}
	}

	// Check GOPATH/bin.
	if gopath != "" {
		gopathBin := filepath.ToSlash(filepath.Join(gopath, "bin"))
		if strings.HasPrefix(normalized, gopathBin) {
			return InstallMethodGo
		}
	}

	return InstallMethodDirect
}

// UninstallInstructions returns the human-readable instructions for uninstalling
// via a package manager, or an empty string for direct installs.
func UninstallInstructions(method InstallMethod) string {
	switch method {
	case InstallMethodHomebrew:
		return "Installed via Homebrew. Run: brew uninstall agentmap"
	case InstallMethodScoop:
		return "Installed via Scoop. Run: scoop uninstall agentmap"
	case InstallMethodGo:
		return "Installed via go install. Run: go clean -i github.com/RKelln/agentmap/cmd/agentmap"
	default:
		return ""
	}
}
