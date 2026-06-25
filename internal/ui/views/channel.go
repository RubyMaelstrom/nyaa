package views

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/nyaa/internal/rss"
	"github.com/user/nyaa/internal/ui/theme"
)

// ChannelBrowse lists a single channel's recent uploads (from its RSS feed) so
// the user can browse beyond a single search result. Enter emits a PlayVideoMsg,
// reusing the same playback path as the subscriptions view.
type ChannelBrowse struct {
	channelID string
	name      string
	items     []rss.RSSItem
	cursor    int
	// offset is the first item kept on screen, decoupled from the cursor so mouse
	// hover never scrolls the list (see EnsureCursorVisible in components.List).
	offset      int
	width       int
	height      int
	toastHeight int
}

func NewChannelBrowse(channelID, name string, items []rss.RSSItem) ChannelBrowse {
	return ChannelBrowse{channelID: channelID, name: name, items: items}
}

func (c *ChannelBrowse) SetDimensions(width, height int) {
	c.width = width
	c.height = height
}

func (c *ChannelBrowse) SetToastLines(n int) {
	c.toastHeight = n
}

func (c ChannelBrowse) ChannelID() string { return c.channelID }
func (c ChannelBrowse) Name() string      { return c.name }

func (c ChannelBrowse) SelectedItem() *rss.RSSItem {
	if c.cursor < 0 || c.cursor >= len(c.items) {
		return nil
	}
	return &c.items[c.cursor]
}

// SetCursor moves the selection to index i (clamped); used by mouse hover.
func (c *ChannelBrowse) SetCursor(i int) {
	if i < 0 {
		i = 0
	}
	if i > len(c.items)-1 {
		i = len(c.items) - 1
	}
	if i < 0 {
		i = 0
	}
	c.cursor = i
}

func (c ChannelBrowse) titleView() string {
	return theme.Theme.TitleStyle.Render("♡ " + c.name + " ♡")
}

func (c ChannelBrowse) footerView() string {
	return theme.Theme.DimStyle.Render("↑↓ navigate  •  enter play  •  s subscribe  •  a audio  •  esc back")
}

// ItemAt maps a mouse Y coordinate to the video index under it, or ok=false when
// the pointer is outside the list rows. Each item is a single row here.
func (c ChannelBrowse) ItemAt(y int) (int, bool) {
	if len(c.items) == 0 {
		return 0, false
	}
	title := c.titleView()
	_, innerH, bodyTop := frameGeometry(c.width, c.height-c.toastHeight, title, c.footerView())
	row := y - bodyTop
	if row < 0 || row >= innerH {
		return 0, false
	}
	start := clampStart(c.offset, len(c.items), innerH)
	idx := start + row
	if idx < 0 || idx >= len(c.items) {
		return 0, false
	}
	return idx, true
}

// ensureVisible scrolls the offset so the cursor stays on screen after keyboard
// or wheel navigation (each item is one row here).
func (c *ChannelBrowse) ensureVisible() {
	_, innerH, _ := frameGeometry(c.width, c.height-c.toastHeight, c.titleView(), c.footerView())
	c.offset = ensureRowVisible(c.offset, c.cursor, len(c.items), innerH)
}

func (c ChannelBrowse) Update(msg tea.Msg) (ChannelBrowse, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if c.cursor > 0 {
				c.cursor--
			}
		case "down", "j":
			if c.cursor < len(c.items)-1 {
				c.cursor++
			}
		case "pageup", "ctrl+u":
			c.cursor -= 10
			if c.cursor < 0 {
				c.cursor = 0
			}
		case "pagedown", "ctrl+d":
			c.cursor += 10
			if c.cursor > len(c.items)-1 {
				c.cursor = len(c.items) - 1
			}
		case "home", "g":
			c.cursor = 0
		case "end", "G":
			c.cursor = len(c.items) - 1
		case "enter":
			if item := c.SelectedItem(); item != nil {
				play := PlayVideoMsg{Video: *item}
				return c, func() tea.Msg { return play }
			}
		}
		c.ensureVisible()
	}
	return c, nil
}

func (c ChannelBrowse) View() string {
	title := c.titleView()
	footer := c.footerView()

	if len(c.items) == 0 {
		body := theme.Theme.DimStyle.Render("no videos found for this channel~ (・_・;)")
		return Frame(c.width, c.height-c.toastHeight, title, footer, lipgloss.Center, lipgloss.Center,
			func(innerW, innerH int) string { return body })
	}

	if c.cursor >= len(c.items) {
		c.cursor = len(c.items) - 1
	}

	rows := make([]string, 0, len(c.items))
	for i, item := range c.items {
		line := item.PublishedAt.Format("Jan 2, 2006") + "  ♡  " + item.Title
		if i == c.cursor {
			rows = append(rows, theme.Theme.SelectedStyle.Render("▸ "+line))
		} else {
			rows = append(rows, theme.Theme.BaseStyle.Render("  • "+line))
		}
	}

	return Frame(c.width, c.height-c.toastHeight, title, footer, lipgloss.Left, lipgloss.Top,
		func(innerW, innerH int) string {
			return windowedLinesFrom(rows, c.offset, innerW, innerH)
		})
}
