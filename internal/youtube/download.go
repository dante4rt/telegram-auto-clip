package youtube

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"telegram-auto-clip/internal/logger"
)

type DownloadOptions struct {
	URL         string
	OutputDir   string
	StartSec    float64
	EndSec      float64
	OutputFile  string
	CookiesFile string
}

func DownloadSegment(opts DownloadOptions) (string, error) {
	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output dir: %w", err)
	}

	outputPath := filepath.Join(opts.OutputDir, opts.OutputFile)

	// Base args for all strategies
	baseArgs := []string{
		"-f", "bestvideo[height<=1080]+bestaudio/best[height<=1080]/best",
		"--merge-output-format", "mp4",
		"-o", outputPath,
		"--no-warnings",
		"--no-playlist",
	}

	// Add time range if specified
	if opts.StartSec > 0 || opts.EndSec > 0 {
		section := fmt.Sprintf("*%.0f-%.0f", opts.StartSec, opts.EndSec)
		baseArgs = append(baseArgs, "--download-sections", section)
	}

	// Try multiple strategies (ios+cookies is often the winning combo)
	strategies := []struct {
		name string
		args []string
	}{
		{
			name: "ios_cookies",
			args: func() []string {
				args := append(append([]string{}, baseArgs...), "--extractor-args", "youtube:player_client=ios")
				if opts.CookiesFile != "" {
					args = append(args, "--cookies", opts.CookiesFile)
				}
				return args
			}(),
		},
		{
			name: "tv_cookies",
			args: func() []string {
				args := append(append([]string{}, baseArgs...), "--extractor-args", "youtube:player_client=tv_embedded")
				if opts.CookiesFile != "" {
					args = append(args, "--cookies", opts.CookiesFile)
				}
				return args
			}(),
		},
		{
			name: "android_cookies",
			args: func() []string {
				args := append(append([]string{}, baseArgs...), "--extractor-args", "youtube:player_client=android")
				if opts.CookiesFile != "" {
					args = append(args, "--cookies", opts.CookiesFile)
				}
				return args
			}(),
		},
		{
			name: "ios",
			args: append(append([]string{}, baseArgs...), "--extractor-args", "youtube:player_client=ios"),
		},
		{
			name: "cookies",
			args: func() []string {
				args := make([]string, len(baseArgs))
				copy(args, baseArgs)
				if opts.CookiesFile != "" {
					args = append(args, "--cookies", opts.CookiesFile)
				}
				return args
			}(),
		},
	}

	logger.Info("Downloading video segment: %.0f-%.0f seconds", opts.StartSec, opts.EndSec)

	var lastErr error
	for _, strategy := range strategies {
		// Remove old output file if exists from previous attempt
		os.Remove(outputPath)

		args := append(strategy.args, opts.URL)
		logger.Debug("Trying strategy: %s", strategy.name)

		cmd := exec.Command("yt-dlp", args...)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			stderrStr := stderr.String()
			if strings.Contains(stderrStr, "Sign in") || strings.Contains(stderrStr, "bot") {
				logger.Debug("Strategy %s failed: auth required, trying next...", strategy.name)
				lastErr = fmt.Errorf("auth required")
				continue
			}
			logger.Error("yt-dlp stderr: %s", stderrStr)
			lastErr = fmt.Errorf("download failed: %w", err)
			continue
		}

		logger.Info("Download completed with strategy: %s", strategy.name)
		return outputPath, nil
	}

	return "", lastErr
}

func DownloadFull(url, outputDir, outputFile string) (string, error) {
	return DownloadSegment(DownloadOptions{
		URL:        url,
		OutputDir:  outputDir,
		OutputFile: outputFile,
	})
}
