# 🗳️ Durg Voter Web UI & REST API

A high-performance, lightweight **Go Web Application & RESTful API** with an embedded **Interactive Dashboard UI** for managing and querying Durg Electoral Roll dataset. Built with **Go 1.22**, utilizing standard library `net/http` routing, custom middleware architecture, and containerized with Docker.

---

## 🌟 Key Features

1. **Embedded Web UI Dashboard**:
   - Built-in interactive API documentation and testing dashboard served directly at `/`.
   - Sleek glassmorphism UI design with dark mode aesthetic, Google Fonts (*Outfit* & *JetBrains Mono*), and direct endpoint quick-test actions.

2. **Comprehensive RESTful Endpoints**:
   - **Health & Metrics**: `/api/v1/health` for uptime, Go runtime info, and application status.
   - **Electoral Demographics**: `/api/v1/stats` for total voter counts, gender distribution (Male/Female/Other), and assembly constituency breakdowns.
   - **Paginated Voter Directory**: `/api/v1/voters` with query parameter filtering by search term, assembly constituency, and gender.
   - **EPIC Lookup**: `/api/v1/voters/{epic_no}` for instant fetching of specific voter profile details using Go 1.22 path parameter matching.
   - **Advanced Search POST API**: `/api/v1/voters/search` supporting JSON payloads for complex multi-field filtering (age ranges, constituency, EPIC/Name search).

3. **Enterprise Middleware Architecture**:
   - **Request Logger**: Tracks incoming HTTP method, route, status code, request duration, and remote IP.
   - **CORS Support**: Enforces cross-origin headers (`Access-Control-Allow-Origin: *`) for seamless frontend integrations.
   - **Security Headers**: Standard HTTP security headers (`X-Content-Type-Options`, `X-Frame-Options`, `X-XSS-Protection`).
   - **Panic Recovery**: Middleware wrapper preventing server crashes by recovering from runtime panics and returning HTTP 500 cleanly.

4. **Production-Ready Operations**:
   - **Graceful Shutdown**: Intercepts OS signals (`SIGINT`, `SIGTERM`, `SIGHUP`, `SIGQUIT`) with a 30-second context shutdown timeout.
   - **Ultra-Minimal Docker Container**: Multi-stage Docker build producing a minimal deployment container using `scratch` base image (~10MB size).

---

## 🛠️ Tech Stack

- **Backend**: Go 1.22 (`net/http`, standard library only)
- **Frontend / Dashboard**: HTML5, CSS3 (Vanilla Glassmorphism UI)
- **Containerization**: Docker (Multi-stage build: `golang:1.22-alpine` -> `scratch`)
- **Data Model**: Structured Go structs with JSON tags & pagination metadata

---

## 📁 Repository Structure

```
durg-voter-WEB-UI/
├── main.go               # Server entry point, Go 1.22 ServeMux routes, graceful shutdown
├── pkg/
│   ├── handlers/         # HTTP handler functions & embedded HTML dashboard UI
│   │   └── handlers.go
│   ├── middleware/       # Logger, CORS, Panic Recovery, & Security Headers middleware
│   │   └── middleware.go
│   └── models/           # Voter, SearchFilter, StatsSummary, & APIResponse data structures
│       └── models.go
├── Dockerfile            # Multi-stage minimal Docker build definition
├── .dockerignore         # Excludes build artifacts and local dataset from Docker contexts
├── .gitignore            # Excludes compiled binary (server) and database files (dataset/)
└── README.md             # Project documentation
```

---

## 🚀 Getting Started

### 1. Prerequisites
- **Go 1.22** or higher installed on your machine.
- Optional: **Docker** for containerized execution.

### 2. Running Locally

```bash
# Clone the repository
git clone https://github.com/work-Sameerbanchhor/durg-voter-WEB-UI.git
cd durg-voter-WEB-UI

# Run the Go Web Server
go run main.go
```

The server will start at `http://localhost:8080`. Open your browser and navigate to:
- **Interactive Dashboard**: `http://localhost:8080/`
- **Health Check**: `http://localhost:8080/api/v1/health`

### 3. Building & Running the Binary

```bash
# Build the production executable
go build -o server main.go

# Execute binary
./server
```

### 4. Running with Docker

```bash
# Build Docker image
docker build -t durg-voter-api .

# Run Docker container
docker run -d -p 8080:8080 --name durg-voter-server durg-voter-api
```

---

## 📡 API Documentation & Reference

### Endpoints Overview

| Method | Endpoint | Description |
| :--- | :--- | :--- |
| `GET` | `/` | Web Dashboard & Interactive API Documentation UI |
| `GET` | `/api/v1/health` | Health check, server uptime, and Go runtime information |
| `GET` | `/api/v1/stats` | Demographic statistics, total voters, gender ratios, booth counts |
| `GET` | `/api/v1/voters` | Paginated list of voters with query/assembly/gender filter support |
| `GET` | `/api/v1/voters/{epic_no}` | Fetch details of a specific voter by EPIC Number |
| `POST` | `/api/v1/voters/search` | Advanced multi-field voter search using JSON request body |

---

### Request & Response Examples

#### 1. Health Check (`GET /api/v1/health`)
**Response:**
```json
{
  "success": true,
  "message": "Server is operating normally",
  "data": {
    "status": "healthy",
    "uptime": "2m14s",
    "timestamp": "2026-07-27T15:00:00Z",
    "go_version": "go1.22.5",
    "app_name": "Durg Voter API",
    "version": "1.0.0"
  }
}
```

#### 2. Electorate Statistics (`GET /api/v1/stats`)
**Response:**
```json
{
  "success": true,
  "data": {
    "total_voters": 8,
    "male_voters": 4,
    "female_voters": 4,
    "other_voters": 0,
    "total_booths": 142,
    "assembly_breakdown": {
      "Bhilai Nagar": 2,
      "Durg City": 2,
      "Durg Rural": 1,
      "Patan": 1,
      "Vaishali Nagar": 2
    }
  }
}
```

#### 3. Paginated Voter Listing (`GET /api/v1/voters?search=Rajesh&page=1&limit=5`)
**Response:**
```json
{
  "success": true,
  "data": [
    {
      "epic_no": "DRG1029384",
      "full_name": "Rajesh Sharma",
      "relative_name": "Ramesh Sharma",
      "relation_type": "Father",
      "gender": "Male",
      "age": 38,
      "house_no": "12-B",
      "polling_station_name": "Govt School Durg Central",
      "polling_station_no": 14,
      "assembly_constituency": "Durg City",
      "assembly_no": 64,
      "district": "Durg",
      "state": "Chhattisgarh"
    }
  ],
  "meta": {
    "current_page": 1,
    "page_size": 5,
    "total_items": 1,
    "total_pages": 1
  }
}
```

#### 4. Advanced Search (`POST /api/v1/voters/search`)
**Request Body:**
```json
{
  "query": "Thakur",
  "assembly_constituency": "Durg City",
  "gender": "Female",
  "min_age": 18,
  "max_age": 50
}
```

---

## 🔒 Security & Environment Configuration

- **Environment Variables**:
  - `PORT`: Sets the HTTP listening port (Default: `8080`).
- **Data Privacy Note**:
  - Database files (e.g. `dataset/durg_voters.duckdb`) and binary executables are strictly ignored via `.gitignore` and `.dockerignore` to keep the repository lightweight, performant, and secure.

---

## 📄 License

Distributed under the MIT License. See `LICENSE` for details.