package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/awaistahir/smart-run/internal/engine"
	"github.com/awaistahir/smart-run/internal/prices"
	"github.com/awaistahir/smart-run/internal/store"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	dbPath  string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "smart-run",
		Short: "SmartRun - Optimize when to run appliances based on energy prices",
		Long: `SmartRun helps you save money by finding the cheapest times to run
your household appliances based on Octopus Agile pricing.`,
	}

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.smartrun/config.yaml)")
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", "", "database path (default is $HOME/.smartrun/smartrun.db)")

	cobra.OnInitialize(initConfig)

	rootCmd.AddCommand(fetchCmd())
	rootCmd.AddCommand(planCmd())
	rootCmd.AddCommand(initCmd())
	rootCmd.AddCommand(applianceCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		configDir := filepath.Join(home, ".smartrun")
		os.MkdirAll(configDir, 0755)

		viper.AddConfigPath(configDir)
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	viper.AutomaticEnv()
	viper.ReadInConfig()

	// Set defaults
	if dbPath == "" {
		home, _ := os.UserHomeDir()
		dbPath = filepath.Join(home, ".smartrun", "smartrun.db")
	}
}

func fetchCmd() *cobra.Command {
	var region string
	var date string

	cmd := &cobra.Command{
		Use:   "fetch",
		Short: "Fetch energy prices from Octopus Agile",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			client := prices.NewOctopusClient(region)

			var priceSlots []engine.PriceSlot
			var err error

			if date == "today" {
				priceSlots, err = client.FetchTodayAndTomorrow(ctx, region)
			} else {
				day, parseErr := time.Parse("2006-01-02", date)
				if parseErr != nil {
					return fmt.Errorf("invalid date format (use YYYY-MM-DD): %w", parseErr)
				}
				priceSlots, err = client.HalfHourly(ctx, day, region)
			}

			if err != nil {
				return err
			}

			// Output as JSON
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(priceSlots)
		},
	}

	cmd.Flags().StringVarP(&region, "region", "r", "C", "Octopus region (A-P)")
	cmd.Flags().StringVarP(&date, "date", "d", "today", "Date to fetch (YYYY-MM-DD or 'today')")

	return cmd
}

func planCmd() *cobra.Command {
	var region string
	var lat, lon float64
	var applianceID string

	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Generate optimal schedule for appliances",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			// Open store
			st, err := store.NewStore(dbPath)
			if err != nil {
				return fmt.Errorf("opening database: %w", err)
			}
			defer st.Close()

			// Fetch prices
			pricesClient := prices.NewOctopusClient(region)
			priceSlots, err := pricesClient.FetchTodayAndTomorrow(ctx, region)
			if err != nil {
				return fmt.Errorf("fetching prices: %w", err)
			}

			fmt.Fprintf(os.Stderr, "Fetched %d price slots\n", len(priceSlots))

			// Get household
			household, err := st.GetHousehold("default")
			if err != nil {
				return fmt.Errorf("getting household: %w (run 'smart-run init' first)", err)
			}

			// Get appliances
			appliances, err := st.GetAppliances(household.ID)
			if err != nil {
				return fmt.Errorf("getting appliances: %w", err)
			}

			if len(appliances) == 0 {
				return fmt.Errorf("no appliances configured (use 'smart-run appliance add')")
			}

			// Filter if specific appliance requested
			if applianceID != "" {
				filtered := []*engine.Appliance{}
				for _, a := range appliances {
					if a.ID == applianceID {
						filtered = append(filtered, a)
					}
				}
				appliances = filtered
				if len(appliances) == 0 {
					return fmt.Errorf("appliance not found: %s", applianceID)
				}
			}

			// Generate recommendations for each appliance
			type applianceRec struct {
				Appliance       string                   `json:"appliance"`
				Recommendations []engine.Recommendation  `json:"recommendations"`
			}

			results := []applianceRec{}

			for _, a := range appliances {
				if !a.Enabled {
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

				opts := engine.Options{
					EstKWh:       a.EstKWh,
					CarbonWeight: household.CarbonWeight,
				}

				recs, err := engine.BestWindows(priceSlots, a.CycleMinutes, constraints, opts, 3)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: %s - %v\n", a.Name, err)
					continue
				}

				results = append(results, applianceRec{
					Appliance:       a.Name,
					Recommendations: recs,
				})
			}

			// Output as JSON
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(results)
		},
	}

	cmd.Flags().StringVarP(&region, "region", "r", "C", "Octopus region")
	cmd.Flags().Float64Var(&lat, "lat", 51.5074, "Latitude for weather")
	cmd.Flags().Float64Var(&lon, "lon", -0.1278, "Longitude for weather")
	cmd.Flags().StringVarP(&applianceID, "appliance", "a", "", "Specific appliance ID (optional)")

	return cmd
}

