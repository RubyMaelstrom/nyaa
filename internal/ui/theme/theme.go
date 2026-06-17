package theme

import "github.com/charmbracelet/lipgloss"

// Palette is one named color scheme. Every visible style is derived from a
// Palette (see buildStyles), so swapping the active palette recolors the whole
// app in one assignment. Colors are tuned to read on a *dark* terminal — we no
// longer paint a full-screen background, so text floats clean on the user's own
// canvas instead of leaving patchy pink rectangles.
type Palette struct {
	Name       string
	Primary    string // titles, banners, card borders
	Accent     string // secondary accent (mint / cyan)
	Highlight  string // kaomoji sparkle + toasts
	Text       string // body text — must read on dark
	TextBright string // prompts + selected text
	Dim        string // hints, metadata, dim rows
	Border     string // box borders
	SelectBg   string // selection + status-bar background bar
	Error      string
	Success    string
}

// Sakura — the original pastel cherry-blossom dream, retuned for a dark canvas.
var Sakura = Palette{
	Name:       "sakura",
	Primary:    "#FFB7C5",
	Accent:     "#B5EAD7",
	Highlight:  "#FFDAC1",
	Text:       "#F3E1E8",
	TextBright: "#C9B6FF",
	Dim:        "#B8A9D9",
	Border:     "#C9B1FF",
	SelectBg:   "#4A3B63",
	Error:      "#FF9AA2",
	Success:    "#BAFFC9",
}

// Matcha — warm green tea & honey, cozy and soft.
var Matcha = Palette{
	Name:       "matcha",
	Primary:    "#A8E6A1",
	Accent:     "#FFE3A3",
	Highlight:  "#FFD6E8",
	Text:       "#E4F3D6",
	TextBright: "#D4F7B0",
	Dim:        "#9DB98C",
	Border:     "#88C988",
	SelectBg:   "#2E4A2E",
	Error:      "#FFB3BA",
	Success:    "#CFFFB0",
}

// Midnight — neon cyberpunk: electric pink & cyan glowing on the void.
var Midnight = Palette{
	Name:       "midnight",
	Primary:    "#FF6EC7",
	Accent:     "#6EE7FF",
	Highlight:  "#C792EA",
	Text:       "#D7E0FF",
	TextBright: "#7DF9FF",
	Dim:        "#6B7BA8",
	Border:     "#B36EFF",
	SelectBg:   "#2A2150",
	Error:      "#FF5C8A",
	Success:    "#5CFFA0",
}

// Cottoncandy — dreamy pink & sky-blue spun sugar.
var Cottoncandy = Palette{
	Name:       "cottoncandy",
	Primary:    "#FFB3DE",
	Accent:     "#B3E5FF",
	Highlight:  "#FFF0B3",
	Text:       "#FCE4F1",
	TextBright: "#B3D9FF",
	Dim:        "#C9A9D9",
	Border:     "#FFC2E2",
	SelectBg:   "#5A4566",
	Error:      "#FFA3B1",
	Success:    "#B3FFD9",
}

// palettes is the cycle order for the in-app theme switcher (the `t` key).
var palettes = []Palette{Sakura, Matcha, Midnight, Cottoncandy}

var activeIdx = 0

var KawaiiBorder = lipgloss.Border{
	Top:         "━",
	Bottom:      "━",
	Left:        "┃",
	Right:       "┃",
	TopLeft:     "╭",
	TopRight:    "╮",
	BottomLeft:  "╰",
	BottomRight: "╯",
}

var HeartBorder = lipgloss.Border{
	Top:         "♡",
	Bottom:      "♡",
	Left:        "♡",
	Right:       "♡",
	TopLeft:     "♡",
	TopRight:    "♡",
	BottomLeft:  "♡",
	BottomRight: "♡",
}

// SparkleBorder twinkles softly around special cards (now-playing).
var SparkleBorder = lipgloss.Border{
	Top:         "✦",
	Bottom:      "✦",
	Left:        "✧",
	Right:       "✧",
	TopLeft:     "✶",
	TopRight:    "✶",
	BottomLeft:  "✶",
	BottomRight: "✶",
}

// BannerLines is the "nyaa" wordmark (toilet's `pagga` font). It carries no
// color of its own; RenderBanner paints it with a per-line palette gradient.
var BannerLines = []string{
	"░█▀█░█░█░█▀█░█▀█",
	"░█░█░░█░░█▀█░█▀█",
	"░▀░▀░░▀░░▀░▀░▀░▀",
}

