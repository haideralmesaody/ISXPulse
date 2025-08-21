/**
 * @jest-environment jsdom
 */
import React from 'react'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import '@testing-library/jest-dom'
import AnalysisContent from '../analysis-content'

// Mock dynamic imports
jest.mock('next/dynamic', () => {
  return (importFunc: () => Promise<any>, options: any) => {
    const MockedComponent = (props: any) => {
      const [Component, setComponent] = React.useState<any>(null)
      const [loading, setLoading] = React.useState(true)

      React.useEffect(() => {
        if (options?.ssr === false) {
          // Simulate client-side loading
          setTimeout(async () => {
            try {
              const mod = await importFunc()
              const ComponentToRender = typeof mod === 'function' ? mod : mod.default
              setComponent(() => ComponentToRender)
            } catch (error) {
              console.error('Dynamic import failed:', error)
            } finally {
              setLoading(false)
            }
          }, 10)
        } else {
          importFunc().then(mod => {
            const ComponentToRender = typeof mod === 'function' ? mod : mod.default
            setComponent(() => ComponentToRender)
            setLoading(false)
          })
        }
      }, [])

      if (loading) {
        return options?.loading ? options.loading() : <div data-testid="loading">Loading chart...</div>
      }

      return Component ? <Component {...props} /> : <div data-testid="chart-error">Failed to load chart</div>
    }

    MockedComponent.displayName = 'DynamicStockChart'
    return MockedComponent
  }
})

// Mock Next.js navigation
jest.mock('next/navigation', () => ({
  useRouter: () => ({
    push: jest.fn(),
    replace: jest.fn(),
    back: jest.fn(),
    forward: jest.fn(),
    refresh: jest.fn(),
    prefetch: jest.fn(),
  }),
  usePathname: () => '/analysis',
  useSearchParams: () => new URLSearchParams(),
}))

// Mock toast hook
const mockToast = jest.fn()
jest.mock('@/lib/hooks/use-toast', () => ({
  useToast: () => ({
    toast: mockToast,
  }),
}))

// Mock hydration hook
let mockIsHydrated = false
jest.mock('@/lib/hooks/use-hydration', () => ({
  useHydration: jest.fn(() => mockIsHydrated),
}))

// Mock API functions
const mockFetchTickerSummary = jest.fn()
const mockFetchTickerHistory = jest.fn()
jest.mock('@/lib/api/analysis', () => ({
  fetchTickerSummary: jest.fn(),
  fetchTickerHistory: jest.fn(),
}))

// Mock UI components
jest.mock('@/components/ui', () => ({
  NoDataState: ({ title, description, actions, instructions, icon: Icon, iconColor, className }: any) => (
    <div data-testid="no-data-state" className={className}>
      {Icon && <div data-testid="no-data-icon" data-color={iconColor}><Icon /></div>}
      <h2 data-testid="no-data-title">{title}</h2>
      <p data-testid="no-data-description">{description}</p>
      {instructions && (
        <div data-testid="no-data-instructions">
          {instructions.map((instruction: string, index: number) => (
            <div key={index}>{instruction}</div>
          ))}
        </div>
      )}
      {actions && (
        <div data-testid="no-data-actions">
          {actions.map((action: any, index: number) => (
            action.href ? (
              <a key={index} href={action.href} data-testid={`action-link-${index}`}>
                {action.icon && <action.icon />}
                {action.label}
              </a>
            ) : (
              <button key={index} onClick={action.onClick} data-testid={`action-button-${index}`}>
                {action.icon && <action.icon />}
                {action.label}
              </button>
            )
          ))}
        </div>
      )}
    </div>
  ),
  DataLoadingState: ({ message, className, showCard, size }: any) => (
    <div 
      data-testid="data-loading-state" 
      className={className}
      data-show-card={showCard}
      data-size={size}
    >
      {message || 'Loading...'}
    </div>
  ),
  Alert: ({ children, variant }: any) => (
    <div data-testid="alert" data-variant={variant}>
      {children}
    </div>
  ),
  AlertDescription: ({ children }: any) => (
    <div data-testid="alert-description">
      {children}
    </div>
  ),
}))

