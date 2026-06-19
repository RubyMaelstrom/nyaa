package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/nyaa-tui/internal/ui/theme"
)

// handleMouse translates mouse events into the same actions as the keyboard:
//
//   - hover (motion) highlights the row under the pointer,
//   - left click highlights then activates it (move forward, like Enter),
//   - right click goes back one screen (and quits at the main menu),
//   - the wheel scrolls the selection.
func (m *Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// Right click: back one screen (quit at the main menu).
	if msg.Button == tea.MouseButtonRight && msg.Action == tea.MouseActionPress {
		return m.mouseBack()
	}

	// Wheel scroll nudges the selection.
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		return m.scrollSelection(-1)
	case tea.MouseButtonWheelDown:
		return m.scrollSelection(1)
	}

	// Hover (and the press half of a left click) highlights the row under the
	// pointer; a left press then activates it.
	if idx, ok := m.itemAtScreen(msg.Y); ok {
		m.setCursor(idx)
		if msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress {
			return m.mouseActivate()
		}
	}
	return m, nil
}

// mouseBack mirrors pressing Esc for the current screen — and Quit at the menu,
// since that's the "main screen" with nowhere further back to go.
func (m *Model) mouseBack() (tea.Model, tea.Cmd) {
	esc := tea.KeyMsg{Type: tea.KeyEsc}
	switch m.state {
	case StateMenu:
		return m, tea.Quit
	case StateHelp:
		m.help.Hide()
		m.state = m.helpReturn
		return m, nil
	case StateSearch, StateLoading:
		m.state = StateMenu
		return m, nil
	case StatePlaying:
		return m, nil // mpv owns playback; nothing to back out of here
	case StateResults:
		return m.updateResults(esc)
	case StateSubscription:
		return m.updateSubscriptions(esc)
	case StateChannel:
		return m.updateChannel(esc)
	case StateError:
		return m.updateError(esc)
	}
	return m, nil
}

// mouseActivate performs the current screen's Enter action (play / expand /
// select), reusing the keyboard handlers so the behavior can't drift.
func (m *Model) mouseActivate() (tea.Model, tea.Cmd) {
	enter := tea.KeyMsg{Type: tea.KeyEnter}
	switch m.state {
	case StateMenu:
		return m.updateMenu(enter)
	case StateResults:
		return m.updateResults(enter)
	case StateSubscription:
		return m.updateSubscriptions(enter)
	case StateChannel:
		return m.updateChannel(enter)
	case StateError:
		return m.updateError(enter)
	}
	return m, nil
}

// scrollSelection moves the selection by one row in the active list, reusing the
// arrow-key handlers (delta < 0 = up, delta > 0 = down).
func (m *Model) scrollSelection(delta int) (tea.Model, tea.Cmd) {
	key := tea.KeyMsg{Type: tea.KeyUp}
	if delta > 0 {
		key = tea.KeyMsg{Type: tea.KeyDown}
	}
	switch m.state {
	case StateMenu:
		return m.updateMenu(key)
	case StateResults:
		return m.updateResults(key)
	case StateSubscription:
		return m.updateSubscriptions(key)
	case StateChannel:
		return m.updateChannel(key)
	case StateError:
		return m.updateError(key)
	}
	return m, nil
}

// itemAtScreen resolves the list index under screen row y for the active view.
// It first re-applies the dimensions/toast height the renderer uses so the
// hit-test geometry matches what's on screen.
func (m *Model) itemAtScreen(y int) (int, bool) {
	toast := m.toastLines()
	switch m.state {
	case StateMenu:
		m.menu.SetDimensions(m.width, m.height)
		m.menu.SetToastLines(toast)
		return m.menu.ItemAt(y)
	case StateResults:
		m.results.SetDimensions(m.width, m.height)
		m.results.SetToastLines(toast)
		return m.results.ItemAt(y)
	case StateChannel:
		m.channelBrowse.SetDimensions(m.width, m.height)
		m.channelBrowse.SetToastLines(toast)
		return m.channelBrowse.ItemAt(y)
	case StateSubscription:
		m.subscriptionsView.SetDimensions(m.width, m.height)
		m.subscriptionsView.SetToastLines(toast)
		return m.subscriptionsView.ItemAt(y)
	}
	return 0, false
}

// setCursor highlights index idx in the active list view.
func (m *Model) setCursor(idx int) {
	switch m.state {
	case StateMenu:
		m.menu.SetCursor(idx)
	case StateResults:
		m.results.SetCursor(idx)
	case StateChannel:
		m.channelBrowse.SetCursor(idx)
	case StateSubscription:
		m.subscriptionsView.SetCursor(idx)
	}
}

// toastBlock renders the active toast card and its height, or ("", 0) when no
// toast is showing. The height is reserved at the bottom of every list view.
func (m Model) toastBlock() (string, int) {
	if m.toast == "" || m.toastTimer <= 0 {
		return "", 0
	}
	block := theme.Theme.CardStyle.Render(theme.Theme.KaomojiStyle.Render(m.toast))
	return block, lipgloss.Height(block)
}

func (m Model) toastLines() int {
	_, n := m.toastBlock()
	return n
}
