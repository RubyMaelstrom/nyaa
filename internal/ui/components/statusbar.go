package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/user/nyaa/internal/ui/theme"
)

type StatusBar struct {
	width int
	page  int
	total int
	style lipgloss.Style
}

func NewStatusBar(page, total int, style lipgloss.Style) StatusBar {
	return StatusBar{
		page:  page,
		total: total,
		style: style,
	}
}

func (s *StatusBar) SetPage(page, total int) {
	s.page = page
	s.total = total
}

func (s *StatusBar) SetWidth(w int) {
	s.width = w
}

func (s StatusBar) View() string {
	text := fmt.Sprintf("♡ page %d/%d · ↑↓ navigate · enter play · c chan · s sub · a audio · t theme · ?",
		s.page, s.total)

	// Live palette so theme switches recolor the bar (cached s.style is ignored).
	style := theme.Theme.StatusBarStyle
	if s.width > 0 {
		return style.Width(s.width).Render(text)
	}
	return style.Render(text)
}
