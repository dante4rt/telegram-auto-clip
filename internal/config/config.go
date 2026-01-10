package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	PollTimeoutSec     int    `json:"poll_timeout_sec"`
	MaxClipDurationSec int    `json:"max_clip_duration_sec"`
	OutputDir          string `json:"output_dir"`
}

func Load(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return Default(), nil // fallback to defaults
	}
	defer file.Close()

	var cfg Config
	if err := json.NewDecoder(file).Decode(&cfg); err != nil {
		return Default(), nil
	}

	return &cfg, nil
}

func Default() *Config {
	return &Config{
		PollTimeoutSec:     10,
		MaxClipDurationSec: 60,
		OutputDir:          "tmp",
	}
}
