// Package engine loads YAML task definitions and runs the
// pre_check → backup → exec → post_check → rollback lifecycle.
package engine

import (
	"context"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	"github.com/suleymanmercan/sur/internal/store"
)

// Step is a single shell command inside a task phase.
type Step struct {
	Command    string `yaml:"command"`
	ExpectExit *int   `yaml:"expect_exit,omitempty"`
}

// Phase is a list of steps.
type Phase []Step

// PreCheck describes a single command + expected exit code.
type PreCheck struct {
	Command    string `yaml:"command"`
	ExpectExit int    `yaml:"expect_exit"`
}

// Task is the YAML schema for a hardening task.
type Task struct {
	ID               string   `yaml:"id"`
	Name             string   `yaml:"name"`
	Description      string   `yaml:"description"`
	RollbackPossible bool     `yaml:"rollback_possible"`
	BackupFiles      []string `yaml:"backup_files"`
	RiskLevel        string   `yaml:"risk_level"`
	Distros          []string `yaml:"distros"`
	PreCheck         PreCheck `yaml:"pre_check"`
	Exec             Phase    `yaml:"exec"`
	PostCheck        PreCheck `yaml:"post_check"`
	Rollback         Phase    `yaml:"rollback"`

	// SourcePath is populated by LoadTasks; not part of YAML schema.
	SourcePath string `yaml:"-"`
}

// LoadTasks parses every *.yaml file under dir and returns them sorted by ID.
func LoadTasks(dir string) ([]Task, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return nil, err
	}
	yml, _ := filepath.Glob(filepath.Join(dir, "*.yml"))
	matches = append(matches, yml...)

	var tasks []Task
	for _, m := range matches {
		t, err := loadTaskFile(m)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", m, err)
		}
		tasks = append(tasks, t)
	}
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].ID < tasks[j].ID })
	return tasks, nil
}

// LoadTasksFS reads tasks from an embedded filesystem.
func LoadTasksFS(fs embed.FS, dir string) ([]Task, error) {
	entries, err := fs.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var tasks []Task
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}
		b, err := fs.ReadFile(dir + "/" + name)
		if err != nil {
			return nil, err
		}
		var t Task
		if err := yaml.Unmarshal(b, &t); err != nil {
			return nil, err
		}
		if t.ID == "" {
			return nil, fmt.Errorf("%s: task id is required", name)
		}
		tasks = append(tasks, t)
	}
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].ID < tasks[j].ID })
	return tasks, nil
}

// LoadAllRunnableTasks parses all YAML and Lua tasks under dir and returns them sorted by ID.
func LoadAllRunnableTasks(dir string) ([]RunnableTask, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, nil
	}

	matches, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return nil, err
	}
	yml, _ := filepath.Glob(filepath.Join(dir, "*.yml"))
	matches = append(matches, yml...)

	var tasks []RunnableTask
	for _, m := range matches {
		t, err := loadTaskFile(m)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", m, err)
		}
		tasks = append(tasks, t)
	}

	luaMatches, _ := filepath.Glob(filepath.Join(dir, "*.lua"))
	for _, m := range luaMatches {
		t, err := LoadLuaTask(m)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", m, err)
		}
		tasks = append(tasks, t)
	}

	sort.Slice(tasks, func(i, j int) bool { return tasks[i].GetID() < tasks[j].GetID() })
	return tasks, nil
}

// LoadAllRunnableTasksFS reads YAML and Lua tasks from an embedded filesystem.
func LoadAllRunnableTasksFS(fs embed.FS, dir string) ([]RunnableTask, error) {
	entries, err := fs.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var tasks []RunnableTask
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		path := dir + "/" + name
		if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
			b, err := fs.ReadFile(path)
			if err != nil {
				return nil, err
			}
			var t Task
			if err := yaml.Unmarshal(b, &t); err != nil {
				return nil, err
			}
			if t.ID == "" {
				return nil, fmt.Errorf("%s: task id is required", name)
			}
			t.SourcePath = path
			tasks = append(tasks, t)
		} else if strings.HasSuffix(name, ".lua") {
			t, err := LoadLuaTaskFS(fs, path)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}
			tasks = append(tasks, t)
		}
	}
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].GetID() < tasks[j].GetID() })
	return tasks, nil
}

