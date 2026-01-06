package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/awaistahir/smart-run/internal/engine"
	_ "modernc.org/sqlite"
)

// Store handles persistent storage using SQLite
type Store struct {
	db *sql.DB
}

// NewStore creates a new store and initializes the database
func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	store := &Store{db: db}
	if err := store.initialize(); err != nil {
		db.Close()
		return nil, err
	}

	return store, nil
}

// Close closes the database connection
func (s *Store) Close() error {
	return s.db.Close()
}

// initialize creates the database schema
func (s *Store) initialize() error {
	schema := `
	CREATE TABLE IF NOT EXISTS households (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		region TEXT DEFAULT 'C',
		latitude REAL DEFAULT 51.5074,
		longitude REAL DEFAULT -0.1278,
		quiet_hours TEXT,
		blocked_windows TEXT,
		stagger_heavy_loads INTEGER DEFAULT 0,
		carbon_weight REAL DEFAULT 0.0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS appliances (
		id TEXT PRIMARY KEY,
		household_id TEXT NOT NULL,
		name TEXT NOT NULL,
		cycle_minutes INTEGER NOT NULL,
		tolerance_minutes INTEGER DEFAULT 0,
		allowed_windows TEXT,
		blocked_windows TEXT,
		finish_by TEXT,
		start_by TEXT,
		noise_level INTEGER DEFAULT 3,
		price_cap_pence REAL,
		priority INTEGER DEFAULT 3,
		est_kwh REAL DEFAULT 1.0,
		enabled INTEGER DEFAULT 1,
		control_type TEXT DEFAULT 'manual',
		usage_frequency TEXT DEFAULT 'on_demand',
		class TEXT DEFAULT 'standalone',
		coupled_appliance_id TEXT,
		can_wait_days INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (household_id) REFERENCES households(id)
	);

	CREATE TABLE IF NOT EXISTS price_cache (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		region TEXT NOT NULL,
		date TEXT NOT NULL,
		slots TEXT NOT NULL,
		fetched_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(region, date)
	);

	CREATE TABLE IF NOT EXISTS weather_cache (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		latitude REAL NOT NULL,
		longitude REAL NOT NULL,
		date TEXT NOT NULL,
		slots TEXT NOT NULL,
		fetched_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(latitude, longitude, date)
	);

	CREATE INDEX IF NOT EXISTS idx_appliances_household ON appliances(household_id);
	CREATE INDEX IF NOT EXISTS idx_price_cache_date ON price_cache(region, date);
	CREATE INDEX IF NOT EXISTS idx_weather_cache_date ON weather_cache(latitude, longitude, date);
	`

	_, err := s.db.Exec(schema)
	return err
}

