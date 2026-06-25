package views

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/nyaa/internal/ui/theme"
)

type HelpOverlay struct {
	show bool
}

type helpKey struct {
	keys string
	desc string
}

func NewHelpOverlay() HelpOverlay {
	return HelpOverlay{show: false}
}

func (h HelpOverlay) Init() tea.Cmd {
	return nil
}

func (h HelpOverlay) Update(msg tea.Msg) (HelpOverlay, tea.Cmd) {
	return h, nil
}

func (h HelpOverlay) View() string {
	if !h.show {
		return ""
	}

	keyStyle := theme.Theme.PromptStyle
	title := theme.Theme.TitleStyle.Render("(◕ᴗ◕✿) help ♡")

	keys := []helpKey{
		{"↑ / k", "Move selection up"},
		{"↓ / j", "Move selection down"},
		{"Mouse", "Hover to highlight • click to play • right-click back"},
		{"Enter", "Play selected / Submit / Expand"},
		{"Esc", "Cancel / Go back"},
		{"c", "Copy the video's link to the clipboard"},
		{"o", "Browse the channel's videos (results)"},
		{"a", "Toggle audio-only playback"},
		{"s", "Subscribe to channel"},
		{"t", "Cycle the color theme ♡"},
		{"/", "Back to menu (results)"},
		{"g / Home", "Go to top of results"},
		{"G / End", "Go to bottom of results"},
		{"Ctrl+u / Ctrl+d", "Page up / down"},
		{"Tab", "Channels ↔ recent (subscriptions)"},
		{"x", "Unsubscribe (subscriptions)"},
		{"r", "Refresh feeds / retry search"},
		{"d", "Mark all read (subscriptions)"},
		{"?", "Toggle this help"},
		{"q", "Quit"},
	}

	var contentLines []string
	for _, k := range keys {
		line := fmt.Sprintf("  %s    %s", keyStyle.Render(k.keys), k.desc)
		contentLines = append(contentLines, line)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, contentLines...)

	footer := theme.Theme.DimStyle.Render("press ? to close ♡")

	body := lipgloss.JoinVertical(lipgloss.Left,
		content,
		"",
		footer,
	)

	return lipgloss.JoinVertical(lipgloss.Center,
		"",
		title,
		"",
		theme.Theme.BoxStyle.Render(body),
	)
}

func (h *HelpOverlay) Toggle() {
	h.show = !h.show
}

func (h *HelpOverlay) IsVisible() bool {
	return h.show
}

func (h *HelpOverlay) Hide() {
	h.show = false
}
