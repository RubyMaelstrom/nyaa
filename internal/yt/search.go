package yt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

type SearchResult struct {
	Videos      []Video
	Partial     bool
	PartialCount int
}

func Search(query string, count int) (SearchResult, error) {
	return searchWithRetry(query, count, 3)
}

func searchWithRetry(query string, count int, maxRetries int) (SearchResult, error) {
	backoff := []time.Duration{2 * time.Second, 4 * time.Second, 8 * time.Second}
	timeouts := []time.Duration{15 * time.Second, 30 * time.Second, 60 * time.Second}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		timeout := 60 * time.Second
		if attempt < len(timeouts) {
			timeout = timeouts[attempt]
		}

		results, err := runYtDlpSearch(query, count, timeout)
		if err == nil {
			return results, nil
		}

		if results.Partial && attempt < maxRetries {
			if IsRetryable(err) {
				time.Sleep(backoff[attempt])
				continue
			}
			return results, nil
		}

		if IsRetryable(err) && attempt < maxRetries {
			time.Sleep(backoff[attempt])
			continue
		}

		return results, err
	}

	return SearchResult{}, fmt.Errorf("all retries exhausted (qwq)")
}

func runYtDlpSearch(query string, count int, timeout time.Duration) (SearchResult, error) {
	searchQuery := fmt.Sprintf("ytsearch%d:%s", count, query)

	args := []string{
		searchQuery,
		"--flat-playlist",
		"--print-json",
		"--no-download",
		"--socket-timeout", "15",
		"--retries", "3",
		"--fragment-retries", "3",
		"--no-check-certificates",
		"--no-warnings",
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "yt-dlp", args...)
	cmd.Stdin = nil

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	stdoutBytes := stdout.Bytes()
	if len(bytes.TrimSpace(stdoutBytes)) > 0 {
		videos, parseErr := parseJSONOutput(stdoutBytes)
		if len(videos) > 0 {
			return SearchResult{
				Videos:      videos,
				Partial:     parseErr != nil || err != nil,
				PartialCount: len(videos),
			}, nil
		}
	}

	if err != nil {
		return SearchResult{}, ClassifyError(stderr.String(), cmd.ProcessState.ExitCode(), err)
	}

	if strings.TrimSpace(stdout.String()) == "" {
		return SearchResult{}, ClassifyError("", 0, nil)
	}

	videos, parseErr := parseJSONOutput(stdoutBytes)
	if parseErr != nil {
		return SearchResult{}, ClassifyError(parseErr.Error(), 0, parseErr)
	}

	return SearchResult{Videos: videos}, nil
}

type ytDlpEntry struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Channel    string   `json:"channel"`
	ChannelID  string   `json:"channel_id"`
	Duration   *float64 `json:"duration"`
	ViewCount  *int     `json:"view_count"`
	UploadDate string   `json:"upload_date"`
	URL        string   `json:"url"`
	WebpageURL string   `json:"webpage_url"`
}

func parseJSONOutput(data []byte) ([]Video, error) {
	lines := bytes.Split(data, []byte("\n"))
	var videos []Video

	for _, line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		var entry ytDlpEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}

		if entry.ID == "" || entry.Title == "" {
			continue
		}

		var durationStr string
		if entry.Duration != nil {
			durationStr = formatDuration(*entry.Duration)
		} else {
			durationStr = "??:??"
		}

		var viewCountStr string
		if entry.ViewCount != nil {
			viewCountStr = formatViewCount(*entry.ViewCount)
		} else {
			viewCountStr = "0"
		}

		video := Video{
			ID:         entry.ID,
			Title:      entry.Title,
			Channel:    entry.Channel,
			ChannelID:  entry.ChannelID,
			Duration:   durationStr,
			ViewCount:  viewCountStr,
			UploadDate: formatDate(entry.UploadDate),
			URL:        entry.WebpageURL,
		}
		if video.URL == "" {
			video.URL = fmt.Sprintf("https://www.youtube.com/watch?v=%s", entry.ID)
		}
		videos = append(videos, video)
	}

	return videos, nil
}

func formatDuration(seconds float64) string {
	if seconds <= 0 {
		return "??:??"
	}
	hours := int(seconds) / 3600
	minutes := (int(seconds) % 3600) / 60
	secs := int(seconds) % 60

	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, secs)
	}
	return fmt.Sprintf("%d:%02d", minutes, secs)
}

func formatViewCount(count int) string {
	if count >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(count)/1000000)
	}
	if count >= 1000 {
		return fmt.Sprintf("%.1fK", float64(count)/1000)
	}
	return fmt.Sprintf("%d", count)
}

func formatDate(date string) string {
	if len(date) != 8 {
		return date
	}
	t, err := time.Parse("20060102", date)
	if err != nil {
		return date
	}
	return t.Format("Jan 2, 2006")
}

func CheckVersion() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "yt-dlp", "--version")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to check yt-dlp version: %w", err)
	}

	version := strings.TrimSpace(stdout.String())
	if version == "" {
		return "", fmt.Errorf("yt-dlp version output was empty")
	}

	return version, nil
}

func UpdateYtDlp() error {
	cmd := exec.Command("yt-dlp", "-U")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
