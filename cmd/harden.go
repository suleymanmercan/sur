package cmd

import (
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/suleymanmercan/sur/internal/engine"
	"github.com/suleymanmercan/sur/internal/store"
)

var embeddedTaskFS embed.FS

func SetTaskFS(fs embed.FS) {
	embeddedTaskFS = fs
}

var (
	dryRun    bool
	yesFlag   bool
	resume    bool
	taskDir   string
	allowAll  bool
	onlyIDs   []string
	stateFile string
)

var hardenCmd = &cobra.Command{
	Use:   "harden",
	Short: "Interactively pick and apply hardening tasks",
	RunE: func(cmd *cobra.Command, args []string) error {
		if os.Geteuid() != 0 && !dryRun {
			return errors.New("sur harden requires root privileges (run with sudo) — or pass --dry-run")
		}

		tasks, err := loadTaskSet(embeddedTaskFS, "tasks", taskDir)
		if err != nil {
			return err
		}
		sessionID, results, err := runTaskSet(cmd.Context(), tasks, taskRunOptions{
			DryRun:   dryRun,
			Yes:      yesFlag,
			Resume:   resume,
			All:      allowAll,
			OnlyIDs:  onlyIDs,
			State:    stateFile,
			Timeout:  30 * time.Minute,
			TUITitle: "sur — choose hardening tasks",
		})
		if err != nil || results == nil {
			return err
		}
		if len(results) == 0 {
			return nil
		}

		if jsonOutput {
			return emitJSON(map[string]any{
				"session_id": sessionID,
				"results":    results,
			})
		}
		printResults(sessionID, results)
		return nil
	},
}

func printResults(sessionID string, results []engine.Result) {
	fmt.Println()
	fmt.Println("Session:", sessionID)
	for _, r := range results {
		marker := "✓"
		switch r.Status {
		case store.TaskFailed:
			marker = "✗"
		case store.TaskRolledBack:
			marker = "↺"
		case store.TaskSkipped:
			marker = "·"
		}
		fmt.Printf("  %s  %-30s  %-12s  %s\n", marker, r.TaskID, r.Status, r.Duration.Truncate(time.Millisecond))
		if r.Err != nil {
			fmt.Printf("      └─ %v\n", r.Err)
		}
	}
}

func resolveTaskDir(override string) (string, error) {
	if override != "" {
		return override, nil
	}
	// 1) ./tasks relative to binary
	exe, err := os.Executable()
	if err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "tasks")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	// 2) /etc/sur/tasks
	if _, err := os.Stat("/etc/sur/tasks"); err == nil {
		return "/etc/sur/tasks", nil
	}
	// 3) working dir ./tasks
	if _, err := os.Stat("tasks"); err == nil {
		return "tasks", nil
	}
	return "", errors.New("could not locate tasks directory; use --tasks <dir>")
}

func init() {
	hardenCmd.Flags().BoolVar(&dryRun, "dry-run", false, "show planned actions without touching the system")
	hardenCmd.Flags().BoolVar(&yesFlag, "yes", false, "skip TUI and apply every task (CI mode)")
	hardenCmd.Flags().BoolVar(&resume, "resume", false, "resume the last unfinished session")
	hardenCmd.Flags().BoolVar(&allowAll, "all", false, "apply every task without prompting")
	hardenCmd.Flags().StringSliceVar(&onlyIDs, "only", nil, "comma-separated task IDs to run")
	hardenCmd.Flags().StringVar(&taskDir, "tasks", "", "directory containing task YAML files")
	hardenCmd.Flags().StringVar(&stateFile, "state", "", "override SQLite path (default: /var/lib/sur/sur.db or $SUR_DB)")
	rootCmd.AddCommand(hardenCmd)
}
