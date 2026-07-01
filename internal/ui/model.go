package ui

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/nyaa/internal/player"
	"github.com/user/nyaa/internal/rss"
	"github.com/user/nyaa/internal/subscriptions"
	"github.com/user/nyaa/internal/ui/theme"
	"github.com/user/nyaa/internal/ui/views"
	"github.com/user/nyaa/internal/yt"
)

// tickCmd schedules the next 100ms animation/toast frame.
func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// ensureTick starts the shared ticker if it isn't already running, so toasts
// and the now-playing animation never stack overlapping timers.
func (m *Model) ensureTick() tea.Cmd {
	if m.ticking {
		return nil
	}
	m.ticking = true
	return tickCmd()
}

type viewState int

const (
	StateMenu viewState = iota
	StateSearch
	StateLoading
	StateResults
	StateSubscription
	StateChannel
	StatePlaying
	StateError
	StateHelp
)

type tickMsg time.Time

type Model struct {
	state     viewState
	menu      views.Menu
	search    views.SearchInput
	results   views.ResultsList
	loading   views.Loading
	errorView views.ErrorView
	help      views.HelpOverlay

	videos            []yt.Video
	lastQuery         string
	searchHistory     []string
	historyIdx        int
	width             int
	height            int
	nowPlaying        *yt.Video
	retryVideo        *yt.Video // video whose playback just failed, replayed by [R] Retry
	playAttempts      int       // transient stream-open retries used for the current video
	cachedVideos      []yt.Video
	cachedQuery       string
	partialWarning    string
	ytDlpVersion      string
	toast             string
	toastTimer        float64 // seconds
	subs              *subscriptions.SubscriptionsFile
	subscriptionsView views.SubscriptionsView
	channelBrowse     views.ChannelBrowse
	previousState     viewState
	helpReturn        viewState
	feedCache         map[string]cachedFeed
	feedsPending      int
	feedRetry         map[string]string // channelID -> rss url, still failing this refresh
	feedRound         int
	feedDeadline      time.Time
	feedRefreshGen    int // bumped each refresh so stale in-flight results are ignored
	audioOnly         bool
	frame             int  // animation tick counter (now-playing, toasts)
	ticking           bool // guards against stacking multiple 100ms tickers
}

// cachedFeed holds a channel's last-fetched RSS items so re-entering the
// subscriptions view is instant until the entry goes stale.
type cachedFeed struct {
	items     []rss.RSSItem
	err       error
	fetchedAt time.Time
}

const feedTTL = 5 * time.Minute

// Feed refresh retry policy. YouTube's RSS backend load-sheds (404/5xx) per
// shared IP, worst during the morning feed-sync peak. A failed feed isn't dead —
// retrying it on a fresh connection usually lands it on a healthy backend. We
// blast all feeds at once (round 1 = unchanged behavior when not throttled),
// then retry stragglers: rounds up to feedFastRounds fire back-to-back, later
// rounds are spaced out, and the whole thing gives up at feedRefreshDeadline so
// the user can get on to watching rather than babysitting the list.
const (
	feedRefreshDeadline = 20 * time.Second
	feedFastRounds      = 3
	feedRetrySpacing    = 3 * time.Second
)

// Playback retry policy. mpv exiting 2 is nearly always a transient throttle at
// stream-open (see player.IsRetryable). Rather than dropping straight to the
// error screen, we silently relaunch on a fresh backend a few times — the
// now-playing animation stays up, so it just reads as "loading" — and only
// surface the error once the whole throttle window looks bad.
const (
	maxPlayRetries   = 3
	playRetrySpacing = 1500 * time.Millisecond
)

// errFeedThrottled is shown for a feed still failing at the deadline that has no
// cached items to fall back on — deliberately not a raw "status 404".
var errFeedThrottled = errors.New("couldn't refresh — YouTube is throttling feeds, try again soon (｡•́︿•̀｡)")

func (m *Model) updateSubscriptionsView() {
	var channels []views.ChannelGroup
	for _, entry := range m.subs.Entries {
		channels = append(channels, views.NewChannelGroup(entry))
	}
	m.subscriptionsView.SetChannels(channels)
	m.subscriptionsView.SetDimensions(m.width, m.height)
	// Reseed any feeds we already have so rebuilding the view (after
	// subscribe/unsubscribe/mark-read) doesn't blank out fetched items.
	for id, c := range m.feedCache {
		m.subscriptionsView.Fetched(id, c.items, c.err)
	}
}

