package views

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/nyaa-tui/internal/ui/components"
	"github.com/user/nyaa-tui/internal/ui/theme"
)

type MenuSelection int

const (
	MenuSearch MenuSelection = iota
	MenuSubscriptions
	MenuQuit
)

type Menu struct {
	cursor      int
	subCount    int
	width       int
	height      int
	toastHeight int
	greeting    string
}

func NewMenu(subCount int) Menu {
	return Menu{
		cursor:   0,
		subCount: subCount,
		greeting: components.RandomGreeting(),
	}
}

func (m *Menu) SetDimensions(width, height int) {
	m.width = width
	m.height = height
}

// SetToastLines reserves rows for an active toast so the menu shrinks to fit
// instead of being pushed off the top of the terminal.
func (m *Menu) SetToastLines(n int) {
	m.toastHeight = n
}

func (m Menu) Init() tea.Cmd {
	return nil
}

func (m Menu) Update(msg tea.Msg) (Menu, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k", "shift+tab":
			if m.cursor > 0 {
				m.cursor--
			} else {
				m.cursor = 2
			}
		case "down", "j", "tab":
			if m.cursor < 2 {
				m.cursor++
			} else {
				m.cursor = 0
			}
		}
	}
	return m, nil
}

func (m *Menu) SetSubscriptionsCount(count int) {
	m.subCount = count
}

func (m Menu) View() string {
	// An active toast reserves its own rows at the bottom; shrink the frame by
	// that much so the banner is never shoved off the top of the terminal.
	height := m.height - m.toastHeight

	// The full gradient wordmark needs vertical and horizontal room; on tight
	// terminals fall back to the one-line header so nothing overflows. The
	// leading blank line gives the banner some breathing room from the top edge.
	var title string
	if height >= 17 && m.width >= 30 {
		title = "\n" + theme.RenderBanner()
	} else {
		title = theme.Theme.TitleStyle.Render(theme.SimpleHeader)
	}

	pluralSuffix := "s"
	if m.subCount == 1 {
		pluralSuffix = ""
	}
	items := []string{
		"search  ♡  find youtube videos",
		fmt.Sprintf("subscriptions  ♡  feed from %d channel%s", m.subCount, pluralSuffix),
		"quit  ♡  see you soon~",
	}

	greeting := theme.Theme.KaomojiStyle.Render(m.greeting + "  welcome back, nya~")

	lines := []string{greeting, ""}
	for i, item := range items {
		if i == m.cursor {
			lines = append(lines, theme.Theme.SelectedStyle.Render("♡ "+item))
		} else {
			lines = append(lines, theme.Theme.BaseStyle.Render("   "+item))
		}
	}
	body := lipgloss.JoinVertical(lipgloss.Center, lines...)

	footer := theme.Theme.DimStyle.Render("↑↓ / tab navigate  •  enter select  •  t theme  •  q quit  •  ? help")

	return Frame(m.width, height, title, footer, lipgloss.Center, lipgloss.Center,
		func(innerW, innerH int) string { return body })
}

func (m Menu) GetSelected() MenuSelection {
	return MenuSelection(m.cursor)
}
