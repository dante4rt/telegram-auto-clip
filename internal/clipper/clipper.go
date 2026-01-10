package clipper

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"telegram-auto-clip/internal/ai"
	"telegram-auto-clip/internal/config"
	"telegram-auto-clip/internal/logger"
	"telegram-auto-clip/internal/video"
	"telegram-auto-clip/internal/youtube"
)

type Clipper struct {
	gemini *ai.GeminiClient
	cfg    *config.Config
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

type StatusCallback func(status string)

func New(geminiKey string, cfg *config.Config) (*Clipper, error) {
	gemini, err := ai.NewGeminiClient(geminiKey)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		return nil, err
	}

	return &Clipper{
		gemini: gemini,
		cfg:    cfg,
	}, nil
}

func (c *Clipper) Close() {
	c.gemini.Close()
}

func (c *Clipper) Process(url string, onStatus StatusCallback) (*ClipResult, error) {
	maxDur := float64(c.cfg.MaxClipDurationSec)
	outDir := c.cfg.OutputDir

	onStatus("Fetching video info...")
	meta, err := youtube.FetchMetadata(url, c.cfg.CookiesFile)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch metadata: %w", err)
	}

	onStatus(fmt.Sprintf("Found: %s | %s | %s",
		meta.Title, youtube.FormatDuration(meta.Duration), meta.Channel))

	videoID := youtube.ExtractVideoID(url)
	if videoID == "" {
		return nil, fmt.Errorf("invalid YouTube URL")
	}

	// Unique request ID to handle concurrent requests for same video
	requestID := fmt.Sprintf("%s_%d_%d", videoID, time.Now().UnixMilli(), rand.Intn(1000))
	logger.Info("[%s] Processing request", requestID)

	var startSec, clipDurationSec float64
	var clipReason string

	if meta.Duration <= maxDur+10 {
		startSec = 0
		clipDurationSec = meta.Duration
		clipReason = "Full video"
		onStatus("Short video, clipping entire video...")
	} else {
		// Strategy 1: Try heatmap first (fastest if available)
		onStatus("Analyzing engagement data...")
		markers, heatmapErr := youtube.FetchHeatmap(videoID, c.cfg.MinHeatmapScore)
		if heatmapErr != nil {
			logger.Debug("Heatmap fetch failed: %v", heatmapErr)
		}
		if heatmapErr == nil && len(markers) > 0 {
			segment := youtube.FindBestSegment(markers, maxDur)
			if segment != nil {
				startSec = segment.StartSec
				clipDurationSec = segment.EndSec - segment.StartSec
				clipReason = "High engagement segment"
				onStatus(fmt.Sprintf("Found viral segment at %s!",
					youtube.FormatDuration(startSec)))
			}
		}

		// Strategy 2: Use Gemini to watch the video and find best moment
		// Only for videos within AI duration limit (longer videos exceed Gemini token limit)
		if startSec == 0 && clipDurationSec == 0 && meta.Duration <= float64(c.cfg.MaxAIVideoDurationSec) {
			onStatus("AI is watching the video to find best moment...")
			suggestion, err := c.gemini.AnalyzeYouTubeVideo(url, meta.Title, meta.Duration, maxDur)
			if err == nil && suggestion != nil {
				startSec = suggestion.StartSec
				clipDurationSec = suggestion.Duration // Use dynamic duration from AI
				clipReason = suggestion.Reason
				onStatus(fmt.Sprintf("AI found best moment at %s (%.0f sec): %s",
					youtube.FormatDuration(startSec), clipDurationSec, suggestion.Reason))
			} else {
				if err != nil {
					logger.Error("AI video analysis failed: %v", err)
				}
			}
		} else if startSec == 0 && clipDurationSec == 0 {
			logger.Debug("Video too long for AI analysis (%.0f min), skipping", meta.Duration/60)
		}

		// Strategy 3: Fallback - pick from middle of video (more interesting than intro)
		if startSec == 0 && clipDurationSec == 0 {
			clipDurationSec = float64(c.cfg.FallbackClipDuration)
			if meta.Duration > 300 { // > 5 min: start from configured percentage into video
				startSec = meta.Duration * c.cfg.FallbackStartPercent
				clipReason = "Early highlight"
				onStatus(fmt.Sprintf("Using segment from %s...", youtube.FormatDuration(startSec)))
			} else {
				startSec = 0
				clipReason = "Video intro"
				onStatus(fmt.Sprintf("Using first %d seconds...", c.cfg.FallbackClipDuration))
			}
		}
	}

	// Add small buffer for context
	endSec := startSec + clipDurationSec + 5
	if endSec > meta.Duration {
		endSec = meta.Duration
	}

	onStatus("Downloading...")
	rawFile := fmt.Sprintf("raw_%s.mp4", requestID)
	rawPath, err := youtube.DownloadSegment(youtube.DownloadOptions{
		URL:         url,
		OutputDir:   outDir,
		StartSec:    startSec,
		EndSec:      endSec,
		OutputFile:  rawFile,
		CookiesFile: c.cfg.CookiesFile,
	})
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}
	defer os.Remove(rawPath)

	onStatus("Processing video to vertical...")
	// Use actual clip duration instead of max duration
	actualDuration := int(clipDurationSec) + 5
	if actualDuration > c.cfg.MaxClipDurationSec {
		actualDuration = c.cfg.MaxClipDurationSec
	}
	finalPath, err := video.ConvertToVertical(rawPath, outDir, actualDuration)
	if err != nil {
		return nil, fmt.Errorf("video processing failed: %w", err)
	}

	onStatus("Generating caption...")
	var caption, hashtags string
	// Use fast caption generation (no video watching, just text)
	captionResult, err := c.gemini.GenerateCaptionFast(meta.Title, meta.Channel, clipReason)
	if err == nil {
		caption = captionResult.Caption
		hashtags = captionResult.Hashtags
	} else {
		logger.Debug("Caption generation failed: %v, using title", err)
		caption = meta.Title
		hashtags = "#viral #fyp #trending #indonesia"
	}

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
	files, _ := filepath.Glob(filepath.Join(c.cfg.OutputDir, "raw_*"))
	for _, f := range files {
		os.Remove(f)
	}
	files, _ = filepath.Glob(filepath.Join(c.cfg.OutputDir, "subs*"))
	for _, f := range files {
		os.Remove(f)
	}
}
