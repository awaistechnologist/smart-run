package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/awaistahir/smart-run/internal/engine"
)

const openMeteoAPI = "https://api.open-meteo.com/v1/forecast"

// ForecastClient fetches weather forecasts
type ForecastClient struct {
	lat float64
	lon float64
}

// NewForecastClient creates a weather forecast client for a location
func NewForecastClient(lat, lon float64) *ForecastClient {
	return &ForecastClient{
		lat: lat,
		lon: lon,
	}
}

type dailyForecastResponse struct {
	Daily struct {
		Time          []string  `json:"time"`
		MaxTemp       []float64 `json:"temperature_2m_max"`
		MinTemp       []float64 `json:"temperature_2m_min"`
		PrecipProb    []float64 `json:"precipitation_probability_max"`
		SunshineHours []float64 `json:"sunshine_duration"`
	} `json:"daily"`
}

// GetForecast fetches weather forecast for next N days
func (c *ForecastClient) GetForecast(ctx context.Context, days int) ([]engine.WeatherForecast, error) {
	url := fmt.Sprintf("%s?latitude=%.4f&longitude=%.4f&daily=temperature_2m_max,temperature_2m_min,precipitation_probability_max,sunshine_duration&timezone=Europe/London&forecast_days=%d",
		openMeteoAPI, c.lat, c.lon, days)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("weather API returned status %d", resp.StatusCode)
	}

	var data dailyForecastResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	forecasts := make([]engine.WeatherForecast, 0, len(data.Daily.Time))
	for i := range data.Daily.Time {
		date, err := time.Parse("2006-01-02", data.Daily.Time[i])
		if err != nil {
			continue
		}

		sunshineHours := 0.0
		if i < len(data.Daily.SunshineHours) {
			sunshineHours = data.Daily.SunshineHours[i] / 3600.0 // Convert seconds to hours
		}

		precipProb := 0.0
		if i < len(data.Daily.PrecipProb) {
			precipProb = data.Daily.PrecipProb[i]
		}

		// Good drying conditions: >3 hours sunshine, <30% rain probability, temp >12C
		isSunny := sunshineHours > 3.0 && precipProb < 30.0 && data.Daily.MaxTemp[i] > 12.0

		forecasts = append(forecasts, engine.WeatherForecast{
			Date:          date,
			SunshineHours: sunshineHours,
			MaxTempC:      data.Daily.MaxTemp[i],
			MinTempC:      data.Daily.MinTemp[i],
			PrecipProb:    precipProb,
			IsSunny:       isSunny,
		})
	}

	return forecasts, nil
}
