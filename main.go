package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"durg-voter-api/pkg/config"
	"durg-voter-api/pkg/db"
	"durg-voter-api/pkg/handlers"
	"durg-voter-api/pkg/middleware"
	"durg-voter-api/pkg/repository"
	"durg-voter-api/pkg/service"
)

func main() {
	// 1. Load Environment Configuration
	cfg := config.LoadConfig()
	log.Printf("Initializing Durg Voter REST API [Env: %s, DB: %s]", cfg.Environment, cfg.DBPath)

	// 2. Initialize DuckDB Connection Pool
	duckDB, err := db.NewDuckDB(cfg.DBPath)
	if err != nil {
		log.Fatalf("Fatal: Failed to connect to DuckDB: %v", err)
	}
	defer func() {
		if err := duckDB.Close(); err != nil {
			log.Printf("Error closing DuckDB: %v", err)
		}
	}()

	// 3. Initialize Repository, Service, and Handlers
	repo := repository.NewVoterRepository(duckDB)
	voterService := service.NewVoterService(repo)
	h := handlers.NewHandler(voterService, duckDB)

	// 4. Configure HTTP Routes
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", h.APIDocsHandler)
	mux.HandleFunc("GET /api/v1/health", h.HealthCheckHandler)
	mux.HandleFunc("GET /api/v1/stats", h.GetStatsHandler)
	mux.HandleFunc("GET /api/v1/voters", h.ListVotersHandler)
	mux.HandleFunc("GET /api/v1/voters/{epic_no}", h.GetVoterByIDHandler)
	mux.HandleFunc("POST /api/v1/voters/search", h.SearchVotersHandler)
	mux.HandleFunc("GET /api/v1/voters/group-by", h.GroupByHandler)
	mux.HandleFunc("POST /api/v1/voters/group-by", h.GroupByHandler)
	mux.HandleFunc("GET /api/v1/polling-stations", h.ListPollingStationsHandler)
	mux.HandleFunc("GET /api/v1/polling-stations/detail", h.GetPollingStationHandler)
	mux.HandleFunc("GET /api/v1/constituencies", h.ListConstituenciesHandler)
	mux.HandleFunc("GET /api/v1/openapi.json", h.OpenAPIHandler)

	// Auth & Role-based Access Endpoints
	mux.HandleFunc("POST /api/v1/auth/login", h.LoginHandler)
	mux.HandleFunc("GET /api/v1/auth/me", h.MeHandler)

	// Admin-Only Raw SQL Execution
	mux.HandleFunc("POST /api/v1/admin/sql", h.ExecuteSQLHandler)
	mux.HandleFunc("POST /api/v1/sql", h.ExecuteSQLHandler)

	// Geolocation & Spatial Proximity Endpoints
	mux.HandleFunc("GET /api/v1/geo/nearby-polling-stations", h.GeoNearbyPollingStationsHandler)
	mux.HandleFunc("GET /api/v1/geo/nearby-voters", h.GeoNearbyVotersHandler)
	mux.HandleFunc("GET /api/v1/geo/distance", h.GeoDistanceHandler)

	// 5. Build Middleware Stack: RequestID -> Recovery -> CORS -> SecurityHeaders -> Authenticate -> Logger -> RateLimiter
	rateLimiter := middleware.NewIPRateLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst)

	var handler http.Handler = mux
	handler = middleware.Authenticate(handler)
	handler = middleware.RateLimit(rateLimiter)(handler)
	handler = middleware.Logger(handler)
	handler = middleware.SecurityHeaders(handler)
	handler = middleware.CORS(cfg.CORSAllowedOrigins)(handler)
	handler = middleware.Recovery(handler)
	handler = middleware.RequestID(handler)

	// 6. Configure HTTP Server
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	// 7. Graceful Shutdown Setup
	serverCtx, serverStopCtx := context.WithCancel(context.Background())

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		<-sig
		shutdownCtx, cancel := context.WithTimeout(serverCtx, 30*time.Second)
		defer cancel()

		log.Println("Received termination signal. Shutting down Go HTTP Server gracefully...")
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Fatalf("Server shutdown forced with error: %v", err)
		}
		serverStopCtx()
	}()

	// 8. Start HTTP Server
	log.Printf("==================================================================")
	log.Printf("🚀 Durg Voter Production API active on port %s", cfg.Port)
	log.Printf("📊 Engine             : DuckDB Vectorized SQL Engine")
	log.Printf("📍 Dashboard UI       : http://localhost:%s/", cfg.Port)
	log.Printf("📍 Health Check       : http://localhost:%s/api/v1/health", cfg.Port)
	log.Printf("📍 Electorate Stats   : http://localhost:%s/api/v1/stats", cfg.Port)
	log.Printf("📍 Voters Listing     : http://localhost:%s/api/v1/voters", cfg.Port)
	log.Printf("📍 Polling Stations   : http://localhost:%s/api/v1/polling-stations", cfg.Port)
	log.Printf("📍 Constituencies     : http://localhost:%s/api/v1/constituencies", cfg.Port)
	log.Printf("📍 OpenAPI Spec       : http://localhost:%s/api/v1/openapi.json", cfg.Port)
	log.Printf("==================================================================")

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed to start: %v", err)
	}

	<-serverCtx.Done()
	log.Println("Server exited cleanly.")
}