func InitialModel() Model {
	search := views.NewSearchInput()

	sub, _ := subscriptions.Load()
	if sub == nil {
		sub = &subscriptions.SubscriptionsFile{Entries: make(map[string]subscriptions.SubscriptionEntry)}
	}

	return Model{
		state:             StateMenu,
		menu:              views.NewMenu(sub.Count()),
		search:            search,
		help:              views.NewHelpOverlay(),
		subs:              sub,
		subscriptionsView: views.NewSubscriptionsView(),
		feedCache:         make(map[string]cachedFeed),
		feedRetry:         make(map[string]string),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m *Model) versionCheckCmd() tea.Cmd {
	return func() tea.Msg {
		version, err := yt.CheckVersion()
		if err != nil {
			return versionCheckMsg{version: "unknown", err: err}
		}
		return versionCheckMsg{version: version}
	}
}

type versionCheckMsg struct {
	version string
	err     error
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.state == StateResults {
			m.results.SetDimensions(msg.Width, msg.Height)
		}
		if m.state == StateSubscription {
			m.subscriptionsView.SetDimensions(msg.Width, msg.Height)
		}

	case tea.KeyMsg:
		// `?` toggles the help overlay from anywhere except the search input
		// (where it is a typeable character), returning to wherever it opened.
		if msg.String() == "?" && m.state != StateSearch {
			if m.help.IsVisible() {
				m.help.Hide()
				m.state = m.helpReturn
			} else {
				m.helpReturn = m.state
				m.help.Toggle()
				m.state = StateHelp
			}
			return m, nil
		}

		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if msg.String() == "q" && m.state != StateSearch {
			return m, tea.Quit
		}
		// `t` cycles the color theme from anywhere except the search input
		// (where it's a typeable character). Recolors the whole app instantly.
		if msg.String() == "t" && m.state != StateSearch {
			name := theme.NextPalette()
			return m, m.showToast(fmt.Sprintf("theme: %s (◕ᴗ◕✿) ♡", name))
		}
		// `c` (or `C`) copies the highlighted video's link from anywhere except
		// the search input. copyCurrentLink returns nil when no video is under
		// the cursor, so we fall through to the screen's own `c` handling (the
		// error view uses `c` for "show cached results").
		if (msg.String() == "c" || msg.String() == "C") && m.state != StateSearch {
			if cmd := m.copyCurrentLink(); cmd != nil {
				return m, cmd
			}
		}

		switch m.state {
		case StateMenu:
			return m.updateMenu(msg)
		case StateSearch:
			return m.updateSearch(msg)
		case StateLoading:
			if msg.String() == "esc" {
				m.state = StateMenu
				return m, nil
			}
			return m, nil
		case StateResults:
			return m.updateResults(msg)
		case StateSubscription:
			return m.updateSubscriptions(msg)
		case StateChannel:
			return m.updateChannel(msg)
		case StateError:
			return m.updateError(msg)
		case StatePlaying:
			return m, nil
		}

	case tea.MouseMsg:
		return m.handleMouse(msg)

	case versionCheckMsg:
		m.ytDlpVersion = msg.version
		if msg.err != nil {
			m.ytDlpVersion = "unknown"
		}
		return m, nil

	case searchResultMsg:
		m.handleSearchResult(msg)
		return m, nil

	case tickMsg:
		m.frame++
		if m.toastTimer > 0 {
			m.toastTimer -= 0.1
			if m.toastTimer <= 0 {
				m.toast = ""
				m.toastTimer = 0
			}
		}
		// Keep ticking while a toast is fading or a video is playing (so the
		// now-playing screen animates); otherwise let the ticker rest.
		if m.toastTimer > 0 || m.state == StatePlaying {
			return m, tickCmd()
		}
		m.ticking = false
		return m, nil

	case subscribeResultMsg:
		m.menu.SetSubscriptionsCount(m.subs.Count())
		m.updateSubscriptionsView()
		return m, m.showToast(msg.toast)

	case errorMsg:
		m.handleError(msg)
		return m, nil

	case ytDlpUpdatedMsg:
		m.errorView = views.NewErrorView("yt-dlp updated! (≧◡≦) ♡\nPlease retry your search~")
		return m, nil

	case playFinishedMsg:
		return m, m.handlePlayFinished(msg)

	case replayMsg:
		// Auto-retry after a transient stream-open failure. Guard on StatePlaying
		// so a stray tick can't yank the user out of wherever they navigated to.
		if m.state == StatePlaying && m.nowPlaying != nil {
			return m, tea.Batch(m.playCmd(m.nowPlaying.URL), m.ensureTick())
		}
		return m, nil

	case feedFetchedMsg:
		return m, m.handleFeedFetched(msg)

	case feedRetryTickMsg:
		if msg.gen != m.feedRefreshGen || m.state != StateSubscription {
			return m, nil // superseded refresh, or the user navigated away
		}
		if time.Now().After(m.feedDeadline) {
			m.finalizeFeedRefresh()
			return m, nil
		}
		return m, m.fireFeedRound()

	case channelFeedMsg:
		m.handleChannelFeed(msg)
		return m, nil

	case views.PlayVideoMsg:
		video := msg.Video
		channel := ""
		if m.state == StateChannel {
			channel = m.channelBrowse.Name()
		}
		m.nowPlaying = &yt.Video{
			ID:      video.ID,
			Title:   video.Title,
			Channel: channel,
			URL:     video.URL,
		}
		m.previousState = m.state
		m.state = StatePlaying
		m.playAttempts = 0
		return m, tea.Batch(m.playCmd(video.URL), m.ensureTick())
	}

	var cmd tea.Cmd
	switch m.state {
	case StateMenu:
		m.menu, cmd = m.menu.Update(msg)
	case StateSearch:
		m.search, cmd = m.search.Update(msg)
	case StateLoading:
		m.loading, cmd = m.loading.Update(msg)
	case StateResults:
		m.results, cmd = m.results.Update(msg)
	case StateSubscription:
		m.subscriptionsView, cmd = m.subscriptionsView.Update(msg)
	case StateChannel:
		m.channelBrowse, cmd = m.channelBrowse.Update(msg)
	case StateError:
		m.errorView, cmd = m.errorView.Update(msg)
	}

	return m, cmd
}

