# Go PDF Report Generation Service

A microservice built in Go that generates PDF reports for students by consuming the Node.js backend API.

## Quick Start

### 1. Prerequisites

- Go 1.21+
- Node.js backend running on port 5007
- PostgreSQL with seeded database

### 2. Build & Run

```bash
cd go-service

# Install dependencies
go mod tidy

# Build (recommended for macOS)
CGO_ENABLED=0 go build -o pdf-service cmd/server/main.go

# Run
./pdf-service
```

### 3. Test

```bash
# Health check
curl http://localhost:8080/health

# Generate PDF (requires auth - see docs/API.md for full testing guide)
curl -b cookies.txt -H "X-CSRF-Token: <token>" \
  http://localhost:8080/api/v1/students/2/report \
  --output student_report.pdf
```

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Service health check |
| `/api/v1/students/:id/report` | GET | Generate PDF report |

## Features

| Feature | Description |
|---------|-------------|
| **Caching** | In-memory cache with 5min TTL reduces backend API calls |
| **Rate Limiting** | Token bucket (10 req/s, burst 20) protects against abuse |
| **PDF Generation** | Formatted report with student info, academic details, family info |
| **Graceful Shutdown** | Clean resource cleanup on SIGINT/SIGTERM |

## Project Structure

```
go-service/
├── cmd/server/main.go           # Entry point
├── internal/
│   ├── cache/                   # In-memory cache with TTL
│   ├── config/                  # Environment configuration
│   ├── handlers/                # HTTP handlers
│   ├── middleware/              # Rate limiting middleware
│   ├── models/                  # Data models
│   ├── ratelimiter/             # Token bucket implementation
│   └── services/                # Business logic (API client, PDF generation)
├── docs/API.md                  # Detailed API documentation
├── screenshots/                 # Test screenshots
└── README.md
```

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Service port |
| `NODE_API_URL` | `http://localhost:5007` | Node.js backend URL |
| `CACHE_TTL_SECONDS` | `300` | Cache duration |
| `RATE_LIMIT_PER_SECOND` | `10` | Rate limit |
| `RATE_LIMIT_BURST` | `20` | Burst size |

## Documentation

See [docs/API.md](docs/API.md) for:
- Complete API documentation
- Authentication flow
- Step-by-step testing guide with Postman/cURL
- Screenshots
- Troubleshooting guide

## Screenshots

| Screenshot | Description |
|------------|-------------|
| [Login API](screenshots/03-login-api.png) | Authentication response |
| [Get Student](screenshots/02-get-student-api.png) | Student data from Node.js API |
| [PDF Report](screenshots/01-pdf-report-output.png) | Generated PDF output |
