package handlers

import (
	"encoding/json"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	"durg-voter-api/pkg/db"
	"durg-voter-api/pkg/middleware"
	"durg-voter-api/pkg/models"
	"durg-voter-api/pkg/service"
)

var startTime = time.Now()

type Handler struct {
	voterService service.VoterService
	duckDB       *db.DuckDB
}

func NewHandler(voterService service.VoterService, duckDB *db.DuckDB) *Handler {
	return &Handler{
		voterService: voterService,
		duckDB:       duckDB,
	}
}

func SendJSON(w http.ResponseWriter, r *http.Request, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	reqID := middleware.GetRequestID(r.Context())

	switch resp := data.(type) {
	case models.APIResponse:
		resp.RequestID = reqID
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(resp)
	default:
		apiResp := models.APIResponse{
			Success:   statusCode >= 200 && statusCode < 300,
			Data:      data,
			RequestID: reqID,
		}
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(apiResp)
	}
}

func (h *Handler) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	dbStatus := "healthy"
	if err := h.duckDB.Ping(ctx); err != nil {
		dbStatus = "unhealthy: " + err.Error()
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	status := models.HealthStatus{
		Status:        "healthy",
		Database:      dbStatus,
		DatabasePath:  h.duckDB.DBPath,
		Uptime:        time.Since(startTime).String(),
		Timestamp:     time.Now(),
		GoVersion:     runtime.Version(),
		AppName:       "Durg Voter REST API",
		Version:       "2.0.0-production",
		MemoryAllocMB: float64(m.Alloc) / 1024 / 1024,
		Goroutines:    runtime.NumGoroutine(),
	}

	SendJSON(w, r, http.StatusOK, models.APIResponse{
		Success: true,
		Message: "DuckDB backend operating nominally",
		Data:    status,
	})
}

func (h *Handler) GetStatsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	stats, err := h.voterService.GetStats(ctx)
	if err != nil {
		SendJSON(w, r, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	SendJSON(w, r, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    stats,
	})
}

func (h *Handler) ListVotersHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	filter := models.SearchFilter{
		Query:                strings.TrimSpace(r.URL.Query().Get("search")),
		AssemblyConstituency: strings.TrimSpace(r.URL.Query().Get("assembly")),
		Gender:               strings.TrimSpace(r.URL.Query().Get("gender")),
		TownVillage:          strings.TrimSpace(r.URL.Query().Get("town")),
		SectionNumberAndName: strings.TrimSpace(r.URL.Query().Get("section")),
		SortBy:               strings.TrimSpace(r.URL.Query().Get("sort_by")),
		SortOrder:            strings.TrimSpace(r.URL.Query().Get("sort_order")),
	}
	if filter.SectionNumberAndName == "" {
		filter.SectionNumberAndName = strings.TrimSpace(r.URL.Query().Get("section_number_and_name"))
	}

	if minAgeStr := r.URL.Query().Get("min_age"); minAgeStr != "" {
		filter.MinAge, _ = strconv.Atoi(minAgeStr)
	}
	if maxAgeStr := r.URL.Query().Get("max_age"); maxAgeStr != "" {
		filter.MaxAge, _ = strconv.Atoi(maxAgeStr)
	}
	if partStr := r.URL.Query().Get("part_number"); partStr != "" {
		p, _ := strconv.ParseInt(partStr, 10, 64)
		filter.PartNumber = p
	}

	filter.Page, _ = strconv.Atoi(r.URL.Query().Get("page"))
	filter.Limit, _ = strconv.Atoi(r.URL.Query().Get("limit"))

	voters, meta, err := h.voterService.ListVoters(ctx, filter)
	if err != nil {
		SendJSON(w, r, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	SendJSON(w, r, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    voters,
		Meta:    meta,
	})
}

func (h *Handler) GetVoterByIDHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	epicNo := r.PathValue("epic_no")
	if epicNo == "" {
		epicNo = r.URL.Query().Get("epic_no")
	}

	if epicNo == "" {
		SendJSON(w, r, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "EPIC Number parameter is required",
		})
		return
	}

	voter, err := h.voterService.GetVoterByEPIC(ctx, epicNo)
	if err != nil {
		SendJSON(w, r, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	if voter == nil {
		SendJSON(w, r, http.StatusNotFound, models.APIResponse{
			Success: false,
			Error:   "Voter record not found with EPIC: " + epicNo,
		})
		return
	}

	SendJSON(w, r, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    voter,
	})
}

func (h *Handler) SearchVotersHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var filter models.SearchFilter
	if err := json.NewDecoder(r.Body).Decode(&filter); err != nil {
		SendJSON(w, r, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Invalid JSON payload: " + err.Error(),
		})
		return
	}

	voters, meta, err := h.voterService.ListVoters(ctx, filter)
	if err != nil {
		SendJSON(w, r, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	SendJSON(w, r, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    voters,
		Meta:    meta,
	})
}

