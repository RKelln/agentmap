package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

const hookScript = `#!/bin/sh
agentmap check . || {
  echo "AGENT:NAV blocks are out of sync. Run: agentmap update ."
  exit 1
}
`

const hookYAML = `repos:
  - repo: local
    hooks:
      - id: agentmap-check
        name: Validate AGENT:NAV blocks
        entry: agentmap check .
        language: system
        types: [markdown]
        pass_filenames: false
`

var hookCmd = &cobra.Command{
	Use:   "hook",
	Short: "Print the recommended pre-commit hook script",
	Long: `Print the recommended pre-commit hook to stdout.

By default, prints a shell script suitable for .git/hooks/pre-commit.
Use --yaml to print a .pre-commit-config.yaml snippet instead.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, _ []string) {
		yaml, _ := cmd.Flags().GetBool("yaml")
		if yaml {
			_, _ = fmt.Fprint(cmd.OutOrStdout(), hookYAML)
		} else {
			_, _ = fmt.Fprint(cmd.OutOrStdout(), hookScript)
		}
	},
}

func init() {
	hookCmd.Flags().Bool("yaml", false, "Print .pre-commit-config.yaml snippet instead of shell script")
}
