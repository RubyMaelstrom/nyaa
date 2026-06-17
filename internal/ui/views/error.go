package views

import (
	"errors"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/nyaa-tui/internal/ui/theme"
	"github.com/user/nyaa-tui/internal/yt"
)

type ErrorView struct {
	message     string
	suggestion  string
	showUpdate  bool
	cursor      int
	originalErr error
	style       lipgloss.Style
	title       lipgloss.Style
	errorStyle  lipgloss.Style
	help        lipgloss.Style
	dim         lipgloss.Style
	kaomoji     lipgloss.Style
	box         lipgloss.Style
	kaomojiStr  string
}

func NewErrorView(message string) ErrorView {
	return ErrorView{
		message:    message,
		cursor:     0,
		style:      theme.Theme.BaseStyle,
		title:      theme.Theme.TitleStyle,
		errorStyle: theme.Theme.ErrorStyle,
		help:       theme.Theme.HelpStyle,
		dim:        theme.Theme.DimStyle,
		kaomoji:    theme.Theme.KaomojiStyle,
		box:        theme.Theme.CardStyle,
		kaomojiStr: "(◕﹏◕✿)",
	}
}

func NewErrorViewWithDetails(message, suggestion string, showUpdate bool) ErrorView {
	kaomoji := "(◕﹏◕✿)"
	if suggestion != "" {
		kaomoji = "(qwq)"
	}
	return ErrorView{
		message:    message,
		suggestion: suggestion,
		showUpdate: showUpdate,
		cursor:     0,
		style:      theme.Theme.BaseStyle,
		title:      theme.Theme.TitleStyle,
		errorStyle: theme.Theme.ErrorStyle,
		help:       theme.Theme.HelpStyle,
		dim:        theme.Theme.DimStyle,
		kaomoji:    theme.Theme.KaomojiStyle,
		box:        theme.Theme.CardStyle,
		kaomojiStr: kaomoji,
	}
}

func NewErrorViewWithError(err error) ErrorView {
	var ytErr *yt.YtDlpError
	if errors.As(err, &ytErr) {
		return NewErrorViewWithDetails(ytErr.Message, ytErr.Suggestion, ytErr.ShowUpdate)
	}
	return ErrorView{
		message:     err.Error(),
		cursor:      0,
		originalErr: err,
		style:       theme.Theme.BaseStyle,
		title:       theme.Theme.TitleStyle,
		errorStyle:  theme.Theme.ErrorStyle,
		help:        theme.Theme.HelpStyle,
		dim:         theme.Theme.DimStyle,
		kaomoji:     theme.Theme.KaomojiStyle,
		box:         theme.Theme.CardStyle,
		kaomojiStr:  "(◕﹏◕✿)",
	}
}

func (e ErrorView) Init() tea.Cmd {
	return nil
}

func (e ErrorView) Update(msg tea.Msg) (ErrorView, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if e.cursor > 0 {
				e.cursor--
			}
		case "down", "j":
			maxCursor := 1
			if e.showUpdate {
				maxCursor = 2
			}
			if e.cursor < maxCursor {
				e.cursor++
			}
		}
	}
	return e, nil
}

func (e ErrorView) View() string {
	title := e.title.Render("♡ ─── Oopsie! ─── ♡")
	kaomoji := e.kaomoji.Render("       " + e.kaomojiStr)
	errorMsg := e.errorStyle.Render(e.message)

	var lines []string
	lines = append(lines, title, "")
	lines = append(lines, kaomoji, "")
	lines = append(lines, errorMsg)

	if e.suggestion != "" {
		suggestion := e.dim.Render("💡 " + e.suggestion)
		lines = append(lines, "", suggestion)
	}

	lines = append(lines, "")

	helpLines := []string{
		e.formatAction("[R] Retry", 0),
		e.formatAction("[C] Change search", 1),
		e.formatAction("[Esc] Menu", 99),
	}
	if e.showUpdate {
		helpLines = append(helpLines, e.formatAction("[U] Update yt-dlp", 2))
	}
	helpLines = append(helpLines, e.formatAction("[Q] Quit", -1))

	help := e.help.Render(joinActions(helpLines))
	lines = append(lines, help)

	content := lipgloss.JoinVertical(lipgloss.Center, lines...)
	return e.box.Render(content)
}

func (e ErrorView) formatAction(label string, idx int) string {
	if idx == e.cursor {
		return theme.Theme.SelectedStyle.Render("▸ " + label)
	}
	return label
}

func joinActions(actions []string) string {
	result := ""
	for i, a := range actions {
		if i > 0 {
			result += "    "
		}
		result += a
	}
	return result
}

func (e *ErrorView) SetMessage(msg string) {
	e.message = msg
}

func (e *ErrorView) SetSuggestion(suggestion string) {
	e.suggestion = suggestion
}

func (e *ErrorView) SetShowUpdate(show bool) {
	e.showUpdate = show
}

func (e *ErrorView) Cursor() int {
	return e.cursor
}

func (e *ErrorView) GetError() error {
	return e.originalErr
}

func (e *ErrorView) SetError(err error) {
	e.originalErr = err
	var ytErr *yt.YtDlpError
	if errors.As(err, &ytErr) {
		e.message = ytErr.Message
		e.suggestion = ytErr.Suggestion
		e.showUpdate = ytErr.ShowUpdate
		if ytErr.Suggestion != "" {
			e.kaomojiStr = "(qwq)"
		}
	}
}
