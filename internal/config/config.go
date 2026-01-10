package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	PollTimeoutSec        int     `json:"poll_timeout_sec"`
	MaxClipDurationSec    int     `json:"max_clip_duration_sec"`
	OutputDir             string  `json:"output_dir"`
	MinHeatmapScore       float64 `json:"min_heatmap_score"`
	MaxAIVideoDurationSec int     `json:"max_ai_video_duration_sec"`
	FallbackClipDuration  int     `json:"fallback_clip_duration_sec"`
	FallbackStartPercent  float64 `json:"fallback_start_percent"`
	CookiesFile           string  `json:"cookies_file"`
	CobaltAPIURL          string  `json:"cobalt_api_url"`
}

func Load(path string) (*Config, error) {
	cfg := Default()

	file, err := os.Open(path)
	if err == nil {
		defer file.Close()
		json.NewDecoder(file).Decode(cfg)
	}

	// Override with environment variables
	if cobaltURL := os.Getenv("COBALT_API_URL"); cobaltURL != "" {
		cfg.CobaltAPIURL = cobaltURL
	}

	return cfg, nil
}

func Default() *Config {
	return &Config{
		PollTimeoutSec:        10,
		MaxClipDurationSec:    60,
		OutputDir:             "tmp",
		MinHeatmapScore:       0.15,
		MaxAIVideoDurationSec: 1200, // 20 minutes
		FallbackClipDuration:  45,
		FallbackStartPercent:  0.2,
	}
}
