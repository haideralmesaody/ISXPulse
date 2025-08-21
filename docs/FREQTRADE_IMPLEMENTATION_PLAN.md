# COMPREHENSIVE FREQTRADE BACKTESTING IMPLEMENTATION PLAN
## Following CLAUDE.md Requirements & Industry Best Practices

---

## EXECUTIVE SUMMARY
This document provides a complete implementation plan for the ISX Pulse Freqtrade backtesting system, strictly adhering to CLAUDE.md requirements and industry best practices. Each task includes acceptance criteria, dependencies, and tracking mechanisms.

---

## 🎯 PROJECT OBJECTIVES
1. **Complete Freqtrade Integration** with 130+ technical indicators
2. **Professional UI/UX** following Next.js 14 patterns
3. **Enterprise Security** with encrypted data and license protection
4. **90% Test Coverage** for critical paths (per CLAUDE.md)
5. **Full Observability** with structured logging and OpenTelemetry

---

## 📋 TASK TRACKING SYSTEM

### Task Status Legend:
- ⬜ **NOT_STARTED** - Task not begun
- 🟦 **IN_PROGRESS** - Currently working
- ✅ **COMPLETED** - Finished and tested
- 🔄 **IN_REVIEW** - Code review pending
- ❌ **BLOCKED** - Waiting on dependencies
- 🟨 **NEEDS_REVISION** - Failed review

### Priority Levels:
- 🔴 **P0** - Critical path, blocks other work
- 🟠 **P1** - High priority, core functionality
- 🟡 **P2** - Medium priority, enhanced features
- 🟢 **P3** - Low priority, nice to have

---

## PHASE 1: FRONTEND COMPONENTS [CLAUDE.md COMPLIANT]

### Task 1.1: BacktestChart Component 🔴 P0
**Status:** ⬜ NOT_STARTED  
**Assignee:** TBD  
**Est. Time:** 3 hours  
**Dependencies:** lightweight-charts (already in package.json)

#### Requirements (Per CLAUDE.md Standards):
```typescript
// File: web/components/backtester/BacktestChart.tsx
'use client'  // Client component for interactivity

import { useHydration } from '@/lib/hooks/use-hydration'  // MANDATORY hydration hook
import { useCallback, useMemo, useRef } from 'react'
import dynamic from 'next/dynamic'

// CLAUDE.md React Hydration Best Practice
const LightweightCharts = dynamic(
  () => import('lightweight-charts'),
  { 
    ssr: false,  // Prevent SSR issues
    loading: () => <ChartSkeleton />  // Always provide loading state
  }
)
```

#### Acceptance Criteria:
- [ ] Uses `useHydration` hook for client-only content
- [ ] No direct Date operations without hydration guard
- [ ] Implements virtual scrolling for > 1000 candles
- [ ] Exports chart to PNG/CSV per requirements
- [ ] Structured logging with slog pattern
- [ ] Error boundaries implemented
- [ ] TypeScript strict mode - no `any` types

#### Testing Requirements (80% minimum):
```typescript
// web/components/backtester/__tests__/BacktestChart.test.tsx
describe('BacktestChart', () => {
  // Unit tests
  it('renders without hydration errors')
  it('displays trade markers correctly')
  it('handles empty data gracefully')
  it('exports chart data successfully')
  
  // Performance tests
  it('renders 10,000 candles in < 2 seconds')
  it('maintains 60fps during zoom/pan')
})
```

---

### Task 1.2: PerformanceMetrics Component 🔴 P0
**Status:** ⬜ NOT_STARTED  
**Est. Time:** 2 hours  
**Dependencies:** None

#### Requirements (Industry Best Practices):
```typescript
// Implement comprehensive risk metrics per institutional standards
interface PerformanceMetrics {
  // Sharpe Ratio (Risk-adjusted returns)
  sharpeRatio: number  // Target: > 1.0 good, > 2.0 excellent
  
  // Sortino Ratio (Downside risk)
  sortinoRatio: number  // Better than Sharpe for daily data
  
  // Maximum Drawdown
  maxDrawdown: {
    value: number      // Peak to trough %
    duration: number   // Days in drawdown
    recovery: number   // Days to recover
  }
  
  // Value at Risk (95% confidence)
  var95: number  // Maximum expected loss
  cvar95: number // Conditional VaR (tail risk)
}
```

#### Error Handling (RFC 7807 compliant):
```typescript
// All API errors must follow RFC 7807 Problem Details
interface ProblemDetails {
  type: string
  title: string
  status: number
  detail: string
  instance: string
  trace_id: string  // From OpenTelemetry
}
```

