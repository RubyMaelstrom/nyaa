package components

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/user/nyaa/internal/ui/theme"
	"github.com/user/nyaa/internal/yt"
)

type List struct {
	items  []yt.Video
	cursor int
	// offset is the index of the first item kept on screen. It is decoupled from
	// the cursor so mouse hover (which moves the cursor) never scrolls the list;
	// only keyboard/wheel navigation calls EnsureCursorVisible to pull it along.
	offset   int
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
	l.offset = 0
}

func (l *List) SetDimensions(width, height int) {
	l.width = width
	l.height = height
}

func (l *List) Cursor() int {
	return l.cursor
}

// SetCursor moves the selection to index i, clamped to the list bounds. Used by
// mouse hover to highlight the row under the pointer; it deliberately leaves the
// scroll offset alone so hovering never scrolls the list.
func (l *List) SetCursor(i int) {
	if i < 0 {
		i = 0
	}
	if i > len(l.items)-1 {
		i = len(l.items) - 1
	}
	if i < 0 {
		i = 0
	}
	l.cursor = i
}

// EnsureCursorVisible scrolls the offset the minimum amount so the cursor sits
// within the visible window of `visible` items. Called after keyboard/wheel
// navigation (not hover), so the list only moves when the user asks it to.
func (l *List) EnsureCursorVisible(visible int) {
	if visible <= 0 {
		return
	}
	if l.cursor < l.offset {
		l.offset = l.cursor
	}
	if l.cursor >= l.offset+visible {
		l.offset = l.cursor - visible + 1
	}
	if max := len(l.items) - visible; l.offset > max {
		l.offset = max
	}
	if l.offset < 0 {
		l.offset = 0
	}
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

// window returns the [start, end) slice of items kept on screen for a body that
// fits `visible` items. It is anchored to the scroll offset (clamped so the
// window never runs past the end), not the cursor — keeping the offset in sync
// with the cursor is EnsureCursorVisible's job, called only on keyboard/wheel
// navigation so hover doesn't scroll.
func (l List) window(visible int) (start, end int) {
	if visible <= 0 {
		visible = len(l.items)
	}
	start = l.offset
	if max := len(l.items) - visible; start > max {
		start = max
	}
	if start < 0 {
		start = 0
	}
	end = start + visible
	if end > len(l.items) {
		end = len(l.items)
	}
	return start, end
}

// IndexAtRow maps a body row (0-based, within a body bodyRows tall) to the item
// index it shows, or ok=false if that row holds no item. Each item occupies two
// rows (title + meta), so this mirrors View's layout for mouse hit-testing.
func (l List) IndexAtRow(bodyRow, bodyRows int) (int, bool) {
	if len(l.items) == 0 || bodyRow < 0 {
		return 0, false
	}
	visible := bodyRows / 2
	if visible <= 0 {
		visible = len(l.items)
	}
	start, end := l.window(visible)
	idx := start + bodyRow/2
	if idx < start || idx >= end {
		return 0, false
	}
	return idx, true
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

	visible := (l.height - 2) / 2
	if visible <= 0 {
		visible = len(l.items)
	}
	start, end := l.window(visible)

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
