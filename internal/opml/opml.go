// Package opml parses OPML subscription exports (e.g. YouTube's
// "Takeout/subscriptions.opml") into channel subscriptions.
package opml

import (
	"encoding/xml"
	"fmt"
	"net/url"
)

// Subscription is one channel discovered in an OPML file.
type Subscription struct {
	ChannelID string
	Name      string
	RSSURL    string
}

type outline struct {
	Title    string    `xml:"title,attr"`
	Text     string    `xml:"text,attr"`
	XMLURL   string    `xml:"xmlUrl,attr"`
	Children []outline `xml:"outline"`
}

type document struct {
	XMLName xml.Name  `xml:"opml"`
	Body    struct {
		Outlines []outline `xml:"outline"`
	} `xml:"body"`
}

// Parse reads OPML data and returns the YouTube channel subscriptions it
// contains. Outlines may be nested under folders (YouTube nests them under a
// "YouTube Subscriptions" outline), so the tree is walked recursively.
// Duplicate channel IDs are returned only once.
func Parse(data []byte) ([]Subscription, error) {
	var doc document
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse OPML: %w", err)
	}

	var subs []Subscription
	seen := make(map[string]bool)

	var walk func(o outline)
	walk = func(o outline) {
		if id := channelIDFromURL(o.XMLURL); id != "" && !seen[id] {
			name := o.Title
			if name == "" {
				name = o.Text
			}
			subs = append(subs, Subscription{ChannelID: id, Name: name, RSSURL: o.XMLURL})
			seen[id] = true
		}
		for _, child := range o.Children {
			walk(child)
		}
	}
	for _, o := range doc.Body.Outlines {
		walk(o)
	}
	return subs, nil
}

// channelIDFromURL extracts the channel_id query parameter from a YouTube feed
// URL, returning "" if the URL isn't a recognizable channel feed.
func channelIDFromURL(raw string) string {
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return u.Query().Get("channel_id")
}
