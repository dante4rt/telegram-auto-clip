package youtube

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type DownloadOptions struct {
	URL        string
	OutputDir  string
	StartSec   float64
	EndSec     float64
	OutputFile string
}

func DownloadSegment(opts DownloadOptions) (string, error) {
	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output dir: %w", err)
	}

	outputPath := filepath.Join(opts.OutputDir, opts.OutputFile)

	args := []string{
		"-f", "bestvideo[height<=1080]+bestaudio/best[height<=1080]",
		"--merge-output-format", "mp4",
		"-o", outputPath,
		"--no-warnings",
		"--no-playlist",
	}

	// Add time range if specified
	if opts.StartSec > 0 || opts.EndSec > 0 {
		section := fmt.Sprintf("*%.0f-%.0f", opts.StartSec, opts.EndSec)
		args = append(args, "--download-sections", section)
	}

	args = append(args, opts.URL)

	cmd := exec.Command("yt-dlp", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}

	return outputPath, nil
}

func DownloadFull(url, outputDir, outputFile string) (string, error) {
	return DownloadSegment(DownloadOptions{
		URL:        url,
		OutputDir:  outputDir,
		OutputFile: outputFile,
	})
}
