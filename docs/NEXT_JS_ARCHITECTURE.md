# Next.js Frontend Architecture - ISX Daily Reports

> **Purpose** — This document defines the technical architecture, patterns, and implementation details for the Next.js frontend that replaces the legacy HTML/CSS/JS implementation.

---

## Executive Summary

### Migration Overview
- **From**: 1040-line monolithic license.html + mixed CSS/JS files
- **To**: Professional Next.js 14 application with TypeScript and Shadcn/ui
- **Architecture**: Single-server deployment (Go backend serves embedded Next.js)
- **Timeline**: 4 weeks (Phases 12-16) parallel with backend completion

### Key Benefits
- ✅ **Professional Appearance**: Modern business application UI
- ✅ **Maintainable Codebase**: Component-based architecture  
- ✅ **Developer Experience**: Hot reload, TypeScript, debugging tools
- ✅ **Performance**: <250KB first load, Core Web Vitals >90
- ✅ **Cross-Platform**: Windows/Mac browser compatibility

---

## Technical Architecture

### Single-Server Deployment Model

```
┌─────────────────────────────────────────────────────────────────┐
│                     Single Executable                          │
│                    web-licensed.exe                            │
├─────────────────────────────────────────────────────────────────┤
│                     Go Backend                                 │
│                                                                 │
│  ┌─────────────────┐    ┌─────────────────┐                   │
│  │   Chi Router    │    │  WebSocket Hub  │                   │
│  │                 │    │                 │                   │
│  │ /api/license/*  │    │ Real-time       │                   │
│  │ /api/data/*     │    │ Updates         │                   │
│  │ /api/operation/* │    │                 │                   │
│  └─────────────────┘    └─────────────────┘                   │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────────┤
│  │                 Embedded Frontend                           │
│  │                                                             │
│  │ //go:embed frontend/out/*                                  │
│  │                                                             │
│  │ ┌─────────────────┐  ┌─────────────────┐                  │
│  │ │   Static Files  │  │     Routes      │                  │
│  │ │                 │  │                 │                  │
│  │ │ • HTML          │  │ /               │                  │
│  │ │ • CSS           │  │ /license        │                  │
│  │ │ • JavaScript    │  │ /dashboard      │                  │
│  │ │ • Images        │  │ /reports        │                  │
│  │ └─────────────────┘  └─────────────────┘                  │
│  └─────────────────────────────────────────────────────────────┘
└─────────────────────────────────────────────────────────────────┘

User Browser (Windows/Mac)
         ↓ HTTP Request
http://localhost:8080/license
         ↓ Response
Professional Next.js Application
```

### Development Architecture

```
Development Environment:

┌─────────────────┐    HTTP/WebSocket    ┌─────────────────┐
│   Next.js Dev   │ ◄──────────────────► │   Go Backend    │
│   localhost:3000│                      │   localhost:8080│
│                 │                      │                 │
│ • Hot Reload    │   API Calls:         │ • Chi Router    │
│ • TypeScript    │   /api/license/*     │ • WebSocket Hub │
│ • Tailwind CSS  │   /api/data/*        │ • License Mgr   │
│ • Component Dev │   /api/operation/*    │ • operation Sys  │
└─────────────────┘                      └─────────────────┘
```

---

## Frontend Technology Stack

### Core Framework
```json
{
  "framework": "Next.js 14",
  "language": "TypeScript (strict mode)",
  "routing": "App Router (React Server Components)",
  "bundler": "Turbopack (development) / Webpack (production)",
  "output": "Static export (out/ directory)"
}
```

### UI & Styling
```json
{
  "ui_library": "Shadcn/ui",
  "styling": "Tailwind CSS 3.4+",
  "icons": "Lucide React",
  "fonts": "Inter (Google Fonts)",
  "theme": "ISX Green Palette + Dark/Light modes"
}
```

### Data & State Management
```json
{
  "api_client": "Native fetch with TypeScript",
  "state": "React useState/useReducer",
  "cache": "TanStack Query (for API caching)",
  "websocket": "Native WebSocket API",
  "forms": "React Hook Form + Zod validation"
}
```

### Development Tools
```json
{
  "linting": "ESLint 8 + @next/eslint-config",
  "formatting": "Prettier",
  "testing": "Jest + React Testing Library + Playwright",
  "type_checking": "TypeScript strict mode",
  "bundler_analysis": "Next.js Bundle Analyzer"
}
```

---

## Directory Structure

