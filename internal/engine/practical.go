package engine

// ApplyPracticalConstraints adjusts constraints based on appliance control type
// to ensure recommendations are actually usable
func ApplyPracticalConstraints(appliance *Appliance, household *Household, constraints *Constraints) {
	// For MANUAL appliances: only recommend times when user is available
	if appliance.ControlType == ControlManual {
		// If household has defined available hours, use those
		if len(household.AvailableHours) > 0 {
			constraints.Allowed = household.AvailableHours
		} else {
			// Default: assume user is available 7am-11:30pm
			// Only the START time needs to be in this window (user needs to be awake to press start)
			// The appliance can finish running after 11:30pm
			constraints.Allowed = []TimeWindow{
				{Start: "07:00", End: "23:30", DaysOfWeek: []int{1, 2, 3, 4, 5, 6, 7}},
			}
		}
	}

	// For SMART appliances: can run anytime (user loads it, automation starts it)
	// No additional constraints needed - keep existing allowed windows
}

// ShouldShowRecommendation determines if we should show a recommendation today
// based on usage frequency
func ShouldShowRecommendation(appliance *Appliance, lastRunDate string, currentDate string) bool {
	switch appliance.UsageFrequency {
	case FrequencyDaily:
		return true // Show every day

	case Frequency3xWeek:
		// TODO: Track last 3 runs in database
		// For now, show on Mon/Wed/Fri
		// This is a simplified implementation
		return true // Placeholder

	case FrequencyWeekly:
		// TODO: Track last run
		// For now, show on Mondays
		return true // Placeholder

	case FrequencyOnDemand:
		return false // Never show automatically

	default:
		return false
	}
}
