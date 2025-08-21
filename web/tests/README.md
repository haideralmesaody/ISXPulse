# Frontend Test Suite

This directory contains the comprehensive test suite for the ISX Daily Reports Scrapper frontend components, following Test-Driven Development (TDD) principles.

## Table of Contents
- [Test Structure](#test-structure)
- [Testing Standards](#testing-standards)
- [Test Patterns](#test-patterns)
- [Running Tests](#running-tests)
- [Writing New Tests](#writing-new-tests)
- [Coverage Requirements](#coverage-requirements)
- [Best Practices](#best-practices)

## Test Structure

```
__tests__/
├── app/                      # Page component tests
│   └── operations/           # Operations page tests
│       └── page.test.tsx
├── components/               # Component tests
│   └── operations/           # Operations components
│       ├── OperationConfiguration.test.tsx
│       ├── OperationProgress.test.tsx
│       └── OperationHistory.test.tsx
├── lib/                      # Library tests
│   ├── api.test.ts          # API client tests
│   └── api-operations.test.ts # Operations API tests
├── setup.ts                 # Test setup and global mocks
└── README.md                # This file
```

## Testing Standards

### Test File Naming
- Test files must be colocated with their source files or in `__tests__` directory
- Use `.test.tsx` for React components
- Use `.test.ts` for TypeScript utilities
- Use `.spec.ts` for E2E tests

### Test Organization
```typescript
describe('ComponentName', () => {
  // Setup
  const defaultProps = { /* ... */ }
  
  beforeEach(() => {
    jest.clearAllMocks()
  })

  describe('Feature/Behavior Group', () => {
    it('specific behavior description', () => {
      // Arrange
      // Act
      // Assert
    })
  })
})
```

## Test Patterns

### 1. Table-Driven Tests
Use table-driven tests for testing multiple scenarios with similar structure:

```typescript
describe('Component Rendering', () => {
  const testCases = [
    {
      name: 'renders in idle state',
      props: { status: 'idle' },
      expectedText: 'Ready to start',
      expectedClass: 'text-gray-500',
    },
    {
      name: 'renders in running state',
      props: { status: 'running' },
      expectedText: 'In progress',
      expectedClass: 'text-blue-500',
    },
  ]

  testCases.forEach(({ name, props, expectedText, expectedClass }) => {
    it(name, () => {
      render(<StatusIndicator {...props} />)
      const element = screen.getByText(expectedText)
      expect(element).toHaveClass(expectedClass)
    })
  })
})
```

### 2. Mock Patterns

#### API Mocking
```typescript
// Mock successful response
;(apiClient.getOperations as jest.Mock).mockResolvedValue([
  { id: 'op-1', name: 'Test Operation' }
])

// Mock error response
;(apiClient.startOperation as jest.Mock).mockRejectedValue(
  new Error('Permission denied')
)

// Mock with implementation
;(apiClient.getOperationHistory as jest.Mock).mockImplementation(
  async (filters) => {
    if (filters.status === 'failed') {
      return { items: failedItems, total: 2 }
    }
    return { items: allItems, total: 10 }
  }
)
```

#### WebSocket Mocking
```typescript
const mockWebSocket = {
  connected: true,
  subscribe: jest.fn(),
  unsubscribe: jest.fn(),
  send: jest.fn(),
}

// Capture event handlers
let updateHandler: (data: any) => void
mockWebSocket.subscribe.mockImplementation((event, handler) => {
  if (event === 'operation:update') {
    updateHandler = handler
  }
})

// Trigger updates in tests
act(() => {
  updateHandler({ operationId: 'op-1', progress: 50 })
})
```

### 3. Async Testing
```typescript
// Wait for async operations
await waitFor(() => {
  expect(screen.getByText('Operation completed')).toBeInTheDocument()
})

// Test loading states
it('shows loading indicator', async () => {
  // Mock never-resolving promise
  ;(apiClient.getOperations as jest.Mock).mockReturnValue(
    new Promise(() => {})
  )
  
  render(<OperationsPage />)
  expect(screen.getByTestId('loading')).toBeInTheDocument()
})
```

### 4. User Interaction Testing
```typescript
// Click events
await userEvent.click(screen.getByRole('button', { name: /Start/i }))

// Form inputs
await userEvent.type(screen.getByLabelText(/Email/i), 'test@example.com')

// Select options
await userEvent.selectOptions(screen.getByLabelText(/Status/i), 'running')

// Keyboard navigation
await userEvent.tab()
await userEvent.keyboard('{Enter}')
```

### 5. Accessibility Testing
```typescript
it('has proper ARIA labels', () => {
  render(<OperationProgress {...props} />)
  
  const progressBar = screen.getByRole('progressbar')
  expect(progressBar).toHaveAttribute('aria-valuenow', '50')
  expect(progressBar).toHaveAttribute('aria-valuemin', '0')
  expect(progressBar).toHaveAttribute('aria-valuemax', '100')
})

it('announces status changes', async () => {
  render(<OperationStatus {...props} />)
  
  // Trigger status change
  act(() => {
    updateStatus('completed')
  })
  
  const announcement = screen.getByRole('status')
  expect(announcement).toHaveTextContent('Operation completed')
})
```

### 6. Error Boundary Testing
```typescript
it('handles component errors gracefully', () => {
  const ThrowError = () => {
    throw new Error('Test error')
  }
  
  render(
    <ErrorBoundary>
      <ThrowError />
    </ErrorBoundary>
  )
  
  expect(screen.getByText(/Something went wrong/i)).toBeInTheDocument()
})
```

## Running Tests

```bash
# Run all tests
npm test

# Run tests in watch mode
npm test -- --watch

# Run tests with coverage
npm test -- --coverage

# Run specific test file
npm test OperationProgress.test.tsx

# Run tests matching pattern
npm test -- --testNamePattern="renders progress"

# Update snapshots
npm test -- -u
```

## Writing New Tests

### Test Template
```typescript
import React from 'react'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ComponentName } from '@/components/ComponentName'
import '@testing-library/jest-dom'

// Mock dependencies
jest.mock('@/lib/api')

describe('ComponentName', () => {
  const defaultProps = {
    // Default props
  }

  beforeEach(() => {
    jest.clearAllMocks()
  })

  describe('Initial Render', () => {
    it('renders with default props', () => {
      render(<ComponentName {...defaultProps} />)
      // Assertions
    })
  })

  describe('User Interactions', () => {
    it('handles user actions', async () => {
      render(<ComponentName {...defaultProps} />)
      // User interactions and assertions
    })
  })

  describe('Error Handling', () => {
    it('displays errors appropriately', () => {
      // Error scenario tests
    })
  })

  describe('Accessibility', () => {
    it('meets WCAG standards', () => {
      // Accessibility tests
    })
  })
})
```

### Testing Checklist
- [ ] Component renders without errors
- [ ] Props are handled correctly
- [ ] User interactions work as expected
- [ ] Loading states are displayed
- [ ] Error states are handled
- [ ] Success states show correct feedback
- [ ] Accessibility requirements are met
- [ ] Edge cases are covered
- [ ] Component unmounts cleanly

## Coverage Requirements

### Target Coverage
- **Operations Components**: ≥90% coverage
- **Critical Business Logic**: ≥90% coverage  
- **Utility Functions**: ≥80% coverage
- **UI Components**: ≥80% coverage

### Coverage Reports
```bash
# Generate coverage report
npm test -- --coverage

# View coverage in browser
npm test -- --coverage --coverageDirectory=coverage

# Coverage thresholds are defined in jest.config.js
```

### What to Test
1. **Component Behavior**: User interactions, state changes
2. **Business Logic**: Calculations, data transformations
3. **Error Paths**: Error handling, edge cases
4. **Integration Points**: API calls, WebSocket messages
5. **Accessibility**: ARIA attributes, keyboard navigation

### What NOT to Test
1. **Third-party Libraries**: Already tested by maintainers
2. **Static Content**: Hard-coded text without logic
3. **Styles**: CSS classes (unless conditional)
4. **Framework Internals**: React's rendering engine

## Best Practices

### 1. Test Behavior, Not Implementation
```typescript
// ❌ Bad: Testing implementation details
expect(component.state.isLoading).toBe(true)

// ✅ Good: Testing user-visible behavior
expect(screen.getByTestId('loading')).toBeInTheDocument()
```

### 2. Use Descriptive Test Names
```typescript
// ❌ Bad: Vague description
it('works correctly', () => {})

// ✅ Good: Clear, specific description
it('displays error message when operation start fails with permission error', () => {})
```

### 3. Keep Tests Independent
```typescript
// ❌ Bad: Tests depend on execution order
let sharedState

it('test 1', () => {
  sharedState = createSomething()
})

it('test 2', () => {
  use(sharedState) // Depends on test 1
})

// ✅ Good: Each test is self-contained
it('test 1', () => {
  const state = createSomething()
  // Use state
})

it('test 2', () => {
  const state = createSomething()
  // Use state
})
```

### 4. Use Data-TestId for Reliable Selection
```typescript
// In component
<div data-testid="operation-status">{status}</div>

// In test
const status = screen.getByTestId('operation-status')
```

### 5. Mock at the Right Level
```typescript
// ❌ Bad: Mocking too deep
jest.mock('node-fetch')

// ✅ Good: Mock at module boundary
jest.mock('@/lib/api')
```

### 6. Test Error Boundaries
```typescript
// Console error suppression for expected errors
const originalError = console.error
beforeAll(() => {
  console.error = jest.fn()
})

afterAll(() => {
  console.error = originalError
})
```

### 7. Performance Considerations
```typescript
// Use React Testing Library's built-in async utilities
await waitFor(() => {
  expect(screen.getByText('Loaded')).toBeInTheDocument()
}, { timeout: 3000 })

// Avoid unnecessary waits
// ❌ Bad
await new Promise(resolve => setTimeout(resolve, 1000))

// ✅ Good
await waitFor(() => expect(mockFn).toHaveBeenCalled())
```

### 8. Snapshot Testing Guidelines
- Use sparingly for complex UI structures
- Review snapshot changes carefully
- Keep snapshots small and focused
- Update with `npm test -- -u` when intentional

## Common Testing Scenarios

### Testing Forms
```typescript
it('submits form with valid data', async () => {
  render(<ConfigurationForm onSubmit={mockSubmit} />)
  
  await userEvent.type(screen.getByLabelText(/Name/i), 'Test Operation')
  await userEvent.selectOptions(screen.getByLabelText(/Type/i), 'report')
  await userEvent.click(screen.getByRole('button', { name: /Save/i }))
  
  expect(mockSubmit).toHaveBeenCalledWith({
    name: 'Test Operation',
    type: 'report',
  })
})
```

### Testing Real-time Updates
```typescript
it('updates in real-time via WebSocket', async () => {
  let wsHandler: (data: any) => void
  mockWebSocket.subscribe.mockImplementation((event, handler) => {
    wsHandler = handler
  })
  
  render(<OperationMonitor />)
  
  act(() => {
    wsHandler({ progress: 75 })
  })
  
  expect(screen.getByText('75%')).toBeInTheDocument()
})
```

### Testing Lists and Tables
```typescript
it('filters and sorts data correctly', async () => {
  render(<OperationHistory items={mockItems} />)
  
  // Filter
  await userEvent.selectOptions(screen.getByLabelText(/Filter/i), 'failed')
  expect(screen.getAllByRole('row')).toHaveLength(2) // Only failed items
  
  // Sort
  await userEvent.click(screen.getByRole('columnheader', { name: /Date/i }))
  const rows = screen.getAllByRole('row')
  expect(rows[0]).toHaveTextContent('2025-01-30') // Most recent first
})
```

## Debugging Tests

### Common Issues and Solutions

1. **Test Timeouts**
   ```typescript
   // Increase timeout for specific test
   it('handles long operation', async () => {
     // test code
   }, 10000) // 10 second timeout
   ```

2. **Act Warnings**
   ```typescript
   // Wrap state updates in act()
   act(() => {
     fireEvent.click(button)
   })
   ```

3. **Async Debugging**
   ```typescript
   // Debug what's in the DOM
   screen.debug()
   
   // Debug specific element
   screen.debug(screen.getByTestId('operation-list'))
   ```

4. **Finding Elements**
   ```typescript
   // Use logRoles to see available roles
   import { logRoles } from '@testing-library/react'
   logRoles(container)
   ```

## Contributing

When adding new tests:
1. Follow the patterns established in this guide
2. Ensure tests are readable and maintainable
3. Add comments for complex test scenarios
4. Update this README if introducing new patterns
5. Run coverage report to verify thresholds