func loadTaskFile(path string) (Task, error) {
	var t Task
	b, err := os.ReadFile(path) // #nosec G304 -- path comes from filepath.Glob over controlled task directories
	if err != nil {
		return t, err
	}
	if err := yaml.Unmarshal(b, &t); err != nil {
		return t, err
	}
	if t.ID == "" {
		return t, fmt.Errorf("task id is required")
	}
	t.SourcePath = path
	return t, nil
}

// ---------- runtime ----------

// RunnableTask is the interface implemented by both YAML and Lua tasks.
type RunnableTask interface {
	GetID() string
	GetName() string
	GetDescription() string
	GetRollbackPossible() bool
	GetBackupFiles() []string
	GetRiskLevel() string
	GetDistros() []string
	NeedsRun(ctx context.Context) (bool, int)
	Execute(ctx context.Context, dryRun bool, logFn func(format string, args ...any)) error
	Verify(ctx context.Context) error
	Revert(ctx context.Context, backupPath string, logFn func(format string, args ...any)) error
}

// GetID returns task ID.
func (t Task) GetID() string { return t.ID }

// GetName returns task name.
func (t Task) GetName() string { return t.Name }

// GetDescription returns task description.
func (t Task) GetDescription() string { return t.Description }

// GetRollbackPossible returns rollback capability.
func (t Task) GetRollbackPossible() bool { return t.RollbackPossible }

// GetBackupFiles returns file paths to backup.
func (t Task) GetBackupFiles() []string { return t.BackupFiles }

// GetRiskLevel returns risk level.
func (t Task) GetRiskLevel() string { return t.RiskLevel }

// GetDistros returns supported distributions.
func (t Task) GetDistros() []string { return t.Distros }

// NeedsRun returns whether the task needs execution.
func (t Task) NeedsRun(ctx context.Context) (bool, int) {
	if t.PreCheck.Command == "" {
		return true, 0
	}
	_, code := runShell(ctx, t.PreCheck.Command)
	return code == t.PreCheck.ExpectExit, code
}

// Execute runs the exec phase commands.
func (t Task) Execute(ctx context.Context, dryRun bool, logFn func(format string, args ...any)) error {
	if dryRun {
		logFn("[%s] DRY-RUN — would run %d step(s)", t.ID, len(t.Exec))
		for _, s := range t.Exec {
			logFn("       $ %s", s.Command)
		}
		return nil
	}
	for _, s := range t.Exec {
		out, code := runShell(ctx, s.Command)
		expected := 0
		if s.ExpectExit != nil {
			expected = *s.ExpectExit
		}
		if code != expected {
			return fmt.Errorf("step %q exited %d: %s", s.Command, code, strings.TrimSpace(out))
		}
	}
	return nil
}

// Verify runs the post check command.
func (t Task) Verify(ctx context.Context) error {
	if t.PostCheck.Command == "" {
		return nil
	}
	_, code := runShell(ctx, t.PostCheck.Command)
	if code != t.PostCheck.ExpectExit {
		return fmt.Errorf("post_check failed (exit %d)", code)
	}
	return nil
}

// Revert runs the rollback phase commands.
func (t Task) Revert(ctx context.Context, backupPath string, logFn func(format string, args ...any)) error {
	for _, s := range t.Rollback {
		cmd := strings.ReplaceAll(s.Command, "{backup_path}", backupPath)
		out, code := runShell(ctx, cmd)
		if code != 0 {
			return fmt.Errorf("rollback step %q exit %d: %s", cmd, code, strings.TrimSpace(out))
		}
	}
	return nil
}

// Result describes what happened to a single task.
type Result struct {
	TaskID   string
	Status   store.TaskStatus
	Err      error
	Duration time.Duration
}

