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

// titleView picks the banner or the compact header based on the room available.
// Shared with ItemAt so mouse hit-testing measures the same title height.
func (m Menu) titleView() string {
	// The full gradient wordmark needs vertical and horizontal room; on tight
	// terminals fall back to the one-line header so nothing overflows. The
	// leading blank line gives the banner some breathing room from the top edge.
	if m.height-m.toastHeight >= 17 && m.width >= 30 {
		return "\n" + theme.RenderBanner()
	}
	return theme.Theme.TitleStyle.Render(theme.SimpleHeader)
}

// bodyView builds the greeting + the three menu rows. The layout (a greeting, a
// blank line, then one row per item) is what menuFirstItemRow assumes.
func (m Menu) bodyView() string {
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
	return lipgloss.JoinVertical(lipgloss.Center, lines...)
}

func (m Menu) footerView() string {
	return theme.Theme.DimStyle.Render("↑↓ / tab navigate  •  enter select  •  t theme  •  q quit  •  ? help")
}

func (m Menu) View() string {
	// An active toast reserves its own rows at the bottom; shrink the frame by
	// that much so the banner is never shoved off the top of the terminal.
	title := m.titleView()
	body := m.bodyView()
	return Frame(m.width, m.height-m.toastHeight, title, m.footerView(), lipgloss.Center, lipgloss.Center,
		func(innerW, innerH int) string { return body })
}

// SetCursor moves the selection to item i, clamped to the three menu entries.
func (m *Menu) SetCursor(i int) {
	if i < 0 {
		i = 0
	}
	if i > 2 {
		i = 2
	}
	m.cursor = i
}

// ItemAt maps a mouse Y coordinate to the menu item under it. The body is
// centered vertically, so it accounts for the centering pad the frame inserts.
func (m Menu) ItemAt(y int) (int, bool) {
	title := m.titleView()
	body := m.bodyView()
	_, innerH, bodyTop := frameGeometry(m.width, m.height-m.toastHeight, title, m.footerView())

	topPad := (innerH - lipgloss.Height(body)) / 2 // mirrors lipgloss.Place(Center)
	if topPad < 0 {
		topPad = 0
	}
	// The first two body rows are the greeting and a blank; items follow.
	idx := y - (bodyTop + topPad + 2)
	if idx < 0 || idx > 2 {
		return 0, false
	}
	return idx, true
}

func (m Menu) GetSelected() MenuSelection {
	return MenuSelection(m.cursor)
}
