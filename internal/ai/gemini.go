package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/genai"

	"telegram-auto-clip/internal/logger"
)

type GeminiClient struct {
	client *genai.Client
}

type CaptionResult struct {
	Caption  string
	Hashtags string
}

type SegmentSuggestion struct {
	StartSec float64
	Duration float64 // dynamic duration
	Reason   string
}

func NewGeminiClient(apiKey string) (*GeminiClient, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create gemini client: %w", err)
	}

	return &GeminiClient{
		client: client,
	}, nil
}

func (g *GeminiClient) Close() {
	// New SDK doesn't require explicit close
}

func (g *GeminiClient) generateWithRetry(ctx context.Context, model string, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
	maxRetries := 2
	for i := 0; i <= maxRetries; i++ {
		result, err := g.client.Models.GenerateContent(ctx, model, contents, config)
		if err == nil {
			return result, nil
		}
		if !strings.Contains(err.Error(), "429") && !strings.Contains(err.Error(), "RESOURCE_EXHAUSTED") {
			return nil, err
		}
		if i == maxRetries {
			return nil, err
		}
		wait := 30 * time.Second
		logger.Info("Gemini rate limited, retrying in %v... (%d/%d)", wait, i+1, maxRetries)
		time.Sleep(wait)
	}
	return nil, fmt.Errorf("unreachable")
}

// AnalyzeYouTubeVideo uses Gemini to directly analyze a YouTube video and find the best segment
func (g *GeminiClient) AnalyzeYouTubeVideo(youtubeURL, title string, videoDuration float64, maxClipDuration float64) (*SegmentSuggestion, error) {
	ctx := context.Background()

	prompt := fmt.Sprintf(`Kamu adalah analis video YouTube yang mencari momen paling VIRAL untuk dijadikan clip pendek.

VIDEO: %s (%.0f detik total)

Tonton video dan temukan momen PALING MENARIK. Cari:
- Momen lucu atau mengejutkan
- Puncak emosi (excitement, tension, reveal)
- Highlight atau klimaks
- Kutipan yang memorable
- Aksi seru

PENTING: Durasi clip harus DINAMIS (15-60 detik) tergantung kontennya:
- Jika momennya singkat (joke, reaction): 15-30 detik
- Jika momennya butuh konteks: 30-45 detik
- Jika momennya panjang (cerita, gameplay epic): 45-60 detik

Respond dalam format EXACT ini:
START_SECOND: [angka detik untuk mulai]
DURATION: [durasi dalam detik, 15-60]
REASON: [penjelasan singkat 1 kalimat kenapa momen ini menarik]

Penting: Start time harus antara 0 dan %.0f detik.`,
		title, videoDuration, videoDuration-maxClipDuration)

	parts := []*genai.Part{
		genai.NewPartFromURI(youtubeURL, "video/mp4"),
		genai.NewPartFromText(prompt),
	}

	contents := []*genai.Content{
		genai.NewContentFromParts(parts, genai.RoleUser),
	}

	config := &genai.GenerateContentConfig{
		Temperature: genai.Ptr(float32(0.7)),
		TopP:        genai.Ptr(float32(0.9)),
	}

	logger.Info("Asking Gemini to analyze YouTube video...")
	result, err := g.generateWithRetry(ctx, "gemini-2.5-flash", contents, config)
	if err != nil {
		return nil, fmt.Errorf("gemini video analysis failed: %w", err)
	}

	text := result.Text()
	logger.Debug("Gemini response: %s", truncateLog(text, 200))

	return parseSegmentResponse(text, videoDuration, maxClipDuration), nil
}

