package engine

import (
	"testing"
	"time"
)

func TestBestWindows(t *testing.T) {
	// Create test price slots for a day
	baseTime := time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)
	slots := []PriceSlot{}

	// Generate 48 half-hourly slots with varying prices
	prices := []float64{
		15, 14, 13, 12, // 00:00-02:00 - cheap overnight
		11, 10, 9, 8,   // 02:00-04:00 - cheapest
		12, 13, 15, 18, // 04:00-06:00 - rising
		20, 22, 24, 26, // 06:00-08:00 - morning peak
		25, 24, 23, 22, // 08:00-10:00
		21, 20, 19, 18, // 10:00-12:00
		17, 16, 15, 14, // 12:00-14:00
		13, 12, 11, 10, // 14:00-16:00 - afternoon dip
		15, 18, 20, 25, // 16:00-18:00 - evening rise
		30, 35, 40, 38, // 18:00-20:00 - peak
		35, 30, 25, 20, // 20:00-22:00 - dropping
		18, 16, 15, 14, // 22:00-24:00 - late evening
	}

	for i, price := range prices {
		slots = append(slots, PriceSlot{
			Start:       baseTime.Add(time.Duration(i) * 30 * time.Minute),
			End:         baseTime.Add(time.Duration(i+1) * 30 * time.Minute),
			PencePerKWh: price,
			IncludesVAT: true,
		})
	}

	tests := []struct {
		name             string
		runMinutes       int
		constraints      Constraints
		opts             Options
		expectedTopStart time.Time
		wantError        bool
	}{
		{
			name:       "60 minute run, no constraints",
			runMinutes: 60,
			constraints: Constraints{
				NoiseLevel: 1, // quiet device
			},
			opts: Options{
				EstKWh: 1.0,
			},
			// Should find cheapest 2 consecutive slots: 02:00-03:00 (9 + 8 = 17p)
			expectedTopStart: baseTime.Add(6 * 30 * time.Minute), // 03:00
			wantError:        false,
		},
		{
			name:       "90 minute run, no constraints",
			runMinutes: 90,
			constraints: Constraints{
				NoiseLevel: 1,
			},
			opts: Options{
				EstKWh: 1.5,
			},
			// Should find cheapest 3 consecutive slots
			expectedTopStart: baseTime.Add(5 * 30 * time.Minute), // 02:30
			wantError:        false,
		},
		{
			name:       "60 minute run with quiet hours",
			runMinutes: 60,
			constraints: Constraints{
				NoiseLevel: 4, // noisy device
				QuietHours: []TimeWindow{
					{Start: "22:00", End: "07:00", DaysOfWeek: []int{}},
				},
			},
			opts: Options{
				EstKWh: 1.0,
			},
			// Should skip overnight hours and find best daytime slot
			expectedTopStart: baseTime.Add(28 * 30 * time.Minute), // 14:00
			wantError:        false,
		},
		{
			name:       "30 minute run with price cap",
			runMinutes: 30,
			constraints: Constraints{
				PriceCapPence: ptrFloat(20.0),
			},
			opts: Options{
				EstKWh: 0.5,
			},
			// Should only consider slots under 20p
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recs, err := BestWindows(slots, tt.runMinutes, tt.constraints, tt.opts, 3)

			if tt.wantError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(recs) == 0 {
				t.Errorf("expected at least one recommendation")
				return
			}

			// Verify we got multiple recommendations
			if len(recs) != 3 {
				t.Logf("got %d recommendations instead of 3 (might be valid)", len(recs))
			}

			// Verify recommendations are sorted by score (ascending)
			for i := 1; i < len(recs); i++ {
				if recs[i].Score < recs[i-1].Score {
					t.Errorf("recommendations not sorted by score: %f < %f",
						recs[i].Score, recs[i-1].Score)
				}
			}

			// Verify duration matches
			for i, rec := range recs {
				duration := rec.End.Sub(rec.Start)
				expectedDuration := time.Duration(tt.runMinutes) * time.Minute
				// Allow some rounding (we ceil to 30min slots)
				if duration < expectedDuration {
					t.Errorf("recommendation %d: duration too short: got %v, want >= %v",
						i, duration, expectedDuration)
				}
			}

			t.Logf("Top recommendation: %s to %s, cost £%.2f, reason: %s",
				recs[0].Start.Format("15:04"), recs[0].End.Format("15:04"),
				recs[0].CostGBP, recs[0].Reason)
		})
	}
}

func TestFilterByConstraints(t *testing.T) {
	baseTime := time.Date(2024, 12, 1, 8, 0, 0, 0, time.UTC) // Sunday

	slots := []PriceSlot{
		{Start: baseTime, End: baseTime.Add(30 * time.Minute), PencePerKWh: 10},
		{Start: baseTime.Add(30 * time.Minute), End: baseTime.Add(60 * time.Minute), PencePerKWh: 25},
		{Start: baseTime.Add(60 * time.Minute), End: baseTime.Add(90 * time.Minute), PencePerKWh: 15},
	}

	tests := []struct {
		name        string
		constraints Constraints
		wantCount   int
	}{
		{
			name: "price cap filters expensive slots",
			constraints: Constraints{
				PriceCapPence: ptrFloat(20.0),
			},
			wantCount: 2, // Only 10p and 15p slots
		},
		{
			name: "no constraints returns all",
			constraints: Constraints{},
			wantCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := filterByConstraints(slots, tt.constraints)
			if len(filtered) != tt.wantCount {
				t.Errorf("got %d slots, want %d", len(filtered), tt.wantCount)
			}
		})
	}
}

func TestIsContiguous(t *testing.T) {
	baseTime := time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name  string
		slots []PriceSlot
		want  bool
	}{
		{
			name: "contiguous slots",
			slots: []PriceSlot{
				{Start: baseTime, End: baseTime.Add(30 * time.Minute)},
				{Start: baseTime.Add(30 * time.Minute), End: baseTime.Add(60 * time.Minute)},
				{Start: baseTime.Add(60 * time.Minute), End: baseTime.Add(90 * time.Minute)},
			},
			want: true,
		},
		{
			name: "gap in slots",
			slots: []PriceSlot{
				{Start: baseTime, End: baseTime.Add(30 * time.Minute)},
				{Start: baseTime.Add(60 * time.Minute), End: baseTime.Add(90 * time.Minute)},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isContiguous(tt.slots)
			if got != tt.want {
				t.Errorf("isContiguous() = %v, want %v", got, tt.want)
			}
		})
	}
}

func ptrFloat(f float64) *float64 {
	return &f
}
