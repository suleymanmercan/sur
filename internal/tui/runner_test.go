package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestRunnerFinishedEnterReturnsToPicker(t *testing.T) {
	model := runnerModel{finished: true}

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	final := updated.(runnerModel)

	if !final.goBack {
		t.Fatal("expected enter after completion to return to the task picker")
	}
}

func TestRunnerFinishedQQuitsWithoutReturningToPicker(t *testing.T) {
	model := runnerModel{finished: true}

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	final := updated.(runnerModel)

	if final.goBack {
		t.Fatal("expected q after completion to quit instead of returning to the task picker")
	}
}
