package cmd

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/suleymanmercan/sur/internal/engine"
	"github.com/suleymanmercan/sur/internal/store"
	"github.com/suleymanmercan/sur/internal/tui"
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

		var tasks []engine.Task
		var err error

		if taskDir != "" {
		    tasks, err = engine.LoadTasks(taskDir)
		} else {
		    tasks, err = engine.LoadTasksFS(embeddedTaskFS, "tasks")
		}
		if err != nil {
		    return err
		}
		if len(tasks) == 0 {
			return fmt.Errorf("no tasks found")
		}

		s, err := store.Open(stateFile)
		if err != nil {
			return err
		}
		defer s.Close()

		r := &engine.Runner{Store: s, DryRun: dryRun}

		// pick which tasks to run
		toRun, sessionID, err := selectTasks(r, tasks)
		if err != nil || toRun == nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Minute)
		defer cancel()
		results := r.Apply(ctx, sessionID, toRun)

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

func selectTasks(r *engine.Runner, tasks []engine.Task) ([]engine.Task, string, error) {
	// --resume picks up the last running session
	if resume {
		sessions, err := r.Store.ListSessions(1)
		if err != nil {
			return nil, "", err
		}
		if len(sessions) == 0 || sessions[0].Status != store.SessionRunning {
			return nil, "", errors.New("no resumable session found")
		}
		fmt.Println("Resuming session", sessions[0].ID)
		// (MVP: re-run every task; engine.pre_check will skip already-applied ones)
		sid := sessions[0].ID
		return tasks, sid, nil
	}

	sid, err := r.StartSession()
	if err != nil {
		return nil, "", err
	}

	// non-interactive paths
	if yesFlag || allowAll || len(onlyIDs) > 0 || !term.IsTerminal(int(os.Stdin.Fd())) {
		filtered := filterTasks(tasks, onlyIDs)
		return filtered, sid, nil
	}

	selected, canceled, err := tui.Run(tasks)
	if err != nil {
		return nil, "", err
	}
	if canceled || len(selected) == 0 {
		fmt.Println("Aborted — nothing applied.")
		_ = r.Store.FinishSession(sid, store.SessionFailed)
		return nil, "", nil
	}
	return selected, sid, nil
}

func filterTasks(all []engine.Task, ids []string) []engine.Task {
	if len(ids) == 0 {
		return all
	}
	want := map[string]bool{}
	for _, id := range ids {
		want[id] = true
	}
	var out []engine.Task
	for _, t := range all {
		if want[t.ID] {
			out = append(out, t)
		}
	}
	return out
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
