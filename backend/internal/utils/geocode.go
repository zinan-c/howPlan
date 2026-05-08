package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type nominatimItem struct {
	Lat string `json:"lat"`
	Lon string `json:"lon"`
}

// GeocodeLocation resolves free-text query into latitude/longitude using Nominatim.
func GeocodeLocation(query string) (float64, float64, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return 0, 0, fmt.Errorf("empty location")
	}

	u := "https://nominatim.openstreetmap.org/search?format=jsonv2&limit=1&q=" + url.QueryEscape(q)
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return 0, 0, err
	}
	req.Header.Set("User-Agent", "travel-planner-viewer/1.0 (+import geocoding)")

	client := &http.Client{Timeout: 6 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("geocode http status: %d", resp.StatusCode)
	}

	var items []nominatimItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return 0, 0, err
	}
	if len(items) == 0 {
		return 0, 0, fmt.Errorf("no geocode result")
	}

	lat, err := strconv.ParseFloat(items[0].Lat, 64)
	if err != nil {
		return 0, 0, err
	}
	lon, err := strconv.ParseFloat(items[0].Lon, 64)
	if err != nil {
		return 0, 0, err
	}

	return lat, lon, nil
}
