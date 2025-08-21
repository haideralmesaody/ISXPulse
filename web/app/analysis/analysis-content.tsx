/**
 * Analysis Content Component
 * Main technical analysis page with ticker list and advanced charting
 * Following CLAUDE.md hydration best practices
 * This component is dynamically imported with SSR disabled
 */

'use client'

import React, { useState, useEffect, useCallback } from 'react'
import dynamic from 'next/dynamic'
import { Loader2, ChartCandlestick, Activity, RefreshCw, Info } from 'lucide-react'
import { useToast } from '@/lib/hooks/use-toast'
import { useHydration } from '@/lib/hooks/use-hydration'
import { TickerList } from '@/components/analysis/TickerList'
import { fetchTickerSummary, fetchTickerHistory } from '@/lib/api/analysis'
import { NoDataState, DataLoadingState, Alert, AlertDescription } from '@/components/ui'
import type { TickerSummary, TickerHistoricalData } from '@/types/analysis'
import { 
  trackNoDataResolved, 
  trackRetryAttempt, 
  debug,
  type NoDataStateContext 
} from '@/lib/observability/no-data-metrics'

// Dynamically import StockChart to avoid SSR issues with Highcharts
const StockChart = dynamic(() => import('@/components/analysis/StockChart').then(mod => mod.StockChart), {
  ssr: false,
  loading: () => (
    <div className="h-full flex items-center justify-center">
      <div className="text-center">
        <Loader2 className="h-8 w-8 animate-spin mx-auto mb-4" />
        <p className="text-muted-foreground">Loading chart...</p>
      </div>
    </div>
  )
})

