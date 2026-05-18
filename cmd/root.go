package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	jsonOutput bool
)

var rootCmd = &cobra.Command{
	Use:   "sur",
	Short: "sur — interactive Linux/VPS hardening CLI",
	Long: `sur is a local, interactive Linux hardening tool.
It audits your system, lets you pick hardening tasks via a TUI,
applies them safely with backup + rollback, and tracks state in SQLite.`,
	Version: "0.1.0",
}

// Execute runs the root cobra command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "machine-readable JSON output")
}
