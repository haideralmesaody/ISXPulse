# UI Component Architecture

## Overview

The ISX Pulse frontend uses a simplified, unified component architecture for displaying operation progress and metadata. This document describes the component structure after the 2025-08-09 simplification that reduced UI code by 70%.

## Component Hierarchy

```
OperationProgress (Main container)
├── StepProgress (Unified step renderer)
│   ├── Step header with expand/collapse
│   ├── Progress bar
│   ├── File progress visualization (for scraping)
│   └── MetadataGrid (Generic metadata display)
└── Operation-level UI elements
    ├── Overall progress
    ├── Status indicators
    └── Error messages
```

## Core Components

### 1. StepProgress (`components/operations/StepProgress.tsx`)
**Purpose**: Unified component for rendering all types of operation steps

**Features**:
- Handles all step types (scraping, processing, analysis, etc.)
- Dynamic metadata display based on step type
- Specialized file progress visualization for scraping steps
- Expandable/collapsible detail view
- Status-based styling (running, completed, failed)

**Props**:
```typescript
interface StepProgressProps {
  step: OperationStep
  operationId: string
  isExpanded: boolean
  onToggle: () => void
}
```

### 2. MetadataGrid (`components/operations/MetadataGrid.tsx`)
**Purpose**: Reusable grid component for displaying any key-value metadata

**Features**:
- Configurable column layout (1-4 columns)
- Smart formatting based on key patterns
- Priority key ordering
- Custom formatters and labels
- Automatic value formatting (dates, percentages, bytes, etc.)

**Props**:
```typescript
interface MetadataGridProps {
  metadata: Record<string, any>
  columns?: 1 | 2 | 3 | 4
  maxItems?: number
  priorityKeys?: string[]
  hiddenKeys?: string[]
  formatters?: Record<string, (value: any) => string>
  labels?: Record<string, string>
}
```

### 3. OperationProgress (`components/operations/OperationProgress.tsx`)
**Purpose**: Main container for operation monitoring

**Features**:
- Real-time WebSocket updates
- Overall operation status and progress
- Step management and rendering
- Error handling and display
- Performance metrics tracking

## Data Flow

### WebSocket Updates (Unidirectional)
```
Backend (StatusBroadcaster)
    ↓
WebSocket (operation:snapshot)
    ↓
Frontend (useWebSocket hook)
    ↓
OperationProgress (state update)
    ↓
StepProgress (re-render)
```

### User Commands (REST API)
```
User Action → HTTP REST API → Backend
```

## Key Design Decisions

### 1. Component Unification
- **Before**: Specialized components for each step type (ScrapingProgress, ProcessingProgress, etc.)
- **After**: Single StepProgress component with dynamic rendering based on metadata
- **Benefit**: 70% code reduction, easier maintenance, consistent UX

### 2. Metadata-Driven UI
- Step appearance determined by metadata content, not hardcoded logic
- Generic MetadataGrid handles any metadata structure
- Automatic formatting based on key patterns

### 3. Progressive Disclosure
- Collapsed view shows essential information
- Expanded view reveals detailed metadata and metrics
- Automatic expansion for running steps

## Metadata Conventions

### Common Metadata Keys

**File Operations**:
- `total_expected`: Total number of files to process
- `files_downloaded`: Number of downloaded files
- `files_existing`: Number of files already present
- `current_file`: Currently processing file number
- `current_page`: Current page being processed

**Processing Metrics**:
- `records_processed`: Number of records processed
- `processing_rate`: Records per second
- `error_count`: Number of errors encountered
- `warning_count`: Number of warnings

**Timing**:
- `started_at`: Step start time
- `completed_at`: Step completion time
- `duration`: Step duration in milliseconds
- `estimated_completion`: Estimated completion time

## Styling Guidelines

### Status Colors
- **idle**: `text-muted-foreground`
- **running**: `text-blue-600` with pulse animation
- **completed**: `text-green-600`
- **failed**: `text-red-600`
- **cancelled**: `text-amber-600`

### Progress Visualization
- Green segment: Downloaded/processed items
- Amber segment: Existing/skipped items
- Gray background: Remaining items

## Performance Optimizations

1. **Memoization**: Heavy computations cached with `useMemo`
2. **Selective Updates**: Only re-render changed steps
3. **Batched WebSocket Updates**: Process multiple updates in single render
4. **Virtual Scrolling**: For large step lists (future enhancement)

## Testing Strategy

### Unit Tests
- Component rendering with various metadata shapes
- Progress calculations
- Status transitions
- Error handling

### Integration Tests
- WebSocket update handling
- User interaction flows
- Data flow from backend to UI

## Migration from Old Architecture

### Removed Components
- `ScrapingProgress.tsx` (434 lines)
- `ScrapingProgress.types.ts` (156 lines)
- `ScrapingProgress.utils.ts` (134 lines)
- `ScrapingProgress.test.tsx` (45 lines)
- `ScrapingProgress.utils.test.ts` (31 lines)

### Total Reduction
- **Files**: 5 → 2 (60% reduction)
- **Lines**: ~1200 → ~450 (70% reduction)
- **Complexity**: Specialized logic → Generic metadata-driven

## Future Enhancements

1. **Virtual Scrolling**: For operations with 100+ steps
2. **Custom Step Renderers**: Plugin system for special step types
3. **Metric Dashboards**: Aggregate metrics across operations
4. **Step Filtering**: Show/hide steps by status or type
5. **Export Functionality**: Save operation logs and metrics

## Best Practices

1. **Add Metadata, Not Components**: Extend functionality through metadata, not new components
2. **Use MetadataGrid**: For any key-value display needs
3. **Follow Naming Conventions**: Use snake_case for metadata keys
4. **Document Special Keys**: Add new metadata keys to this document
5. **Test with Production Data**: Ensure components handle real-world metadata shapes