# Operations Page

## Purpose
The operations page provides a comprehensive interface for managing data processing operations in the ISX Daily Reports Scrapper system, featuring real-time progress updates via WebSocket connections.

## Components

### Page Structure
```tsx
export default function OperationsPage() {
  return (
    <div className="container mx-auto p-6">
      <h1>Data Processing Operations</h1>
      <Tabs defaultValue="configure">
        <TabsList>
          <TabsTrigger value="configure">Configure</TabsTrigger>
          <TabsTrigger value="progress">Progress</TabsTrigger>
          <TabsTrigger value="history">History</TabsTrigger>
        </TabsList>
        <TabsContent value="configure">
          <OperationConfiguration onStart={handleStart} />
        </TabsContent>
        <TabsContent value="progress">
          <OperationProgress operationId={currentOperationId} />
        </TabsContent>
        <TabsContent value="history">
          <OperationHistory />
        </TabsContent>
      </Tabs>
    </div>
  )
}
```

## Features

### Operation Configuration
- **Operation Types**: Scraping, Processing, Export
- **Execution Modes**: Daily, Accumulative
- **Date Selection**: Custom date ranges
- **Validation**: Input validation before submission

### Real-Time Progress
- **WebSocket Updates**: Live progress tracking
- **Step Visualization**: Current step highlighting
- **Progress Bar**: Overall completion percentage
- **Log Streaming**: Real-time operation logs

### Operation History
- **Past Operations**: List of completed operations
- **Status Filters**: Success, Failed, Cancelled
- **Result Downloads**: Access processed data
- **Execution Details**: Timing and performance

## State Management

### Local State
```typescript
const [currentOperationId, setCurrentOperationId] = useState<string | null>(null)
const [operationStatus, setOperationStatus] = useState<OperationStatus | null>(null)
const [isConnected, setIsConnected] = useState(false)
```

### WebSocket Connection
```typescript
useEffect(() => {
  const ws = new WebSocket('ws://localhost:8080/ws/operations')
  
  ws.onopen = () => setIsConnected(true)
  ws.onclose = () => setIsConnected(false)
  
  ws.onmessage = (event) => {
    const update = JSON.parse(event.data)
    if (update.operationId === currentOperationId) {
      setOperationStatus(update)
    }
  }
  
  return () => ws.close()
}, [currentOperationId])
```

## API Integration

### Starting Operations
```typescript
const handleStart = async (config: OperationConfig) => {
  try {
    const response = await fetch('/api/operations/start', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(config)
    })
    
    const { operationId } = await response.json()
    setCurrentOperationId(operationId)
    setActiveTab('progress')
  } catch (error) {
    console.error('Failed to start operation:', error)
  }
}
```

### Fetching History
```typescript
const fetchHistory = async () => {
  const response = await fetch('/api/operations/history')
  const operations = await response.json()
  setOperationHistory(operations)
}
```

## UI Components

### OperationConfiguration
- Form inputs for operation parameters
- Date range picker for time selection
- Mode selector (radio buttons)
- Submit button with loading state

### OperationProgress
- Step list with current step indicator
- Progress bar showing completion
- Real-time log viewer
- Cancel operation button

### OperationHistory
- Data table with sorting
- Status badges (success/failed)
- Action buttons (view/download)
- Pagination for large datasets

## Error Handling

### Connection Errors
```typescript
ws.onerror = (error) => {
  console.error('WebSocket error:', error)
  toast({
    title: 'Connection Error',
    description: 'Failed to connect to real-time updates',
    variant: 'destructive'
  })
}
```

### Operation Failures
```typescript
if (update.status === 'failed') {
  toast({
    title: 'Operation Failed',
    description: update.error || 'An error occurred during processing',
    variant: 'destructive'
  })
}
```

## Accessibility

### Keyboard Navigation
- Tab order for form controls
- Enter key form submission
- Escape key for dialogs
- Arrow keys for list navigation

### Screen Reader Support
- ARIA labels for all controls
- Status announcements
- Progress updates
- Error descriptions

## Performance Considerations

### WebSocket Management
- Automatic reconnection logic
- Message buffering
- Connection pooling
- Cleanup on unmount

### Data Optimization
- Paginated history loading
- Virtualized log viewer
- Debounced search inputs
- Memoized calculations

## Testing

### Unit Tests
```typescript
describe('OperationsPage', () => {
  it('should start operation with valid config', async () => {
    const { getByText, getByLabelText } = render(<OperationsPage />)
    
    fireEvent.change(getByLabelText('Operation Type'), { 
      target: { value: 'scraping' } 
    })
    fireEvent.click(getByText('Start Operation'))
    
    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledWith('/api/operations/start', 
        expect.objectContaining({
          method: 'POST',
          body: expect.stringContaining('scraping')
        })
      )
    })
  })
})
```

### Integration Tests
- WebSocket connection tests
- Full operation flow tests
- Error scenario testing
- Performance benchmarks

## Change Log
- 2025-07-30: Created operations page with WebSocket integration (Phase 3)
- 2025-07-30: Added OperationConfiguration component for operation setup
- 2025-07-30: Added OperationProgress component for real-time updates
- 2025-07-30: Added OperationHistory component for past operations
- 2025-07-30: Created README.md documentation for operations page

## Future Enhancements
1. **Batch Operations**: Queue multiple operations
2. **Operation Templates**: Save common configurations
3. **Advanced Filtering**: Complex history queries
4. **Export Options**: Multiple format support
5. **Notifications**: Browser notifications for completion