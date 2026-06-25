package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/user/nyaa/internal/ui/views"
	"github.com/user/nyaa/internal/yt"
)

func resultsModel(t *testing.T) Model {
	t.Helper()
	m := newTestModel(t)
	m.width, m.height = 80, 24
	var videos []yt.Video
	for i := 0; i < 20; i++ {
		videos = append(videos, yt.Video{
			ID: string(rune('a' + i)), Title: "Vid", Channel: "Chan",
			URL: "https://youtu.be/" + string(rune('a'+i)),
		})
	}
	m.results = views.NewResultsList(videos, 80, 24)
	m.state = StateResults
	return m
}

// Hovering over a row highlights it (moves the cursor) without activating.
func TestMouseHoverHighlights(t *testing.T) {
	m := resultsModel(t)
	// Row 6 at 80x24 is item 1's title row (see views layout tests).
	res, _ := m.Update(tea.MouseMsg{Y: 6, Action: tea.MouseActionMotion, Button: tea.MouseButtonNone})
	m = asModel(res)
	if got := m.results.SelectedVideo(); got == nil || got.ID != "b" {
		t.Errorf("after hover, selected = %v, want item 'b'", got)
	}
	if m.state != StateResults {
		t.Errorf("hover should not change state, got %v", m.state)
	}
}

// A left click highlights then activates (plays) the row under the pointer.
func TestMouseLeftClickPlays(t *testing.T) {
	m := resultsModel(t)
	res, cmd := m.Update(tea.MouseMsg{Y: 4, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	m = asModel(res)
	if m.state != StatePlaying {
		t.Fatalf("left click state = %v, want StatePlaying", m.state)
	}
	if m.nowPlaying == nil || m.nowPlaying.ID != "a" {
		t.Errorf("nowPlaying = %v, want item 'a'", m.nowPlaying)
	}
	if cmd == nil {
		t.Error("expected a playback command from left click")
	}
}

// Right click goes back one screen (results → menu).
func TestMouseRightClickGoesBack(t *testing.T) {
	m := resultsModel(t)
	res, _ := m.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonRight})
	m = asModel(res)
	if m.state != StateMenu {
		t.Errorf("right click from results state = %v, want StateMenu", m.state)
	}
}

// Right click at the main menu exits the app.
func TestMouseRightClickMenuQuits(t *testing.T) {
	m := newTestModel(t)
	m.state = StateMenu
	_, cmd := m.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonRight})
	if cmd == nil {
		t.Fatal("expected a quit command from right click at menu")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("expected tea.QuitMsg from right click at menu")
	}
}

// The wheel scrolls the selection.
func TestMouseWheelScrolls(t *testing.T) {
	m := resultsModel(t)
	res, _ := m.Update(tea.MouseMsg{Button: tea.MouseButtonWheelDown})
	m = asModel(res)
	if got := m.results.SelectedVideo(); got == nil || got.ID != "b" {
		t.Errorf("after wheel down, selected = %v, want item 'b'", got)
	}
}

// currentVideoURL reports the highlighted result's link for the c/C copy shortcut.
func TestCurrentVideoURL(t *testing.T) {
	m := resultsModel(t)
	if got := m.currentVideoURL(); got != "https://youtu.be/a" {
		t.Errorf("currentVideoURL = %q, want the first result's URL", got)
	}
	// On a screen with no video selected there's nothing to copy.
	m.state = StateMenu
	if got := m.currentVideoURL(); got != "" {
		t.Errorf("currentVideoURL on menu = %q, want empty", got)
	}
}

// `C` while typing a search query must be typed, not intercepted as the copy
// shortcut.
func TestCopyShortcutIgnoredInSearch(t *testing.T) {
	m := newTestModel(t)
	m.state = StateSearch
	res, _ := m.Update(key("C"))
	m = asModel(res)
	if got := m.search.Value(); got != "C" {
		t.Errorf("search value = %q, want \"C\" (the key should be typed, not a copy shortcut)", got)
	}
	if m.toast != "" {
		t.Errorf("no copy toast expected in search, got %q", m.toast)
	}
}
