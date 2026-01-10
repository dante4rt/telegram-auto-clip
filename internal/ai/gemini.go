package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type GeminiClient struct {
	client *genai.Client
	model  *genai.GenerativeModel
}

type CaptionResult struct {
	Caption   string
	Hashtags  string
	Thumbnail float64 // suggested timestamp for thumbnail
}

type SegmentSuggestion struct {
	StartSec float64
	Reason   string
}

func NewGeminiClient(apiKey string) (*GeminiClient, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create gemini client: %w", err)
	}

	model := client.GenerativeModel("gemini-2.5-flash-lite")
	model.SetTemperature(0.7)
	model.SetTopP(0.9)

	return &GeminiClient{
		client: client,
		model:  model,
	}, nil
}

func (g *GeminiClient) Close() {
	g.client.Close()
}

func (g *GeminiClient) GenerateCaption(title, channel, transcript string) (*CaptionResult, error) {
	prompt := fmt.Sprintf(`You are a social media caption writer. Based ONLY on the following REAL data, generate a caption.

ACTUAL VIDEO DATA:
- Title: %s
- Channel: %s
- Transcript: %s

Generate:
1. A catchy 1-2 sentence caption that summarizes THIS specific content
2. 5-7 relevant hashtags based on the actual topic
3. Suggest a timestamp (in seconds from start) that would make a good thumbnail moment

DO NOT invent or assume any information not provided above.
Respond in this EXACT format:
CAPTION: [your caption here]
HASHTAGS: #tag1 #tag2 #tag3 #tag4 #tag5
THUMBNAIL: [number in seconds]`, title, channel, truncate(transcript, 1500))

	ctx := context.Background()
	resp, err := g.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("gemini generation failed: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response from gemini")
	}

	text := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])
	return parseCaptionResponse(text), nil
}

func (g *GeminiClient) SuggestBestSegment(title, transcript string, videoDuration float64) (*SegmentSuggestion, error) {
	prompt := fmt.Sprintf(`Analyze this video transcript to find the most engaging 60-second segment.

VIDEO: %s (%.0f seconds total)

TRANSCRIPT:
%s

Instructions:
- Use bullet points to identify key moments with timestamps
- Focus on: climactic points, revelations, funny moments, important info
- Select the best 60-second window based on content density

Respond in this EXACT format:
START_SECOND: [number]
REASON: [brief 1-sentence explanation]`, title, videoDuration, truncate(transcript, 2000))

	ctx := context.Background()
	resp, err := g.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("gemini generation failed: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response from gemini")
	}

	text := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])
	return parseSegmentResponse(text, videoDuration), nil
}

func parseCaptionResponse(text string) *CaptionResult {
	result := &CaptionResult{
		Thumbnail: 10, // default
	}

	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "CAPTION:") {
			result.Caption = strings.TrimSpace(strings.TrimPrefix(line, "CAPTION:"))
		} else if strings.HasPrefix(line, "HASHTAGS:") {
			result.Hashtags = strings.TrimSpace(strings.TrimPrefix(line, "HASHTAGS:"))
		} else if strings.HasPrefix(line, "THUMBNAIL:") {
			fmt.Sscanf(strings.TrimPrefix(line, "THUMBNAIL:"), "%f", &result.Thumbnail)
		}
	}

	// Fallback if parsing failed
	if result.Caption == "" {
		result.Caption = "Check out this clip!"
	}
	if result.Hashtags == "" {
		result.Hashtags = "#viral #fyp #trending"
	}

	return result
}

func parseSegmentResponse(text string, maxDuration float64) *SegmentSuggestion {
	result := &SegmentSuggestion{
		StartSec: 0,
		Reason:   "First segment",
	}

	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "START_SECOND:") {
			fmt.Sscanf(strings.TrimPrefix(line, "START_SECOND:"), "%f", &result.StartSec)
		} else if strings.HasPrefix(line, "REASON:") {
			result.Reason = strings.TrimSpace(strings.TrimPrefix(line, "REASON:"))
		}
	}

	// Ensure start is valid
	if result.StartSec < 0 {
		result.StartSec = 0
	}
	if result.StartSec > maxDuration-60 {
		result.StartSec = maxDuration - 60
		if result.StartSec < 0 {
			result.StartSec = 0
		}
	}

	return result
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
