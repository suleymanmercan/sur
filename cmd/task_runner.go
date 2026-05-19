package cmd

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

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

func loadTaskSet(fs embed.FS, embeddedDir, overrideDir string) ([]engine.Task, error) {
	if overrideDir != "" {
		return engine.LoadTasks(overrideDir)
	}
	return engine.LoadTasksFS(fs, embeddedDir)
}

func runTaskSet(ctx context.Context, tasks []engine.Task, opts taskRunOptions) (string, []engine.Result, error) {
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
	return sessionID, r.Apply(runCtx, sessionID, toRun), nil
}

func applicableTasks(ctx context.Context, tasks []engine.Task) ([]engine.Task, []skippedTask) {
	osInfo, _ := osdetect.Detect()
	var out []engine.Task
	var skipped []skippedTask
	for _, t := range tasks {
		if !supportsOS(t, osInfo) {
			skipped = append(skipped, skippedTask{ID: t.ID, Reason: unsupportedReason(osInfo)})
			continue
		}
		needsRun, code := engine.NeedsRun(ctx, t)
		if !needsRun {
			skipped = append(skipped, skippedTask{
				ID:     t.ID,
				Reason: fmt.Sprintf("already satisfied (pre_check exit %d, expected %d)", code, t.PreCheck.ExpectExit),
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

func supportsOS(t engine.Task, info *osdetect.OSInfo) bool {
	if len(t.Distros) == 0 {
		return true
	}
	if info == nil || info.ID == "" {
		return false
	}
	id := normalizeDistro(info.ID)
	family := normalizeDistro(string(info.Family))
	for _, d := range t.Distros {
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

func selectTaskSet(r *engine.Runner, tasks []engine.Task, opts taskRunOptions) ([]engine.Task, string, error) {
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

	if opts.Yes || opts.All || len(opts.OnlyIDs) > 0 || !term.IsTerminal(int(os.Stdin.Fd())) {
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

func filterTasks(all []engine.Task, ids []string) ([]engine.Task, error) {
	if len(ids) == 0 {
		return all, nil
	}
	want := map[string]bool{}
	for _, id := range ids {
		want[id] = true
	}
	var out []engine.Task
	for _, t := range all {
		if want[t.ID] {
			out = append(out, t)
			delete(want, t.ID)
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
