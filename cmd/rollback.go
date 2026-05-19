package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/suleymanmercan/sur/internal/engine"
	"github.com/suleymanmercan/sur/internal/store"
)

var rollbackCmd = &cobra.Command{
	Use:   "rollback <session-id>",
	Short: "Roll back every task in a previous session (in reverse order)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := store.Open(stateFile)
		if err != nil {
			return err
		}
		defer s.Close()

		tasks, err := loadRollbackTasks()
		if err != nil {
			return err
		}
		r := &engine.Runner{Store: s}

		ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Minute)
		defer cancel()
		results, err := r.RollbackSessionTasks(ctx, args[0], tasks)
		if err != nil {
			return err
		}
		if jsonOutput {
			return emitJSON(results)
		}
		for _, res := range results {
			fmt.Printf("  ↺ %-30s %-12s %v\n", res.TaskID, res.Status, res.Err)
		}
		return nil
	},
}

func loadRollbackTasks() ([]engine.Task, error) {
	if taskDir != "" {
		return engine.LoadTasks(taskDir)
	}
	return engine.LoadTasksFS(embeddedTaskFS, "tasks")
}

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "List past hardening sessions",
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := store.Open(stateFile)
		if err != nil {
			return err
		}
		defer s.Close()
		sessions, err := s.ListSessions(50)
		if err != nil {
			return err
		}
		if jsonOutput {
			return emitJSON(sessions)
		}
		if len(sessions) == 0 {
			fmt.Println("No sessions yet — run `sur harden`.")
			return nil
		}
		fmt.Printf("%-36s  %-20s  %-12s  %s\n", "SESSION", "HOST", "STATUS", "STARTED")
		for _, s := range sessions {
			fmt.Printf("%-36s  %-20s  %-12s  %s\n",
				s.ID, s.Hostname, s.Status, s.StartedAt.Local().Format("2006-01-02 15:04:05"))
		}
		return nil
	},
}

func init() {
	rollbackCmd.Flags().StringVar(&taskDir, "tasks", "", "directory containing task YAML files")
	rollbackCmd.Flags().StringVar(&stateFile, "state", "", "override SQLite path")
	historyCmd.Flags().StringVar(&stateFile, "state", "", "override SQLite path")
	rootCmd.AddCommand(rollbackCmd)
	rootCmd.AddCommand(historyCmd)
}
