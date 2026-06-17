package components

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/user/nyaa-tui/internal/ui/theme"
	"github.com/user/nyaa-tui/internal/yt"
)

type List struct {
	items    []yt.Video
	cursor   int
	width    int
	height   int
	selected lipgloss.Style
	normal   lipgloss.Style
	dim      lipgloss.Style
}

func NewList(items []yt.Video, selected, normal, dim lipgloss.Style) List {
	return List{
		items:    items,
		cursor:   0,
		selected: selected,
		normal:   normal,
		dim:      dim,
	}
}

func (l *List) SetItems(items []yt.Video) {
	l.items = items
	l.cursor = 0
}

func (l *List) SetDimensions(width, height int) {
	l.width = width
	l.height = height
}

func (l *List) Cursor() int {
	return l.cursor
}

func (l *List) SelectedItem() *yt.Video {
	if len(l.items) == 0 {
		return nil
	}
	return &l.items[l.cursor]
}

func (l *List) Length() int {
	return len(l.items)
}

func (l *List) CursorUp() {
	if l.cursor > 0 {
		l.cursor--
	}
}

func (l *List) CursorDown() {
	if l.cursor < len(l.items)-1 {
		l.cursor++
	}
}

func (l *List) PageUp() {
	l.cursor -= 10
	if l.cursor < 0 {
		l.cursor = 0
	}
}

func (l *List) PageDown() {
	l.cursor += 10
	if l.cursor >= len(l.items) {
		l.cursor = len(l.items) - 1
	}
}

func (l *List) GoTop() {
	l.cursor = 0
}

func (l *List) GoBottom() {
	if len(l.items) > 0 {
		l.cursor = len(l.items) - 1
	}
}

func (l *List) VisibleCount() int {
	if l.height <= 0 {
		return len(l.items)
	}
	eachItemTakes := 2
	return (l.height - 2) / eachItemTakes
}

func (l *List) CurrentPage() int {
	if len(l.items) == 0 {
		return 1
	}
	visible := l.VisibleCount()
	if visible <= 0 {
		visible = 1
	}
	return (l.cursor / visible) + 1
}

func (l *List) TotalPages() int {
	if len(l.items) == 0 {
		return 1
	}
	visible := l.VisibleCount()
	if visible <= 0 {
		visible = 1
	}
	pages := len(l.items) / visible
	if len(l.items)%visible > 0 {
		pages++
	}
	return pages
}

func (l List) View() string {
	// Read styles from the live palette so an in-app theme switch recolors the
	// results list immediately (the cached l.selected/normal/dim are ignored).
	selected := theme.Theme.SelectedStyle
	normal := theme.Theme.BaseStyle
	dim := theme.Theme.DimStyle

	if len(l.items) == 0 {
		return dim.Render("no videos found yet~ " + theme.Theme.KaomojiStyle.Render("(´｡• ω •｡`)"))
	}

	eachItemTakes := 2
	visible := (l.height - 2) / eachItemTakes
	if visible <= 0 {
		visible = len(l.items)
	}

	start := 0
	end := visible
	if end > len(l.items) {
		end = len(l.items)
	}

	if l.cursor >= end {
		start = l.cursor - visible + 1
		end = l.cursor + 1
	}
	if l.cursor < start {
		start = l.cursor
		end = start + visible
		if end > len(l.items) {
			end = len(l.items)
		}
	}

	if start < 0 {
		start = 0
	}
	if end > len(l.items) {
		end = len(l.items)
	}

	var lines []string
	for i := start; i < end && i < len(l.items); i++ {
		item := &l.items[i]
		line := item.DisplayTitle()
		meta := item.DisplayMeta()

		if i == l.cursor {
			lines = append(lines, selected.Render("♡ "+line))
			lines = append(lines, dim.Render("   ♬ "+meta))
		} else {
			lines = append(lines, normal.Render("  "+line))
			lines = append(lines, dim.Render("    "+meta))
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}
