# Developer Quick Reference Card

## 🚀 Getting Started with Your Phase

### Find Your Team & Phase
1. Check the team allocation in MASTER_DEVELOPMENT_PLAN_V2.md
2. Locate your phase number and tasks
3. Verify dependencies are complete before starting

### Before You Start Any Task
- [ ] Read CLAUDE.md for standards
- [ ] Check task dependencies are ✅ 
- [ ] Pull latest from main branch
- [ ] Create feature branch: `feature/phase-X-task-Y`

---

## 📋 Task Execution Checklist

### 1. Starting a Task
```bash
# Create your branch
git checkout -b feature/phase-X-task-Y

# Update task status in plan
# Change 🔴 to 🟡 in MASTER_DEVELOPMENT_PLAN_V2.md
```

### 2. During Development
- Follow CLAUDE.md standards strictly
- Use slog for all logging (no fmt.Println!)
- Include trace_id in all log entries
- Write tests as you go (not after)

### 3. Code Standards Quick Check
```go
// ✅ GOOD - Using slog with context
slog.InfoContext(ctx, "processing request",
    "user_id", userID,
    "action", "update",
)

// ❌ BAD - Using fmt or log package
fmt.Println("processing request")
log.Printf("user %s action update", userID)
```

### 4. Completing a Task
- [ ] All acceptance criteria met
- [ ] Tests written and passing
- [ ] Code reviewed by teammate
- [ ] Update task status to 🟢
- [ ] Create PR with clear description

---

## 🔧 Common Patterns

### Logger Setup (Phase 1-2)
```go
// Get logger from context
logger := infrastructure.LoggerFromContext(ctx)

// Add fields
logger = logger.With(
    "component", "operation",
    "step", stageID,
)

// Use it
logger.InfoContext(ctx, "step started")
```

### Type Migration (Phase 3)
```go
// In old location - temporary alias
type TradeRecord = domain.TradeRecord

// In new location
package domain

type TradeRecord struct {
    ID     int       `json:"id" db:"id" validate:"required"`
    Symbol string    `json:"symbol" db:"symbol" validate:"required,max=10"`
    // ... rest of fields
}
```

### Service Migration (Phase 4)
```go
// Before
type Service struct {
    logger Logger // custom interface
}

// After  
type Service struct {
    logger *slog.Logger // concrete type
}

// Method update
func (s *Service) Process(ctx context.Context, data *domain.Data) error {
    s.logger.InfoContext(ctx, "processing started",
        "data_id", data.ID,
    )
    // ... rest of method
}
```

---

## 🚨 Critical Rules

### NEVER DO THIS
1. ❌ Create custom Logger interfaces
2. ❌ Use fmt.Print* for logging
3. ❌ Skip trace_id in logs
4. ❌ Use text format logs (always JSON)
5. ❌ Ignore test coverage requirements

### ALWAYS DO THIS
1. ✅ Use slog.Logger directly
2. ✅ Pass context everywhere
3. ✅ Include trace_id in logs
4. ✅ Write tests first or during
5. ✅ Follow middleware order

---

## 📊 Phase Quick Reference

| Phase | Focus | Key Deliverable | Team |
|-------|-------|-----------------|------|
| 1 | Infrastructure | Core logger working | Backend Core |
| 2 | Middleware | Proper ordering, tracing | Backend Core |
| 3 | SSOT Types | All types in pkg/contracts | Architecture |
| 4 | Services | Services using slog | Services A/B/C |
| 5 | Handlers | Handlers using slog | Handlers |
| 6 | App Core | Single logger instance | Backend Core |
| 7 | WebSocket | Real-time with tracing | Real-time |
| 8 | Frontend | Structured JS logging | Frontend |
| 9 | Testing | 90% coverage | QA |
| 10 | Telemetry | Traces & metrics | Observability |
| 11 | Docs | Everything documented | All |

---

## 🛠️ Useful Commands

### Running Tests
```bash
# With race detection and coverage
go test -race -cover ./...

# Generate coverage report
go test -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Checking Your Work
```bash
# Lint your code
golangci-lint run

# Check for slog usage
grep -r "fmt.Print\|log.Print" --include="*.go" .

# Find custom Logger interfaces  
grep -r "type.*Logger.*interface" --include="*.go" .
```

### Building & Running
```bash
# Build with race detector
go build -race ./cmd/web-licensed

# Run with proper config
./web-licensed -config config.yaml
```

---

## 📞 Getting Help

### Blocked? Check These First:
1. Are dependencies really complete?
2. Did you read the task details fully?
3. Is your branch up to date?
4. Did you check existing examples?

### Still Blocked?
1. Check #dev-help Slack channel
2. Tag your team lead in PR
3. Check pkg/contracts documentation
4. Review CLAUDE.md again

---

## ✅ Definition of Done

A task is DONE when:
- [ ] All acceptance criteria met
- [ ] Tests written and passing (coverage met)
- [ ] Code follows CLAUDE.md standards
- [ ] PR approved by reviewer
- [ ] Documentation updated if needed
- [ ] Task status updated to 🟢
- [ ] No TODOs left in code

---

## 🎯 Quick Wins

### Easy Improvements While You Work:
1. Add context to error messages
2. Include helpful fields in logs
3. Write descriptive test names
4. Add comments for "why" not "what"
5. Improve variable names

### Example Enhanced Error:
```go
// Before
return fmt.Errorf("failed to process")

// After
return fmt.Errorf("failed to process trade: symbol=%s, date=%s: %w", 
    symbol, date, err)
```

---

## 📈 Progress Tracking

### Update Status in Plan:
- 🔴 Not Started → 🟡 In Progress → 🟢 Completed
- ⚫ Blocked? Add comment why

### Daily Standup Template:
```
Yesterday: Completed Task X.Y [what specifically]
Today: Working on Task X.Z [what specifically]  
Blockers: [None | Waiting on Task A.B]
```

---

Remember: **Quality > Speed**. Better to do it right than do it twice!