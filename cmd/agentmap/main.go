// Package main is the entry point for the agentmap CLI tool.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/RKelln/agentmap/internal/check"
	"github.com/RKelln/agentmap/internal/config"
	"github.com/RKelln/agentmap/internal/generate"
	"github.com/RKelln/agentmap/internal/guide"
	"github.com/RKelln/agentmap/internal/index"
	"github.com/RKelln/agentmap/internal/initcmd"
	"github.com/RKelln/agentmap/internal/navblock"
	"github.com/RKelln/agentmap/internal/next"
	"github.com/RKelln/agentmap/internal/parser"
	"github.com/RKelln/agentmap/internal/update"
	"github.com/spf13/cobra"
)

var version = "dev"

var commit = ""

var rootCmd = &cobra.Command{
	Use:   "agentmap",
	Short: "Navigation maps for AI agents",
	Long:  "agentmap generates and maintains compact navigation blocks at the top of markdown files.",
}

var guideCmd = &cobra.Command{
	Use:   "guide",
	Short: "Print the nav writing guide",
	Long:  "Print the nav writing guide: how to write purpose, about, and see fields.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, _ []string) {
		fmt.Print(guide.Content)
	},
}

var generateCmd = &cobra.Command{
	Use:   "generate [path]",
	Short: "Generate nav blocks for markdown files",
	Long:  "Parse markdown headings, generate descriptions, write full nav blocks.\nPath can be a directory (recursive) or a single .md file.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root := "."
		if len(args) > 0 {
			root = args[0]
		}

		cfgPath, err := config.FindConfig(root)
		if err != nil {
			return fmt.Errorf("find config: %w", err)
		}

		cfg := config.Defaults()
		if cfgPath != "" {
			loaded, err := config.Load(cfgPath)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			cfg = loaded
		}

		// Override with flags if provided
		if cmd.Flags().Changed("min-lines") {
			cfg.MinLines, _ = cmd.Flags().GetInt("min-lines")
		}
		if cmd.Flags().Changed("sub-threshold") {
			cfg.SubThreshold, _ = cmd.Flags().GetInt("sub-threshold")
		}
		if cmd.Flags().Changed("expand-threshold") {
			cfg.ExpandThreshold, _ = cmd.Flags().GetInt("expand-threshold")
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")
		output, _ := cmd.Flags().GetString("output")
		debug, _ := cmd.Flags().GetBool("debug")

		// If path is a single file, process it directly
		info, err := os.Stat(root)
		if err == nil && !info.IsDir() {
			if debug {
				content, err := os.ReadFile(root)
				if err != nil {
					return fmt.Errorf("read file: %w", err)
				}
				totalLines := strings.Count(string(content), "\n")
				headings, parseWarnings := parser.ParseHeadings(string(content), cfg.MaxDepth)
				for _, w := range parseWarnings {
					fmt.Fprintf(os.Stderr, "warning: %s: %s\n", root, w)
				}
				sections := parser.ComputeSections(headings, totalLines)
				pr := navblock.ParseNavBlock(string(content))
				existingBlock, found := pr.Block, pr.Found

				fmt.Printf("File: %s (%d lines)\n\n", root, totalLines)

				fmt.Printf("Found %d headings, %d sections\n", len(headings), len(sections))
				if found {
					fmt.Printf("Existing nav block: purpose=%q, %d nav entries, %d see entries\n",
						existingBlock.Purpose, len(existingBlock.Nav), len(existingBlock.See))
				} else {
					fmt.Println("No existing nav block")
				}
				fmt.Println()

				for _, s := range sections {
					prefix := strings.Repeat("#", s.Depth)
					size := s.End - s.Start + 1
					fmt.Printf("  %d-%d (%3d lines)  %s%s\n", s.Start, s.End, size, prefix, s.Text)
				}
				return nil
			}
			report, err := generate.File(root, cfg, dryRun, output)
			if err != nil {
				return err
			}
			fmt.Println(report)
			return nil
		}

		return generate.Generate(root, cfg, dryRun)
	},
}

