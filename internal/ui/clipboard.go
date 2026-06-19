package ui

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
)

// currentVideoURL returns the link of whatever video is highlighted in the
// active view, or "" if the current screen has no video under the cursor.
func (m Model) currentVideoURL() string {
	switch m.state {
	case StateResults:
		if v := m.results.SelectedVideo(); v != nil {
			return v.URL
		}
	case StateChannel:
		if it := m.channelBrowse.SelectedItem(); it != nil {
			return it.URL
		}
	case StateSubscription:
		return m.subscriptionsView.SelectedVideoURL()
	case StatePlaying:
		if m.nowPlaying != nil {
			return m.nowPlaying.URL
		}
	}
	return ""
}

// copyCurrentLink copies the highlighted video's URL to the system clipboard and
// reports the outcome via a toast. It returns a nil cmd when nothing playable is
// selected, so the caller can fall through to the screen's own `c` binding (the
// error view uses `c` for "show cached results", and the menu has no link).
func (m *Model) copyCurrentLink() tea.Cmd {
	url := m.currentVideoURL()
	if url == "" {
		return nil
	}
	if err := copyToClipboard(url); err != nil {
		return m.showToast("couldn't copy link~ install wl-clipboard or xclip (｡•́︿•̀｡)")
	}
	return m.showToast("link copied to clipboard! (≧◡≦) ♡")
}

var errNoClipboardTool = errors.New("no clipboard utility found (wl-copy/xclip/xsel)")

// copyToClipboard writes text to the system clipboard by shelling out to the
// first available copy tool. Wayland (the common case here) is tried first, then
// X11. Each tool is fully detached from nyaa's terminal so a clipboard daemon
// like wl-copy outlives the app and never holds the alt-screen fd open — the
// thing that made copying flaky when run from inside the TUI.
func copyToClipboard(text string) error {
	type tool struct {
		name string
		args []string
	}

	var tools []tool
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		tools = append(tools, tool{"wl-copy", nil})
	}
	tools = append(tools,
		tool{"xclip", []string{"-selection", "clipboard"}},
		tool{"xsel", []string{"--input", "--clipboard"}},
	)

	var firstErr error
	for _, t := range tools {
		path, err := exec.LookPath(t.name)
		if err != nil {
			continue // tool not installed; try the next one
		}
		if err := runCopyTool(path, t.args, text); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		return nil
	}
	if firstErr != nil {
		return firstErr
	}
	return errNoClipboardTool
}

// runCopyTool feeds text to a copy command's stdin and detaches it from the TUI:
// its own session (so it survives nyaa exiting and isn't tied to the terminal)
// and discarded stdout/stderr (so a lingering daemon never scribbles on the
// alt-screen).
func runCopyTool(path string, args []string, text string) error {
	cmd := exec.Command(path, args...)
	cmd.Stdin = strings.NewReader(text)
	if devnull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0); err == nil {
		defer devnull.Close()
		cmd.Stdout = devnull
		cmd.Stderr = devnull
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	return cmd.Run()
}
