package video

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	OutputWidth  = 720
	OutputHeight = 1280
	MaxDuration  = 60
)

func ConvertToVertical(inputPath, outputDir string) (string, error) {
	outputPath := filepath.Join(outputDir, "vertical_"+filepath.Base(inputPath))

	// ffmpeg filter: scale to 1920 width, then center crop to 1080x1920, then scale to 720x1280
	vf := "scale=1920:-1,crop=1080:1920:(in_w-1080)/2:(in_h-1920)/2,scale=720:1280"

	args := []string{
		"-i", inputPath,
		"-vf", vf,
		"-c:v", "libx264",
		"-crf", "23",
		"-preset", "fast",
		"-c:a", "aac",
		"-b:a", "128k",
		"-t", fmt.Sprintf("%d", MaxDuration),
		"-y",
		outputPath,
	}

	cmd := exec.Command("ffmpeg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ffmpeg failed: %w", err)
	}

	return outputPath, nil
}

func ExtractThumbnail(videoPath, outputPath string, timestampSec float64) error {
	args := []string{
		"-i", videoPath,
		"-ss", fmt.Sprintf("%.2f", timestampSec),
		"-vframes", "1",
		"-y",
		outputPath,
	}

	cmd := exec.Command("ffmpeg", args...)
	return cmd.Run()
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

func Cleanup(paths ...string) {
	for _, p := range paths {
		os.Remove(p)
	}
}
