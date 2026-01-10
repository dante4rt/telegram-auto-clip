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
	Proxies     []string
}

func DownloadSegment(opts DownloadOptions) (string, error) {
	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output dir: %w", err)
	}

	outputPath := filepath.Join(opts.OutputDir, opts.OutputFile)

	logger.Info("Downloading video segment: %.0f-%.0f seconds", opts.StartSec, opts.EndSec)

	var lastErr error
	for _, proxyURL := range opts.Proxies {
		os.Remove(outputPath)

		args := []string{
			"-f", "bv*[height>=480]+ba/bv*+ba/b",
			"-S", "+res:1080,+br",
			"--merge-output-format", "mp4",
			"-o", outputPath,
			"--no-playlist",
			"--extractor-args", "youtube:player_client=ios,web,android",
			"--print-to-file", "%(height)sp", outputPath + ".info",
		}
		if proxyURL != "" {
			args = append(args, "--proxy", proxyURL)
		}
		if opts.StartSec > 0 || opts.EndSec > 0 {
			section := fmt.Sprintf("*%.0f-%.0f", opts.StartSec, opts.EndSec)
			args = append(args, "--download-sections", section)
		}
		if opts.CookiesFile != "" {
			args = append(args, "--cookies", opts.CookiesFile)
		}
		args = append(args, opts.URL)

		cmd := exec.Command("yt-dlp", args...)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			stderrStr := stderr.String()
			method := "direct"
			if proxyURL != "" {
				method = "proxy"
			}
			if strings.Contains(stderrStr, "Sign in") || strings.Contains(stderrStr, "bot") {
				logger.Debug("Auth failed (%s), trying next...", method)
				lastErr = fmt.Errorf("auth required")
				continue
			}
			logger.Error("yt-dlp error (%s): %s", method, stderrStr)
			lastErr = fmt.Errorf("download failed: %w", err)
			continue
		}

		// Log the format that was downloaded
		method := "direct"
		if proxyURL != "" {
			method = "proxy"
		}
		infoFile := outputPath + ".info"
		if data, err := os.ReadFile(infoFile); err == nil {
			logger.Info("Download completed (%s): %s", method, strings.TrimSpace(string(data)))
			os.Remove(infoFile)
		} else {
			logger.Info("Download completed (%s)", method)
		}
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
