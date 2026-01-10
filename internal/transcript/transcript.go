package transcript

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

func FetchTranscript(videoURL, outputDir string) (string, error) {
	// Try YouTube auto-captions first
	transcript, err := fetchYouTubeSubtitles(videoURL, outputDir)
	if err == nil && transcript != "" {
		return transcript, nil
	}

	// Fallback: use yt-dlp with whisper (if available)
	return "", fmt.Errorf("no transcript available")
}

func fetchYouTubeSubtitles(videoURL, outputDir string) (string, error) {
	outputTemplate := filepath.Join(outputDir, "subs")

	args := []string{
		"--write-auto-sub",
		"--sub-lang", "en,id",
		"--skip-download",
		"--sub-format", "vtt",
		"-o", outputTemplate,
		"--no-warnings",
		videoURL,
	}

	cmd := exec.Command("yt-dlp", args...)
	if err := cmd.Run(); err != nil {
		return "", err
	}

	// Look for the subtitle file
	patterns := []string{
		outputTemplate + ".en.vtt",
		outputTemplate + ".id.vtt",
		outputTemplate + ".*.vtt",
	}

	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		if len(matches) > 0 {
			content, err := parseVTT(matches[0])
			os.Remove(matches[0]) // cleanup
			return content, err
		}
	}

	return "", fmt.Errorf("no subtitle file found")
}

func parseVTT(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	timestampRe := regexp.MustCompile(`^\d{2}:\d{2}:\d{2}`)
	tagRe := regexp.MustCompile(`<[^>]+>`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip headers, timestamps, and empty lines
		if line == "" || line == "WEBVTT" || strings.HasPrefix(line, "Kind:") ||
			strings.HasPrefix(line, "Language:") || timestampRe.MatchString(line) ||
			strings.Contains(line, "-->") {
			continue
		}

		// Remove VTT tags
		line = tagRe.ReplaceAllString(line, "")
		line = strings.TrimSpace(line)

		if line != "" {
			lines = append(lines, line)
		}
	}

	// Remove duplicates (VTT often has overlapping captions)
	seen := make(map[string]bool)
	var unique []string
	for _, line := range lines {
		if !seen[line] {
			seen[line] = true
			unique = append(unique, line)
		}
	}

	return strings.Join(unique, " "), nil
}

func ExtractSegmentTranscript(fullTranscript string, startSec, endSec float64) string {
	// For now, return the full transcript
	// In a more sophisticated version, we'd parse timestamps
	if len(fullTranscript) > 2000 {
		return fullTranscript[:2000] + "..."
	}
	return fullTranscript
}
