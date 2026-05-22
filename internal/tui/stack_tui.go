// Package tui — stack_tui.go provides the interactive stack management TUI.
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/suleymanmercan/sur/internal/stack"
)

// ── styles ────────────────────────────────────────────────────────────────────

var (
	stkTitle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7CE38B"))
	stkSub      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#74B9FF"))
	stkCursor   = lipgloss.NewStyle().Foreground(lipgloss.Color("#F28C28"))
	stkSelected = lipgloss.NewStyle().Foreground(lipgloss.Color("#7CE38B"))
	stkDim      = lipgloss.NewStyle().Faint(true)
	stkWarn     = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFBF00"))
	stkFail     = lipgloss.NewStyle().Foreground(lipgloss.Color("#F25F5C"))
	stkOK       = lipgloss.NewStyle().Foreground(lipgloss.Color("#7CE38B"))
	stkSecret   = lipgloss.NewStyle().Foreground(lipgloss.Color("#B0B8C1")).Faint(true)
	stkCustom   = lipgloss.NewStyle().Foreground(lipgloss.Color("#DDA0DD"))
)

// ── screen type ───────────────────────────────────────────────────────────────

type stkScreen int

const (
	stkMainMenu    stkScreen = iota // top level
	stkInstallList                  // choose a stack to install
	stkConfigForm                   // fill in config fields
	stkConfirmInstall               // confirm before running
	stkRunInstall                   // show install progress
	stkInstalledList                // choose an installed stack
	stkActionMenu                   // choose action for installed stack
	stkRunAction                    // show lifecycle action output
	stkLogView                      // show recent logs
)

// mainMenuItem is one row in the top-level menu.
type mainMenuItem struct {
	label string
	desc  string
}

var mainMenuItems = []mainMenuItem{
	{"Install stack", "Browse and install a new stack from the catalog"},
	{"Installed stacks", "Manage your running stacks"},
	{"Fetch / update catalog", "Re-download the stack catalog from GitHub"},
	{"Quit", ""},
}

type actionItem struct {
	label string
	key   string // internal action key
}

var actionItems = []actionItem{
	{"Status", "status"},
	{"Logs", "logs"},
	{"Edit config", "edit"},
	{"Restart", "restart"},
	{"Rotate password", "rotate"},
	{"Backup", "backup"},
	{"Update (pull + restart)", "update"},
	{"Stop (down)", "down"},
	{"← Back", "back"},
}

// ── model ─────────────────────────────────────────────────────────────────────

// StackModel is the Bubble Tea model for the full stack management TUI.
type StackModel struct {
	screen stkScreen

	// program reference — used to send streaming log messages from goroutines.
	prog *tea.Program

	// main menu
	mainCursor int

	// install list
	availableDefs []stack.StackDef
	installCursor int
	loadErr       string // non-empty when catalog fetch failed

	// config form
	selectedDef    stack.StackDef
	configValues   map[string]string // field.ID → entered value
	configCursor   int               // which field is being edited
	configEditing  bool              // user is typing in a field
	configInput    string            // current input buffer
	configErr      string

	// installed list
	installedStacks []stack.InstalledStack
	installedCursor int

	// action menu
	actionCursor int
	actionTarget stack.InstalledStack

	// output / log view
	outputLines []string
	outputErr   string

	// terminal size
	width  int
	height int

	// done / quit
	quit bool
}

// NewStackModel constructs the initial model.
func NewStackModel() StackModel {
	return StackModel{
		screen:       stkMainMenu,
		configValues: make(map[string]string),
		width:        100,
		height:       30,
	}
}

func (m StackModel) Init() tea.Cmd { return nil }

// ── update ────────────────────────────────────────────────────────────────────

func (m StackModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = v.Width
		m.height = v.Height

	case tea.KeyMsg:
		return m.handleKey(v.String())

	case catalogLoadedMsg:
		m.availableDefs = v.defs
		m.loadErr = v.err
		m.screen = stkInstallList

	case installedLoadedMsg:
		m.installedStacks = v.stacks
		m.installedCursor = 0
		m.screen = stkInstalledList

	case outputMsg:
		m.outputLines = append(m.outputLines, v.line)

	case actionDoneMsg:
		m.outputLines = append(m.outputLines, "")
		if v.err != "" {
			m.outputLines = append(m.outputLines, stkFail.Render("Error: "+v.err))
		} else {
			m.outputLines = append(m.outputLines, stkOK.Render("Done. Press Enter or q to go back."))
		}
		m.screen = stkRunAction

	case logLoadedMsg:
		m.outputLines = v.lines
		m.screen = stkLogView
	}

	return m, nil
}

