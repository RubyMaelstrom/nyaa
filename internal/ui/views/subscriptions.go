package views

import (
	"fmt"
	"sort"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/user/nyaa-tui/internal/rss"
	"github.com/user/nyaa-tui/internal/subscriptions"
	"github.com/user/nyaa-tui/internal/ui/theme"
)

type viewMode int

const (
	channelView viewMode = iota
	recentView
)

type SubscriptionsView struct {
	channels    []ChannelGroup
	cursor      int
	width       int
	height      int
	toastHeight int
	mode        viewMode
	pending     int
}

type ChannelGroup struct {
	Entry    subscriptions.SubscriptionEntry
	Items    []rss.RSSItem
	Err      error
	Expanded bool
}

func NewChannelGroup(entry subscriptions.SubscriptionEntry) ChannelGroup {
	return ChannelGroup{
		Entry: entry,
	}
}

func NewSubscriptionsView() SubscriptionsView {
	return SubscriptionsView{
		channels: []ChannelGroup{},
		cursor:   0,
	}
}

func (v SubscriptionsView) Init() tea.Cmd {
	return nil
}

type PlayVideoMsg struct {
	Video rss.RSSItem
}

func (v SubscriptionsView) Update(msg tea.Msg) (SubscriptionsView, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			if v.mode == channelView {
				v.mode = recentView
			} else {
				v.mode = channelView
			}
			v.cursor = 0
			return v, nil
		case "up", "k":
			if v.cursor > 0 {
				v.cursor--
			}
		case "down", "j":
			maxIdx := v.visibleItemCount() - 1
			if maxIdx >= 0 && v.cursor < maxIdx {
				v.cursor++
			}
		case "pageup":
			v.cursor -= 10
			if v.cursor < 0 {
				v.cursor = 0
			}
		case "pagedown":
			maxIdx := v.visibleItemCount() - 1
			if v.cursor+10 < maxIdx {
				v.cursor += 10
			} else if maxIdx >= 0 {
				v.cursor = maxIdx
			}
		case "enter":
			if v.mode == recentView {
				return v.playRecentItem()
			}
			return v.playOrToggleChannel()
		}
	}
	return v, nil
}

func (v *SubscriptionsView) SetDimensions(width, height int) {
	v.width = width
	v.height = height
}

func (v *SubscriptionsView) SetToastLines(n int) {
	v.toastHeight = n
}

// SetLoading records how many feeds are still being fetched, shown in the header.
func (v *SubscriptionsView) SetLoading(pending int) {
	v.pending = pending
}

// SelectedChannelID returns the channel under the cursor (the parent channel
// when a video is selected), or "" if nothing is selected.
func (v SubscriptionsView) SelectedChannelID() string {
	if v.mode == recentView {
		recent := v.buildRecentList()
		if v.cursor < 0 || v.cursor >= len(recent) {
			return ""
		}
		target := recent[v.cursor].ID
		for _, ch := range v.channels {
			for _, it := range ch.Items {
				if it.ID == target {
					return ch.Entry.ChannelID
				}
			}
		}
		return ""
	}
	flat := v.buildFlatList()
	if v.cursor < 0 || v.cursor >= len(flat) {
		return ""
	}
	return flat[v.cursor].channelID
}

// LatestItemIDs maps each channel to its newest video ID (for "mark all read").
func (v SubscriptionsView) LatestItemIDs() map[string]string {
	out := make(map[string]string)
	for _, ch := range v.channels {
		if len(ch.Items) > 0 {
			out[ch.Entry.ChannelID] = ch.Items[0].ID
		}
	}
	return out
}

func (v *SubscriptionsView) SetChannels(channels []ChannelGroup) {
	expandedMap := make(map[string]bool)
	for _, old := range v.channels {
		if old.Expanded {
			expandedMap[old.Entry.ChannelID] = true
		}
	}
	sort.Slice(channels, func(i, j int) bool {
		return channels[i].Entry.ChannelName < channels[j].Entry.ChannelName
	})
	for i := range channels {
		if expandedMap[channels[i].Entry.ChannelID] {
			channels[i].Expanded = true
		}
	}
	v.channels = channels
	v.cursor = 0
	v.mode = channelView
}

func (v SubscriptionsView) flatListLen() int {
	var count int
	for _, ch := range v.channels {
		count++
		if ch.Expanded {
			count += len(ch.Items)
		}
	}
	return count
}

type flatListEntry struct {
	channelID string
	isChannel bool
	video     *rss.RSSItem
	title     string
}

func (v SubscriptionsView) buildFlatList() []flatListEntry {
	var flat []flatListEntry
	for _, ch := range v.channels {
		flat = append(flat, flatListEntry{
			channelID: ch.Entry.ChannelID,
			isChannel: true,
		})
		if ch.Expanded {
			for i := range ch.Items {
				flat = append(flat, flatListEntry{
					channelID: ch.Entry.ChannelID,
					isChannel: false,
					video:     &ch.Items[i],
					title:     ch.Items[i].Title,
				})
			}
		}
	}
	return flat
}

