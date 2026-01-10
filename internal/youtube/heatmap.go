package youtube

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"telegram-auto-clip/internal/logger"
)

type HeatmapMarker struct {
	StartMillis int     `json:"startMillis"`
	DurationMS  int     `json:"durationMillis"`
	Intensity   float64 `json:"intensityScoreNormalized"`
}

type Segment struct {
	StartSec float64
	EndSec   float64
	Score    float64
}

func FetchHeatmap(videoID string, minScore float64) ([]HeatmapMarker, error) {
	url := fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID)

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	logger.Debug("Fetched YouTube page, size: %d bytes", len(body))
	return parseHeatmapFromHTML(string(body), minScore)
}

func parseHeatmapFromHTML(html string, minScore float64) ([]HeatmapMarker, error) {
	// Find "heatMarkerRenderer" which is specific to heatmap data
	if !strings.Contains(html, "heatMarkerRenderer") {
		logger.Debug("No heatMarkerRenderer found in page")
		return nil, fmt.Errorf("no heatmap data found")
	}

	// Find markers array that contains heatMarkerRenderer
	startIdx := strings.Index(html, `"markers":[{"heatMarkerRenderer"`)
	if startIdx == -1 {
		logger.Debug("No heatmap markers array found")
		return nil, fmt.Errorf("no heatmap data found")
	}

	// Move to the start of the array
	startIdx = strings.Index(html[startIdx:], "[") + startIdx

	// Find matching closing bracket using bracket counting
	depth := 0
	endIdx := startIdx
outer:
	for i := startIdx; i < len(html); i++ {
		switch html[i] {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				endIdx = i + 1
				break outer
			}
		}
	}

	if endIdx <= startIdx {
		logger.Debug("Could not find closing bracket for markers array")
		return nil, fmt.Errorf("no heatmap data found")
	}

	jsonStr := html[startIdx:endIdx]
	logger.Debug("Found heatmap data, parsing JSON (%d chars)...", len(jsonStr))

	// Parse the JSON array
	var rawMarkers []map[string]json.RawMessage
	if err := json.Unmarshal([]byte(jsonStr), &rawMarkers); err != nil {
		logger.Error("Failed to parse markers JSON: %v", err)
		return nil, fmt.Errorf("failed to parse heatmap: %w", err)
	}

	logger.Debug("Parsed %d raw markers", len(rawMarkers))

	var allMarkers []HeatmapMarker
	for _, rm := range rawMarkers {
		// Check if it has heatMarkerRenderer
		if renderer, ok := rm["heatMarkerRenderer"]; ok {
			var marker HeatmapMarker
			if err := json.Unmarshal(renderer, &marker); err != nil {
				continue
			}
			allMarkers = append(allMarkers, marker)
		}
	}

	if len(allMarkers) == 0 {
		logger.Debug("No heatmap markers found")
		return nil, fmt.Errorf("no heatmap data found")
	}

	// Sort by intensity (highest first)
	sort.Slice(allMarkers, func(i, j int) bool {
		return allMarkers[i].Intensity > allMarkers[j].Intensity
	})

	// Filter by minimum score, but always return at least top 5
	var markers []HeatmapMarker
	for _, m := range allMarkers {
		if m.Intensity >= minScore {
			markers = append(markers, m)
		}
	}

	// If no markers pass threshold, take top 5 anyway
	if len(markers) == 0 {
		maxTake := 5
		if len(allMarkers) < maxTake {
			maxTake = len(allMarkers)
		}
		markers = allMarkers[:maxTake]
		logger.Debug("No markers above threshold, using top %d (best score: %.2f)", len(markers), markers[0].Intensity)
	}

	logger.Info("Found %d engagement segments (best: %.2f)", len(markers), markers[0].Intensity)
	return markers, nil
}

func FindBestSegment(markers []HeatmapMarker, maxDuration float64) *Segment {
	if len(markers) == 0 {
		return nil
	}

	// Markers are already sorted by intensity (highest first)
	// Take the highest scoring marker
	best := markers[0]
	startSec := float64(best.StartMillis) / 1000.0
	durationSec := float64(best.DurationMS) / 1000.0

	// Use the marker's own duration or maxDuration, whichever is smaller
	if durationSec > maxDuration {
		durationSec = maxDuration
	}

	// Add padding (5s before, 5s after)
	const padding = 5.0
	paddedStart := startSec - padding
	if paddedStart < 0 {
		paddedStart = 0
	}

	endSec := startSec + durationSec + padding

	logger.Info("Best segment: start=%.0fs, score=%.2f", startSec, best.Intensity)

	return &Segment{
		StartSec: paddedStart,
		EndSec:   endSec,
		Score:    best.Intensity,
	}
}
