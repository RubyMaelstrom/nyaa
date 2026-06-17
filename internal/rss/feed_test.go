package rss

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

var sampleRSS = []byte(`<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Test Channel</title>
  <link rel="alternate" href="https://www.youtube.com/channel/UC123"/>
  <entry>
    <id>yt:video:abc123</id>
    <title>Video One</title>
    <link rel="alternate" href="https://www.youtube.com/watch?v=abc123"/>
    <published>2024-01-01T12:00:00Z</published>
  </entry>
  <entry>
    <id>yt:video:def456</id>
    <title>Video Two</title>
    <link rel="alternate" href="https://www.youtube.com/watch?v=def456"/>
    <published>2024-01-02T08:00:00Z</published>
  </entry>
</feed>`)

func TestParseFeed(t *testing.T) {
	items, err := ParseFeed(sampleRSS)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	// First item
	if items[0].Title != "Video One" {
		t.Errorf("expected 'Video One', got %q", items[0].Title)
	}
	if items[0].URL != "https://www.youtube.com/watch?v=abc123" {
		t.Errorf("unexpected URL: %s", items[0].URL)
	}

	expectedID := "yt:video:abc123"
	if items[0].ID != expectedID {
		t.Errorf("expected ID=%q, got %q", expectedID, items[0].ID)
	}

	// Second item
	if items[1].Title != "Video Two" {
		t.Errorf("expected 'Video Two', got %q", items[1].Title)
	}
	if items[1].URL != "https://www.youtube.com/watch?v=def456" {
		t.Errorf("unexpected URL: %s", items[1].URL)
	}
}

func TestParseFeed_InvalidXML(t *testing.T) {
	_, err := ParseFeed([]byte("not xml"))
	if err == nil {
		t.Error("expected error for invalid XML, got nil")
	}
}

func TestParseFeed_Empty(t *testing.T) {
	items, err := ParseFeed([]byte(`<?xml version="1.0"?><feed xmlns="http://www.w3.org/2005/Atom"><title>Empty</title></feed>`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

func TestFetchFeed_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/feed.xml" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write(sampleRSS)
	}))
	defer server.Close()

	items, err := FetchFeed(server.URL + "/feed.xml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
}

func TestFetchFeed_NetworkError(t *testing.T) {
	_, err := FetchFeed("http://invalid.invalid.test/")
	if err == nil {
		t.Error("expected error for unreachable URL")
	}
}

// YouTube's RSS backend answers a throttled request with 404/5xx; those must be
// classified retryable so the refresh loop retries them on a fresh connection
// instead of treating the channel as dead.
func TestFetchFeed_ThrottleStatusIsRetryable(t *testing.T) {
	cases := []struct {
		status    int
		retryable bool
	}{
		{http.StatusNotFound, true},
		{http.StatusInternalServerError, true},
		{http.StatusBadGateway, true},
		{http.StatusServiceUnavailable, true},
		{http.StatusTooManyRequests, true},
		{http.StatusForbidden, false},
		{http.StatusUnauthorized, false},
	}
	for _, tc := range cases {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(tc.status)
		}))
		_, err := FetchFeed(server.URL)
		server.Close()

		if err == nil {
			t.Errorf("status %d: expected error, got nil", tc.status)
			continue
		}
		var fe *FeedError
		if !errors.As(err, &fe) {
			t.Errorf("status %d: expected *FeedError, got %T", tc.status, err)
			continue
		}
		if fe.StatusCode != tc.status {
			t.Errorf("status %d: FeedError.StatusCode = %d", tc.status, fe.StatusCode)
		}
		if got := IsRetryable(err); got != tc.retryable {
			t.Errorf("status %d: IsRetryable = %v, want %v", tc.status, got, tc.retryable)
		}
	}
}

// A bad-XML body is a permanent failure, not a throttle — it must not be retried.
func TestFetchFeed_ParseErrorNotRetryable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not valid xml"))
	}))
	defer server.Close()

	_, err := FetchFeed(server.URL)
	if err == nil {
		t.Fatal("expected parse error")
	}
	if IsRetryable(err) {
		t.Error("parse error should not be retryable")
	}
}

func TestFetchFeed_ParseError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not valid xml"))
	}))
	defer server.Close()

	_, err := FetchFeed(server.URL)
	if err == nil {
		t.Error("expected parse error for invalid XML")
	}
}
