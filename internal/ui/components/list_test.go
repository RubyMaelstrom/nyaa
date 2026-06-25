package components

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/user/nyaa/internal/yt"
)

func TestNewList(t *testing.T) {
	videos := []yt.Video{
		{ID: "1", Title: "Video 1"},
		{ID: "2", Title: "Video 2"},
	}

	l := NewList(videos, lipgloss.NewStyle(), lipgloss.NewStyle(), lipgloss.NewStyle())

	if l.Length() != 2 {
		t.Errorf("NewList() length = %d, want 2", l.Length())
	}
	if l.Cursor() != 0 {
		t.Errorf("NewList() cursor = %d, want 0", l.Cursor())
	}
}

func TestListCursorMovement(t *testing.T) {
	videos := []yt.Video{
		{ID: "1", Title: "Video 1"},
		{ID: "2", Title: "Video 2"},
		{ID: "3", Title: "Video 3"},
	}

	l := NewList(videos, lipgloss.NewStyle(), lipgloss.NewStyle(), lipgloss.NewStyle())

	// CursorDown
	l.CursorDown()
	if l.Cursor() != 1 {
		t.Errorf("CursorDown() cursor = %d, want 1", l.Cursor())
	}

	// CursorDown again
	l.CursorDown()
	if l.Cursor() != 2 {
		t.Errorf("CursorDown() cursor = %d, want 2", l.Cursor())
	}

	// CursorDown at bottom should stay
	l.CursorDown()
	if l.Cursor() != 2 {
		t.Errorf("CursorDown() at bottom cursor = %d, want 2", l.Cursor())
	}

	// CursorUp
	l.CursorUp()
	if l.Cursor() != 1 {
		t.Errorf("CursorUp() cursor = %d, want 1", l.Cursor())
	}

	// CursorUp to top
	l.CursorUp()
	if l.Cursor() != 0 {
		t.Errorf("CursorUp() cursor = %d, want 0", l.Cursor())
	}

	// CursorUp at top should stay
	l.CursorUp()
	if l.Cursor() != 0 {
		t.Errorf("CursorUp() at top cursor = %d, want 0", l.Cursor())
	}
}

func TestListPageMovement(t *testing.T) {
	videos := make([]yt.Video, 25)
	for i := range videos {
		videos[i] = yt.Video{ID: string(rune(i)), Title: "Video"}
	}

	l := NewList(videos, lipgloss.NewStyle(), lipgloss.NewStyle(), lipgloss.NewStyle())
	l.SetDimensions(80, 24)

	// PageDown
	l.PageDown()
	if l.Cursor() != 10 {
		t.Errorf("PageDown() cursor = %d, want 10", l.Cursor())
	}

	// PageDown again
	l.PageDown()
	if l.Cursor() != 20 {
		t.Errorf("PageDown() cursor = %d, want 20", l.Cursor())
	}

	// PageDown near bottom should clamp
	l.PageDown()
	if l.Cursor() != 24 {
		t.Errorf("PageDown() at bottom cursor = %d, want 24", l.Cursor())
	}

	// GoTop
	l.GoTop()
	if l.Cursor() != 0 {
		t.Errorf("GoTop() cursor = %d, want 0", l.Cursor())
	}

	// GoBottom
	l.GoBottom()
	if l.Cursor() != 24 {
		t.Errorf("GoBottom() cursor = %d, want 24", l.Cursor())
	}
}

func TestListSelectedItem(t *testing.T) {
	videos := []yt.Video{
		{ID: "1", Title: "Video 1"},
		{ID: "2", Title: "Video 2"},
	}

	l := NewList(videos, lipgloss.NewStyle(), lipgloss.NewStyle(), lipgloss.NewStyle())

	// First item
	item := l.SelectedItem()
	if item == nil || item.ID != "1" {
		t.Errorf("SelectedItem() = %v, want Video 1", item)
	}

	// Move cursor
	l.CursorDown()
	item = l.SelectedItem()
	if item == nil || item.ID != "2" {
		t.Errorf("SelectedItem() = %v, want Video 2", item)
	}

	// Empty list
	emptyList := NewList([]yt.Video{}, lipgloss.NewStyle(), lipgloss.NewStyle(), lipgloss.NewStyle())
	item = emptyList.SelectedItem()
	if item != nil {
		t.Errorf("SelectedItem() on empty list = %v, want nil", item)
	}
}

func TestListSetItems(t *testing.T) {
	videos := []yt.Video{
		{ID: "1", Title: "Video 1"},
	}

	l := NewList(videos, lipgloss.NewStyle(), lipgloss.NewStyle(), lipgloss.NewStyle())
	l.CursorDown()

	// SetItems should reset cursor
	newVideos := []yt.Video{
		{ID: "2", Title: "Video 2"},
		{ID: "3", Title: "Video 3"},
	}
	l.SetItems(newVideos)

	if l.Cursor() != 0 {
		t.Errorf("SetItems() cursor = %d, want 0", l.Cursor())
	}
	if l.Length() != 2 {
		t.Errorf("SetItems() length = %d, want 2", l.Length())
	}
}

func TestListPagination(t *testing.T) {
	videos := make([]yt.Video, 25)
	for i := range videos {
		videos[i] = yt.Video{ID: string(rune(i)), Title: "Video"}
	}

	l := NewList(videos, lipgloss.NewStyle(), lipgloss.NewStyle(), lipgloss.NewStyle())
	l.SetDimensions(80, 24)

	// Each item takes 2 lines (title + meta), so visible = (height - 2) / 2
	visible := l.VisibleCount()
	if visible != 11 {
		t.Errorf("VisibleCount() = %d, want 11", visible)
	}

	// Current page at start
	if l.CurrentPage() != 1 {
		t.Errorf("CurrentPage() = %d, want 1", l.CurrentPage())
	}

	// Total pages: ceil(25/11) = 3
	totalPages := l.TotalPages()
	if totalPages != 3 {
		t.Errorf("TotalPages() = %d, want 3", totalPages)
	}
}

func TestListViewEmpty(t *testing.T) {
	l := NewList([]yt.Video{}, lipgloss.NewStyle(), lipgloss.NewStyle(), lipgloss.NewStyle())
	view := l.View()

	if view == "" {
		t.Error("View() on empty list should not be empty string")
	}
}