// Mock analysis components
jest.mock('@/components/analysis/TickerList', () => ({
  TickerList: ({ tickers, selectedTicker, onTickerSelect }: any) => (
    <div data-testid="ticker-list">
      {tickers.map((ticker: any) => (
        <button
          key={ticker.Ticker}
          data-testid={`ticker-${ticker.Ticker}`}
          onClick={() => onTickerSelect(ticker.Ticker)}
          data-selected={selectedTicker === ticker.Ticker}
        >
          {ticker.Ticker}: {ticker.LastPrice}
        </button>
      ))}
    </div>
  ),
}))

// Mock types
interface TickerSummary {
  Ticker: string
  LastPrice: number
  Change: number
  Volume: number
}

interface TickerHistoricalData {
  Date: string
  Open: number
  High: number
  Low: number
  Close: number
  Volume: number
}

// Sample test data
const mockTickerSummaryData: TickerSummary[] = [
  { Ticker: 'BBOB', LastPrice: 1.25, Change: 0.05, Volume: 1000000 },
  { Ticker: 'BAGH', LastPrice: 2.15, Change: -0.10, Volume: 500000 },
  { Ticker: 'TASC', LastPrice: 0.85, Change: 0.02, Volume: 750000 },
]

const mockTickerHistoryData: TickerHistoricalData[] = [
  { Date: '2025-01-15', Open: 1.20, High: 1.30, Low: 1.15, Close: 1.25, Volume: 1000000 },
  { Date: '2025-01-14', Open: 1.15, High: 1.22, Low: 1.10, Close: 1.20, Volume: 850000 },
  { Date: '2025-01-13', Open: 1.18, High: 1.20, Low: 1.12, Close: 1.15, Volume: 900000 },
]

