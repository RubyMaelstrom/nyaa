package yt

import (
	"errors"
	"fmt"
	"strings"
)

type ErrorKind int

const (
	ErrorKindRateLimited ErrorKind = iota
	ErrorKindOutdated
	ErrorKindAuthRequired
	ErrorKindGeoBlocked
	ErrorKindAgeRestricted
	ErrorKindNetwork
	ErrorKindParse
	ErrorKindEmptyResults
	ErrorKindUnknown
)

type YtDlpError struct {
	Kind        ErrorKind
	Message     string
	Suggestion  string
	ShowUpdate  bool
	Err         error
}

func (e *YtDlpError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Err.Error()
}

func (e *YtDlpError) Unwrap() error {
	return e.Err
}

func ClassifyError(stderr string, exitCode int, err error) error {
	lower := strings.ToLower(stderr)

	// Check for empty results first (exit code 0 but no output)
	if exitCode == 0 && (lower == "" || strings.TrimSpace(lower) == "") && err == nil {
		return &YtDlpError{
			Kind:       ErrorKindEmptyResults,
			Message:    "No videos found... try a different search? (·_·;)",
			Suggestion: "Try using different keywords or check your spelling",
			Err:        err,
		}
	}

	if err == nil && exitCode == 0 {
		return nil
	}

	switch {
	case strings.Contains(lower, "429") || strings.Contains(lower, "too many requests"):
		return &YtDlpError{
			Kind:       ErrorKindRateLimited,
			Message:    "YouTube says slow down... retrying in a moment~ (◕﹏◕✿)",
			Suggestion: "Try again in a few seconds or change your search query",
			Err:        err,
		}

	case strings.Contains(lower, "signature") || strings.Contains(lower, "cipher") || strings.Contains(lower, "extractor"):
		return &YtDlpError{
			Kind:       ErrorKindOutdated,
			Message:    "yt-dlp needs an update! (＠_＠;)",
			Suggestion: "Run: yt-dlp -U",
			ShowUpdate: true,
			Err:        err,
		}

	case strings.Contains(lower, "cookie") || strings.Contains(lower, "login") || strings.Contains(lower, "consent"):
		return &YtDlpError{
			Kind:       ErrorKindAuthRequired,
			Message:    "Try adding cookies: --cookies-from-browser (◔_◔)",
			Suggestion: "Use --cookies-from-browser or provide a cookie file",
			Err:        err,
		}

	case strings.Contains(lower, "geo") || strings.Contains(lower, "not available") || strings.Contains(lower, "region"):
		return &YtDlpError{
			Kind:       ErrorKindGeoBlocked,
			Message:    "This video isn't available in your region~ (╥_╥)",
			Suggestion: "Try using a VPN or proxy",
			Err:        err,
		}

	case strings.Contains(lower, "age") || strings.Contains(lower, "confirm your age"):
		return &YtDlpError{
			Kind:       ErrorKindAgeRestricted,
			Message:    "Age-restricted video. Try adding cookies~ (◔_◔)",
			Suggestion: "Authenticate with --cookies-from-browser to access age-restricted content",
			Err:        err,
		}

	case strings.Contains(lower, "connection refused") || strings.Contains(lower, "timeout") || strings.Contains(lower, "network"):
		return &YtDlpError{
			Kind:       ErrorKindNetwork,
			Message:    "Connection failed... check your internet~ (T_T)",
			Suggestion: "Check your network connection and try again",
			Err:        err,
		}

	case strings.Contains(lower, "unable to extract"):
		return &YtDlpError{
			Kind:       ErrorKindParse,
			Message:    "Oops, couldn't parse results... (＠_＠;)",
			Suggestion: "YouTube may have changed their layout. Try updating yt-dlp",
			ShowUpdate: true,
			Err:        err,
		}

	default:
		if err != nil {
			return &YtDlpError{
				Kind:       ErrorKindUnknown,
				Message:    fmt.Sprintf("Something went wrong... (qwq)\n%s", stderr),
				Suggestion: "Try again or check if yt-dlp is up to date",
				Err:        err,
			}
		}
		return err
	}
}

func IsRetryable(err error) bool {
	var ytErr *YtDlpError
	if errors.As(err, &ytErr) {
		switch ytErr.Kind {
		case ErrorKindRateLimited, ErrorKindNetwork, ErrorKindEmptyResults:
			return true
		}
	}
	return false
}