func (m Model) View() string {
	// Keep every view sized to the current terminal so they all resize uniformly.
	m.menu.SetDimensions(m.width, m.height)
	m.search.SetDimensions(m.width, m.height)
	m.results.SetDimensions(m.width, m.height)
	m.subscriptionsView.SetDimensions(m.width, m.height)
	m.channelBrowse.SetDimensions(m.width, m.height)

	// A toast reserves exactly its own height at the bottom; the active list
	// view shrinks its frame by that much so nothing overflows the terminal.
	toast, toastLines := m.toastBlock()
	m.menu.SetToastLines(toastLines)
	m.search.SetToastLines(toastLines)
	m.results.SetToastLines(toastLines)
	m.subscriptionsView.SetToastLines(toastLines)
	m.channelBrowse.SetToastLines(toastLines)

	var base string
	switch m.state {
	case StateMenu:
		base = m.menu.View()
	case StateSearch:
		base = m.search.View()
	case StateLoading:
		base = m.loading.View()
	case StateResults:
		base = m.results.View()
	case StateSubscription:
		base = m.subscriptionsView.View()
	case StateChannel:
		base = m.channelBrowse.View()
	case StatePlaying:
		base = m.playingView()
	case StateError:
		base = m.errorView.View()
	case StateHelp:
		base = m.help.View()
	default:
		base = m.menu.View()
	}

	if toast != "" {
		return lipgloss.JoinVertical(lipgloss.Center,
			lipgloss.NewStyle().Width(m.width).Render(base),
			toast,
		)
	}
	return base
}

