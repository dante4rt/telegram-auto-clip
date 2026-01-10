package youtube

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type VideoMetadata struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Channel     string  `json:"channel"`
	Duration    float64 `json:"duration"`
	Description string  `json:"description"`
	Thumbnail   string  `json:"thumbnail"`
	URL         string  `json:"webpage_url"`
}

func FetchMetadata(url string) (*VideoMetadata, error) {
	cmd := exec.Command("yt-dlp",
		"--dump-json",
		"--no-download",
		"--no-warnings",
		url,
	)

	output, err := cmd.Output()
	if err != nil {
		// Get stderr for better error messages
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("yt-dlp failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("yt-dlp failed: %w", err)
	}

	var meta VideoMetadata
	if err := json.Unmarshal(output, &meta); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	return &meta, nil
}

func FormatDuration(seconds float64) string {
	mins := int(seconds) / 60
	secs := int(seconds) % 60
	return fmt.Sprintf("%d:%02d", mins, secs)
}

func ExtractVideoID(url string) string {
	// Handle various YouTube URL formats
	if strings.Contains(url, "youtu.be/") {
		parts := strings.Split(url, "youtu.be/")
		if len(parts) > 1 {
			id := strings.Split(parts[1], "?")[0]
			return strings.Split(id, "&")[0]
		}
	}
	if strings.Contains(url, "v=") {
		parts := strings.Split(url, "v=")
		if len(parts) > 1 {
			id := strings.Split(parts[1], "&")[0]
			return strings.Split(id, "#")[0]
		}
	}
	if strings.Contains(url, "/shorts/") {
		parts := strings.Split(url, "/shorts/")
		if len(parts) > 1 {
			id := strings.Split(parts[1], "?")[0]
			return strings.Split(id, "&")[0]
		}
	}
	return ""
}

func IsValidYouTubeURL(url string) bool {
	return strings.Contains(url, "youtube.com") || strings.Contains(url, "youtu.be")
}

func GetPlatformType(url string, duration float64) string {
	if strings.Contains(url, "/shorts/") {
		return "YouTube Shorts"
	}
	if duration <= 60 {
		return "YouTube Short"
	}
	return "YouTube"
}
