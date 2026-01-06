package uiapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/awaistahir/smart-run/internal/engine"
	"github.com/awaistahir/smart-run/internal/prices"
	"github.com/awaistahir/smart-run/internal/store"
	"github.com/awaistahir/smart-run/internal/weather"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	store *store.Store
}

func NewServer(store *store.Store) *Server {
	return &Server{
		store: store,
	}
}

// getRegion retrieves the region from household settings
func (s *Server) getRegion() string {
	household, err := s.store.GetHousehold("default")
	if err != nil || household.Region == "" {
		return "C" // Default to London if not set
	}
	return household.Region
}

func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	// CORS for local development
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	// Serve static files
	r.Get("/", s.serveUI)
	r.Get("/static/*", s.serveStatic)

	// API routes
	r.Route("/api", func(r chi.Router) {
		r.Get("/status", s.handleStatus)
		r.Get("/prices", s.handleGetPrices)
		r.Get("/household", s.handleGetHousehold)
		r.Put("/household", s.handleUpdateHousehold)
		r.Get("/appliances", s.handleGetAppliances)
		r.Post("/appliances", s.handleCreateAppliance)
		r.Get("/appliances/{id}", s.handleGetAppliance)
		r.Put("/appliances/{id}", s.handleUpdateAppliance)
		r.Delete("/appliances/{id}", s.handleDeleteAppliance)
		r.Post("/recommendations", s.handleGetRecommendations)
		r.Post("/smart-recommendations", s.handleSmartRecommendations)
		r.Get("/weather", s.handleGetWeather)
	})

	return r
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	region := s.getRegion()
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "ok",
		"version": "1.0.0",
		"region":  region,
	})
}

func (s *Server) handleGetPrices(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	region := s.getRegion()
	client := prices.NewOctopusClient(region)

	priceSlots, err := client.FetchTodayAndTomorrow(ctx, region)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Filter to current and future slots
	now := time.Now()
	futureSlots := []engine.PriceSlot{}
	for _, slot := range priceSlots {
		if slot.End.After(now) {
			futureSlots = append(futureSlots, slot)
		}
	}

	respondJSON(w, http.StatusOK, futureSlots)
}

func (s *Server) handleGetHousehold(w http.ResponseWriter, r *http.Request) {
	household, err := s.store.GetHousehold("default")
	if err != nil {
		respondError(w, http.StatusNotFound, "household not found")
		return
	}

	respondJSON(w, http.StatusOK, household)
}

func (s *Server) handleUpdateHousehold(w http.ResponseWriter, r *http.Request) {
	var household engine.Household
	if err := json.NewDecoder(r.Body).Decode(&household); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	household.ID = "default"
	if err := s.store.SaveHousehold(&household); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, household)
}

func (s *Server) handleGetAppliances(w http.ResponseWriter, r *http.Request) {
	appliances, err := s.store.GetAppliances("default")
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, appliances)
}

func (s *Server) handleCreateAppliance(w http.ResponseWriter, r *http.Request) {
	var appliance engine.Appliance
	if err := json.NewDecoder(r.Body).Decode(&appliance); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if appliance.ID == "" {
		appliance.ID = appliance.Name + "-" + time.Now().Format("20060102150405")
	}

	if err := s.store.SaveAppliance(&appliance, "default"); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, appliance)
}

func (s *Server) handleGetAppliance(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	appliance, err := s.store.GetAppliance(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "appliance not found")
		return
	}
	respondJSON(w, http.StatusOK, appliance)
}

func (s *Server) handleUpdateAppliance(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var appliance engine.Appliance
	if err := json.NewDecoder(r.Body).Decode(&appliance); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	appliance.ID = id
	if err := s.store.SaveAppliance(&appliance, "default"); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, appliance)
}

func (s *Server) handleDeleteAppliance(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.store.DeleteAppliance(id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "deleted", "id": id})
}

type RecommendationRequest struct {
	ApplianceIDs []string `json:"appliance_ids"`
}

type RecommendationResponse struct {
	Appliance       string                  `json:"appliance"`
	Recommendations []engine.Recommendation `json:"recommendations"`
}

func (s *Server) handleGetRecommendations(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// Get region from household
	region := s.getRegion()

	// Fetch prices
	client := prices.NewOctopusClient(region)
	priceSlots, err := client.FetchTodayAndTomorrow(ctx, region)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch prices: "+err.Error())
		return
	}

	// Get household
	household, err := s.store.GetHousehold("default")
	if err != nil {
		respondError(w, http.StatusNotFound, "household not found")
		return
	}

	// Get appliances
	appliances, err := s.store.GetAppliances("default")
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Generate recommendations
	results := []RecommendationResponse{}
	currentDate := time.Now().Format("2006-01-02")

	for _, a := range appliances {
		if !a.Enabled {
			continue
		}

		// Check if we should show recommendation based on usage frequency
		if !engine.ShouldShowRecommendation(a, "", currentDate) {
			continue
		}

		constraints := engine.Constraints{
			Allowed:       a.AllowedWindows,
			Blocked:       a.BlockedWindows,
			QuietHours:    household.QuietHours,
			FinishBy:      a.FinishBy,
			StartBy:       a.StartBy,
			PriceCapPence: a.PriceCapPencePerKWh,
			NoiseLevel:    a.NoiseLevel,
		}

		// Apply practical constraints based on control type
		engine.ApplyPracticalConstraints(a, household, &constraints)

		opts := engine.Options{
			EstKWh:       a.EstKWh,
			CarbonWeight: household.CarbonWeight,
		}

		// Get recommendations for remaining TODAY and TOMORROW separately
		now := time.Now()
		todayEnd := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())

		// Split slots into today and tomorrow
		todaySlots := []engine.PriceSlot{}
		tomorrowSlots := []engine.PriceSlot{}
		for _, slot := range priceSlots {
			if slot.Start.After(now) && slot.Start.Before(todayEnd) {
				todaySlots = append(todaySlots, slot)
			} else if slot.Start.After(todayEnd) {
				tomorrowSlots = append(tomorrowSlots, slot)
			}
		}

		var bestRecs []engine.Recommendation

		// Get best for today (if any slots left today)
		if len(todaySlots) > 0 {
			todayRecs, err := engine.BestWindows(todaySlots, a.CycleMinutes, constraints, opts, 1)
			if err == nil && len(todayRecs) > 0 {
				bestRecs = append(bestRecs, todayRecs...)
			}
		}

		// Get best for tomorrow
		if len(tomorrowSlots) > 0 {
			tomorrowRecs, err := engine.BestWindows(tomorrowSlots, a.CycleMinutes, constraints, opts, 1)
			if err == nil && len(tomorrowRecs) > 0 {
				bestRecs = append(bestRecs, tomorrowRecs...)
			}
		}

		// Skip if no recommendations
		if len(bestRecs) == 0 {
			continue
		}

		results = append(results, RecommendationResponse{
			Appliance:       a.Name,
			Recommendations: bestRecs,
		})
	}

	respondJSON(w, http.StatusOK, results)
}

