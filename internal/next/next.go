// Package next implements the agentmap next command: find the next unchecked
// task in index-tasks.md, run update+check-off on any previously-emitted files,
// and print a self-contained single-file prompt for a small-model agent.
//
// State is tracked in .agentmap/next-state (one relPath per line). On each
// invocation, next flushes the state — running update+check-off on every path
// listed — then emits the next N unchecked entries and records them as the new
// state.
package next

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/RKelln/agentmap/internal/config"
	"github.com/RKelln/agentmap/internal/index"
	"github.com/RKelln/agentmap/internal/navblock"
	"github.com/RKelln/agentmap/internal/update"
)

// Task is a single unchecked entry from index-tasks.md.
type Task struct {
	// RelPath is the file path relative to the repo root.
	RelPath string
	// AbsPath is the absolute path to the file.
	AbsPath string
	// NavBlockRaw is the raw AGENT:NAV block text read from the file.
	NavBlockRaw string
	// RepoRoot is the root directory of the repo.
	RepoRoot string
}

// stateFileName is the name of the state file inside .agentmap/.
const stateFileName = "next-state"

// statePath returns the path to the next-state file.
func statePath(taskListPath string) string {
	return filepath.Join(filepath.Dir(taskListPath), stateFileName)
}

// FindTaskList searches upward from startDir for .agentmap/index-tasks.md.
// Returns the absolute path or an error if not found.
func FindTaskList(startDir string) (string, error) {
	abs, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("next: resolve dir: %w", err)
	}
	for {
		candidate := filepath.Join(abs, ".agentmap", "index-tasks.md")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			break
		}
		abs = parent
	}
	return "", fmt.Errorf("next: no .agentmap/index-tasks.md found searching upward from %s", startDir)
}

// FlushResult is the result of flushing the state file.
type FlushResult struct {
	// Blocked is set if a file still has ~ descriptions and we should stop.
	Blocked bool
	// BlockedPath is the relPath of the file that blocked progress.
	BlockedPath string
}

// FlushState reads the next-state file and runs update+check-off on each path.
// Returns a FlushResult. If any file still has ~ descriptions, Blocked is set
// and the caller should stop and warn the user.
// Missing files are warned about but do not block progress.
func FlushState(taskListPath, repoRoot string) (FlushResult, error) {
	sp := statePath(taskListPath)
	data, err := os.ReadFile(sp)
	if os.IsNotExist(err) {
		return FlushResult{}, nil // nothing to flush
	}
	if err != nil {
		return FlushResult{}, fmt.Errorf("next: read state: %w", err)
	}

	paths := parseState(string(data))
	if len(paths) == 0 {
		return FlushResult{}, nil
	}

	cfg, _ := loadConfig(repoRoot)
	taskList := index.TaskListPath(repoRoot)

	for _, relPath := range paths {
		absPath := filepath.Join(repoRoot, relPath)

		// Check if file still has ~ descriptions.
		if hasRemainingAuto(absPath) {
			return FlushResult{Blocked: true, BlockedPath: relPath}, nil
		}

		// File is clean: run update to refresh line numbers.
		if _, statErr := os.Stat(absPath); statErr == nil {
			if _, updErr := update.File(absPath, cfg, false, true); updErr != nil {
				// Non-fatal: warn via stderr but continue.
				fmt.Fprintf(os.Stderr, "warning: %s: update: %v\n", relPath, updErr)
			}
		} else {
			fmt.Fprintf(os.Stderr, "warning: %s: file not found, skipping\n", relPath)
		}

		// Check off the entry in the task list.
		if err := index.CheckOffTaskEntry(taskList, absPath, relPath); err != nil {
			fmt.Fprintf(os.Stderr, "warning: %s: check-off: %v\n", relPath, err)
		}
	}

	// Clear the state file — all paths flushed successfully.
	if err := os.WriteFile(sp, []byte(""), 0o644); err != nil {
		return FlushResult{}, fmt.Errorf("next: clear state: %w", err)
	}
	return FlushResult{}, nil
}

// WriteState writes the given relPaths to the next-state file.
func WriteState(taskListPath string, relPaths []string) error {
	if len(relPaths) == 0 {
		return clearState(taskListPath)
	}
	content := strings.Join(relPaths, "\n") + "\n"
	return os.WriteFile(statePath(taskListPath), []byte(content), 0o644)
}

// clearState removes or empties the state file.
func clearState(taskListPath string) error {
	sp := statePath(taskListPath)
	if err := os.Remove(sp); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("next: clear state: %w", err)
	}
	return nil
}

// parseState splits a state file into relPaths, ignoring blank lines.
func parseState(content string) []string {
	var paths []string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			paths = append(paths, line)
		}
	}
	return paths
}