---

### Task 1.3: TradeList Component 🟠 P1
**Status:** ⬜ NOT_STARTED  
**Est. Time:** 2 hours  
**Dependencies:** react-window for virtualization

#### Performance Requirements:
- Virtual scrolling for > 100 trades
- Lazy loading of trade details
- Memoized sorting/filtering
- Export to CSV in < 1 second for 10,000 trades

---

## PHASE 2: GO BACKEND INTEGRATION [CHI V5 ONLY]

### Task 2.1: Backtest Handler 🔴 P0
**Status:** ⬜ NOT_STARTED  
**Est. Time:** 3 hours  
**Dependencies:** Freqtrade client (✅ COMPLETED)

#### CLAUDE.md Mandatory Patterns:
```go
// api/internal/transport/http/backtest_handler.go
package http

import (
    "context"
    "log/slog"  // ONLY slog, no fmt.Println or log.Printf
    "net/http"
    
    "github.com/go-chi/chi/v5"  // ONLY Chi v5, no Gin/Echo
    "github.com/go-chi/render"
    
    "isxcli/internal/errors"  // RFC 7807 errors MANDATORY
    customMiddleware "isxcli/internal/middleware"
)

type BacktestHandler struct {
    service BacktestService
    logger  *slog.Logger  // Single slog instance per CLAUDE.md
}

// Context ALWAYS first parameter
func (h *BacktestHandler) RunBacktest(ctx context.Context, req BacktestRequest) (*BacktestResult, error) {
    // Structured logging with trace_id
    h.logger.InfoContext(ctx, "starting backtest",
        "trace_id", middleware.GetTraceID(ctx),
        "strategy", req.Strategy,
        "tickers", req.Tickers,
    )
    
    // Error wrapping with context
    if err := h.validateRequest(ctx, req); err != nil {
        return nil, fmt.Errorf("validate backtest request: %w", err)
    }
    
    // RFC 7807 error response
    if err != nil {
        return nil, errors.NewValidationError("invalid backtest parameters", err).
            WithDetail("strategy parameters out of range").
            WithField("strategy", req.Strategy)
    }
}
```

#### Security Requirements:
- [ ] License validation before execution
- [ ] Input sanitization for all parameters
- [ ] Rate limiting (max 10 backtests/minute)
- [ ] Resource limits (max 100MB memory per backtest)

---

### Task 2.2: WebSocket Proxy 🔴 P0  
**Status:** ⬜ NOT_STARTED  
**Est. Time:** 2 hours

#### Goroutine Management (CLAUDE.md):
```go
// Proper goroutine lifecycle management
func (p *WebSocketProxy) Start(ctx context.Context) error {
    var wg sync.WaitGroup
    
    // Error channel for goroutine errors
    errCh := make(chan error, 2)
    
    // Frontend -> Python forwarding
    wg.Add(1)
    go func() {
        defer wg.Done()
        if err := p.forwardToPython(ctx); err != nil {
            errCh <- fmt.Errorf("forward to python: %w", err)
        }
    }()
    
    // Python -> Frontend forwarding
    wg.Add(1)
    go func() {
        defer wg.Done()
        if err := p.forwardToFrontend(ctx); err != nil {
            errCh <- fmt.Errorf("forward to frontend: %w", err)
        }
    }()
    
    // Wait for context or error
    select {
    case <-ctx.Done():
        return ctx.Err()
    case err := <-errCh:
        return err
    }
}
```

---

## PHASE 3: TESTING REQUIREMENTS [90% COVERAGE]

### Task 3.1: Unit Tests 🔴 P0
**Status:** ⬜ NOT_STARTED  
**Est. Time:** 4 hours

#### Test Structure (CLAUDE.md Table-Driven):
```go
func TestBacktestHandler_RunBacktest(t *testing.T) {
    tests := []struct {
        name    string
        input   BacktestRequest
        want    *BacktestResult
        wantErr error
    }{
        {
            name: "valid momentum strategy",
            input: BacktestRequest{
                Strategy: "ISXMomentumBreakout",
                Tickers:  []string{"BGUC", "BNOI"},
                // ...
            },
            want: &BacktestResult{
                TotalReturn: 15.5,
                SharpeRatio: 1.85,
            },
        },
        {
            name: "invalid date range",
            input: BacktestRequest{
                StartDate: "2025-01-01",
                EndDate:   "2024-01-01",
            },
            wantErr: errors.ErrInvalidDateRange,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Parallel tests where possible
            t.Parallel()
            
            handler := NewBacktestHandler(mockService, testLogger)
            got, err := handler.RunBacktest(context.Background(), tt.input)
            
            // Use testify for assertions
            if tt.wantErr != nil {
                require.Error(t, err)
                assert.ErrorIs(t, err, tt.wantErr)
                return
            }
            
            require.NoError(t, err)
            assert.Equal(t, tt.want.TotalReturn, got.TotalReturn)
        })
    }
}
```