func (m StackModel) handleKey(key string) (tea.Model, tea.Cmd) {
	switch m.screen {

	case stkMainMenu:
		return m.handleMainMenu(key)

	case stkInstallList:
		return m.handleInstallList(key)

	case stkConfigForm:
		return m.handleConfigForm(key)

	case stkConfirmInstall:
		return m.handleConfirmInstall(key)

	case stkRunInstall:
		return m.handleRunScreen(key)

	case stkInstalledList:
		return m.handleInstalledList(key)

	case stkActionMenu:
		return m.handleActionMenu(key)

	case stkRunAction, stkLogView:
		return m.handleRunScreen(key)
	}
	return m, nil
}

// ── main menu ─────────────────────────────────────────────────────────────────

func (m StackModel) handleMainMenu(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "ctrl+c", "q", "esc":
		m.quit = true
		return m, tea.Quit
	case "up", "k":
		if m.mainCursor > 0 {
			m.mainCursor--
		}
	case "down", "j":
		if m.mainCursor < len(mainMenuItems)-1 {
			m.mainCursor++
		}
	case "enter", " ":
		switch m.mainCursor {
		case 0: // Install stack
			m.loadErr = ""
			m.availableDefs = nil
			return m, loadCatalog()
		case 1: // Installed stacks
			return m, loadInstalled()
		case 2: // Fetch/update catalog
			m.outputLines = nil
			m.screen = stkRunInstall
			return m, refreshCatalog()
		case 3: // Quit
			m.quit = true
			return m, tea.Quit
		}
	}
	return m, nil
}

// ── install list ──────────────────────────────────────────────────────────────

func (m StackModel) handleInstallList(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "ctrl+c", "q", "esc":
		m.screen = stkMainMenu
	case "up", "k":
		if m.installCursor > 0 {
			m.installCursor--
		}
	case "down", "j":
		if m.installCursor < len(m.availableDefs)-1 {
			m.installCursor++
		}
	case "enter", " ":
		if len(m.availableDefs) == 0 {
			break
		}
		m.selectedDef = m.availableDefs[m.installCursor]
		m.configValues = make(map[string]string)
		m.configCursor = 0
		m.configEditing = false
		m.configInput = ""
		m.configErr = ""
		m.screen = stkConfigForm
	}
	return m, nil
}

// ── config form ───────────────────────────────────────────────────────────────

func (m StackModel) handleConfigForm(key string) (tea.Model, tea.Cmd) {
	fields := m.selectedDef.Config

	if m.configEditing {
		switch key {
		case "enter":
			// Commit the current input.
			f := fields[m.configCursor]
			m.configValues[f.ID] = m.configInput
			m.configEditing = false
			// Advance to next field automatically.
			if m.configCursor < len(fields)-1 {
				m.configCursor++
			}
		case "esc":
			m.configEditing = false
			m.configInput = ""
		case "ctrl+c":
			m.quit = true
			return m, tea.Quit
		case "backspace":
			if len(m.configInput) > 0 {
				m.configInput = m.configInput[:len(m.configInput)-1]
			}
		default:
			if len(key) == 1 {
				m.configInput += key
			}
		}
		return m, nil
	}

	switch key {
	case "ctrl+c", "q":
		m.screen = stkInstallList
	case "esc":
		m.screen = stkInstallList
	case "up", "k":
		if m.configCursor > 0 {
			m.configCursor--
		}
	case "down", "j":
		if m.configCursor < len(fields)-1 {
			m.configCursor++
		}
	case " ", "enter":
		if len(fields) == 0 {
			break
		}
		f := fields[m.configCursor]
		switch f.Type {
		case stack.FieldTypeSelect:
			// Cycle through options.
			current := m.configValues[f.ID]
			if current == "" {
				current = f.Default
			}
			opts := f.Options
			idx := 0
			for i, o := range opts {
				if o == current {
					idx = i
					break
				}
			}
			m.configValues[f.ID] = opts[(idx+1)%len(opts)]
		case stack.FieldTypeBool:
			current := m.configValues[f.ID]
			if current == "" {
				current = f.Default
			}
			if current == "true" {
				m.configValues[f.ID] = "false"
			} else {
				m.configValues[f.ID] = "true"
			}
		case stack.FieldTypeSecret:
			// For secrets, Enter opens text input (user can leave blank to auto-generate).
			m.configInput = m.configValues[f.ID]
			m.configEditing = true
		default:
			// text / number: open inline input.
			m.configInput = m.configValues[f.ID]
			if m.configInput == "" {
				m.configInput = f.Default
			}
			m.configEditing = true
		}
	case "i":
		// Shortcut: go to install confirm from anywhere in the form.
		m.screen = stkConfirmInstall
	}
	return m, nil
}

