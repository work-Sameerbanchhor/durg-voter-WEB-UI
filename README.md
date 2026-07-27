# 🗳️ Durg Voter Production REST API & Embedded Dashboard

A high-performance, production-grade **Go RESTful API & Interactive Dashboard Backend** powered by **Go 1.24** and the **DuckDB 1.5 Vectorized Analytics Engine**, serving **1,045,426 voter records** and **1,513 polling station booths**.

---

## 🌟 Key Features

1. **Vectorized Analytics Engine (DuckDB 1.5)**:
   - Direct integration with `database/durg_voters.duckdb` using Go `database/sql` driver (`github.com/marcboeker/go-duckdb`).
   - Query latency under 15ms across 1.04M records.
   - Vectorized SQL aggregations for instant electorate demography calculations.

2. **Full RESTful Endpoints**:
   - **Health & Metrics**: `/api/v1/health` (DuckDB connection status, ping latency, memory alloc, uptime, goroutine count).
   - **Electorate Demographics**: `/api/v1/stats` (Total voters: 1.04M+, male/female/other ratios, booth breakdown).
   - **Voter Directory & Pagination**: `/api/v1/voters` (Filtering by search term, assembly constituency, gender, age range, town/village, part number).
   - **EPIC Lookup**: `/api/v1/voters/{epic_no}` (Instant fetching of specific voter profile details).
   - **Advanced Search POST API**: `/api/v1/voters/search` (Structured multi-criteria JSON body search).
   - **Polling Stations Directory**: `/api/v1/polling-stations` (List 1,513 booths with address & GPS coordinates).
   - **Constituencies Summary**: `/api/v1/constituencies` (Voter distribution per assembly constituency).
   - **OpenAPI 3.0 Specification**: `/api/v1/openapi.json` (OpenAPI standard API schema).

3. **Enterprise Middleware Architecture**:
   - **Request ID Middleware**: Attaches unique `X-Request-ID` to all HTTP requests and contexts for distributed tracing.
   - **Token Bucket Rate Limiting**: IP-based throttling preventing DDoS & API abuse (HTTP 429 with `Retry-After`).
   - **Structured Logger**: Tracks HTTP method, route, status code, execution duration, IP, and Request ID.
   - **Configurable CORS**: Supports environment-configurable allowed origin headers.
   - **Security Headers**: Injects `X-Content-Type-Options`, `X-Frame-Options`, `X-XSS-Protection`, and `Referrer-Policy`.
   - **Panic Recovery**: Recovers from panics gracefully without leaking sensitive stack traces.

4. **Embedded Glassmorphism UI Dashboard**:
   - Interactive UI served at `/` featuring live dataset stat counters, quick EPIC voter profile lookup widget, and interactive endpoint execution tester with JSON syntax highlighting and real-time response timer.

---

## 🛠️ Architecture & Tech Stack

```
durg-voter-WEB-UI/
├── main.go                     # Server entrypoint, Go 1.24 ServeMux routes, graceful signal shutdown
├── pkg/
│   ├── config/                 # Environment configuration loader (PORT, DB_PATH, RATE_LIMIT)
│   ├── db/                     # DuckDB connection pool manager with health ping
│   ├── models/                 # Voter, PollingStation, StatsSummary, APIResponse models
│   ├── repository/             # DuckDB SQL data access layer (VoterRepository)
│   ├── service/                # Business logic, input validation, and in-memory TTL caching
│   ├── middleware/             # RequestID, RateLimiting, Logger, CORS, Security, Panic Recovery
│   └── handlers/               # HTTP endpoints, OpenAPI spec, embedded UI dashboard
├── database/
│   └── durg_voters.duckdb      # 1.04 Million Voters DuckDB Database (~131 MB)
├── Dockerfile                  # Production CGO multi-stage build container with healthcheck
├── Makefile                    # Developer workflow automation
├── go.mod                      # Go 1.24 module definition
└── README.md                   # Comprehensive documentation
```

---

## 🚀 Getting Started

### 1. Prerequisites
- **Go 1.24+**
- **DuckDB 1.5** dataset at `database/durg_voters.duckdb`

### 2. Environment Variables

| Variable | Description | Default |
| :--- | :--- | :--- |
| `PORT` | HTTP Listening Port | `7860` |
| `DB_PATH` | Path to DuckDB file | `database/durg_voters.duckdb` |
| `ENVIRONMENT` | Execution mode (`production`/`development`) | `production` |
| `RATE_LIMIT_RPS` | Allowed requests per second per IP | `50` |
| `RATE_LIMIT_BURST` | Max burst requests allowed per IP | `100` |
| `CORS_ALLOWED_ORIGINS` | Allowed CORS origins | `*` |

### 3. Run Tests
```bash
make test
# Or: go test -v ./...
```

### 4. Build & Run
```bash
# Build binary
make build

# Start server
make run
```

Access Dashboard at: `http://localhost:7860/`

---

## 📡 API Reference

| Method | Endpoint | Description |
| :--- | :--- | :--- |
| `GET` | `/` | Interactive Glassmorphism Dashboard UI |
| `GET` | `/api/v1/health` | Health check, DuckDB ping, Go runtime memory |
| `GET` | `/api/v1/stats` | Electorate demographic statistics |
| `GET` | `/api/v1/voters` | Paginated voters list with search/age/assembly filters |
| `GET` | `/api/v1/voters/{epic_no}` | Specific voter profile by EPIC Card Number |
| `POST` | `/api/v1/voters/search` | Advanced JSON multi-criteria voter query |
| `GET` | `/api/v1/polling-stations` | Polling station booths listing |
| `GET` | `/api/v1/constituencies` | Electorate breakdown by assembly constituency |
| `GET` | `/api/v1/openapi.json` | OpenAPI 3.0 specification |

---

## 📄 License
MIT License