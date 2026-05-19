// Package tui renders the Bubble Tea interactive checkbox list for sur task sets.
package tui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/suleymanmercan/sur/internal/engine"
)

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7CE38B"))
	helpStyle     = lipgloss.NewStyle().Faint(true)
	cursorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#F28C28"))
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#7CE38B"))
	dangerStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#F25F5C"))
	riskHigh      = lipgloss.NewStyle().Foreground(lipgloss.Color("#F25F5C")).Bold(true)
	riskMed       = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFBF00"))
	riskLow       = lipgloss.NewStyle().Foreground(lipgloss.Color("#7CE38B"))
)

// Run shows the hardening picker and returns the user's selected tasks.
func Run(tasks []engine.RunnableTask) (selected []engine.RunnableTask, canceled bool, err error) {
	return RunWithTitle(tasks, "sur — choose hardening tasks")
}

// RunWithTitle shows the picker with a custom title.
// canceled=true when the user pressed q/esc.
func RunWithTitle(tasks []engine.RunnableTask, title string) (selected []engine.RunnableTask, canceled bool, err error) {
	m := initialModel(tasks, title)
	p := tea.NewProgram(m, tea.WithInput(os.Stdin), tea.WithOutput(os.Stderr))
	out, err := p.Run()
	if err != nil {
		return nil, false, err
	}
	final := out.(model)
	if final.quit {
		return nil, true, nil
	}
	for i, t := range final.tasks {
		if final.selected[i] {
			selected = append(selected, t)
		}
	}
	return selected, false, nil
}

type model struct {
	tasks    []engine.RunnableTask
	title    string
	cursor   int
	selected map[int]bool
	confirm  bool
	quit     bool
	height   int
}

func initialModel(tasks []engine.RunnableTask, title string) model {
	return model{tasks: tasks, title: title, selected: make(map[int]bool), height: 12}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quit = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.tasks)-1 {
				m.cursor++
			}
		case " ":
			m.selected[m.cursor] = !m.selected[m.cursor]
		case "a":
			for i := range m.tasks {
				m.selected[i] = true
			}
		case "n":
			m.selected = map[int]bool{}
		case "enter":
			m.confirm = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.confirm || m.quit {
		return ""
	}
	var b strings.Builder
	b.WriteString(titleStyle.Render(m.title) + "\n\n")

	for i, t := range m.tasks {
		cursor := "  "
		if i == m.cursor {
			cursor = cursorStyle.Render("➤ ")
		}
		box := "[ ]"
		if m.selected[i] {
			box = selectedStyle.Render("[x]")
		}

		risk := riskLow.Render("low")
		switch strings.ToLower(t.GetRiskLevel()) {
		case "medium":
			risk = riskMed.Render("med")
		case "high":
			risk = riskHigh.Render("high")
		}

		name := t.GetName()
		if name == "" {
			name = t.GetID()
		}
		line := fmt.Sprintf("%s%s  %s  (%s)", cursor, box, name, risk)
		if !t.GetRollbackPossible() {
			line += dangerStyle.Render("  ⚠ no rollback")
		}
		b.WriteString(line + "\n")
		if i == m.cursor && t.GetDescription() != "" {
			b.WriteString(helpStyle.Render("       "+t.GetDescription()) + "\n")
		}
	}

	b.WriteString("\n" + helpStyle.Render("↑/↓ move • space toggle • a=all • n=none • enter=apply • q=quit"))
	return b.String()
}