func (m *Model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state = StateMenu
		return m, nil

	case "enter":
		query := m.search.Value()
		if query == "" {
			return m, nil
		}

		m.lastQuery = query
		m.searchHistory = append(m.searchHistory, query)
		m.historyIdx = len(m.searchHistory) - 1

		return m.enterLoading("searching for "+query+"...", m.searchCmd(query))

	case "up":
		if len(m.searchHistory) > 0 {
			if m.historyIdx > 0 {
				m.historyIdx--
			}
			m.search.SetValue(m.searchHistory[m.historyIdx])
		}
		return m, nil

	case "down":
		if len(m.searchHistory) > 0 {
			if m.historyIdx < len(m.searchHistory)-1 {
				m.historyIdx++
				m.search.SetValue(m.searchHistory[m.historyIdx])
			} else {
				m.historyIdx = len(m.searchHistory)
				m.search.SetValue("")
			}
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.search, cmd = m.search.Update(msg)
	return m, cmd
}

func (m *Model) updateResults(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "s":
		video := m.results.SelectedVideo()
		if video != nil {
			return m, m.subscribeCmd(video)
		}

	case "esc":
		m.state = StateMenu
		return m, nil

	case "enter":
		video := m.results.SelectedVideo()
		if video != nil {
			m.nowPlaying = video
			m.previousState = StateResults
			m.state = StatePlaying
			m.playAttempts = 0
			return m, tea.Batch(m.playCmd(video.URL), m.ensureTick())
		}

	case "o":
		video := m.results.SelectedVideo()
		if video != nil && video.ChannelID != "" {
			m.previousState = StateResults
			return m.enterLoading("loading "+video.Channel+"...", m.fetchChannelCmd(video.ChannelID, video.Channel))
		}
		return m, nil

	case "a":
		return m, m.toggleAudio()

	case "/":
		m.state = StateMenu
		return m, nil

	case "r":
		if m.lastQuery != "" {
			return m.enterLoading("retrying search for "+m.lastQuery+"...", m.searchCmd(m.lastQuery))
		}
	}

	var cmd tea.Cmd
	m.results, cmd = m.results.Update(msg)
	return m, cmd
}

// retryFromError handles [R] Retry on the error screen. A failed playback
// replays the same video (a fresh connection often dodges a transient YouTube
// throttle); otherwise it re-runs the last search. If there's nothing to retry
// it's a no-op.
func (m *Model) retryFromError() (tea.Model, tea.Cmd) {
	if m.retryVideo != nil {
		m.nowPlaying = m.retryVideo
		m.state = StatePlaying
		return m, tea.Batch(m.playCmd(m.retryVideo.URL), m.ensureTick())
	}
	if m.lastQuery != "" {
		return m.enterLoading("retrying search for "+m.lastQuery+"...", m.searchCmd(m.lastQuery))
	}
	return m, nil
}

func (m *Model) updateError(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "r":
		return m.retryFromError()

	case "c":
		if len(m.cachedVideos) > 0 {
			m.videos = m.cachedVideos
			m.results = views.NewResultsList(m.cachedVideos, m.width, m.height)
			m.state = StateResults
			return m, nil
		}
		m.state = StateSearch
		return m, nil

	case "esc":
		m.state = StateMenu
		return m, nil

	case "u":
		var ytErr *yt.YtDlpError
		if errors.As(m.errorView.GetError(), &ytErr) && ytErr.ShowUpdate {
			return m.enterLoading("updating yt-dlp... (◕‿◕✿)", m.updateYtDlpCmd())
		}

	case "enter":
		switch m.errorView.Cursor() {
		case 0:
			return m.retryFromError()
		case 1:
			if len(m.cachedVideos) > 0 {
				m.videos = m.cachedVideos
				m.results = views.NewResultsList(m.cachedVideos, m.width, m.height)
				m.state = StateResults
				return m, nil
			}
			m.state = StateSearch
			return m, nil
		case 2:
			var ytErr *yt.YtDlpError
			if errors.As(m.errorView.GetError(), &ytErr) && ytErr.ShowUpdate {
				return m.enterLoading("updating yt-dlp... (◕‿◕✿)", m.updateYtDlpCmd())
			}
		}
	}

	var cmd tea.Cmd
	m.errorView, cmd = m.errorView.Update(msg)
	return m, cmd
}

func (m *Model) updateYtDlpCmd() tea.Cmd {
	return func() tea.Msg {
		err := yt.UpdateYtDlp()
		if err != nil {
			return errorMsg{err: fmt.Errorf("failed to update yt-dlp: %w", err)}
		}
		return ytDlpUpdatedMsg{}
	}
}

type ytDlpUpdatedMsg struct{}

func (m *Model) searchCmd(query string) tea.Cmd {
	return func() tea.Msg {
		result, err := yt.Search(query, 20)
		if err != nil {
			if len(result.Videos) > 0 {
				return searchResultMsg{
					videos:       result.Videos,
					partial:      true,
					partialCount: result.PartialCount,
					warning:      "Showing partial results (" + fmt.Sprintf("%d videos)", result.PartialCount),
				}
			}
			return errorMsg{err: err}
		}
		return searchResultMsg{videos: result.Videos}
	}
}

