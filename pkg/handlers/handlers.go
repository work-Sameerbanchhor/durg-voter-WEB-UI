package handlers

import (
	"encoding/json"
	"math"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"durg-voter-api/pkg/models"
)

var (
	startTime = time.Now()

	// In-memory mock database of sample voter records for high-speed API demonstration
	mockVoters = []models.Voter{
		{EPICNo: "DRG1029384", FullName: "Rajesh Sharma", RelativeName: "Ramesh Sharma", RelationType: "Father", Gender: "Male", Age: 38, HouseNo: "12-B", PollingStationName: "Govt School Durg Central", PollingStationNo: 14, AssemblyConstituency: "Durg City", AssemblyNo: 64, District: "Durg", State: "Chhattisgarh"},
		{EPICNo: "DRG1029385", FullName: "Priyanka Sahu", RelativeName: "Sanjay Sahu", RelationType: "Husband", Gender: "Female", Age: 34, HouseNo: "45/1", PollingStationName: "Community Hall Bhilai", PollingStationNo: 22, AssemblyConstituency: "Bhilai Nagar", AssemblyNo: 65, District: "Durg", State: "Chhattisgarh"},
		{EPICNo: "DRG1029386", FullName: "Amit Kumar Verma", RelativeName: "Harish Verma", RelationType: "Father", Gender: "Male", Age: 29, HouseNo: "88-C", PollingStationName: "Govt Primary School Vaishali Nagar", PollingStationNo: 8, AssemblyConstituency: "Vaishali Nagar", AssemblyNo: 66, District: "Durg", State: "Chhattisgarh"},
		{EPICNo: "DRG1029387", FullName: "Sunita Deshmukh", RelativeName: "Prakash Deshmukh", RelationType: "Husband", Gender: "Female", Age: 45, HouseNo: "102", PollingStationName: "Panchayat Bhavan Patan", PollingStationNo: 3, AssemblyConstituency: "Patan", AssemblyNo: 62, District: "Durg", State: "Chhattisgarh"},
		{EPICNo: "DRG1029388", FullName: "Karan Patel", RelativeName: "Vijay Patel", RelationType: "Father", Gender: "Male", Age: 22, HouseNo: "77", PollingStationName: "Govt School Durg Rural", PollingStationNo: 19, AssemblyConstituency: "Durg Rural", AssemblyNo: 63, District: "Durg", State: "Chhattisgarh"},
		{EPICNo: "DRG1029389", FullName: "Neha Thakur", RelativeName: "Mohan Thakur", RelationType: "Father", Gender: "Female", Age: 27, HouseNo: "31-A", PollingStationName: "Govt School Durg Central", PollingStationNo: 14, AssemblyConstituency: "Durg City", AssemblyNo: 64, District: "Durg", State: "Chhattisgarh"},
		{EPICNo: "DRG1029390", FullName: "Vikram Singh", RelativeName: "Dharmendra Singh", RelationType: "Father", Gender: "Male", Age: 52, HouseNo: "503", PollingStationName: "Community Hall Bhilai", PollingStationNo: 22, AssemblyConstituency: "Bhilai Nagar", AssemblyNo: 65, District: "Durg", State: "Chhattisgarh"},
		{EPICNo: "DRG1029391", FullName: "Anjali Gupta", RelativeName: "Rakesh Gupta", RelationType: "Husband", Gender: "Female", Age: 41, HouseNo: "14/9", PollingStationName: "Govt Primary School Vaishali Nagar", PollingStationNo: 8, AssemblyConstituency: "Vaishali Nagar", AssemblyNo: 66, District: "Durg", State: "Chhattisgarh"},
	}

	mu sync.RWMutex
)

// SendJSON helper to write JSON responses consistently
func SendJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// HealthCheckHandler returns system uptime and server status
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	status := models.HealthStatus{
		Status:    "healthy",
		Uptime:    time.Since(startTime).String(),
		Timestamp: time.Now(),
		GoVersion: runtime.Version(),
		AppName:   "Durg Voter API",
		Version:   "1.0.0",
	}

	SendJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Server is operating normally",
		Data:    status,
	})
}

// GetStatsHandler returns aggregated demographic metrics
func GetStatsHandler(w http.ResponseWriter, r *http.Request) {
	mu.RLock()
	defer mu.RUnlock()

	var male, female, other int64
	assemblies := make(map[string]int)

	for _, v := range mockVoters {
		if strings.EqualFold(v.Gender, "Male") {
			male++
		} else if strings.EqualFold(v.Gender, "Female") {
			female++
		} else {
			other++
		}
		assemblies[v.AssemblyConstituency]++
	}

	stats := models.StatsSummary{
		TotalVoters:       int64(len(mockVoters)),
		MaleVoters:        male,
		FemaleVoters:      female,
		OtherVoters:       other,
		TotalBooths:       142,
		AssemblyBreakdown: assemblies,
	}

	SendJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    stats,
	})
}