// hasRemainingAuto returns true if the file at path has any ~ descriptions.
func hasRemainingAuto(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false // missing file doesn't block
	}
	pr := navblock.ParseNavBlock(string(data))
	if !pr.Found {
		return false
	}
	if navblock.IsAutoGenerated(pr.Block.Purpose) {
		return true
	}
	for _, e := range pr.Block.Nav {
		if navblock.IsAutoGenerated(e.About) {
			return true
		}
	}
	return false
}

// loadConfig loads agentmap.yml from the repo root, falling back to defaults.
func loadConfig(repoRoot string) (config.Config, error) {
	cfgPath := filepath.Join(repoRoot, "agentmap.yml")
	return config.Load(cfgPath)
}

// Next reads taskListPath, skips the first skip unchecked entries, and returns
// the (skip+1)th unchecked entry with its nav block. Returns (nil, nil) when
// all tasks are done.
func Next(taskListPath string, skip int) (*Task, error) {
	data, err := os.ReadFile(taskListPath)
	if err != nil {
		return nil, fmt.Errorf("next: read task list: %w", err)
	}
	content := string(data)

	lines := strings.Split(content, "\n")
	repoRoot := filepath.Dir(filepath.Dir(taskListPath)) // .agentmap/ -> root

	type candidate struct {
		relPath string
	}

	var current *candidate
	found := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			rest := line[3:]
			// File entry headings end with " (N lines)" or " (1 line)".
			if parenIdx := strings.LastIndex(rest, " ("); parenIdx >= 0 {
				suffix := rest[parenIdx:]
				if strings.HasSuffix(suffix, " lines)") || strings.HasSuffix(suffix, " line)") {
					relPath := rest[:parenIdx]
					if strings.HasSuffix(relPath, ".md") {
						current = &candidate{relPath: relPath}
					} else {
						current = nil
					}
				} else {
					current = nil
				}
			} else {
				current = nil
			}
			continue
		}
		if current != nil && line == "- [ ]" {
			if found >= skip {
				absPath := filepath.Join(repoRoot, current.relPath)
				navBlockRaw := readNavBlock(absPath)
				return &Task{
					RelPath:     current.relPath,
					AbsPath:     absPath,
					NavBlockRaw: navBlockRaw,
					RepoRoot:    repoRoot,
				}, nil
			}
			found++
			current = nil
		}
		if current != nil && line == "- [x]" {
			current = nil
		}
	}

	return nil, nil // all done
}

// readNavBlock reads the raw AGENT:NAV block text from a file.
// Returns empty string if the file can't be read or has no nav block.
func readNavBlock(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	pr := navblock.ParseNavBlock(string(data))
	if !pr.Found {
		return ""
	}
	lines := strings.Split(string(data), "\n")
	if pr.Start < 1 || pr.End < 1 || pr.End > len(lines) {
		return ""
	}
	return strings.Join(lines[pr.Start-1:pr.End], "\n")
}

// RenderPrompt formats a Task as a prompt for a small-model agent.
func RenderPrompt(t *Task) string {
	var b strings.Builder

	b.WriteString("Rewrite the nav block descriptions for: ")
	b.WriteString(t.RelPath)
	b.WriteString("\n\n")

	if t.NavBlockRaw != "" {
		b.WriteString("Current nav block (from the file):\n\n")
		b.WriteString("```\n")
		b.WriteString(t.NavBlockRaw)
		b.WriteString("\n```\n\n")
	}

	b.WriteString("Instructions:\n\n")
	b.WriteString("1. Open `" + t.RelPath + "` and read it.\n")
	b.WriteString("2. Rewrite every `~`-prefixed `purpose` and `about` value.\n")
	b.WriteString("   - Remove the `~` prefix.\n")
	b.WriteString("   - Replace keyword noise with a concise human description.\n")
	b.WriteString("   - `purpose`: one-line file summary; under 10 words; semicolons not commas.\n")
	b.WriteString("   - `about`: one-line section summary; don't restate the heading; under 10 words.\n")
	b.WriteString("   - Never edit `s`, `n`, `nav[N]`, `see[N]`, or line numbers.\n")
	b.WriteString("3. Optionally add `see` entries for closely related files.\n")
	b.WriteString("4. Save the file.\n")
	b.WriteString("5. Run: `agentmap next` to advance and get the next file.\n")

	return b.String()
}

// RenderDone formats the completion message when all tasks are checked off.
func RenderDone(repoRoot string) string {
	return "All tasks complete. Run: agentmap check " + repoRoot + "\n"
}

// RenderBlocked formats the warning message when a file still has ~ descriptions.
func RenderBlocked(relPath string) string {
	return fmt.Sprintf(
		"Blocked: %s still has ~ descriptions. Finish rewriting it, save the file, then run `agentmap next` again.\n",
		relPath,
	)
}
