package yt

import (
	"fmt"
	"strings"
)

type Video struct {
	ID         string
	Title      string
	Channel    string
	ChannelID  string
	Duration   string
	ViewCount  string
	UploadDate string
	URL        string
}

func (v Video) DisplayTitle() string {
	return v.Title
}

// DisplayMeta joins only the fields we actually have, so search results (which
// lack an upload date under --flat-playlist) don't trail a dangling " • ".
func (v Video) DisplayMeta() string {
	var parts []string
	if v.Channel != "" {
		parts = append(parts, v.Channel)
	}
	if v.Duration != "" {
		parts = append(parts, v.Duration)
	}
	if v.ViewCount != "" {
		parts = append(parts, v.ViewCount+" views")
	}
	if v.UploadDate != "" {
		parts = append(parts, v.UploadDate)
	}
	return strings.Join(parts, "  •  ")
}

func (v Video) String() string {
	return fmt.Sprintf("%s [%s] - %s", v.Title, v.Channel, v.URL)
}
