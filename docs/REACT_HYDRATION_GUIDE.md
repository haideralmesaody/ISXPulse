# React Hydration Guide for ISX Pulse

## Overview

This guide explains how to handle React hydration errors in the ISX Pulse application. React hydration errors occur when the server-rendered HTML doesn't match what React expects to render on the client side.

## Common Hydration Error Types

### Error #418: Text content does not match
This error occurs when text content differs between server and client rendering.

### Error #423: Hydration failed because initial UI does not match
This error happens when the component tree structure differs between server and client.

## Root Causes in Our Application

1. **Date/Time Operations**
   - `new Date()` returns different values on server vs client
   - `Date.now()` produces different timestamps
   - Time formatting functions like `toLocaleString()`

2. **Dynamic Content**
   - WebSocket updates arriving before hydration completes
   - Asynchronous data fetching
   - Browser-only APIs used during initial render

3. **Conditional Rendering**
   - Using `window` or `document` checks
   - Browser feature detection
   - User preferences from localStorage

## Solution Pattern

### 1. Hydration State Management

```typescript
const [isHydrated, setIsHydrated] = useState(false)

useEffect(() => {
  setIsHydrated(true)
}, [])

// Pre-hydration loading state
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

### 2. Guard Date Operations

```typescript
// BAD: Causes hydration mismatch
const elapsed = Date.now() - startTime

// GOOD: Delay until after hydration
useEffect(() => {
  if (!isHydrated) return
  
  const calculateElapsed = () => {
    const elapsed = Date.now() - startTime
    setElapsedTime(elapsed)
  }
  
  // Use setTimeout to delay first calculation
  const timeout = setTimeout(calculateElapsed, 0)
  
  return () => clearTimeout(timeout)
}, [isHydrated, startTime])
```

### 3. WebSocket and Async Operations

```typescript
// Delay WebSocket connections until after hydration
useEffect(() => {
  if (!isHydrated) return
  
  const unsubscribe = subscribe('event', handler)
  return () => unsubscribe()
}, [isHydrated])
```

### 4. Suppress Dynamic Content Warnings

```typescript
// For content that must be dynamic
<span suppressHydrationWarning>
  {new Date().toLocaleString()}
</span>
```

## Implementation Checklist

### For Page Components

- [ ] Add `isHydrated` state
- [ ] Set `isHydrated` to true in useEffect
- [ ] Return loading state when not hydrated
- [ ] Guard all date operations with hydration check
- [ ] Delay WebSocket subscriptions until after hydration
- [ ] Add `isHydrated` to useCallback dependencies

### For Date/Time Display

- [ ] Move date calculations into useEffect
- [ ] Use state for calculated values
- [ ] Add setTimeout(fn, 0) for first calculation
- [ ] Use suppressHydrationWarning for display

### For Data Fetching

- [ ] Delay initial fetch until after hydration
- [ ] Guard polling/refresh logic with hydration check
- [ ] Handle loading states properly

## Component Examples

### Operations Page (Fixed)

```typescript
// State for hydration
const [isHydrated, setIsHydrated] = useState(false)

// Set hydration state
useEffect(() => {
  setIsHydrated(true)
}, [])

// Guard async operations
useEffect(() => {
  if (!isHydrated) return
  
  // Stale operation cleanup
  const staleThreshold = new Date()
  staleThreshold.setHours(staleThreshold.getHours() - 24)
  
  const staleOps = operations.filter(op => 
    op.status === 'running' && 
    new Date(op.startedAt) < staleThreshold
  )
  
  staleOps.forEach(op => {
    // Cleanup logic
  })
}, [isHydrated, operations])

// Pre-hydration loading
if (!isHydrated) {
  return <LoadingState />
}
```

### Time Display Component

```typescript
const [elapsedTime, setElapsedTime] = useState<string | null>(null)

useEffect(() => {
  const calculateElapsed = () => {
    const elapsed = Date.now() - startTime
    const minutes = Math.floor(elapsed / 60000)
    const seconds = Math.floor((elapsed % 60000) / 1000)
    setElapsedTime(`${minutes}m ${seconds}s`)
  }
  
  // Delay first calculation
  const timeout = setTimeout(() => {
    calculateElapsed()
    const interval = setInterval(calculateElapsed, 1000)
    return () => clearInterval(interval)
  }, 0)
  
  return () => clearTimeout(timeout)
}, [startTime])

// Render with suppressHydrationWarning
return (
  <span suppressHydrationWarning>
    {elapsedTime || 'Calculating...'}
  </span>
)
```

## Testing for Hydration Issues

1. **Build and run production build**
   ```bash
   npm run build
   npm start
   ```

2. **Check browser console for errors**
   - Look for "Text content did not match"
   - Check for "Hydration failed" messages

3. **Common problem areas to test**
   - Pages with timestamps
   - Real-time data displays
   - Conditional content based on time
   - WebSocket-connected components

## Prevention Guidelines

1. **Always use hydration guards** for:
   - Date/time operations
   - Browser API access
   - Async data fetching
   - WebSocket connections

2. **Avoid during initial render**:
   - `new Date()`
   - `Date.now()`
   - `Math.random()`
   - `window` or `document` access
   - localStorage/sessionStorage

3. **Use these patterns**:
   - Hydration state management
   - useEffect for side effects
   - State for dynamic values
   - suppressHydrationWarning sparingly

## Debugging Tips

1. **Enable React DevTools Profiler** to see hydration timing
2. **Add console logs** to track render cycles
3. **Use React.StrictMode** in development
4. **Test with slow network** to expose timing issues

## Class Components and Hydration

### Error Boundaries
Class components like ErrorBoundary need special handling:

```typescript
// Good: Check for client-side before using window
if (typeof window !== 'undefined' && window.location) {
  window.location.reload()
}

