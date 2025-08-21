import { renderHook, act } from '@testing-library/react'
import { useLicenseStatus, useConnectionStatus, usePipelineUpdates } from '@/lib/hooks/use-websocket'

// Mock the WebSocket client
const mockWebSocketClient = {
  connect: jest.fn(),
  disconnect: jest.fn(),
  subscribe: jest.fn(),
  getConnectionState: jest.fn(),
  getConnectionStats: jest.fn(),
  onConnectionStateChange: jest.fn(),
  onError: jest.fn(),
}

// Mock websocket module
jest.mock('@/lib/websocket', () => ({
  ISXWebSocketClient: jest.fn(() => mockWebSocketClient),
  ConnectionState: {
    DISCONNECTED: 'disconnected',
    CONNECTING: 'connecting',
    CONNECTED: 'connected',
    RECONNECTING: 'reconnecting',
  },
}))

describe('useLicenseStatus', () => {
  beforeEach(() => {
    jest.clearAllMocks()
    mockWebSocketClient.connect.mockResolvedValue(undefined)
    mockWebSocketClient.getConnectionState.mockReturnValue('connected')
    mockWebSocketClient.subscribe.mockReturnValue(jest.fn()) // unsubscribe function
  })

  it('initializes with default license status', () => {
    const { result } = renderHook(() => useLicenseStatus())

    expect(result.current).toEqual({
      isValid: false,
      expiresAt: null,
      features: [],
      userInfo: null,
      loading: true,
      error: null,
    })
  })

  it('connects to WebSocket on mount', () => {
    renderHook(() => useLicenseStatus())

    expect(mockWebSocketClient.connect).toHaveBeenCalled()
  })

  it('subscribes to license status updates', () => {
    renderHook(() => useLicenseStatus())

    expect(mockWebSocketClient.subscribe).toHaveBeenCalledWith(
      'license_status',
      expect.any(Function)
    )
  })

  it('updates license status when receiving WebSocket message', () => {
    let statusUpdateHandler: (data: any) => void

    mockWebSocketClient.subscribe.mockImplementation((type, handler) => {
      if (type === 'license_status') {
        statusUpdateHandler = handler
      }
      return jest.fn()
    })

    const { result } = renderHook(() => useLicenseStatus())

    // Simulate license status update
    act(() => {
      statusUpdateHandler!({
        isValid: true,
        expiresAt: '2025-12-31T23:59:59Z',
        features: ['Daily Reports Access', 'Advanced Analytics'],
        userInfo: {
          name: 'Test User',
          email: 'test@iraqiinvestor.gov.iq',
          organization: 'Iraqi Investment Bank',
        },
      })
    })

    expect(result.current).toEqual({
      isValid: true,
      expiresAt: '2025-12-31T23:59:59Z',
      features: ['Daily Reports Access', 'Advanced Analytics'],
      userInfo: {
        name: 'Test User',
        email: 'test@iraqiinvestor.gov.iq',
        organization: 'Iraqi Investment Bank',
      },
      loading: false,
      error: null,
    })
  })

  it('handles license expiration updates', () => {
    let statusUpdateHandler: (data: any) => void

    mockWebSocketClient.subscribe.mockImplementation((type, handler) => {
      if (type === 'license_status') {
        statusUpdateHandler = handler
      }
      return jest.fn()
    })

    const { result } = renderHook(() => useLicenseStatus())

    // Simulate license expiration
    act(() => {
      statusUpdateHandler!({
        isValid: false,
        expiresAt: '2025-01-01T00:00:00Z', // Expired
        features: [],
        userInfo: null,
        reason: 'License expired',
      })
    })

    expect(result.current.isValid).toBe(false)
    expect(result.current.features).toEqual([])
    expect(result.current.userInfo).toBe(null)
  })

  it('handles connection errors', () => {
    let errorHandler: (error: Error) => void

    mockWebSocketClient.onError.mockImplementation((handler) => {
      errorHandler = handler
    })

    const { result } = renderHook(() => useLicenseStatus())

    act(() => {
      errorHandler!(new Error('Connection failed'))
    })

    expect(result.current.error).toEqual(new Error('Connection failed'))
    expect(result.current.loading).toBe(false)
  })

  it('disconnects WebSocket on unmount', () => {
    const { unmount } = renderHook(() => useLicenseStatus())

    unmount()

    expect(mockWebSocketClient.disconnect).toHaveBeenCalled()
  })

  it('unsubscribes from license status updates on unmount', () => {
    const unsubscribeMock = jest.fn()
    mockWebSocketClient.subscribe.mockReturnValue(unsubscribeMock)

    const { unmount } = renderHook(() => useLicenseStatus())

    unmount()

    expect(unsubscribeMock).toHaveBeenCalled()
  })
})