var updateCmd = &cobra.Command{
	Use:   "update [path]",
	Short: "Refresh line numbers in existing nav blocks",
	Long:  "Fast line-number refresh. Preserves all descriptions.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root := "."
		if len(args) > 0 {
			root = args[0]
		}

		cfgPath, err := config.FindConfig(root)
		if err != nil {
			return fmt.Errorf("find config: %w", err)
		}

		cfg := config.Defaults()
		if cfgPath != "" {
			loaded, err := config.Load(cfgPath)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			cfg = loaded
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")
		quiet, _ := cmd.Flags().GetBool("quiet")

		// If path is a single file, process it directly
		info, err := os.Stat(root)
		if err == nil && !info.IsDir() {
			report, err := update.File(root, cfg, dryRun, quiet)
			if err != nil {
				return err
			}
			if !quiet && report != "" {
				fmt.Println(report)
			}
			// Auto-check the task list entry if the file is now fully reviewed.
			if !dryRun {
				repoRoot := findRepoRoot(root, cfgPath)
				if repoRoot != "" {
					absFile, _ := filepath.Abs(root)
					relPath, relErr := filepath.Rel(repoRoot, absFile)
					if relErr == nil {
						taskListPath := index.TaskListPath(repoRoot)
						if err := index.CheckOffTaskEntry(taskListPath, absFile, relPath); err != nil {
							fmt.Fprintf(os.Stderr, "warning: task list check-off: %v\n", err)
						}
					}
				}
			}
			return nil
		}

		return update.Update(root, cfg, dryRun, quiet)
	},
}

var checkCmd = &cobra.Command{
	Use:           "check [path]",
	Short:         "Validate nav blocks are in sync with headings",
	Long:          "Verify nav blocks match current headings and line numbers. Never modifies files.",
	Args:          cobra.MaximumNArgs(1),
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, args []string) error {
		root := "."
		if len(args) > 0 {
			root = args[0]
		}

		cfgPath, err := config.FindConfig(root)
		if err != nil {
			return fmt.Errorf("find config: %w", err)
		}

		cfg := config.Defaults()
		if cfgPath != "" {
			loaded, err := config.Load(cfgPath)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			cfg = loaded
		}

		warnUnreviewed, _ := cmd.Flags().GetBool("warn-unreviewed")

		info, err := os.Stat(root)
		if err == nil && !info.IsDir() {
			failed, report, warnings, err := check.CheckFile(root, cfg, warnUnreviewed)
			if err != nil {
				return err
			}
			if len(warnings) > 0 {
				fmt.Printf("WARN: %s has unreviewed descriptions\n", root)
				for _, w := range warnings {
					fmt.Println(w)
				}
				fmt.Println()
				fmt.Println("1 file with unreviewed descriptions.")
			}
			if failed {
				fmt.Println(report)
				fmt.Println("1 file failed validation.")
				return fmt.Errorf("validation failed")
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "All nav blocks in sync (1 file checked)")
			return nil
		}

		n, err := check.Check(root, cfg, warnUnreviewed)
		if err != nil {
			return err
		}
		if n == 1 {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "All nav blocks in sync (1 file checked)")
		} else {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "All nav blocks in sync (%d files checked)\n", n)
		}
		return nil
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version",
	Run: func(cmd *cobra.Command, _ []string) {
		if commit != "" {
			cmd.Printf("%s (%s)\n", version, commit)
		} else {
			cmd.Println(version)
		}
	},
}

var indexCmd = &cobra.Command{
	Use:   "index [path]",
	Short: "Bulk index markdown files and generate task list",
	Long: `Discover markdown files, generate nav block skeletons for unindexed files,
and write .agentmap/index-tasks.md with a checklist for agent-written descriptions.

Files with no nav block → generate skeleton; add to task list.
Files with ~ descriptions → add to task list; keep existing nav block.
Files with no ~ anywhere → skip (already fully indexed).`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root := "."
		if len(args) > 0 {
			root = args[0]
		}

		cfgPath, err := config.FindConfig(root)
		if err != nil {
			return fmt.Errorf("find config: %w", err)
		}

		cfg := config.Defaults()
		if cfgPath != "" {
			loaded, err := config.Load(cfgPath)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			cfg = loaded
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")
		force, _ := cmd.Flags().GetBool("force")
		filesOnly, _ := cmd.Flags().GetBool("files-only")

		if force && filesOnly {
			fmt.Fprintf(os.Stderr, "WARN: --force has no effect when combined with --files-only (nav block regeneration is skipped)\n")
		}

		if !filesOnly {
			result, err := index.BuildIndex(root, cfg, dryRun, force)
			if err != nil {
				return err
			}
			fmt.Printf("Generated: %d  Tasks: %d  Skipped: %d\n",
				result.Generated, result.TaskFiles, result.Skipped)
			if result.TaskPath != "" {
				fmt.Printf("Task list: %s\n", result.TaskPath)
				fmt.Printf(
					"\nNext step:\n\n" +
						"  Run `agentmap next` — it prints a single-file prompt for your agent.\n" +
						"  The agent rewrites the ~-prefixed descriptions, saves the file,\n" +
						"  then runs `agentmap next` again to advance to the next file.\n" +
						"  When all files are done, `agentmap next` prints the final check command.\n\n")
			} else if dryRun {
				fmt.Println("(dry-run: no files written)")
			}
		}

		// Build and write files block.
		entries, err := index.BuildFilesBlock(root, cfg)
		if err != nil {
			return err
		}
		if len(entries) > 0 {
			dest, err := index.WriteFilesBlock(root, entries, cfg, dryRun)
			if err != nil {
				return err
			}
			if dryRun {
				fmt.Printf("Files block: would write to %s (%d entries)\n", dest, len(entries))
			} else {
				fmt.Printf("Files block: %s (%d entries)\n", dest, len(entries))
			}
		} else if filesOnly {
			fmt.Println("No indexed files found; run agentmap index first.")
		}

		return nil
	},
}

