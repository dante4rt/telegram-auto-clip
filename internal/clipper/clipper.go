package clipper

import (
	"fmt"
	"os"
	"path/filepath"
	"telegram-auto-clip/internal/ai"
	"telegram-auto-clip/internal/transcript"
	"telegram-auto-clip/internal/video"
	"telegram-auto-clip/internal/youtube"
)

type Clipper struct {
	gemini    *ai.GeminiClient
	outputDir string
}

type ClipResult struct {
	VideoPath   string
	Title       string
	Channel     string
	Duration    string
	Platform    string
	Caption     string
	Hashtags    string
	OriginalURL string
}

func New(geminiKey, outputDir string) (*Clipper, error) {
	gemini, err := ai.NewGeminiClient(geminiKey)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, err
	}

	return &Clipper{
		gemini:    gemini,
		outputDir: outputDir,
	}, nil
}

func (c *Clipper) Close() {
	c.gemini.Close()
}

type StatusCallback func(status string)

func (c *Clipper) Process(url string, onStatus StatusCallback) (*ClipResult, error) {
	// 1. Fetch metadata
	onStatus("Fetching video info...")
	meta, err := youtube.FetchMetadata(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch metadata: %w", err)
	}

	onStatus(fmt.Sprintf("Video found: %s | Duration: %s | Channel: %s",
		meta.Title, youtube.FormatDuration(meta.Duration), meta.Channel))

	videoID := youtube.ExtractVideoID(url)
	if videoID == "" {
		return nil, fmt.Errorf("invalid YouTube URL")
	}

	// 2. Find best segment
	var startSec, endSec float64

	if meta.Duration <= 70 {
		// Short video, use entire thing
		startSec = 0
		endSec = meta.Duration
		onStatus("Short video, clipping entire video...")
	} else {
		// Try heatmap first
		onStatus("Analyzing engagement data...")
		markers, err := youtube.FetchHeatmap(videoID)
		if err == nil && len(markers) > 0 {
			segment := youtube.FindBestSegment(markers, 60)
			if segment != nil {
				startSec = segment.StartSec
				endSec = segment.EndSec
				onStatus(fmt.Sprintf("Found high-engagement segment at %s",
					youtube.FormatDuration(startSec)))
			}
		}

		// Fallback to Gemini if no heatmap
		if startSec == 0 && endSec == 0 {
			onStatus("No heatmap, using AI to find best segment...")

			// Get transcript for AI analysis
			trans, _ := transcript.FetchTranscript(url, c.outputDir)
			if trans != "" {
				suggestion, err := c.gemini.SuggestBestSegment(meta.Title, trans, meta.Duration)
				if err == nil {
					startSec = suggestion.StartSec
					endSec = startSec + 70 // 60s + padding
					onStatus(fmt.Sprintf("AI selected segment at %s: %s",
						youtube.FormatDuration(startSec), suggestion.Reason))
				}
			}
		}

		// Final fallback: first minute
		if startSec == 0 && endSec == 0 {
			startSec = 0
			endSec = 70
			onStatus("Using first minute of video...")
		}
	}

	// 3. Download segment
	onStatus("Downloading...")
	rawFile := fmt.Sprintf("raw_%s.mp4", videoID)
	rawPath, err := youtube.DownloadSegment(youtube.DownloadOptions{
		URL:        url,
		OutputDir:  c.outputDir,
		StartSec:   startSec,
		EndSec:     endSec,
		OutputFile: rawFile,
	})
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}
	defer os.Remove(rawPath)

	// 4. Convert to vertical
	onStatus("Processing video...")
	finalPath, err := video.ConvertToVertical(rawPath, c.outputDir)
	if err != nil {
		return nil, fmt.Errorf("video processing failed: %w", err)
	}

	// 5. Get transcript for caption
	onStatus("Generating caption...")
	trans, _ := transcript.FetchTranscript(url, c.outputDir)

	// 6. Generate caption with Gemini
	var caption, hashtags string
	captionResult, err := c.gemini.GenerateCaption(meta.Title, meta.Channel, trans)
	if err == nil {
		caption = captionResult.Caption
		hashtags = captionResult.Hashtags
	} else {
		caption = meta.Title
		hashtags = "#viral #fyp #trending"
	}

	// Get actual clip duration
	clipDuration, _ := video.GetVideoDuration(finalPath)

	return &ClipResult{
		VideoPath:   finalPath,
		Title:       meta.Title,
		Channel:     meta.Channel,
		Duration:    youtube.FormatDuration(clipDuration),
		Platform:    youtube.GetPlatformType(url, meta.Duration),
		Caption:     caption,
		Hashtags:    hashtags,
		OriginalURL: url,
	}, nil
}

func (c *Clipper) Cleanup(result *ClipResult) {
	if result != nil && result.VideoPath != "" {
		os.Remove(result.VideoPath)
	}
	// Clean temp files
	files, _ := filepath.Glob(filepath.Join(c.outputDir, "raw_*"))
	for _, f := range files {
		os.Remove(f)
	}
	files, _ = filepath.Glob(filepath.Join(c.outputDir, "subs*"))
	for _, f := range files {
		os.Remove(f)
	}
}