// playCmd picks a playback strategy based on the environment:
//
//   - GUI: run mpv in the background in its own window (the WM handles its
//     keyboard) and keep the cute "now playing" screen up — the TUI never drops.
//   - Bare TTY: there's no window to capture keys, so hand the terminal to mpv
//     via tea.Exec (suspending the TUI) and print a banner so the user knows
//     what's playing while we're away.
func (m *Model) playCmd(url string) tea.Cmd {
	audioOnly := m.audioOnly
	if guiAvailable() {
		return func() tea.Msg {
			err := player.Command(url, audioOnly, true).Run()
			return playFinishedMsg{err: player.ClassifyError(err)}
		}
	}

	cmd := player.Command(url, audioOnly, false)
	return tea.Exec(newMpvExec(cmd, m.nowPlayingBanner()), func(err error) tea.Msg {
		return playFinishedMsg{err: player.ClassifyError(err)}
	})
}

// nowPlayingBanner renders the cute "now playing" card as plain text for the
// terminal we're about to hand to mpv (TTY path only).
func (m Model) nowPlayingBanner() string {
	if m.nowPlaying == nil {
		return ""
	}
	mode := "♪ audio-only"
	if !m.audioOnly {
		mode = "✿ video"
	}
	block := lipgloss.JoinVertical(lipgloss.Center,
		"",
		theme.Theme.TitleStyle.Render("✿ now playing (≧◡≦) ♡ ✿"),
		theme.Theme.BaseStyle.Bold(true).Render(m.nowPlaying.Title),
		theme.Theme.DimStyle.Render("by "+m.nowPlaying.Channel),
		theme.Theme.KaomojiStyle.Render(mode),
		theme.Theme.DimStyle.Render("controls are in mpv now — press q there to come back ♡"),
		"",
	)
	if m.width > 0 {
		block = lipgloss.PlaceHorizontal(m.width, lipgloss.Center, block)
	}
	return block + "\n"
}

func (m *Model) subscribeCmd(video *yt.Video) tea.Cmd {
	return func() tea.Msg {
		if video.ChannelID == "" {
			return subscribeResultMsg{
				toast: fmt.Sprintf("can't subscribe to %v: no channel ID", video.Channel),
			}
		}
		if m.subs.IsSubscribed(video.ChannelID) {
			return subscribeResultMsg{
				toast: fmt.Sprintf("already subscribed to %s (~‿◿✧", video.Channel),
			}
		}
		added, err := m.subs.Add(video.ChannelID, video.Channel)
		if err != nil {
			return subscribeResultMsg{
				toast: fmt.Sprintf("failed to subscribe: %v", err),
			}
		}
		if !added {
			return subscribeResultMsg{
				toast: fmt.Sprintf("already subscribed to %v (~‿◿✧", video.Channel),
			}
		}
		if err := m.subs.Save(); err != nil {
			return subscribeResultMsg{
				toast: fmt.Sprintf("failed to save: %v", err),
			}
		}
		return subscribeResultMsg{
			toast: fmt.Sprintf("subscribed to %s (≧◡≦) ♡", video.Channel),
		}
	}
}

type searchResultMsg struct {
	videos       []yt.Video
	partial      bool
	partialCount int
	warning      string
}

type subscribeResultMsg struct {
	toast string
}

type errorMsg struct {
	err error
}

type playFinishedMsg struct {
	err error
}

func (m *Model) handleSearchResult(msg searchResultMsg) {
	m.videos = msg.videos
	m.cachedVideos = msg.videos
	m.cachedQuery = m.lastQuery
	m.partialWarning = msg.warning
	m.results = views.NewResultsList(msg.videos, m.width, m.height)
	if msg.partial {
		m.results.SetWarning(msg.warning)
	}
	m.state = StateResults
}

func (m *Model) handleError(msg errorMsg) {
	m.retryVideo = nil // search/extraction error: [R] Retry re-runs the search
	m.errorView = views.NewErrorViewWithError(msg.err)
	m.errorView.SetError(msg.err)
	m.state = StateError
}

