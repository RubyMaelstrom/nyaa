package player

import (
	"strings"
	"testing"
)

func TestClassifyMpvError(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantContain string
	}{
		{
			name:        "nil error",
			err:         nil,
			wantContain: "",
		},
		{
			name:        "executable not found",
			err:         &execError{msg: "exec: \"mpv\": executable file not found in $PATH"},
			wantContain: "mpv not found",
		},
		{
			name:        "permission denied",
			err:         &execError{msg: "permission denied"},
			wantContain: "Permission denied",
		},
		{
			name:        "invalid url",
			err:         &execError{msg: "no such file or invalid url"},
			wantContain: "Invalid video URL",
		},
		{
			name:        "unknown error",
			err:         &execError{msg: "something went wrong"},
			wantContain: "Playback ended unexpectedly",
		},
		{
			name:        "mpv stream open failure",
			err:         &execError{msg: "exit status 2"},
			wantContain: "throttling",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ClassifyError(tt.err)
			if tt.err == nil {
				if err != nil {
					t.Errorf("ClassifyError(nil) = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("ClassifyError() returned nil, want error containing %q", tt.wantContain)
			}
			if !strings.Contains(err.Error(), tt.wantContain) {
				t.Errorf("ClassifyError() = %v, want to contain %q", err.Error(), tt.wantContain)
			}
		})
	}
}

func TestMpvArgsForeground(t *testing.T) {
	// Foreground (bare TTY): mpv owns the terminal, so it must NOT be muted with
	// --no-terminal or the keyboard dies.
	video := mpvArgs("http://x/v", false, false)
	if !contains(video, "--force-window") || contains(video, "--no-video") {
		t.Errorf("video args = %v, want --force-window and no --no-video", video)
	}
	if video[len(video)-1] != "http://x/v" {
		t.Errorf("url should be last arg, got %v", video)
	}

	// Audio-only is windowless; controlled purely from the terminal.
	audio := mpvArgs("http://x/v", true, false)
	if !contains(audio, "--no-video") || contains(audio, "--force-window") {
		t.Errorf("audio args = %v, want --no-video and no --force-window", audio)
	}

	for _, audioOnly := range []bool{false, true} {
		if contains(mpvArgs("u", audioOnly, false), "--no-terminal") {
			t.Errorf("foreground mpvArgs(audioOnly=%v) must not contain --no-terminal", audioOnly)
		}
	}
}

func TestMpvArgsBackground(t *testing.T) {
	// Background (GUI): mpv runs in its own window while the TUI stays up, so it
	// must stay off the terminal and always have a window to capture keys.
	for _, audioOnly := range []bool{false, true} {
		args := mpvArgs("http://x/v", audioOnly, true)
		if !contains(args, "--no-terminal") {
			t.Errorf("background mpvArgs(audioOnly=%v) = %v, want --no-terminal", audioOnly, args)
		}
		if !contains(args, "--force-window") {
			t.Errorf("background mpvArgs(audioOnly=%v) = %v, want --force-window", audioOnly, args)
		}
	}
	if !contains(mpvArgs("u", true, true), "--no-video") {
		t.Error("background audio-only should still pass --no-video")
	}
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

type execError struct {
	msg string
}

func (e *execError) Error() string {
	return e.msg
}