var initCmd = &cobra.Command{
	Use:   "init [path]",
	Short: "Configure agent tools to use agentmap",
	Long: `Detect agent tool configs (AGENTS.md; CLAUDE.md; .cursor/rules; .windsurf/rules;
.continue/rules; .roo/rules; .amazonq/rules; .opencode; .aider.conf.yml) and
append agentmap instructions. Optionally install a pre-commit hook.

If no tool configs are detected, creates AGENTS.md (works with all major agent tools).`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root := "."
		if len(args) > 0 {
			root = args[0]
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")
		yes, _ := cmd.Flags().GetBool("yes")
		noHook, _ := cmd.Flags().GetBool("no-hook")
		tool, _ := cmd.Flags().GetString("tool")

		opts := initcmd.Options{
			Root:       root,
			DryRun:     dryRun,
			Yes:        yes,
			NoHook:     noHook,
			ToolFilter: tool,
		}

		plan, err := initcmd.Apply(opts)
		if err != nil {
			return err
		}

		fmt.Print(plan.String())
		return nil
	},
}

var uninitCmd = &cobra.Command{
	Use:   "uninit [path]",
	Short: "Remove agentmap configuration injected by init",
	Long: `Reverse agentmap init: find <!-- agentmap:init --> markers and remove the
injected blocks. Delete files that were created entirely by init. Remove
pre-commit hook entries. Never touches AGENT:NAV blocks or other content.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root := "."
		if len(args) > 0 {
			root = args[0]
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")
		yes, _ := cmd.Flags().GetBool("yes")

		opts := initcmd.UninitOptions{
			Root:   root,
			DryRun: dryRun,
			Yes:    yes,
		}

		plan, err := initcmd.Uninit(opts)
		if err != nil {
			return err
		}

		fmt.Print(plan.String())
		return nil
	},
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove agentmap binary and config",
	Long: `Detect how agentmap was installed and print uninstall instructions (for
Homebrew/Scoop/go install), or remove the binary directly for direct installs.
Runs uninit first unless --keep-config is set.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		keepConfig, _ := cmd.Flags().GetBool("keep-config")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		yes, _ := cmd.Flags().GetBool("yes")

		// Run uninit first (unless --keep-config).
		if !keepConfig {
			opts := initcmd.UninitOptions{
				Root:   ".",
				DryRun: dryRun,
				Yes:    yes,
			}
			plan, err := initcmd.Uninit(opts)
			if err != nil {
				return fmt.Errorf("uninit: %w", err)
			}
			if len(plan.Actions) > 0 {
				fmt.Print(plan.String())
			}
		}

		// Detect install method.
		exePath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("resolve executable: %w", err)
		}
		resolved, err := os.Readlink(exePath)
		if err != nil {
			resolved = exePath
		}

		gopath := os.Getenv("GOPATH")
		gobin := os.Getenv("GOBIN")
		method := initcmd.DetectInstallMethod(resolved, gopath, gobin)

		if instructions := initcmd.UninstallInstructions(method); instructions != "" {
			fmt.Println(instructions)
			return nil
		}

		// Direct install: remove the binary.
		if dryRun {
			fmt.Printf("[dry-run] Would remove: %s\n", resolved)
			return nil
		}

		fmt.Printf("Removing: %s\n", resolved)
		if err := os.Remove(resolved); err != nil {
			return fmt.Errorf("remove binary: %w", err)
		}
		fmt.Println("agentmap uninstalled.")
		return nil
	},
}

