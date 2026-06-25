package views

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/nyaa/internal/ui/theme"
)

type SearchInput struct {
	input       textinput.Model
	width       int
	height      int
	toastHeight int
}

func (s *SearchInput) SetDimensions(width, height int) {
	s.width = width
	s.height = height
}

// SetToastLines reserves rows for an active toast so the view shrinks to fit
// rather than overflowing the terminal.
func (s *SearchInput) SetToastLines(n int) {
	s.toastHeight = n
}

func NewSearchInput() SearchInput {
	input := textinput.New()
	input.Placeholder = "search for something cute..."
	input.Focus()
	input.CharLimit = 128

	return SearchInput{input: input}
}

func (s SearchInput) Init() tea.Cmd {
	return textinput.Blink
}

func (s SearchInput) Update(msg tea.Msg) (SearchInput, tea.Cmd) {
	var cmd tea.Cmd
	s.input, cmd = s.input.Update(msg)
	return s, cmd
}

func (s SearchInput) View() string {
	// Pull input colors from the live palette each frame so a theme switch
	// recolors the prompt and typed text immediately.
	s.input.PromptStyle = theme.Theme.PromptStyle
	s.input.TextStyle = theme.Theme.BaseStyle

	title := theme.Theme.TitleStyle.Render(theme.SimpleHeader)
	body := theme.Theme.PromptStyle.Render("search ♡ ") + s.input.View()
	footer := theme.Theme.DimStyle.Render("enter search  •  ↑↓ history  •  t theme  •  esc menu  •  ? help")

	return Frame(s.width, s.height-s.toastHeight, title, footer, lipgloss.Center, lipgloss.Center,
		func(innerW, innerH int) string { return body })
}

func (s *SearchInput) Value() string {
	return s.input.Value()
}

func (s *SearchInput) SetValue(value string) {
	s.input.SetValue(value)
}
