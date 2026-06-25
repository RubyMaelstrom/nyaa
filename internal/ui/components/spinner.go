package components

import (
	"math/rand"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/nyaa/internal/ui/theme"
)

type Spinner struct {
	spinner    spinner.Model
	message    string
	style      lipgloss.Style
	frameCount int
}

// kaomojiHappy / kaomojiWaiting rotate through the spinner and seed greetings,
// toasts and empty states. New friends welcome — keep them cute and legible.
var kaomojiHappy = []string{
	"(◕‿◕✿)", "(◕ᴗ◕✿)", "(≧◡≦) ♡", "(◕‿◕)", "(✿◠‿◠)",
	"(◕‿‿◕)", "(◠‿◠✿)", "(◕ᴗ◕)", "(✿♡‿♡)", "ʕ•ᴥ•ʔ",
	"(=^･ω･^=)", "(⁀ᗢ⁀)", "(*≧ω≦*)", "(づ｡◕‿‿◕｡)づ", "(◍•ᴗ•◍)♡",
}

var kaomojiWaiting = []string{
	"(◕﹏◕✿)", "(◕_◕)", "(・_・;)", "(＠_＠;)", "(｡•́︿•̀｡)",
	"(◕﹏◕)", "(>﹏<)", "(￣﹃￣)", "(´｡• ω •｡`)", "(ᵕ﹏ᵕ｡)",
	"(„• ᴗ •„)", "(っ- ‸ - ς)", "( -_-)旦", "(◍•﹏•◍)", "(◕﹏◕✿)",
}

// kaomojiDecorations are the little trailing sparkles after the spinner.
var kaomojiDecorations = []string{
	"✿", "♡", "✧", "⋆", "˚", "·", "✦", "❀", "✾", "❁",
}

// sparkleFrames drive the custom spinner — a soft twinkle instead of plain dots.
var sparkleFrames = []string{"⋆", "✦", "✧", "✩", "✫", "✬", "✭", "✮"}

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

// RandomGreeting returns a random happy kaomoji for the menu's session greeting.
func RandomGreeting() string {
	return kaomojiHappy[rng.Intn(len(kaomojiHappy))]
}

// RandomHappy / RandomWaiting return a random friend from each mood, for toasts
// and empty/loading states that want a little variety.
func RandomHappy() string   { return kaomojiHappy[rng.Intn(len(kaomojiHappy))] }
func RandomWaiting() string { return kaomojiWaiting[rng.Intn(len(kaomojiWaiting))] }

func NewSpinner(message string, style lipgloss.Style) Spinner {
	s := spinner.New()
	s.Spinner = spinner.Spinner{
		Frames: sparkleFrames,
		FPS:    time.Second / 8,
	}
	s.Style = theme.Theme.SpinnerStyle

	return Spinner{
		spinner: s,
		message: message,
		style:   style,
	}
}

func (s Spinner) Init() tea.Cmd {
	return s.spinner.Tick
}

func (s Spinner) Update(msg tea.Msg) (Spinner, tea.Cmd) {
	var cmd tea.Cmd
	s.spinner, cmd = s.spinner.Update(msg)
	if _, ok := msg.(spinner.TickMsg); ok {
		s.frameCount++
	}
	return s, cmd
}

func (s Spinner) View() string {
	// Recolor the spinner glyph from the live palette so theme switches ripple.
	s.spinner.Style = theme.Theme.SpinnerStyle
	kaomoji := theme.Theme.KaomojiStyle.Render(kaomojiHappy[s.frameCount%len(kaomojiHappy)])
	deco := theme.Theme.KaomojiStyle.Render(kaomojiDecorations[s.frameCount%len(kaomojiDecorations)])
	return s.style.Render(s.spinner.View()+" "+s.message+" ") + kaomoji + " " + deco
}

func (s Spinner) ViewWithKaomoji(kaomojiList []string) string {
	kaomoji := kaomojiList[s.frameCount%len(kaomojiList)]
	return s.style.Render(s.spinner.View() + " " + s.message + " " + kaomoji)
}
