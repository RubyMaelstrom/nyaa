package views

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/nyaa/internal/rss"
	"github.com/user/nyaa/internal/yt"
)

var termSizes = [][2]int{{80, 24}, {120, 40}, {100, 30}, {40, 14}, {200, 60}}

func assertFits(t *testing.T, label string, out string, w, h int) {
	t.Helper()
	if gh := lipgloss.Height(out); gh > h {
		t.Errorf("%s at %dx%d: height %d exceeds terminal height %d", label, w, h, gh, h)
	}
	for i, line := range strings.Split(out, "\n") {
		if lw := lipgloss.Width(line); lw > w {
			t.Errorf("%s at %dx%d: line %d width %d exceeds terminal width %d", label, w, h, i, lw, w)
		}
	}
}

// The reported bug: many expanded channels scrolled the header off the top.
// The frame must keep every render within the terminal, at any size and with
// the cursor anywhere in the list.
func TestChannelViewFitsTerminal(t *testing.T) {
	var chans []ChannelGroup
	for c := 0; c < 10; c++ {
		cg := makeChannel(
			fmt.Sprintf("UC%02d", c),
			fmt.Sprintf("Channel number %d with a fairly long display name", c),
			"",
			"v1", "v2", "v3", "v4", "v5",
		)
		cg.Items[0].Title = "An extremely long video title that definitely exceeds any reasonable terminal width and then keeps going"
		cg.Expanded = true
		chans = append(chans, cg)
	}

	v := NewSubscriptionsView()
	v.SetChannels(chans)

	for _, s := range termSizes {
		w, h := s[0], s[1]
		v.SetDimensions(w, h)

		// Top of the list.
		assertFits(t, "channel-view top", v.View(), w, h)

		// Drive the cursor to the bottom and re-check (scrolling must still fit).
		for i := 0; i < 200; i++ {
			v, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		}
		assertFits(t, "channel-view bottom", v.View(), w, h)

		// Reset cursor for the next size.
		v, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
		for i := 0; i < 200; i++ {
			v, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
		}
	}
}

func TestRecentViewFitsTerminal(t *testing.T) {
	var chans []ChannelGroup
	for c := 0; c < 6; c++ {
		chans = append(chans, makeChannel(fmt.Sprintf("UC%d", c), fmt.Sprintf("Chan %d", c), "", "v1", "v2", "v3", "v4", "v5"))
	}
	v := NewSubscriptionsView()
	v.SetChannels(chans)
	v, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("\t")}) // switch to recent view

	for _, s := range termSizes {
		v.SetDimensions(s[0], s[1])
		assertFits(t, "recent-view", v.View(), s[0], s[1])
	}
}

func TestResultsViewFitsTerminal(t *testing.T) {
	var videos []yt.Video
	for i := 0; i < 20; i++ {
		videos = append(videos, yt.Video{
			ID:         fmt.Sprintf("id%d", i),
			Title:      "A very long video title that goes well past the edge of a narrow terminal window and keeps on going",
			Channel:    "Some Channel With A Long Name",
			Duration:   "12:34",
			ViewCount:  "1.2M",
			UploadDate: "Jan 1, 2024",
		})
	}
	for _, s := range termSizes {
		r := NewResultsList(videos, s[0], s[1])
		assertFits(t, "results", r.View(), s[0], s[1])
	}
}

func TestChannelBrowseFitsTerminal(t *testing.T) {
	var items []rss.RSSItem
	for i := 0; i < 25; i++ {
		items = append(items, rss.RSSItem{
			ID:          fmt.Sprintf("v%d", i),
			Title:       "An exceedingly long channel video title that runs way past the edge of any sensible terminal width",
			PublishedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		})
	}
	c := NewChannelBrowse("UCx", "A Channel With A Fairly Long Name", items)

	for _, s := range termSizes {
		w, h := s[0], s[1]
		c.SetDimensions(w, h)
		assertFits(t, "channel-browse top", c.View(), w, h)
		for i := 0; i < 50; i++ {
			c, _ = c.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		}
		assertFits(t, "channel-browse bottom", c.View(), w, h)
		for i := 0; i < 50; i++ {
			c, _ = c.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
		}
	}
}

func TestEmptyAndSmallViewsFit(t *testing.T) {
	for _, s := range termSizes {
		w, h := s[0], s[1]

		empty := NewSubscriptionsView()
		empty.SetDimensions(w, h)
		assertFits(t, "subscriptions-empty", empty.View(), w, h)

		menu := NewMenu(3)
		menu.SetDimensions(w, h)
		assertFits(t, "menu", menu.View(), w, h)

		search := NewSearchInput()
		search.SetDimensions(w, h)
		assertFits(t, "search", search.View(), w, h)
	}
}
