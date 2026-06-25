package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/user/nyaa/internal/ui/components"
	"github.com/user/nyaa/internal/ui/theme"
	"github.com/user/nyaa/internal/yt"
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
		r.ensureVisible()
		r.statusBar.SetPage(r.list.CurrentPage(), r.list.TotalPages())
	}
	return r, nil
}

// ensureVisible scrolls the list so the cursor stays on screen after keyboard or
// wheel navigation. The visible item count is derived from the live frame
// geometry (each item is two rows), so it matches what View actually renders
// regardless of the list's last-set height.
func (r *ResultsList) ensureVisible() {
	_, innerH, _ := frameGeometry(r.width, r.height-r.toastHeight, r.titleView(), r.footerView())
	r.list.EnsureCursorVisible(innerH / 2)
}

// titleView builds the results header (with the partial-results warning, if
// any). Shared by View and ItemAt so mouse hit-testing measures the same chrome.
func (r ResultsList) titleView() string {
	title := theme.Theme.TitleStyle.Render(fmt.Sprintf("♡ results ♡ %d videos found ♡", r.list.Length()))
	if r.warning != "" {
		title = lipgloss.JoinVertical(lipgloss.Center, title, theme.Theme.ErrorStyle.Render("⚠ "+r.warning))
	}
	return title
}

func (r ResultsList) footerView() string {
	r.statusBar.SetWidth(r.width)
	return r.statusBar.View()
}

// SetCursor moves the selection to index i (clamped); used by mouse hover.
func (r *ResultsList) SetCursor(i int) {
	r.list.SetCursor(i)
	r.statusBar.SetPage(r.list.CurrentPage(), r.list.TotalPages())
}

// ItemAt maps a mouse Y coordinate to the result index under it, or ok=false if
// the pointer is outside the list rows.
func (r ResultsList) ItemAt(y int) (int, bool) {
	title := r.titleView()
	_, innerH, bodyTop := frameGeometry(r.width, r.height-r.toastHeight, title, r.footerView())
	return r.list.IndexAtRow(y-bodyTop, innerH)
}

func (r ResultsList) View() string {
	title := r.titleView()
	footer := r.footerView()

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