#### Coverage Requirements:
- **90%** for security-critical paths (licensing, auth)
- **80%** for business logic
- **100%** for error handling paths

---

## PHASE 4: OBSERVABILITY [OPENTELEMETRY]

### Task 4.1: Structured Logging 🟠 P1
**Status:** ⬜ NOT_STARTED  
**Est. Time:** 2 hours

#### Logging Standards (slog only):
```go
// CONSISTENT field names across application
logger.InfoContext(ctx, "backtest completed",
    "operation", "backtest.run",           // Operation name
    "trace_id", trace.SpanContext().TraceID(), // Always include
    "user_id", auth.GetUserID(ctx),       // User context
    "duration_ms", time.Since(start).Milliseconds(),
    "strategy", strategy,
    "total_trades", result.TotalTrades,
    "profit_pct", result.TotalReturn,
)

// Error logging with full context
logger.ErrorContext(ctx, "backtest failed",
    "error", err,                          // Full error
    "operation", "backtest.run",
    "trace_id", trace.SpanContext().TraceID(),
    "strategy", strategy,
    "stage", "optimization",               // Where it failed
    "attempt", retryCount,
)
```

### Task 4.2: Metrics & Tracing 🟠 P1
**Status:** ⬜ NOT_STARTED  
**Est. Time:** 2 hours

```go
// OpenTelemetry metrics
meter := otel.Meter("isx-backtesting")

backtestCounter, _ := meter.Int64Counter("backtest.executions",
    metric.WithDescription("Total backtests executed"),
)

backtestDuration, _ := meter.Float64Histogram("backtest.duration",
    metric.WithDescription("Backtest execution time"),
    metric.WithUnit("seconds"),
)

// Record metrics
backtestCounter.Add(ctx, 1,
    attribute.String("strategy", strategy),
    attribute.Bool("optimized", isOptimized),
)
```

---

## PHASE 5: BUILD SYSTEM INTEGRATION

### Task 5.1: Python Service Integration 🔴 P0
**Status:** ⬜ NOT_STARTED  
**Est. Time:** 1 hour

#### build.bat Modifications (CLAUDE.md MANDATORY):
```batch
@echo off
REM CLAUDE.md: NEVER build in api/ or web/ directories

REM Check Python
:check_python
where python >nul 2>&1
if %errorlevel% neq 0 (
    echo [WARNING] Python not found - Backtesting unavailable
    set PYTHON_AVAILABLE=false
) else (
    python -c "import sys; exit(0 if sys.version_info >= (3,8) else 1)"
    if %errorlevel% neq 0 (
        echo [ERROR] Python 3.8+ required for backtesting
        exit /b 1
    )
    set PYTHON_AVAILABLE=true
)

REM Install TA-Lib binary (Windows specific)
if "%PYTHON_AVAILABLE%"=="true" (
    python -c "import talib" >nul 2>&1
    if %errorlevel% neq 0 (
        echo Installing TA-Lib...
        pip install TA-Lib-0.4.24-cp38-cp38-win_amd64.whl
    )
)

REM ALWAYS build to dist/ directory
set OUTPUT_DIR=dist
if not exist %OUTPUT_DIR% mkdir %OUTPUT_DIR%

REM Clear logs before build (CLAUDE.md requirement)
echo Clearing logs...
del /F /Q logs\*.log 2>nul
```

---

## PHASE 6: SECURITY & PERFORMANCE

### Task 6.1: Security Hardening 🔴 P0
**Status:** ⬜ NOT_STARTED  
**Est. Time:** 3 hours

#### Security Checklist:
- [ ] **License Validation**: Check before every backtest
- [ ] **Input Validation**: Sanitize all user inputs
- [ ] **Rate Limiting**: Max 10 backtests/minute per user
- [ ] **Resource Limits**: CPU/Memory caps per backtest
- [ ] **Encryption**: All stored strategies encrypted with AES-256
- [ ] **Audit Logging**: Every backtest logged with user/timestamp
- [ ] **CORS**: Restricted to localhost and production domain

### Task 6.2: Performance Optimization 🟠 P1
**Status:** ⬜ NOT_STARTED  
**Est. Time:** 2 hours

