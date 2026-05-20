package cmd

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"

	"github.com/suleymanmercan/sur/internal/engine"
	"github.com/suleymanmercan/sur/internal/osdetect"
	"github.com/suleymanmercan/sur/internal/store"
	"github.com/suleymanmercan/sur/internal/tui"
)

type taskRunOptions struct {
	DryRun   bool
	Yes      bool
	Resume   bool
	All      bool
	OnlyIDs  []string
	State    string
	Timeout  time.Duration
	TUITitle string
}

type skippedTask struct {
	ID     string
	Reason string
}

func loadTaskSet(fs embed.FS, embeddedDir, overrideDir string) ([]engine.RunnableTask, error) {
	var merged []engine.RunnableTask
	byID := map[string]engine.RunnableTask{}

	// 1. Load embedded tasks
	embedded, err := engine.LoadAllRunnableTasksFS(fs, embeddedDir)
	if err == nil {
		for _, t := range embedded {
			byID[t.GetID()] = t
		}
	}

	// 2. Load default directories
	// Default system directory e.g., /etc/sur/tasks or /etc/sur/install_tasks
	sysDir := filepath.Join("/etc/sur", embeddedDir)
	sysTasks, err := engine.LoadAllRunnableTasks(sysDir)
	if err == nil {
		for _, t := range sysTasks {
			byID[t.GetID()] = t
		}
	}

	// Also check local directory (e.g. ./tasks or ./install_tasks)
	localTasks, err := engine.LoadAllRunnableTasks(embeddedDir)
	if err == nil {
		for _, t := range localTasks {
			byID[t.GetID()] = t
		}
	}

	// 3. Load override directory if specified, and merge/override
	if overrideDir != "" {
		overrideTasks, err := engine.LoadAllRunnableTasks(overrideDir)
		if err != nil {
			return nil, err
		}
		for _, t := range overrideTasks {
			byID[t.GetID()] = t
		}
	}

	for _, t := range byID {
		merged = append(merged, t)
	}

	sort.Slice(merged, func(i, j int) bool {
		return merged[i].GetID() < merged[j].GetID()
	})

	return merged, nil
}

func runTaskSet(ctx context.Context, tasks []engine.RunnableTask, opts taskRunOptions) (string, []engine.Result, error) {
	if len(tasks) == 0 {
		return "", nil, fmt.Errorf("no tasks found")
	}
	if opts.Timeout == 0 {
		opts.Timeout = 30 * time.Minute
	}
	if !opts.Resume {
		var skipped []skippedTask
		tasks, skipped = applicableTasks(ctx, tasks)
		printSkippedTaskSummary(skipped)
		if len(tasks) == 0 {
			fmt.Println("No applicable tasks to run. The supported tasks are already satisfied or not available on this OS.")
			return "", []engine.Result{}, nil
		}
	}

	s, err := store.Open(opts.State)
	if err != nil {
		return "", nil, err
	}
	defer s.Close()

	r := &engine.Runner{Store: s, DryRun: opts.DryRun}
	toRun, sessionID, err := selectTaskSet(r, tasks, opts)
	if err != nil || toRun == nil {
		return sessionID, nil, err
	}

	runCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	// Use the live progress TUI when we have a real interactive terminal
	// and the caller is not requesting JSON output.
	if term.IsTerminal(int(os.Stdout.Fd())) && !jsonOutput { // #nosec G115
		title := opts.TUITitle
		if title == "" {
			title = "sur — running tasks"
		}
		runResult, err := tui.RunProgress(toRun, title, func(send func(tea.Msg)) {
			// Wire engine callbacks → Bubble Tea messages.
			r.Progress = engine.Progress{
				OnTaskStart: func(id, name string, index, total int) {
					send(tui.TaskStartMsg{ID: id, Name: name, Index: index, Total: total})
				},
				OnTaskLog: func(line string) {
					send(tui.TaskLogMsg{Line: line})
				},
				OnTaskDone: func(id string, status store.TaskStatus, dur time.Duration, execErr error) {
					send(tui.TaskDoneMsg{ID: id, Status: status, Duration: dur, Err: execErr})
				},
			}
			results := r.Apply(runCtx, sessionID, toRun)
			send(tui.AllDoneMsg{Results: results})
		})
		if err != nil {
			return sessionID, nil, err
		}
		// User pressed Enter → go back to the task picker.
		if runResult.GoBack {
			return runTaskSet(ctx, tasks, opts)
		}
		return sessionID, runResult.Results, nil
	}

	// Fallback: plain stderr logging (CI / pipe / JSON mode).
	return sessionID, r.Apply(runCtx, sessionID, toRun), nil
}

