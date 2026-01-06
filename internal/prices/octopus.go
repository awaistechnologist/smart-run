package prices

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

const (
	octopusAPIBase = "https://api.octopus.energy/v1"
	// Current Agile product code - update as needed
	defaultAgileProduct = "AGILE-24-10-01"
)

// OctopusClient fetches electricity prices from Octopus Energy Agile tariff
type OctopusClient struct {
	httpClient *http.Client
	product    string
	region     string
}

// NewOctopusClient creates a new client for the Octopus Agile API
func NewOctopusClient(region string) *OctopusClient {
	return &OctopusClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		product:    defaultAgileProduct,
		region:     region,
	}
}

// octopusResponse represents the API response structure
type octopusResponse struct {
	Count    int          `json:"count"`
	Next     *string      `json:"next"`
	Previous *string      `json:"previous"`
	Results  []resultItem `json:"results"`
}

type resultItem struct {
	ValueExcVAT  float64   `json:"value_exc_vat"`
	ValueIncVAT  float64   `json:"value_inc_vat"`
	ValidFrom    time.Time `json:"valid_from"`
	ValidTo      time.Time `json:"valid_to"`
	PaymentMethod *string  `json:"payment_method"`
}

// HalfHourly fetches half-hourly prices for a specific day and region
func (c *OctopusClient) HalfHourly(ctx context.Context, day time.Time, region string) ([]engine.PriceSlot, error) {
	if region == "" {
		region = c.region
	}

	// Construct tariff code: E-1R-{PRODUCT}-{REGION}
	tariffCode := fmt.Sprintf("E-1R-%s-%s", c.product, region)

	// Build URL
	endpoint := fmt.Sprintf("%s/products/%s/electricity-tariffs/%s/standard-unit-rates/",
		octopusAPIBase, c.product, tariffCode)

	// Set period for the full day in UTC
	startOfDay := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
	endOfDay := startOfDay.Add(24 * time.Hour)

	// Build query params
	params := url.Values{}
	params.Add("period_from", startOfDay.Format(time.RFC3339))
	params.Add("period_to", endOfDay.Format(time.RFC3339))

	fullURL := fmt.Sprintf("%s?%s", endpoint, params.Encode())

	// Make request
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching prices: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var octResp octopusResponse
	if err := json.NewDecoder(resp.Body).Decode(&octResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	// Convert to PriceSlots
	slots := make([]engine.PriceSlot, 0, len(octResp.Results))
	for _, r := range octResp.Results {
		slots = append(slots, engine.PriceSlot{
			Start:        r.ValidFrom,
			End:          r.ValidTo,
			PencePerKWh:  r.ValueIncVAT,
			IncludesVAT:  true,
		})
	}

	// Sort by start time (API returns in reverse chronological order)
	sortSlotsByTime(slots)

	return slots, nil
}

// sortSlotsByTime sorts price slots in ascending time order
func sortSlotsByTime(slots []engine.PriceSlot) {
	// Simple bubble sort since we're dealing with max 48 slots
	for i := 0; i < len(slots)-1; i++ {
		for j := 0; j < len(slots)-i-1; j++ {
			if slots[j].Start.After(slots[j+1].Start) {
				slots[j], slots[j+1] = slots[j+1], slots[j]
			}
		}
	}
}

// FetchTodayAndTomorrow fetches prices for today and tomorrow (if available)
func (c *OctopusClient) FetchTodayAndTomorrow(ctx context.Context, region string) ([]engine.PriceSlot, error) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	tomorrow := today.Add(24 * time.Hour)

	// Fetch today
	todaySlots, err := c.HalfHourly(ctx, today, region)
	if err != nil {
		return nil, fmt.Errorf("fetching today's prices: %w", err)
	}

	// Fetch tomorrow (may fail if not yet published)
	tomorrowSlots, err := c.HalfHourly(ctx, tomorrow, region)
	if err != nil {
		// Tomorrow not available yet, that's okay
		return todaySlots, nil
	}

	return append(todaySlots, tomorrowSlots...), nil
}
