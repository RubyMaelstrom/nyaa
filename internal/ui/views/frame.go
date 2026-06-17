package views

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/user/nyaa-tui/internal/ui/theme"
)

// Shared full-screen layout used by every browsing window so they resize
// identically: the box border always sits at the terminal edges, and the body
// is sized to exactly fill the space between an optional title and footer. The
// caller's bodyFn receives the usable inner dimensions and must return at most
// innerH lines; Place pads (or the caller scrolls) so nothing overflows.

const (
	frameMinWidth  = 20
	frameMinHeight = 8
)

// frameInnerSize returns the usable content width/height inside the box for a
// given terminal size and chrome (title/footer heights plus blank separators).
func frameInnerSize(width, height, chrome int) (innerW, innerH int) {
	if width < frameMinWidth {
		width = frameMinWidth
	}
	if height < frameMinHeight {
		height = frameMinHeight
	}
	boxOuterH := height - chrome
	if boxOuterH < 3 {
		boxOuterH = 3
	}
	innerW = width - theme.Theme.BoxStyle.GetHorizontalFrameSize()
	innerH = boxOuterH - theme.Theme.BoxStyle.GetVerticalFrameSize()
	if innerW < 1 {
		innerW = 1
	}
	if innerH < 1 {
		innerH = 1
	}
	return innerW, innerH
}

// blockHeight reports the rendered line count of s, treating "" as zero lines.
func blockHeight(s string) int {
	if s == "" {
		return 0
	}
	return lipgloss.Height(s)
}

// Frame renders title (optional, already styled), a bordered body filling the
// terminal, and footer (optional, already styled). hAlign/vAlign position the
// body within the inner area — Left/Top for scrolling lists, Center/Center for
// small prompts. bodyFn builds the body given the exact inner dimensions.
func Frame(width, height int, title, footer string, hAlign, vAlign lipgloss.Position, bodyFn func(innerW, innerH int) string) string {
	if width < frameMinWidth {
		width = frameMinWidth
	}
	if height < frameMinHeight {
		height = frameMinHeight
	}

	titleH := blockHeight(title)
	footerH := blockHeight(footer)

	// chrome = title + blank separator + box(0, sized below) + blank + footer.
	chrome := titleH + footerH
	if titleH > 0 {
		chrome++ // blank line under the title
	}
	if footerH > 0 {
		chrome++ // blank line above the footer
	}

	innerW, innerH := frameInnerSize(width, height, chrome)
	boxOuterH := innerH + theme.Theme.BoxStyle.GetVerticalFrameSize()

	var body string
	if bodyFn != nil {
		body = bodyFn(innerW, innerH)
	}
	// Clamp every region to its column budget so a long title, hint, or item can
	// never spill past the terminal edge on a narrow window.
	body = truncateToWidth(body, innerW)
	placed := lipgloss.Place(innerW, innerH, hAlign, vAlign, body)

	box := theme.Theme.BoxStyle.
		Width(width - 2).        // border (1 each side) lands at the terminal edges
		Height(boxOuterH - 2).   // border (top+bottom) lands at the reserved rows
		Render(placed)

	parts := make([]string, 0, 5)
	if titleH > 0 {
		parts = append(parts, truncateToWidth(title, width), "")
	}
	parts = append(parts, box)
	if footerH > 0 {
		parts = append(parts, "", truncateToWidth(footer, width))
	}
	return lipgloss.JoinVertical(lipgloss.Center, parts...)
}

// truncateToWidth trims each line of s to width columns (ANSI-aware).
func truncateToWidth(s string, width int) string {
	if s == "" || width <= 0 {
		return s
	}
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = ansi.Truncate(lines[i], width, "…")
	}
	return strings.Join(lines, "\n")
}

// windowedLines scrolls a fitted height×width window over rows, keeping
// cursorRow visible, and truncates each shown row to width so long titles never
// wrap and push the layout past the terminal edge.
func windowedLines(rows []string, cursorRow, width, height int) string {
	if len(rows) == 0 {
		return ""
	}
	start := windowStart(cursorRow, len(rows), height)
	end := start + height
	if end > len(rows) {
		end = len(rows)
	}
	out := make([]string, 0, end-start)
	for _, r := range rows[start:end] {
		out = append(out, ansi.Truncate(r, width, "…"))
	}
	return strings.Join(out, "\n")
}

// windowStart returns the first index of a size-tall window over total items
// that keeps cursor visible, biased to center the cursor. Preserves the
// familiar scrolling feel while guaranteeing the window never overflows.
func windowStart(cursor, total, size int) int {
	if total <= size || size <= 0 {
		return 0
	}
	start := cursor - size/2
	if start < 0 {
		start = 0
	}
	if start > total-size {
		start = total - size
	}
	return start
}