// ── confirm install ───────────────────────────────────────────────────────────

func (m StackModel) handleConfirmInstall(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "ctrl+c", "q", "esc":
		m.screen = stkConfigForm
	case "enter", "y":
		m.outputLines = nil
		m.screen = stkRunInstall
		return m, doInstall(m.prog, m.selectedDef, m.configValues)
	case "n":
		m.screen = stkConfigForm
	}
	return m, nil
}

// ── installed list ────────────────────────────────────────────────────────────

func (m StackModel) handleInstalledList(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "ctrl+c", "q", "esc":
		m.screen = stkMainMenu
	case "up", "k":
		if m.installedCursor > 0 {
			m.installedCursor--
		}
	case "down", "j":
		if m.installedCursor < len(m.installedStacks)-1 {
			m.installedCursor++
		}
	case "enter", " ":
		if len(m.installedStacks) == 0 {
			break
		}
		m.actionTarget = m.installedStacks[m.installedCursor]
		m.actionCursor = 0
		m.screen = stkActionMenu
	}
	return m, nil
}

// ── action menu ───────────────────────────────────────────────────────────────

func (m StackModel) handleActionMenu(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "ctrl+c", "q", "esc":
		m.screen = stkInstalledList
	case "up", "k":
		if m.actionCursor > 0 {
			m.actionCursor--
		}
	case "down", "j":
		if m.actionCursor < len(actionItems)-1 {
			m.actionCursor++
		}
	case "enter", " ":
		item := actionItems[m.actionCursor]
		switch item.key {
		case "back":
			m.screen = stkInstalledList
		case "logs":
			m.outputLines = nil
			return m, loadLogs(m.actionTarget.Dir)
		default:
			m.outputLines = nil
			m.screen = stkRunAction
			return m, doAction(m.prog, m.actionTarget, item.key)
		}
	}
	return m, nil
}

// ── run / output screen ───────────────────────────────────────────────────────

func (m StackModel) handleRunScreen(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "ctrl+c", "q", "esc", "enter":
		m.screen = stkMainMenu
		m.outputLines = nil
	}
	return m, nil
}

// ── view ──────────────────────────────────────────────────────────────────────

func (m StackModel) View() string {
	if m.quit {
		return ""
	}

	var b strings.Builder
	b.WriteString(stkTitle.Render("sur stack") + "\n\n")

	switch m.screen {
	case stkMainMenu:
		b.WriteString(renderMainMenu(m))
	case stkInstallList:
		b.WriteString(renderInstallList(m))
	case stkConfigForm:
		b.WriteString(renderConfigForm(m))
	case stkConfirmInstall:
		b.WriteString(renderConfirmInstall(m))
	case stkRunInstall, stkRunAction:
		b.WriteString(renderOutput(m, "Running…"))
	case stkInstalledList:
		b.WriteString(renderInstalledList(m))
	case stkActionMenu:
		b.WriteString(renderActionMenu(m))
	case stkLogView:
		b.WriteString(renderOutput(m, "Logs"))
	}

	return b.String()
}

// ── render helpers ────────────────────────────────────────────────────────────

func renderMainMenu(m StackModel) string {
	var b strings.Builder
	for i, item := range mainMenuItems {
		cursor := "  "
		if i == m.mainCursor {
			cursor = stkCursor.Render("➤ ")
		}
		line := cursor + item.label
		if i == m.mainCursor && item.desc != "" {
			line += "\n    " + stkDim.Render(item.desc)
		}
		b.WriteString(line + "\n")
	}
	b.WriteString("\n" + stkDim.Render("↑/↓ move • enter select • q quit"))
	return b.String()
}

func renderInstallList(m StackModel) string {
	var b strings.Builder
	b.WriteString(stkSub.Render("Install stack") + "\n\n")

	if m.loadErr != "" {
		b.WriteString(stkFail.Render("Catalog error: "+m.loadErr) + "\n")
	}

	if len(m.availableDefs) == 0 && m.loadErr == "" {
		b.WriteString(stkDim.Render("Loading catalog…") + "\n")
	}

	for i, def := range m.availableDefs {
		cursor := "  "
		if i == m.installCursor {
			cursor = stkCursor.Render("➤ ")
		}
		label := def.Name
		if def.Source == "custom" {
			label += " " + stkCustom.Render("[custom]")
		}
		if i == m.installCursor {
			label = stkSelected.Render(label)
			b.WriteString(cursor + label + "\n")
			b.WriteString("    " + stkDim.Render(def.Description) + "\n")
		} else {
			b.WriteString(cursor + label + "\n")
		}
	}
	b.WriteString("\n" + stkDim.Render("↑/↓ move • enter select • esc back"))
	return b.String()
}