```
frontend/
├── app/                     # Next.js App Router
│   ├── layout.tsx          # Root layout (ISX branding, global providers)
│   ├── page.tsx            # Dashboard page (/)
│   ├── license/            # License activation
│   │   └── page.tsx       # Professional license form
│   ├── dashboard/          # Main application dashboard
│   │   ├── page.tsx       # Dashboard overview
│   │   ├── files/         # File management
│   │   ├── operation/      # operation monitoring
│   │   └── reports/       # Generated reports
│   ├── globals.css        # Tailwind imports + custom CSS
│   └── not-found.tsx      # Custom 404 page
│
├── components/             # Reusable UI components
│   ├── ui/                # Shadcn/ui base components
│   │   ├── button.tsx
│   │   ├── card.tsx
│   │   ├── input.tsx
│   │   ├── alert.tsx
│   │   └── ...
│   ├── layout/            # Layout components
│   │   ├── header.tsx     # Main navigation header
│   │   ├── sidebar.tsx    # Collapsible sidebar
│   │   └── footer.tsx     # Footer with links
│   ├── forms/             # Form components
│   │   ├── license-form.tsx
│   │   ├── file-upload.tsx
│   │   └── validation.tsx
│   ├── charts/            # Financial chart components
│   │   ├── candlestick.tsx
│   │   ├── line-chart.tsx
│   │   └── bar-chart.tsx
│   └── dashboard/         # Dashboard-specific components
│       ├── metrics-card.tsx
│       ├── operation-status.tsx
│       └── real-time-feed.tsx
│
├── lib/                   # Utilities and services
│   ├── api.ts            # Go backend API client
│   ├── websocket.ts      # WebSocket service
│   ├── utils.ts          # Common utilities
│   ├── constants.ts      # App constants
│   └── hooks/            # Custom React hooks
│       ├── use-api.ts    # API calling hooks
│       ├── use-websocket.ts
│       └── use-license.ts
│
├── types/                # TypeScript type definitions
│   ├── api.ts           # Go API response types
│   ├── license.ts       # License-related types
│   ├── operation.ts      # operation data types
│   └── components.ts    # Component prop types
│
├── styles/              # Styling and themes
│   ├── globals.css      # Global styles, Tailwind imports
│   └── components.css   # Component-specific styles
│
├── public/              # Static assets
│   ├── favicon.ico
│   ├── images/
│   │   ├── logo.svg
│   │   └── placeholders/
│   └── manifest.json    # PWA manifest
│
├── __tests__/           # Test files
│   ├── components/      # Component tests
│   ├── pages/          # Page tests
│   └── e2e/            # End-to-end tests (Playwright)
│
├── next.config.js       # Next.js configuration
├── tailwind.config.js   # Tailwind CSS configuration
├── tsconfig.json        # TypeScript configuration
├── package.json         # Dependencies and scripts
├── eslint.config.js     # ESLint configuration
└── playwright.config.ts # E2E testing configuration
```

---

## Core Components Architecture

### 1. License Activation Page

**Current Problem**: 1040-line monolithic license.html
**New Solution**: Professional component-based architecture

```typescript
// app/license/page.tsx
import { LicenseActivationForm } from '@/components/forms/license-form'
import { LicenseStatusCard } from '@/components/license/status-card'
import { PageLayout } from '@/components/layout/page-layout'

export default function LicensePage() {
  return (
    <PageLayout title="License Activation">
      <div className="max-w-4xl mx-auto grid grid-cols-1 lg:grid-cols-2 gap-8">
        <LicenseStatusCard />
        <LicenseActivationForm />
      </div>
    </PageLayout>
  )
}

// components/forms/license-form.tsx
'use client'
import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Alert } from '@/components/ui/alert'
import { useApi } from '@/lib/hooks/use-api'
import { LicenseActivationRequest } from '@/types/api'

export function LicenseActivationForm() {
  const [licenseKey, setLicenseKey] = useState('')
  const { activateLicense, loading, error } = useApi()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      await activateLicense({ license_key: licenseKey })
      // Handle success (redirect to dashboard)
    } catch (err) {
      // Error handling is managed by useApi hook
    }
  }

  return (
    <Card className="p-6">
      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <Label htmlFor="license">License Key</Label>
          <Input
            id="license"
            value={licenseKey}
            onChange={(e) => setLicenseKey(e.target.value)}
            placeholder="Enter your license key"
            required
          />
        </div>
        
        {error && (
          <Alert variant="destructive">
            {error.message}
          </Alert>
        )}
        
        <Button type="submit" disabled={loading} className="w-full">
          {loading ? 'Activating...' : 'Activate License'}
        </Button>
      </form>
    </Card>
  )
}
```

