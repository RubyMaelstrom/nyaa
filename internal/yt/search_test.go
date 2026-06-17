package yt

import (
	"strings"
	"testing"
)

func TestParseJSONOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantLen  int
		wantErr  bool
	}{
		{
			name: "single valid video",
			input: `{"id":"abc123","title":"Test Video","channel":"Test Channel","duration":120,"view_count":1000,"upload_date":"20240101","webpage_url":"https://www.youtube.com/watch?v=abc123"}`,
			wantLen: 1,
			wantErr: false,
		},
		{
			name: "multiple valid videos",
			input: `{"id":"abc123","title":"Video 1","channel":"Channel 1","duration":60,"view_count":100,"upload_date":"20240101","webpage_url":"https://www.youtube.com/watch?v=abc123"}
{"id":"def456","title":"Video 2","channel":"Channel 2","duration":180,"view_count":200,"upload_date":"20240102","webpage_url":"https://www.youtube.com/watch?v=def456"}`,
			wantLen: 2,
			wantErr: false,
		},
		{
			name:     "empty input",
			input:    "",
			wantLen:  0,
			wantErr:  false,
		},
		{
			name:     "invalid JSON",
			input:    "not json at all",
			wantLen:  0,
			wantErr:  false, // parse skips invalid lines silently
		},
		{
			name:     "mixed valid and invalid",
			input:    "garbage\n{\"id\":\"abc\",\"title\":\"Valid\",\"channel\":\"Ch\",\"duration\":30,\"view_count\":10,\"upload_date\":\"20240101\",\"webpage_url\":\"https://youtube.com/watch?v=abc\"}\nmore garbage",
			wantLen:  1,
			wantErr:  false,
		},
		{
			name:     "missing required fields",
			input:    `{"id":"abc123"}`,
			wantLen:  0,
			wantErr:  false, // skips entries without title
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			videos, err := parseJSONOutput([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("parseJSONOutput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(videos) != tt.wantLen {
				t.Errorf("parseJSONOutput() got %d videos, want %d", len(videos), tt.wantLen)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		seconds float64
		want    string
	}{
		{0, "??:??"},
		{-1, "??:??"},
		{30, "0:30"},
		{60, "1:00"},
		{125, "2:05"},
		{3661, "1:01:01"},
		{7265, "2:01:05"},
	}

	for _, tt := range tests {
		got := formatDuration(tt.seconds)
		if got != tt.want {
			t.Errorf("formatDuration(%v) = %v, want %v", tt.seconds, got, tt.want)
		}
	}
}

func TestFormatViewCount(t *testing.T) {
	tests := []struct {
		count int
		want  string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1.0K"},
		{1500, "1.5K"},
		{999999, "1000.0K"},
		{1000000, "1.0M"},
		{1500000, "1.5M"},
		{12345678, "12.3M"},
	}

	for _, tt := range tests {
		got := formatViewCount(tt.count)
		if got != tt.want {
			t.Errorf("formatViewCount(%d) = %v, want %v", tt.count, got, tt.want)
		}
	}
}

func TestFormatDate(t *testing.T) {
	tests := []struct {
		date string
		want string
	}{
		{"20240101", "Jan 1, 2024"},
		{"20241225", "Dec 25, 2024"},
		{"invalid", "invalid"},
		{"", ""},
		{"2024", "2024"},
	}

	for _, tt := range tests {
		got := formatDate(tt.date)
		if got != tt.want {
			t.Errorf("formatDate(%q) = %v, want %v", tt.date, got, tt.want)
		}
	}
}

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name     string
		stderr   string
		exitCode int
		wantKind ErrorKind
	}{
		{
			name:     "rate limited",
			stderr:   "ERROR: HTTP Error 429: Too Many Requests",
			exitCode: 1,
			wantKind: ErrorKindRateLimited,
		},
		{
			name:     "signature error",
			stderr:   "ERROR: Unable to extract signature",
			exitCode: 1,
			wantKind: ErrorKindOutdated,
		},
		{
			name:     "cookie required",
			stderr:   "ERROR: Please use --cookies-from-browser or specify a cookie file",
			exitCode: 1,
			wantKind: ErrorKindAuthRequired,
		},
		{
			name:     "geo blocked",
			stderr:   "ERROR: This video is not available in your region",
			exitCode: 1,
			wantKind: ErrorKindGeoBlocked,
		},
		{
			name:     "age restricted",
			stderr:   "ERROR: Confirm your age",
			exitCode: 1,
			wantKind: ErrorKindAgeRestricted,
		},
		{
			name:     "network error",
			stderr:   "ERROR: Connection refused",
			exitCode: 1,
			wantKind: ErrorKindNetwork,
		},
		{
			name:     "parse error",
			stderr:   "ERROR: Unable to extract initial data",
			exitCode: 1,
			wantKind: ErrorKindParse,
		},
		{
			name:     "empty results",
			stderr:   "",
			exitCode: 0,
			wantKind: ErrorKindEmptyResults,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ClassifyError(tt.stderr, tt.exitCode, nil)
			if err == nil {
				t.Fatalf("ClassifyError() returned nil error")
			}
			ytErr, ok := err.(*YtDlpError)
			if !ok {
				t.Fatalf("ClassifyError() returned non-YtDlpError: %T", err)
			}
			if ytErr.Kind != tt.wantKind {
				t.Errorf("ClassifyError() kind = %v, want %v", ytErr.Kind, tt.wantKind)
			}
		})
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "rate limited is retryable",
			err: &YtDlpError{
				Kind: ErrorKindRateLimited,
			},
			want: true,
		},
		{
			name: "network is retryable",
			err: &YtDlpError{
				Kind: ErrorKindNetwork,
			},
			want: true,
		},
		{
			name: "empty results is retryable",
			err: &YtDlpError{
				Kind: ErrorKindEmptyResults,
			},
			want: true,
		},
		{
			name: "outdated is not retryable",
			err: &YtDlpError{
				Kind: ErrorKindOutdated,
			},
			want: false,
		},
		{
			name: "geo blocked is not retryable",
			err: &YtDlpError{
				Kind: ErrorKindGeoBlocked,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRetryable(tt.err)
			if got != tt.want {
				t.Errorf("IsRetryable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVideoDisplayTitle(t *testing.T) {
	v := Video{
		ID:    "abc123",
		Title: "Test Video Title",
	}
	if v.DisplayTitle() != "Test Video Title" {
		t.Errorf("DisplayTitle() = %v, want %v", v.DisplayTitle(), "Test Video Title")
	}
}

func TestVideoDisplayMeta(t *testing.T) {
	v := Video{
		Channel:    "Test Channel",
		Duration:   "3:45",
		ViewCount:  "1.5K",
		UploadDate: "Jan 1, 2024",
	}
	meta := v.DisplayMeta()
	if !strings.Contains(meta, "Test Channel") {
		t.Errorf("DisplayMeta() should contain channel, got %v", meta)
	}
	if !strings.Contains(meta, "3:45") {
		t.Errorf("DisplayMeta() should contain duration, got %v", meta)
	}
	if !strings.Contains(meta, "1.5K") {
		t.Errorf("DisplayMeta() should contain view count, got %v", meta)
	}
}
