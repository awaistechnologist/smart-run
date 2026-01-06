package engine

import "time"

// PriceSlot represents a 30-minute electricity pricing period
type PriceSlot struct {
	Start        time.Time
	End          time.Time
	PencePerKWh  float64
	IncludesVAT  bool
}

// WeatherSlot represents weather conditions at a point in time
type WeatherSlot struct {
	Time           time.Time
	TempC          float64
	Humidity       float64 // percentage 0-100
	WindMps        float64 // meters per second
	PrecipProb     float64 // percentage 0-100
}

// WeatherForecast represents daily weather summary
type WeatherForecast struct {
	Date          time.Time
	SunshineHours float64 // Hours of sunshine
	MaxTempC      float64
	MinTempC      float64
	PrecipProb    float64 // percentage 0-100
	IsSunny       bool    // Good drying conditions
}

// TimeWindow represents a time range with optional day-of-week filtering
type TimeWindow struct {
	Start      string // HH:mm format
	End        string // HH:mm format
	DaysOfWeek []int  // 1=Monday, 7=Sunday; empty = all days
}

// Constraints defines the scheduling constraints for an appliance
type Constraints struct {
	Allowed       []TimeWindow
	Blocked       []TimeWindow
	QuietHours    []TimeWindow
	FinishBy      *time.Time
	StartBy       *time.Time
	PriceCapPence *float64
	NoiseLevel    int // 1-5, affects quiet hours filtering
}

// Options contains parameters for the optimization algorithm
type Options struct {
	EstKWh       float64 // Estimated energy consumption
	CarbonWeight float64 // 0-1, weight for carbon optimization
	PVWeight     float64 // 0-1, weight for PV self-consumption
}

// Recommendation represents a suggested start window for an appliance
type Recommendation struct {
	Start   time.Time
	End     time.Time
	CostGBP float64
	Reason  string
	Score   float64
}

// SmartRecommendation represents an intelligent recommendation that considers weather, coupling, and multi-day options
type SmartRecommendation struct {
	ApplianceName    string
	Options          []RecommendationOption // Multiple options (today, tomorrow, etc.)
	BestOptionIndex  int                    // Index of the recommended option
}

// RecommendationOption represents one possible scheduling option
type RecommendationOption struct {
	Day              string               // "Today", "Tomorrow", "Wednesday"
	Date             time.Time
	PrimarySlot      Recommendation       // Main appliance time
	CoupledSlot      *Recommendation      // Coupled appliance time (e.g., dryer after washer)
	TotalCostGBP     float64              // Combined cost
	Weather          *WeatherForecast     // Weather conditions for this day
	UsesNaturalDry   bool                 // If true, skips tumble dryer and line-dries
	SavingsVsToday   float64              // Money saved vs running today (negative if more expensive)
	Recommendation   string               // Human-readable recommendation
}

// ControlType defines how an appliance is controlled
type ControlType string

const (
	ControlManual ControlType = "manual" // User presses button
	ControlSmart  ControlType = "smart"  // Smart plug/HA automation
)

// UsageFrequency defines how often an appliance should be run
type UsageFrequency string

const (
	FrequencyDaily     UsageFrequency = "daily"      // Every day
	Frequency3xWeek    UsageFrequency = "3x_week"    // 3 times per week
	FrequencyWeekly    UsageFrequency = "weekly"     // Once per week
	FrequencyOnDemand  UsageFrequency = "on_demand"  // Only when requested
)

// ApplianceClass defines the operational type of an appliance
type ApplianceClass string

const (
	ClassStandalone       ApplianceClass = "standalone"        // Runs independently (dishwasher, EV)
	ClassCoupled          ApplianceClass = "coupled"           // Requires another appliance after (washing machine → dryer)
	ClassWeatherDependent ApplianceClass = "weather_dependent" // Can be replaced by natural conditions (dryer → sun)
)

// Appliance represents a household appliance to be scheduled
type Appliance struct {
	ID                  string
	Name                string
	CycleMinutes        int
	ToleranceMinutes    int
	AllowedWindows      []TimeWindow
	BlockedWindows      []TimeWindow
	FinishBy            *time.Time
	StartBy             *time.Time
	NoiseLevel          int
	PriceCapPencePerKWh *float64
	Priority            int
	EstKWh              float64
	Enabled             bool
	ControlType         ControlType    // manual or smart
	UsageFrequency      UsageFrequency // how often to run
	Class               ApplianceClass // standalone, coupled, or weather_dependent
	CoupledApplianceID  string         // ID of appliance that runs after this one
	CanWaitDays         int            // How many days user can wait for better conditions (0 = must run today)
}

// Household represents household-level preferences and constraints
type Household struct {
	ID                string
	Name              string
	Region            string // Octopus region code (A-P)
	Latitude          float64 // For weather forecasts
	Longitude         float64 // For weather forecasts
	QuietHours        []TimeWindow
	BlockedWindows    []TimeWindow
	AvailableHours    []TimeWindow // When you're home to start manual appliances
	StaggerHeavyLoads bool
	CarbonWeight      float64
}
