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

func FetchMetadata(url string, cookiesFile string) (*VideoMetadata, error) {
	// Try multiple strategies in order (ios+cookies is often the winning combo)
	strategies := []struct {
		name string
		args []string
	}{
		{
			name: "ios_cookies",
			args: func() []string {
				args := []string{"--dump-json", "--skip-download", "--no-warnings", "--extractor-args", "youtube:player_client=ios"}
				if cookiesFile != "" {
					args = append(args, "--cookies", cookiesFile)
				}
				return args
			}(),
		},
		{
			name: "tv_cookies",
			args: func() []string {
				args := []string{"--dump-json", "--skip-download", "--no-warnings", "--extractor-args", "youtube:player_client=tv_embedded"}
				if cookiesFile != "" {
					args = append(args, "--cookies", cookiesFile)
				}
				return args
			}(),
		},
		{
			name: "android_cookies",
			args: func() []string {
				args := []string{"--dump-json", "--skip-download", "--no-warnings", "--extractor-args", "youtube:player_client=android"}
				if cookiesFile != "" {
					args = append(args, "--cookies", cookiesFile)
				}
				return args
			}(),
		},
		{
			name: "ios",
			args: []string{"--dump-json", "--skip-download", "--no-warnings", "--extractor-args", "youtube:player_client=ios"},
		},
		{
			name: "cookies",
			args: func() []string {
				args := []string{"--dump-json", "--skip-download", "--no-warnings"}
				if cookiesFile != "" {
					args = append(args, "--cookies", cookiesFile)
				}
				return args
			}(),
		},
	}

	var lastErr error
	for _, strategy := range strategies {
		args := append(strategy.args, url)
		cmd := exec.Command("yt-dlp", args...)

		output, err := cmd.Output()
		if err == nil {
			var meta VideoMetadata
			if err := json.Unmarshal(output, &meta); err != nil {
				lastErr = fmt.Errorf("failed to parse metadata: %w", err)
				continue
			}
			return &meta, nil
		}

		// Check if it's an auth error, try next strategy
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := string(exitErr.Stderr)
			if strings.Contains(stderr, "Sign in") || strings.Contains(stderr, "bot") {
				lastErr = fmt.Errorf("strategy %s failed: auth required", strategy.name)
				continue
			}
			lastErr = fmt.Errorf("yt-dlp failed: %s", stderr)
		} else {
			lastErr = fmt.Errorf("yt-dlp failed: %w", err)
		}
	}

	return nil, lastErr
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