func (m *Model) handlePlayFinished(msg playFinishedMsg) tea.Cmd {
	if msg.err != nil {
		// Transient throttle at stream-open: relaunch on a fresh backend after a
		// short pause instead of failing. Stay in StatePlaying (and keep
		// nowPlaying) so the animation carries on and the next attempt has a URL.
		if player.IsRetryable(msg.err) && m.playAttempts < maxPlayRetries && m.nowPlaying != nil {
			m.playAttempts++
			return tea.Tick(playRetrySpacing, func(time.Time) tea.Msg {
				return replayMsg{}
			})
		}
		m.retryVideo = m.nowPlaying // remember it so [R] Retry can replay the video
		m.errorView = views.NewErrorView(msg.err.Error())
		m.state = StateError
	} else {
		m.retryVideo = nil
		if m.previousState != 0 {
			m.state = m.previousState
		} else {
			m.state = StateResults
		}
	}
	m.playAttempts = 0
	m.nowPlaying = nil
	return nil
}

// replayMsg re-launches mpv for the current nowPlaying after a transient
// stream-open failure. It deliberately routes around the normal play entry
// points so it doesn't reset playAttempts (which would loop forever).
type replayMsg struct{}

// danceFrames cycle a little dancing kaomoji while a video plays.
var danceFrames = []string{
	"(っ◔◡◔)っ ♬",
	"♪ ⊂(◕‿◕⊂)",
	"ヾ(◍'౪`◍)ﾉ ♫",
	"♩ ٩(◕‿◕)۶ ♪",
}

// equalizer renders a bouncing block-bar VU meter driven by the frame counter.
func equalizer(frame int) string {
	levels := []rune("▁▂▃▄▅▆▇█")
	const bars = 11
	span := (len(levels) - 1) * 2 // up then back down
	var b strings.Builder
	for i := 0; i < bars; i++ {
		phase := (frame + i*2) % span
		if phase >= len(levels) {
			phase = span - phase
		}
		b.WriteRune(levels[phase])
		b.WriteRune(' ')
	}
	return strings.TrimRight(b.String(), " ")
}

func (m Model) playingView() string {
	if m.nowPlaying == nil {
		return ""
	}

	title := theme.Theme.TitleStyle.Render("♡ now playing ♡")
	dance := theme.Theme.KaomojiStyle.Render(danceFrames[(m.frame/3)%len(danceFrames)])
	eq := theme.Theme.SpinnerStyle.Render(equalizer(m.frame))
	videoTitle := theme.Theme.BaseStyle.Bold(true).Render(m.nowPlaying.Title)
	channel := theme.Theme.DimStyle.Render("by " + m.nowPlaying.Channel)

	mode := theme.Theme.KaomojiStyle.Render("♪ audio-only")
	if !m.audioOnly {
		mode = theme.Theme.KaomojiStyle.Render("✿ video")
	}

	content := lipgloss.JoinVertical(lipgloss.Center,
		dance,
		eq,
		"",
		videoTitle,
		channel,
		mode,
		"",
		theme.Theme.DimStyle.Render("mpv is playing... press q in mpv to return ♡"),
	)

	card := theme.Theme.BoxStyle.Border(theme.SparkleBorder).Render(content)

	return lipgloss.JoinVertical(lipgloss.Center,
		"",
		title,
		"",
		card,
	)
}

func (m *Model) updateSubscriptions(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state = StateMenu
		return m, nil
	case "r":
		return m, m.fetchAllFeedsCmd(true)
	case "x":
		return m, m.unsubscribeSelected()
	case "d":
		return m, m.markAllRead()
	case "a":
		return m, m.toggleAudio()
	}
	var cmd tea.Cmd
	m.subscriptionsView, cmd = m.subscriptionsView.Update(msg)
	return m, cmd
}

