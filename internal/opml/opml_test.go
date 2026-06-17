package opml

import "testing"

const youtubeExport = `<?xml version="1.0" encoding="UTF-8"?>
<opml version="1.1">
  <body>
    <outline text="YouTube Subscriptions" title="YouTube Subscriptions">
      <outline text="Channel Alpha" title="Channel Alpha" type="rss"
        xmlUrl="https://www.youtube.com/feeds/videos.xml?channel_id=UCalpha0000000000000000"/>
      <outline text="Channel Beta" title="Channel Beta" type="rss"
        xmlUrl="https://www.youtube.com/feeds/videos.xml?channel_id=UCbeta00000000000000000"/>
    </outline>
  </body>
</opml>`

func TestParseNested(t *testing.T) {
	subs, err := Parse([]byte(youtubeExport))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if len(subs) != 2 {
		t.Fatalf("got %d subscriptions, want 2", len(subs))
	}
	if subs[0].ChannelID != "UCalpha0000000000000000" {
		t.Errorf("channel id = %q", subs[0].ChannelID)
	}
	if subs[0].Name != "Channel Alpha" {
		t.Errorf("name = %q, want Channel Alpha", subs[0].Name)
	}
	if subs[1].RSSURL == "" {
		t.Error("expected RSSURL to be preserved")
	}
}

func TestParseFlatAndDedupe(t *testing.T) {
	flat := `<opml><body>
		<outline title="A" xmlUrl="https://www.youtube.com/feeds/videos.xml?channel_id=UC1"/>
		<outline title="A again" xmlUrl="https://www.youtube.com/feeds/videos.xml?channel_id=UC1"/>
		<outline title="B" text="B text" xmlUrl="https://www.youtube.com/feeds/videos.xml?channel_id=UC2"/>
		<outline title="folder only, no feed"/>
	</body></opml>`

	subs, err := Parse([]byte(flat))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if len(subs) != 2 {
		t.Fatalf("got %d subscriptions, want 2 (deduped)", len(subs))
	}
	if subs[0].ChannelID != "UC1" || subs[1].ChannelID != "UC2" {
		t.Errorf("unexpected channels: %+v", subs)
	}
}

func TestParseFallsBackToText(t *testing.T) {
	data := `<opml><body>
		<outline text="Only Text" xmlUrl="https://www.youtube.com/feeds/videos.xml?channel_id=UCx"/>
	</body></opml>`
	subs, err := Parse([]byte(data))
	if err != nil {
		t.Fatal(err)
	}
	if len(subs) != 1 || subs[0].Name != "Only Text" {
		t.Errorf("expected name fallback to text attr, got %+v", subs)
	}
}

func TestParseInvalid(t *testing.T) {
	if _, err := Parse([]byte("not xml at all <<<")); err == nil {
		t.Error("expected error for invalid OPML")
	}
}
