package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/user/nyaa-tui/internal/ui/components"
	"github.com/user/nyaa-tui/internal/ui/theme"
	"github.com/user/nyaa-tui/internal/yt"
)

type ResultsList struct {
	list        components.List
	statusBar   components.StatusBar
	warning     string
	width       int
	height      int
	toastHeight int
	style       lipgloss.Style
	title       lipgloss.Style
	dim         lipgloss.Style
	box         lipgloss.Style
}

func NewResultsList(videos []yt.Video, width, height int) ResultsList {
	list := components.NewList(
		videos,
		theme.Theme.SelectedStyle,
		theme.Theme.BaseStyle,
		theme.Theme.DimStyle,
	)
	list.SetDimensions(width, height)

	statusBar := components.NewStatusBar(1, 1, theme.Theme.StatusBarStyle)
	statusBar.SetWidth(width)

	return ResultsList{
		list:      list,
		statusBar: statusBar,
		width:     width,
		height:    height,
		style:     theme.Theme.BaseStyle,
		title:     theme.Theme.TitleStyle,
		dim:       theme.Theme.DimStyle,
		box:       theme.Theme.BoxStyle,
	}
}

func (r *ResultsList) SetToastLines(n int) {
	r.toastHeight = n
}

func (r ResultsList) Init() tea.Cmd {
	return nil
}

func (r ResultsList) Update(msg tea.Msg) (ResultsList, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			r.list.CursorUp()
		case "down", "j":
			r.list.CursorDown()
		case "pgup", "ctrl+u":
			r.list.PageUp()
		case "pgdown", "ctrl+d":
			r.list.PageDown()
		case "home", "g":
			r.list.GoTop()
		case "end", "G":
			r.list.GoBottom()
		}
		r.statusBar.SetPage(r.list.CurrentPage(), r.list.TotalPages())
	}
	return r, nil
}

func (r ResultsList) View() string {
	count := r.list.Length()
	title := theme.Theme.TitleStyle.Render(fmt.Sprintf("♡ results ♡ %d videos found ♡", count))
	if r.warning != "" {
		title = lipgloss.JoinVertical(lipgloss.Center, title, theme.Theme.ErrorStyle.Render("⚠ "+r.warning))
	}

	r.statusBar.SetWidth(r.width)
	footer := r.statusBar.View()

	return Frame(r.width, r.height-r.toastHeight, title, footer, lipgloss.Left, lipgloss.Top,
		func(innerW, innerH int) string {
			// The list reserves 2 rows of its own chrome; add them back so it
			// fills the frame's inner height exactly.
			r.list.SetDimensions(innerW, innerH+2)
			lines := strings.Split(r.list.View(), "\n")
			for i := range lines {
				lines[i] = ansi.Truncate(lines[i], innerW, "…")
			}
			return strings.Join(lines, "\n")
		})
}

func (r *ResultsList) SetVideos(videos []yt.Video) {
	r.list.SetItems(videos)
	r.statusBar.SetPage(r.list.CurrentPage(), r.list.TotalPages())
}

func (r *ResultsList) SelectedVideo() *yt.Video {
	return r.list.SelectedItem()
}

func (r *ResultsList) SetDimensions(width, height int) {
	r.width = width
	r.height = height
	r.list.SetDimensions(width, height)
	r.statusBar.SetWidth(width)
}

func (r *ResultsList) SetWarning(warning string) {
	r.warning = warning
}