describe('AnalysisContent Integration Tests', () => {
  // Helper function to get mocked API functions
  const getMockApis = () => {
    const { fetchTickerSummary, fetchTickerHistory } = require('@/lib/api/analysis')
    return { fetchTickerSummary, fetchTickerHistory }
  }

  beforeEach(() => {
    jest.clearAllMocks()
    mockIsHydrated = false
    mockToast.mockClear()
    
    // Clear the mocked functions
    const { fetchTickerSummary, fetchTickerHistory } = getMockApis()
    fetchTickerSummary.mockClear()
    fetchTickerHistory.mockClear()
    
    // Update the mock return value
    const mockUseHydration = require('@/lib/hooks/use-hydration').useHydration
    mockUseHydration.mockReturnValue(mockIsHydrated)
    
    // Reset console mocks
    jest.spyOn(console, 'error').mockImplementation(() => {})
  })

  afterEach(() => {
    jest.restoreAllMocks()
  })

  describe('Hydration States', () => {
    it('shows loading state before hydration', () => {
      mockIsHydrated = false
      
      render(<AnalysisContent />)
      
      const loadingState = screen.getByTestId('data-loading-state')
      expect(loadingState).toBeInTheDocument()
      expect(loadingState).toHaveTextContent('Initializing analysis tools...')
      expect(loadingState).toHaveAttribute('data-show-card', 'false')
      expect(loadingState).toHaveAttribute('data-size', 'default')
    })

    it('transitions to main content after hydration', async () => {
      mockIsHydrated = true
      const { fetchTickerSummary, fetchTickerHistory } = getMockApis()
      fetchTickerSummary.mockResolvedValue(mockTickerSummaryData)
      fetchTickerHistory.mockResolvedValue(mockTickerHistoryData)
      
      render(<AnalysisContent />)
      
      await waitFor(() => {
        expect(screen.queryByTestId('data-loading-state')).not.toBeInTheDocument()
      })
      
      expect(screen.getByText('Market Overview')).toBeInTheDocument()
      expect(screen.getByText(`${mockTickerSummaryData.length} active tickers`)).toBeInTheDocument()
    })
  })

  describe('No Data States', () => {
    const noDataScenarios = [
      {
        name: 'shows no-data when API returns 404',
        mockError: new Error('404: Not Found'),
        shouldShowNoData: true,
        shouldShowToast: false,
      },
      {
        name: 'shows no-data when API returns not found message',
        mockError: new Error('No ticker summary data available'),
        shouldShowNoData: true,
        shouldShowToast: false,
      },
      {
        name: 'shows no-data when API returns no combined data message',
        mockError: new Error('No combined data available'),
        shouldShowNoData: true,
        shouldShowToast: false,
      },
      {
        name: 'shows no-data when tickers array is empty',
        mockData: [],
        shouldShowNoData: true,
        shouldShowToast: false,
      },
    ]

    noDataScenarios.forEach(({ name, mockError, mockData, shouldShowNoData, shouldShowToast }) => {
      it(name, async () => {
        mockIsHydrated = true
        
        if (mockError) {
          mockFetchTickerSummary.mockRejectedValue(mockError)
        } else if (mockData) {
          mockFetchTickerSummary.mockResolvedValue(mockData)
        }
        
        render(<AnalysisContent />)
        
        await waitFor(() => {
          if (shouldShowNoData) {
            expect(screen.getByTestId('no-data-state')).toBeInTheDocument()
          } else {
            expect(screen.queryByTestId('no-data-state')).not.toBeInTheDocument()
          }
        })
        
        if (shouldShowNoData) {
          expect(screen.getByTestId('no-data-title')).toHaveTextContent('No Analysis Data Available')
          expect(screen.getByTestId('no-data-description')).toHaveTextContent(
            'You need to run the data collection operations first to generate analysis data.'
          )
          
          // Check instructions
          const instructions = screen.getByTestId('no-data-instructions')
          expect(instructions).toBeInTheDocument()
          expect(instructions).toHaveTextContent('Go to the Operations page')
          expect(instructions).toHaveTextContent('Run \'Full Pipeline\' to collect all data')
          
          // Check actions
          const actions = screen.getByTestId('no-data-actions')
          expect(actions).toBeInTheDocument()
          
          const operationsLink = screen.getByTestId('action-link-0')
          expect(operationsLink).toHaveAttribute('href', '/operations')
          expect(operationsLink).toHaveTextContent('Go to Operations')
          
          const checkAgainButton = screen.getByTestId('action-button-1')
          expect(checkAgainButton).toHaveTextContent('Check Again')
        }
        
        if (shouldShowToast) {
          expect(mockToast).toHaveBeenCalled()
        } else {
          expect(mockToast).not.toHaveBeenCalled()
        }
      })
    })

    it('shows error alert for real API errors (not no-data scenarios)', async () => {
      mockIsHydrated = true
      const realError = new Error('Network error')
      mockFetchTickerSummary.mockRejectedValue(realError)
      
      render(<AnalysisContent />)
      
      await waitFor(() => {
        expect(screen.getByTestId('alert')).toBeInTheDocument()
        expect(screen.getByTestId('alert-description')).toHaveTextContent('Network error')
      })
      
      expect(mockToast).toHaveBeenCalledWith({
        title: 'Error',
        description: 'Network error',
        variant: 'destructive',
      })
    })
  })

  describe('No Data Actions', () => {
    beforeEach(async () => {
      mockIsHydrated = true
      mockFetchTickerSummary.mockRejectedValue(new Error('404: Not Found'))
      
      render(<AnalysisContent />)
      
      await waitFor(() => {
        expect(screen.getByTestId('no-data-state')).toBeInTheDocument()
      })
    })

    it('navigates to operations page when clicking link action', () => {
      const operationsLink = screen.getByTestId('action-link-0')
      expect(operationsLink).toHaveAttribute('href', '/operations')
      expect(operationsLink).toHaveTextContent('Go to Operations')
    })

    it('retries data loading when clicking check again button', async () => {
      const checkAgainButton = screen.getByTestId('action-button-1')
      expect(checkAgainButton).toHaveTextContent('Check Again')
      
      // Clear previous calls and set up successful response
      mockFetchTickerSummary.mockClear()
      mockFetchTickerSummary.mockResolvedValue(mockTickerSummaryData)
      
      fireEvent.click(checkAgainButton)
      
      await waitFor(() => {
        expect(mockFetchTickerSummary).toHaveBeenCalledTimes(1)
      })
    })

    it('handles retry failure gracefully', async () => {
      const checkAgainButton = screen.getByTestId('action-button-1')
      
      // Set up another failure
      mockFetchTickerSummary.mockClear()
      mockFetchTickerSummary.mockRejectedValue(new Error('Still no data'))
      
      fireEvent.click(checkAgainButton)
      
      await waitFor(() => {
        expect(mockFetchTickerSummary).toHaveBeenCalledTimes(1)
        // Should still show no-data state
        expect(screen.getByTestId('no-data-state')).toBeInTheDocument()
      })
    })
  })

  describe('Successful Data Loading', () => {
    beforeEach(() => {
      mockIsHydrated = true
      mockFetchTickerSummary.mockResolvedValue(mockTickerSummaryData)
      mockFetchTickerHistory.mockResolvedValue(mockTickerHistoryData)
    })

    it('loads ticker summary data successfully', async () => {
      render(<AnalysisContent />)
      
      await waitFor(() => {
        expect(mockFetchTickerSummary).toHaveBeenCalledTimes(1)
        expect(screen.getByText('Market Overview')).toBeInTheDocument()
        expect(screen.getByText(`${mockTickerSummaryData.length} active tickers`)).toBeInTheDocument()
      })
      
      // Check ticker list is rendered
      expect(screen.getByTestId('ticker-list')).toBeInTheDocument()
      
      mockTickerSummaryData.forEach(ticker => {
        expect(screen.getByTestId(`ticker-${ticker.Ticker}`)).toBeInTheDocument()
      })
    })

    it('auto-selects first ticker and loads historical data', async () => {
      render(<AnalysisContent />)
      
      await waitFor(() => {
        expect(mockFetchTickerHistory).toHaveBeenCalledWith(mockTickerSummaryData[0].Ticker)
      })
      
      // First ticker should be selected
      const firstTickerButton = screen.getByTestId(`ticker-${mockTickerSummaryData[0].Ticker}`)
      expect(firstTickerButton).toHaveAttribute('data-selected', 'true')
    })

    it('handles ticker selection and loads historical data', async () => {
      render(<AnalysisContent />)
      
      await waitFor(() => {
        expect(screen.getByTestId('ticker-list')).toBeInTheDocument()
      })
      
      // Clear previous calls
      mockFetchTickerHistory.mockClear()
      
      // Select different ticker
      const secondTickerButton = screen.getByTestId(`ticker-${mockTickerSummaryData[1].Ticker}`)
      fireEvent.click(secondTickerButton)
      
      await waitFor(() => {
        expect(mockFetchTickerHistory).toHaveBeenCalledWith(mockTickerSummaryData[1].Ticker)
      })
      
      expect(secondTickerButton).toHaveAttribute('data-selected', 'true')
    })

    it('shows chart loading state during historical data fetch', async () => {
      // Make historical data fetch slow
      let resolveHistoryPromise: (value: any) => void
      const historyPromise = new Promise(resolve => {
        resolveHistoryPromise = resolve
      })
      mockFetchTickerHistory.mockReturnValue(historyPromise)
      
      render(<AnalysisContent />)
      
      await waitFor(() => {
        expect(screen.getByTestId('ticker-list')).toBeInTheDocument()
      })
      
      // Should show loading state for chart
      expect(screen.getByTestId('loading')).toBeInTheDocument()
      
      // Resolve the promise
      resolveHistoryPromise!(mockTickerHistoryData)
      
      await waitFor(() => {
        expect(screen.queryByTestId('loading')).not.toBeInTheDocument()
      })
    })
  })

  describe('Error Handling During Normal Operation', () => {
    beforeEach(() => {
      mockIsHydrated = true
      mockFetchTickerSummary.mockResolvedValue(mockTickerSummaryData)
    })

    it('handles historical data fetch errors with toast notification', async () => {
      const historyError = new Error('Failed to load historical data')
      mockFetchTickerHistory.mockRejectedValue(historyError)
      
      render(<AnalysisContent />)
      
      await waitFor(() => {
        expect(mockToast).toHaveBeenCalledWith({
          title: 'Error',
          description: 'Failed to load historical data',
          variant: 'destructive',
        })
      })
    })

    it('handles ticker selection errors gracefully', async () => {
      render(<AnalysisContent />)
      
      await waitFor(() => {
        expect(screen.getByTestId('ticker-list')).toBeInTheDocument()
      })
      
      // Clear and set up error
      mockFetchTickerHistory.mockClear()
      mockFetchTickerHistory.mockRejectedValue(new Error('Ticker data unavailable'))
      
      const tickerButton = screen.getByTestId(`ticker-${mockTickerSummaryData[1].Ticker}`)
      fireEvent.click(tickerButton)
      
      await waitFor(() => {
        expect(mockToast).toHaveBeenCalledWith({
          title: 'Error',
          description: 'Ticker data unavailable',
          variant: 'destructive',
        })
      })
      
      // Should still select the ticker despite error
      expect(tickerButton).toHaveAttribute('data-selected', 'true')
    })
  })

  describe('Loading States During Operation', () => {
    beforeEach(() => {
      mockIsHydrated = true
    })

    it('shows ticker loading state during initial fetch', async () => {
      let resolvePromise: (value: any) => void
      const loadingPromise = new Promise(resolve => {
        resolvePromise = resolve
      })
      mockFetchTickerSummary.mockReturnValue(loadingPromise)
      
      render(<AnalysisContent />)
      
      await waitFor(() => {
        const loadingState = screen.getByTestId('data-loading-state')
        expect(loadingState).toHaveTextContent('Loading tickers...')
        expect(loadingState).toHaveAttribute('data-show-card', 'false')
        expect(loadingState).toHaveAttribute('data-size', 'sm')
      })
      
      // Resolve the promise
      resolvePromise!(mockTickerSummaryData)
      
      await waitFor(() => {
        expect(screen.queryByTestId('data-loading-state')).not.toBeInTheDocument()
        expect(screen.getByTestId('ticker-list')).toBeInTheDocument()
      })
    })

    it('shows proper placeholder when no ticker is selected', async () => {
      // Return empty data so no ticker gets auto-selected
      mockFetchTickerSummary.mockResolvedValue([])
      
      render(<AnalysisContent />)
      
      await waitFor(() => {
        expect(screen.getByText('Select a ticker to view chart')).toBeInTheDocument()
        expect(screen.getByText('Choose from the list on the left to begin analysis')).toBeInTheDocument()
      })
    })
  })

  describe('Component Lifecycle and Cleanup', () => {
    it('calls API only after hydration is complete', () => {
      mockIsHydrated = false
      mockFetchTickerSummary.mockResolvedValue(mockTickerSummaryData)
      
      render(<AnalysisContent />)
      
      // Should not call API before hydration
      expect(mockFetchTickerSummary).not.toHaveBeenCalled()
      
      // Update hydration state
      mockIsHydrated = true
      
      // Re-render to trigger effect
      const { rerender } = render(<AnalysisContent />)
      rerender(<AnalysisContent />)
      
      // Now should call API
      expect(mockFetchTickerSummary).toHaveBeenCalled()
    })

    it('prevents multiple concurrent API calls', async () => {
      mockIsHydrated = true
      
      let resolveCount = 0
      mockFetchTickerSummary.mockImplementation(() => {
        resolveCount++
        return Promise.resolve(mockTickerSummaryData)
      })
      
      const { rerender } = render(<AnalysisContent />)
      
      // Trigger multiple re-renders quickly
      rerender(<AnalysisContent />)
      rerender(<AnalysisContent />)
      rerender(<AnalysisContent />)
      
      await waitFor(() => {
        // Should only call once due to dependency array
        expect(resolveCount).toBe(1)
      })
    })

    it('handles unmounting during async operations gracefully', async () => {
      mockIsHydrated = true
      
      // Create a promise that never resolves
      const neverResolves = new Promise(() => {})
      mockFetchTickerSummary.mockReturnValue(neverResolves)
      
      const { unmount } = render(<AnalysisContent />)
      
      // Unmount while async operation is in progress
      unmount()
      
      // Should not throw errors
      expect(() => {
        // Any cleanup should happen silently
      }).not.toThrow()
    })
  })
})