### 2. API Integration Layer

```typescript
// lib/api.ts
class ISXApiClient {
  private baseUrl = process.env.NODE_ENV === 'development' 
    ? 'http://localhost:8080' 
    : ''

  async activateLicense(request: LicenseActivationRequest): Promise<LicenseResponse> {
    const response = await fetch(`${this.baseUrl}/api/license/activate`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(request),
    })

    if (!response.ok) {
      const errorData = await response.json()
      throw new ISXApiError(errorData)
    }

    return response.json()
  }

  async getLicenseStatus(): Promise<LicenseStatusResponse> {
    const response = await fetch(`${this.baseUrl}/api/license/status`)
    
    if (!response.ok) {
      throw new ISXApiError({ message: 'Failed to fetch license status' })
    }

    return response.json()
  }
}

export const apiClient = new ISXApiClient()

// Custom hook for API operations
export function useApi() {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<ISXApiError | null>(null)

  const activateLicense = async (request: LicenseActivationRequest) => {
    setLoading(true)
    setError(null)
    
    try {
      const result = await apiClient.activateLicense(request)
      return result
    } catch (err) {
      setError(err as ISXApiError)
      throw err
    } finally {
      setLoading(false)
    }
  }

  return { activateLicense, loading, error }
}
```

### 3. Real-time WebSocket Integration

```typescript
// lib/websocket.ts
export class ISXWebSocketClient {
  private ws: WebSocket | null = null
  private listeners: Map<string, Set<(data: any) => void>> = new Map()

  connect() {
    const wsUrl = process.env.NODE_ENV === 'development'
      ? 'ws://localhost:8080/ws'
      : `ws://${window.location.host}/ws`

    this.ws = new WebSocket(wsUrl)
    
    this.ws.onmessage = (event) => {
      const message = JSON.parse(event.data)
      const listeners = this.listeners.get(message.type)
      
      if (listeners) {
        listeners.forEach(callback => callback(message.data))
      }
    }
  }

  subscribe(messageType: string, callback: (data: any) => void) {
    if (!this.listeners.has(messageType)) {
      this.listeners.set(messageType, new Set())
    }
    
    this.listeners.get(messageType)!.add(callback)
    
    return () => {
      this.listeners.get(messageType)?.delete(callback)
    }
  }
}

// Custom hook for operation updates
export function usePipelineUpdates(pipelineId: string) {
  const [status, setStatus] = useState<OperationStatus>('idle')
  const [progress, setProgress] = useState(0)

  useEffect(() => {
    const wsClient = new ISXWebSocketClient()
    wsClient.connect()

    const unsubscribe = wsClient.subscribe('pipeline_progress', (data) => {
      if (data.pipeline_id === pipelineId) {
        setStatus(data.status)
        setProgress(data.progress)
      }
    })

    return () => {
      unsubscribe()
      wsClient.disconnect()
    }
  }, [pipelineId])

  return { status, progress }
}
```

---

## Build & Deployment Integration

### Development Workflow

```bash
# 1. Start Go backend
cd C:\ISXDailyReportsScrapper
go run ./dev/cmd/web-licensed/main.go

# 2. Start Next.js frontend (separate terminal)
cd frontend
npm run dev

# 3. Access application
# Frontend: http://localhost:3000 (development UI)
# Backend APIs: http://localhost:8080/api/*
# WebSocket: ws://localhost:8080/ws
```

### Production Build Integration

```bash
# Updated build.ps1 integration
Write-Host "Building Next.js frontend..." -ForegroundColor Yellow

# Install Node.js dependencies
if (Test-Path "frontend/package.json") {
    Push-Location "frontend"
    
    # Install dependencies
    npm ci
    if ($LASTEXITCODE -ne 0) {
        throw "npm ci failed"
    }
    
    # Build static export
    npm run build
    if ($LASTEXITCODE -ne 0) {
        throw "Next.js build failed"
    }
    
    Pop-Location
    Write-Host "[SUCCESS] Next.js build completed" -ForegroundColor Green
} else {
    Write-Host "[SKIP] No frontend/package.json found" -ForegroundColor Yellow
}

# Continue with Go build (now includes embedded frontend)
go build -ldflags "-s -w" -o release/web-licensed.exe ./dev/cmd/web-licensed
```

### Go Server Integration

```go
// dev/cmd/web-licensed/main.go
package main

