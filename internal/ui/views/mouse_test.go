package views

import (
	"fmt"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/user/nyaa-tui/internal/rss"
	"github.com/user/nyaa-tui/internal/yt"
)

func makeRSSItems(n int) []rss.RSSItem {
	var items []rss.RSSItem
	for i := 0; i < n; i++ {
		items = append(items, rss.RSSItem{
			ID:          fmt.Sprintf("v%d", i),
			Title:       fmt.Sprintf("Video %d", i),
			URL:         fmt.Sprintf("https://youtu.be/v%d", i),
			PublishedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		})
	}
	return items
}

// At 80x24 a single-line title yields this layout: row0 title, row1 blank,
// row2 box top border, row3 top padding, row4 first body line. So the list's
// first row sits at screen row 4.
const listBodyTop = 4

func makeVideos(n int) []yt.Video {
	var v []yt.Video
	for i := 0; i < n; i++ {
		v = append(v, yt.Video{ID: fmt.Sprint(i), Title: fmt.Sprintf("Video %d", i), Channel: "Chan"})
	}
	return v
}

func TestResultsItemAt(t *testing.T) {
	r := NewResultsList(makeVideos(20), 80, 24)

	cases := []struct {
		y    int
		want int
		ok   bool
	}{
		{listBodyTop - 1, 0, false}, // padding row, above the first item
		{listBodyTop, 0, true},      // item 0 title row
		{listBodyTop + 1, 0, true},  // item 0 meta row (still item 0)
		{listBodyTop + 2, 1, true},  // item 1 title row
		{listBodyTop + 4, 2, true},  // item 2 title row
	}
	for _, c := range cases {
		got, ok := r.ItemAt(c.y)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("ItemAt(%d) = (%d,%v), want (%d,%v)", c.y, got, ok, c.want, c.ok)
		}
	}
}

// When the list is scrolled by keyboard navigation, the top visible row maps to
// the scrolled-in item, not item 0 — proving ItemAt accounts for the scroll
// window.
func TestResultsItemAtScrolled(t *testing.T) {
	r := NewResultsList(makeVideos(20), 80, 24)
	r, _ = r.Update(tea.KeyMsg{Type: tea.KeyEnd}) // jump to the bottom so the list scrolls

	// innerH=16 → 8 items visible → first visible item is 19-8+1 = 12.
	got, ok := r.ItemAt(listBodyTop)
	if !ok || got != 12 {
		t.Fatalf("ItemAt(top) after scroll = (%d,%v), want (12,true)", got, ok)
	}
	// The cursor's own row should map back to the cursor.
	if got, ok := r.ItemAt(listBodyTop + (19-12)*2); !ok || got != 19 {
		t.Errorf("ItemAt(cursor row) = (%d,%v), want (19,true)", got, ok)
	}
}

// Hover (SetCursor) moves the highlight but must NOT scroll the list: the window
// stays put even when the cursor lands on the last item, so ItemAt still reports
// the same top row it did before the hover.
func TestResultsHoverDoesNotScroll(t *testing.T) {
	r := NewResultsList(makeVideos(20), 80, 24)

	r.SetCursor(19) // simulate hovering the bottom item
	if got, ok := r.ItemAt(listBodyTop); !ok || got != 0 {
		t.Errorf("ItemAt(top) after hover = (%d,%v), want (0,true) — hover must not scroll", got, ok)
	}
	if r.SelectedVideo() == nil || r.SelectedVideo().ID != "19" {
		t.Errorf("hover should still move the highlight to item 19, got %v", r.SelectedVideo())
	}
}

func TestMenuItemAt(t *testing.T) {
	m := NewMenu(3)
	m.SetDimensions(80, 24)

	// Banner title (5 rows) is centered above three menu rows at 13/14/15.
	for y, want := range map[int]int{13: 0, 14: 1, 15: 2} {
		if got, ok := m.ItemAt(y); !ok || got != want {
			t.Errorf("menu ItemAt(%d) = (%d,%v), want (%d,true)", y, got, ok, want)
		}
	}
	if _, ok := m.ItemAt(12); ok {
		t.Error("menu ItemAt(12) should miss (above the items)")
	}
	if _, ok := m.ItemAt(16); ok {
		t.Error("menu ItemAt(16) should miss (below the items)")
	}
}

func TestSubscriptionsItemAt(t *testing.T) {
	v := NewSubscriptionsView()
	v.SetChannels([]ChannelGroup{
		makeChannel("UCa", "Alpha", "", "a1", "a2"),
		makeChannel("UCb", "Beta", "", "b1"),
	})
	v.SetDimensions(80, 24)

	// Collapsed: row 4 = Alpha header (flat index 0), row 5 = Beta (index 1).
	if got, ok := v.ItemAt(listBodyTop); !ok || got != 0 {
		t.Errorf("ItemAt(top) = (%d,%v), want (0,true)", got, ok)
	}
	if got, ok := v.ItemAt(listBodyTop + 1); !ok || got != 1 {
		t.Errorf("ItemAt(second row) = (%d,%v), want (1,true)", got, ok)
	}

	// Expand Alpha; its videos now occupy flat indices 1 and 2.
	v, _ = v.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if got, ok := v.ItemAt(listBodyTop + 1); !ok || got != 1 {
		t.Errorf("expanded ItemAt(a1 row) = (%d,%v), want (1,true)", got, ok)
	}
	if got, ok := v.ItemAt(listBodyTop + 2); !ok || got != 2 {
		t.Errorf("expanded ItemAt(a2 row) = (%d,%v), want (2,true)", got, ok)
	}
}

func TestChannelBrowseItemAt(t *testing.T) {
	c := NewChannelBrowse("UCx", "Chan", makeRSSItems(10))
	c.SetDimensions(80, 24)

	if got, ok := c.ItemAt(listBodyTop); !ok || got != 0 {
		t.Errorf("ItemAt(top) = (%d,%v), want (0,true)", got, ok)
	}
	if got, ok := c.ItemAt(listBodyTop + 3); !ok || got != 3 {
		t.Errorf("ItemAt(row 3) = (%d,%v), want (3,true)", got, ok)
	}
	if _, ok := c.ItemAt(listBodyTop - 1); ok {
		t.Error("ItemAt above the list should miss")
	}
}
