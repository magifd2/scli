// Package cmd provides the CLI command definitions for scli.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Build-time variables injected via -ldflags.
var version = "dev"

// Global flags shared across all subcommands.
var (
	workspace  string
	jsonOutput bool
	noColor    bool
)

var rootCmd = &cobra.Command{
	Use:   "scli",
	Short: "A CLI Slack client for users",
	Long: `scli lets you post and read Slack messages from the terminal,
without switching to a GUI client.`,
	Version: version,
}

// RootCmd returns the root cobra command (used in tests).
func RootCmd() *cobra.Command {
	return rootCmd
}

// Execute runs the root command and exits on error.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&workspace, "workspace", "w", "", "Workspace to use (overrides default)")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable color output")
}
