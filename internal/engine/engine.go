package engine

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"time"
)

var (
	ErrNoFeasibleSlots = errors.New("no feasible time slots found matching constraints")
	ErrInvalidInput    = errors.New("invalid input parameters")
)

// BestWindows finds the top N optimal start windows for an appliance
// given price slots and constraints
func BestWindows(slots []PriceSlot, runMinutes int, constraints Constraints, opts Options, topN int) ([]Recommendation, error) {
	if len(slots) == 0 {
		return nil, ErrInvalidInput
	}
	if runMinutes <= 0 {
		return nil, ErrInvalidInput
	}
	if topN <= 0 {
		topN = 3
	}

	// Calculate number of contiguous 30-min slots needed
	requiredSlots := int(math.Ceil(float64(runMinutes) / 30.0))

	// Filter slots by constraints
	feasible := filterByConstraints(slots, constraints)
	if len(feasible) < requiredSlots {
		return nil, ErrNoFeasibleSlots
	}

	// Find all valid contiguous windows
	candidates := []Recommendation{}
	for i := 0; i+requiredSlots <= len(feasible); i++ {
		window := feasible[i : i+requiredSlots]

		// Verify contiguous
		if !isContiguous(window) {
			continue
		}

		// Calculate cost
		totalPence := 0.0
		for _, slot := range window {
			totalPence += slot.PencePerKWh * (opts.EstKWh / float64(requiredSlots))
		}
		costGBP := totalPence / 100.0

		// Calculate score (lower is better)
		score := totalPence

		rec := Recommendation{
			Start:   window[0].Start,
			End:     window[len(window)-1].End,
			CostGBP: costGBP,
			Score:   score,
			Reason:  generateReason(window, totalPence, slots),
		}
		candidates = append(candidates, rec)
	}

	if len(candidates) == 0 {
		return nil, ErrNoFeasibleSlots
	}

	// Sort by score (ascending)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score < candidates[j].Score
	})

	// Return top N
	if len(candidates) > topN {
		candidates = candidates[:topN]
	}

	return candidates, nil
}

// filterByConstraints returns only slots that meet all constraints
func filterByConstraints(slots []PriceSlot, c Constraints) []PriceSlot {
	result := []PriceSlot{}

	for _, slot := range slots {
		// Check price cap
		if c.PriceCapPence != nil && slot.PencePerKWh > *c.PriceCapPence {
			continue
		}

		// Check if slot falls within allowed windows (if specified)
		if len(c.Allowed) > 0 && !isInTimeWindows(slot.Start, c.Allowed) {
			continue
		}

		// Check if slot falls within blocked windows
		if len(c.Blocked) > 0 && isInTimeWindows(slot.Start, c.Blocked) {
			continue
		}

		// Check quiet hours (if noise level matters)
		if c.NoiseLevel >= 3 && len(c.QuietHours) > 0 && isInTimeWindows(slot.Start, c.QuietHours) {
			continue
		}

		// Check startBy constraint
		if c.StartBy != nil && slot.Start.After(*c.StartBy) {
			continue
		}

		// Check finishBy constraint (slot end must be before finishBy)
		if c.FinishBy != nil && slot.End.After(*c.FinishBy) {
			continue
		}

		result = append(result, slot)
	}

	return result
}

// isInTimeWindows checks if a time falls within any of the specified time windows
func isInTimeWindows(t time.Time, windows []TimeWindow) bool {
	for _, w := range windows {
		if matchesTimeWindow(t, w) {
			return true
		}
	}
	return false
}

// matchesTimeWindow checks if a time matches a specific time window
func matchesTimeWindow(t time.Time, window TimeWindow) bool {
	// Check day of week
	if len(window.DaysOfWeek) > 0 {
		dayMatches := false
		weekday := int(t.Weekday())
		if weekday == 0 {
			weekday = 7 // Sunday = 7
		}
		for _, d := range window.DaysOfWeek {
			if d == weekday {
				dayMatches = true
				break
			}
		}
		if !dayMatches {
			return false
		}
	}

	// Parse window start/end times
	startTime, err := parseTimeOfDay(window.Start)
	if err != nil {
		return false
	}
	endTime, err := parseTimeOfDay(window.End)
	if err != nil {
		return false
	}

	// Get time of day from t
	todayStart := time.Date(t.Year(), t.Month(), t.Day(), startTime.Hour(), startTime.Minute(), 0, 0, t.Location())
	todayEnd := time.Date(t.Year(), t.Month(), t.Day(), endTime.Hour(), endTime.Minute(), 0, 0, t.Location())

	// Handle overnight windows (e.g., 22:00 - 07:00)
	if todayEnd.Before(todayStart) {
		todayEnd = todayEnd.Add(24 * time.Hour)
	}

	return (t.Equal(todayStart) || t.After(todayStart)) && t.Before(todayEnd)
}

// parseTimeOfDay parses HH:mm format
func parseTimeOfDay(s string) (time.Time, error) {
	return time.Parse("15:04", s)
}

// isContiguous verifies that slots are continuous 30-minute periods
func isContiguous(slots []PriceSlot) bool {
	for i := 1; i < len(slots); i++ {
		if !slots[i].Start.Equal(slots[i-1].End) {
			return false
		}
	}
	return true
}

// generateReason creates a human-readable explanation for the recommendation
func generateReason(window []PriceSlot, totalPence float64, allSlots []PriceSlot) string {
	// Calculate percentile
	allPrices := make([]float64, len(allSlots))
	for i, s := range allSlots {
		allPrices[i] = s.PencePerKWh
	}
	sort.Float64s(allPrices)

	avgPence := totalPence / float64(len(window))
	percentile := 0.0
	for i, p := range allPrices {
		if avgPence <= p {
			percentile = float64(i) / float64(len(allPrices))
			break
		}
	}

	if percentile < 0.2 {
		return fmt.Sprintf("Excellent price (bottom %.0f%% of the day)", percentile*100)
	} else if percentile < 0.4 {
		return fmt.Sprintf("Good price (%.0f%% percentile)", percentile*100)
	} else if percentile < 0.6 {
		return "Moderate pricing"
	} else {
		return fmt.Sprintf("Higher price (%.0f%% percentile) but fits constraints", percentile*100)
	}
}
