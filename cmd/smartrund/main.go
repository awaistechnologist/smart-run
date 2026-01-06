package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/awaistahir/smart-run/internal/store"
	"github.com/awaistahir/smart-run/internal/uiapi"
	"github.com/spf13/cobra"
)

func main() {
	var port int
	var dbPath string

	rootCmd := &cobra.Command{
		Use:   "smartrund",
		Short: "SmartRun HTTP server with web UI",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Set default db path
			if dbPath == "" {
				home, _ := os.UserHomeDir()
				dbPath = filepath.Join(home, ".smartrun", "smartrun.db")
			}

			// Open store
			st, err := store.NewStore(dbPath)
			if err != nil {
				return fmt.Errorf("opening database: %w", err)
			}
			defer st.Close()

			// Create server
			srv := uiapi.NewServer(st)

			// Start server
			addr := fmt.Sprintf(":%d", port)
			log.Printf("SmartRun UI server starting on port %d", port)
			log.Printf("Database: %s", dbPath)
			log.Println("\nAccess from this device: http://localhost:8080")
			log.Println("Access from mobile/other devices: http://YOUR_LOCAL_IP:8080")
			log.Println("Configure your region in Settings")

			return http.ListenAndServe(addr, srv.Handler())
		},
	}

	rootCmd.Flags().IntVarP(&port, "port", "p", 8080, "HTTP port")
	rootCmd.Flags().StringVar(&dbPath, "db", "", "Database path")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
