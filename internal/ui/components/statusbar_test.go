package components

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestNewStatusBar(t *testing.T) {
	sb := NewStatusBar(1, 3, lipgloss.NewStyle())

	if sb.page != 1 {
		t.Errorf("NewStatusBar() page = %d, want 1", sb.page)
	}
	if sb.total != 3 {
		t.Errorf("NewStatusBar() total = %d, want 3", sb.total)
	}
}

func TestStatusBarSetPage(t *testing.T) {
	sb := NewStatusBar(1, 3, lipgloss.NewStyle())
	sb.SetPage(2, 3)

	if sb.page != 2 {
		t.Errorf("SetPage() page = %d, want 2", sb.page)
	}
	if sb.total != 3 {
		t.Errorf("SetPage() total = %d, want 3", sb.total)
	}
}

func TestStatusBarView(t *testing.T) {
	sb := NewStatusBar(1, 3, lipgloss.NewStyle())
	view := sb.View()

	if !strings.Contains(view, "page 1/3") {
		t.Errorf("View() should contain page info, got %v", view)
	}
	if !strings.Contains(view, "navigate") {
		t.Errorf("View() should contain navigate hint, got %v", view)
	}
}

func TestStatusBarViewWithWidth(t *testing.T) {
	sb := NewStatusBar(1, 1, lipgloss.NewStyle())
	sb.SetWidth(80)
	view := sb.View()

	// View should not be empty
	if view == "" {
		t.Error("View() with width should not be empty")
	}
}
