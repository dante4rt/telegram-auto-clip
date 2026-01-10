package youtube

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
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

const MinHeatmapScore = 0.40 // Minimum intensity to be considered viral

func FetchHeatmap(videoID string) ([]HeatmapMarker, error) {
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
	return parseHeatmapFromHTML(string(body))
}

func parseHeatmapFromHTML(html string) ([]HeatmapMarker, error) {
	// Match the pattern from reference: "markers":[...],"markersMetadata"
	// Use (?s) for DOTALL mode in Go regex
	re := regexp.MustCompile(`(?s)"markers":\s*(\[.*?\])\s*,\s*"?markersMetadata"?`)
	matches := re.FindStringSubmatch(html)

	if len(matches) < 2 {
		logger.Debug("No heatmap markers found in page")
		return nil, fmt.Errorf("no heatmap data found")
	}

	logger.Debug("Found heatmap data, parsing JSON...")

	// Clean up the JSON (remove escaped quotes if needed)
	jsonStr := strings.ReplaceAll(matches[1], `\"`, `"`)

	// Parse the JSON array - markers contain heatMarkerRenderer
	var rawMarkers []map[string]json.RawMessage
	if err := json.Unmarshal([]byte(jsonStr), &rawMarkers); err != nil {
		logger.Error("Failed to parse markers JSON: %v", err)
		return nil, fmt.Errorf("failed to parse heatmap: %w", err)
	}

	logger.Debug("Parsed %d raw markers", len(rawMarkers))

	var markers []HeatmapMarker
	for _, rm := range rawMarkers {
		// Check if it has heatMarkerRenderer
		if renderer, ok := rm["heatMarkerRenderer"]; ok {
			var marker HeatmapMarker
			if err := json.Unmarshal(renderer, &marker); err != nil {
				continue
			}
			// Only include high-engagement segments
			if marker.Intensity >= MinHeatmapScore {
				markers = append(markers, marker)
			}
		}
	}

	if len(markers) == 0 {
		logger.Debug("No high-engagement markers found (score >= %.2f)", MinHeatmapScore)
		return nil, fmt.Errorf("no high-engagement segments found")
	}

	// Sort by intensity (highest first)
	sort.Slice(markers, func(i, j int) bool {
		return markers[i].Intensity > markers[j].Intensity
	})

	logger.Info("Found %d high-engagement segments", len(markers))
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