func (v SubscriptionsView) visibleItemCount() int {
	switch v.mode {
	case recentView:
		recent := v.buildRecentList()
		return len(recent)
	case channelView:
		return v.flatListLen()
	default:
		return 0
	}
}

func (v SubscriptionsView) playRecentItem() (SubscriptionsView, tea.Cmd) {
	recent := v.buildRecentList()
	if v.cursor < 0 || v.cursor >= len(recent) {
		return v, nil
	}
	item := recent[v.cursor]
	playMsg := PlayVideoMsg{Video: item}
	return v, func() tea.Msg { return playMsg }
}

func (v SubscriptionsView) playOrToggleChannel() (SubscriptionsView, tea.Cmd) {
	flat := v.buildFlatList()
	if v.cursor < 0 {
		v.cursor = 0
	}
	if v.cursor >= len(flat) {
		return v, nil
	}
	item := flat[v.cursor]
	if !item.isChannel {
		playMsg := PlayVideoMsg{Video: *item.video}
		return v, func() tea.Msg { return playMsg }
	}
	for i := range v.channels {
		if v.channels[i].Entry.ChannelID == item.channelID {
			v.channels[i].Expanded = !v.channels[i].Expanded
			if !v.channels[i].Expanded {
				channelStart := v.cursor
				v.cursor = v.cursor - len(v.channels[i].Items)
				if v.cursor < channelStart {
					v.cursor = channelStart
				}
			}
			break
		}
	}
	return v, nil
}

const maxRecentVideos = 30

func (v SubscriptionsView) buildRecentList() []rss.RSSItem {
	var allItems []rss.RSSItem
	for _, ch := range v.channels {
		allItems = append(allItems, ch.Items...)
	}

	sort.Slice(allItems, func(i, j int) bool {
		return allItems[i].PublishedAt.After(allItems[j].PublishedAt)
	})

	if len(allItems) > maxRecentVideos {
		return allItems[:maxRecentVideos]
	}
	return allItems
}

func (v *SubscriptionsView) View() string {
	if len(v.channels) == 0 {
		return v.renderEmptyState()
	}

	switch v.mode {
	case recentView:
		return v.renderRecentView()
	default:
		return v.renderChannelView()
	}
}

// header renders the view title, appending a fetch indicator while feeds load.
func (v *SubscriptionsView) header(base string) string {
	if v.pending > 0 {
		base += fmt.Sprintf("  ·  fetching %d…", v.pending)
	}
	return theme.Theme.TitleStyle.Render(base)
}

// frameHeight is the terminal height available to the frame, minus any rows
// the toast notification reserves at the bottom.
func (v *SubscriptionsView) frameHeight() int {
	return v.height - v.toastHeight
}

func (v *SubscriptionsView) renderEmptyState() string {
	var title string
	if v.mode == recentView {
		title = v.header("♡ recent videos ◡ 30 most recent ♡")
	} else {
		title = v.header("♡ subscriptions (≧◡≦) ♡")
	}

	body := lipgloss.JoinVertical(lipgloss.Center,
		theme.Theme.DimStyle.Render("no subscriptions yet~"),
		theme.Theme.DimStyle.Render("press 's' on any video in search to subscribe ♡"),
		theme.Theme.DimStyle.Render("or import a YouTube export:  nyaa import-opml <file>"),
	)
	footer := theme.Theme.DimStyle.Render("esc to menu")

	return Frame(v.width, v.frameHeight(), title, footer, lipgloss.Center, lipgloss.Center,
		func(innerW, innerH int) string { return body })
}

func (v *SubscriptionsView) renderChannelView() string {
	title := v.header("♡ subscriptions (≧◡≦) ♡")
	hints := theme.Theme.DimStyle.Render(
		"↑↓ navigate  •  tab recent  •  enter expand/play  •  x unsub  •  r refresh  •  d mark read  •  esc menu",
	)

	flat := v.buildFlatList()
	v.clampCursor()

	// Render every row up front, tracking the line that holds the cursor so the
	// body closure can scroll a fitted window around it.
	rows, cursorRow := v.channelRows(flat)

	return Frame(v.width, v.frameHeight(), title, hints, lipgloss.Left, lipgloss.Top,
		func(innerW, innerH int) string {
			return windowedLines(rows, cursorRow, innerW, innerH)
		})
}