func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize SmartRun with default household",
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := store.NewStore(dbPath)
			if err != nil {
				return err
			}
			defer st.Close()

			household := &engine.Household{
				ID:        "default",
				Name:      "My Household",
				Region:    "C",       // Default to London region
				Latitude:  51.5074,   // Default to London latitude
				Longitude: -0.1278,   // Default to London longitude
				QuietHours: []engine.TimeWindow{
					{Start: "22:00", End: "07:00", DaysOfWeek: []int{1, 2, 3, 4, 5, 6, 7}},
				},
				StaggerHeavyLoads: true,
				CarbonWeight:      0.0,
			}

			if err := st.SaveHousehold(household); err != nil {
				return err
			}

			fmt.Println("✓ Initialized default household")
			fmt.Printf("Database: %s\n", dbPath)
			fmt.Println("\nNext steps:")
			fmt.Println("  1. Add appliances: smart-run appliance add")
			fmt.Println("  2. Generate plan: smart-run plan")

			return nil
		},
	}
}

func applianceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "appliance",
		Short: "Manage appliances",
	}

	cmd.AddCommand(applianceAddCmd())
	cmd.AddCommand(applianceListCmd())

	return cmd
}

func applianceAddCmd() *cobra.Command {
	var name string
	var cycleMin int
	var estKWh float64
	var noiseLevel int
	var priority int

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new appliance",
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := store.NewStore(dbPath)
			if err != nil {
				return err
			}
			defer st.Close()

			appliance := &engine.Appliance{
				ID:           fmt.Sprintf("%s-%d", name, time.Now().Unix()),
				Name:         name,
				CycleMinutes: cycleMin,
				EstKWh:       estKWh,
				NoiseLevel:   noiseLevel,
				Priority:     priority,
				Enabled:      true,
			}

			if err := st.SaveAppliance(appliance, "default"); err != nil {
				return err
			}

			fmt.Printf("✓ Added appliance: %s\n", name)
			fmt.Printf("  ID: %s\n", appliance.ID)
			fmt.Printf("  Cycle: %d minutes\n", cycleMin)
			fmt.Printf("  Est. consumption: %.2f kWh\n", estKWh)

			return nil
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Appliance name (required)")
	cmd.Flags().IntVarP(&cycleMin, "cycle", "c", 60, "Cycle duration in minutes")
	cmd.Flags().Float64VarP(&estKWh, "kwh", "k", 1.0, "Estimated kWh consumption")
	cmd.Flags().IntVar(&noiseLevel, "noise", 3, "Noise level (1-5)")
	cmd.Flags().IntVar(&priority, "priority", 3, "Priority (1-5)")

	cmd.MarkFlagRequired("name")

	return cmd
}

func applianceListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all appliances",
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := store.NewStore(dbPath)
			if err != nil {
				return err
			}
			defer st.Close()

			appliances, err := st.GetAppliances("default")
			if err != nil {
				return err
			}

			if len(appliances) == 0 {
				fmt.Println("No appliances configured")
				return nil
			}

			fmt.Printf("%-30s %-15s %10s %10s %8s\n", "NAME", "ID", "CYCLE", "KWH", "ENABLED")
			fmt.Println("--------------------------------------------------------------------------------")

			for _, a := range appliances {
				enabled := "Yes"
				if !a.Enabled {
					enabled = "No"
				}
				fmt.Printf("%-30s %-15s %8dm %9.2f %8s\n",
					a.Name, a.ID[:min(15, len(a.ID))], a.CycleMinutes, a.EstKWh, enabled)
			}

			return nil
		},
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
