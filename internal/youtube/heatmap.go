package youtube

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
)

type HeatmapMarker struct {
	StartMillis int     `json:"timeRangeStartMillis"`
	MarkerDurMS int     `json:"markerDurationMillis"`
	Intensity   float64 `json:"heatMarkerIntensityScoreNormalized"`
}

type Segment struct {
	StartSec float64
	EndSec   float64
	Score    float64
}

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

	return parseHeatmapFromHTML(string(body))
}

func parseHeatmapFromHTML(html string) ([]HeatmapMarker, error) {
	// Look for heatMarkers in the page
	re := regexp.MustCompile(`"heatMarkers":\s*(\[.*?\])`)
	matches := re.FindStringSubmatch(html)

	if len(matches) < 2 {
		return nil, fmt.Errorf("no heatmap data found")
	}

	// Parse the JSON array
	var rawMarkers []struct {
		HeatMarkerRenderer HeatmapMarker `json:"heatMarkerRenderer"`
	}

	if err := json.Unmarshal([]byte(matches[1]), &rawMarkers); err != nil {
		return nil, fmt.Errorf("failed to parse heatmap: %w", err)
	}

	markers := make([]HeatmapMarker, len(rawMarkers))
	for i, rm := range rawMarkers {
		markers[i] = rm.HeatMarkerRenderer
	}

	return markers, nil
}

func FindBestSegment(markers []HeatmapMarker, maxDuration float64) *Segment {
	if len(markers) == 0 {
		return nil
	}

	// Convert to segments with scores
	type scoredPoint struct {
		timeSec float64
		score   float64
	}

	points := make([]scoredPoint, len(markers))
	for i, m := range markers {
		points[i] = scoredPoint{
			timeSec: float64(m.StartMillis) / 1000.0,
			score:   m.Intensity,
		}
	}

	// Sort by time
	sort.Slice(points, func(i, j int) bool {
		return points[i].timeSec < points[j].timeSec
	})

	// Sliding window to find best 60-second segment
	bestStart := 0.0
	bestScore := 0.0

	for i := range points {
		startTime := points[i].timeSec
		endTime := startTime + maxDuration
		windowScore := 0.0

		for j := i; j < len(points) && points[j].timeSec < endTime; j++ {
			windowScore += points[j].score
		}

		if windowScore > bestScore {
			bestScore = windowScore
			bestStart = startTime
		}
	}

	// Add 5s padding before if possible
	paddedStart := bestStart - 5
	if paddedStart < 0 {
		paddedStart = 0
	}

	return &Segment{
		StartSec: paddedStart,
		EndSec:   paddedStart + maxDuration + 10, // 5s padding each side
		Score:    bestScore,
	}
}
