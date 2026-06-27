package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/mohit838/mtz-go-migrator/test/internal/config"
	"github.com/mohit838/mtz-go-migrator/test/internal/database"
	"github.com/mohit838/mtz-go-migrator/test/internal/router"
)

func main() {
	// ========================
	// Migrator Test Service
	// ========================
	// This service exists as a testing ground for the libs/migrator library.
	// It keeps a minimal footprint: Postgres + chi router + health checks.
	// No auth, no gRPC, no Redis, no MongoDB, no MinIO.
	fmt.Println("\n=== Migrator Test Service ===")

	// Load configuration from .env
	cfg, err := config.LoadConfig("./.env")
	if err != nil {
		log.Println("Error loading config:", err)
		return
	}

	fmt.Printf("App: %s | Env: %s | Port: %s | Debug: %v\n\n",
		cfg.AppName, cfg.AppEnv, cfg.AppPort, cfg.AppDebug)

	// ========================
	// Database
	// ========================
	db, err := database.ConnectDB(cfg.DBURL)
	if err != nil {
		log.Fatalf("error connecting to postgres: %v", err)
	}
	defer db.Close()
	log.Println(">>-->> PostgreSQL connected")

	// ========================
	// HTTP Server
	// ========================
	handler := router.NewRouter(db)

	port := ":" + cfg.AppPort
	log.Printf("Server starting on port %s...\n", cfg.AppPort)
	if err := http.ListenAndServe(port, handler); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
