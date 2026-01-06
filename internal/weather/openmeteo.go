package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/awaistahir/smart-run/internal/engine"
)

const openMeteoAPIBase = "https://api.open-meteo.com/v1/forecast"

// OpenMeteoClient fetches weather forecasts from Open-Meteo API
type OpenMeteoClient struct {
	httpClient *http.Client
	latitude   float64
	longitude  float64
}

// NewOpenMeteoClient creates a new Open-Meteo client
func NewOpenMeteoClient(lat, lon float64) *OpenMeteoClient {
	return &OpenMeteoClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		latitude:   lat,
		longitude:  lon,
	}
}

// openMeteoResponse represents the API response
type openMeteoResponse struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Hourly    struct {
		Time               []string  `json:"time"`
		Temperature2m      []float64 `json:"temperature_2m"`
		RelativeHumidity2m []int     `json:"relative_humidity_2m"`
		WindSpeed10m       []float64 `json:"wind_speed_10m"`
		PrecipitationProb  []int     `json:"precipitation_probability"`
	} `json:"hourly"`
}

// Forecast fetches hourly weather forecast for the next 2 days
func (c *OpenMeteoClient) Forecast(ctx context.Context) ([]engine.WeatherSlot, error) {
	params := url.Values{}
	params.Add("latitude", fmt.Sprintf("%.4f", c.latitude))
	params.Add("longitude", fmt.Sprintf("%.4f", c.longitude))
	params.Add("hourly", "temperature_2m,relative_humidity_2m,wind_speed_10m,precipitation_probability")
	params.Add("forecast_days", "2")
	params.Add("timezone", "Europe/London")

	fullURL := fmt.Sprintf("%s?%s", openMeteoAPIBase, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching weather: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var meteoResp openMeteoResponse
	if err := json.NewDecoder(resp.Body).Decode(&meteoResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	// Convert to WeatherSlots
	slots := make([]engine.WeatherSlot, 0, len(meteoResp.Hourly.Time))
	for i := range meteoResp.Hourly.Time {
		t, err := time.Parse("2006-01-02T15:04", meteoResp.Hourly.Time[i])
		if err != nil {
			continue
		}

		// Load London timezone
		loc, _ := time.LoadLocation("Europe/London")
		t = t.In(loc)

		slots = append(slots, engine.WeatherSlot{
			Time:       t,
			TempC:      meteoResp.Hourly.Temperature2m[i],
			Humidity:   float64(meteoResp.Hourly.RelativeHumidity2m[i]),
			WindMps:    meteoResp.Hourly.WindSpeed10m[i],
			PrecipProb: float64(meteoResp.Hourly.PrecipitationProb[i]),
		})
	}

	return slots, nil
}

// GetWeatherForTime finds the weather slot closest to a given time
func GetWeatherForTime(slots []engine.WeatherSlot, t time.Time) *engine.WeatherSlot {
	if len(slots) == 0 {
		return nil
	}

	// Find closest slot
	closest := 0
	minDiff := absDuration(slots[0].Time.Sub(t))

	for i := 1; i < len(slots); i++ {
		diff := absDuration(slots[i].Time.Sub(t))
		if diff < minDiff {
			minDiff = diff
			closest = i
		}
	}

	return &slots[closest]
}

func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}