// ListVotersHandler handles paginated list and basic text search
func ListVotersHandler(w http.ResponseWriter, r *http.Request) {
	mu.RLock()
	defer mu.RUnlock()

	query := strings.ToLower(r.URL.Query().Get("search"))
	assembly := strings.ToLower(r.URL.Query().Get("assembly"))
	gender := strings.ToLower(r.URL.Query().Get("gender"))

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 10
	}

	var filtered []models.Voter
	for _, v := range mockVoters {
		matchesQuery := query == "" ||
			strings.Contains(strings.ToLower(v.FullName), query) ||
			strings.Contains(strings.ToLower(v.EPICNo), query) ||
			strings.Contains(strings.ToLower(v.HouseNo), query)

		matchesAssembly := assembly == "" || strings.Contains(strings.ToLower(v.AssemblyConstituency), assembly)
		matchesGender := gender == "" || strings.EqualFold(v.Gender, gender)

		if matchesQuery && matchesAssembly && matchesGender {
			filtered = append(filtered, v)
		}
	}

	totalItems := int64(len(filtered))
	totalPages := int(math.Ceil(float64(totalItems) / float64(limit)))

	startIdx := (page - 1) * limit
	endIdx := startIdx + limit

	if startIdx >= len(filtered) {
		filtered = []models.Voter{}
	} else {
		if endIdx > len(filtered) {
			endIdx = len(filtered)
		}
		filtered = filtered[startIdx:endIdx]
	}

	pagination := &models.Pagination{
		CurrentPage: page,
		PageSize:    limit,
		TotalItems:  totalItems,
		TotalPages:  totalPages,
	}

	SendJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    filtered,
		Meta:    pagination,
	})
}

// GetVoterByIDHandler fetches details of a specific voter by EPIC number
func GetVoterByIDHandler(w http.ResponseWriter, r *http.Request) {
	epicNo := r.PathValue("epic_no")
	if epicNo == "" {
		epicNo = r.URL.Query().Get("epic_no")
	}

	if epicNo == "" {
		SendJSON(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "EPIC Number is required",
		})
		return
	}

	mu.RLock()
	defer mu.RUnlock()

	for _, v := range mockVoters {
		if strings.EqualFold(v.EPICNo, epicNo) {
			SendJSON(w, http.StatusOK, models.APIResponse{
				Success: true,
				Data:    v,
			})
			return
		}
	}

	SendJSON(w, http.StatusNotFound, models.APIResponse{
		Success: false,
		Error:   "Voter record not found",
	})
}

// SearchVotersHandler handles POST payload structured filtering
func SearchVotersHandler(w http.ResponseWriter, r *http.Request) {
	var filter models.SearchFilter
	if err := json.NewDecoder(r.Body).Decode(&filter); err != nil {
		SendJSON(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Invalid JSON payload: " + err.Error(),
		})
		return
	}

	mu.RLock()
	defer mu.RUnlock()

	q := strings.ToLower(filter.Query)
	assembly := strings.ToLower(filter.AssemblyConstituency)
	gender := strings.ToLower(filter.Gender)

	var matches []models.Voter
	for _, v := range mockVoters {
		if q != "" && !strings.Contains(strings.ToLower(v.FullName), q) && !strings.Contains(strings.ToLower(v.EPICNo), q) {
			continue
		}
		if assembly != "" && !strings.Contains(strings.ToLower(v.AssemblyConstituency), assembly) {
			continue
		}
		if gender != "" && !strings.EqualFold(v.Gender, gender) {
			continue
		}
		if filter.MinAge > 0 && v.Age < filter.MinAge {
			continue
		}
		if filter.MaxAge > 0 && v.Age > filter.MaxAge {
			continue
		}
		matches = append(matches, v)
	}

	SendJSON(w, http.StatusOK, models.APIResponse{
		Success: true,
		Data:    matches,
		Meta: &models.Pagination{
			CurrentPage: 1,
			PageSize:    len(matches),
			TotalItems:  int64(len(matches)),
			TotalPages:  1,
		},
	})
}

// APIDocsHandler serves a sleek HTML API documentation dashboard
func APIDocsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(dashboardHTML))
}

