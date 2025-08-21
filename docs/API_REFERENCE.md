# ISX Daily Reports Scrapper - API Documentation

This document provides comprehensive API documentation for the ISX Daily Reports Scrapper system, including REST API endpoints, WebSocket integration, error handling, and TypeScript type definitions.

## Table of Contents

1. [Overview](#overview)
2. [Authentication & License](#authentication--license)
3. [Base URLs & Versioning](#base-urls--versioning)
4. [Common Patterns](#common-patterns)
5. [Error Handling (RFC 7807)](#error-handling-rfc-7807)
6. [Health & System Endpoints](#health--system-endpoints)
7. [License Management API](#license-management-api)
8. [Data API](#data-api)
9. [Operations API](#operations-api)
10. [WebSocket API](#websocket-api)
11. [Analytics API](#analytics-api)
12. [TypeScript Types](#typescript-types)
13. [cURL Examples](#curl-examples)
14. [Client SDKs](#client-sdks)

## Overview

The ISX Daily Reports Scrapper provides a RESTful API with WebSocket support for real-time updates. All data structures follow the Single Source of Truth (SSOT) architecture defined in `pkg/contracts/`.

### Key Features
- RESTful API with JSON responses
- Real-time WebSocket updates for operations
- RFC 7807 compliant error responses
- Hardware-based license validation
- Comprehensive data analytics
- TypeScript type safety

### Technology Stack
- **Backend**: Go with Chi router
- **WebSocket**: Gorilla WebSocket
- **Frontend**: Next.js with embedded static export
- **Validation**: go-playground/validator
- **Observability**: OpenTelemetry with structured logging

## Authentication & License

All API endpoints require a valid license. The system uses hardware-based license activation and validation.

### License Validation
Every API request (except health checks) validates the license status:

```http
HTTP/1.1 401 Unauthorized
Content-Type: application/problem+json

{
  "type": "https://example.com/problems/license-expired",
  "title": "License Expired",
  "status": 401,
  "detail": "Your license has expired. Please renew to continue using the service.",
  "instance": "/api/data/reports"
}
```

### Exempt Endpoints
The following endpoints do not require license validation:
- `/api/health*` - Health check endpoints
- `/api/version` - Version information
- `/api/license/status` - License status check
- `/api/license/activate` - License activation

## Base URLs & Versioning

### Development
```
Base URL: http://localhost:8080
WebSocket: ws://localhost:8080/ws
```

### Production
```
Base URL: https://your-domain.com
WebSocket: wss://your-domain.com/ws
```

### API Versioning
Currently using implicit v1 versioning. All endpoints are under `/api/` prefix.

Future versions will use explicit versioning:
- `/api/v1/` - Current stable API
- `/api/v2/` - Future API version

## Common Patterns

### Request Headers
```http
Content-Type: application/json
Accept: application/json
X-Request-ID: optional-trace-id
```

### Response Headers
```http
Content-Type: application/json
X-Request-ID: trace-id-for-correlation
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
```

### Pagination
Many endpoints support pagination using query parameters:

```json
{
  "page": 1,
  "page_size": 20,
  "sort": "asc",
  "sort_by": "created_at"
}
```

### Date Formats
All dates use ISO 8601 format: `2006-01-02` for dates, `2006-01-02T15:04:05Z07:00` for timestamps.

## Error Handling (RFC 7807)

All API errors follow RFC 7807 Problem Details specification.

### Standard Problem Response
```json
{
  "type": "https://example.com/problems/validation-failed",
  "title": "Validation Failed",
  "status": 400,
  "detail": "The request contains invalid parameters",
  "instance": "/api/data/reports",
  "errors": [
    {
      "field": "date_from",
      "message": "must be a valid date in YYYY-MM-DD format"
    }
  ]
}
```

### Error Types

#### 400 Bad Request
```json
{
  "type": "https://example.com/problems/validation-failed",
  "title": "Validation Failed",
  "status": 400,
  "detail": "Request validation failed",
  "instance": "/api/data/reports"
}
```

#### 401 Unauthorized
```json
{
  "type": "https://example.com/problems/license-expired",
  "title": "License Expired", 
  "status": 401,
  "detail": "Your license has expired",
  "instance": "/api/data/reports"
}
```

#### 404 Not Found
```json
{
  "type": "https://example.com/problems/not-found",
  "title": "Resource Not Found",
  "status": 404,
  "detail": "The requested resource was not found",
  "instance": "/api/data/reports/123"
}
```

#### 500 Internal Server Error
```json
{
  "type": "https://example.com/problems/internal-error",
  "title": "Internal Server Error",
  "status": 500,
  "detail": "An unexpected error occurred",
  "instance": "/api/data/reports"
}
```

## Health & System Endpoints

### GET /api/health
Basic health check endpoint.

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2025-07-31T10:00:00Z",
  "version": "1.0.0",
  "uptime": "24h30m15s"
}
```

### GET /api/health/ready
Readiness check for load balancers.

**Response:**
```json
{
  "ready": true,
  "components": {
    "license": "ready",
    "websocket": "ready",
    "operations": "ready"
  }
}
```

### GET /api/health/live
Liveness check for orchestrators.

**Response:**
```json
{
  "alive": true,
  "last_check": "2025-07-31T10:00:00Z"
}
```

### GET /api/version
Application version information.

**Response:**
```json
{
  "version": "1.0.0",
  "build_time": "2025-07-31T09:00:00Z",
  "git_commit": "abc123def456",
  "go_version": "go1.22.0"
}
```

## License Management API

### GET /api/license/status
Get current license status (no auth required).

**Response:**
```json
{
  "valid": true,
  "status": "active",
  "expires_at": "2025-12-31T23:59:59Z",
  "days_remaining": 153,
  "features": ["scraping", "analysis", "reporting"],
  "tier": "professional"
}
```

### POST /api/license/activate
Activate a license key (no auth required).

**Request:**
```json
{
  "license_key": "ISX1M02LYE1F9QJHR9D7Z",
  "email": "user@example.com"
}
```

**Response:**
```json
{
  "success": true,
  "license_info": {
    "license_key": "ISX1M***************",
    "user_email": "user@example.com",
    "status": "active",
    "activation_date": "2025-07-31T10:00:00Z",
    "expiry_date": "2025-12-31T23:59:59Z",
    "features": ["scraping", "analysis", "reporting"],
    "tier": "professional"
  },
  "message": "License activated successfully"
}
```

### GET /api/license/detailed
Get detailed license information (requires auth).

**Response:**
```json
{
  "license_key": "ISX1M***************",
  "user_email": "user@example.com",
  "status": "active",
  "activation_date": "2025-07-31T10:00:00Z",
  "expiry_date": "2025-12-31T23:59:59Z",
  "last_check_date": "2025-07-31T10:00:00Z",
  "features": ["scraping", "analysis", "reporting"],
  "max_activations": 3,
  "current_activations": 1,
  "tier": "professional",
  "duration": "yearly"
}
```

### GET /api/license/renewal
Get license renewal information.

**Response:**
```json
{
  "eligible": true,
  "renewal_date": "2025-12-01T00:00:00Z",
  "grace_period_end": "2026-01-31T23:59:59Z",
  "discount_percent": 10.0,
  "renewal_url": "https://licensing.example.com/renew?token=xyz"
}
```

## Data API

### GET /api/data/reports
List available reports with pagination and filtering.

**Query Parameters:**
- `page` (int): Page number (default: 1)
- `page_size` (int): Items per page (default: 20, max: 100)
- `date_from` (string): Start date (YYYY-MM-DD)
- `date_to` (string): End date (YYYY-MM-DD)
- `type` (string): Report type (daily, weekly, monthly, etc.)
- `sort` (string): Sort order (asc, desc)
- `sort_by` (string): Sort field

**Response:**
```json
{
  "reports": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "type": "daily",
      "title": "Daily Trading Report - 2025-07-31",
      "status": "completed",
      "format": "csv",
      "file_size": 1024576,
      "generated_at": "2025-07-31T10:00:00Z",
      "date_from": "2025-07-31T00:00:00Z",
      "date_to": "2025-07-31T23:59:59Z",
      "download_url": "/api/data/download/reports/daily_2025-07-31.csv",
      "metadata": {
        "record_count": 150,
        "processing_time": "2.5s",
        "data_sources": ["ISX_API"],
        "version": "1.0"
      }
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 45,
    "total_pages": 3,
    "has_next": true,
    "has_prev": false
  }
}
```

### GET /api/data/tickers
List stock tickers with filtering.

**Query Parameters:**
- `page`, `page_size`: Pagination
- `sector` (string): Filter by sector
- `status` (string): Filter by status (active, suspended, delisted)
- `search` (string): Search term for symbol or company name

**Response:**
```json
{
  "tickers": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440001",
      "symbol": "BMFI",
      "company_name": "Bank of Mesopotamia for Investment",
      "isin_code": "IQBMFI000001",
      "sector": "Banking",
      "status": "active",
      "currency": "IQD",
      "last_price": 1250.0,
      "previous_close": 1200.0,
      "market_cap": 150000000000,
      "listing_date": "2010-01-01T00:00:00Z",
      "last_trade_date": "2025-07-31T10:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 125,
    "total_pages": 7
  }
}
```

### GET /api/data/ticker/{symbol}/chart
Get chart data for a specific ticker.

**Path Parameters:**
- `symbol` (string): Ticker symbol (e.g., "BMFI")

**Query Parameters:**
- `from` (string): Start date (YYYY-MM-DD)
- `to` (string): End date (YYYY-MM-DD)
- `interval` (string): Data interval (1d, 1w, 1m)

**Response:**
```json
{
  "symbol": "BMFI",
  "interval": "1d",
  "data": [
    {
      "date": "2025-07-31T00:00:00Z",
      "open": 1200.0,
      "high": 1280.0,
      "low": 1190.0,
      "close": 1250.0,
      "volume": 15000000,
      "value": 18750000000.0,
      "trades": 45
    }
  ],
  "indicators": {
    "sma_20": 1225.5,
    "sma_50": 1205.2,
    "rsi": 65.4,
    "macd": {
      "macd": 12.5,
      "signal": 8.3,
      "histogram": 4.2
    }
  }
}
```

### GET /api/data/indices
Get market indices data.

**Response:**
```json
{
  "indices": [
    {
      "name": "ISX60",
      "value": 1850.45,
      "change": 12.30,
      "change_percent": 0.67,
      "date": "2025-07-31T10:00:00Z",
      "components": 60,
      "market_cap": 25000000000000.0
    },
    {
      "name": "ISX15",
      "value": 2150.75,
      "change": -5.25,
      "change_percent": -0.24,
      "date": "2025-07-31T10:00:00Z",
      "components": 15,
      "market_cap": 18500000000000.0
    }
  ]
}
```

### GET /api/data/files
List available data files.

**Query Parameters:**
- `type` (string): File type filter (excel, csv, pdf)
- `date_from`, `date_to`: Date range

**Response:**
```json
{
  "files": [
    {
      "name": "ISX_Daily_2025-07-31.xlsx",
      "type": "excel",
      "size": 2048576,
      "date": "2025-07-31T00:00:00Z",
      "path": "/data/downloads/ISX_Daily_2025-07-31.xlsx",
      "download_url": "/api/data/download/excel/ISX_Daily_2025-07-31.xlsx",
      "processed": true,
      "record_count": 150
    }
  ]
}
```

### GET /api/data/market-movers
Get market movers (gainers, losers, most active).

**Query Parameters:**
- `category` (string): gainers, losers, active, volume
- `limit` (int): Number of results (default: 10, max: 50)
- `date` (string): Specific date (YYYY-MM-DD)

**Response:**
```json
{
  "category": "gainers",
  "date": "2025-07-31T00:00:00Z",
  "movers": [
    {
      "symbol": "BMFI",
      "name": "Bank of Mesopotamia for Investment",
      "price": 1250.0,
      "change": 50.0,
      "change_percent": 4.17,
      "volume": 15000000,
      "value": 18750000000.0
    }
  ]
}
```

### GET /api/data/download/{type}/{filename}
Download a specific file.

**Path Parameters:**
- `type` (string): File type (reports, excel, csv)
- `filename` (string): File name

**Response:**
- File download with appropriate Content-Type
- Content-Disposition header for filename

## Operations API

Operations represent multi-step data processing workflows (formerly called "pipelines").

### POST /api/operations/start
Start a new operation.

**Request:**
```json
{
  "type": "scraping",
  "mode": "accumulative",
  "start_date": "2025-07-01",
  "end_date": "2025-07-31",
  "config": {
    "max_retries": 3,
    "parallel": true,
    "max_workers": 4,
    "notify_on_complete": true
  }
}
```

**Response:**
```json
{
  "operation_id": "550e8400-e29b-41d4-a716-446655440002",
  "status": "running",
  "message": "Operation started successfully",
  "started_at": "2025-07-31T10:00:00Z",
  "websocket_url": "ws://localhost:8080/ws"
}
```

### GET /api/operations/{id}/status
Get operation status.

**Path Parameters:**
- `id` (string): Operation ID

**Response:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440002",
  "name": "Data Scraping Operation",
  "type": "scraping",
  "status": "running",
  "created_at": "2025-07-31T10:00:00Z",
  "started_at": "2025-07-31T10:00:00Z",
  "progress": 65.5,
  "current_step": "processing",
  "steps_complete": 2,
  "steps_total": 4,
  "items_processed": 1310,
  "items_total": 2000,
  "estimated_time": "5m30s",
  "steps": [
    {
      "id": "scraping",
      "name": "Data Scraping",
      "type": "scraping",
      "status": "completed",
      "order": 0,
      "started_at": "2025-07-31T10:00:00Z",
      "completed_at": "2025-07-31T10:05:00Z",
      "duration": "5m0s"
    },
    {
      "id": "processing",
      "name": "Data Processing",
      "type": "processing", 
      "status": "running",
      "order": 1,
      "started_at": "2025-07-31T10:05:00Z",
      "state": {
        "progress": 65.5,
        "current_item": "ISX_Daily_2025-07-31.xlsx",
        "items_processed": 131,
        "items_total": 200
      }
    }
  ],
  "metrics": {
    "total_duration": "10m0s",
    "steps_completed": 2,
    "steps_failed": 0,
    "items_processed": 1310,
    "bytes_processed": 52428800,
    "error_rate": 0.0
  }
}
```

### POST /api/operations/{id}/stop
Stop a running operation.

**Path Parameters:**
- `id` (string): Operation ID

**Request:**
```json
{
  "force": false
}
```

**Response:**
```json
{
  "operation_id": "550e8400-e29b-41d4-a716-446655440002",
  "status": "cancelled",
  "message": "Operation stopped successfully",
  "stopped_at": "2025-07-31T10:15:00Z"
}
```

### GET /api/operations
List operations with filtering.

**Query Parameters:**
- `page`, `page_size`: Pagination
- `status` (string): Filter by status
- `type` (string): Filter by type
- `date_from`, `date_to`: Date range

**Response:**
```json
{
  "operations": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440002",
      "name": "Data Scraping Operation",
      "type": "scraping",
      "status": "completed",
      "created_at": "2025-07-31T10:00:00Z",
      "started_at": "2025-07-31T10:00:00Z",
      "completed_at": "2025-07-31T10:30:00Z",
      "metrics": {
        "total_duration": "30m0s",
        "steps_completed": 4,
        "items_processed": 2000
      }
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 15
  }
}
```

### DELETE /api/operations/{id}
Delete an operation (only if completed/failed).

**Path Parameters:**
- `id` (string): Operation ID

**Response:**
```json
{
  "message": "Operation deleted successfully"
}
```

## WebSocket API

Real-time updates are provided via WebSocket connection at `ws://localhost:8080/ws`.

### Connection Establishment

```javascript
const ws = new WebSocket('ws://localhost:8080/ws');

ws.onopen = function(event) {
  console.log('WebSocket connected');
  
  // Subscribe to operation updates
  ws.send(JSON.stringify({
    type: 'subscribe',
    data: {
      channels: ['operations', 'market'],
      filters: {
        operation_type: 'scraping'
      }
    }
  }));
};
```

### Message Format

All WebSocket messages follow this structure:

```json
{
  "id": "msg-550e8400-e29b-41d4-a716-446655440003",
  "type": "operation:progress",
  "channel": "operations",
  "timestamp": "2025-07-31T10:00:00Z",
  "sequence": 1,
  "trace_id": "trace-550e8400-e29b-41d4-a716-446655440004",
  "data": {
    "operation_id": "550e8400-e29b-41d4-a716-446655440002",
    "progress": 65.5,
    "current_step": "processing",
    "message": "Processing file ISX_Daily_2025-07-31.xlsx"
  }
}
```

### Message Types

#### Control Messages

**Connect Response:**
```json
{
  "type": "connect",
  "data": {
    "session_id": "session-123",
    "version": "1.0.0",
    "capabilities": ["subscriptions", "operations", "market"],
    "server_time": "2025-07-31T10:00:00Z"
  }
}
```

**Subscription Confirmation:**
```json
{
  "type": "ack",
  "data": {
    "message_id": "msg-123",
    "success": true
  }
}
```

**Error Message:**
```json
{
  "type": "error",
  "data": {
    "code": "SUBSCRIPTION_FAILED",
    "message": "Invalid channel name",
    "retry": false,
    "fatal": false
  }
}
```

#### Operation Messages

**Operation Started:**
```json
{
  "type": "operation:start",
  "data": {
    "operation_id": "550e8400-e29b-41d4-a716-446655440002",
    "operation_type": "scraping",
    "mode": "accumulative",
    "started_by": "system",
    "started_at": "2025-07-31T10:00:00Z",
    "config": {
      "mode": "accumulative",
      "max_retries": 3
    }
  }
}
```

**Operation Progress:**
```json
{
  "type": "operation:progress",
  "data": {
    "operation_id": "550e8400-e29b-41d4-a716-446655440002",
    "progress": 65.5,
    "current_step": "processing",
    "steps_complete": 2,
    "steps_total": 4,
    "items_processed": 1310,
    "items_total": 2000,
    "estimated_time": "5m30s",
    "message": "Processing file ISX_Daily_2025-07-31.xlsx"
  }
}
```

**Step Progress:**
```json
{
  "type": "step:progress",
  "data": {
    "operation_id": "550e8400-e29b-41d4-a716-446655440002",
    "step_id": "processing",
    "progress": 65.5,
    "items_processed": 131,
    "items_total": 200,
    "current_item": "ISX_Daily_2025-07-31.xlsx",
    "message": "Converting Excel to CSV format"
  }
}
```

**Operation Complete:**
```json
{
  "type": "operation:complete",
  "data": {
    "operation_id": "550e8400-e29b-41d4-a716-446655440002",
    "status": "completed",
    "duration": "30m0s",
    "started_at": "2025-07-31T10:00:00Z",
    "completed_at": "2025-07-31T10:30:00Z",
    "results": {
      "files_processed": 31,
      "records_extracted": 4650,
      "output_files": ["daily_2025-07-31.csv"]
    },
    "metrics": {
      "total_duration": "30m0s",
      "items_processed": 2000,
      "error_rate": 0.0
    },
    "output_files": ["/data/reports/daily_2025-07-31.csv"]
  }
}
```

**Operation Failed:**
```json
{
  "type": "operation:failed",
  "data": {
    "operation_id": "550e8400-e29b-41d4-a716-446655440002",
    "error": "Failed to download file: network timeout",
    "error_code": "NETWORK_TIMEOUT",
    "failed_step": "scraping",
    "failed_at": "2025-07-31T10:15:00Z",
    "can_retry": true,
    "retry_count": 2
  }
}
```

#### Market Data Messages

**Market Update:**
```json
{
  "type": "market:update",
  "data": {
    "market_status": "open",
    "trading_date": "2025-07-31T00:00:00Z",
    "last_update": "2025-07-31T10:00:00Z",
    "summary": {
      "total_market_cap": 25000000000000.0,
      "total_volume": 150000000,
      "total_value": 187500000000.0,
      "active_symbols": 125
    },
    "top_gainers": [
      {
        "symbol": "BMFI",
        "name": "Bank of Mesopotamia",
        "price": 1250.0,
        "change": 50.0,
        "change_percent": 4.17,
        "volume": 15000000
      }
    ]
  }
}
```

**Ticker Update:**
```json
{
  "type": "ticker:update",
  "data": {
    "symbol": "BMFI",
    "price": 1250.0,
    "change": 50.0,
    "change_percent": 4.17,
    "volume": 15000000,
    "high": 1280.0,
    "low": 1190.0,
    "open": 1200.0,
    "prev_close": 1200.0,
    "timestamp": "2025-07-31T10:00:00Z"
  }
}
```

### Subscription Management

**Subscribe to Channels:**
```json
{
  "type": "subscribe",
  "data": {
    "channels": ["operations", "market", "ticker"],
    "filters": {
      "operation_type": "scraping",
      "symbols": ["BMFI", "TASC"]
    },
    "options": {
      "buffer_size": 100,
      "max_frequency": 10,
      "quality": "realtime"
    }
  }
}
```

**Unsubscribe:**
```json
{
  "type": "unsubscribe", 
  "data": {
    "channels": ["ticker"],
    "all": false
  }
}
```

### Error Handling & Reconnection

```javascript
ws.onerror = function(error) {
  console.error('WebSocket error:', error);
};

ws.onclose = function(event) {
  console.log('WebSocket closed:', event.code, event.reason);
  
  // Implement exponential backoff reconnection
  setTimeout(() => {
    reconnectWebSocket();
  }, Math.min(1000 * Math.pow(2, retryCount), 30000));
};

function reconnectWebSocket() {
  // Reconnection logic with exponential backoff
  // Re-subscribe to previous channels
}
```

## Analytics API

### POST /api/analytics/market
Request market analytics.

**Request:**
```json
{
  "date_range": {
    "from": "2025-07-01",
    "to": "2025-07-31"
  },
  "metrics": ["market_cap", "volume", "volatility"],
  "sectors": ["Banking", "Insurance", "Services"],
  "time_frame": "daily"
}
```

**Response:**
```json
{
  "analytics_id": "550e8400-e29b-41d4-a716-446655440005",
  "status": "processing",
  "estimated_time": "2m30s",
  "websocket_url": "ws://localhost:8080/ws"
}
```

### GET /api/analytics/{id}
Get analytics results.

**Response:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440005",
  "type": "market",
  "status": "completed",
  "created_at": "2025-07-31T10:00:00Z",
  "completed_at": "2025-07-31T10:02:30Z",
  "duration": "2m30s",
  "data": {
    "market_statistics": {
      "total_market_cap": 25000000000000.0,
      "total_volume": 4500000000,
      "total_value": 5625000000000.0,
      "active_symbols": 125,
      "advance_decline": {
        "advancing": 75,
        "declining": 40,
        "unchanged": 10,
        "ratio": 1.875
      }
    },
    "sector_performance": [
      {
        "sector": "Banking",
        "performance": 2.34,
        "volume": 2250000000,
        "market_cap": 15000000000000.0
      }
    ]
  }
}
```

## TypeScript Types

The frontend uses TypeScript types generated from Go structs to ensure type safety.

### Core Domain Types

```typescript
// Generated from pkg/contracts/domain/license.go
export interface LicenseInfo {
  license_key: string;
  user_email: string;
  expiry_date: string;
  status: LicenseStatus;
  activation_date: string;
  last_check_date: string;
  features: string[];
  max_activations: number;
  current_activations: number;
  duration: LicenseDuration;
  tier: string;
  metadata?: Record<string, any>;
  created_at: string;
  updated_at: string;
}

export type LicenseStatus = 'active' | 'suspended' | 'expired' | 'revoked';
export type LicenseDuration = 'monthly' | 'quarterly' | 'yearly' | 'lifetime';
```

```typescript
// Generated from pkg/contracts/domain/operations.go
export interface Operation {
  id: string;
  name: string;
  type: OperationType;
  status: OperationStatus;
  config: OperationConfig;
  steps: Step[];
  created_at: string;
  started_at?: string;
  completed_at?: string;
  created_by: string;
  metadata?: Record<string, any>;
  metrics: OperationMetrics;
}

export type OperationType = 'scraping' | 'processing' | 'indexing' | 'analysis';
export type OperationStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled' | 'paused' | 'retrying';
```

```typescript
// Generated from pkg/contracts/domain/ticker.go
export interface Ticker {
  id: string;
  symbol: string;
  company_id: string;
  company_name: string;
  company_name_ar?: string;
  isin_code: string;
  sector: string;
  sub_sector?: string;
  market_cap: number;
  currency: string;
  status: TickerStatus;
  listing_date: string;
  delisting_date?: string;
  last_trade_date: string;
  last_price: number;
  previous_close: number;
  metadata?: Record<string, any>;
  created_at: string;
  updated_at: string;
}

export type TickerStatus = 'active' | 'suspended' | 'delisted' | 'halted';
```

### WebSocket Message Types

```typescript
// Generated from pkg/contracts/events/websocket.go
export interface WebSocketMessage {
  id: string;
  type: MessageType;
  channel: string;
  timestamp: string;
  sequence: number;
  trace_id?: string;
  data?: any;
  metadata?: any;
}

export type MessageType = 
  | 'ping' | 'pong' | 'connect' | 'disconnect'
  | 'subscribe' | 'unsubscribe' | 'error' | 'ack'
  | 'operation:start' | 'operation:progress' | 'operation:complete' | 'operation:failed'
  | 'step:start' | 'step:progress' | 'step:complete' | 'step:failed'
  | 'market:update' | 'ticker:update' | 'trade:update';
```

### API Request/Response Types

```typescript
// Generated from pkg/contracts/api/v1/requests.go
export interface LicenseActivateRequest {
  license_key: string;
  email: string;
}

export interface OperationStartRequest {
  type: OperationType;
  mode: string;
  start_date?: string;
  end_date?: string;
  config?: Record<string, any>;
}

export interface PaginationRequest {
  page: number;
  page_size: number;
  sort?: string;
  sort_by?: string;
}
```

### Error Types (RFC 7807)

```typescript
export interface ProblemDetails {
  type: string;
  title: string;
  status: number;
  detail?: string;
  instance?: string;
  [key: string]: any; // Extensions
}

export interface APIError extends Error {
  problem: ProblemDetails;
}
```

### Type Generation Process

1. **Source**: Go structs in `pkg/contracts/`
2. **Generation**: Using `go2ts` or similar tool
3. **Output**: TypeScript definitions in `frontend/types/`
4. **Build**: Integrated into frontend build process

```bash
# Generate TypeScript types from Go structs
go2ts -input=./pkg/contracts -output=./frontend/types
```

## cURL Examples

### Health Check
```bash
curl -X GET http://localhost:8080/api/health \
  -H "Accept: application/json"
```

### License Activation
```bash
curl -X POST http://localhost:8080/api/license/activate \
  -H "Content-Type: application/json" \
  -H "Accept: application/json" \
  -d '{
    "license_key": "ISX1M02LYE1F9QJHR9D7Z",
    "email": "user@example.com"
  }'
```

### List Reports
```bash
curl -X GET "http://localhost:8080/api/data/reports?page=1&page_size=10&date_from=2025-07-01&date_to=2025-07-31" \
  -H "Accept: application/json"
```

### Get Ticker Data
```bash
curl -X GET "http://localhost:8080/api/data/ticker/BMFI/chart?from=2025-07-01&to=2025-07-31&interval=1d" \
  -H "Accept: application/json"
```

### Start Operation
```bash
curl -X POST http://localhost:8080/api/operations/start \
  -H "Content-Type: application/json" \
  -H "Accept: application/json" \
  -d '{
    "type": "scraping",
    "mode": "accumulative",
    "start_date": "2025-07-01",
    "end_date": "2025-07-31",
    "config": {
      "max_retries": 3,
      "parallel": true,
      "max_workers": 4
    }
  }'
```

### Get Operation Status
```bash
curl -X GET http://localhost:8080/api/operations/550e8400-e29b-41d4-a716-446655440002/status \
  -H "Accept: application/json"
```

### Stop Operation
```bash
curl -X POST http://localhost:8080/api/operations/550e8400-e29b-41d4-a716-446655440002/stop \
  -H "Content-Type: application/json" \
  -H "Accept: application/json" \
  -d '{"force": false}'
```

### Market Analytics
```bash
curl -X POST http://localhost:8080/api/analytics/market \
  -H "Content-Type: application/json" \
  -H "Accept: application/json" \
  -d '{
    "date_range": {
      "from": "2025-07-01",
      "to": "2025-07-31"
    },
    "metrics": ["market_cap", "volume", "volatility"],
    "time_frame": "daily"
  }'
```

### WebSocket Connection Test
```bash
# Using websocat (install with: cargo install websocat)
echo '{"type":"subscribe","data":{"channels":["operations"]}}' | \
  websocat ws://localhost:8080/ws
```

## Client SDKs

### JavaScript/TypeScript SDK

```typescript
import { ISXClient } from './lib/api-client';

const client = new ISXClient({
  baseURL: 'http://localhost:8080',
  websocketURL: 'ws://localhost:8080/ws'
});

// License activation
await client.license.activate({
  license_key: 'ISX1M02LYE1F9QJHR9D7Z',
  email: 'user@example.com'
});

// Start operation with real-time updates
const operation = await client.operations.start({
  type: 'scraping',
  mode: 'accumulative',
  start_date: '2025-07-01',
  end_date: '2025-07-31'
});

// Subscribe to operation updates
client.websocket.subscribe(['operations'], {
  onOperationProgress: (data) => {
    console.log(`Progress: ${data.progress}%`);
  },
  onOperationComplete: (data) => {
    console.log('Operation completed:', data.results);
  }
});

// Get market data
const reports = await client.data.getReports({
  page: 1,
  page_size: 20,
  date_from: '2025-07-01',
  date_to: '2025-07-31'
});
```

### React Hooks

```typescript
import { useOperations, useWebSocket, useMarketData } from './hooks';

function OperationsPage() {
  const { operations, startOperation, isLoading } = useOperations();
  const { subscribe, unsubscribe } = useWebSocket();
  const { tickers, marketMovers } = useMarketData();

  useEffect(() => {
    subscribe(['operations', 'market'], {
      onOperationProgress: (data) => {
        // Update UI with progress
      }
    });

    return () => unsubscribe(['operations', 'market']);
  }, []);

  const handleStartOperation = async () => {
    await startOperation({
      type: 'scraping',
      mode: 'accumulative'
    });
  };

  return (
    <div>
      <button onClick={handleStartOperation} disabled={isLoading}>
        Start Scraping
      </button>
      {/* UI components */}
    </div>
  );
}
```

---

## Notes

- All timestamps are in ISO 8601 format with timezone information
- File sizes are in bytes
- Monetary values are in the respective currency (IQD for Iraqi stocks)
- WebSocket connections support automatic reconnection with exponential backoff
- Rate limiting applies: 100 requests per minute per IP for API endpoints
- WebSocket connections are limited to 10 per IP address
- All API responses include correlation IDs for debugging
- The system supports both English and Arabic text in company names
- Market data is delayed by 15 minutes unless real-time access is licensed

For additional support or questions, contact the development team or refer to the source code documentation in `pkg/contracts/`.