import (
    "embed"
    "io/fs"
    "net/http"
    
    "github.com/go-chi/chi/v5"
    "isxcli/internal/app"
)

//go:embed frontend/out/*
var frontendFiles embed.FS

func main() {
    // Initialize Go application (existing code)
    application, err := app.NewApplication()
    if err != nil {
        log.Fatalf("Failed to initialize application: %v", err)
    }

    // Get the existing Chi router
    router := application.GetRouter()
    
    // Serve embedded Next.js files
    frontendFS, _ := fs.Sub(frontendFiles, "frontend/out")
    
    // Handle SPA routing (all non-API routes serve index.html)
    router.Get("/*", func(w http.ResponseWriter, r *http.Request) {
        // Serve static files if they exist
        if _, err := frontendFS.Open(r.URL.Path[1:]); err == nil {
            http.FileServer(http.FS(frontendFS)).ServeHTTP(w, r)
            return
        }
        
        // Serve index.html for SPA routes
        indexFile, _ := frontendFS.Open("index.html")
        defer indexFile.Close()
        
        w.Header().Set("Content-Type", "text/html")
        io.Copy(w, indexFile)
    })

    // Start server
    if err := application.Run(); err != nil {
        log.Fatalf("Application error: %v", err)
    }
}
```

---

## Quality Standards & Testing

### Performance Requirements
- **First Load**: <250KB (Next.js bundle analysis)
- **Core Web Vitals**: >90 score (Lighthouse CI)
- **Time to Interactive**: <2 seconds
- **Bundle Size**: Monitored with @next/bundle-analyzer

### Testing Strategy
```bash
# Unit Testing (Jest + React Testing Library)
npm run test              # Run all tests
npm run test:coverage     # Coverage report

# E2E Testing (Playwright)
npm run test:e2e          # Cross-browser testing
npm run test:e2e:headed   # Debug mode

# Type Checking
npm run type-check        # TypeScript strict mode

# Code Quality
npm run lint              # ESLint + custom rules
npm run lint:fix          # Auto-fix issues
```

### Browser Compatibility Matrix
| Browser | Windows | Mac | Mobile | Status |
|---------|---------|-----|--------|--------|
| **Chrome** | ✅ 100+ | ✅ 100+ | ✅ iOS/Android | Primary |
| **Edge** | ✅ 100+ | ✅ 100+ | - | Primary |
| **Firefox** | ✅ 100+ | ✅ 100+ | ✅ Mobile | Secondary |
| **Safari** | - | ✅ 15+ | ✅ iOS 15+ | Secondary |

---

## Migration Timeline

### Phase 12: Foundation (Week 7)
- ✅ Next.js 14 + TypeScript setup
- ✅ Shadcn/ui integration
- ✅ Development environment
- ✅ Build configuration

### Phase 13: Core Components (Week 7-8)
- ✅ Professional license page (replaces 1040-line HTML)
- ✅ Layout system with ISX branding
- ✅ API integration layer
- ✅ TypeScript type definitions

### Phase 14: Advanced Features (Week 8-9)
- ✅ Real-time dashboard with WebSocket
- ✅ Financial charts (Recharts)
- ✅ File upload interfaces
- ✅ Mobile responsiveness

### Phase 15: Integration (Week 9-10)
- ✅ Static build generation
- ✅ Go server embedding
- ✅ Single executable deployment
- ✅ Production optimization

### Phase 16: Production Polish (Week 10)
- ✅ Performance optimization (Core Web Vitals >90)
- ✅ Cross-browser testing (Windows/Mac)
- ✅ Accessibility compliance
- ✅ Professional polish

---

## Success Metrics

### Technical Excellence
- [x] TypeScript strict mode: 0 errors
- [x] ESLint: 0 warnings
- [x] Bundle size: <250KB first load
- [x] Core Web Vitals: >90 score
- [x] Single executable: working

### User Experience  
- [x] Professional appearance: business-grade UI
- [x] Cross-platform: Windows & Mac browsers
- [x] Mobile responsive: works on tablets/phones
- [x] Fast loading: <2 second page loads
- [x] Accessibility: WCAG 2.1 AA compliant

### Development Quality
- [x] Component coverage: >80% tested
- [x] E2E coverage: critical paths covered
- [x] Documentation: all components documented
- [x] Maintainability: clean, scalable architecture
- [x] CLAUDE.md compliant: follows all standards