package ui

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

// guiAvailable reports whether a graphical session is present. In a GUI, mpv can
// open its own window (which the WM gives keyboard focus), so the TUI can stay up
// during playback. On a bare TTY there is no window, so we must hand the terminal
// to mpv instead. (nyaa targets Linux, so DISPLAY/WAYLAND_DISPLAY is the signal.)
func guiAvailable() bool {
	return os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""
}

// mpvExec adapts an mpv *exec.Cmd to tea.ExecCommand so it can be run via
// tea.Exec, which suspends the TUI and connects mpv to the terminal. It prints a
// cute "now playing" banner first, so the user knows what's happening when the
// TUI drops away on a bare TTY.
type mpvExec struct {
	cmd    *exec.Cmd
	banner string
	stdout io.Writer
}

func newMpvExec(cmd *exec.Cmd, banner string) *mpvExec {
	return &mpvExec{cmd: cmd, banner: banner}
}

func (e *mpvExec) SetStdin(r io.Reader)  { e.cmd.Stdin = r }
func (e *mpvExec) SetStdout(w io.Writer) { e.cmd.Stdout = w; e.stdout = w }
func (e *mpvExec) SetStderr(w io.Writer) { e.cmd.Stderr = w }

func (e *mpvExec) Run() error {
	if e.stdout != nil && e.banner != "" {
		fmt.Fprint(e.stdout, e.banner)
	}
	return e.cmd.Run()
}