func (s *Server) handleSmartRecommendations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get household with location
	household, err := s.store.GetHousehold("default")
	if err != nil {
		respondError(w, http.StatusNotFound, "household not found")
		return
	}

	// Get all appliances
	appliances, err := s.store.GetAppliances("default")
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Fetch weather forecast for next 3 days
	weatherClient := weather.NewForecastClient(household.Latitude, household.Longitude)
	forecasts, err := weatherClient.GetForecast(ctx, 3)
	if err != nil {
		// Continue without weather if forecast fails
		forecasts = []engine.WeatherForecast{}
	}

	// Map forecasts by date
	weatherByDay := make(map[string]*engine.WeatherForecast)
	for i := range forecasts {
		dateStr := forecasts[i].Date.Format("2006-01-02")
		weatherByDay[dateStr] = &forecasts[i]
	}

	// Fetch prices for next 3 days
	region := s.getRegion()
	pricesClient := prices.NewOctopusClient(region)

	pricesByDay := make(map[string][]engine.PriceSlot)
	for dayOffset := 0; dayOffset < 3; dayOffset++ {
		day := time.Now().AddDate(0, 0, dayOffset)
		dateStr := day.Format("2006-01-02")

		dayPrices, err := pricesClient.HalfHourly(ctx, day, region)
		if err == nil {
			pricesByDay[dateStr] = dayPrices
		}
	}

	// Generate smart recommendations for coupled appliances only
	smartResults := []engine.SmartRecommendation{}

	for _, a := range appliances {
		if !a.Enabled || a.Class != engine.ClassCoupled {
			continue
		}

		// Check if we should show recommendation based on usage frequency
		currentDate := time.Now().Format("2006-01-02")
		if !engine.ShouldShowRecommendation(a, "", currentDate) {
			continue
		}

		// Find coupled appliance (dryer)
		var coupledAppliance *engine.Appliance
		if a.CoupledApplianceID != "" {
			for _, ca := range appliances {
				if ca.ID == a.CoupledApplianceID {
					coupledAppliance = ca
					break
				}
			}
		}

		// Build constraints
		constraints := engine.Constraints{
			Allowed:       a.AllowedWindows,
			Blocked:       a.BlockedWindows,
			QuietHours:    household.QuietHours,
			FinishBy:      a.FinishBy,
			StartBy:       a.StartBy,
			PriceCapPence: a.PriceCapPencePerKWh,
			NoiseLevel:    a.NoiseLevel,
		}

		// Apply practical constraints
		engine.ApplyPracticalConstraints(a, household, &constraints)

		opts := engine.Options{
			EstKWh:       a.EstKWh,
			CarbonWeight: household.CarbonWeight,
		}

		// Generate smart recommendations
		smartRec, err := engine.GenerateSmartRecommendations(
			a, coupledAppliance, pricesByDay, weatherByDay, household, constraints, opts)

		if err == nil && smartRec != nil {
			// Filter out past options
			now := time.Now()
			futureOptions := []engine.RecommendationOption{}
			for _, opt := range smartRec.Options {
				if opt.PrimarySlot.Start.After(now) {
					futureOptions = append(futureOptions, opt)
				}
			}

			// Only include if there are future options
			if len(futureOptions) > 0 {
				smartRec.Options = futureOptions
				// Recalculate best option index
				if smartRec.BestOptionIndex >= len(futureOptions) {
					smartRec.BestOptionIndex = 0
				}
				smartResults = append(smartResults, *smartRec)
			}
		}
	}

	respondJSON(w, http.StatusOK, smartResults)
}

func (s *Server) handleGetWeather(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get household location
	household, err := s.store.GetHousehold("default")
	if err != nil {
		respondError(w, http.StatusNotFound, "household not found")
		return
	}

	// Fetch 3-day weather forecast
	weatherClient := weather.NewForecastClient(household.Latitude, household.Longitude)
	forecasts, err := weatherClient.GetForecast(ctx, 3)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch weather: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, forecasts)
}

func (s *Server) serveUI(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/index.html")
}

func (s *Server) serveStatic(w http.ResponseWriter, r *http.Request) {
	// Disable caching for development
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))).ServeHTTP(w, r)
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
