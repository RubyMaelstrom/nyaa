package rss

import (
	"crypto/tls"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

type AtomFeed struct {
	XMLName xml.Name `xml:"http://www.w3.org/2005/Atom feed"`
	Entries []AtomEntry `xml:"http://www.w3.org/2005/Atom entry"`
}

type AtomEntry struct {
	ID        string `xml:"http://www.w3.org/2005/Atom id"`
	Title     string `xml:"http://www.w3.org/2005/Atom title"`
	Published string `xml:"http://www.w3.org/2005/Atom published"`
	Link      []Link `xml:"http://www.w3.org/2005/Atom link"`
}

type Link struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
}

type RSSItem struct {
	ID           string
	Title        string
	URL          string
	PublishedAt  time.Time
	Description  string
}

// FeedError describes a failed feed fetch. Throttled marks the transient cases
// worth retrying — the 404/429/5xx statuses YouTube's RSS backend hands back
// when it's load-shedding, plus network errors — as opposed to a genuinely dead
// or malformed feed.
type FeedError struct {
	StatusCode int
	Throttled  bool
	Err        error
}

func (e *FeedError) Error() string {
	if e.StatusCode != 0 {
		return fmt.Sprintf("feed request failed with status %d", e.StatusCode)
	}
	return e.Err.Error()
}

func (e *FeedError) Unwrap() error { return e.Err }

// IsRetryable reports whether a fetch error is a transient throttle/load-shed
// (so the caller should retry on a fresh connection) rather than a permanent
// failure like malformed XML.
func IsRetryable(err error) bool {
	var fe *FeedError
	if errors.As(err, &fe) {
		return fe.Throttled
	}
	return false
}

// isThrottleStatus classifies the HTTP statuses YouTube's "RSS Feeds server"
// returns while shedding load. It answers a throttled request with 404 or 5xx
// (not a clean 429), so for this endpoint a 404 means "can't serve right now",
// not "channel gone".
func isThrottleStatus(code int) bool {
	return code == http.StatusNotFound || code == http.StatusTooManyRequests || code >= 500
}

// freshClient returns a client whose every request opens a new connection.
// YouTube's feed endpoint shards channels across backends and coalesces HTTP/2
// requests onto a single connection; reusing that pinned connection makes a
// retry hit the same (possibly shedding) backend over and over. Forcing a fresh
// connection lets each retry land on a different backend — measurably the single
// biggest lever for recovering throttled feeds.
func freshClient() *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			DisableKeepAlives: true,
			ForceAttemptHTTP2: true,
			TLSClientConfig:   &tls.Config{},
		},
	}
}

func FetchFeed(url string) ([]RSSItem, error) {
	resp, err := freshClient().Get(url)
	if err != nil {
		return nil, &FeedError{Throttled: true, Err: fmt.Errorf("failed to fetch feed: %w", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, &FeedError{
			StatusCode: resp.StatusCode,
			Throttled:  isThrottleStatus(resp.StatusCode),
			Err:        fmt.Errorf("feed request failed with status %d", resp.StatusCode),
		}
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &FeedError{Throttled: true, Err: fmt.Errorf("failed to read response body: %w", err)}
	}

	return ParseFeed(data)
}

func ParseFeed(data []byte) ([]RSSItem, error) {
	var feed AtomFeed
	if err := xml.Unmarshal(data, &feed); err != nil {
		return nil, fmt.Errorf("failed to parse feed: %w", err)
	}

	var items []RSSItem
	for _, entry := range feed.Entries {
		published, err := time.Parse(time.RFC3339, entry.Published)
		if err != nil {
			published = time.Now()
		}

		var url string
		for _, l := range entry.Link {
			if l.Rel == "alternate" || l.Rel == "" {
				url = l.Href
				break
			}
		}
		if url == "" && len(entry.Link) > 0 {
			url = entry.Link[0].Href
		}

		rssItem := RSSItem{
			ID:           entry.ID,
			Title:        entry.Title,
			URL:          url,
			PublishedAt:  published,
		}
		items = append(items, rssItem)
	}

	return items, nil
}