// NeedsRun reports whether a task's pre_check says there is work to do.
// Deprecated: use t.NeedsRun instead.
func NeedsRun(ctx context.Context, t Task) (bool, int) {
	return t.NeedsRun(ctx)
}

// Runner executes tasks and records state.
type Runner struct {
	Store    *store.Store
	DryRun   bool
	Hostname string
	// Logger receives human-readable progress messages.
	Logger func(format string, args ...any)
}

func (r *Runner) log(format string, args ...any) {
	if r.Logger != nil {
		r.Logger(format, args...)
		return
	}
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

// StartSession opens a new persistent session and returns its ID.
func (r *Runner) StartSession() (string, error) {
	id := uuid.NewString()
	host, _ := os.Hostname()
	if r.Hostname != "" {
		host = r.Hostname
	}
	return id, r.Store.CreateSession(store.Session{ID: id, Hostname: host, StartedAt: time.Now()})
}

// Apply runs the given tasks under sessionID. It never aborts the whole run:
// per-task failures rollback that task only.
func (r *Runner) Apply(ctx context.Context, sessionID string, tasks []RunnableTask) []Result {
	var results []Result
	anyFailed := false
	anyDid := false
	for _, t := range tasks {
		select {
		case <-ctx.Done():
			results = append(results, Result{TaskID: t.GetID(), Status: store.TaskSkipped, Err: ctx.Err()})
			continue
		default:
		}
		res := r.runTask(ctx, sessionID, t)
		results = append(results, res)
		switch res.Status {
		case store.TaskFailed, store.TaskRolledBack:
			anyFailed = true
			anyDid = true
		case store.TaskSuccess:
			anyDid = true
		}
	}

	status := store.SessionCompleted
	switch {
	case anyFailed && anyDid:
		status = store.SessionPartial
	case anyFailed:
		status = store.SessionFailed
	}
	_ = r.Store.FinishSession(sessionID, status)
	return results
}

func (r *Runner) runTask(ctx context.Context, sessionID string, t RunnableTask) Result {
	start := time.Now()
	execID := uuid.NewString()

	rec := store.TaskExecution{
		ID: execID, SessionID: sessionID, TaskID: t.GetID(),
		RollbackPossible: t.GetRollbackPossible(), ExecutedAt: time.Now(),
		Status: store.TaskSuccess,
	}

	// 1. pre_check
	needsRun, code := t.NeedsRun(ctx)
	if !needsRun {
		msg := fmt.Sprintf("already satisfied or not applicable (pre_check exit %d)", code)
		r.log("[%s] %s", t.GetID(), msg)
		rec.Status = store.TaskSkipped
		rec.ErrorMessage = msg
		_ = r.Store.RecordTask(rec)
		return Result{TaskID: t.GetID(), Status: store.TaskSkipped, Err: fmt.Errorf("%s", msg), Duration: time.Since(start)}
	}

	if r.DryRun {
		_ = t.Execute(ctx, true, r.log)
		rec.Status = store.TaskSkipped
		rec.ErrorMessage = "dry-run"
		_ = r.Store.RecordTask(rec)
		return Result{TaskID: t.GetID(), Status: store.TaskSkipped, Duration: time.Since(start)}
	}

	// 2. backup files
	var backupPath string
	var backupBlob []byte
	for _, f := range t.GetBackupFiles() {
		if data, err := os.ReadFile(f); err == nil { // #nosec G304 -- backup_files paths come from trusted YAML task definitions
			backupBlob = data
			backupPath = f
			break
		}
	}
	rec.BackupData = backupBlob
	rec.BackupPath = backupPath

	// 3. exec
	r.log("[%s] applying...", t.GetID())
	execErr := t.Execute(ctx, false, r.log)

	// 4. post_check
	if execErr == nil {
		execErr = t.Verify(ctx)
	}

	if execErr != nil {
		rec.Status = store.TaskFailed
		rec.ErrorMessage = execErr.Error()
		r.log("[%s] FAILED: %v", t.GetID(), execErr)
		_ = r.Store.RecordTask(rec)
		// 5. rollback
		if t.GetRollbackPossible() {
			if err := r.rollbackTask(ctx, t, backupPath, backupBlob); err != nil {
				r.log("[%s] rollback error: %v", t.GetID(), err)
			} else {
				r.log("[%s] rolled back", t.GetID())
				_ = r.Store.UpdateTaskStatus(execID, store.TaskRolledBack, execErr.Error())
				return Result{TaskID: t.GetID(), Status: store.TaskRolledBack, Err: execErr, Duration: time.Since(start)}
			}
		}
		return Result{TaskID: t.GetID(), Status: store.TaskFailed, Err: execErr, Duration: time.Since(start)}
	}

	r.log("[%s] OK", t.GetID())
	_ = r.Store.RecordTask(rec)
	return Result{TaskID: t.GetID(), Status: store.TaskSuccess, Duration: time.Since(start)}
}

func (r *Runner) rollbackTask(ctx context.Context, t RunnableTask, backupPath string, blob []byte) error {
	// Restore the backup file first so rollback commands can rely on it.
	if backupPath != "" && len(blob) > 0 {
		if err := os.WriteFile(backupPath, blob, 0o600); err != nil { // #nosec G703 -- backupPath originates from trusted YAML task config
			return fmt.Errorf("restore %s: %w", backupPath, err)
		}
	}
	return t.Revert(ctx, backupPath, r.log)
}

// RollbackSession replays the rollback for every task in the session,
// in reverse order. Tasks marked rollback_possible=false are skipped.
//
// Deprecated: RollbackSession can only load tasks from disk and will fail
// on production installs that use the embedded task FS. Use
// RollbackSessionTasks with tasks loaded via engine.LoadTasksFS instead.
func (r *Runner) RollbackSession(ctx context.Context, sessionID string, taskDir string) ([]Result, error) {
	tasks, err := LoadTasks(taskDir)
	if err != nil {
		return nil, err
	}
	runnableTasks := make([]RunnableTask, len(tasks))
	for i, t := range tasks {
		runnableTasks[i] = t
	}
	return r.RollbackSessionTasks(ctx, sessionID, runnableTasks)
}

// RollbackSessionTasks replays rollback using already loaded task definitions.
func (r *Runner) RollbackSessionTasks(ctx context.Context, sessionID string, tasks []RunnableTask) ([]Result, error) {
	byID := map[string]RunnableTask{}
	for _, t := range tasks {
		byID[t.GetID()] = t
	}

	execs, err := r.Store.TasksForSession(sessionID)
	if err != nil {
		return nil, err
	}
	var out []Result
	for i := len(execs) - 1; i >= 0; i-- {
		e := execs[i]
		t, ok := byID[e.TaskID]
		if !ok {
			out = append(out, Result{TaskID: e.TaskID, Status: store.TaskSkipped, Err: fmt.Errorf("task definition not found")})
			continue
		}
		if !e.RollbackPossible {
			out = append(out, Result{TaskID: e.TaskID, Status: store.TaskSkipped, Err: fmt.Errorf("not rollback-able")})
			continue
		}
		if e.Status == store.TaskSkipped || e.Status == store.TaskRolledBack {
			out = append(out, Result{TaskID: e.TaskID, Status: store.TaskSkipped, Err: fmt.Errorf("task status is %s", e.Status)})
			continue
		}
		if err := r.rollbackTask(ctx, t, e.BackupPath, e.BackupData); err != nil {
			out = append(out, Result{TaskID: e.TaskID, Status: store.TaskFailed, Err: err})
			continue
		}
		_ = r.Store.UpdateTaskStatus(e.ID, store.TaskRolledBack, "manual rollback")
		out = append(out, Result{TaskID: e.TaskID, Status: store.TaskRolledBack})
	}
	return out, nil
}

func runShell(ctx context.Context, command string) (string, int) {
	cmd := exec.CommandContext(ctx, "sh", "-c", command) // #nosec G204 -- command comes from operator-authored YAML task definitions
	out, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return string(out), exitErr.ExitCode()
		}
		return string(out), -1
	}
	return string(out), 0
}