func renderConfigForm(m StackModel) string {
	def := m.selectedDef
	var b strings.Builder
	b.WriteString(stkSub.Render("Configure: "+def.Name) + "\n\n")

	for i, f := range def.Config {
		cursor := "  "
		if i == m.configCursor {
			cursor = stkCursor.Render("➤ ")
		}

		val := m.configValues[f.ID]
		if val == "" {
			val = f.Default
		}

		display := val
		if f.Type == stack.FieldTypeSecret {
			if val == "" {
				display = stkDim.Render("(auto-generate)")
			} else {
				display = stkSecret.Render(strings.Repeat("●", min(len(val), 12)))
			}
		}

		// If this field is being edited, show input buffer.
		if i == m.configCursor && m.configEditing {
			if f.Type == stack.FieldTypeSecret {
				display = stkSecret.Render(strings.Repeat("●", len(m.configInput))) + stkCursor.Render("▌")
			} else {
				display = m.configInput + stkCursor.Render("▌")
			}
		}

		typeHint := string(f.Type)
		if f.Type == stack.FieldTypeSelect && len(f.Options) > 0 {
			typeHint = "select: " + strings.Join(f.Options, " | ")
		}

		line := fmt.Sprintf("%s%-20s %s", cursor, f.Label+":", display)
		b.WriteString(line + "\n")
		if i == m.configCursor {
			b.WriteString("    " + stkDim.Render(typeHint) + "\n")
		}
	}

	if m.configErr != "" {
		b.WriteString("\n" + stkFail.Render(m.configErr) + "\n")
	}

	b.WriteString("\n" + stkDim.Render("↑/↓ move • enter/space edit • i=install now • esc back"))
	return b.String()
}

func renderConfirmInstall(m StackModel) string {
	def := m.selectedDef
	var b strings.Builder
	b.WriteString(stkSub.Render("Confirm install: "+def.Name) + "\n\n")

	for _, f := range def.Config {
		val := m.configValues[f.ID]
		if val == "" {
			val = f.Default
		}
		display := val
		if f.Type == stack.FieldTypeSecret {
			if val == "" {
				display = stkDim.Render("(will be auto-generated)")
			} else {
				display = stkSecret.Render("●●●●●●●●")
			}
		}
		b.WriteString(fmt.Sprintf("  %-20s %s\n", f.Label+":", display))
	}

	b.WriteString("\n" + stkWarn.Render("Stack will be installed to /opt/sur/stacks/"+def.ID+"/") + "\n")
	b.WriteString("\n" + stkDim.Render("enter/y = install • n/esc = go back"))
	return b.String()
}

func renderInstalledList(m StackModel) string {
	var b strings.Builder
	b.WriteString(stkSub.Render("Installed stacks") + "\n\n")

	if len(m.installedStacks) == 0 {
		b.WriteString(stkDim.Render("No stacks installed yet.") + "\n")
	}

	for i, s := range m.installedStacks {
		cursor := "  "
		if i == m.installedCursor {
			cursor = stkCursor.Render("➤ ")
		}
		status := stkFail.Render("stopped")
		if s.Running {
			status = stkOK.Render("running")
		}
		b.WriteString(fmt.Sprintf("%s%-20s %s\n", cursor, s.Def.Name, status))
	}
	b.WriteString("\n" + stkDim.Render("↑/↓ move • enter select • esc back"))
	return b.String()
}

func renderActionMenu(m StackModel) string {
	var b strings.Builder
	b.WriteString(stkSub.Render(m.actionTarget.Def.Name) + "\n")
	status := stkFail.Render("stopped")
	if m.actionTarget.Running {
		status = stkOK.Render("running")
	}
	b.WriteString(stkDim.Render("Status: ") + status + "\n\n")

	for i, item := range actionItems {
		cursor := "  "
		if i == m.actionCursor {
			cursor = stkCursor.Render("➤ ")
		}
		b.WriteString(cursor + item.label + "\n")
	}
	b.WriteString("\n" + stkDim.Render("↑/↓ move • enter select • esc back"))
	return b.String()
}