// Good: Use lifecycle methods for client-only operations
componentDidMount() {
  // Safe to use window/document here
  this.setState({ mounted: true })
}
```

### Environment Variables

Environment variables can cause hydration mismatches if they differ between build and runtime:

```typescript
// BAD: Direct env access
const buildId = process.env.NEXT_PUBLIC_BUILD_ID

// GOOD: Use state with consistent initial value
const [buildId] = useState(process.env.NEXT_PUBLIC_BUILD_ID || 'unknown')
```

## Utility Hooks

### useHydration Hook

The project provides a reusable hydration hook:

```typescript
import { useHydration } from '@/lib/hooks'

function MyComponent() {
  const isHydrated = useHydration()
  
  if (!isHydrated) {
    return <LoadingState />
  }
  
  // Client-only content
  return <div>{new Date().toLocaleString()}</div>
}
```

### useClientValue Hook

For values that differ between server and client:

```typescript
import { useClientValue } from '@/lib/hooks'

function Footer() {
  const currentYear = useClientValue(
    () => new Date().getFullYear(),
    2025 // SSR fallback
  )
  
  return <span>© {currentYear} ISX Pulse</span>
}
```

### withHydration HOC

For wrapping entire components:

```typescript
import { withHydration } from '@/lib/hooks'

const ClientOnlyChart = withHydration(
  ChartComponent,
  <ChartSkeleton /> // Loading state
)
```

## Real-World Examples from ISX Pulse

### License Page Pattern
```typescript
// Good: Comprehensive hydration handling
const [isHydrated, setIsHydrated] = useState(false)

useEffect(() => {
  setIsHydrated(true)
}, [])

// Guard dynamic content
{isHydrated && licenseState !== 'none' && (
  <Alert>
    <span suppressHydrationWarning>
      {getStatusMessage().message}
    </span>
  </Alert>
)}
```

### Version Info Pattern
```typescript
// Good: Delay API calls until after hydration
useEffect(() => {
  if (!isHydrated) return
  
  fetchVersion()
}, [isHydrated])
```

### Footer Pattern
```typescript
// Good: suppressHydrationWarning for dynamic dates
<span suppressHydrationWarning>
  © {currentYear} ISX Pulse
</span>
```

## WebSocket and Toast Patterns

### WebSocket Connections

WebSocket connections are inherently client-side but can cause issues if not handled properly:

```typescript
// Good: WebSocket client already checks for SSR
if (typeof window === 'undefined') {
  return // Skip on server
}

// Good: Connection in useEffect
useEffect(() => {
  const client = getWebSocketClient()
  client.connect()
  return () => client.disconnect()
}, [])
```

### Toast Notifications

While toast calls in event handlers are safe, guard them for future streaming SSR:

```typescript
// Good: Guard toast calls
const handleSuccess = () => {
  if (isHydrated) {
    toast({
      title: "Success",
      description: "Operation completed"
    })
  }
}

// Also safe: In callbacks/promises
.then(() => {
  if (isHydrated) {
    toast({ title: "Done" })
  }
})
```

### Client-Side Redirects

Redirects in useEffect are safe but consider alternatives:

```typescript
// Current: Client-side redirect
useEffect(() => {
  if (condition) {
    router.push('/dashboard')
  }
}, [condition])

// Better: Server-side redirect in middleware
// middleware.ts
export function middleware(request: NextRequest) {
  const hasLicense = checkLicense(request)
  if (!hasLicense) {
    return NextResponse.redirect(new URL('/license', request.url))
  }
}
```

## Advanced Patterns

### Defer Hooks Until Hydration

For hooks with side effects, defer execution:

```typescript
// Instead of:
const { data } = useWebSocket()

// Do this:
const socketData = isHydrated ? useWebSocket() : { data: null }
```

### SSR Data Prefetching

Eliminate loading states with server-side data:

```typescript
// app/license/page.tsx with SSR
export default async function LicensePage() {
  // Fetch on server
  const initialLicense = await getLicenseStatus()
  
  return <LicenseClient initialData={initialLicense} />
}

// Client component uses initial data
function LicenseClient({ initialData }: { initialData: LicenseData }) {
  const [license, setLicense] = useState(initialData)
  // No loading state needed!
}
```

### Progressive Enhancement

Show static content immediately, enhance after hydration:

```typescript
function StatusCard() {
  const isHydrated = useHydration()
  
  return (
    <Card>
      <CardHeader>
        <CardTitle>License Status</CardTitle>
      </CardHeader>
      <CardContent>
        {!isHydrated ? (
          // Static placeholder with animation
          <div className="space-y-4">
            <Skeleton className="h-4 w-full" />
            <Progress value={33} />
            <p className="text-sm">Checking status...</p>
          </div>
        ) : (
          // Full interactive content
          <DynamicLicenseInfo />
        )}
      </CardContent>
    </Card>
  )
}
```

## Best Practices Summary

1. **Always use hydration guards** for:
   - Dynamic content that changes between server/client
   - Toast notifications (future-proofing)
   - Any browser-only APIs
   - Hooks with side effects

2. **Already safe patterns**:
   - WebSocket connections (check window in client)
   - Event handlers (onClick, onSubmit)
   - useEffect hooks

3. **Consider server-side alternatives**:
   - Redirects in middleware
   - Data fetching in server components
   - Initial state from cookies/headers
   - SSR prefetching for instant UI

## References

- [Next.js Hydration Documentation](https://nextjs.org/docs/messages/react-hydration-error)
- [React Hydration Errors](https://react.dev/errors/418)
- [ISX Pulse Architecture](./README.md)