// GenerateCaptionFast generates caption quickly without watching video (text-only)
func (g *GeminiClient) GenerateCaptionFast(title, channel, reason string) (*CaptionResult, error) {
	ctx := context.Background()

	prompt := fmt.Sprintf(`Kamu adalah penulis caption social media yang kreatif dan engaging.

DATA VIDEO:
- Judul: %s
- Channel: %s
- Momen yang diclip: %s

Buatkan caption dalam Bahasa Indonesia yang:
1. Catchy dan bikin penasaran (1-2 kalimat)
2. Pakai bahasa gaul/slang yang relate sama anak muda
3. Bisa bikin orang mau nonton dan share

Generate juga 5-7 hashtag yang relevan (mix Indonesia & English).

Format response:
CAPTION: [caption kamu]
HASHTAGS: #hashtag1 #hashtag2 #hashtag3 #hashtag4 #hashtag5`, title, channel, reason)

	parts := []*genai.Part{
		genai.NewPartFromText(prompt),
	}

	contents := []*genai.Content{
		genai.NewContentFromParts(parts, genai.RoleUser),
	}

	config := &genai.GenerateContentConfig{
		Temperature: genai.Ptr(float32(0.9)), // more creative
		TopP:        genai.Ptr(float32(0.95)),
	}

	result, err := g.generateWithRetry(ctx, "gemini-2.5-flash", contents, config)
	if err != nil {
		return nil, fmt.Errorf("gemini caption generation failed: %w", err)
	}

	text := result.Text()
	return parseCaptionResponse(text), nil
}

func parseCaptionResponse(text string) *CaptionResult {
	result := &CaptionResult{}

	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "CAPTION:") {
			result.Caption = strings.TrimSpace(strings.TrimPrefix(line, "CAPTION:"))
		} else if strings.HasPrefix(line, "HASHTAGS:") {
			result.Hashtags = strings.TrimSpace(strings.TrimPrefix(line, "HASHTAGS:"))
		}
	}

	// Fallback if parsing failed
	if result.Caption == "" {
		result.Caption = "Cek clip ini! 🔥"
	}
	if result.Hashtags == "" {
		result.Hashtags = "#viral #fyp #trending #indonesia"
	}

	return result
}

func parseSegmentResponse(text string, videoDuration, maxDuration float64) *SegmentSuggestion {
	result := &SegmentSuggestion{
		StartSec: 0,
		Duration: 60, // default
		Reason:   "Best moment",
	}

	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "START_SECOND:") {
			timeStr := strings.TrimSpace(strings.TrimPrefix(line, "START_SECOND:"))
			result.StartSec = parseTimeToSeconds(timeStr)
		} else if strings.HasPrefix(line, "DURATION:") {
			fmt.Sscanf(strings.TrimPrefix(line, "DURATION:"), "%f", &result.Duration)
		} else if strings.HasPrefix(line, "REASON:") {
			result.Reason = strings.TrimSpace(strings.TrimPrefix(line, "REASON:"))
		}
	}

	// Validate duration (15-60 seconds)
	if result.Duration < 15 {
		result.Duration = 15
	}
	if result.Duration > maxDuration {
		result.Duration = maxDuration
	}

	// Ensure start is valid
	if result.StartSec < 0 {
		result.StartSec = 0
	}
	if result.StartSec > videoDuration-result.Duration {
		result.StartSec = videoDuration - result.Duration
		if result.StartSec < 0 {
			result.StartSec = 0
		}
	}

	return result
}

// parseTimeToSeconds converts time strings like "7:55", "1:23:45", or "475" to seconds
func parseTimeToSeconds(timeStr string) float64 {
	timeStr = strings.TrimSpace(timeStr)

	// Try parsing as plain number first
	var seconds float64
	if _, err := fmt.Sscanf(timeStr, "%f", &seconds); err == nil && !strings.Contains(timeStr, ":") {
		return seconds
	}

	// Parse MM:SS or HH:MM:SS format
	parts := strings.Split(timeStr, ":")
	switch len(parts) {
	case 2: // MM:SS
		var min, sec int
		fmt.Sscanf(parts[0], "%d", &min)
		fmt.Sscanf(parts[1], "%d", &sec)
		return float64(min*60 + sec)
	case 3: // HH:MM:SS
		var hour, min, sec int
		fmt.Sscanf(parts[0], "%d", &hour)
		fmt.Sscanf(parts[1], "%d", &min)
		fmt.Sscanf(parts[2], "%d", &sec)
		return float64(hour*3600 + min*60 + sec)
	}

	return 0
}

func truncateLog(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
