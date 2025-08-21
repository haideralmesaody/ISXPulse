# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Table of Contents

1. [Project Overview](#project-overview)
2. [Core Principles](#core-principles)
3. [Architecture](#architecture)
4. [Build & Development Commands](#build--development-commands)
5. [Key Standards](#key-standards)
6. [Critical Patterns](#critical-patterns)
7. [Error Handling Best Practices](#error-handling-best-practices)
8. [Interface & Dependency Management](#interface--dependency-management)
9. [Testing Requirements](#testing-requirements)
10. [Configuration Management](#configuration-management)
11. [Integration Patterns](#integration-patterns)
12. [Performance Guidelines](#performance-guidelines)
13. [Security Considerations](#security-considerations)
14. [Common Workflows](#common-workflows)
15. [Observability](#observability)
16. [Monitoring & Alerting](#monitoring--alerting)
17. [Documentation Standards](#documentation-standards)
18. [Code Review Checklist](#code-review-checklist)
19. [Version Management](#version-management)
20. [Debugging & Troubleshooting](#debugging--troubleshooting)
21. [Database/Storage Patterns](#databasestorage-patterns)
22. [Deployment Notes](#deployment-notes)
23. [React Hydration Best Practices](#react-hydration-best-practices)
24. [Important Notes](#important-notes)
25. [Project Structure](#project-structure)

## Project Overview

ISX Daily Reports Scrapper is a professional-grade financial data processing system for the Iraqi Stock Exchange (ISX). It provides automated data collection, processing, analysis, and reporting capabilities with enterprise-level security and reliability.

**Key Technologies:**
- **Backend**: Go 1.21+ with Chi v5 router, slog logging
- **Frontend**: Next.js 14 with TypeScript, Tailwind CSS, Shadcn/ui
- **Data**: CSV/Excel processing, Google Sheets integration
- **Security**: AES-256 encryption, hardware-locked licensing
- **Deployment**: Single binary with embedded frontend

## Core Principles

1. **Security First**: All sensitive data encrypted, license-protected operations
2. **Clean Architecture**: Clear separation of concerns, dependency injection
3. **Type Safety**: Strong typing in both Go and TypeScript
4. **Testability**: TDD approach, minimum 80% coverage
5. **Performance**: Concurrent processing, efficient resource usage
6. **Observability**: Structured logging, metrics, distributed tracing

## Architecture

### Backend Structure (Go)
```
api/
├── cmd/                    # Application entry points
│   ├── web-licensed/       # Main web server
│   ├── scraper/           # ISX data scraper
│   ├── processor/         # Data processor
│   └── indexcsv/          # Index extractor
├── internal/              # Private application code
│   ├── app/              # Application initialization
│   ├── config/           # Configuration management
│   ├── errors/           # Error handling (RFC 7807)
│   ├── license/          # License management
│   ├── middleware/       # HTTP middleware (Chi)
│   ├── operations/       # Pipeline operations
│   ├── security/         # Encryption, auth
│   ├── services/         # Business logic
│   ├── transport/http/   # HTTP handlers
│   └── websocket/        # WebSocket handlers
└── pkg/                  # Public packages
    └── contracts/        # Shared types/interfaces
```

### Frontend Structure (Next.js)
```
web/
├── app/                  # Next.js app router pages
├── components/           # React components
│   ├── ui/              # Shadcn/ui components
│   ├── operations/      # Operation-specific
│   └── layout/          # Layout components
├── lib/                 # Utilities and hooks
├── types/               # TypeScript definitions
└── public/              # Static assets
```

## Build & Development Commands

### 🚨 MANDATORY BUILD RULES - CLAUDE CODE MUST FOLLOW
```
═══════════════════════════════════════════════════════════════════════
⚠️  ABSOLUTE BUILD RULES - NO EXCEPTIONS EVER  ⚠️
═══════════════════════════════════════════════════════════════════════
1. NEVER run 'npm run build' in web directory
2. NEVER run 'go build' in api/ directory  
3. NEVER create .next/ directory in web
4. NEVER create out/ directory in web (except via build.bat)
5. ALWAYS use ./build.bat from project root for ALL builds
6. ALWAYS clear logs before each build (automatic in build.bat)
7. ALL builds MUST output to dist/ directory ONLY
8. NO build artifacts allowed in api/ or web/ directories ever
9. BEFORE any build, ALWAYS verify you're in project root
10. If web/.next exists, DELETE it immediately

ENFORCEMENT: Claude Code will refuse to run any build commands 
that violate these rules and will suggest ./build.bat instead.
═══════════════════════════════════════════════════════════════════════
```

### Primary Build Command (THE ONLY WAY TO BUILD)
```bash
# ✅ CORRECT - From project root ONLY:
./build.bat              # Builds all to dist/ (clears logs first)
./build.bat -target=all  # Same as above
./build.bat -target=web  # Build web-licensed only to dist/
./build.bat -target=frontend  # Build frontend (embedded in exe)
./build.bat -target=clean     # Clean artifacts AND logs
./build.bat -target=test      # Run all tests
./build.bat -target=release   # Create release in dist/

# ❌ FORBIDDEN - Claude Code will REFUSE these:
cd web && npm run build              # NEVER
cd api && go build ./...             # NEVER  
cd web && next build                 # NEVER
cd web && npm run export             # NEVER
```

### Development Commands (NO BUILDS ALLOWED)
```bash
# Backend development (NO BUILDING)
cd api
go run cmd/web-licensed/main.go  # Run server (dev mode only)
go test ./... -race              # Run tests
go test ./... -cover             # Coverage analysis

# Frontend development (DEV SERVER ONLY - NO BUILDS)
cd web
npm run dev                      # ✅ Dev server ONLY (OK)
npm run test                     # ✅ Run tests (OK)
npm run lint                     # ✅ Lint code (OK)
npm run type-check              # ✅ TypeScript check (OK)

# ⛔ ABSOLUTELY FORBIDDEN IN web:
npm run build                    # ❌ NEVER - Use ./build.bat
npm run export                   # ❌ NEVER - Use ./build.bat  
next build                       # ❌ NEVER - Use ./build.bat
npx next build                   # ❌ NEVER - Use ./build.bat
```

## Frontend Embedding Standards

### Industry-Standard Pattern
Following Grafana, CockroachDB, and Kubernetes Dashboard best practices, we use **explicit file patterns** for embedding frontend assets instead of wildcards.

### Embedding Rules
1. **NEVER use wildcards** (`all:frontend/*`) - causes empty directory errors
2. **ALWAYS use explicit patterns** - security, performance, reproducibility
3. **Validate before embedding** - ensure all required files present
4. **Clean empty directories** - Next.js creates them, Go can't embed them

### Approved Embed Pattern
```go
//go:embed frontend/index.html frontend/404.html frontend/index.txt
//go:embed frontend/_next
//go:embed frontend/*.ico frontend/*.png frontend/*.svg 
//go:embed frontend/site.webmanifest
var frontendFiles embed.FS
```

### Build Process for Frontend
1. `npm run build` in web (via build.bat ONLY)
2. Output copied to `api/cmd/web-licensed/frontend/`
3. Empty directories removed (Next.js quirk)
4. Validation ensures required files present
5. Go embeds using explicit patterns
6. Binary includes optimized frontend

### Required Frontend Assets
- `index.html` - Main entry point
- `404.html` - Error page
- `_next/` - Next.js assets (must not be empty)
- `favicon.ico` - Browser icon
- `site.webmanifest` - PWA manifest

### Forbidden in Production Build
- `*.map` files (source maps)
- `.env*` files (environment configs)
- `node_modules/` (dependencies)
- Empty directories

## Key Standards

### Go Standards
- **Router**: Chi v5 only - no Gin, Echo, or other routers
- **Logging**: slog only - no fmt.Println, log.Printf
- **Errors**: RFC 7807 Problem Details for all API errors
- **Context**: Always pass context.Context as first parameter
- **Testing**: Table-driven tests, minimum 80% coverage
- **Interfaces**: Define interfaces in consumer packages
- **Frontend Embedding**: Explicit file patterns only (no wildcards) following Grafana/CockroachDB standards
- **Embed Validation**: All embedded assets must be validated before build

### TypeScript/React Standards
- **Types**: Strict mode enabled, no any types
- **Components**: Functional components with hooks
- **State**: useState, useReducer for local; Context for global
- **Styling**: Tailwind CSS with Shadcn/ui components
- **Forms**: react-hook-form with zod validation
- **API**: Centralized API client with proper error handling

### Next.js Component Architecture

#### Server vs Client Components
Next.js 14+ uses React Server Components by default. Understanding when to use server vs client components is critical:

**Server Components (default - no 'use client'):**
- Can export metadata for SEO
- Can directly fetch data from databases
- Cannot use hooks, event handlers, or browser APIs
- Better performance (smaller bundle size)
- Use for static content, data fetching, SEO metadata

**Client Components ('use client' directive):**
- Required for interactivity (onClick, onChange, etc.)
- Required for hooks (useState, useEffect, etc.)
- Required for browser APIs (localStorage, window, etc.)
- Cannot export metadata (will cause build warnings)
- Use for forms, modals, real-time updates, animations

#### Server/Client Component Pattern
When you need both metadata (SEO) and interactivity, split components:

```typescript
// app/reports/page.tsx - Server component with metadata
import ReportsClient from './reports-client'

export const metadata = {
  title: 'Reports - ISX Pulse',
  description: 'Financial reports for the Iraqi Stock Exchange.',
  robots: { index: false, follow: false }
}

export default function ReportsPage() {
  return <ReportsClient />
}
```

```typescript
// app/reports/reports-client.tsx - Client component with interactivity
'use client'

import { useState, useCallback } from 'react'
import { useToast } from '@/lib/hooks/use-toast'

export default function ReportsClient() {
  const [data, setData] = useState()
  const { toast } = useToast()
  
  const handleClick = useCallback(() => {
    toast({ title: "Action completed" })
  }, [toast])
  
  return <button onClick={handleClick}>Interactive Button</button>
}
```

#### Key Rules:
1. **Never export metadata from client components** - It will be silently ignored
2. **Split pages when needed** - Use wrapper pattern for metadata + interactivity
3. **Minimize client components** - Only use when interactivity is required
4. **Import client components into server components** - Not vice versa
5. **Use proper file naming** - `page.tsx` for route, `*-client.tsx` for client components

## Critical Patterns

### 1. Error Handling (Go)
```go
// Always use internal/errors package
import "internal/errors"

// Return RFC 7807 compliant errors
if err != nil {
    return errors.NewValidationError("invalid input", err).
        WithDetail("license key format is invalid").
        WithField("license_key", key)
}

// In handlers, use error middleware
func (h *Handler) GetData(w http.ResponseWriter, r *http.Request) {
    data, err := h.service.GetData(r.Context())
    if err != nil {
        errors.HandleError(w, r, err)
        return
    }
    // ... success response
}
```

### 2. Context Usage
```go
// Always propagate context
func (s *Service) ProcessData(ctx context.Context, data []byte) error {
    // Add timeout
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    
    // Check context in loops
    for _, item := range items {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            // Process item
        }
    }
}
```

### 3. Structured Logging
```go
// Use slog with structured fields
slog.InfoContext(ctx, "processing started",
    "operation", "data_import",
    "trace_id", middleware.GetTraceID(ctx),
    "user_id", userID,
    "record_count", len(records),
)

// Log errors with full context
slog.ErrorContext(ctx, "processing failed",
    "error", err,
    "operation", "data_import",
    "trace_id", middleware.GetTraceID(ctx),
    "duration", time.Since(start),
)
```

### 4. Testing Patterns
```go
func TestService_ProcessData(t *testing.T) {
    tests := []struct {
        name    string
        input   []byte
        want    *Result
        wantErr error
    }{
        {
            name:  "valid data",
            input: []byte(`{"id": 1}`),
            want:  &Result{ID: 1},
        },
        {
            name:    "invalid data",
            input:   []byte(`invalid`),
            wantErr: errors.ErrInvalidFormat,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            svc := NewService()
            got, err := svc.ProcessData(context.Background(), tt.input)
            
            if tt.wantErr != nil {
                require.Error(t, err)
                assert.ErrorIs(t, err, tt.wantErr)
                return
            }
            
            require.NoError(t, err)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

## Error Handling Best Practices

### 1. Use Custom Error Types
```go
// Define in internal/errors
type ValidationError struct {
    BaseError
    Field string
    Value interface{}
}

type NotFoundError struct {
    BaseError
    Resource string
    ID       string
}

// Usage
return &NotFoundError{
    Resource: "report",
    ID:       reportID,
    BaseError: BaseError{
        Message: "report not found",
        Code:    "REPORT_NOT_FOUND",
    },
}
```

### 2. Error Wrapping
```go
// Wrap errors with context
if err := db.Query(ctx, query); err != nil {
    return fmt.Errorf("query reports: %w", err)
}

// Check wrapped errors
if errors.Is(err, sql.ErrNoRows) {
    return NewNotFoundError("report", id)
}
```

### 3. HTTP Error Responses
```go
// All HTTP errors must follow RFC 7807
{
    "type": "/errors/validation-failed",
    "title": "Validation Failed",
    "status": 400,
    "detail": "The license key format is invalid",
    "instance": "/api/v1/license/activate",
    "trace_id": "abc123",
    "errors": {
        "license_key": "must be in format XXXX-XXXX-XXXX"
    }
}
```

## Interface & Dependency Management

### 1. Interface Definition
```go
// Define interfaces where they're used, not where implemented
package service

type Repository interface {
    GetReport(ctx context.Context, id string) (*Report, error)
    SaveReport(ctx context.Context, report *Report) error
}

// Accept interfaces, return structs
func NewService(repo Repository) *Service {
    return &Service{repo: repo}
}
```

### 2. Dependency Injection
```go
// Use constructor injection
type Service struct {
    repo   Repository
    cache  Cache
    logger *slog.Logger
}

func NewService(repo Repository, cache Cache, logger *slog.Logger) *Service {
    return &Service{
        repo:   repo,
        cache:  cache,
        logger: logger,
    }
}
```

### 3. Wire Everything in main()
```go
func main() {
    // Initialize dependencies
    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
    db := database.New(config.DB)
    cache := cache.NewRedis(config.Redis)
    
    // Wire services
    repo := repository.New(db)
    svc := service.New(repo, cache, logger)
    handler := http.NewHandler(svc, logger)
    
    // Start server
    server := http.NewServer(handler, config.Server)
    server.Start()
}
```

## Testing Requirements

### 1. Coverage Requirements
- Minimum 80% coverage for all packages
- 90% for critical paths (licensing, operations, security)
- 100% for security-related functions

### 2. Test Types
```go
// Unit tests - internal/service/report_test.go
func TestReportService_Generate(t *testing.T) {
    // Test with mocks
}

// Integration tests - internal/integration/
func TestReportAPI_EndToEnd(t *testing.T) {
    // Test with real database
}

// Benchmark tests
func BenchmarkReportGeneration(b *testing.B) {
    // Performance testing
}
```

### 3. Test Helpers
```go
// Use testify for assertions
assert.Equal(t, expected, actual)
require.NoError(t, err)

// Use mock interfaces
type MockRepo struct {
    mock.Mock
}

func (m *MockRepo) GetReport(ctx context.Context, id string) (*Report, error) {
    args := m.Called(ctx, id)
    return args.Get(0).(*Report), args.Error(1)
}
```

## Configuration Management

### 1. Environment Variables
```go
// Use internal/config package
type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
    License  LicenseConfig
}

func Load() (*Config, error) {
    return &Config{
        Server: ServerConfig{
            Port: getEnvOrDefault("PORT", "8080"),
            Host: getEnvOrDefault("HOST", "0.0.0.0"),
        },
        Database: DatabaseConfig{
            URL: getEnvRequired("DATABASE_URL"),
        },
    }, nil
}
```

### 2. Configuration Files
```json
// config/production.json
{
    "server": {
        "port": 8080,
        "read_timeout": "30s",
        "write_timeout": "30s"
    },
    "license": {
        "check_interval": "1h",
        "grace_period": "7d"
    }
}
```

### 3. Secrets Management
```go
// Always encrypt sensitive data
encrypted, err := security.Encrypt([]byte(apiKey))
if err != nil {
    return fmt.Errorf("encrypt api key: %w", err)
}

// Store encrypted in files
if err := os.WriteFile("credentials.dat", encrypted, 0600); err != nil {
    return fmt.Errorf("save credentials: %w", err)
}
```

## Integration Patterns

### 1. External APIs
```go
// Use circuit breaker for resilience
client := resty.New().
    SetTimeout(30 * time.Second).
    SetRetryCount(3).
    OnBeforeRequest(func(c *resty.Client, r *resty.Request) error {
        r.SetContext(ctx)
        r.SetHeader("X-Trace-ID", middleware.GetTraceID(ctx))
        return nil
    })
```

### 2. Database Patterns
```go
// Use transactions for consistency
tx, err := db.BeginTx(ctx, nil)
if err != nil {
    return fmt.Errorf("begin transaction: %w", err)
}
defer tx.Rollback()

// Operations...

if err := tx.Commit(); err != nil {
    return fmt.Errorf("commit transaction: %w", err)
}
```

### 3. Message Queue Patterns
```go
// Use channels for internal communication
type JobQueue struct {
    jobs chan Job
    done chan struct{}
}

func (q *JobQueue) Process(ctx context.Context, workers int) {
    var wg sync.WaitGroup
    
    for i := 0; i < workers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            q.worker(ctx)
        }()
    }
    
    wg.Wait()
}
```

## Performance Guidelines

### 1. Concurrent Processing
```go
// Use worker pools for CPU-bound tasks
func ProcessReports(ctx context.Context, reports []Report) error {
    const workers = 10
    jobs := make(chan Report, len(reports))
    results := make(chan error, len(reports))
    
    // Start workers
    var wg sync.WaitGroup
    for i := 0; i < workers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for report := range jobs {
                results <- processReport(ctx, report)
            }
        }()
    }
    
    // Send jobs
    for _, report := range reports {
        jobs <- report
    }
    close(jobs)
    
    // Wait and collect results
    wg.Wait()
    close(results)
    
    for err := range results {
        if err != nil {
            return err
        }
    }
    
    return nil
}
```

### 2. Memory Management
```go
// Use sync.Pool for frequently allocated objects
var bufferPool = sync.Pool{
    New: func() interface{} {
        return new(bytes.Buffer)
    },
}

func ProcessData(data []byte) {
    buf := bufferPool.Get().(*bytes.Buffer)
    defer func() {
        buf.Reset()
        bufferPool.Put(buf)
    }()
    
    // Use buffer...
}
```

### 3. Caching Strategies
```go
// Use in-memory cache with TTL
type Cache struct {
    data sync.Map
    ttl  time.Duration
}

func (c *Cache) Get(key string) (interface{}, bool) {
    if val, ok := c.data.Load(key); ok {
        item := val.(*cacheItem)
        if time.Now().Before(item.expiry) {
            return item.value, true
        }
        c.data.Delete(key)
    }
    return nil, false
}
```

## Security Considerations

### 1. Authentication & Authorization
```go
// Use middleware for auth checks
func RequireLicense(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if !license.IsValid(r.Context()) {
            errors.HandleError(w, r, errors.ErrUnauthorized)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

### 2. Input Validation
```go
// Validate all inputs
func ValidateLicenseKey(key string) error {
    if len(key) != 29 {
        return errors.NewValidationError("invalid length")
    }
    
    pattern := `^[A-Z0-9]{5}-[A-Z0-9]{5}-[A-Z0-9]{5}-[A-Z0-9]{5}-[A-Z0-9]{5}$`
    if !regexp.MustCompile(pattern).MatchString(key) {
        return errors.NewValidationError("invalid format")
    }
    
    return nil
}
```

### 3. Encryption
```go
// Use AES-256-GCM for encryption
func Encrypt(plaintext []byte) ([]byte, error) {
    key := deriveKey()
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, err
    }
    
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }
    
    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return nil, err
    }
    
    return gcm.Seal(nonce, nonce, plaintext, nil), nil
}
```

## Common Workflows

### 1. Adding a New Endpoint
```go
// 1. Define contract in pkg/contracts
type CreateReportRequest struct {
    Title string `json:"title" validate:"required"`
    Data  []byte `json:"data" validate:"required"`
}

// 2. Add service method
func (s *Service) CreateReport(ctx context.Context, req *CreateReportRequest) (*Report, error) {
    // Implementation
}

// 3. Add handler
func (h *Handler) CreateReport(w http.ResponseWriter, r *http.Request) {
    var req CreateReportRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        errors.HandleError(w, r, errors.NewValidationError("invalid request", err))
        return
    }
    
    report, err := h.service.CreateReport(r.Context(), &req)
    if err != nil {
        errors.HandleError(w, r, err)
        return
    }
    
    respond.JSON(w, http.StatusCreated, report)
}

// 4. Add route
r.Post("/api/v1/reports", handler.CreateReport)

// 5. Add tests
func TestCreateReport(t *testing.T) {
    // Test implementation
}
```

### 2. Adding a New Operation
```go
// 1. Define operation in internal/operations
type DataImportOperation struct {
    BaseOperation
}

func (op *DataImportOperation) Execute(ctx context.Context) error {
    // Implementation
}

// 2. Register operation
registry.Register("data_import", NewDataImportOperation)

// 3. Add to pipeline
pipeline.AddStage("import", &DataImportOperation{})
```

## Observability

### 1. Structured Logging
```go
// Use slog with consistent fields
logger := slog.With(
    "service", "report",
    "version", config.Version,
)

// Log with context
logger.InfoContext(ctx, "report generated",
    "report_id", report.ID,
    "duration", time.Since(start),
    "size_bytes", len(data),
)
```

### 2. Metrics
```go
// Use OpenTelemetry for metrics
meter := otel.Meter("isx-scrapper")

requestCounter, _ := meter.Int64Counter("http_requests_total",
    metric.WithDescription("Total HTTP requests"),
)

requestCounter.Add(ctx, 1,
    attribute.String("method", r.Method),
    attribute.String("path", r.URL.Path),
    attribute.Int("status", status),
)
```

### 3. Distributed Tracing
```go
// Use OpenTelemetry for tracing
tracer := otel.Tracer("isx-scrapper")

ctx, span := tracer.Start(ctx, "ProcessReport",
    trace.WithAttributes(
        attribute.String("report.id", reportID),
    ),
)
defer span.End()

// Add events
span.AddEvent("validation_completed")

// Record errors
if err != nil {
    span.RecordError(err)
    span.SetStatus(codes.Error, err.Error())
}
```

## Monitoring & Alerting

### 1. Health Checks
```go
// Implement health endpoint
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
    checks := map[string]string{
        "database": h.checkDatabase(r.Context()),
        "license":  h.checkLicense(r.Context()),
        "storage":  h.checkStorage(r.Context()),
    }
    
    status := http.StatusOK
    for _, check := range checks {
        if check != "ok" {
            status = http.StatusServiceUnavailable
            break
        }
    }
    
    respond.JSON(w, status, checks)
}
```

### 2. Metrics Endpoints
```go
// Expose Prometheus metrics
import "github.com/prometheus/client_golang/prometheus/promhttp"

r.Handle("/metrics", promhttp.Handler())
```

### 3. Alert Rules
```yaml
# Example Prometheus rules
groups:
  - name: isx-scrapper
    rules:
      - alert: HighErrorRate
        expr: rate(http_requests_total{status=~"5.."}[5m]) > 0.05
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: High error rate detected
```

## Documentation Standards

### 1. Code Documentation
```go
// Package reports handles report generation and management.
// It provides functionality for creating, storing, and retrieving
// financial reports from ISX data.
package reports

// GenerateReport creates a new report from the provided data.
// It validates the input, processes the data, and stores the result.
//
// Example:
//
//	report, err := svc.GenerateReport(ctx, data)
//	if err != nil {
//	    return fmt.Errorf("generate report: %w", err)
//	}
func (s *Service) GenerateReport(ctx context.Context, data []byte) (*Report, error) {
    // Implementation
}
```

### 2. API Documentation
```go
// Use OpenAPI annotations
// @Summary Create a new report
// @Description Creates a new financial report from uploaded data
// @Tags reports
// @Accept json
// @Produce json
// @Param request body CreateReportRequest true "Report data"
// @Success 201 {object} Report
// @Failure 400 {object} errors.ProblemDetails
// @Failure 401 {object} errors.ProblemDetails
// @Router /api/v1/reports [post]
func (h *Handler) CreateReport(w http.ResponseWriter, r *http.Request) {
    // Implementation
}
```

### 3. README Files
Every package must have a README.md explaining:
- Purpose and responsibilities
- Key types and interfaces
- Usage examples
- Dependencies
- Testing instructions

## Code Review Checklist

### Before Submitting PR
- [ ] All tests pass with `go test ./... -race`
- [ ] Test coverage > 80% for new code
- [ ] No linting errors (`golangci-lint run`)
- [ ] Documentation updated (code comments, README)
- [ ] Error handling follows RFC 7807
- [ ] Logging includes trace_id and context
- [ ] No sensitive data in logs
- [ ] Security considerations addressed
- [ ] Performance impact considered
- [ ] Database migrations included if needed

### Review Focus Areas
1. **Security**: Auth, input validation, encryption
2. **Error Handling**: Proper error types, wrapping
3. **Testing**: Coverage, edge cases, benchmarks
4. **Performance**: Concurrency, memory usage
5. **Documentation**: Clear, complete, accurate
6. **React Hydration**: Check for unguarded Date/time operations, useHydration usage

## Version Management

### 1. Semantic Versioning
```go
// version/version.go
package version

var (
    Version   = "3.0.0"
    GitCommit = "unknown"
    BuildTime = "unknown"
)

// Set during build
// go build -ldflags "-X version.GitCommit=$(git rev-parse HEAD)"
```

### 2. API Versioning
```go
// Use URL path versioning
r.Route("/api/v1", func(r chi.Router) {
    r.Mount("/reports", reportHandler)
    r.Mount("/operations", operationHandler)
})

// Support multiple versions
r.Route("/api/v2", func(r chi.Router) {
    // V2 endpoints
})
```

### 3. Database Migrations
```sql
-- migrations/001_initial_schema.up.sql
CREATE TABLE reports (
    id UUID PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- migrations/001_initial_schema.down.sql
DROP TABLE reports;
```

## Debugging & Troubleshooting

### 1. Debug Logging
```go
// Use debug level for detailed info
slog.DebugContext(ctx, "processing record",
    "record_id", record.ID,
    "raw_data", hex.EncodeToString(data),
)

// Enable in development
slog.SetLogLevel(slog.LevelDebug)
```

### 2. Profiling
```go
import _ "net/http/pprof"

// Add pprof endpoints
r.Mount("/debug", middleware.Profiler())

// CPU profiling
go tool pprof http://localhost:8080/debug/pprof/profile

// Memory profiling
go tool pprof http://localhost:8080/debug/pprof/heap
```

### 3. Trace Analysis
```go
// Add trace context to all operations
ctx = trace.ContextWithSpan(ctx, span)

// Use trace ID in logs
traceID := span.SpanContext().TraceID().String()
logger.InfoContext(ctx, "operation started",
    "trace_id", traceID,
)
```

## Database/Storage Patterns

### 1. Connection Management
```go
// Use connection pooling
db, err := sql.Open("postgres", dsn)
if err != nil {
    return fmt.Errorf("open database: %w", err)
}

db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(5 * time.Minute)
```

### 2. Query Patterns
```go
// Use prepared statements
stmt, err := db.PrepareContext(ctx, `
    SELECT id, title, created_at
    FROM reports
    WHERE user_id = $1 AND status = $2
`)
if err != nil {
    return fmt.Errorf("prepare statement: %w", err)
}
defer stmt.Close()

// Use scanning helpers
var reports []Report
rows, err := stmt.QueryContext(ctx, userID, status)
if err != nil {
    return fmt.Errorf("query reports: %w", err)
}
defer rows.Close()

for rows.Next() {
    var r Report
    if err := rows.Scan(&r.ID, &r.Title, &r.CreatedAt); err != nil {
        return fmt.Errorf("scan report: %w", err)
    }
    reports = append(reports, r)
}
```

### 3. Migration Management
```go
// Use golang-migrate
import "github.com/golang-migrate/migrate/v4"

func RunMigrations(dbURL string) error {
    m, err := migrate.New(
        "file://migrations",
        dbURL,
    )
    if err != nil {
        return fmt.Errorf("create migrator: %w", err)
    }
    
    if err := m.Up(); err != nil && err != migrate.ErrNoChange {
        return fmt.Errorf("run migrations: %w", err)
    }
    
    return nil
}
```

## Deployment Notes

### Production Build
1. Set credentials in `encrypted_credentials.dat`
2. Run `build.bat` for complete build
3. Frontend embedded in `web-licensed.exe`
4. All paths relative to executable

### Directory Structure
```
release/
├── web-licensed.exe    # Main server
├── scraper.exe        # ISX scraper
├── process.exe        # Data processor  
├── indexcsv.exe       # Index extractor
├── data/
│   ├── downloads/     # Excel files
│   └── reports/       # CSV outputs
├── logs/              # Application logs
└── web/               # Static assets (backup)
```

### Configuration Files
- `credentials.json`: Google Sheets API
- `sheets-config.json`: Sheet ID mappings
- `license.dat`: Activated license data

## React Hydration Best Practices

When developing React components with Next.js SSR, follow these patterns to prevent hydration errors (#418, #423):

### 1. Use Hydration State Management

The project provides a reusable hydration hook in `@/lib/hooks`:

```typescript
import { useHydration } from '@/lib/hooks'

function MyComponent() {
  const isHydrated = useHydration()
  
  if (!isHydrated) {
    return <LoadingState />
  }
  
  // Client-only content here
}
```

Or implement manually:
```typescript
const [isHydrated, setIsHydrated] = useState(false)
useEffect(() => {
  setIsHydrated(true)
}, [])
```

### 2. Guard Dynamic Operations
- **Date operations**: `isHydrated ? new Date().toISOString() : ''`
- **WebSocket updates**: Add `if (!isHydrated) return` at the start of effects
- **API calls**: Delay data fetching until after hydration
- **Dynamic content**: Show consistent loading state until hydrated

### 3. Pre-Hydration Loading State
```typescript
// Early return with loading state
if (!isHydrated) {
  return (
    <div className="min-h-screen p-8 flex items-center justify-center">
      <div className="text-center">
        <Loader2 className="h-8 w-8 animate-spin mx-auto mb-4" />
        <p className="text-muted-foreground">Initializing...</p>
      </div>
    </div>
  )
}
```

### 4. Include Dependencies
Always include `isHydrated` in dependency arrays when used inside callbacks:
```typescript
const handleOperation = useCallback((data) => {
  const date = isHydrated ? new Date().toISOString() : ''
  // ... rest of logic
}, [isHydrated])  // Include in dependencies
```

### 5. Common Patterns to Avoid
- Don't use `Date.now()` or `new Date()` without hydration guards
- Don't fetch data in component body - use `useEffect` with hydration check
- Don't render different content based on `typeof window !== 'undefined'`
- Don't update state from WebSocket before hydration completes

### 6. Testing for Hydration Issues
1. Build with `./build.bat`
2. Clear browser cache and cookies
3. Open browser console before loading page
4. Look for React errors #418 (hydration mismatch) or #423 (text content mismatch)
5. Check for "Warning: Text content did not match" messages

## Important Notes

- **🚨 BUILD RULES**: NEVER build in api/ or web/ - ALWAYS use ./build.bat from root (see BUILD_RULES.md)
- **🚨 NO DEV BUILDS**: Run `./tools/verify-no-dev-builds.bat` to check compliance
- **🚨 LOGS CLEARED**: Every build via ./build.bat automatically clears all logs
- **No Blind Sleeps**: Use context, channels, or timers
- **Test Before Push**: All tests must pass with race detector
- **Update Docs**: Keep README files current - docs in same PR as code
- **Path Resolution**: Always use `paths.GetBaseDir()`
- **WebSocket Reliability**: Implement reconnection logic
- **License Checks**: Validate before operations
- **Chi Only**: Use Chi v5 for all HTTP routing - no other frameworks
- **slog Only**: Use slog for all logging - no fmt.Println or log.Printf
- **RFC 7807**: All API errors must follow RFC 7807 Problem Details
- **TDD Always**: Write tests first, then implementation
- **Context Everywhere**: Pass context.Context as first parameter
- **Structured Logging**: Include trace_id, operation, and relevant fields
- **Interface First**: Define interfaces before implementations
- **No Global State**: Use dependency injection
- **Clean Resources**: Always defer cleanup
- **Benchmark Critical Paths**: Performance matters
- **Project Structure**: See `FILE_INDEX.md` for complete directory organization
- **Command Line**: We are using bash now windows CMD in implementing our commands
- **React Hydration**: Use `useHydration` hook from `@/lib/hooks` for client-only content
- **Hydration Testing**: Always test production builds for React errors #418 and #423
- **Build Location**: NEVER build in `api/` or `web/` directories - always use `./build.bat` to output to `dist/`
- **Frontend Embedding**: Use explicit patterns like `//go:embed frontend/*.html frontend/_next`
- **Empty Directories**: Build process removes empty dirs that Next.js creates
- **Embed Standards**: Follow Grafana/CockroachDB pattern for production embedding
- **Validation Required**: Frontend build must pass validation before embedding

## Project Structure

The project follows a clean, professional Go structure. For detailed file organization and descriptions, see `FILE_INDEX.md`.

### Key Directories
- `api/` - Go backend source code
- `web/` - Next.js frontend source code
- `docs/` - All documentation
- `tools/` - Development utilities and scripts
- `installer/` - Windows installer configuration
- Root directory contains only essential files (12 files)

### Build System
- Single `build.go` file (577 lines) handles all build operations
- Simple `build.bat` wrapper for Windows users
- Targets: all, web, frontend, scraper, processor, indexcsv, test, clean, release

For complete details about file organization, removed files, and structural improvements, refer to `FILE_INDEX.md`.