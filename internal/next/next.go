// Package next implements the agentmap next command: find the next unchecked
// task in index-tasks.md and print a self-contained, single-file prompt for a
// small-model agent. The agent rewrites the nav block, runs agentmap update,
// then calls agentmap next again to get the following task.
package next

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/RKelln/agentmap/internal/navblock"
)

// Task is the next unchecked task found in the index-tasks.md.
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

// Next reads taskListPath, finds the first unchecked entry (skipping the first
// skip entries) that corresponds to a markdown file, reads the nav block from
// the file, and returns a Task. Returns (nil, nil) when all tasks are done.
// Use skip=0 for the first call; increment by 1 for each subsequent call when
// emitting multiple prompts without modifying the task list between calls.
func Next(taskListPath string, skip int) (*Task, error) {
	data, err := os.ReadFile(taskListPath)
	if err != nil {
		return nil, fmt.Errorf("next: read task list: %w", err)
	}
	content := string(data)

	// Scan line by line: find file-entry headings followed by "- [ ]".
	// Entry headings look like: "## path/to/file.md (N lines)"
	// Non-entry headings (preamble): "## Your job", "## Rules (quick ref)", etc.
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
			// File entry headings end with " (N lines)" where N is a number.
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
			current = nil // don't match this entry again
		}
		if current != nil && line == "- [x]" {
			current = nil
		}
	}

	return nil, nil // all done (or no entries found)
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
	// Reconstruct the raw block from the Start/End line indices (1-indexed).
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
	b.WriteString("5. Run: `agentmap update " + t.RelPath + "`\n")
	b.WriteString("6. Run: `agentmap next` to get the next file.\n")

	return b.String()
}

// RenderDone formats the completion message when all tasks are checked off.
func RenderDone(repoRoot string) string {
	return "All tasks complete. Run: agentmap check " + repoRoot + "\n"
}
