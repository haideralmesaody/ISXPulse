# Operations Components

## Purpose
Reusable React components for the operations page, providing modular UI elements for data processing operations management with TypeScript and Shadcn/ui.

## Components

### OperationConfiguration.tsx
Configuration form for starting new operations:

#### Props
```typescript
interface OperationConfigurationProps {
  onStart: (config: OperationConfig) => void
  disabled?: boolean
}
```

#### Features
- **Operation Type Selection**: Dropdown for scraping, processing, export
- **Mode Selection**: Radio buttons for daily/accumulative
- **Date Range Picker**: Calendar interface for date selection
- **Form Validation**: Client-side validation rules
- **Loading State**: Disabled during submission

#### Usage
```tsx
<OperationConfiguration 
  onStart={handleOperationStart}
  disabled={isOperationRunning}
/>
```

### OperationProgress.tsx
Real-time progress tracking for active operations:

#### Props
```typescript
interface OperationProgressProps {
  operationId: string | null
  onCancel?: () => void
}
```

#### Features
- **WebSocket Integration**: Real-time updates
- **Step Progress**: Visual step completion
- **Progress Bar**: Overall completion percentage
- **Log Viewer**: Scrollable log output
- **Cancel Button**: Stop running operations

#### WebSocket Message Handling
```typescript
interface OperationUpdate {
  operationId: string
  status: 'running' | 'completed' | 'failed' | 'cancelled'
  currentStep: number
  totalSteps: number
  progress: number
  logs: string[]
  error?: string
}
```

#### Usage
```tsx
<OperationProgress 
  operationId={currentOperationId}
  onCancel={handleOperationCancel}
/>
```

### OperationHistory.tsx
Historical operations list with filtering and actions:

#### Props
```typescript
interface OperationHistoryProps {
  onViewDetails?: (operationId: string) => void
  onDownloadResult?: (operationId: string) => void
}
```

#### Features
- **Data Table**: Sortable columns
- **Status Filtering**: Filter by success/failed/cancelled
- **Date Filtering**: Date range selection
- **Search**: Operation ID or description search
- **Pagination**: Handle large datasets
- **Actions**: View details, download results

#### Data Structure
```typescript
interface HistoricalOperation {
  id: string
  type: 'scraping' | 'processing' | 'export'
  status: 'completed' | 'failed' | 'cancelled'
  startTime: string
  endTime: string
  duration: number
  resultSize?: number
  error?: string
}
```

#### Usage
```tsx
<OperationHistory 
  onViewDetails={handleViewDetails}
  onDownloadResult={handleDownloadResult}
/>
```

## State Management

### useOperationState Hook
Custom hook for operation state management:
```typescript
export function useOperationState() {
  const [operations, setOperations] = useState<Operation[]>([])
  const [activeOperation, setActiveOperation] = useState<string | null>(null)
  
  const startOperation = async (config: OperationConfig) => {
    // Implementation
  }
  
  const cancelOperation = async (operationId: string) => {
    // Implementation
  }
  
  return {
    operations,
    activeOperation,
    startOperation,
    cancelOperation
  }
}
```

### useWebSocket Hook
Custom hook for WebSocket connections:
```typescript
export function useWebSocket(url: string) {
  const [isConnected, setIsConnected] = useState(false)
  const [lastMessage, setLastMessage] = useState<any>(null)
  
  useEffect(() => {
    const ws = new WebSocket(url)
    
    ws.onopen = () => setIsConnected(true)
    ws.onclose = () => setIsConnected(false)
    ws.onmessage = (event) => setLastMessage(JSON.parse(event.data))
    
    return () => ws.close()
  }, [url])
  
  return { isConnected, lastMessage }
}
```

## Styling

### Component Styles
Using Tailwind CSS with Shadcn/ui:
```tsx
// Consistent spacing and styling
<Card className="p-6 space-y-4">
  <CardHeader>
    <CardTitle>Operation Configuration</CardTitle>
    <CardDescription>Configure and start a new operation</CardDescription>
  </CardHeader>
  <CardContent>
    {/* Form content */}
  </CardContent>
</Card>
```

### Theme Integration
Components respect system theme:
```tsx
// Dark mode support
<div className="bg-background text-foreground">
  <Progress value={progress} className="h-2" />
</div>
```

## Error Handling

### Error Boundaries
Each component wrapped in error boundary:
```tsx
export function OperationConfiguration({ onStart }: Props) {
  return (
    <ErrorBoundary fallback={<ErrorFallback />}>
      {/* Component content */}
    </ErrorBoundary>
  )
}
```

### User Feedback
Toast notifications for user actions:
```tsx
const handleError = (error: Error) => {
  toast({
    title: "Operation Failed",
    description: error.message,
    variant: "destructive"
  })
}
```

## Accessibility

### ARIA Labels
```tsx
<Button 
  onClick={handleStart}
  aria-label="Start data processing operation"
  disabled={!isValid}
>
  Start Operation
</Button>
```

### Focus Management
```tsx
useEffect(() => {
  if (error) {
    errorRef.current?.focus()
  }
}, [error])
```

## Testing

### Component Tests
```typescript
describe('OperationConfiguration', () => {
  it('should validate required fields', () => {
    const onStart = jest.fn()
    const { getByText } = render(
      <OperationConfiguration onStart={onStart} />
    )
    
    fireEvent.click(getByText('Start Operation'))
    expect(onStart).not.toHaveBeenCalled()
    expect(getByText('Please select an operation type')).toBeInTheDocument()
  })
})
```

### Integration Tests
```typescript
describe('OperationProgress WebSocket', () => {
  it('should update progress on message', async () => {
    const { getByText } = render(
      <OperationProgress operationId="test-123" />
    )
    
    // Simulate WebSocket message
    mockWebSocket.simulateMessage({
      operationId: 'test-123',
      progress: 50,
      currentStep: 2,
      totalSteps: 4
    })
    
    await waitFor(() => {
      expect(getByText('50%')).toBeInTheDocument()
      expect(getByText('Step 2 of 4')).toBeInTheDocument()
    })
  })
})
```

## Performance

### Optimization Techniques
1. **Memoization**: React.memo for pure components
2. **Debouncing**: Search and filter inputs
3. **Virtualization**: Large lists in history
4. **Lazy Loading**: Load history on demand
5. **Code Splitting**: Dynamic imports

### Bundle Size
- OperationConfiguration: ~15KB
- OperationProgress: ~20KB (includes WebSocket logic)
- OperationHistory: ~25KB (includes table logic)

## Change Log
- 2025-07-30: Created OperationConfiguration component (Phase 3)
- 2025-07-30: Created OperationProgress component with WebSocket support
- 2025-07-30: Created OperationHistory component with filtering
- 2025-07-30: Added custom hooks for state management
- 2025-07-30: Created README.md documentation for operations components

## Best Practices
1. **Type Safety**: Full TypeScript coverage
2. **Prop Validation**: Runtime prop checking
3. **Error Handling**: Graceful degradation
4. **Accessibility**: WCAG 2.1 AA compliance
5. **Testing**: >90% coverage target