func (h *Handler) ListPollingStationsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	filter := models.SearchFilter{
		Query:                strings.TrimSpace(r.URL.Query().Get("search")),
		AssemblyConstituency: strings.TrimSpace(r.URL.Query().Get("assembly")),
	}
	filter.Page, _ = strconv.Atoi(r.URL.Query().Get("page"))
	filter.Limit, _ = strconv.Atoi(r.URL.Query().Get("limit"))

	stations, meta, err := h.voterService.ListPollingStations(ctx, filter)
	if err != nil {
		SendJSON(w, r, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	SendJSON(w, r, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    stations,
		Meta:    meta,
	})
}

func (h *Handler) GetPollingStationHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	assembly := r.URL.Query().Get("assembly")
	partStr := r.URL.Query().Get("part_number")
	partNo, _ := strconv.ParseInt(partStr, 10, 64)

	if assembly == "" || partNo <= 0 {
		SendJSON(w, r, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Assembly constituency and valid part_number query parameters are required",
		})
		return
	}

	station, err := h.voterService.GetPollingStation(ctx, assembly, partNo)
	if err != nil {
		SendJSON(w, r, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	if station == nil {
		SendJSON(w, r, http.StatusNotFound, models.APIResponse{
			Success: false,
			Error:   "Polling station not found",
		})
		return
	}

	SendJSON(w, r, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    station,
	})
}

func (h *Handler) GetPartDetailsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	assembly := r.URL.Query().Get("assembly")
	partStr := r.URL.Query().Get("part_number")
	partNo, _ := strconv.ParseInt(partStr, 10, 64)

	if assembly == "" || partNo <= 0 {
		SendJSON(w, r, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Assembly constituency and valid part_number query parameters are required",
		})
		return
	}

	details, err := h.voterService.GetPartDetails(ctx, assembly, partNo)
	if err != nil {
		SendJSON(w, r, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	if details == nil {
		SendJSON(w, r, http.StatusNotFound, models.APIResponse{
			Success: false,
			Error:   "Part details not found",
		})
		return
	}

	SendJSON(w, r, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    details,
	})
}

func (h *Handler) ListConstituenciesHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	constituencies, err := h.voterService.ListConstituencies(ctx)
	if err != nil {
		SendJSON(w, r, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	SendJSON(w, r, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    constituencies,
	})
}

func (h *Handler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendJSON(w, r, http.StatusMethodNotAllowed, models.APIResponse{Success: false, Error: "Method not allowed"})
		return
	}
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendJSON(w, r, http.StatusBadRequest, models.APIResponse{Success: false, Error: "Invalid JSON payload"})
		return
	}
	resp, err := h.voterService.AuthenticateUser(req.Username, req.Password)
	if err != nil {
		SendJSON(w, r, http.StatusUnauthorized, models.APIResponse{Success: false, Error: err.Error()})
		return
	}
	SendJSON(w, r, http.StatusOK, models.APIResponse{Success: true, Message: "Authentication successful", Data: resp})
}

func (h *Handler) MeHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		SendJSON(w, r, http.StatusUnauthorized, models.APIResponse{Success: false, Error: "Unauthorized"})
		return
	}
	SendJSON(w, r, http.StatusOK, models.APIResponse{Success: true, Data: user})
}

func (h *Handler) ExecuteSQLHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendJSON(w, r, http.StatusMethodNotAllowed, models.APIResponse{Success: false, Error: "Method not allowed"})
		return
	}

	user, _ := middleware.GetUserFromContext(r.Context())
	if user.Role != models.RoleAdmin {
		SendJSON(w, r, http.StatusForbidden, models.APIResponse{
			Success: false,
			Error:   "Forbidden: Only admin role is authorized to execute custom SQL queries. Guest role access is denied.",
		})
		return
	}

	var req models.SQLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendJSON(w, r, http.StatusBadRequest, models.APIResponse{Success: false, Error: "Invalid JSON payload"})
		return
	}

	result, err := h.voterService.ExecuteSQL(r.Context(), req.SQL)
	if err != nil {
		SendJSON(w, r, http.StatusBadRequest, models.APIResponse{Success: false, Error: err.Error()})
		return
	}

	SendJSON(w, r, http.StatusOK, models.APIResponse{
		Success: true,
		Message: "SQL query executed successfully",
		Data:    result,
	})
}

func (h *Handler) GroupByHandler(w http.ResponseWriter, r *http.Request) {
	var req models.GroupByRequest
	if r.Method == http.MethodPost {
		json.NewDecoder(r.Body).Decode(&req)
	} else {
		req.Field = r.URL.Query().Get("field")
		if lStr := r.URL.Query().Get("limit"); lStr != "" {
			req.Limit, _ = strconv.Atoi(lStr)
		}
		if mStr := r.URL.Query().Get("min_count"); mStr != "" {
			req.MinCount, _ = strconv.Atoi(mStr)
		}
		req.Sort = r.URL.Query().Get("sort")
	}

	result, err := h.voterService.GroupBy(r.Context(), req)
	if err != nil {
		SendJSON(w, r, http.StatusBadRequest, models.APIResponse{Success: false, Error: err.Error()})
		return
	}

	SendJSON(w, r, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    result,
	})
}

