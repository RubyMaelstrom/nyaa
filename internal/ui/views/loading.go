package views

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/nyaa/internal/ui/components"
	"github.com/user/nyaa/internal/ui/theme"
)

type Loading struct {
	spinner components.Spinner
	message string
	style   lipgloss.Style
	title   lipgloss.Style
	box     lipgloss.Style
	kaomoji lipgloss.Style
}

func NewLoading(message string) Loading {
	spinner := components.NewSpinner(message, theme.Theme.BaseStyle)

	return Loading{
		spinner: spinner,
		message: message,
		style:   theme.Theme.BaseStyle,
		title:   theme.Theme.TitleStyle,
		box:     theme.Theme.BoxStyle,
		kaomoji: theme.Theme.KaomojiStyle,
	}
}

func (l Loading) Init() tea.Cmd {
	return l.spinner.Init()
}

func (l Loading) Update(msg tea.Msg) (Loading, tea.Cmd) {
	var cmd tea.Cmd
	l.spinner, cmd = l.spinner.Update(msg)
	return l, cmd
}

func (l Loading) View() string {
	title := l.title.Render("✿ searching... ✿")
	kaomoji := l.kaomoji.Render("(◕‿◕✿)")
	spinner := l.spinner.View()

	content := lipgloss.JoinVertical(lipgloss.Center,
		kaomoji,
		"",
		spinner,
	)

	boxed := l.box.Render(content)

	hints := l.title.Render("esc to menu")

	return lipgloss.JoinVertical(lipgloss.Center,
		"",
		title,
		"",
		boxed,
		"",
		hints,
	)
}
