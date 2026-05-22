package cmd

import (
	"github.com/spf13/cobra"

	"github.com/suleymanmercan/sur/internal/tui"
)

var stackCmd = &cobra.Command{
	Use:   "stack",
	Short: "Manage Docker Compose stacks (databases, services, monitoring)",
	Long: `sur stack provides an interactive TUI to install and manage
Docker Compose based development and monitoring stacks.

Stacks are fetched from the official catalog at:
  https://raw.githubusercontent.com/suleymanmercan/sur/main/catalog/stacks/

Users may also add custom stacks by placing a valid stack directory under:
  /etc/sur/stacks/<stack-id>/

Installed stacks live under /opt/sur/stacks/<stack-id>/.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return tui.RunStack()
	},
}

func init() {
	rootCmd.AddCommand(stackCmd)
}
