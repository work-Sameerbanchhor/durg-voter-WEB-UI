package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"durg-voter-api/pkg/db"
	"durg-voter-api/pkg/handlers"
	"durg-voter-api/pkg/models"
	"durg-voter-api/pkg/repository"
	"durg-voter-api/pkg/service"
)

func setupTestServer(t *testing.T) (*db.DuckDB, http.Handler) {
	dbPath := "../../database/durg_voters.duckdb"
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		dbPath = "database/durg_voters.duckdb"
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
	mux.HandleFunc("GET /api/v1/constituencies", h.ListConstituenciesHandler)
	mux.HandleFunc("POST /api/v1/voters/search", h.SearchVotersHandler)

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

func TestAuthLogin(t *testing.T) {
	duckDB, _ := setupTestServer(t)
	defer duckDB.Close()

	repo := repository.NewVoterRepository(duckDB)
	svc := service.NewVoterService(repo)

	// Test Admin Login
	adminResp, err := svc.AuthenticateUser("admin", "adminpass")
	if err != nil || adminResp.Role != models.RoleAdmin {
		t.Fatalf("expected admin authentication success, got err: %v", err)
	}

	// Test Guest Login
	guestResp, err := svc.AuthenticateUser("guest", "guestpass")
	if err != nil || guestResp.Role != models.RoleGuest {
		t.Fatalf("expected guest authentication success, got err: %v", err)
	}
}

func TestAdminExecuteSQL(t *testing.T) {
	duckDB, _ := setupTestServer(t)
	defer duckDB.Close()

	repo := repository.NewVoterRepository(duckDB)
	svc := service.NewVoterService(repo)

	ctx := context.Background()
	sqlRes, err := svc.ExecuteSQL(ctx, "SELECT gender_english, count(*) FROM voters GROUP BY 1;")
	if err != nil {
		t.Fatalf("expected SQL execution success for admin, got err: %v", err)
	}
	if len(sqlRes.Columns) != 2 {
		t.Errorf("expected 2 columns, got %d", len(sqlRes.Columns))
	}
}

func TestGroupByNames(t *testing.T) {
	duckDB, _ := setupTestServer(t)
	defer duckDB.Close()

	repo := repository.NewVoterRepository(duckDB)
	svc := service.NewVoterService(repo)

	ctx := context.Background()
	res, err := svc.GroupBy(ctx, models.GroupByRequest{Field: "full_name", Limit: 5})
	if err != nil {
		t.Fatalf("expected GroupBy success, got err: %v", err)
	}
	if len(res.Groups) == 0 {
		t.Errorf("expected non-empty group results")
	}
}

func TestGeoNearby(t *testing.T) {
	duckDB, _ := setupTestServer(t)
	defer duckDB.Close()

	repo := repository.NewVoterRepository(duckDB)
	svc := service.NewVoterService(repo)

	ctx := context.Background()
	req := models.GeoNearbyRequest{Latitude: 21.19, Longitude: 81.28, RadiusKM: 10.0, Limit: 5}

	stations, err := svc.GetNearbyPollingStations(ctx, req)
	if err != nil {
		t.Fatalf("expected nearby polling stations success, got err: %v", err)
	}
	if len(stations) == 0 {
		t.Errorf("expected nearby polling stations")
	}

	dist := svc.CalculateDistance(21.19, 81.28, 21.21, 81.38)
	if dist.DistanceKM <= 0 {
		t.Errorf("expected calculated distance > 0, got %f", dist.DistanceKM)
	}
}

func TestListConstituencies(t *testing.T) {
	duckDB, handler := setupTestServer(t)
	defer duckDB.Close()

	req := httptest.NewRequest("GET", "/api/v1/constituencies", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp struct {
		Success bool                       `json:"success"`
		Data    []models.ConstituencySummary `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !resp.Success {
		t.Errorf("expected success true, got false")
	}
	for _, item := range resp.Data {
		t.Logf("Constituency in DB: '%s' (Total: %d)", item.AssemblyConstituency, item.TotalVoters)
	}
	if len(resp.Data) == 0 {
		t.Errorf("expected constituencies list to be non-empty")
	}
}

func TestSearchVoters(t *testing.T) {
	duckDB, handler := setupTestServer(t)
	defer duckDB.Close()

	body := `{"query":"Patel", "min_age":18, "max_age":45, "limit":5}`
	req := httptest.NewRequest("POST", "/api/v1/voters/search", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body: %s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Success bool           `json:"success"`
		Data    []models.Voter `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	t.Logf("Search returned %d voters", len(resp.Data))
}

func TestGetPartDetails(t *testing.T) {
	duckDB, _ := setupTestServer(t)
	defer duckDB.Close()

	ctx := context.Background()
	repo := repository.NewVoterRepository(duckDB)
	svc := service.NewVoterService(repo)

	pd, err := svc.GetPartDetails(ctx, "vaishali-nagar", 293)
	if err != nil {
		t.Fatalf("expected GetPartDetails success, got err: %v", err)
	}
	if pd == nil {
		t.Fatalf("expected non-nil PartDetails")
	}

	if len(pd.Sections) < 5 {
		t.Errorf("expected at least 5 sections for vaishali-nagar part 293, got %d", len(pd.Sections))
	}
}

