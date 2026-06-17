package views

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/user/nyaa-tui/internal/rss"
	"github.com/user/nyaa-tui/internal/subscriptions"
)

func makeChannel(id, name, lastSeen string, itemIDs ...string) ChannelGroup {
	var items []rss.RSSItem
	base := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)
	for i, vid := range itemIDs {
		items = append(items, rss.RSSItem{
			ID:          vid,
			Title:       vid + " title",
			PublishedAt: base.Add(-time.Duration(i) * time.Hour), // newest first
		})
	}
	return ChannelGroup{
		Entry: subscriptions.SubscriptionEntry{
			ChannelID:       id,
			ChannelName:     name,
			LastSeenVideoID: lastSeen,
		},
		Items: items,
	}
}

func TestNewCount(t *testing.T) {
	tests := []struct {
		name     string
		lastSeen string
		items    []string
		want     int
	}{
		{"never read shows zero", "", []string{"v1", "v2", "v3"}, 0},
		{"counts items newer than last seen", "v2", []string{"v1", "v2", "v3"}, 1},
		{"latest already seen shows zero", "v1", []string{"v1", "v2", "v3"}, 0},
		{"last seen aged out counts all", "old", []string{"v1", "v2"}, 2},
		{"no items", "v1", nil, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cg := makeChannel("UCa", "A", tt.lastSeen, tt.items...)
			if got := cg.NewCount(); got != tt.want {
				t.Errorf("NewCount() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestLatestItemIDs(t *testing.T) {
	v := NewSubscriptionsView()
	v.SetChannels([]ChannelGroup{
		makeChannel("UCa", "Alpha", "", "a1", "a2"),
		makeChannel("UCb", "Beta", "", "b1"),
		makeChannel("UCc", "Gamma", ""), // no items
	})

	got := v.LatestItemIDs()
	if got["UCa"] != "a1" {
		t.Errorf("UCa latest = %q, want a1", got["UCa"])
	}
	if got["UCb"] != "b1" {
		t.Errorf("UCb latest = %q, want b1", got["UCb"])
	}
	if _, ok := got["UCc"]; ok {
		t.Error("UCc has no items and should be absent from LatestItemIDs")
	}
}

func TestSelectedChannelID_OnVideo(t *testing.T) {
	v := NewSubscriptionsView()
	// Names chosen so sort order is Alpha (UCa) then Beta (UCb).
	v.SetChannels([]ChannelGroup{
		makeChannel("UCa", "Alpha", "", "a1", "a2"),
		makeChannel("UCb", "Beta", "", "b1"),
	})

	// Cursor starts on the first channel header.
	if got := v.SelectedChannelID(); got != "UCa" {
		t.Fatalf("initial SelectedChannelID = %q, want UCa", got)
	}

	// Enter expands the first channel; ↓ moves onto its first video.
	v, _ = v.Update(tea.KeyMsg{Type: tea.KeyEnter})
	v, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})

	if got := v.SelectedChannelID(); got != "UCa" {
		t.Errorf("SelectedChannelID on video = %q, want parent UCa", got)
	}
}

func TestSetLoadingShowsInHeader(t *testing.T) {
	v := NewSubscriptionsView()
	v.SetChannels([]ChannelGroup{makeChannel("UCa", "Alpha", "", "a1")})
	v.SetDimensions(80, 24)

	v.SetLoading(3)
	if got := v.View(); !contains(got, "fetching 3") {
		t.Errorf("expected loading indicator in header, got:\n%s", got)
	}

	v.SetLoading(0)
	if got := v.View(); contains(got, "fetching") {
		t.Errorf("expected no loading indicator when pending=0, got:\n%s", got)
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
