package video

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"

	"telegram-auto-clip/internal/logger"
)

func ConvertToVertical(inputPath, outputDir string, maxDuration int) (string, error) {
	outputPath := filepath.Join(outputDir, "clip_"+filepath.Base(inputPath))

	// Keep original aspect ratio - no transformation
	// User will add bars/effects later
	args := []string{
		"-i", inputPath,
		"-c:v", "libx264",
		"-crf", "23",
		"-preset", "fast",
		"-c:a", "aac",
		"-b:a", "128k",
		"-t", fmt.Sprintf("%d", maxDuration),
		"-y",
		outputPath,
	}

	logger.Info("Running ffmpeg: input=%s output=%s", inputPath, outputPath)

	cmd := exec.Command("ffmpeg", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		logger.Error("ffmpeg stderr: %s", stderr.String())
		return "", fmt.Errorf("ffmpeg failed: %w", err)
	}

	logger.Info("ffmpeg completed successfully")

	return outputPath, nil
}

func GetVideoDuration(videoPath string) (float64, error) {
	args := []string{
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		videoPath,
	}

	cmd := exec.Command("ffprobe", args...)
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	var duration float64
	fmt.Sscanf(string(output), "%f", &duration)
	return duration, nil
}
