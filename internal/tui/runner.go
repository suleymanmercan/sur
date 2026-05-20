// Package tui — runner.go provides a live progress view while sur tasks execute.
package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/suleymanmercan/sur/internal/engine"
	"github.com/suleymanmercan/sur/internal/store"
)

// ── styles ────────────────────────────────────────────────────────────────────

var (
	runnerTitle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7CE38B"))
	runnerDim     = lipgloss.NewStyle().Faint(true)
	runnerRunning = lipgloss.NewStyle().Foreground(lipgloss.Color("#F28C28")).Bold(true)
	runnerOK      = lipgloss.NewStyle().Foreground(lipgloss.Color("#7CE38B"))
	runnerFail    = lipgloss.NewStyle().Foreground(lipgloss.Color("#F25F5C"))
	runnerRollbk  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFBF00"))
	runnerSkip    = lipgloss.NewStyle().Faint(true)
	runnerLogLine = lipgloss.NewStyle().Foreground(lipgloss.Color("#B0B8C1"))
	runnerCmd     = lipgloss.NewStyle().Foreground(lipgloss.Color("#74B9FF")).Bold(true)
	runnerBar     = lipgloss.NewStyle().Foreground(lipgloss.Color("#7CE38B"))
	runnerBarBg   = lipgloss.NewStyle().Faint(true)
)

// ── messages ─────────────────────────────────────────────────────────────────

// TaskStartMsg is sent when a task begins executing.
type TaskStartMsg struct {
	ID    string
	Name  string
	Index int // 1-based
	Total int
}

// TaskLogMsg is sent for each line of output from a running task.
type TaskLogMsg struct{ Line string }

// TaskDoneMsg is sent when a task finishes.
type TaskDoneMsg struct {
	ID       string
	Status   store.TaskStatus
	Duration time.Duration
	Err      error
}

// AllDoneMsg is sent after all tasks finish; carries the final results.
type AllDoneMsg struct{ Results []engine.Result }

// ── task entry ────────────────────────────────────────────────────────────────

type taskEntry struct {
	id       string
	name     string
	status   store.TaskStatus // "" = pending
	duration time.Duration
	err      error
	logs     []string // captured lines (capped)
}

const maxLogLines = 200 // total kept; only last N rendered

// ── model ─────────────────────────────────────────────────────────────────────

type runnerModel struct {
	title       string
	entries     []taskEntry
	activeIndex int // index into entries of the currently running task (-1 = none)
	total       int
	done        int
	termWidth   int
	termHeight  int
	finished    bool
	results     []engine.Result
}

func newRunnerModel(tasks []engine.RunnableTask, title string) runnerModel {
	entries := make([]taskEntry, len(tasks))
	for i, t := range tasks {
		name := t.GetName()
		if name == "" {
			name = t.GetID()
		}
		entries[i] = taskEntry{id: t.GetID(), name: name}
	}
	return runnerModel{
		title:       title,
		entries:     entries,
		activeIndex: -1,
		total:       len(tasks),
		termWidth:   100,
		termHeight:  30,
	}
}

func (m runnerModel) Init() tea.Cmd { return nil }

func (m runnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {

	case tea.WindowSizeMsg:
		m.termWidth = v.Width
		m.termHeight = v.Height

	case TaskStartMsg:
		// Find the entry by ID and mark it active.
		for i := range m.entries {
			if m.entries[i].id == v.ID {
				m.activeIndex = i
				break
			}
		}

	case TaskLogMsg:
		if m.activeIndex >= 0 && m.activeIndex < len(m.entries) {
			e := &m.entries[m.activeIndex]
			e.logs = append(e.logs, v.Line)
			if len(e.logs) > maxLogLines {
				e.logs = e.logs[len(e.logs)-maxLogLines:]
			}
		}

	case TaskDoneMsg:
		for i := range m.entries {
			if m.entries[i].id == v.ID {
				m.entries[i].status = v.Status
				m.entries[i].duration = v.Duration
				m.entries[i].err = v.Err
				break
			}
		}
		m.done++

	case AllDoneMsg:
		m.results = v.Results
		m.finished = true
		m.activeIndex = -1
		return m, nil

	case tea.KeyMsg:
		switch v.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q", "enter":
			if m.finished {
				return m, tea.Quit
			}
		}
	}

	return m, nil
}

