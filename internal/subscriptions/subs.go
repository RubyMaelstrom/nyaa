package subscriptions

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type SubscriptionsFile struct {
	Entries map[string]SubscriptionEntry `json:"entries"`
}

type SubscriptionEntry struct {
	ChannelID       string    `json:"channel_id"`
	ChannelName     string    `json:"channel_name"`
	RSSURL          string    `json:"rss_url"`
	SubscribedAt    time.Time `json:"subscribed_at"`
	LastSeenVideoID string    `json:"last_seen_video_id"`
}

func RSSURLForChannel(channelID string) string {
	return fmt.Sprintf("https://www.youtube.com/feeds/videos.xml?channel_id=%s", channelID)
}

func configDir() (string, error) {
	configHome, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to locate user config directory: %w", err)
	}

	return filepath.Join(configHome, "nyaa"), nil
}

func dataFile() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, "subscriptions.json"), nil
}

func Load() (*SubscriptionsFile, error) {
	path, err := dataFile()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &SubscriptionsFile{
				Entries: make(map[string]SubscriptionEntry),
			}, nil
		}

		return nil, fmt.Errorf("failed to read subscriptions: %w", err)
	}

	var sf SubscriptionsFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return nil, fmt.Errorf("failed to parse subscriptions: %w", err)
	}

	if sf.Entries == nil {
		sf.Entries = make(map[string]SubscriptionEntry)
	}

	return &sf, nil
}

func (sf *SubscriptionsFile) Save() error {
	dir, err := configDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if sf.Entries == nil {
		sf.Entries = make(map[string]SubscriptionEntry)
	}

	data, err := json.MarshalIndent(sf, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal subscriptions: %w", err)
	}

	path, err := dataFile()
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write subscriptions: %w", err)
	}

	return nil
}

func (sf *SubscriptionsFile) Add(channelID, channelName string) (added bool, err error) {
	if _, exists := sf.Entries[channelID]; exists {
		return false, nil
	}

	sf.Entries[channelID] = SubscriptionEntry{
		ChannelID:    channelID,
		ChannelName:  channelName,
		RSSURL:       RSSURLForChannel(channelID),
		SubscribedAt: time.Now(),
	}

	return true, nil
}

func (sf *SubscriptionsFile) Remove(channelID string) error {
	if _, exists := sf.Entries[channelID]; !exists {
		return fmt.Errorf("channel %s is not subscribed", channelID)
	}

	delete(sf.Entries, channelID)

	return nil
}

func (sf *SubscriptionsFile) IsSubscribed(channelID string) bool {
	_, exists := sf.Entries[channelID]
	return exists
}

func (sf *SubscriptionsFile) MarkSeen(channelID, videoID string) {
	entry, exists := sf.Entries[channelID]
	if !exists {
		return
	}

	entry.LastSeenVideoID = videoID
	sf.Entries[channelID] = entry
}

func (sf *SubscriptionsFile) LastSeen(channelID string) string {
	entry, exists := sf.Entries[channelID]
	if !exists {
		return ""
	}

	return entry.LastSeenVideoID
}

func (sf *SubscriptionsFile) Count() int {
	return len(sf.Entries)
}
