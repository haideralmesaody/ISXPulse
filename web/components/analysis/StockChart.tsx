/**
 * StockChart Component
 * Professional Highcharts Stock technical analysis chart
 * Uses dynamic module loading for Next.js compatibility
 */

'use client'

import React, { useEffect, useState, useMemo, useRef } from 'react'
import dynamic from 'next/dynamic'
import { useTheme } from 'next-themes'
import { Loader2, AlertCircle } from 'lucide-react'
import { transformToHighchartsData } from '@/lib/utils/chart-transformer'
import { useHydration } from '@/lib/hooks/use-hydration'
import { loadHighchartsModules, createChartOptions } from '@/lib/highcharts-loader'
import { applyHighchartsTheme } from '@/lib/highcharts-themes'
import type { TickerHistoricalData } from '@/types/analysis'
// Import official Highcharts CSS files first
import '@/styles/highcharts-gui.css'
import '@/styles/highcharts-popup.css'
// Then our custom overrides for dark mode
import '@/styles/highcharts-stock-overrides.css'

// Dynamically import HighchartsReact to prevent SSR issues
const HighchartsReact = dynamic(
  () => import('highcharts-react-official').then(mod => ({ default: mod.default })),
  { 
    ssr: false,
    loading: () => (
      <div className="h-full flex items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin" />
      </div>
    )
  }
)

interface StockChartProps {
  ticker: string
  data: TickerHistoricalData[]
  isLoading?: boolean
}