func (h *Handler) GeoNearbyPollingStationsHandler(w http.ResponseWriter, r *http.Request) {
	lat, _ := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
	lng, _ := strconv.ParseFloat(r.URL.Query().Get("lng"), 64)
	radius, _ := strconv.ParseFloat(r.URL.Query().Get("radius_km"), 64)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	if lat == 0 || lng == 0 {
		lat = 21.19
		lng = 81.28
	}

	req := models.GeoNearbyRequest{
		Latitude:  lat,
		Longitude: lng,
		RadiusKM:  radius,
		Limit:     limit,
	}

	stations, err := h.voterService.GetNearbyPollingStations(r.Context(), req)
	if err != nil {
		SendJSON(w, r, http.StatusBadRequest, models.APIResponse{Success: false, Error: err.Error()})
		return
	}

	SendJSON(w, r, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    stations,
	})
}

func (h *Handler) GeoNearbyVotersHandler(w http.ResponseWriter, r *http.Request) {
	lat, _ := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
	lng, _ := strconv.ParseFloat(r.URL.Query().Get("lng"), 64)
	radius, _ := strconv.ParseFloat(r.URL.Query().Get("radius_km"), 64)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	if lat == 0 || lng == 0 {
		lat = 21.19
		lng = 81.28
	}

	req := models.GeoNearbyRequest{
		Latitude:  lat,
		Longitude: lng,
		RadiusKM:  radius,
		Limit:     limit,
	}

	voters, err := h.voterService.GetNearbyVoters(r.Context(), req)
	if err != nil {
		SendJSON(w, r, http.StatusBadRequest, models.APIResponse{Success: false, Error: err.Error()})
		return
	}

	SendJSON(w, r, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    voters,
	})
}

func (h *Handler) GeoDistanceHandler(w http.ResponseWriter, r *http.Request) {
	lat1, _ := strconv.ParseFloat(r.URL.Query().Get("lat1"), 64)
	lng1, _ := strconv.ParseFloat(r.URL.Query().Get("lng1"), 64)
	lat2, _ := strconv.ParseFloat(r.URL.Query().Get("lat2"), 64)
	lng2, _ := strconv.ParseFloat(r.URL.Query().Get("lng2"), 64)

	res := h.voterService.CalculateDistance(lat1, lng1, lat2, lng2)
	SendJSON(w, r, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    res,
	})
}

func (h *Handler) OpenAPIHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(openAPISpec))
}

func (h *Handler) APIDocsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(dashboardHTML))
}

const openAPISpec = `{
  "openapi": "3.0.3",
  "info": {
    "title": "Durg Voter Production REST API",
    "description": "High-performance Go 1.24 & DuckDB 1.5 Vectorized Database RESTful API serving 1,045,426 electoral records.",
    "version": "2.0.0"
  },
  "servers": [
    {
      "url": "http://localhost:7860",
      "description": "Local production server"
    }
  ],
  "paths": {
    "/api/v1/health": {
      "get": {
        "summary": "System Health Check",
        "responses": { "200": { "description": "System status, memory metrics, and DuckDB health" } }
      }
    },
    "/api/v1/stats": {
      "get": {
        "summary": "Electorate Demographic Statistics",
        "responses": { "200": { "description": "Aggregated voter demographic counts" } }
      }
    },
    "/api/v1/voters": {
      "get": {
        "summary": "List & Filter Voters",
        "parameters": [
          { "name": "search", "in": "query", "schema": { "type": "string" } },
          { "name": "assembly", "in": "query", "schema": { "type": "string" } },
          { "name": "gender", "in": "query", "schema": { "type": "string" } },
          { "name": "page", "in": "query", "schema": { "type": "integer", "default": 1 } },
          { "name": "limit", "in": "query", "schema": { "type": "integer", "default": 20 } }
        ],
        "responses": { "200": { "description": "Paginated voter list" } }
      }
    },
    "/api/v1/voters/{epic_no}": {
      "get": {
        "summary": "Voter Profile by EPIC",
        "parameters": [
          { "name": "epic_no", "in": "path", "required": true, "schema": { "type": "string" } }
        ],
        "responses": {
          "200": { "description": "Voter detail profile" },
          "404": { "description": "Voter not found" }
        }
      }
    },
    "/api/v1/voters/search": {
      "post": {
        "summary": "Advanced JSON Multi-Criteria Voter Search",
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "properties": {
                  "query": { "type": "string" },
                  "assembly_constituency": { "type": "string" },
                  "gender": { "type": "string" },
                  "min_age": { "type": "integer" },
                  "max_age": { "type": "integer" },
                  "page": { "type": "integer" },
                  "limit": { "type": "integer" }
                }
              }
            }
          }
        },
        "responses": { "200": { "description": "Filtered voter results" } }
      }
    },
    "/api/v1/polling-stations": {
      "get": {
        "summary": "List Polling Stations",
        "responses": { "200": { "description": "List of polling station booths" } }
      }
    },
    "/api/v1/constituencies": {
      "get": {
        "summary": "Assembly Constituencies Overview",
        "responses": { "200": { "description": "Breakdown of assembly constituencies" } }
      }
    }
  }
}`

