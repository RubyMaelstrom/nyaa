package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/nyaa/internal/ui/views"
	"github.com/user/nyaa/internal/yt"
)

func key(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

// newTestModel isolates config (subscriptions) into a temp dir so tests don't
// read or write the developer's real ~/.config/nyaa.
func newTestModel(t *testing.T) Model {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	return InitialModel()
}

// asModel normalizes the tea.Model returned by Update, which may be a Model
// (value-receiver paths) or *Model (pointer-receiver sub-handlers).
func asModel(res tea.Model) Model {
	switch v := res.(type) {
	case Model:
		return v
	case *Model:
		return *v
	default:
		panic("unexpected model type")
	}
}

// Help opens from any state and closing it returns to where it was opened.
func TestHelpReturnsToOpeningState(t *testing.T) {
	m := newTestModel(t)
	m.state = StateSubscription

	res, _ := m.Update(key("?"))
	m2 := asModel(res)
	if m2.state != StateHelp {
		t.Fatalf("after '?' state = %v, want StateHelp", m2.state)
	}
	if m2.helpReturn != StateSubscription {
		t.Fatalf("helpReturn = %v, want StateSubscription", m2.helpReturn)
	}

	res2, _ := m2.Update(key("?"))
	m3 := asModel(res2)
	if m3.state != StateSubscription {
		t.Errorf("after closing help, state = %v, want StateSubscription", m3.state)
	}
}

// '?' must not toggle help while typing a search query.
func TestHelpIgnoredInSearch(t *testing.T) {
	m := newTestModel(t)
	m.state = StateSearch

	res, _ := m.Update(key("?"))
	m2 := asModel(res)
	if m2.state != StateSearch {
		t.Errorf("'?' in search changed state to %v, want StateSearch", m2.state)
	}
}

// q quits from the subscriptions view (previously it was swallowed).
func TestQuitFromSubscriptions(t *testing.T) {
	m := newTestModel(t)
	m.state = StateSubscription

	_, cmd := m.Update(key("q"))
	if cmd == nil {
		t.Fatal("expected a quit command, got nil")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("expected tea.QuitMsg from q in subscriptions")
	}
}

// A fresh cache means fetchAllFeedsCmd issues a batch; a warm cache skips it.
func TestFetchAllFeedsCmdUsesCache(t *testing.T) {
	m := newTestModel(t)
	m.subs.Add("UCa", "Alpha")
	m.updateSubscriptionsView()

	// Cold cache: one feed to fetch.
	if cmd := m.fetchAllFeedsCmd(false); cmd == nil {
		t.Fatal("cold cache should produce a fetch command")
	}
	if m.feedsPending != 1 {
		t.Errorf("feedsPending = %d, want 1", m.feedsPending)
	}

	// Warm the cache as if the fetch completed, then re-enter.
	m.handleFeedFetched(feedFetchedMsg{channelID: "UCa", items: nil, gen: m.feedRefreshGen})
	if cmd := m.fetchAllFeedsCmd(false); cmd != nil {
		t.Error("warm cache should skip fetching (nil cmd)")
	}
	if m.feedsPending != 0 {
		t.Errorf("feedsPending after cache hit = %d, want 0", m.feedsPending)
	}

	// force=true always refetches.
	if cmd := m.fetchAllFeedsCmd(true); cmd == nil {
		t.Error("force refresh should produce a fetch command even when cached")
	}
}

func TestGuiAvailable(t *testing.T) {
	t.Setenv("DISPLAY", "")
	t.Setenv("WAYLAND_DISPLAY", "")
	if guiAvailable() {
		t.Error("guiAvailable() should be false with no display vars")
	}
	t.Setenv("DISPLAY", ":0")
	if !guiAvailable() {
		t.Error("guiAvailable() should be true with DISPLAY set")
	}
	t.Setenv("DISPLAY", "")
	t.Setenv("WAYLAND_DISPLAY", "wayland-0")
	if !guiAvailable() {
		t.Error("guiAvailable() should be true with WAYLAND_DISPLAY set")
	}
}

// The TTY banner names the video so the user knows what's playing when the TUI
// drops away.
func TestNowPlayingBanner(t *testing.T) {
	m := newTestModel(t)
	if m.nowPlayingBanner() != "" {
		t.Error("banner should be empty when nothing is playing")
	}
	m.nowPlaying = &yt.Video{Title: "Cute Cats Compilation", Channel: "Meow TV"}
	banner := m.nowPlayingBanner()
	if !strings.Contains(banner, "Cute Cats Compilation") || !strings.Contains(banner, "Meow TV") {
		t.Errorf("banner missing title/channel: %q", banner)
	}
	if !strings.Contains(banner, "now playing") {
		t.Errorf("banner missing heading: %q", banner)
	}
}

// Pressing 'a' in results toggles audio-only playback.
func TestAudioToggle(t *testing.T) {
	m := newTestModel(t)
	m.state = StateResults
	if m.audioOnly {
		t.Fatal("audioOnly should start false")
	}

	res, cmd := m.Update(key("a"))
	m = asModel(res)
	if !m.audioOnly {
		t.Error("expected audioOnly true after first toggle")
	}
	if cmd == nil {
		t.Error("expected a toast command from toggle")
	}

	res, _ = m.Update(key("a"))
	m = asModel(res)
	if m.audioOnly {
		t.Error("expected audioOnly false after second toggle")
	}
}

// 'c' on a result that has a channel ID kicks off a channel-feed load.
func TestBrowseChannelStartsLoad(t *testing.T) {
	m := newTestModel(t)
	m.results = views.NewResultsList([]yt.Video{
		{ID: "v1", Title: "Vid", Channel: "Chan", ChannelID: "UCabc", URL: "http://x"},
	}, 80, 24)
	m.state = StateResults

	res, cmd := m.Update(key("o"))
	m = asModel(res)
	if m.state != StateLoading {
		t.Errorf("state = %v, want StateLoading", m.state)
	}
	if cmd == nil {
		t.Fatal("expected a channel-fetch command")
	}

	// A channelFeedMsg should land us in the channel view.
	res, _ = m.Update(channelFeedMsg{channelID: "UCabc", name: "Chan", items: nil})
	m = asModel(res)
	if m.state != StateChannel {
		t.Errorf("state = %v, want StateChannel", m.state)
	}
}

// A view with an active toast must still fit the terminal: the list shrinks by
// the toast's height so the composed output doesn't overflow the bottom.
func TestViewWithToastFitsTerminal(t *testing.T) {
	m := newTestModel(t)
	for i := 0; i < 8; i++ {
		m.subs.Add(string(rune('a'+i)), "Channel")
	}
	m.updateSubscriptionsView()

	res, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = asModel(res)
	m.state = StateSubscription
	m.toast = "unsubscribed from Channel (｡•́︿•̀｡)"
	m.toastTimer = 2.0

	out := m.View()
	if h := lipgloss.Height(out); h > 24 {
		t.Errorf("view with toast height %d exceeds terminal height 24", h)
	}
}

// When playback fails, [R] Retry must replay the failed video — even when there
// was no prior search (played from subscriptions/a channel), the case where
// Retry used to silently do nothing.
func TestRetryReplaysFailedPlayback(t *testing.T) {
	m := newTestModel(t)
	m.lastQuery = "" // came from subscriptions, not a search
	m.previousState = StateSubscription
	m.state = StatePlaying
	m.nowPlaying = &yt.Video{Title: "Cute Cats", URL: "https://youtu.be/abc123"}

	// A permanent error goes straight to the error screen (transient throttles
	// are auto-retried instead — see TestAutoRetryOnThrottle).
	res, _ := m.Update(playFinishedMsg{err: someErr("Playback ended unexpectedly")})
	m = asModel(res)
	if m.state != StateError {
		t.Fatalf("after failed playback state = %v, want StateError", m.state)
	}
	if m.retryVideo == nil || m.retryVideo.URL != "https://youtu.be/abc123" {
		t.Fatalf("retryVideo = %v, want the failed video", m.retryVideo)
	}

	res, cmd := m.Update(key("r"))
	m = asModel(res)
	if m.state != StatePlaying {
		t.Fatalf("after [R] Retry state = %v, want StatePlaying", m.state)
	}
	if m.nowPlaying == nil || m.nowPlaying.URL != "https://youtu.be/abc123" {
		t.Fatalf("nowPlaying = %v, want the replayed video", m.nowPlaying)
	}
	if cmd == nil {
		t.Error("Retry should return a playback command, got nil")
	}
}

// A search/extraction error clears retryVideo so [R] Retry re-runs the search
// instead of replaying a stale video.
func TestSearchErrorDoesNotReplayVideo(t *testing.T) {
	m := newTestModel(t)
	m.retryVideo = &yt.Video{URL: "https://youtu.be/stale"}

	res, _ := m.Update(errorMsg{err: someErr("no results")})
	m = asModel(res)
	if m.retryVideo != nil {
		t.Fatalf("search error should clear retryVideo, got %v", m.retryVideo)
	}
}

// A transient stream-open failure (mpv exit 2) is silently retried on a fresh
// launch up to maxPlayRetries times before the error screen ever appears.
func TestAutoRetryOnThrottle(t *testing.T) {
	m := newTestModel(t)
	m.state = StatePlaying
	m.nowPlaying = &yt.Video{Title: "Obscure Clip", URL: "https://youtu.be/xyz789"}

	// Each throttle hit keeps us in StatePlaying and schedules another attempt.
	for i := 1; i <= maxPlayRetries; i++ {
		res, cmd := m.Update(playFinishedMsg{err: someErr("exit status 2")})
		m = asModel(res)
		if m.state != StatePlaying {
			t.Fatalf("attempt %d: state = %v, want StatePlaying (auto-retrying)", i, m.state)
		}
		if m.playAttempts != i {
			t.Fatalf("attempt %d: playAttempts = %d, want %d", i, m.playAttempts, i)
		}
		if cmd == nil {
			t.Fatalf("attempt %d: expected a retry command, got nil", i)
		}
		if m.nowPlaying == nil {
			t.Fatalf("attempt %d: nowPlaying cleared, retry would have no URL", i)
		}
	}

	// Budget exhausted: the next failure surfaces the error screen with the
	// video remembered for manual [R] Retry.
	res, _ := m.Update(playFinishedMsg{err: someErr("exit status 2")})
	m = asModel(res)
	if m.state != StateError {
		t.Fatalf("after exhausting retries state = %v, want StateError", m.state)
	}
	if m.retryVideo == nil || m.retryVideo.URL != "https://youtu.be/xyz789" {
		t.Fatalf("retryVideo = %v, want the failed video", m.retryVideo)
	}
	if m.playAttempts != 0 {
		t.Errorf("playAttempts = %d, want 0 (reset once we give up)", m.playAttempts)
	}
}

type someErr string

func (e someErr) Error() string { return string(e) }
