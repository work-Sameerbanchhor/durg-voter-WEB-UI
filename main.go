package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"durg-voter-api/pkg/handlers"
	"durg-voter-api/pkg/middleware"
)

func main() {
	// Configure port
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Initialize Go 1.22 HTTP ServeMux
	mux := http.NewServeMux()

	// Register routes with HTTP method matching
	mux.HandleFunc("GET /", handlers.APIDocsHandler)
	mux.HandleFunc("GET /api/v1/health", handlers.HealthCheckHandler)
	mux.HandleFunc("GET /api/v1/stats", handlers.GetStatsHandler)
	mux.HandleFunc("GET /api/v1/voters", handlers.ListVotersHandler)
	mux.HandleFunc("GET /api/v1/voters/{epic_no}", handlers.GetVoterByIDHandler)
	mux.HandleFunc("POST /api/v1/voters/search", handlers.SearchVotersHandler)

	// Apply Middleware Stack: Logger -> SecurityHeaders -> CORS -> Recovery
	var handler http.Handler = mux
	handler = middleware.Logger(handler)
	handler = middleware.SecurityHeaders(handler)
	handler = middleware.CORS(handler)
	handler = middleware.Recovery(handler)

	// Configure HTTP Server
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Server run context for graceful shutdown
	serverCtx, serverStopCtx := context.WithCancel(context.Background())

	// Listen for syscall signals for process termination
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		<-sig

		// Shutdown grace period of 30 seconds
		shutdownCtx, cancel := context.WithTimeout(serverCtx, 30*time.Second)
		defer cancel()

		log.Println("Shutting down Go Web Server gracefully...")
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Fatalf("Server shutdown forced with error: %v", err)
		}
		serverStopCtx()
	}()

	// Start server
	log.Printf("==================================================")
	log.Printf("🚀 Durg Voter Go Web Server API starting on port %s", port)
	log.Printf("📍 Dashboard & Docs : http://localhost:%s/", port)
	log.Printf("📍 Health Endpoint  : http://localhost:%s/api/v1/health", port)
	log.Printf("📍 API Endpoints   : http://localhost:%s/api/v1/voters", port)
	log.Printf("==================================================")

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed to start: %v", err)
	}

	<-serverCtx.Done()
	log.Println("Server exited cleanly.")
}
