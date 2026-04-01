// Package main is the entry point for the agentmap CLI tool.
package main

import (
	"fmt"
	"os"

	"github.com/ryankelln/agentmap/internal/config"
	"github.com/ryankelln/agentmap/internal/generate"
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
	Long:  "Parse markdown headings, generate descriptions, write full nav blocks.",
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

		return generate.Generate(root, cfg, dryRun)
	},
}

var updateCmd = &cobra.Command{
	Use:   "update [path]",
	Short: "Refresh line numbers in existing nav blocks",
	Long:  "Fast line-number refresh. Preserves all descriptions.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.Println("update: not yet implemented")
		return nil
	},
}

var checkCmd = &cobra.Command{
	Use:   "check [path]",
	Short: "Validate nav blocks are in sync with headings",
	Long:  "Verify nav blocks match current headings and line numbers. Never modifies files.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.Println("check: not yet implemented")
		return nil
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

	// update flags
	updateCmd.Flags().Bool("quiet", false, "Suppress report output")
	updateCmd.Flags().Bool("dry-run", false, "Print without writing files")

	rootCmd.AddCommand(generateCmd, updateCmd, checkCmd, versionCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
