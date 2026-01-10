package youtube

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

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

	args := []string{
		"-f", "bestvideo[height<=1080]+bestaudio/best[height<=1080]/best",
		"--merge-output-format", "mp4",
		"-o", outputPath,
		"--no-warnings",
		"--no-playlist",
	}

	// Add cookies file if specified (needed for some videos on servers)
	if opts.CookiesFile != "" {
		args = append(args, "--cookies", opts.CookiesFile)
	}

	// Add time range if specified
	if opts.StartSec > 0 || opts.EndSec > 0 {
		section := fmt.Sprintf("*%.0f-%.0f", opts.StartSec, opts.EndSec)
		args = append(args, "--download-sections", section)
	}

	args = append(args, opts.URL)

	logger.Info("Downloading video segment: %.0f-%.0f seconds", opts.StartSec, opts.EndSec)
	logger.Debug("yt-dlp output: %s", outputPath)

	cmd := exec.Command("yt-dlp", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		logger.Error("yt-dlp stderr: %s", stderr.String())
		return "", fmt.Errorf("download failed: %w", err)
	}

	logger.Info("Download completed: %s", outputPath)
	return outputPath, nil
}

func DownloadFull(url, outputDir, outputFile string) (string, error) {
	return DownloadSegment(DownloadOptions{
		URL:        url,
		OutputDir:  outputDir,
		OutputFile: outputFile,
	})
}