// channelRows renders the flat channel/video list to display lines and returns
// them along with the line index of the currently selected entry.
func (v *SubscriptionsView) channelRows(flat []flatListEntry) (rows []string, cursorRow int) {
	for i, item := range flat {
		if i == v.cursor {
			cursorRow = len(rows)
		}

		if item.isChannel {
			var chGroup *ChannelGroup
			for j := range v.channels {
				if v.channels[j].Entry.ChannelID == item.channelID {
					chGroup = &v.channels[j]
					break
				}
			}
			if chGroup == nil {
				continue
			}

			var dateStr string
			if len(chGroup.Items) > 0 {
				dateStr = chGroup.LatestPublished().Format("Jan 2, 2006")
			}
			label := chGroup.Entry.ChannelName + "  ♡  " + dateStr
			if n := chGroup.NewCount(); n > 0 {
				label += "  " + theme.Theme.KaomojiStyle.Render(fmt.Sprintf("(%d new)", n))
			}

			prefix := "▸"
			if chGroup.Expanded {
				prefix = "▾"
			}
			switch {
			case i == v.cursor:
				rows = append(rows, theme.Theme.SelectedStyle.Render(prefix+" "+label))
			case chGroup.Expanded:
				rows = append(rows, theme.Theme.BaseStyle.Render(prefix+" "+label))
			default:
				rows = append(rows, theme.Theme.DimStyle.Render(prefix+" "+label))
			}

			if chGroup.Err != nil {
				rows = append(rows, theme.Theme.ErrorStyle.Render("  ✗ "+chGroup.Err.Error()))
			}
		} else if item.video != nil {
			if i == v.cursor {
				rows = append(rows, theme.Theme.SelectedStyle.Render("▸ "+item.video.Title))
			} else {
				rows = append(rows, theme.Theme.BaseStyle.Render("  • "+item.video.Title))
			}
		}
	}
	return rows, cursorRow
}

func (v *SubscriptionsView) renderRecentView() string {
	title := v.header("♡ recent videos ◡ 30 most recent ♡")
	allItems := v.buildRecentList()

	if len(allItems) == 0 {
		body := theme.Theme.DimStyle.Render("no new videos yet~ (◕‿◕)")
		footer := theme.Theme.DimStyle.Render("tab channels  •  ↑↓ navigate  •  enter play  •  r refresh  •  esc menu")
		return Frame(v.width, v.frameHeight(), title, footer, lipgloss.Center, lipgloss.Center,
			func(innerW, innerH int) string { return body })
	}

	v.clampCursor()

	// Map each video to its channel once, rather than scanning per row.
	nameByVideo := make(map[string]string, len(allItems))
	for _, ch := range v.channels {
		if ch.Err != nil {
			continue
		}
		for _, it := range ch.Items {
			nameByVideo[it.ID] = ch.Entry.ChannelName
		}
	}

	rows := make([]string, 0, len(allItems))
	for i, item := range allItems {
		dateStr := item.PublishedAt.Format("Jan 2, 2006")
		meta := dateStr
		if name := nameByVideo[item.ID]; name != "" {
			meta = fmt.Sprintf("%s  ♡  %s", dateStr, name)
		}
		displayTitle := fmt.Sprintf("%s  %s", meta, item.Title)
		if i == v.cursor {
			rows = append(rows, theme.Theme.SelectedStyle.Render("▸ "+displayTitle))
		} else {
			rows = append(rows, theme.Theme.BaseStyle.Render("  • "+displayTitle))
		}
	}

	footer := theme.Theme.DimStyle.Render(
		fmt.Sprintf("tab channels  •  ↑↓ navigate  •  enter play  •  %d videos  •  r refresh  •  esc menu", len(allItems)),
	)
	return Frame(v.width, v.frameHeight(), title, footer, lipgloss.Left, lipgloss.Top,
		func(innerW, innerH int) string {
			return windowedLines(rows, v.cursor, innerW, innerH)
		})
}

func (cg *ChannelGroup) LatestPublished() time.Time {
	for _, item := range cg.Items {
		return item.PublishedAt
	}
	return time.Time{}
}

// NewCount returns how many items are newer than the last-seen video. A channel
// that has never been marked read reports 0 to avoid flooding fresh subs.
func (cg *ChannelGroup) NewCount() int {
	if cg.Entry.LastSeenVideoID == "" {
		return 0
	}
	n := 0
	for _, item := range cg.Items {
		if item.ID == cg.Entry.LastSeenVideoID {
			break
		}
		n++
	}
	return n
}

func (v *SubscriptionsView) clampCursor() {
	total := len(v.buildFlatList())
	if v.mode == recentView {
		total = len(v.buildRecentList())
	}
	if total <= 0 {
		v.cursor = 0
		return
	}
	if v.cursor >= total {
		v.cursor = total - 1
	}
}

func (v *SubscriptionsView) Fetched(channelID string, items []rss.RSSItem, err error) {
	for i := range v.channels {
		if v.channels[i].Entry.ChannelID == channelID {
			v.channels[i].Items = items
			v.channels[i].Err = err
			return
		}
	}
}