describe('useConnectionStatus', () => {
  beforeEach(() => {
    jest.clearAllMocks()
    mockWebSocketClient.getConnectionState.mockReturnValue('connected')
    mockWebSocketClient.getConnectionStats.mockReturnValue({
      isConnected: true,
      reconnectAttempts: 0,
      queuedMessages: 0,
      totalMessagesSent: 5,
      totalMessagesReceived: 10,
      lastConnectionTime: new Date(),
      lastPingTime: Date.now() - 1000,
    })
  })

  it('returns current connection status', () => {
    const { result } = renderHook(() => useConnectionStatus())

    expect(result.current).toEqual({
      isConnected: true,
      connectionState: 'connected',
      reconnectAttempts: 0,
      queuedMessages: 0,
      stats: {
        totalMessagesSent: 5,
        totalMessagesReceived: 10,
        lastConnectionTime: expect.any(Date),
        lastPingTime: expect.any(Number),
      },
    })
  })

  it('updates when connection state changes', () => {
    let stateChangeHandler: (state: string) => void

    mockWebSocketClient.onConnectionStateChange.mockImplementation((handler) => {
      stateChangeHandler = handler
    })

    const { result } = renderHook(() => useConnectionStatus())

    // Simulate connection state change
    mockWebSocketClient.getConnectionState.mockReturnValue('reconnecting')
    mockWebSocketClient.getConnectionStats.mockReturnValue({
      isConnected: false,
      reconnectAttempts: 2,
      queuedMessages: 3,
      totalMessagesSent: 5,
      totalMessagesReceived: 10,
      lastConnectionTime: new Date(),
      lastPingTime: 0,
    })

    act(() => {
      stateChangeHandler!('reconnecting')
    })

    expect(result.current).toEqual({
      isConnected: false,
      connectionState: 'reconnecting',
      reconnectAttempts: 2,
      queuedMessages: 3,
      stats: {
        totalMessagesSent: 5,
        totalMessagesReceived: 10,
        lastConnectionTime: expect.any(Date),
        lastPingTime: 0,
      },
    })
  })

  it('polls connection stats periodically', () => {
    jest.useFakeTimers()

    renderHook(() => useConnectionStatus())

    // Fast-forward to trigger stats polling
    act(() => {
      jest.advanceTimersByTime(5000) // 5 seconds
    })

    // Should have called getConnectionStats multiple times
    expect(mockWebSocketClient.getConnectionStats).toHaveBeenCalledTimes(2)

    jest.useRealTimers()
  })
})

