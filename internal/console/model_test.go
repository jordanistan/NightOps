package console

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestOriginSelectionMovesToHome(t *testing.T) {
	model := New(MissionControlTheme())
	updated, _ := model.Update(startupDoneMsg{})
	updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	view := updated.(Model).View()
	if !strings.Contains(view, "HOME BASE // Home Base") {
		t.Fatalf("origin selection did not enter home: %s", view)
	}
}

func TestStartupViewHasMissionControlTitle(t *testing.T) {
	if !strings.Contains(New(MissionControlTheme()).View(), "NIGHTOPS // MISSION CONTROL") {
		t.Fatal("startup title missing")
	}
}