const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Durg Voter Production REST API & Embedded Dashboard</title>
    <link href="https://fonts.googleapis.com/css2?family=Outfit:wght@300;400;500;600;700;800&family=JetBrains+Mono:wght@400;600;700&display=swap" rel="stylesheet">
    <style>
        :root {
            --bg-dark: #090d16;
            --card-bg: #131b2e;
            --card-border: #22304d;
            --accent-blue: #38bdf8;
            --accent-emerald: #34d399;
            --accent-purple: #c084fc;
            --accent-amber: #fbbf24;
            --accent-rose: #f43f5e;
            --text-main: #f8fafc;
            --text-muted: #94a3b8;
            --text-subtle: #64748b;
        }

        * { box-sizing: border-box; margin: 0; padding: 0; }

        body {
            font-family: 'Outfit', sans-serif;
            background-color: var(--bg-dark);
            color: var(--text-main);
            line-height: 1.6;
            padding: 2rem;
            min-height: 100vh;
        }

        .container { max-width: 1200px; margin: 0 auto; }

        header {
            text-align: center;
            padding: 3rem 2rem;
            background: linear-gradient(135deg, rgba(56, 189, 248, 0.12), rgba(192, 132, 252, 0.12), rgba(52, 211, 153, 0.1));
            border-radius: 1.25rem;
            border: 1px solid var(--card-border);
            margin-bottom: 2rem;
            position: relative;
            backdrop-filter: blur(16px);
        }

        .badge {
            display: inline-flex;
            align-items: center;
            gap: 0.5rem;
            padding: 0.35rem 0.9rem;
            background: rgba(56, 189, 248, 0.15);
            color: var(--accent-blue);
            border-radius: 9999px;
            font-size: 0.85rem;
            font-weight: 700;
            margin-bottom: 1rem;
            border: 1px solid rgba(56, 189, 248, 0.4);
        }

        .pulse-dot {
            width: 8px;
            height: 8px;
            background-color: var(--accent-emerald);
            border-radius: 50%;
            box-shadow: 0 0 8px var(--accent-emerald);
            animation: pulse 2s infinite;
        }

        @keyframes pulse {
            0% { transform: scale(0.95); box-shadow: 0 0 0 0 rgba(52, 211, 153, 0.7); }
            70% { transform: scale(1); box-shadow: 0 0 0 8px rgba(52, 211, 153, 0); }
            100% { transform: scale(0.95); box-shadow: 0 0 0 0 rgba(52, 211, 153, 0); }
        }

        h1 {
            font-size: 2.8rem;
            font-weight: 800;
            background: linear-gradient(to right, #38bdf8, #c084fc, #34d399);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            margin-bottom: 0.6rem;
        }

        p.subtitle {
            color: var(--text-muted);
            font-size: 1.1rem;
            max-width: 800px;
            margin: 0 auto;
        }

        /* Role Switcher */
        .auth-bar {
            display: flex;
            justify-content: space-between;
            align-items: center;
            background: var(--card-bg);
            border: 1px solid var(--card-border);
            padding: 1rem 1.5rem;
            border-radius: 1rem;
            margin-bottom: 2rem;
        }

        .role-pill {
            padding: 0.3rem 0.8rem;
            border-radius: 0.5rem;
            font-weight: 800;
            font-size: 0.85rem;
            font-family: 'JetBrains Mono', monospace;
        }
        .role-admin { background: rgba(244, 63, 94, 0.2); color: var(--accent-rose); border: 1px solid rgba(244, 63, 94, 0.4); }
        .role-guest { background: rgba(251, 191, 36, 0.2); color: var(--accent-amber); border: 1px solid rgba(251, 191, 36, 0.4); }

        /* Stats Grid */
        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
            gap: 1.25rem;
            margin-bottom: 2.5rem;
        }

        .stat-card {
            background: var(--card-bg);
            border: 1px solid var(--card-border);
            border-radius: 1rem;
            padding: 1.25rem 1.5rem;
        }

        .stat-label { font-size: 0.85rem; color: var(--text-muted); font-weight: 600; text-transform: uppercase; }
        .stat-val { font-size: 1.85rem; font-weight: 800; color: var(--text-main); font-family: 'JetBrains Mono', monospace; }
        .stat-meta { font-size: 0.8rem; color: var(--accent-emerald); font-weight: 500; }

        /* Feature Cards Grid */
        .feature-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(360px, 1fr));
            gap: 1.5rem;
            margin-bottom: 2.5rem;
        }

        .feature-card {
            background: var(--card-bg);
            border: 1px solid var(--card-border);
            border-radius: 1rem;
            padding: 1.5rem;
            display: flex;
            flex-direction: column;
            gap: 1rem;
        }

        .section-title {
            font-size: 1.15rem;
            font-weight: 700;
            display: flex;
            align-items: center;
            justify-content: space-between;
        }

        .form-control {
            background: #090d16;
            border: 1px solid var(--card-border);
            border-radius: 0.6rem;
            padding: 0.75rem 1rem;
            color: #fff;
            font-size: 0.95rem;
            font-family: 'JetBrains Mono', monospace;
            width: 100%;
            outline: none;
        }
        .form-control:focus { border-color: var(--accent-blue); }

        .btn {
            background: var(--accent-blue);
            color: #090d16;
            font-weight: 700;
            border: none;
            padding: 0.65rem 1.25rem;
            border-radius: 0.6rem;
            cursor: pointer;
            transition: all 0.2s ease;
            font-size: 0.9rem;
        }
        .btn:hover { filter: brightness(1.15); transform: translateY(-1px); }
        .btn-rose { background: var(--accent-rose); color: #fff; }
        .btn-purple { background: var(--accent-purple); color: #090d16; }
        .btn-amber { background: var(--accent-amber); color: #090d16; }

        .preset-btn {
            background: rgba(255,255,255,0.05);
            border: 1px solid var(--card-border);
            color: var(--text-muted);
            font-size: 0.75rem;
            padding: 0.25rem 0.5rem;
            border-radius: 0.4rem;
            cursor: pointer;
        }
        .preset-btn:hover { background: rgba(56,189,248,0.15); color: var(--accent-blue); }

        /* Endpoint Explorer Grid */
        .endpoints-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
            gap: 1.25rem;
            margin-bottom: 2.5rem;
        }

        .endpoint-card {
            background: var(--card-bg);
            border: 1px solid var(--card-border);
            border-radius: 1rem;
            padding: 1.25rem;
            display: flex;
            flex-direction: column;
            justify-content: space-between;
            gap: 1rem;
        }

        .method {
            padding: 0.2rem 0.5rem;
            border-radius: 0.3rem;
            font-size: 0.75rem;
            font-weight: 800;
            font-family: 'JetBrains Mono', monospace;
            margin-right: 0.5rem;
        }
        .method.get { background: rgba(52, 211, 153, 0.15); color: var(--accent-emerald); border: 1px solid rgba(52, 211, 153, 0.3); }
        .method.post { background: rgba(192, 132, 252, 0.15); color: var(--accent-purple); border: 1px solid rgba(192, 132, 252, 0.3); }

        .endpoint-path { font-family: 'JetBrains Mono', monospace; font-size: 0.9rem; font-weight: 700; }
        .endpoint-desc { font-size: 0.85rem; color: var(--text-muted); }

        /* Response Viewer */
        .response-viewer {
            background: #060911;
            border: 1px solid var(--card-border);
            border-radius: 1rem;
            padding: 1.5rem;
            margin-bottom: 2.5rem;
        }

        .response-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 1rem;
            padding-bottom: 0.75rem;
            border-bottom: 1px solid var(--card-border);
        }

        .status-badge {
            font-family: 'JetBrains Mono', monospace;
            font-size: 0.85rem;
            font-weight: 700;
            padding: 0.2rem 0.6rem;
            border-radius: 0.3rem;
            background: rgba(52, 211, 153, 0.2);
            color: var(--accent-emerald);
        }

        pre.json-output {
            font-family: 'JetBrains Mono', monospace;
            font-size: 0.85rem;
            color: #38bdf8;
            max-height: 400px;
            overflow-y: auto;
            white-space: pre-wrap;
            word-break: break-word;
        }

        footer {
            text-align: center;
            color: var(--text-subtle);
            font-size: 0.9rem;
            padding-top: 2rem;
            border-top: 1px solid var(--card-border);
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <div style="display: flex; justify-content: center; gap: 0.75rem;">
                <span class="badge"><span class="pulse-dot"></span> DuckDB 1.5 Vectorized Engine</span>
                <span class="badge" style="color: var(--accent-purple); border-color: rgba(192,132,252,0.4); background: rgba(192,132,252,0.15);">Go 1.24 REST API</span>
            </div>
            <h1>Durg Electoral Roll Dashboard</h1>
            <p class="subtitle">Production API with Role-Based Auth, Admin SQL Execution Console, Group-By Analytics, and Spatial Geo-Location Proximity Search serving 1.04M+ voters.</p>
        </header>

        <!-- Auth Role Switcher -->
        <div class="auth-bar">
            <div>
                <span style="font-weight: 700; margin-right: 0.5rem;">🔐 Active Auth Role:</span>
                <span id="roleBadge" class="role-pill role-admin">ADMIN</span>
                <span id="roleDesc" style="font-size: 0.85rem; color: var(--text-muted); margin-left: 0.75rem;">(Full Privileges: Read, Search, Filter, GroupBy, GeoLocation & Raw SQL Console)</span>
            </div>
            <div style="display: flex; gap: 0.5rem;">
                <button class="btn btn-rose" onclick="switchRole('admin')">Login as ADMIN</button>
                <button class="btn btn-amber" onclick="switchRole('guest')">Login as GUEST</button>
            </div>
        </div>

        <!-- Stats Grid -->
        <div class="stats-grid">
            <div class="stat-card">
                <span class="stat-label">Total Electorate</span>
                <span class="stat-val" id="statTotalVoters">1,045,426</span>
                <span class="stat-meta">Verified DuckDB Dataset</span>
            </div>
            <div class="stat-card">
                <span class="stat-label">Male Voters</span>
                <span class="stat-val" id="statMaleVoters">519,448</span>
                <span class="stat-meta">~49.7% Male Ratio</span>
            </div>
            <div class="stat-card">
                <span class="stat-label">Female Voters</span>
                <span class="stat-val" id="statFemaleVoters">524,804</span>
                <span class="stat-meta">~50.2% Female Ratio</span>
            </div>
            <div class="stat-card">
                <span class="stat-label">Polling Station Booths</span>
                <span class="stat-val" id="statBooths">1,513</span>
                <span class="stat-meta">6 Assembly Constituencies</span>
            </div>
        </div>

        <!-- Feature Grid: Admin SQL, Group-By Analytics, Geo Location -->
        <div class="feature-grid">
            <!-- Admin SQL Execution Console -->
            <div class="feature-card" style="grid-column: span 2;">
                <div class="section-title">
                    <span>⚡ Admin SQL Execution Console <span style="font-size: 0.75rem; color: var(--accent-rose); font-weight: 700; border: 1px solid rgba(244,63,94,0.4); padding: 0.1rem 0.4rem; border-radius: 0.3rem;">ADMIN ONLY</span></span>
                    <div style="display: flex; gap: 0.3rem;">
                        <button class="preset-btn" onclick="setSQLPreset('gender')">Gender Count</button>
                        <button class="preset-btn" onclick="setSQLPreset('towns')">Top Towns</button>
                        <button class="preset-btn" onclick="setSQLPreset('constituency')">Constituency Summary</button>
                    </div>
                </div>
                <textarea id="sqlInput" class="form-control" rows="3" style="resize: vertical;">SELECT gender_english, count(*) AS total FROM voters GROUP BY 1;</textarea>
                <div style="display: flex; justify-content: space-between; align-items: center;">
                    <span style="font-size: 0.8rem; color: var(--text-muted);">Note: Only Admin role can run raw DuckDB SQL. Guest role attempts return 403 Forbidden.</span>
                    <button class="btn btn-rose" onclick="runSQLQuery()">Execute SQL &rarr;</button>
                </div>
            </div>

            <!-- Group-By Names & Demographics -->
            <div class="feature-card">
                <div class="section-title">
                    <span>📊 Group-By Analytics</span>
                    <span style="font-size: 0.75rem; color: var(--accent-emerald);">ADMIN & GUEST</span>
                </div>
                <div style="display: flex; flex-direction: column; gap: 0.6rem;">
                    <label style="font-size: 0.85rem; color: var(--text-muted);">Grouping Field:</label>
                    <select id="groupByField" class="form-control">
                        <option value="full_name">Full Name (Voter Name)</option>
                        <option value="relative_name">Relative Name</option>
                        <option value="gender">Gender</option>
                        <option value="assembly_constituency">Assembly Constituency</option>
                        <option value="town_village">Town / Village</option>
                        <option value="age_group">Age Group Breakdown</option>
                    </select>
                </div>
                <button class="btn btn-purple" onclick="runGroupBy()">Run Group-By &rarr;</button>
            </div>

            <!-- Geo-Location Spatial Proximity Search -->
            <div class="feature-card" style="grid-column: span 3;">
                <div class="section-title">
                    <span>📍 Geo-Location Spatial Proximity Search</span>
                    <div style="display: flex; gap: 0.3rem;">
                        <button class="preset-btn" onclick="setGeoPreset(21.19, 81.28)">Durg Center (21.19, 81.28)</button>
                        <button class="preset-btn" onclick="setGeoPreset(21.21, 81.38)">Bhilai Center (21.21, 81.38)</button>
                    </div>
                </div>
                <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(160px, 1fr)); gap: 0.75rem;">
                    <div>
                        <label style="font-size: 0.8rem; color: var(--text-muted);">Latitude:</label>
                        <input type="number" step="0.0001" id="geoLat" class="form-control" value="21.19">
                    </div>
                    <div>
                        <label style="font-size: 0.8rem; color: var(--text-muted);">Longitude:</label>
                        <input type="number" step="0.0001" id="geoLng" class="form-control" value="81.28">
                    </div>
                    <div>
                        <label style="font-size: 0.8rem; color: var(--text-muted);">Radius (KM):</label>
                        <input type="number" step="0.5" id="geoRadius" class="form-control" value="5.0">
                    </div>
                </div>
                <div style="display: flex; gap: 0.75rem; justify-content: flex-end;">
                    <button class="btn" style="background: rgba(56,189,248,0.2); color: var(--accent-blue);" onclick="runGeoStations()">Find Nearby Polling Stations</button>
                    <button class="btn btn-purple" onclick="runGeoVoters()">Find Nearby Voters</button>
                </div>
            </div>
        </div>

        <!-- Quick EPIC Lookup -->
        <div class="search-section">
            <div class="section-title">🔍 Quick EPIC Voter Profile Lookup</div>
            <div class="search-box">
                <input type="text" id="epicInput" class="form-control" placeholder="Enter EPIC Card Number (e.g. IXG1482629, SHJ1301639)..." value="IXG1482629">
                <button class="btn" onclick="lookupEPIC()">Search EPIC Profile</button>
            </div>
        </div>

        <!-- API Explorer Grid -->
        <div class="section-title" style="margin-bottom: 1.25rem;">⚡ Full Endpoint Explorer</div>
        <div class="endpoints-grid">
            <div class="endpoint-card">
                <div>
                    <span class="method get">GET</span> <span class="endpoint-path">/api/v1/health</span>
                    <p class="endpoint-desc">System health, DuckDB status, and memory metrics.</p>
                </div>
                <button class="btn" style="background: rgba(56,189,248,0.2); color: var(--accent-blue);" onclick="testEndpoint('/api/v1/health')">Test &rarr;</button>
            </div>

            <div class="endpoint-card">
                <div>
                    <span class="method get">GET</span> <span class="endpoint-path">/api/v1/stats</span>
                    <p class="endpoint-desc">Electorate demographics and booth totals.</p>
                </div>
                <button class="btn" style="background: rgba(56,189,248,0.2); color: var(--accent-blue);" onclick="testEndpoint('/api/v1/stats')">Test &rarr;</button>
            </div>

            <div class="endpoint-card">
                <div>
                    <span class="method get">GET</span> <span class="endpoint-path">/api/v1/voters?limit=5</span>
                    <p class="endpoint-desc">Paginated list of 1.04M voters.</p>
                </div>
                <button class="btn" style="background: rgba(56,189,248,0.2); color: var(--accent-blue);" onclick="testEndpoint('/api/v1/voters?limit=5')">Test &rarr;</button>
            </div>

            <div class="endpoint-card">
                <div>
                    <span class="method get">GET</span> <span class="endpoint-path">/api/v1/voters/group-by?field=full_name</span>
                    <p class="endpoint-desc">Group voter counts by name or demography.</p>
                </div>
                <button class="btn btn-purple" onclick="testEndpoint('/api/v1/voters/group-by?field=full_name&limit=5')">Test Group-By &rarr;</button>
            </div>

            <div class="endpoint-card">
                <div>
                    <span class="method post">POST</span> <span class="endpoint-path">/api/v1/admin/sql</span>
                    <p class="endpoint-desc">Admin raw DuckDB SQL execution (Forbidden for Guest).</p>
                </div>
                <button class="btn btn-rose" onclick="runSQLQuery()">Test Admin SQL &rarr;</button>
            </div>

            <div class="endpoint-card">
                <div>
                    <span class="method get">GET</span> <span class="endpoint-path">/api/v1/geo/nearby-polling-stations</span>
                    <p class="endpoint-desc">Spatial proximity search for booths within radius.</p>
                </div>
                <button class="btn btn-amber" onclick="runGeoStations()">Test Geo Booths &rarr;</button>
            </div>
        </div>

        <!-- Live Response Output -->
        <div class="response-viewer">
            <div class="response-header">
                <span style="font-weight: 700;">📡 Live Response Payload</span>
                <div style="display: flex; gap: 0.75rem; align-items: center;">
                    <span id="respTime" style="font-family: 'JetBrains Mono', monospace; font-size: 0.8rem; color: var(--text-muted);">0 ms</span>
                    <span id="respStatus" class="status-badge">HTTP 200 OK</span>
                </div>
            </div>
            <pre class="json-output" id="jsonOutput">Select an endpoint above or run an Admin SQL query to execute a real-time query on DuckDB.</pre>
        </div>

        <footer>
            <p>Durg Voter Production REST API &bull; Powered by <strong>Go 1.24</strong> &amp; <strong>DuckDB 1.5</strong> Vectorized SQL Engine</p>
        </footer>
    </div>

    <script>
        let activeToken = "admin-token-secret-key-12345";
        let activeRole = "admin";

        function switchRole(role) {
            if (role === 'admin') {
                activeToken = "admin-token-secret-key-12345";
                activeRole = "admin";
                document.getElementById('roleBadge').className = "role-pill role-admin";
                document.getElementById('roleBadge').innerText = "ADMIN";
                document.getElementById('roleDesc').innerText = "(Full Privileges: Read, Search, Filter, GroupBy, GeoLocation & Raw SQL Console)";
            } else {
                activeToken = "guest-token-secret-key-67890";
                activeRole = "guest";
                document.getElementById('roleBadge').className = "role-pill role-guest";
                document.getElementById('roleBadge').innerText = "GUEST";
                document.getElementById('roleDesc').innerText = "(Guest Privileges: Read, Search, Filter, GroupBy & GeoLocation. Raw SQL Execution Locked)";
            }
        }

        async function loadStats() {
            try {
                const res = await fetch('/api/v1/stats');
                const data = await res.json();
                if (data.success && data.data) {
                    document.getElementById('statTotalVoters').innerText = Number(data.data.total_voters).toLocaleString();
                    document.getElementById('statMaleVoters').innerText = Number(data.data.male_voters).toLocaleString();
                    document.getElementById('statFemaleVoters').innerText = Number(data.data.female_voters).toLocaleString();
                    document.getElementById('statBooths').innerText = Number(data.data.total_booths).toLocaleString();
                }
            } catch (err) {
                console.error("Failed to load live stats:", err);
            }
        }

        async function testEndpoint(url, method = 'GET', body = null) {
            const t0 = performance.now();
            try {
                const opts = {
                    method: method,
                    headers: {
                        'Authorization': 'Bearer ' + activeToken,
                        'Content-Type': 'application/json'
                    }
                };
                if (body) opts.body = JSON.stringify(body);

                const res = await fetch(url, opts);
                const t1 = performance.now();
                const data = await res.json();
                document.getElementById('respTime').innerText = (t1 - t0).toFixed(1) + ' ms';
                document.getElementById('respStatus').innerText = 'HTTP ' + res.status;
                if (res.status === 403) {
                    document.getElementById('respStatus').style.background = 'rgba(244,63,94,0.2)';
                    document.getElementById('respStatus').style.color = 'var(--accent-rose)';
                } else {
                    document.getElementById('respStatus').style.background = 'rgba(52,211,153,0.2)';
                    document.getElementById('respStatus').style.color = 'var(--accent-emerald)';
                }
                document.getElementById('jsonOutput').innerText = JSON.stringify(data, null, 2);
            } catch (err) {
                document.getElementById('jsonOutput').innerText = 'Error: ' + err.message;
            }
        }

        function setSQLPreset(type) {
            if (type === 'gender') {
                document.getElementById('sqlInput').value = "SELECT gender_english, count(*) AS total FROM voters GROUP BY 1;";
            } else if (type === 'towns') {
                document.getElementById('sqlInput').value = "SELECT town_village, count(*) AS total FROM polling_stations GROUP BY 1 ORDER BY 2 DESC LIMIT 5;";
            } else if (type === 'constituency') {
                document.getElementById('sqlInput').value = "SELECT assembly_constituency, count(*) AS voters FROM voters GROUP BY 1 ORDER BY 2 DESC;";
            }
        }

        async function runSQLQuery() {
            const sql = document.getElementById('sqlInput').value.trim();
            if (!sql) return;
            testEndpoint('/api/v1/admin/sql', 'POST', { sql: sql });
        }

        async function runGroupBy() {
            const field = document.getElementById('groupByField').value;
            testEndpoint('/api/v1/voters/group-by?field=' + field + '&limit=10');
        }

        function setGeoPreset(lat, lng) {
            document.getElementById('geoLat').value = lat;
            document.getElementById('geoLng').value = lng;
        }

        async function runGeoStations() {
            const lat = document.getElementById('geoLat').value;
            const lng = document.getElementById('geoLng').value;
            const radius = document.getElementById('geoRadius').value;
            testEndpoint('/api/v1/geo/nearby-polling-stations?lat=' + lat + '&lng=' + lng + '&radius_km=' + radius + '&limit=5');
        }

        async function runGeoVoters() {
            const lat = document.getElementById('geoLat').value;
            const lng = document.getElementById('geoLng').value;
            const radius = document.getElementById('geoRadius').value;
            testEndpoint('/api/v1/geo/nearby-voters?lat=' + lat + '&lng=' + lng + '&radius_km=' + radius + '&limit=5');
        }

        async function lookupEPIC() {
            const epic = document.getElementById('epicInput').value.trim();
            if (!epic) return;
            testEndpoint('/api/v1/voters/' + encodeURIComponent(epic));
        }

        window.onload = loadStats;
    </script>
</body>
</html>`