func (m *Model) updateChannel(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state = StateResults
		return m, nil
	case "a":
		return m, m.toggleAudio()
	case "s":
		channelID := m.channelBrowse.ChannelID()
		if channelID != "" {
			return m, m.subscribeCmd(&yt.Video{ChannelID: channelID, Channel: m.channelBrowse.Name()})
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.channelBrowse, cmd = m.channelBrowse.Update(msg)
	return m, cmd
}

// toggleAudio flips audio-only playback and reports the new state via a toast.
func (m *Model) toggleAudio() tea.Cmd {
	m.audioOnly = !m.audioOnly
	if m.audioOnly {
		return m.showToast("audio-only mode: on ♪")
	}
	return m.showToast("audio-only mode: off ✿")
}

type channelFeedMsg struct {
	channelID string
	name      string
	items     []rss.RSSItem
	err       error
}

// fetchChannelCmd loads a channel's recent uploads via its RSS feed — cheap and
// API-key-free, the same source the subscriptions view uses.
func (m *Model) fetchChannelCmd(channelID, name string) tea.Cmd {
	return func() tea.Msg {
		items, err := rss.FetchFeed(subscriptions.RSSURLForChannel(channelID))
		return channelFeedMsg{channelID: channelID, name: name, items: items, err: err}
	}
}

func (m *Model) handleChannelFeed(msg channelFeedMsg) {
	if msg.err != nil {
		m.handleError(errorMsg{err: fmt.Errorf("couldn't load %s: %w", msg.name, msg.err)})
		return
	}
	m.channelBrowse = views.NewChannelBrowse(msg.channelID, msg.name, msg.items)
	m.channelBrowse.SetDimensions(m.width, m.height)
	m.state = StateChannel
}

func (m *Model) unsubscribeSelected() tea.Cmd {
	channelID := m.subscriptionsView.SelectedChannelID()
	if channelID == "" {
		return nil
	}
	name := channelID
	if e, ok := m.subs.Entries[channelID]; ok {
		name = e.ChannelName
	}
	m.subs.Remove(channelID)
	m.subs.Save()
	delete(m.feedCache, channelID)
	m.menu.SetSubscriptionsCount(m.subs.Count())
	m.updateSubscriptionsView()
	return m.showToast(fmt.Sprintf("unsubscribed from %s (｡•́︿•̀｡)", name))
}

func (m *Model) markAllRead() tea.Cmd {
	latest := m.subscriptionsView.LatestItemIDs()
	if len(latest) == 0 {
		return nil
	}
	for channelID, videoID := range latest {
		m.subs.MarkSeen(channelID, videoID)
	}
	m.subs.Save()
	m.updateSubscriptionsView()
	return m.showToast("marked all as read (◕‿◕✿) ♡")
}

// enterLoading switches to the loading view and kicks the spinner's ticker so
// its sparkle actually animates while the given work runs in the background.
func (m *Model) enterLoading(message string, work tea.Cmd) (tea.Model, tea.Cmd) {
	m.state = StateLoading
	m.loading = views.NewLoading(message)
	return m, tea.Batch(work, m.loading.Init())
}

// showToast displays a transient message and starts the fade timer.
func (m *Model) showToast(text string) tea.Cmd {
	m.toast = text
	m.toastTimer = 2.0
	return m.ensureTick()
}

func (m Model) subscriptionView() string {
	return m.subscriptionsView.View()
}

type feedFetchedMsg struct {
	channelID string
	url       string
	items     []rss.RSSItem
	err       error
	gen       int
}

// feedRetryTickMsg fires a spaced retry round (rounds beyond feedFastRounds).
type feedRetryTickMsg struct {
	gen int
}

// fetchAllFeedsCmd kicks off a feed refresh: round 1 blasts every subscription's
// RSS feed at once (no concurrency cap — identical to the old behavior when
// nothing is throttled). Fresh cache hits (younger than feedTTL) are seeded
// directly and skipped unless force is set. Failures are retried on fresh
// connections by handleFeedFetched/afterFeedRound until feedRefreshDeadline.
func (m *Model) fetchAllFeedsCmd(force bool) tea.Cmd {
	m.feedRefreshGen++
	gen := m.feedRefreshGen
	m.feedRound = 1
	m.feedDeadline = time.Now().Add(feedRefreshDeadline)
	m.feedRetry = make(map[string]string)

	cmds := make([]tea.Cmd, 0, len(m.subs.Entries))
	for channelID, entry := range m.subs.Entries {
		if !force {
			if c, ok := m.feedCache[channelID]; ok && c.err == nil && time.Since(c.fetchedAt) < feedTTL {
				m.subscriptionsView.Fetched(channelID, c.items, nil)
				continue
			}
		}
		cmds = append(cmds, m.fetchFeedCmd(channelID, entry.RSSURL, gen))
	}
	m.feedsPending = len(cmds)
	m.subscriptionsView.SetLoading(m.feedsPending)
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

func (m *Model) fetchFeedCmd(channelID, url string, gen int) tea.Cmd {
	return func() tea.Msg {
		items, err := rss.FetchFeed(url)
		return feedFetchedMsg{
			channelID: channelID,
			url:       url,
			items:     items,
			err:       err,
			gen:       gen,
		}
	}
}

func (m *Model) handleFeedFetched(msg feedFetchedMsg) tea.Cmd {
	if msg.gen != m.feedRefreshGen {
		return nil // result from a superseded refresh
	}

	switch {
	case msg.err == nil:
		m.feedCache[msg.channelID] = cachedFeed{items: msg.items, fetchedAt: time.Now()}
		m.subscriptionsView.Fetched(msg.channelID, msg.items, nil)
		delete(m.feedRetry, msg.channelID)
	case rss.IsRetryable(msg.err):
		// Transient throttle/load-shed: queue for a fresh-connection retry,
		// leaving any previously fetched items on screen meanwhile.
		m.feedRetry[msg.channelID] = msg.url
	default:
		// Permanent failure (e.g. malformed feed): surface it, don't retry.
		m.subscriptionsView.Fetched(msg.channelID, msg.items, msg.err)
		delete(m.feedRetry, msg.channelID)
	}

	if m.feedsPending > 0 {
		m.feedsPending--
	}
	if m.feedsPending > 0 {
		m.subscriptionsView.SetLoading(m.feedsPending + len(m.feedRetry))
		return nil
	}
	return m.afterFeedRound()
}

// afterFeedRound runs when every request in the current round has reported. It
// either finishes, schedules the next retry round (immediately for fast rounds,
// spaced for later ones), or gives up at the deadline and falls back to cache.
func (m *Model) afterFeedRound() tea.Cmd {
	if len(m.feedRetry) == 0 {
		m.subscriptionsView.SetLoading(0)
		return nil
	}
	if time.Now().After(m.feedDeadline) {
		m.finalizeFeedRefresh()
		return nil
	}
	m.feedRound++
	m.subscriptionsView.SetLoading(len(m.feedRetry))
	if m.feedRound <= feedFastRounds {
		return m.fireFeedRound()
	}
	gen := m.feedRefreshGen
	return tea.Tick(feedRetrySpacing, func(time.Time) tea.Msg {
		return feedRetryTickMsg{gen: gen}
	})
}

// fireFeedRound re-fetches every feed still queued in feedRetry on fresh
// connections. Failures repopulate feedRetry for the round after.
func (m *Model) fireFeedRound() tea.Cmd {
	gen := m.feedRefreshGen
	targets := m.feedRetry
	m.feedRetry = make(map[string]string)
	cmds := make([]tea.Cmd, 0, len(targets))
	for channelID, url := range targets {
		cmds = append(cmds, m.fetchFeedCmd(channelID, url, gen))
	}
	m.feedsPending = len(cmds)
	m.subscriptionsView.SetLoading(m.feedsPending)
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

// finalizeFeedRefresh resolves feeds still failing at the deadline: show the
// last good cached items if we have them, otherwise a gentle throttle note —
// never a raw status code.
func (m *Model) finalizeFeedRefresh() {
	for channelID := range m.feedRetry {
		if c, ok := m.feedCache[channelID]; ok && c.err == nil && len(c.items) > 0 {
			m.subscriptionsView.Fetched(channelID, c.items, nil)
		} else {
			m.subscriptionsView.Fetched(channelID, nil, errFeedThrottled)
		}
	}
	m.feedRetry = make(map[string]string)
	m.feedsPending = 0
	m.subscriptionsView.SetLoading(0)
}

func (m *Model) updateMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		return m, tea.Quit
	case "enter":
		switch m.menu.GetSelected() {
		case views.MenuSearch:
			m.state = StateSearch
		case views.MenuSubscriptions:
			m.updateSubscriptionsView()
			m.state = StateSubscription
			return m, m.fetchAllFeedsCmd(false)
		case views.MenuQuit:
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.menu, cmd = m.menu.Update(msg)
	return m, cmd
}

func CheckDependencies() error {
	if _, err := exec.LookPath("yt-dlp"); err != nil {
		return fmt.Errorf("yt-dlp not found! Please install it first~ (>_<)")
	}
	if _, err := exec.LookPath("mpv"); err != nil {
		return fmt.Errorf("mpv not found! Can't play videos without it~ (>_<)")
	}
	return nil
}