func (m runnerModel) View() string {
	var b strings.Builder
	width := m.termWidth
	if width < 40 {
		width = 40
	}

	// ── header ────────────────────────────────────────────────────────────────
	b.WriteString(runnerTitle.Render(m.title) + "\n")
	b.WriteString(progressBar(m.done, m.total, width-2) + "\n\n")

	// ── task list ─────────────────────────────────────────────────────────────
	// Determine how many log lines we can show given terminal height.
	// Reserve: header(3) + footer(2) + task list rows + spacing
	nonLogRows := 5 + len(m.entries)
	maxVisible := m.termHeight - nonLogRows
	if maxVisible < 3 {
		maxVisible = 3
	}

	for i, e := range m.entries {
		icon, label := taskIcon(e.status, i == m.activeIndex)
		durStr := ""
		if e.duration > 0 {
			durStr = runnerDim.Render("  " + e.duration.Truncate(time.Millisecond).String())
		}
		nameStr := e.name
		if i == m.activeIndex {
			nameStr = runnerRunning.Render(e.name)
		}
		fmt.Fprintf(&b, "  %s  %s%s\n", icon, nameStr, durStr)

		// Render error line under a failed task.
		if e.err != nil && e.status == store.TaskFailed {
			msg := ellipsis(e.err.Error(), width-8)
			b.WriteString(runnerFail.Render("      └─ "+msg) + "\n")
		}

		// Render streaming log for the active task.
		if i == m.activeIndex && len(e.logs) > 0 {
			// Pick last N lines that fit.
			show := e.logs
			if len(show) > maxVisible {
				show = show[len(show)-maxVisible:]
			}
			for _, line := range show {
				rendered := renderLogLine(line, width-6)
				b.WriteString("    " + rendered + "\n")
			}
		}
		_ = label
	}

	// ── footer ────────────────────────────────────────────────────────────────
	if m.finished {
		b.WriteString("\n" + runnerTitle.Render("All tasks completed! Press Enter or q to exit"))
	} else {
		b.WriteString("\n" + runnerDim.Render("ctrl+c to abort"))
	}
	return b.String()
}

// ── helpers ──────────────────────────────────────────────────────────────────

func taskIcon(status store.TaskStatus, active bool) (string, string) {
	switch {
	case active:
		return runnerRunning.Render("▶"), "running"
	case status == store.TaskSuccess:
		return runnerOK.Render("✓"), "ok"
	case status == store.TaskFailed:
		return runnerFail.Render("✗"), "failed"
	case status == store.TaskRolledBack:
		return runnerRollbk.Render("↺"), "rolled back"
	case status == store.TaskSkipped:
		return runnerSkip.Render("·"), "skipped"
	default:
		return runnerDim.Render("○"), "pending"
	}
}

func renderLogLine(line string, maxWidth int) string {
	// Highlight lines that look like shell commands (start with "$").
	if strings.HasPrefix(line, "$ ") {
		return runnerCmd.Render(ellipsis(line, maxWidth))
	}
	return runnerLogLine.Render(ellipsis(line, maxWidth))
}

func progressBar(done, total, width int) string {
	if total == 0 {
		return ""
	}
	if width < 10 {
		width = 10
	}
	barWidth := width - 12 // leave room for label
	if barWidth < 4 {
		barWidth = 4
	}
	filled := 0
	if total > 0 {
		filled = barWidth * done / total
	}
	pct := 0
	if total > 0 {
		pct = 100 * done / total
	}
	bar := runnerBar.Render(strings.Repeat("█", filled)) +
		runnerBarBg.Render(strings.Repeat("░", barWidth-filled))
	label := runnerDim.Render(fmt.Sprintf(" %d/%d  %3d%%", done, total, pct))
	return "  " + bar + label
}

func ellipsis(s string, max int) string {
	if max <= 0 {
		return s
	}
	if len(s) <= max {
		return s
	}
	if max <= 1 {
		return "…"
	}
	return s[:max-1] + "…"
}

// ── public API ────────────────────────────────────────────────────────────────

// RunProgress starts the live progress TUI and feeds task events from the
// engine's Progress callbacks. applyFn is called in a goroutine; it should
// call r.Apply(...) and send AllDoneMsg when finished.
//
// Usage:
//
//	results, err := tui.RunProgress(tasks, runner, "sur — hardening", func(send func(tea.Msg)) {
//	    results := runner.Apply(ctx, sessionID, tasks)
//	    send(tui.AllDoneMsg{Results: results})
//	})
func RunProgress(
	tasks []engine.RunnableTask,
	title string,
	applyFn func(send func(tea.Msg)),
) ([]engine.Result, error) {
	m := newRunnerModel(tasks, title)
	p := tea.NewProgram(m, tea.WithAltScreen())

	// Wire engine callbacks to send messages into the Bubble Tea program.
	send := func(msg tea.Msg) { p.Send(msg) }

	go applyFn(send)

	out, err := p.Run()
	if err != nil {
		return nil, err
	}
	final, ok := out.(runnerModel)
	if !ok {
		return nil, fmt.Errorf("unexpected model type")
	}
	return final.results, nil
}
