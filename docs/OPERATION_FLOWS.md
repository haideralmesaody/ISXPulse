# ISX Operation Flow Documentation

This document describes the complete communication flow between frontend and backend for data collection operations.

## Overview

The operation flow consists of three main communication paths:
1. **Frontend → Backend**: Operation configuration and start request
2. **Backend Processing**: Parameter transformation and operation execution
3. **Backend → Frontend**: Real-time status updates via WebSocket

## 1. Frontend Date Configuration

### Default Date Values
When a user clicks "Configure" on any operation type, the frontend receives operation type definitions from the backend with default values:

```javascript
// Backend provides (operations_service.go):
{
  "parameters": [
    {
      "name": "from",
      "type": "date",
      "default": "2025-01-01"  // Fixed default
    },
    {
      "name": "to",
      "type": "date",
      "default": "2025-01-15"  // Dynamic (today's date)
    }
  ]
}
```

### Frontend Initialization (operations/page.tsx)
```javascript
const handleConfigureOperation = (type) => {
  const initialParams = {}
  type.parameters.forEach(param => {
    if (param.default !== undefined) {
      initialParams[param.name] = param.default
    }
  })
  setOperationParams(initialParams)
}
```

## 2. Operation Start Request

### Frontend Request Structure
When the user clicks "Start Operation", the frontend sends:

```javascript
POST /api/operations/start
{
  "mode": "initial",
  "steps": [
    {
      "id": "scraping",
      "type": "scraping",
      "parameters": {
        "mode": "initial",
        "from": "2025-01-01",    // Note: Frontend uses "from"
        "to": "2025-01-15"       // Note: Frontend uses "to"
      }
    },
    // ... other steps
  ]
}
```

### Key Points:
- Frontend sends date parameters as `from` and `to`
- Backend expects `from_date` and `to_date`
- Transformation happens in the service layer

## 3. Backend Processing

### Request Flow Through Layers

#### 1. HTTP Handler (operations_handler.go)
```go
// Extracts parameters from steps
if len(data.Steps) == 1 {
    step := data.Steps[0]
    request.Parameters["step"] = step.ID
    // Merge step parameters
    for k, v := range step.Parameters {
        request.Parameters[k] = v  // Still has "from" and "to"
    }
}
```

#### 2. Service Layer (operations_service.go)
```go
// Parameter transformation happens here
scrapingParams := map[string]interface{}{
    "from_date": getValue(args, "from", ""),  // Maps "from" → "from_date"
    "to_date":   getValue(args, "to", ""),    // Maps "to" → "to_date"
    "mode":      getValue(args, "mode", "initial"),
}
```

#### 3. Operation Manager (manager.go)
```go
// Stores transformed parameters in operation state
if req.Parameters["from_date"] != "" {
    state.SetConfig(ContextKeyFromDate, req.Parameters["from_date"])
}
if req.Parameters["to_date"] != "" {
    state.SetConfig(ContextKeyToDate, req.Parameters["to_date"])
}
```

## 4. WebSocket Status Updates

### WebSocket Event Types
The backend sends these event types for operation updates:

```javascript
// Progress updates
{
  "type": "operation:progress",
  "data": {
    "operation_id": "op-123",
    "step": "scraping",
    "progress": 25,
    "message": "Downloading reports..."
  }
}

// Status changes
{
  "type": "operation:update",
  "data": {
    "operation_id": "op-123",
    "status": "running",
    "step": "processing"
  }
}

// Completion
{
  "type": "operation:complete",
  "data": {
    "operation_id": "op-123",
    "status": "completed",
    "message": "Operation completed successfully"
  }
}

// Errors
{
  "type": "operation:error",
  "data": {
    "operation_id": "op-123",
    "error": "Failed to download report",
    "step": "scraping"
  }
}
```

### Frontend WebSocket Handling (use-websocket.ts)
```javascript
// Subscribe to operation updates
const handleOperationUpdate = (data) => {
  setRunningOperations(prev => prev.map(op => 
    op.id === data.operation_id ? {
      ...op,
      status: data.status,
      progress: data.progress,
      currentStep: data.step,
    } : op
  ))
}
```

## 5. Complete Flow Diagram

```
Frontend                    Backend                     WebSocket
    |                          |                            |
    |-- GET /api/operations/types -->                      |
    |<-- Operation types with defaults --                  |
    |                          |                            |
    | User configures dates    |                            |
    |                          |                            |
    |-- POST /api/operations/start -->                     |
    |   {from: "2025-01-01"}   |                            |
    |                          |                            |
    |                     Transform params                  |
    |                     from → from_date                  |
    |                          |                            |
    |                     Start operation                   |
    |                          |                            |
    |<-- 202 Accepted ---------|                            |
    |   {id: "op-123"}        |                            |
    |                          |                            |
    |                          |-- operation:update ------->|
    |<-------------------- WebSocket Message ---------------|
    |                          |                            |
    |                          |-- operation:progress ----->|
    |<-------------------- WebSocket Message ---------------|
    |                          |                            |
    |                          |-- operation:complete ----->|
    |<-------------------- WebSocket Message ---------------|
```

## 6. Testing the Flow

### Test Files Created:
1. **test-operation-flow.html**: Manual testing interface
   - Tests complete flow from configuration to completion
   - Shows real-time WebSocket messages
   - Displays progress updates

2. **test-api-client.js**: Node.js API testing script
   - Verifies operation types have date defaults
   - Tests parameter transformation
   - Checks operation start/status endpoints

3. **test-websocket.html**: WebSocket testing interface
   - Monitors all WebSocket messages
   - Allows filtering by event type
   - Shows connection statistics

### Running Tests:

1. Start the server:
   ```bash
   cd release
   ./web-licensed.exe
   ```

2. Open test pages in browser:
   - http://localhost:8080/test-operation-flow.html
   - http://localhost:8080/test-websocket.html

3. Run API tests:
   ```bash
   node test-api-client.js
   ```

## 7. Debug Logging

Debug logs have been added at key points:

### Frontend (browser console):
```javascript
Configuring operation: scraping
Operation parameters: [{name: "from", default: "2025-01-01"}, ...]
Initialized params: {from: "2025-01-01", to: "2025-01-15"}
Starting operation with request: {...}
```

### Backend (server logs):
```
INFO: Parameter transformation for scraping
  from_input: 2025-01-01
  from_mapped: 2025-01-01
  to_input: 2025-01-15
  to_mapped: 2025-01-15
  final_params: {from_date: "2025-01-01", to_date: "2025-01-15"}
```

## 8. Common Issues and Solutions

### Issue: Date fields show empty in configuration modal
**Solution**: Backend now provides default values that are automatically populated

### Issue: Invalid date format errors
**Solution**: Both frontend and backend use ISO date format (YYYY-MM-DD)

### Issue: Parameters not reaching scraper executable
**Solution**: Service layer transforms "from"/"to" to "from_date"/"to_date"

### Issue: No real-time updates
**Solution**: Check WebSocket connection status and event subscriptions