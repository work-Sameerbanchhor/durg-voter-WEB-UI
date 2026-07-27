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
		SortBy:               strings.TrimSpace(r.URL.Query().Get("sort_by")),
		SortOrder:            strings.TrimSpace(r.URL.Query().Get("sort_order")),
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
      "url": "http://localhost:8080",
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
            overflow: hidden;
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
            letter-spacing: 0.5px;
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
            font-size: 3rem;
            font-weight: 800;
            background: linear-gradient(to right, #38bdf8, #c084fc, #34d399);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            margin-bottom: 0.6rem;
            letter-spacing: -0.5px;
        }

        p.subtitle {
            color: var(--text-muted);
            font-size: 1.15rem;
            max-width: 800px;
            margin: 0 auto;
        }

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
            display: flex;
            flex-direction: column;
            gap: 0.4rem;
        }

        .stat-label { font-size: 0.85rem; color: var(--text-muted); font-weight: 600; text-transform: uppercase; letter-spacing: 0.5px; }
        .stat-val { font-size: 1.85rem; font-weight: 800; color: var(--text-main); font-family: 'JetBrains Mono', monospace; }
        .stat-meta { font-size: 0.8rem; color: var(--accent-emerald); font-weight: 500; }

        /* Search Section */
        .search-section {
            background: var(--card-bg);
            border: 1px solid var(--card-border);
            border-radius: 1rem;
            padding: 1.75rem;
            margin-bottom: 2.5rem;
        }

        .section-title {
            font-size: 1.25rem;
            font-weight: 700;
            margin-bottom: 1rem;
            display: flex;
            align-items: center;
            gap: 0.5rem;
        }

        .search-box {
            display: flex;
            gap: 0.75rem;
        }

        .search-input {
            flex: 1;
            background: #090d16;
            border: 1px solid var(--card-border);
            border-radius: 0.6rem;
            padding: 0.75rem 1rem;
            color: #fff;
            font-size: 1rem;
            font-family: 'JetBrains Mono', monospace;
            outline: none;
            transition: border-color 0.2s;
        }

        .search-input:focus { border-color: var(--accent-blue); }

        .btn {
            background: var(--accent-blue);
            color: #090d16;
            font-weight: 700;
            border: none;
            padding: 0.75rem 1.5rem;
            border-radius: 0.6rem;
            cursor: pointer;
            transition: all 0.2s ease;
            font-size: 0.95rem;
        }

        .btn:hover { filter: brightness(1.15); transform: translateY(-1px); }

        /* Endpoint Explorer Grid */
        .endpoints-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(340px, 1fr));
            gap: 1.25rem;
            margin-bottom: 2.5rem;
        }

        .endpoint-card {
            background: var(--card-bg);
            border: 1px solid var(--card-border);
            border-radius: 1rem;
            padding: 1.5rem;
            display: flex;
            flex-direction: column;
            justify-content: space-between;
            gap: 1rem;
        }

        .method {
            display: inline-block;
            padding: 0.2rem 0.55rem;
            border-radius: 0.3rem;
            font-size: 0.75rem;
            font-weight: 800;
            font-family: 'JetBrains Mono', monospace;
            margin-right: 0.5rem;
        }

        .method.get { background: rgba(52, 211, 153, 0.15); color: var(--accent-emerald); border: 1px solid rgba(52, 211, 153, 0.3); }
        .method.post { background: rgba(192, 132, 252, 0.15); color: var(--accent-purple); border: 1px solid rgba(192, 132, 252, 0.3); }

        .endpoint-path {
            font-family: 'JetBrains Mono', monospace;
            font-size: 0.95rem;
            font-weight: 700;
            color: var(--text-main);
        }

        .endpoint-desc { font-size: 0.9rem; color: var(--text-muted); }

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
                <span class="badge"><span class="pulse-dot"></span> DuckDB 1.5 Engine Active</span>
                <span class="badge" style="color: var(--accent-purple); border-color: rgba(192,132,252,0.4); background: rgba(192,132,252,0.15);">Go 1.24 REST Backend</span>
            </div>
            <h1>Durg Electoral Roll Production API</h1>
            <p class="subtitle">High-performance, vectorized SQL database API querying 1,045,426 voter records and 1,513 polling station booths in under 15ms.</p>
        </header>

        <!-- Stats Row -->
        <div class="stats-grid" id="statsGrid">
            <div class="stat-card">
                <span class="stat-label">Total Electorate</span>
                <span class="stat-val" id="statTotalVoters">1,045,426</span>
                <span class="stat-meta">Verified DuckDB Dataset</span>
            </div>
            <div class="stat-card">
                <span class="stat-label">Male Voters</span>
                <span class="stat-val" id="statMaleVoters">524,198</span>
                <span class="stat-meta">~50.15% Ratio</span>
            </div>
            <div class="stat-card">
                <span class="stat-label">Female Voters</span>
                <span class="stat-val" id="statFemaleVoters">521,209</span>
                <span class="stat-meta">~49.85% Ratio</span>
            </div>
            <div class="stat-card">
                <span class="stat-label">Polling Station Booths</span>
                <span class="stat-val" id="statBooths">1,513</span>
                <span class="stat-meta">6 Assembly Constituencies</span>
            </div>
        </div>

        <!-- Quick Voter Search -->
        <div class="search-section">
            <div class="section-title">🔍 Quick EPIC Voter Profile Lookup</div>
            <div class="search-box">
                <input type="text" id="epicInput" class="search-input" placeholder="Enter EPIC Card Number (e.g. DVB5080734, SHJ0000059, DVB3380300)..." value="DVB5080734">
                <button class="btn" onclick="lookupEPIC()">Search EPIC Profile</button>
            </div>
        </div>

        <!-- API Explorer -->
        <div class="section-title" style="margin-bottom: 1.25rem;">⚡ Interactive API Endpoint Tester</div>
        <div class="endpoints-grid">
            <div class="endpoint-card">
                <div>
                    <span class="method get">GET</span> <span class="endpoint-path">/api/v1/health</span>
                    <p class="endpoint-desc">System uptime, DuckDB connection health, and Go runtime memory stats.</p>
                </div>
                <button class="btn" style="background: rgba(56,189,248,0.2); color: var(--accent-blue);" onclick="testEndpoint('/api/v1/health')">Run Test &rarr;</button>
            </div>

            <div class="endpoint-card">
                <div>
                    <span class="method get">GET</span> <span class="endpoint-path">/api/v1/stats</span>
                    <p class="endpoint-desc">Full demographic summary, male/female ratios, and booth totals.</p>
                </div>
                <button class="btn" style="background: rgba(56,189,248,0.2); color: var(--accent-blue);" onclick="testEndpoint('/api/v1/stats')">Run Test &rarr;</button>
            </div>

            <div class="endpoint-card">
                <div>
                    <span class="method get">GET</span> <span class="endpoint-path">/api/v1/voters?limit=5</span>
                    <p class="endpoint-desc">Paginated list of 1.04M voters with search, age, and assembly filters.</p>
                </div>
                <button class="btn" style="background: rgba(56,189,248,0.2); color: var(--accent-blue);" onclick="testEndpoint('/api/v1/voters?limit=5')">Run Test &rarr;</button>
            </div>

            <div class="endpoint-card">
                <div>
                    <span class="method get">GET</span> <span class="endpoint-path">/api/v1/polling-stations?limit=5</span>
                    <p class="endpoint-desc">List of all 1,513 polling stations with GPS coordinates & address details.</p>
                </div>
                <button class="btn" style="background: rgba(56,189,248,0.2); color: var(--accent-blue);" onclick="testEndpoint('/api/v1/polling-stations?limit=5')">Run Test &rarr;</button>
            </div>

            <div class="endpoint-card">
                <div>
                    <span class="method get">GET</span> <span class="endpoint-path">/api/v1/constituencies</span>
                    <p class="endpoint-desc">Breakdown of total electorate and booths across all Assembly Constituencies.</p>
                </div>
                <button class="btn" style="background: rgba(56,189,248,0.2); color: var(--accent-blue);" onclick="testEndpoint('/api/v1/constituencies')">Run Test &rarr;</button>
            </div>

            <div class="endpoint-card">
                <div>
                    <span class="method post">POST</span> <span class="endpoint-path">/api/v1/voters/search</span>
                    <p class="endpoint-desc">Advanced multi-field JSON search payload execution on DuckDB.</p>
                </div>
                <button class="btn" style="background: rgba(192,132,252,0.2); color: var(--accent-purple);" onclick="testPostSearch()">Run POST Test &rarr;</button>
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
            <pre class="json-output" id="jsonOutput">Select an endpoint above or search an EPIC number to execute a real-time query on DuckDB.</pre>
        </div>

        <footer>
            <p>Durg Voter Production REST API &bull; Powered by <strong>Go 1.24</strong> &amp; <strong>DuckDB 1.5</strong> Vectorized SQL Engine</p>
        </footer>
    </div>

    <script>
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

        async function testEndpoint(url) {
            const t0 = performance.now();
            try {
                const res = await fetch(url);
                const t1 = performance.now();
                const data = await res.json();
                document.getElementById('respTime').innerText = (t1 - t0).toFixed(1) + ' ms';
                document.getElementById('respStatus').innerText = 'HTTP ' + res.status;
                document.getElementById('jsonOutput').innerText = JSON.stringify(data, null, 2);
            } catch (err) {
                document.getElementById('jsonOutput').innerText = 'Error: ' + err.message;
            }
        }

        async function lookupEPIC() {
            const epic = document.getElementById('epicInput').value.trim();
            if (!epic) return;
            testEndpoint('/api/v1/voters/' + encodeURIComponent(epic));
        }

        async function testPostSearch() {
            const t0 = performance.now();
            try {
                const res = await fetch('/api/v1/voters/search', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        query: "Patel",
                        gender: "Female",
                        min_age: 18,
                        max_age: 45,
                        limit: 5
                    })
                });
                const t1 = performance.now();
                const data = await res.json();
                document.getElementById('respTime').innerText = (t1 - t0).toFixed(1) + ' ms';
                document.getElementById('respStatus').innerText = 'HTTP ' + res.status;
                document.getElementById('jsonOutput').innerText = JSON.stringify(data, null, 2);
            } catch (err) {
                document.getElementById('jsonOutput').innerText = 'Error: ' + err.message;
            }
        }

        window.onload = loadStats;
    </script>
</body>
</html>`
