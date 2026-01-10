package cobalt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"time"

	"telegram-auto-clip/internal/logger"
)

type Client struct {
	apiURL     string
	httpClient *http.Client
}

type DownloadRequest struct {
	URL          string `json:"url"`
	VideoQuality string `json:"videoQuality,omitempty"`
	AudioFormat  string `json:"audioFormat,omitempty"`
	DownloadMode string `json:"downloadMode,omitempty"`
}

type DownloadResponse struct {
	Status   string `json:"status"`
	URL      string `json:"url,omitempty"`
	Filename string `json:"filename,omitempty"`
	Error    *struct {
		Code    string `json:"code"`
		Context *struct {
			Service string `json:"service,omitempty"`
			Limit   int    `json:"limit,omitempty"`
		} `json:"context,omitempty"`
	} `json:"error,omitempty"`
}

func New(apiURL string) *Client {
	return &Client{
		apiURL: apiURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetDownloadURL requests a download URL from cobalt API
func (c *Client) GetDownloadURL(videoURL string, quality string) (*DownloadResponse, error) {
	if quality == "" {
		quality = "1080"
	}

	req := DownloadRequest{
		URL:          videoURL,
		VideoQuality: quality,
		DownloadMode: "auto",
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result DownloadResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Status == "error" && result.Error != nil {
		return nil, fmt.Errorf("cobalt error: %s", result.Error.Code)
	}

	return &result, nil
}

// DownloadToFile downloads video from cobalt tunnel/redirect URL to file
func (c *Client) DownloadToFile(downloadURL, outputPath string) error {
	client := &http.Client{
		Timeout: 10 * time.Minute, // Long timeout for large files
	}

	resp, err := client.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	written, err := io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	logger.Info("Downloaded %d bytes via cobalt", written)
	return nil
}

// Download is a convenience method that gets URL and downloads in one call
func (c *Client) Download(videoURL, outputPath, quality string) error {
	resp, err := c.GetDownloadURL(videoURL, quality)
	if err != nil {
		return err
	}

	if resp.URL == "" {
		return fmt.Errorf("no download URL returned")
	}

	return c.DownloadToFile(resp.URL, outputPath)
}

// DownloadSegment downloads a specific segment using ffmpeg's HTTP seeking
// This avoids downloading the entire video when we only need a portion
func (c *Client) DownloadSegment(videoURL, outputPath, quality string, startSec, endSec float64) error {
	resp, err := c.GetDownloadURL(videoURL, quality)
	if err != nil {
		return err
	}

	if resp.URL == "" {
		return fmt.Errorf("no download URL returned")
	}

	logger.Info("Got cobalt URL, downloading segment %.0f-%.0f with ffmpeg", startSec, endSec)

	// Use ffmpeg to download only the segment we need
	// -ss before -i enables input seeking (fast, doesn't download from start)
	args := []string{
		"-y",
		"-ss", fmt.Sprintf("%.0f", startSec),
	}

	// Add end time if specified
	if endSec > startSec {
		duration := endSec - startSec
		args = append(args, "-t", fmt.Sprintf("%.0f", duration))
	}

	args = append(args,
		"-i", resp.URL,
		"-c", "copy", // Copy streams without re-encoding (fast)
		"-movflags", "+faststart",
		outputPath,
	)

	cmd := exec.Command("ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If copy fails (usually due to seeking issues), try with re-encoding
		logger.Debug("ffmpeg copy failed, trying with re-encode: %s", string(output))
		args = []string{
			"-y",
			"-ss", fmt.Sprintf("%.0f", startSec),
		}
		if endSec > startSec {
			duration := endSec - startSec
			args = append(args, "-t", fmt.Sprintf("%.0f", duration))
		}
		args = append(args,
			"-i", resp.URL,
			"-c:v", "libx264", "-preset", "fast", "-crf", "23",
			"-c:a", "aac", "-b:a", "128k",
			"-movflags", "+faststart",
			outputPath,
		)
		cmd = exec.Command("ffmpeg", args...)
		output, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("ffmpeg failed: %w - %s", err, string(output))
		}
	}

	// Verify file was created
	if info, err := os.Stat(outputPath); err != nil || info.Size() == 0 {
		return fmt.Errorf("output file not created or empty")
	}

	logger.Info("Cobalt segment download completed")
	return nil
}