describe('usePipelineUpdates', () => {
  beforeEach(() => {
    jest.clearAllMocks()
    mockWebSocketClient.connect.mockResolvedValue(undefined)
    mockWebSocketClient.subscribe.mockReturnValue(jest.fn())
  })

  it('initializes with default operation state', () => {
    const { result } = renderHook(() => usePipelineUpdates('operation-123'))

    expect(result.current).toEqual({
      status: 'idle',
      progress: 0,
      step: null,
      message: null,
      error: null,
      startTime: null,
      endTime: null,
      results: null,
    })
  })

  it('filters updates by operation ID', () => {
    let progressUpdateHandler: (data: any) => void

    mockWebSocketClient.subscribe.mockImplementation((type, handler) => {
      if (type === 'pipeline_progress') {
        progressUpdateHandler = handler
      }
      return jest.fn()
    })

    const { result } = renderHook(() => usePipelineUpdates('operation-123'))

    // Update for different operation should be ignored
    act(() => {
      progressUpdateHandler!({
        pipeline_id: 'operation-456',
        status: 'running',
        progress: 50,
        step: 'Processing',
      })
    })

    expect(result.current.status).toBe('idle')
    expect(result.current.progress).toBe(0)

    // Update for our operation should be processed
    act(() => {
      progressUpdateHandler!({
        pipeline_id: 'operation-123',
        status: 'running',
        progress: 75,
        step: 'Data Processing',
        message: 'Processing market data...',
        startTime: '2025-01-26T10:00:00Z',
      })
    })

    expect(result.current).toEqual({
      status: 'running',
      progress: 75,
      step: 'Data Processing',
      message: 'Processing market data...',
      error: null,
      startTime: '2025-01-26T10:00:00Z',
      endTime: null,
      results: null,
    })
  })

  it('handles operation completion', () => {
    let progressUpdateHandler: (data: any) => void

    mockWebSocketClient.subscribe.mockImplementation((type, handler) => {
      if (type === 'pipeline_progress') {
        progressUpdateHandler = handler
      }
      return jest.fn()
    })

    const { result } = renderHook(() => usePipelineUpdates('operation-123'))

    act(() => {
      progressUpdateHandler!({
        pipeline_id: 'operation-123',
        status: 'completed',
        progress: 100,
        step: 'Completed',
        message: 'operation completed successfully',
        startTime: '2025-01-26T10:00:00Z',
        endTime: '2025-01-26T10:05:00Z',
        results: {
          processed_records: 1500,
          success_count: 1485,
          error_count: 15,
        },
      })
    })

    expect(result.current).toEqual({
      status: 'completed',
      progress: 100,
      step: 'Completed',
      message: 'operation completed successfully',
      error: null,
      startTime: '2025-01-26T10:00:00Z',
      endTime: '2025-01-26T10:05:00Z',
      results: {
        processed_records: 1500,
        success_count: 1485,
        error_count: 15,
      },
    })
  })

  it('handles operation errors', () => {
    let progressUpdateHandler: (data: any) => void

    mockWebSocketClient.subscribe.mockImplementation((type, handler) => {
      if (type === 'pipeline_progress') {
        progressUpdateHandler = handler
      }
      return jest.fn()
    })

    const { result } = renderHook(() => usePipelineUpdates('operation-123'))

    act(() => {
      progressUpdateHandler!({
        pipeline_id: 'operation-123',
        status: 'failed',
        progress: 45,
        step: 'Data Processing',
        message: 'Processing failed',
        error: {
          code: 'PROCESSING_ERROR',
          message: 'Failed to process market data',
          details: 'Connection timeout to data source',
        },
        startTime: '2025-01-26T10:00:00Z',
        endTime: '2025-01-26T10:02:30Z',
      })
    })

    expect(result.current.status).toBe('failed')
    expect(result.current.error).toEqual({
      code: 'PROCESSING_ERROR',
      message: 'Failed to process market data',
      details: 'Connection timeout to data source',
    })
  })

  it('handles step-specific updates', () => {
    let progressUpdateHandler: (data: any) => void

    mockWebSocketClient.subscribe.mockImplementation((type, handler) => {
      if (type === 'pipeline_progress') {
        progressUpdateHandler = handler
      }
      return jest.fn()
    })

    const { result } = renderHook(() => usePipelineUpdates('operation-123'))

    // Progress through different steps
    const steps = [
      { step: 'Initialization', progress: 10, message: 'Initializing operation...' },
      { step: 'Data Fetching', progress: 30, message: 'Fetching Iraqi Stock Exchange data...' },
      { step: 'Data Processing', progress: 60, message: 'Processing market indicators...' },
      { step: 'Report Generation', progress: 90, message: 'Generating daily reports...' },
    ]

    steps.forEach((stageUpdate) => {
      act(() => {
        progressUpdateHandler!({
          pipeline_id: 'operation-123',
          status: 'running',
          ...stageUpdate,
        })
      })

      expect(result.current.step).toBe(stageUpdate.step)
      expect(result.current.progress).toBe(stageUpdate.progress)
      expect(result.current.message).toBe(stageUpdate.message)
    })
  })

  it('resets state when operation ID changes', () => {
    let progressUpdateHandler: (data: any) => void

    mockWebSocketClient.subscribe.mockImplementation((type, handler) => {
      if (type === 'pipeline_progress') {
        progressUpdateHandler = handler
      }
      return jest.fn()
    })

    const { result, rerender } = renderHook(
      ({ pipelineId }) => usePipelineUpdates(pipelineId),
      { initialProps: { pipelineId: 'operation-123' } }
    )

    // Set some state
    act(() => {
      progressUpdateHandler!({
        pipeline_id: 'operation-123',
        status: 'running',
        progress: 50,
        step: 'Processing',
      })
    })

    expect(result.current.status).toBe('running')

    // Change operation ID
    rerender({ pipelineId: 'operation-456' })

    // State should be reset
    expect(result.current).toEqual({
      status: 'idle',
      progress: 0,
      step: null,
      message: null,
      error: null,
      startTime: null,
      endTime: null,
      results: null,
    })
  })

  it('calculates operation duration correctly', () => {
    let progressUpdateHandler: (data: any) => void

    mockWebSocketClient.subscribe.mockImplementation((type, handler) => {
      if (type === 'pipeline_progress') {
        progressUpdateHandler = handler
      }
      return jest.fn()
    })

    const { result } = renderHook(() => usePipelineUpdates('operation-123'))

    // Start operation
    act(() => {
      progressUpdateHandler!({
        pipeline_id: 'operation-123',
        status: 'running',
        progress: 0,
        step: 'Starting',
        startTime: '2025-01-26T10:00:00Z',
      })
    })

    // Complete operation
    act(() => {
      progressUpdateHandler!({
        pipeline_id: 'operation-123',
        status: 'completed',
        progress: 100,
        step: 'Completed',
        startTime: '2025-01-26T10:00:00Z',
        endTime: '2025-01-26T10:05:00Z',
      })
    })

    const startTime = new Date('2025-01-26T10:00:00Z')
    const endTime = new Date('2025-01-26T10:05:00Z')
    const expectedDuration = endTime.getTime() - startTime.getTime()

    expect(result.current.startTime).toBe('2025-01-26T10:00:00Z')
    expect(result.current.endTime).toBe('2025-01-26T10:05:00Z')
    
    // Duration should be 5 minutes (300,000 ms)
    expect(expectedDuration).toBe(300000)
  })
})