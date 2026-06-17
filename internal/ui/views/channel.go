package views

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/nyaa-tui/internal/rss"
	"github.com/user/nyaa-tui/internal/ui/theme"
)

// ChannelBrowse lists a single channel's recent uploads (from its RSS feed) so
// the user can browse beyond a single search result. Enter emits a PlayVideoMsg,
// reusing the same playback path as the subscriptions view.
type ChannelBrowse struct {
	channelID   string
	name        string
	items       []rss.RSSItem
	cursor      int
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
	}
	return c, nil
}

func (c ChannelBrowse) View() string {
	title := theme.Theme.TitleStyle.Render("♡ " + c.name + " ♡")
	footer := theme.Theme.DimStyle.Render("↑↓ navigate  •  enter play  •  s subscribe  •  a audio  •  esc back")

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
			return windowedLines(rows, c.cursor, innerW, innerH)
		})
}