var nextCmd = &cobra.Command{
	Use:   "next [task-list-path]",
	Short: "Advance progress and print the next unchecked task prompt",
	Long: `Flush any previously-emitted files (update + check-off), then find the
next unchecked entry in index-tasks.md and print a self-contained prompt.

State is tracked in .agentmap/next-state. On each call, next runs
agentmap update + check-off on every file in the state, then emits the
next N unchecked entries and records them as the new state.

If a file in state still has ~ descriptions, next prints a warning and
stops so the agent can fix it before advancing.

With no arguments, searches upward from the current directory for
.agentmap/index-tasks.md. An explicit path may be given instead.

Use --count N to emit prompts for N consecutive unchecked files.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		count, _ := cmd.Flags().GetInt("count")
		if count < 1 {
			count = 1
		}

		// Resolve the task list path.
		var taskListPath string
		if len(args) > 0 {
			abs, err := filepath.Abs(args[0])
			if err != nil {
				return fmt.Errorf("next: resolve path: %w", err)
			}
			taskListPath = abs
		} else {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("next: get cwd: %w", err)
			}
			taskListPath, err = next.FindTaskList(cwd)
			if err != nil {
				return err
			}
		}

		repoRoot := filepath.Dir(filepath.Dir(taskListPath))

		// Flush state: update + check-off previously-emitted files.
		result, err := next.FlushState(taskListPath, repoRoot)
		if err != nil {
			return err
		}
		if result.Blocked {
			fmt.Print(next.RenderBlocked(result.BlockedPath))
			return nil
		}

		// Collect the next N unchecked entries.
		var tasks []*next.Task
		for i := 0; i < count; i++ {
			task, err := next.Next(taskListPath, i)
			if err != nil {
				return err
			}
			if task == nil {
				break
			}
			tasks = append(tasks, task)
		}

		if len(tasks) == 0 {
			// Clear state and report done.
			_ = next.WriteState(taskListPath, nil)
			fmt.Print(next.RenderDone(repoRoot))
			return nil
		}

		// Write new state.
		relPaths := make([]string, len(tasks))
		for i, t := range tasks {
			relPaths[i] = t.RelPath
		}
		if err := next.WriteState(taskListPath, relPaths); err != nil {
			return err
		}

		// Print prompts.
		for i, task := range tasks {
			if i > 0 {
				fmt.Println("---")
			}
			fmt.Print(next.RenderPrompt(task))
		}
		return nil
	},
}

func init() {
	// generate flags
	generateCmd.Flags().Int("min-lines", 50, "Minimum file size for full nav block")
	generateCmd.Flags().Int("sub-threshold", 50, "Minimum section size for subsection info")
	generateCmd.Flags().Int("expand-threshold", 150, "Section size for full h3 entries")
	generateCmd.Flags().Bool("dry-run", false, "Print without writing files")
	generateCmd.Flags().StringP("output", "o", "", "Write output to file instead of modifying source")
	generateCmd.Flags().BoolP("debug", "D", false, "Show parsed headings and section ranges instead of generating")

	// update flags
	updateCmd.Flags().Bool("quiet", false, "Suppress report output")
	updateCmd.Flags().Bool("dry-run", false, "Print without writing files")

	// check flags
	checkCmd.Flags().Bool("warn-unreviewed", false, "Warn about auto-generated descriptions that haven't been reviewed")

	// index flags
	indexCmd.Flags().Bool("dry-run", false, "Print without writing files")
	indexCmd.Flags().Bool("force", false, "Regenerate nav blocks even for files with existing nav blocks")
	indexCmd.Flags().Bool("files-only", false, "Skip task list; only generate files block")

	// init flags
	initCmd.Flags().Bool("dry-run", false, "Preview changes without writing files")
	initCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")
	initCmd.Flags().Bool("no-hook", false, "Skip pre-commit hook installation")
	initCmd.Flags().String("tool", "", "Only configure a specific tool (cursor; claude; windsurf; continue; roo; amazonq; opencode; aider)")

	// uninit flags
	uninitCmd.Flags().Bool("dry-run", false, "Preview changes without writing files")
	uninitCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")

	// uninstall flags
	uninstallCmd.Flags().Bool("dry-run", false, "Preview changes without writing files")
	uninstallCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")
	uninstallCmd.Flags().Bool("keep-config", false, "Only remove binary; skip uninit")

	// upgrade flags
	upgradeCmd.Flags().Bool("check", false, "Only check if an update is available; do not update")

	// next flags
	nextCmd.Flags().Int("count", 1, "Number of consecutive task prompts to print")

	rootCmd.AddCommand(generateCmd, updateCmd, checkCmd, versionCmd, hookCmd, guideCmd, indexCmd, initCmd, uninitCmd, uninstallCmd, upgradeCmd, nextCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// findRepoRoot returns the repository root for a given file path and optional
// config file path. When cfgPath is non-empty, its directory is the root.
// Otherwise, walk upward from the file's directory to find a .agentmap/ dir.
func findRepoRoot(filePath, cfgPath string) string {
	if cfgPath != "" {
		return filepath.Dir(cfgPath)
	}
	dir, err := filepath.Abs(filepath.Dir(filePath))
	if err != nil {
		return ""
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".agentmap")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}