#### Performance Targets:
- Backtest 1 year daily data: < 10 seconds
- Optimization (100 iterations): < 5 minutes
- Chart rendering (10K candles): < 2 seconds
- WebSocket latency: < 100ms
- Memory usage: < 500MB per session

---

## PHASE 7: DOCUMENTATION

### Task 7.1: API Documentation 🟡 P2
**Status:** ⬜ NOT_STARTED  
**Est. Time:** 2 hours

```go
// OpenAPI annotations (CLAUDE.md standard)
// @Summary Run backtest
// @Description Execute trading strategy backtest
// @Tags backtesting
// @Accept json
// @Produce json
// @Param request body BacktestRequest true "Backtest configuration"
// @Success 200 {object} BacktestResult
// @Failure 400 {object} errors.ProblemDetails "RFC 7807 error"
// @Failure 401 {object} errors.ProblemDetails "Unauthorized"
// @Router /api/v1/backtesting/run [post]
// @Security BearerAuth
func (h *BacktestHandler) RunBacktest(w http.ResponseWriter, r *http.Request)
```

### Task 7.2: User Guide 🟡 P2
**Status:** ⬜ NOT_STARTED  
**Est. Time:** 1 hour

Create `docs/BACKTESTING_GUIDE.md` with:
- Getting started walkthrough
- Strategy explanations
- Metric interpretations
- Troubleshooting guide

---

## 📊 PROGRESS TRACKING DASHBOARD

### Overall Progress: 15% Complete

| Phase | Tasks | Not Started | In Progress | Completed | Blocked |
|-------|-------|-------------|-------------|-----------|---------|
| Frontend | 6 | 4 | 1 | 1 | 0 |
| Backend | 4 | 4 | 0 | 0 | 0 |
| Testing | 3 | 3 | 0 | 0 | 0 |
| Observability | 2 | 2 | 0 | 0 | 0 |
| Build | 1 | 1 | 0 | 0 | 0 |
| Security | 2 | 2 | 0 | 0 | 0 |
| Docs | 2 | 2 | 0 | 0 | 0 |
| **TOTAL** | **20** | **18** | **1** | **1** | **0** |

### Critical Path Items (P0):
1. ⬜ BacktestChart Component
2. ⬜ PerformanceMetrics Component  
3. ⬜ Backtest Handler (Go)
4. ⬜ WebSocket Proxy
5. ⬜ Unit Tests (90% coverage)
6. ⬜ Python Service Integration
7. ⬜ Security Hardening

### Daily Standup Template:
```markdown
## Date: [DATE]
### Yesterday:
- Completed: [TASK IDs]
- Challenges: [BLOCKERS]

### Today:
- Working on: [TASK IDs]
- Goal: [SPECIFIC OUTCOMES]

### Blockers:
- [ISSUE]: [IMPACT] - [RESOLUTION PLAN]

### Metrics:
- Test Coverage: X%
- Build Status: ✅/❌
- Performance: [METRICS]
```

---

## 🚀 IMPLEMENTATION TIMELINE

### Week 1 (40 hours)
**Monday-Tuesday**: Frontend Components (P0)
- [ ] Task 1.1: BacktestChart
- [ ] Task 1.2: PerformanceMetrics
- [ ] Task 1.3: TradeList

**Wednesday-Thursday**: Backend Integration (P0)
- [ ] Task 2.1: Backtest Handler
- [ ] Task 2.2: WebSocket Proxy
- [ ] Task 5.1: Build Integration

**Friday**: Testing & Security (P0)
- [ ] Task 3.1: Unit Tests
- [ ] Task 6.1: Security Hardening

### Week 2 (24 hours)
**Monday-Tuesday**: Observability & Performance
- [ ] Task 4.1: Structured Logging
- [ ] Task 4.2: Metrics & Tracing
- [ ] Task 6.2: Performance Optimization

**Wednesday**: Documentation & Review
- [ ] Task 7.1: API Documentation
- [ ] Task 7.2: User Guide
- [ ] Code Review & Fixes

---

## ✅ DEFINITION OF DONE

Each task is considered DONE when:

1. **Code Complete**
   - [ ] Follows CLAUDE.md standards
   - [ ] TypeScript strict mode (no `any`)
   - [ ] Go uses slog only (no fmt.Println)
   - [ ] Chi v5 router only
   - [ ] RFC 7807 error handling

2. **Testing**
   - [ ] Unit tests written
   - [ ] 80% minimum coverage (90% for critical)
   - [ ] Integration tests pass
   - [ ] No race conditions (`go test -race`)