// SaveHousehold saves or updates a household
func (s *Store) SaveHousehold(h *engine.Household) error {
	quietHoursJSON, _ := json.Marshal(h.QuietHours)
	blockedWindowsJSON, _ := json.Marshal(h.BlockedWindows)

	query := `INSERT OR REPLACE INTO households
		(id, name, region, latitude, longitude, quiet_hours, blocked_windows, stagger_heavy_loads, carbon_weight, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.Exec(query, h.ID, h.Name, h.Region, h.Latitude, h.Longitude, string(quietHoursJSON), string(blockedWindowsJSON),
		boolToInt(h.StaggerHeavyLoads), h.CarbonWeight, time.Now())

	return err
}

// GetHousehold retrieves a household by ID
func (s *Store) GetHousehold(id string) (*engine.Household, error) {
	query := `SELECT id, name, region, latitude, longitude, quiet_hours, blocked_windows, stagger_heavy_loads, carbon_weight
		FROM households WHERE id = ?`

	var h engine.Household
	var quietHoursJSON, blockedWindowsJSON string
	var staggerInt int

	err := s.db.QueryRow(query, id).Scan(&h.ID, &h.Name, &h.Region, &h.Latitude, &h.Longitude, &quietHoursJSON, &blockedWindowsJSON,
		&staggerInt, &h.CarbonWeight)

	if err != nil {
		return nil, err
	}

	json.Unmarshal([]byte(quietHoursJSON), &h.QuietHours)
	json.Unmarshal([]byte(blockedWindowsJSON), &h.BlockedWindows)
	h.StaggerHeavyLoads = staggerInt == 1

	return &h, nil
}

// SaveAppliance saves or updates an appliance
func (s *Store) SaveAppliance(a *engine.Appliance, householdID string) error {
	allowedJSON, _ := json.Marshal(a.AllowedWindows)
	blockedJSON, _ := json.Marshal(a.BlockedWindows)

	var finishByStr, startByStr sql.NullString
	if a.FinishBy != nil {
		finishByStr = sql.NullString{String: a.FinishBy.Format(time.RFC3339), Valid: true}
	}
	if a.StartBy != nil {
		startByStr = sql.NullString{String: a.StartBy.Format(time.RFC3339), Valid: true}
	}

	var priceCap sql.NullFloat64
	if a.PriceCapPencePerKWh != nil {
		priceCap = sql.NullFloat64{Float64: *a.PriceCapPencePerKWh, Valid: true}
	}

	// Default control type and usage frequency if not set
	controlType := string(a.ControlType)
	if controlType == "" {
		controlType = "manual"
	}
	usageFrequency := string(a.UsageFrequency)
	if usageFrequency == "" {
		usageFrequency = "on_demand"
	}
	class := string(a.Class)
	if class == "" {
		class = "standalone"
	}

	query := `INSERT OR REPLACE INTO appliances
		(id, household_id, name, cycle_minutes, tolerance_minutes, allowed_windows, blocked_windows,
		 finish_by, start_by, noise_level, price_cap_pence, priority, est_kwh, enabled,
		 control_type, usage_frequency, class, coupled_appliance_id, can_wait_days, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.Exec(query, a.ID, householdID, a.Name, a.CycleMinutes, a.ToleranceMinutes,
		string(allowedJSON), string(blockedJSON), finishByStr, startByStr, a.NoiseLevel,
		priceCap, a.Priority, a.EstKWh, boolToInt(a.Enabled), controlType, usageFrequency,
		class, a.CoupledApplianceID, a.CanWaitDays, time.Now())

	return err
}

// GetAppliances retrieves all appliances for a household
func (s *Store) GetAppliances(householdID string) ([]*engine.Appliance, error) {
	query := `SELECT id, name, cycle_minutes, tolerance_minutes, allowed_windows, blocked_windows,
		finish_by, start_by, noise_level, price_cap_pence, priority, est_kwh, enabled,
		control_type, usage_frequency, class, coupled_appliance_id, can_wait_days
		FROM appliances WHERE household_id = ? ORDER BY priority DESC, name`

	rows, err := s.db.Query(query, householdID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	appliances := []*engine.Appliance{}
	for rows.Next() {
		var a engine.Appliance
		var allowedJSON, blockedJSON string
		var finishByStr, startByStr sql.NullString
		var priceCap sql.NullFloat64
		var enabledInt int
		var controlType, usageFrequency, class string
		var coupledApplianceID sql.NullString
		var canWaitDays int

		err := rows.Scan(&a.ID, &a.Name, &a.CycleMinutes, &a.ToleranceMinutes, &allowedJSON, &blockedJSON,
			&finishByStr, &startByStr, &a.NoiseLevel, &priceCap, &a.Priority, &a.EstKWh, &enabledInt,
			&controlType, &usageFrequency, &class, &coupledApplianceID, &canWaitDays)

		if err != nil {
			continue
		}

		json.Unmarshal([]byte(allowedJSON), &a.AllowedWindows)
		json.Unmarshal([]byte(blockedJSON), &a.BlockedWindows)
		a.ControlType = engine.ControlType(controlType)
		a.UsageFrequency = engine.UsageFrequency(usageFrequency)
		a.Class = engine.ApplianceClass(class)
		if coupledApplianceID.Valid {
			a.CoupledApplianceID = coupledApplianceID.String
		}
		a.CanWaitDays = canWaitDays

		if finishByStr.Valid {
			t, _ := time.Parse(time.RFC3339, finishByStr.String)
			a.FinishBy = &t
		}
		if startByStr.Valid {
			t, _ := time.Parse(time.RFC3339, startByStr.String)
			a.StartBy = &t
		}
		if priceCap.Valid {
			a.PriceCapPencePerKWh = &priceCap.Float64
		}
		a.Enabled = enabledInt == 1

		appliances = append(appliances, &a)
	}

	return appliances, nil
}

// CachePrices stores fetched prices
func (s *Store) CachePrices(region string, date time.Time, slots []engine.PriceSlot) error {
	slotsJSON, _ := json.Marshal(slots)
	dateStr := date.Format("2006-01-02")

	query := `INSERT OR REPLACE INTO price_cache (region, date, slots, fetched_at)
		VALUES (?, ?, ?, ?)`

	_, err := s.db.Exec(query, region, dateStr, string(slotsJSON), time.Now())
	return err
}

// GetCachedPrices retrieves cached prices
func (s *Store) GetCachedPrices(region string, date time.Time) ([]engine.PriceSlot, error) {
	dateStr := date.Format("2006-01-02")
	query := `SELECT slots FROM price_cache WHERE region = ? AND date = ?`

	var slotsJSON string
	err := s.db.QueryRow(query, region, dateStr).Scan(&slotsJSON)
	if err != nil {
		return nil, err
	}

	var slots []engine.PriceSlot
	if err := json.Unmarshal([]byte(slotsJSON), &slots); err != nil {
		return nil, err
	}

	return slots, nil
}

// DeleteAppliance deletes an appliance by ID
func (s *Store) DeleteAppliance(id string) error {
	query := `DELETE FROM appliances WHERE id = ?`
	_, err := s.db.Exec(query, id)
	return err
}

// GetAppliance retrieves a single appliance by ID
func (s *Store) GetAppliance(id string) (*engine.Appliance, error) {
	query := `SELECT id, household_id, name, cycle_minutes, tolerance_minutes, allowed_windows, blocked_windows,
		finish_by, start_by, noise_level, price_cap_pence, priority, est_kwh, enabled,
		control_type, usage_frequency, class, coupled_appliance_id, can_wait_days
		FROM appliances WHERE id = ?`

	var a engine.Appliance
	var householdID string
	var allowedJSON, blockedJSON string
	var finishByStr, startByStr sql.NullString
	var priceCap sql.NullFloat64
	var enabledInt int
	var controlType, usageFrequency, class string
	var coupledApplianceID sql.NullString
	var canWaitDays int

	err := s.db.QueryRow(query, id).Scan(&a.ID, &householdID, &a.Name, &a.CycleMinutes, &a.ToleranceMinutes,
		&allowedJSON, &blockedJSON, &finishByStr, &startByStr, &a.NoiseLevel, &priceCap, &a.Priority,
		&a.EstKWh, &enabledInt, &controlType, &usageFrequency, &class, &coupledApplianceID, &canWaitDays)

	if err != nil {
		return nil, err
	}

	json.Unmarshal([]byte(allowedJSON), &a.AllowedWindows)
	json.Unmarshal([]byte(blockedJSON), &a.BlockedWindows)
	a.ControlType = engine.ControlType(controlType)
	a.UsageFrequency = engine.UsageFrequency(usageFrequency)
	a.Class = engine.ApplianceClass(class)
	if coupledApplianceID.Valid {
		a.CoupledApplianceID = coupledApplianceID.String
	}
	a.CanWaitDays = canWaitDays

	if finishByStr.Valid {
		t, _ := time.Parse(time.RFC3339, finishByStr.String)
		a.FinishBy = &t
	}
	if startByStr.Valid {
		t, _ := time.Parse(time.RFC3339, startByStr.String)
		a.StartBy = &t
	}
	if priceCap.Valid {
		a.PriceCapPencePerKWh = &priceCap.Float64
	}
	a.Enabled = enabledInt == 1

	return &a, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
