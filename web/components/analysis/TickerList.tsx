/**
 * TickerList Component
 * Displays sortable, searchable list of tickers with sparklines
 */

'use client'

import React, { useState, useMemo, useCallback } from 'react'
import { Input } from '@/components/ui/input'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Search, TrendingUp, TrendingDown, Minus, ChevronUp, ChevronDown } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { TickerSummary, SortConfig } from '@/types/analysis'

interface TickerListProps {
  tickers: TickerSummary[]
  selectedTicker: string | null
  onTickerSelect: (ticker: string) => void
}

export function TickerList({ tickers, selectedTicker, onTickerSelect }: TickerListProps) {
  const [searchTerm, setSearchTerm] = useState('')
  const [sortConfig, setSortConfig] = useState<SortConfig[]>([
    { column: 'LastDate', direction: 'desc' },      // Newest trades first
    { column: 'ChangePercent', direction: 'desc' }, // Biggest gainers second
    { column: 'Ticker', direction: 'asc' }          // Alphabetical third
  ])
  
  // Handle column sort
  const handleSort = useCallback((column: keyof TickerSummary) => {
    setSortConfig(prev => {
      // Check if column is already in sort config
      const existingIndex = prev.findIndex(s => s.column === column)
      
      if (existingIndex === 0) {
        // Toggle direction if it's the primary sort
        return [
          { column, direction: prev[0].direction === 'asc' ? 'desc' : 'asc' },
          ...prev.slice(1)
        ]
      } else if (existingIndex > 0) {
        // Move to primary sort if it exists but isn't primary
        const existing = prev[existingIndex]
        return [
          { column, direction: existing.direction },
          ...prev.slice(0, existingIndex),
          ...prev.slice(existingIndex + 1)
        ]
      } else {
        // Add as primary sort
        return [
          { column, direction: 'desc' },
          ...prev.slice(0, 2) // Keep only top 3 sorts
        ]
      }
    })
  }, [])
  
  // Filter and sort tickers
  const filteredAndSortedTickers = useMemo(() => {
    let filtered = tickers
    
    // Apply search filter
    if (searchTerm) {
      const term = searchTerm.toLowerCase()
      filtered = tickers.filter(
        ticker =>
          ticker.Ticker.toLowerCase().includes(term) ||
          ticker.CompanyName.toLowerCase().includes(term)
      )
    }
    
    // Apply multi-level sort
    const sorted = [...filtered].sort((a, b) => {
      for (const sort of sortConfig) {
        const aVal = a[sort.column]
        const bVal = b[sort.column]
        
        let comparison = 0
        if (typeof aVal === 'string' && typeof bVal === 'string') {
          comparison = aVal.localeCompare(bVal)
        } else if (typeof aVal === 'number' && typeof bVal === 'number') {
          comparison = aVal - bVal
        }
        
        if (comparison !== 0) {
          return sort.direction === 'asc' ? comparison : -comparison
        }
      }
      return 0
    })
    
    return sorted
  }, [tickers, searchTerm, sortConfig])
  
  // Generate sparkline path
  const generateSparkline = useCallback((last10Days: string) => {
    if (!last10Days) return null
    
    const prices = last10Days.split(',').map(p => parseFloat(p.trim())).filter(p => !isNaN(p))
    if (prices.length < 2) return null
    
    const min = Math.min(...prices)
    const max = Math.max(...prices)
    const range = max - min || 1
    
    const width = 60
    const height = 20
    const points = prices.map((price, i) => {
      const x = (i / (prices.length - 1)) * width
      const y = height - ((price - min) / range) * height
      return `${x},${y}`
    }).join(' ')
    
    return `M ${points}`
  }, [])
  
  // Format number with locale
  const formatNumber = useCallback((num: number, decimals = 2) => {
    return new Intl.NumberFormat('en-US', {
      minimumFractionDigits: decimals,
      maximumFractionDigits: decimals
    }).format(num)
  }, [])
  
  // Format date for display
  const formatDate = useCallback((dateStr: string) => {
    if (!dateStr) return '-'
    try {
      const date = new Date(dateStr)
      return date.toLocaleDateString('en-US', { 
        month: 'short', 
        day: 'numeric' 
      }) // Returns "Aug 13"
    } catch {
      return dateStr
    }
  }, [])
  
  // Get sort indicator
  const getSortIndicator = (column: keyof TickerSummary) => {
    const sort = sortConfig.find(s => s.column === column)
    if (!sort) return null
    
    const index = sortConfig.findIndex(s => s.column === column)
    return (
      <span className="inline-flex items-center ml-1">
        {sort.direction === 'asc' ? (
          <ChevronUp className="h-3 w-3" />
        ) : (
          <ChevronDown className="h-3 w-3" />
        )}
        {sortConfig.length > 1 && (
          <span className="text-[10px] text-muted-foreground ml-0.5">
            {index + 1}
          </span>
        )}
      </span>
    )
  }
  
  return (
    <div className="flex flex-col h-full">
      {/* Search Bar */}
      <div className="p-3 border-b">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            type="text"
            placeholder="Search ticker or company..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="pl-9 h-9"
          />
        </div>
      </div>
      
      {/* Table Header */}
      <div className="grid grid-cols-12 gap-2 px-3 py-2 border-b bg-muted/50 text-xs font-medium text-muted-foreground">
        <div 
          className="col-span-2 cursor-pointer hover:text-foreground flex items-center"
          onClick={() => handleSort('Ticker')}
        >
          Ticker {getSortIndicator('Ticker')}
        </div>
        <div 
          className="col-span-3 cursor-pointer hover:text-foreground flex items-center"
          onClick={() => handleSort('CompanyName')}
        >
          Company {getSortIndicator('CompanyName')}
        </div>
        <div 
          className="col-span-2 text-right cursor-pointer hover:text-foreground flex items-center justify-end"
          onClick={() => handleSort('LastPrice')}
        >
          Price {getSortIndicator('LastPrice')}
        </div>
        <div 
          className="col-span-2 text-center cursor-pointer hover:text-foreground flex items-center justify-center"
          onClick={() => handleSort('LastDate')}
        >
          Date {getSortIndicator('LastDate')}
        </div>
        <div 
          className="col-span-2 text-right cursor-pointer hover:text-foreground flex items-center justify-end"
          onClick={() => handleSort('ChangePercent')}
        >
          Change {getSortIndicator('ChangePercent')}
        </div>
        <div className="col-span-1 text-center">
          Trend
        </div>
      </div>
      
      {/* Ticker List */}
      <ScrollArea className="flex-1">
        <div className="pb-2">
          {filteredAndSortedTickers.map((ticker) => {
            const changePercent = ticker.ChangePercent || 0
            const didTrade = ticker.LastTradingStatus !== false // Default to true if undefined
            const isPositive = changePercent > 0
            const isNegative = changePercent < 0
            const sparklinePath = generateSparkline(ticker.Last10Days)
            
            return (
              <div
                key={ticker.Ticker}
                className={cn(
                  "grid grid-cols-12 gap-2 px-3 py-2 hover:bg-muted/50 cursor-pointer transition-colors border-b",
                  selectedTicker === ticker.Ticker && "bg-primary/10 hover:bg-primary/15"
                )}
                onClick={() => onTickerSelect(ticker.Ticker)}
              >
                <div className="col-span-2 font-medium text-sm">
                  {ticker.Ticker}
                </div>
                <div className="col-span-3 text-sm text-muted-foreground truncate" title={ticker.CompanyName}>
                  {ticker.CompanyName}
                </div>
                <div className="col-span-2 text-right text-sm font-medium">
                  {formatNumber(ticker.LastPrice)}
                </div>
                <div className="col-span-2 text-center text-sm text-muted-foreground">
                  {formatDate(ticker.LastDate)}
                </div>
                <div className={cn(
                  "col-span-2 text-right text-sm font-medium flex items-center justify-end",
                  didTrade && isPositive && "text-green-600",
                  didTrade && isNegative && "text-red-600",
                  !didTrade && "text-muted-foreground"
                )}>
                  {!didTrade ? (
                    <>
                      <Minus className="h-3 w-3 mr-1" />
                      <span>-</span>
                    </>
                  ) : (
                    <>
                      {isPositive && <TrendingUp className="h-3 w-3 mr-1" />}
                      {isNegative && <TrendingDown className="h-3 w-3 mr-1" />}
                      {!isPositive && !isNegative && <Minus className="h-3 w-3 mr-1" />}
                      {formatNumber(Math.abs(changePercent))}%
                    </>
                  )}
                </div>
                <div className="col-span-1 flex items-center justify-center">
                  {sparklinePath && (
                    <svg width="60" height="20" className="overflow-visible">
                      <path
                        d={sparklinePath}
                        fill="none"
                        stroke={isPositive ? "#10b981" : isNegative ? "#ef4444" : "#6b7280"}
                        strokeWidth="1.5"
                      />
                    </svg>
                  )}
                </div>
              </div>
            )
          })}
        </div>
      </ScrollArea>
      
      {/* Footer */}
      <div className="p-3 border-t text-xs text-muted-foreground">
        Showing {filteredAndSortedTickers.length} of {tickers.length} tickers
      </div>
    </div>
  )
}