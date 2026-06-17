package subscriptions

import (
	"os"
	"testing"
)

func setupTestEnv(t *testing.T) {
	t.Helper()
	tmpDir := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Cleanup(func() {
		os.Unsetenv("XDG_CONFIG_HOME")
	})
}

func TestLoad_Empty(t *testing.T) {
	setupTestEnv(t)

	sf, err := Load()
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if sf.Entries == nil {
		t.Fatal("expected Entries map, got nil")
	}
	if sf.Count() != 0 {
		t.Errorf("expected 0 entries, got %d", sf.Count())
	}
}

func TestAdd(t *testing.T) {
	setupTestEnv(t)

	sf, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	added, err := sf.Add("UC123", "Test Channel")
	if err != nil {
		t.Fatalf("Add() unexpected error: %v", err)
	}
	if !added {
		t.Error("expected added=true")
	}
	if sf.Count() != 1 {
		t.Errorf("expected 1 entry, got %d", sf.Count())
	}

	entry := sf.Entries["UC123"]
	if entry.ChannelID != "UC123" {
		t.Errorf("expected ChannelID=UC123, got %s", entry.ChannelID)
	}
	if entry.ChannelName != "Test Channel" {
		t.Errorf("expected ChannelName='Test Channel', got %q", entry.ChannelName)
	}
	if entry.RSSURL != "https://www.youtube.com/feeds/videos.xml?channel_id=UC123" {
		t.Errorf("unexpected RSSURL: %s", entry.RSSURL)
	}
}

func TestAdd_Duplicate(t *testing.T) {
	setupTestEnv(t)

	sf, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	_, err = sf.Add("UC123", "Test Channel")
	if err != nil {
		t.Fatal(err)
	}

	added, err := sf.Add("UC123", "Different Name")
	if err != nil {
		t.Fatalf("Add() error for duplicate: %v", err)
	}
	if added {
		t.Error("expected added=false for duplicate")
	}
	entry := sf.Entries["UC123"]
	if entry.ChannelName != "Test Channel" {
		t.Errorf("expected unchanged entry, got %q", entry.ChannelName)
	}
	if sf.Count() != 1 {
		t.Errorf("expected 1 entry, got %d", sf.Count())
	}
}

func TestRemove(t *testing.T) {
	setupTestEnv(t)

	sf, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	_, err = sf.Add("UC123", "Test Channel")
	if err != nil {
		t.Fatal(err)
	}
	if err := sf.Remove("UC123"); err != nil {
		t.Fatalf("Remove() unexpected error: %v", err)
	}
	if sf.Count() != 0 {
		t.Errorf("expected 0 entries after remove, got %d", sf.Count())
	}

	if err := sf.Remove("UC123"); err == nil {
		t.Error("expected error when removing non-existent channel")
	}
}

func TestIsSubscribed(t *testing.T) {
	setupTestEnv(t)

	sf, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	sf.Add("UC123", "Test")

	if !sf.IsSubscribed("UC123") {
		t.Error("expected IsSubscribed to return true")
	}
	if sf.IsSubscribed("UC999") {
		t.Error("expected IsSubscribed to return false for unknown channel")
	}
}

func TestMarkSeen(t *testing.T) {
	setupTestEnv(t)

	sf, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	sf.Add("UC123", "Test")

	sf.MarkSeen("UC123", "vid123")
	if sf.LastSeen("UC123") != "vid123" {
		t.Errorf("expected LastSeen=vid123, got %q", sf.LastSeen("UC123"))
	}

	sf.MarkSeen("UC123", "vid456")
	if sf.LastSeen("UC123") != "vid456" {
		t.Errorf("expected LastSeen=vid456, got %q", sf.LastSeen("UC123"))
	}
}

func TestSaveLoad(t *testing.T) {
	setupTestEnv(t)

	sf, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	sf.Add("UC123", "Channel A")
	sf.Add("UC456", "Channel B")

	if err := sf.Save(); err != nil {
		t.Fatalf("Save() unexpected error: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if loaded.Count() != 2 {
		t.Errorf("expected 2 entries after reload, got %d", loaded.Count())
	}
	if entryA, exists := loaded.Entries["UC123"]; !exists {
		t.Error("expected UC123 to exist after reload")
	} else {
		if entryA.ChannelName != "Channel A" {
			t.Errorf("expected ChannelName='Channel A', got %q", entryA.ChannelName)
		}
	}
	if entryB, exists := loaded.Entries["UC456"]; !exists {
		t.Error("expected UC456 to exist after reload")
	} else {
		if entryB.ChannelName != "Channel B" {
			t.Errorf("expected ChannelName='Channel B', got %q", entryB.ChannelName)
		}
	}
}

func TestRSSURLForChannel(t *testing.T) {
	url := RSSURLForChannel("UCtest123")
	expected := "https://www.youtube.com/feeds/videos.xml?channel_id=UCtest123"
	if url != expected {
		t.Errorf("expected %s, got %s", expected, url)
	}
}