3. **Documentation**
   - [ ] Code comments added
   - [ ] README updated if needed
   - [ ] API documentation current
   - [ ] CHANGELOG entry added

4. **Performance**
   - [ ] Meets performance targets
   - [ ] No memory leaks
   - [ ] Profiling completed

5. **Security**
   - [ ] Security review passed
   - [ ] No sensitive data in logs
   - [ ] Input validation complete
   - [ ] License check implemented

6. **Review**
   - [ ] Code review completed
   - [ ] CI/CD pipeline passes
   - [ ] No linting errors
   - [ ] Deployed to staging

---

## 🔧 DEVELOPMENT ENVIRONMENT SETUP

### Prerequisites Checklist:
- [ ] Go 1.21+ installed
- [ ] Node.js 18+ installed  
- [ ] Python 3.8+ installed
- [ ] TA-Lib binary installed
- [ ] Git configured
- [ ] VS Code with Go/TypeScript extensions

### First-Time Setup:
```bash
# 1. Clone repository
git clone [REPO_URL]
cd ISXDailyReportsScrapper

# 2. Install frontend dependencies
cd web
npm install
cd ..

# 3. Install Go dependencies
cd api
go mod download
cd ..

# 4. Install Python dependencies
cd api/services/freqtrade
pip install -r requirements.txt
cd ../../..

# 5. Run build (CLAUDE.md way)
./build.bat

# 6. Start development
./build.bat -target=run
```

---

## 📝 NOTES & DECISIONS

### Architectural Decisions:
1. **Why Freqtrade?** - Industry standard, 130+ indicators, battle-tested
2. **Why WebSocket?** - Real-time progress essential for UX
3. **Why Python service?** - Freqtrade is Python-only
4. **Why Chi v5?** - CLAUDE.md mandate, excellent middleware

### Known Limitations:
1. Daily data only (ISX constraint)
2. Windows-primary development
3. Python required for backtesting
4. TA-Lib binary installation complex

### Risk Mitigation:
1. **Python unavailable**: Graceful degradation, disable backtesting
2. **WebSocket fails**: Polling fallback
3. **Memory issues**: Implement result pagination
4. **Slow backtests**: Add caching layer

---

## 📞 CONTACT & ESCALATION

### Team Contacts:
- **Technical Lead**: [NAME] - [CONTACT]
- **Frontend Dev**: [NAME] - [CONTACT]  
- **Backend Dev**: [NAME] - [CONTACT]
- **DevOps**: [NAME] - [CONTACT]

### Escalation Path:
1. Try to resolve with team member
2. Escalate to Technical Lead
3. Escalate to Project Manager
4. Emergency: Use #incident Slack channel

---

## 📚 REFERENCES

### Documentation:
- [CLAUDE.md](./CLAUDE.md) - Project standards
- [Freqtrade Docs](https://www.freqtrade.io/)
- [Chi Router](https://github.com/go-chi/chi)
- [Next.js 14](https://nextjs.org/docs)
- [RFC 7807](https://datatracker.ietf.org/doc/html/rfc7807)

### Internal Docs:
- [API Specification](./docs/API_REFERENCE.md)
- [Deployment Guide](./docs/DEPLOYMENT_GUIDE.md)
- [Security Policy](./docs/SECURITY.md)

---

## 🔄 CURRENT IMPLEMENTATION STATUS

### Completed Components ✅
1. **Python Freqtrade Service** (`api/services/freqtrade/`)
   - FastAPI server with backtesting engine
   - ISX-specific strategies (Momentum, Mean Reversion)
   - WebSocket support for real-time updates

2. **Go Freqtrade Client** (`api/internal/backtesting/`)
   - Complete HTTP/WebSocket client
   - Type-safe request/response models
   - Health check functionality

3. **Frontend Page Structure** (`web/app/backtester/`)
   - Server component with SEO metadata
   - Client component with WebSocket integration
   - Loading skeletons

4. **Strategy Configuration Panel** (`web/components/backtester/StrategyPanel.tsx`)
   - Three built-in strategies
   - Parameter tuning interface
   - Multi-ticker selection
   - Optimization settings

### In Progress 🟦
- BacktestChart Component (visualization)
- PerformanceMetrics Component (results display)

### Not Started ⬜
- TradeList Component
- Go Backend API routes
- WebSocket proxy
- Build system integration
- Testing suite
- Documentation

---

This comprehensive plan ensures full compliance with CLAUDE.md requirements while following industry best practices for financial software development. Each task is trackable, measurable, and has clear acceptance criteria.