export function StockChart({ ticker, data, isLoading }: StockChartProps) {
  const chartRef = useRef<any>(null)
  const isHydrated = useHydration()
  const { theme, resolvedTheme } = useTheme()
  const [highcharts, setHighcharts] = useState<any>(null)
  const [moduleLoadError, setModuleLoadError] = useState<string | null>(null)
  const [isLoadingModules, setIsLoadingModules] = useState(true)

  // Transform data for Highcharts
  const chartData = useMemo(() => {
    if (!data || data.length === 0) return { ohlc: [], volume: [] }
    return transformToHighchartsData(data)
  }, [data])

  // Calculate resistance zone from data
  const resistanceZone = useMemo(() => {
    if (!data || data.length === 0) return { from: 0, to: 0 }
    
    const recentData = data.slice(-30)
    const highs = recentData.map(d => d.high).filter(h => h > 0)
    const lows = recentData.map(d => d.low).filter(l => l > 0)
    
    if (highs.length === 0 || lows.length === 0) return { from: 0, to: 0 }
    
    const avgHigh = highs.reduce((a, b) => a + b, 0) / highs.length
    const avgLow = lows.reduce((a, b) => a + b, 0) / lows.length
    
    return {
      from: avgHigh * 0.98,
      to: avgHigh * 1.02
    }
  }, [data])

  // Calculate Fibonacci points
  const fibonacciPoints = useMemo(() => {
    if (!chartData.ohlc || chartData.ohlc.length < 2) {
      return { point1: null, point2: null, height: 0 }
    }
    
    const prices = chartData.ohlc.map(d => d[4])
    const maxPrice = Math.max(...prices)
    const minPrice = Math.min(...prices)
    const maxIndex = prices.indexOf(maxPrice)
    const minIndex = prices.indexOf(minPrice)
    
    const [p1Index, p2Index] = maxIndex < minIndex ? [maxIndex, minIndex] : [minIndex, maxIndex]
    
    return {
      point1: {
        x: chartData.ohlc[p1Index][0],
        y: chartData.ohlc[p1Index][4]
      },
      point2: {
        x: chartData.ohlc[p2Index][0],
        y: chartData.ohlc[p2Index][4]
      },
      height: chartData.ohlc[p2Index][4] - chartData.ohlc[p1Index][4]
    }
  }, [chartData])

  // Load Highcharts modules when component mounts and is hydrated
  useEffect(() => {
    if (!isHydrated) return

    let mounted = true

    const loadModules = async () => {
      try {
        setIsLoadingModules(true)
        setModuleLoadError(null)
        
        const HC = await loadHighchartsModules()
        
        if (mounted) {
          // Apply theme after loading modules
          const currentTheme = resolvedTheme || 'light'
          applyHighchartsTheme(HC, currentTheme as 'light' | 'dark')
          
          setHighcharts(HC)
          setIsLoadingModules(false)
        }
      } catch (error) {
        if (mounted) {
          console.error('Failed to load Highcharts modules:', error)
          setModuleLoadError(error instanceof Error ? error.message : 'Failed to load chart modules')
          setIsLoadingModules(false)
        }
      }
    }

    loadModules()

    return () => {
      mounted = false
    }
  }, [isHydrated, resolvedTheme])

  // Apply theme changes to existing chart
  useEffect(() => {
    if (!highcharts || !isHydrated || !resolvedTheme) return
    
    // Apply new theme
    applyHighchartsTheme(highcharts, resolvedTheme as 'light' | 'dark')
    
    // Update existing chart if it exists
    if (chartRef.current && chartRef.current.chart) {
      // Force chart redraw with new theme
      chartRef.current.chart.update({}, true, true)
    }
  }, [resolvedTheme, highcharts, isHydrated])
  
  // Create chart options
  const options = useMemo(() => {
    if (!highcharts || !chartData.ohlc || chartData.ohlc.length === 0) {
      return null
    }
    
    // Pass the resolved theme to ensure single source of truth
    const currentTheme = resolvedTheme || 'light'
    return createChartOptions(ticker, chartData, resistanceZone, fibonacciPoints, currentTheme as 'light' | 'dark')
  }, [ticker, chartData, resistanceZone, fibonacciPoints, highcharts, resolvedTheme])

  // Loading states
  if (!isHydrated) {
    return (
      <div className="h-full flex items-center justify-center">
        <div className="text-center">
          <Loader2 className="h-8 w-8 animate-spin mx-auto mb-4" />
          <p className="text-muted-foreground">Initializing chart...</p>
        </div>
      </div>
    )
  }

  if (isLoadingModules) {
    return (
      <div className="h-full flex items-center justify-center">
        <div className="text-center">
          <Loader2 className="h-8 w-8 animate-spin mx-auto mb-4" />
          <p className="text-muted-foreground">Loading technical analysis modules...</p>
          <p className="text-sm text-muted-foreground mt-2">This may take a moment on first load</p>
        </div>
      </div>
    )
  }

  if (moduleLoadError) {
    return (
      <div className="h-full flex items-center justify-center">
        <div className="text-center max-w-md">
          <AlertCircle className="h-12 w-12 text-destructive mx-auto mb-4" />
          <p className="text-lg font-semibold mb-2">Chart Loading Error</p>
          <p className="text-sm text-muted-foreground mb-4">{moduleLoadError}</p>
          <button
            onClick={() => window.location.reload()}
            className="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90"
          >
            Reload Page
          </button>
        </div>
      </div>
    )
  }

  if (isLoading) {
    return (
      <div className="h-full flex items-center justify-center">
        <div className="text-center">
          <Loader2 className="h-8 w-8 animate-spin mx-auto mb-4" />
          <p className="text-muted-foreground">Loading chart data...</p>
        </div>
      </div>
    )
  }

  if (!data || data.length === 0) {
    return (
      <div className="h-full flex items-center justify-center">
        <div className="text-center">
          <p className="text-lg text-muted-foreground mb-2">No data available</p>
          <p className="text-sm text-muted-foreground">
            Historical data for {ticker} is not available
          </p>
        </div>
      </div>
    )
  }

  if (!highcharts || !options) {
    return (
      <div className="h-full flex items-center justify-center">
        <div className="text-center">
          <AlertCircle className="h-8 w-8 text-warning mx-auto mb-4" />
          <p className="text-muted-foreground">Unable to initialize chart</p>
        </div>
      </div>
    )
  }

  return (
    <div className="highcharts-chart-container w-full" style={{ height: '730px' }}>
      <HighchartsReact
        highcharts={highcharts}
        constructorType={'stockChart'}
        options={options}
        ref={chartRef}
        containerProps={{ 
          className: 'chart',
          style: { 
            height: '100%', 
            width: '100%',
            position: 'relative'
          } 
        }}
      />
    </div>
  )
}