// BannerTagline sits under the wordmark on the menu.
var BannerTagline = "♡ a kawaii youtube tui ♡"

var SimpleHeader = "(◕‿◕✿) nyaa-tui"

// RenderBanner colors the wordmark with a gentle three-stop gradient drawn from
// the active palette, then drops the tagline beneath it in dim.
func RenderBanner() string {
	p := Theme.Palette
	grad := []string{p.Primary, p.Highlight, p.Accent}
	lines := make([]string, 0, len(BannerLines)+1)
	for i, l := range BannerLines {
		c := grad[i%len(grad)]
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color(c)).Bold(true).Render(l))
	}
	lines = append(lines, Theme.DimStyle.Render(BannerTagline))
	return lipgloss.JoinVertical(lipgloss.Center, lines...)
}

// Styles is the full set of derived lipgloss styles for one palette. The active
// set lives in the package var Theme; reassign it (NextPalette) to recolor.
type Styles struct {
	Palette        Palette
	BaseStyle      lipgloss.Style
	TitleStyle     lipgloss.Style
	PromptStyle    lipgloss.Style
	ErrorStyle     lipgloss.Style
	SuccessStyle   lipgloss.Style
	SelectedStyle  lipgloss.Style
	DimStyle       lipgloss.Style
	BorderStyle    lipgloss.Style
	BoxStyle       lipgloss.Style
	CardStyle      lipgloss.Style
	StatusBarStyle lipgloss.Style
	HelpStyle      lipgloss.Style
	HeaderStyle    lipgloss.Style
	SpinnerStyle   lipgloss.Style
	KaomojiStyle   lipgloss.Style
}

func buildStyles(p Palette) Styles {
	c := func(s string) lipgloss.Color { return lipgloss.Color(s) }
	return Styles{
		Palette: p,

		// No Background: text floats on the terminal's own canvas (the old
		// full-screen pink fill painted patchy rectangles on dark terminals).
		BaseStyle: lipgloss.NewStyle().
			Foreground(c(p.Text)),

		TitleStyle: lipgloss.NewStyle().
			Foreground(c(p.Primary)).
			Bold(true),

		PromptStyle: lipgloss.NewStyle().
			Foreground(c(p.TextBright)).
			Bold(true),

		ErrorStyle: lipgloss.NewStyle().
			Foreground(c(p.Error)).
			Bold(true),

		SuccessStyle: lipgloss.NewStyle().
			Foreground(c(p.Success)).
			Bold(true),

		SelectedStyle: lipgloss.NewStyle().
			Foreground(c(p.TextBright)).
			Background(c(p.SelectBg)).
			Bold(true).
			Padding(0, 1),

		DimStyle: lipgloss.NewStyle().
			Foreground(c(p.Dim)),

		BorderStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(c(p.Border)).
			Padding(1, 2),

		BoxStyle: lipgloss.NewStyle().
			Border(KawaiiBorder).
			BorderForeground(c(p.Border)).
			Padding(1, 2),

		CardStyle: lipgloss.NewStyle().
			Border(HeartBorder).
			BorderForeground(c(p.Primary)).
			Padding(1, 2).
			Margin(1, 0),

		StatusBarStyle: lipgloss.NewStyle().
			Foreground(c(p.Text)).
			Background(c(p.SelectBg)).
			Padding(0, 1),

		HelpStyle: lipgloss.NewStyle().
			Foreground(c(p.Dim)).
			Padding(0, 1),

		HeaderStyle: lipgloss.NewStyle().
			Foreground(c(p.Primary)).
			Bold(true).
			Padding(0, 1),

		SpinnerStyle: lipgloss.NewStyle().
			Foreground(c(p.Primary)),

		KaomojiStyle: lipgloss.NewStyle().
			Foreground(c(p.Highlight)).
			Bold(true),
	}
}

// Theme is the live style set every view reads at render time. NextPalette
// reassigns it so a theme switch ripples everywhere on the next frame.
var Theme = buildStyles(Sakura)

// NextPalette advances to the next palette, rebuilds Theme, and returns the new
// palette's name (for a toast). In-memory only; persistence lands with config.
func NextPalette() string {
	activeIdx = (activeIdx + 1) % len(palettes)
	Theme = buildStyles(palettes[activeIdx])
	return palettes[activeIdx].Name
}