func renderOutput(m StackModel, title string) string {
	var b strings.Builder
	b.WriteString(stkSub.Render(title) + "\n\n")

	// Show last N lines to fit terminal.
	lines := m.outputLines
	maxLines := m.height - 6
	if maxLines < 4 {
		maxLines = 4
	}
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	for _, l := range lines {
		b.WriteString("  " + l + "\n")
	}
	b.WriteString("\n" + stkDim.Render("enter/q = back to main menu"))
	return b.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ── tea.Cmd constructors ──────────────────────────────────────────────────────

// catalogLoadedMsg is sent when FetchIndex returns.
type catalogLoadedMsg struct {
	defs []stack.StackDef
	err  string
}

// installedLoadedMsg is sent after scanning InstallDir.
type installedLoadedMsg struct {
	stacks []stack.InstalledStack
}

// outputMsg carries a single log line from an async operation.
type outputMsg struct{ line string }

// actionDoneMsg signals that an async action has completed.
type actionDoneMsg struct{ err string }

// logLoadedMsg carries loaded log lines.
type logLoadedMsg struct{ lines []string }

func loadCatalog() tea.Cmd {
	return func() tea.Msg {
		metas, err := stack.FetchIndex()
		if err != nil {
			return catalogLoadedMsg{err: err.Error()}
		}
		// Fetch full StackDef for each meta.
		var defs []stack.StackDef
		for _, m := range metas {
			def, ferr := stack.FetchStackDef(m.ID)
			if ferr != nil {
				continue
			}
			defs = append(defs, def)
		}
		// Append custom stacks.
		custom, _ := stack.ListCustom()
		defs = append(defs, custom...)
		return catalogLoadedMsg{defs: defs}
	}
}

func loadInstalled() tea.Cmd {
	return func() tea.Msg {
		stacks, _ := stack.ListInstalled()
		return installedLoadedMsg{stacks: stacks}
	}
}

func refreshCatalog() tea.Cmd {
	return func() tea.Msg {
		err := stack.RefreshCache()
		if err != nil {
			return actionDoneMsg{err: err.Error()}
		}
		return actionDoneMsg{}
	}
}

// doInstall runs the stack installation in a goroutine and streams each log
// line into the TUI via prog.Send so the user can see live progress.
func doInstall(prog *tea.Program, def stack.StackDef, values map[string]string) tea.Cmd {
	return func() tea.Msg {
		go func() {
			err := stack.Install(def, values, func(line string) {
				prog.Send(outputMsg{line: line})
			})
			if err != nil {
				prog.Send(actionDoneMsg{err: err.Error()})
			} else {
				prog.Send(actionDoneMsg{})
			}
		}()
		// Return nil immediately; all updates come via prog.Send from the goroutine.
		return nil
	}
}

// doAction runs a lifecycle action in a goroutine and streams log lines live.
func doAction(prog *tea.Program, s stack.InstalledStack, action string) tea.Cmd {
	return func() tea.Msg {
		go func() {
			logFn := func(line string) {
				prog.Send(outputMsg{line: line})
			}
			var err error
			switch action {
			case "restart":
				err = stack.Restart(s.Dir, logFn)
			case "rotate":
				err = stack.Rotate(s.Dir, logFn)
			case "down":
				err = stack.Down(s.Dir, logFn)
			case "update":
				err = stack.Update(s.Dir, logFn)
			case "backup":
				err = stack.Backup(s.Dir, logFn)
			case "status":
				rows, serr := stack.Status(s.Dir)
				if serr != nil {
					prog.Send(actionDoneMsg{err: serr.Error()})
					return
				}
				var lines []string
				for _, r := range rows {
					lines = append(lines, fmt.Sprintf("  %-30s %s", r.Name, r.State))
				}
				prog.Send(logLoadedMsg{lines: lines})
				return
			}
			if err != nil {
				prog.Send(actionDoneMsg{err: err.Error()})
			} else {
				prog.Send(actionDoneMsg{})
			}
		}()
		return nil
	}
}

func loadLogs(dir string) tea.Cmd {
	return func() tea.Msg {
		out, err := stack.Logs(dir, 80)
		if err != nil {
			return logLoadedMsg{lines: []string{stkFail.Render("Error: " + err.Error())}}
		}
		lines := strings.Split(out, "\n")
		return logLoadedMsg{lines: lines}
	}
}

// ── public entry point ────────────────────────────────────────────────────────

// RunStack launches the full stack management TUI.
func RunStack() error {
	m := NewStackModel()
	p := tea.NewProgram(m, tea.WithAltScreen())
	// Store the program reference in the model so commands can stream messages.
	m.prog = p
	_, err := p.Run()
	return err
}