const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Durg Voter REST API Documentation</title>
    <link href="https://fonts.googleapis.com/css2?family=Outfit:wght@300;400;600;700&family=JetBrains+Mono:wght@400;600&display=swap" rel="stylesheet">
    <style>
        :root {
            --bg-dark: #0f172a;
            --card-bg: #1e293b;
            --border-color: #334155;
            --accent-blue: #38bdf8;
            --accent-green: #4ade80;
            --accent-purple: #c084fc;
            --text-main: #f8fafc;
            --text-muted: #94a3b8;
        }

        * {
            box-sizing: border-box;
            margin: 0;
            padding: 0;
        }

        body {
            font-family: 'Outfit', sans-serif;
            background-color: var(--bg-dark);
            color: var(--text-main);
            line-height: 1.6;
            padding: 2rem;
        }

        .container {
            max-width: 1100px;
            margin: 0 auto;
        }

        header {
            text-align: center;
            padding: 3rem 1rem;
            background: linear-gradient(135deg, rgba(56, 189, 248, 0.1), rgba(192, 132, 252, 0.1));
            border-radius: 1rem;
            border: 1px solid var(--border-color);
            margin-bottom: 2.5rem;
            backdrop-filter: blur(10px);
        }

        .badge {
            display: inline-block;
            padding: 0.25rem 0.75rem;
            background: rgba(56, 189, 248, 0.2);
            color: var(--accent-blue);
            border-radius: 9999px;
            font-size: 0.85rem;
            font-weight: 600;
            margin-bottom: 1rem;
            border: 1px solid rgba(56, 189, 248, 0.4);
        }

        h1 {
            font-size: 2.75rem;
            font-weight: 700;
            background: linear-gradient(to right, #38bdf8, #c084fc);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            margin-bottom: 0.5rem;
        }

        p.subtitle {
            color: var(--text-muted);
            font-size: 1.1rem;
        }

        .grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
            gap: 1.5rem;
            margin-bottom: 3rem;
        }

        .card {
            background: var(--card-bg);
            border: 1px solid var(--border-color);
            border-radius: 0.75rem;
            padding: 1.5rem;
            transition: transform 0.2s ease, border-color 0.2s ease;
        }

        .card:hover {
            transform: translateY(-3px);
            border-color: var(--accent-blue);
        }

        .method {
            display: inline-block;
            padding: 0.2rem 0.5rem;
            border-radius: 0.25rem;
            font-size: 0.8rem;
            font-weight: 700;
            font-family: 'JetBrains Mono', monospace;
            margin-right: 0.5rem;
        }

        .method.get { background: rgba(74, 222, 128, 0.2); color: var(--accent-green); border: 1px solid var(--accent-green); }
        .method.post { background: rgba(192, 132, 252, 0.2); color: var(--accent-purple); border: 1px solid var(--accent-purple); }

        .endpoint-path {
            font-family: 'JetBrains Mono', monospace;
            font-size: 1rem;
            color: var(--text-main);
            font-weight: 600;
        }

        .endpoint-desc {
            margin: 0.75rem 0 1rem 0;
            color: var(--text-muted);
            font-size: 0.95rem;
        }

        a.try-btn {
            display: inline-block;
            padding: 0.5rem 1rem;
            background: rgba(56, 189, 248, 0.15);
            color: var(--accent-blue);
            text-decoration: none;
            border-radius: 0.5rem;
            font-size: 0.875rem;
            font-weight: 600;
            border: 1px solid rgba(56, 189, 248, 0.3);
            transition: all 0.2s ease;
        }

        a.try-btn:hover {
            background: var(--accent-blue);
            color: var(--bg-dark);
        }

        footer {
            text-align: center;
            color: var(--text-muted);
            font-size: 0.9rem;
            margin-top: 3rem;
            padding-top: 1.5rem;
            border-top: 1px solid var(--border-color);
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <span class="badge">Go 1.22 REST Web Server</span>
            <h1>Durg Voter API Dashboard</h1>
            <p class="subtitle">Fast, lightweight, and structured RESTful Web Service built with Go net/http</p>
        </header>

        <div class="grid">
            <div class="card">
                <span class="method get">GET</span> <span class="endpoint-path">/api/v1/health</span>
                <p class="endpoint-desc">Check system health status, Go runtime version, and server uptime metrics.</p>
                <a href="/api/v1/health" target="_blank" class="try-btn">Test Endpoint &rarr;</a>
            </div>

            <div class="card">
                <span class="method get">GET</span> <span class="endpoint-path">/api/v1/stats</span>
                <p class="endpoint-desc">Retrieve overall electorate statistics, gender breakdowns, and constituency counts.</p>
                <a href="/api/v1/stats" target="_blank" class="try-btn">Test Endpoint &rarr;</a>
            </div>

            <div class="card">
                <span class="method get">GET</span> <span class="endpoint-path">/api/v1/voters</span>
                <p class="endpoint-desc">Get paginated list of voter records with support for search, assembly, and gender filters.</p>
                <a href="/api/v1/voters?limit=5" target="_blank" class="try-btn">Test Endpoint &rarr;</a>
            </div>

            <div class="card">
                <span class="method get">GET</span> <span class="endpoint-path">/api/v1/voters/{epic_no}</span>
                <p class="endpoint-desc">Fetch complete details for a specific voter using their unique EPIC Card Number.</p>
                <a href="/api/v1/voters/DRG1029384" target="_blank" class="try-btn">Test Sample Record &rarr;</a>
            </div>

            <div class="card">
                <span class="method post">POST</span> <span class="endpoint-path">/api/v1/voters/search</span>
                <p class="endpoint-desc">Advanced multi-criteria filtering via JSON request body (age range, constituency, query).</p>
                <span style="color: var(--text-muted); font-size: 0.85rem; font-style: italic;">Requires POST Request Body</span>
            </div>
        </div>

        <footer>
            <p>Go Web Server API &bull; Built with standard library <code>net/http</code></p>
        </footer>
    </div>
</body>
</html>`
