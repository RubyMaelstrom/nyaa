package player

import (
	"fmt"
	"os/exec"
	"strings"
)

// Command builds the mpv invocation for a URL.
//
// background distinguishes the two playback strategies the UI uses:
//
//   - background=true  (GUI): the TUI stays up and mpv runs in its own window,
//     which the window manager gives keyboard focus. mpv is kept off the terminal
//     (--no-terminal) so it can't corrupt the live TUI, and a window is forced so
//     there's always a surface to capture keypresses.
//   - background=false (bare TTY): there is no GUI window, so the caller hands the
//     terminal to mpv via tea.Exec. mpv must therefore read the terminal (no
//     --no-terminal) to get keyboard control (q / volume / seek).
//
// It only sets arguments; stdio is left for the caller / tea.Exec to wire up.
func Command(url string, audioOnly, background bool) *exec.Cmd {
	return exec.Command("mpv", mpvArgs(url, audioOnly, background)...)
}

func mpvArgs(url string, audioOnly, background bool) []string {
	// Let mpv show its on-screen controller and OSD bar (the default). We used
	// to pass --no-osc/--no-osd-bar to keep playback chrome-free, but that hid
	// the seek/volume/play controls people reach for with the mouse.
	var args []string
	switch {
	case background:
		// Stay off the terminal; rely on the (forced) window for input.
		args = append(args, "--no-terminal", "--force-window")
		if audioOnly {
			args = append(args, "--no-video")
		}
	case audioOnly:
		// Windowless: controlled purely from the handed-over terminal.
		args = append(args, "--no-video")
	default:
		args = append(args, "--force-window")
	}
	return append(args, url)
}

// ClassifyError maps an mpv launch/exit error to a friendly message.
func ClassifyError(err error) error {
	if err == nil {
		return nil
	}

	errMsg := err.Error()
	lower := strings.ToLower(errMsg)

	switch {
	case strings.Contains(lower, "executable"):
		return fmt.Errorf("mpv not found! Can't play videos without it~ (>_<)")
	case strings.Contains(lower, "permission"):
		return fmt.Errorf("Permission denied running mpv~ (T_T)")
	case strings.Contains(lower, "no such file") || strings.Contains(lower, "invalid url"):
		return fmt.Errorf("Invalid video URL... (・_・;)")
	default:
		return fmt.Errorf("Playback ended unexpectedly... (qwq)\nError: %v", err)
	}
}
