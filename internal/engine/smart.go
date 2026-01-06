package engine

import (
	"fmt"
	"time"
)

// GenerateSmartRecommendations creates intelligent recommendations considering weather and coupling
func GenerateSmartRecommendations(
	appliance *Appliance,
	coupledAppliance *Appliance,
	pricesByDay map[string][]PriceSlot, // Map of "2006-01-02" -> price slots
	weatherByDay map[string]*WeatherForecast,
	household *Household,
	constraints Constraints,
	opts Options,
) (*SmartRecommendation, error) {

	if appliance.Class == ClassStandalone {
		// For standalone appliances, just return best time for today
		return generateStandaloneRecommendation(appliance, pricesByDay, household, constraints, opts)
	}

	if appliance.Class == ClassCoupled && coupledAppliance != nil {
		// For coupled appliances, consider multiple days and weather
		return generateCoupledRecommendation(appliance, coupledAppliance, pricesByDay, weatherByDay, household, constraints, opts)
	}

	return nil, fmt.Errorf("unsupported appliance class or missing coupled appliance")
}

func generateStandaloneRecommendation(
	appliance *Appliance,
	pricesByDay map[string][]PriceSlot,
	household *Household,
	constraints Constraints,
	opts Options,
) (*SmartRecommendation, error) {

	options := []RecommendationOption{}

	// Just find best time for today
	today := time.Now().Format("2006-01-02")
	if prices, ok := pricesByDay[today]; ok {
		recs, err := BestWindows(prices, appliance.CycleMinutes, constraints, opts, 1)
		if err == nil && len(recs) > 0 {
			options = append(options, RecommendationOption{
				Day:            "Today",
				Date:           recs[0].Start,
				PrimarySlot:    recs[0],
				TotalCostGBP:   recs[0].CostGBP,
				Recommendation: fmt.Sprintf("Best time: %s - %s", recs[0].Start.Format("15:04"), recs[0].End.Format("15:04")),
			})
		}
	}

	if len(options) == 0 {
		return nil, fmt.Errorf("no feasible slots found")
	}

	return &SmartRecommendation{
		ApplianceName:   appliance.Name,
		Options:         options,
		BestOptionIndex: 0,
	}, nil
}

func generateCoupledRecommendation(
	washer *Appliance,
	dryer *Appliance,
	pricesByDay map[string][]PriceSlot,
	weatherByDay map[string]*WeatherForecast,
	household *Household,
	washerConstraints Constraints,
	washerOpts Options,
) (*SmartRecommendation, error) {

	options := []RecommendationOption{}
	daysToCheck := washer.CanWaitDays
	if daysToCheck == 0 {
		daysToCheck = 1 // At least check today
	}
	if daysToCheck > 3 {
		daysToCheck = 3 // Max 3 days
	}

	now := time.Now()
	for dayOffset := 0; dayOffset < daysToCheck; dayOffset++ {
		checkDate := now.AddDate(0, 0, dayOffset)
		dateStr := checkDate.Format("2006-01-02")

		prices, hasPrices := pricesByDay[dateStr]
		if !hasPrices {
			continue
		}

		// Find best time for washer
		washerRecs, err := BestWindows(prices, washer.CycleMinutes, washerConstraints, washerOpts, 1)
		if err != nil || len(washerRecs) == 0 {
			continue
		}

		washerSlot := washerRecs[0]
		dayName := getDayName(dayOffset)

		// Check weather for this day
		weather, hasWeather := weatherByDay[dateStr]

		// Option 1: Tumble dry (use dryer)
		if dryer != nil {
			// Dryer starts after washer ends
			dryerStart := washerSlot.End
			dryerEnd := dryerStart.Add(time.Duration(dryer.CycleMinutes) * time.Minute)

			// Find price for dryer slot
			dryerCost := estimateCost(prices, dryerStart, dryerEnd, dryer.EstKWh)

			totalCost := washerSlot.CostGBP + dryerCost

			option := RecommendationOption{
				Day:          dayName,
				Date:         checkDate,
				PrimarySlot:  washerSlot,
				CoupledSlot:  &Recommendation{Start: dryerStart, End: dryerEnd, CostGBP: dryerCost},
				TotalCostGBP: totalCost,
				Weather:      weather,
				UsesNaturalDry: false,
			}

			if len(options) > 0 {
				option.SavingsVsToday = options[0].TotalCostGBP - totalCost
			}

			option.Recommendation = fmt.Sprintf("Start wash at %s, finishes at %s. Then tumble dry until %s (£%.2f total)",
				washerSlot.Start.Local().Format("15:04"), washerSlot.End.Local().Format("15:04"),
				dryerEnd.Local().Format("15:04"), totalCost)

			options = append(options, option)
		}

		// Option 2: Line dry (if weather is good)
		if hasWeather && weather.IsSunny && dryer != nil && dryer.Class == ClassWeatherDependent {
			option := RecommendationOption{
				Day:            dayName,
				Date:           checkDate,
				PrimarySlot:    washerSlot,
				CoupledSlot:    nil,
				TotalCostGBP:   washerSlot.CostGBP,
				Weather:        weather,
				UsesNaturalDry: true,
			}

			if len(options) > 0 {
				option.SavingsVsToday = options[0].TotalCostGBP - washerSlot.CostGBP
			}

			option.Recommendation = fmt.Sprintf("Start wash at %s, finishes at %s. Then hang outside to dry in sunshine (£%.2f, save £%.2f!)",
				washerSlot.Start.Local().Format("15:04"), washerSlot.End.Local().Format("15:04"), washerSlot.CostGBP, option.SavingsVsToday)

			options = append(options, option)
		}
	}

	if len(options) == 0 {
		return nil, fmt.Errorf("no feasible options found")
	}

	// Find best option (lowest cost)
	bestIdx := 0
	for i := 1; i < len(options); i++ {
		if options[i].TotalCostGBP < options[bestIdx].TotalCostGBP {
			bestIdx = i
		}
	}

	return &SmartRecommendation{
		ApplianceName:   washer.Name,
		Options:         options,
		BestOptionIndex: bestIdx,
	}, nil
}

func getDayName(dayOffset int) string {
	switch dayOffset {
	case 0:
		return "Today"
	case 1:
		return "Tomorrow"
	default:
		return time.Now().AddDate(0, 0, dayOffset).Format("Monday")
	}
}

func estimateCost(prices []PriceSlot, start, end time.Time, kwh float64) float64 {
	totalCostPence := 0.0
	slots := 0

	// Count matching slots
	for _, p := range prices {
		if (p.Start.Equal(start) || p.Start.After(start)) && p.Start.Before(end) {
			slots++
		}
	}

	if slots == 0 {
		return 0
	}

	// Split energy evenly across slots and calculate cost for each
	kwhPerSlot := kwh / float64(slots)
	for _, p := range prices {
		if (p.Start.Equal(start) || p.Start.After(start)) && p.Start.Before(end) {
			totalCostPence += p.PencePerKWh * kwhPerSlot
		}
	}

	return totalCostPence / 100.0 // Convert pence to pounds
}