export default function AnalysisContent() {
  // Hydration state
  const isHydrated = useHydration()
  
  // State management
  const [tickers, setTickers] = useState<TickerSummary[]>([])
  const [selectedTicker, setSelectedTicker] = useState<string | null>(null)
  const [historicalData, setHistoricalData] = useState<TickerHistoricalData[]>([])
  const [isLoadingTickers, setIsLoadingTickers] = useState(false)
  const [isLoadingChart, setIsLoadingChart] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [splitPosition, setSplitPosition] = useState(35) // percentage for left panel
  const [retryCount, setRetryCount] = useState(0)
  const [loadStartTime, setLoadStartTime] = useState<number | null>(null)
  
  const { toast } = useToast()
  
  // Helper function to detect no-data scenarios
  const isNoDataError = useCallback((error: string | Error) => {
    const errorMessage = error instanceof Error ? error.message : error
    return errorMessage.includes('404') || 
           errorMessage.includes('not found') ||
           errorMessage.includes('No ticker summary data available') ||
           errorMessage.includes('No combined data available')
  }, [])
  
  // Load ticker summary data
  const loadTickerSummary = useCallback(async (isRetry = false) => {
    setIsLoadingTickers(true)
    setError(null)
    
    // Track start time for performance measurement
    const startTime = Date.now()
    if (!isRetry) {
      setLoadStartTime(startTime)
    }
    
    try {
      debug.logApiResponse('/api/analysis/ticker-summary', null, false)
      
      const summaryData = await fetchTickerSummary()
      setTickers(summaryData)
      
      // Track successful data resolution if this was after an error state
      if (isRetry || loadStartTime) {
        const resolutionTime = Date.now() - (loadStartTime || startTime)
        trackNoDataResolved('analysis', resolutionTime, isRetry ? 'retry' : 'api_success')
        
        debug.logPerformance('Analysis Data Load', resolutionTime, {
          is_retry: isRetry,
          ticker_count: summaryData.length,
          retry_count: retryCount
        })
      }
      
      // Auto-select first ticker if available
      if (summaryData.length > 0 && !selectedTicker && summaryData[0]) {
        const firstTicker = summaryData[0].Ticker
        setSelectedTicker(firstTicker) // Just set the ticker, don't load data yet
      }
      
      // Reset retry count on success
      if (isRetry) {
        setRetryCount(0)
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to load ticker summary'
      setError(message)
      
      debug.logApiResponse('/api/analysis/ticker-summary', err, true)
      
      // Track retry attempt if this is a retry
      if (isRetry) {
        trackRetryAttempt('analysis', retryCount + 1, false)
        setRetryCount(prev => prev + 1)
      }
      
      // Only show toast for real errors, not no-data scenarios
      if (!isNoDataError(message)) {
        toast({
          title: 'Error',
          description: message,
          variant: 'destructive'
        })
      }
    } finally {
      setIsLoadingTickers(false)
    }
  }, [selectedTicker, toast, isNoDataError, loadStartTime, retryCount])
  
  // Load historical data for a ticker
  const loadTickerHistory = useCallback(async (ticker: string) => {
    setIsLoadingChart(true)
    setHistoricalData([])
    
    try {
      const history = await fetchTickerHistory(ticker)
      setHistoricalData(history)
    } catch (err) {
      const message = err instanceof Error ? err.message : `Failed to load data for ${ticker}`
      toast({
        title: 'Error',
        description: message,
        variant: 'destructive'
      })
    } finally {
      setIsLoadingChart(false)
    }
  }, [toast])
  
  // Load ticker summary on mount (only after hydration)
  useEffect(() => {
    if (isHydrated) {
      loadTickerSummary()
    }
  }, [isHydrated, loadTickerSummary])
  
  // Load historical data when ticker is selected (only after hydration)
  useEffect(() => {
    if (isHydrated && selectedTicker) {
      loadTickerHistory(selectedTicker)
    }
  }, [isHydrated, selectedTicker, loadTickerHistory])
  
  // Handle ticker selection
  const handleTickerSelect = useCallback((ticker: string) => {
    setSelectedTicker(ticker) // Just set the ticker, loading happens in useEffect
  }, [])
  
  // Handle split panel resize
  const handleSplitResize = useCallback((e: React.MouseEvent) => {
    e.preventDefault()
    
    const startX = e.clientX
    const startSplit = splitPosition
    
    const handleMouseMove = (e: MouseEvent) => {
      const deltaX = e.clientX - startX
      const containerWidth = window.innerWidth
      const deltaPercent = (deltaX / containerWidth) * 100
      const newSplit = Math.min(50, Math.max(20, startSplit + deltaPercent))
      setSplitPosition(newSplit)
    }
    
    const handleMouseUp = () => {
      document.removeEventListener('mousemove', handleMouseMove)
      document.removeEventListener('mouseup', handleMouseUp)
    }
    
    document.addEventListener('mousemove', handleMouseMove)
    document.addEventListener('mouseup', handleMouseUp)
  }, [splitPosition])
  
  // Show loading state during hydration
  if (!isHydrated) {
    return (
      <DataLoadingState 
        message="Initializing analysis tools..."
        className="h-[calc(100vh-128px)]"
        showCard={false}
        size="default"
        page="analysis"
        operation="hydration"
        trackPerformance={true}
      />
    )
  }
  
  // Show no-data state when appropriate
  if (!isLoadingTickers && error && (isNoDataError(error) || tickers.length === 0)) {
    return (
      <NoDataState
        icon={ChartCandlestick}
        iconColor="blue"
        title="No Analysis Data Available"
        description="You need to run the data collection operations first to generate analysis data."
        className="h-[calc(100vh-128px)] p-8"
        page="analysis"
        reason={error || "no_data_available"}
        componentName="AnalysisNoDataState"
        instructions={[
          "Go to the Operations page",
          "Run 'Full Pipeline' to collect all data",
          "Wait for the analysis to complete",
          "Return here to view the technical analysis"
        ]}
        actions={[
          {
            label: "Go to Operations",
            variant: "default",
            href: "/operations",
            icon: Activity
          },
          {
            label: "Check Again",
            variant: "outline",
            onClick: () => loadTickerSummary(true),
            icon: RefreshCw
          }
        ]}
      />
    )
  }
  
  return (
    <div className="h-[calc(100vh-128px)] flex relative">
      {/* Left Panel - Ticker List */}
      <div 
        className="border-r bg-card overflow-hidden flex flex-col"
        style={{ width: `${splitPosition}%` }}
      >
        <div className="p-4 border-b">
          <h2 className="text-lg font-semibold">Market Overview</h2>
          <p className="text-sm text-muted-foreground">
            {tickers.length} active tickers
          </p>
        </div>
        
        <div className="flex-1 overflow-hidden">
          {isLoadingTickers ? (
            <DataLoadingState 
              message="Loading tickers..."
              showCard={false}
              size="sm"
              className="h-full"
              page="analysis"
              operation="ticker_loading"
              trackPerformance={true}
            />
          ) : error && !isNoDataError(error) ? (
            <div className="p-4">
              <Alert variant="destructive">
                <Info className="h-4 w-4" />
                <AlertDescription>
                  <div className="space-y-2">
                    <p className="text-sm">{error}</p>
                    <button
                      onClick={() => loadTickerSummary(true)}
                      className="text-sm text-primary hover:underline underline-offset-4 transition-colors"
                    >
                      Try again
                    </button>
                  </div>
                </AlertDescription>
              </Alert>
            </div>
          ) : (
            <TickerList
              tickers={tickers}
              selectedTicker={selectedTicker}
              onTickerSelect={handleTickerSelect}
            />
          )}
        </div>
      </div>
      
      {/* Resize Handle */}
      <div
        className="w-1 bg-border hover:bg-primary/20 cursor-col-resize transition-colors"
        onMouseDown={handleSplitResize}
      />
      
      {/* Right Panel - Chart */}
      <div className="flex-1 overflow-hidden bg-background">
        {selectedTicker ? (
          <StockChart
            ticker={selectedTicker}
            data={historicalData}
            isLoading={isLoadingChart}
          />
        ) : (
          <div className="h-full flex items-center justify-center text-muted-foreground">
            <div className="text-center">
              <p className="text-lg mb-2">Select a ticker to view chart</p>
              <p className="text-sm">Choose from the list on the left to begin analysis</p>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}