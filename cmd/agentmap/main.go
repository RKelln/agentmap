// Package main is the entry point for the agentmap CLI tool.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ryankelln/agentmap/internal/check"
	"github.com/ryankelln/agentmap/internal/config"
	"github.com/ryankelln/agentmap/internal/generate"
	"github.com/ryankelln/agentmap/internal/navblock"
	"github.com/ryankelln/agentmap/internal/parser"
	"github.com/ryankelln/agentmap/internal/update"
	"github.com/spf13/cobra"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "agentmap",
	Short: "Navigation maps for AI agents",
	Long:  "agentmap generates and maintains compact navigation blocks at the top of markdown files.",
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
				lines := strings.Split(string(content), "\n")
				headings := parser.ParseHeadings(string(content), cfg.MaxDepth)
				sections := parser.ComputeSections(headings, len(lines))
				pr := navblock.ParseNavBlock(string(content))
				existingBlock, found := pr.Block, pr.Found

				fmt.Printf("File: %s (%d lines)\n\n", root, len(lines))

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
			return nil
		}

		return check.Check(root, cfg, warnUnreviewed)
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version",
	Run: func(cmd *cobra.Command, _ []string) {
		cmd.Println(version)
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

	rootCmd.AddCommand(generateCmd, updateCmd, checkCmd, versionCmd, hookCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