func applicableTasks(ctx context.Context, tasks []engine.RunnableTask) ([]engine.RunnableTask, []skippedTask) {
	osInfo, _ := osdetect.Detect()
	var out []engine.RunnableTask
	var skipped []skippedTask
	for _, t := range tasks {
		if !supportsOS(t, osInfo) {
			skipped = append(skipped, skippedTask{ID: t.GetID(), Reason: unsupportedReason(osInfo)})
			continue
		}
		needsRun, code := t.NeedsRun(ctx)
		if !needsRun {
			skipped = append(skipped, skippedTask{
				ID:     t.GetID(),
				Reason: fmt.Sprintf("already satisfied (pre_check exit %d)", code),
			})
			continue
		}
		out = append(out, t)
	}
	return out, skipped
}

func printSkippedTaskSummary(skipped []skippedTask) {
	if len(skipped) == 0 {
		return
	}
	fmt.Printf("Skipped %d task(s):\n", len(skipped))
	for _, s := range skipped {
		fmt.Printf("  - %s: %s\n", s.ID, s.Reason)
	}
}

func supportsOS(t engine.RunnableTask, info *osdetect.OSInfo) bool {
	distros := t.GetDistros()
	if len(distros) == 0 {
		return true
	}
	if info == nil || info.ID == "" {
		return false
	}
	id := normalizeDistro(info.ID)
	family := normalizeDistro(string(info.Family))
	for _, d := range distros {
		want := normalizeDistro(d)
		if want == id || want == family {
			return true
		}
	}
	return false
}

func unsupportedReason(info *osdetect.OSInfo) string {
	if info == nil || info.ID == "" {
		return "unsupported OS"
	}
	return fmt.Sprintf("unsupported OS: %s", info.ID)
}

func normalizeDistro(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	switch v {
	case "alma", "almalinux":
		return "almalinux"
	case "opensuse-leap", "opensuse-tumbleweed", "sles", "suse":
		return "opensuse"
	}
	return v
}

func selectTaskSet(r *engine.Runner, tasks []engine.RunnableTask, opts taskRunOptions) ([]engine.RunnableTask, string, error) {
	if opts.Resume {
		sessions, err := r.Store.ListSessions(1)
		if err != nil {
			return nil, "", err
		}
		if len(sessions) == 0 || sessions[0].Status != store.SessionRunning {
			return nil, "", errors.New("no resumable session found")
		}
		fmt.Println("Resuming session", sessions[0].ID)
		return tasks, sessions[0].ID, nil
	}

	sid, err := r.StartSession()
	if err != nil {
		return nil, "", err
	}

	if opts.Yes || opts.All || len(opts.OnlyIDs) > 0 || !term.IsTerminal(int(os.Stdin.Fd())) { // #nosec G115 -- fd value fits in int on all supported platforms
		filtered, err := filterTasks(tasks, opts.OnlyIDs)
		if err != nil {
			_ = r.Store.FinishSession(sid, store.SessionFailed)
			return nil, "", err
		}
		return filtered, sid, nil
	}

	title := opts.TUITitle
	if title == "" {
		title = "sur — choose tasks"
	}
	selected, canceled, err := tui.RunWithTitle(tasks, title)
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

func filterTasks(all []engine.RunnableTask, ids []string) ([]engine.RunnableTask, error) {
	if len(ids) == 0 {
		return all, nil
	}
	want := map[string]bool{}
	for _, id := range ids {
		want[id] = true
	}
	var out []engine.RunnableTask
	for _, t := range all {
		if want[t.GetID()] {
			out = append(out, t)
			delete(want, t.GetID())
		}
	}
	if len(want) > 0 {
		var missing []string
		for id := range want {
			missing = append(missing, id)
		}
		sort.Strings(missing)
		return nil, fmt.Errorf("unknown task id(s): %s", strings.Join(missing, ", "))
	}
	return out, nil
}
