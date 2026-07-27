package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"durg-voter-api/pkg/db"
	"durg-voter-api/pkg/handlers"
	"durg-voter-api/pkg/models"
	"durg-voter-api/pkg/repository"
	"durg-voter-api/pkg/service"
)

func setupTestServer(t *testing.T) (*db.DuckDB, http.Handler) {
	dbPath := "../../dataset/durg_voters.duckdb"
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		dbPath = "dataset/durg_voters.duckdb"
	}

	duckDB, err := db.NewDuckDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize DuckDB for test: %v", err)
	}

	repo := repository.NewVoterRepository(duckDB)
	svc := service.NewVoterService(repo)
	h := handlers.NewHandler(svc, duckDB)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", h.HealthCheckHandler)
	mux.HandleFunc("GET /api/v1/stats", h.GetStatsHandler)
	mux.HandleFunc("GET /api/v1/voters", h.ListVotersHandler)
	mux.HandleFunc("GET /api/v1/voters/{epic_no}", h.GetVoterByIDHandler)

	return duckDB, mux
}

func TestHealthCheck(t *testing.T) {
	duckDB, handler := setupTestServer(t)
	defer duckDB.Close()

	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp models.APIResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !resp.Success {
		t.Errorf("expected success true, got false")
	}
}

func TestGetStats(t *testing.T) {
	duckDB, handler := setupTestServer(t)
	defer duckDB.Close()

	req := httptest.NewRequest("GET", "/api/v1/stats", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp struct {
		Success bool                `json:"success"`
		Data    models.StatsSummary `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Data.TotalVoters <= 1000000 {
		t.Errorf("expected total voters > 1,000,000, got %d", resp.Data.TotalVoters)
	}
	if resp.Data.TotalBooths <= 1000 {
		t.Errorf("expected total booths > 1,000, got %d", resp.Data.TotalBooths)
	}
}

func TestListVotersPagination(t *testing.T) {
	duckDB, handler := setupTestServer(t)
	defer duckDB.Close()

	req := httptest.NewRequest("GET", "/api/v1/voters?limit=5&page=1", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp struct {
		Success bool           `json:"success"`
		Data    []models.Voter `json:"data"`
		Meta    models.Pagination `json:"meta"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Data) != 5 {
		t.Errorf("expected 5 voters, got %d", len(resp.Data))
	}
	if resp.Meta.TotalItems <= 1000000 {
		t.Errorf("expected total items > 1,000,000, got %d", resp.Meta.TotalItems)
	}
}
