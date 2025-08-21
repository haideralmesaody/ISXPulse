/**
 * Segmented Day Progress Component
 * Professional progress bar where each segment represents one day
 * with intelligent holiday detection and color coding
 */

'use client'

import React, { useMemo } from 'react'
import { cn } from '@/lib/utils'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'

interface DaySegment {
  date: Date
  dateString: string
  dayOfWeek: number
  dayName: string
  dayNumber: number
  monthName: string
  status: 'downloaded' | 'pending' | 'weekend' | 'holiday' | 'downloading'
}

interface SegmentedDayProgressProps {
  fromDate?: string
  toDate?: string
  downloadedFiles?: string[]
  currentFile?: string
  metadata?: {
    files_processed?: number
    total_files?: number
    phase?: string
    skipped_files?: string[]
    completed?: boolean
    status?: string
  }
  className?: string
}

export function SegmentedDayProgress({
  fromDate,
  toDate,
  downloadedFiles = [],
  currentFile,
  metadata,
  className
}: SegmentedDayProgressProps) {
  
  // Generate day segments
  const segments = useMemo(() => {
    if (!fromDate || !toDate) return []
    
    const result: DaySegment[] = []
    const start = new Date(fromDate)
    const end = new Date(toDate)
    
    // Create set of downloaded dates for fast lookup
    // Handle both space-separated (2025 08 07) and hyphenated (2025-08-07) formats
    const downloadedDates = new Set(
      downloadedFiles.map(f => {
        // Try space-separated format first (backend sends this)
        const spaceMatch = f.match(/(\d{4})\s+(\d{2})\s+(\d{2})/)
        if (spaceMatch) {
          return `${spaceMatch[1]}-${spaceMatch[2]}-${spaceMatch[3]}`
        }
        // Fallback to hyphenated format
        const hyphenMatch = f.match(/\d{4}-\d{2}-\d{2}/)
        return hyphenMatch ? hyphenMatch[0] : null
      }).filter(Boolean)
    )
    
    // Check if operation is truly complete (multiple checks for safety)
    const isReallyComplete = metadata?.phase === 'completed' || 
                            metadata?.phase === 'complete' ||
                            metadata?.status === 'completed' ||
                            metadata?.completed === true
    
    // Extract currently downloading date (handle both formats)
    // CRITICAL: Never set downloading date if operation is complete
    let downloadingDate: string | undefined
    if (!isReallyComplete && currentFile) {
      const spaceMatch = currentFile.match(/(\d{4})\s+(\d{2})\s+(\d{2})/)
      if (spaceMatch) {
        downloadingDate = `${spaceMatch[1]}-${spaceMatch[2]}-${spaceMatch[3]}`
      } else {
        downloadingDate = currentFile.match(/\d{4}-\d{2}-\d{2}/)?.[0]
      }
    }
    
    // Build list of all trading days that should have files (in REVERSE order)
    const tradingDays: string[] = []
    // Start from END date and go backwards to START date
    for (let d = new Date(end); d >= start; d.setDate(d.getDate() - 1)) {
      const current = new Date(d)
      const dayOfWeek = current.getDay()
      // Iraq weekend is Friday (5) and Saturday (6)
      if (dayOfWeek !== 5 && dayOfWeek !== 6) {
        tradingDays.push(current.toISOString().split('T')[0])
      }
    }
    
    // Get skipped files from metadata (real-time holiday detection)
    const skippedFilesArray = metadata?.skipped_files || []
    
    // Create set of skipped dates for fast lookup
    const skippedDates = new Set(
      skippedFilesArray.map(f => {
        // Try space-separated format first (backend sends this)
        const spaceMatch = f.match(/(\d{4})\s+(\d{2})\s+(\d{2})/)
        if (spaceMatch) {
          return `${spaceMatch[1]}-${spaceMatch[2]}-${spaceMatch[3]}`
        }
        // Fallback to hyphenated format
        const hyphenMatch = f.match(/\d{4}-\d{2}-\d{2}/)
        return hyphenMatch ? hyphenMatch[0] : null
      }).filter(Boolean)
    )
    
    // Check if operation is complete (based on metadata phase)
    const isOperationComplete = metadata?.phase === 'complete' || metadata?.phase === 'completed'
    
    // Detect holidays: trading days with missing files
    const holidays = new Set<string>()
    
    // First, add explicitly skipped files (real-time detection)
    for (const tradingDay of tradingDays) {
      if (skippedDates.has(tradingDay)) {
        holidays.add(tradingDay)
      }
    }
    
    // Then, use completion-based detection for any remaining
    if (isOperationComplete) {
      // If operation is complete, all missing trading days are holidays
      for (const tradingDay of tradingDays) {
        if (!downloadedDates.has(tradingDay) && !holidays.has(tradingDay)) {
          holidays.add(tradingDay)
        }
      }
    } else if (holidays.size === 0) {
      // Fallback: Operation still running and no explicit skips yet
      // Find the oldest downloaded file (since we download newest to oldest)
      let oldestDownloadedDate: string | null = null
      
      // Sort trading days from oldest to newest
      const sortedTradingDays = [...tradingDays].sort()
      
      // Find the oldest downloaded date
      for (const day of sortedTradingDays) {
        if (downloadedDates.has(day)) {
          oldestDownloadedDate = day
          break
        }
      }
      
      // Any missing trading day newer than the oldest downloaded is likely a holiday
      if (oldestDownloadedDate) {
        for (const day of tradingDays) {
          if (day > oldestDownloadedDate && !downloadedDates.has(day)) {
            holidays.add(day)
          }
        }
      }
    }
    
    // Generate segments for each day (REVERSE ORDER - most recent first)
    // Start from END date and go backwards to START date
    for (let d = new Date(end); d >= start; d.setDate(d.getDate() - 1)) {
      const current = new Date(d)
      const dateStr = current.toISOString().split('T')[0]
      const dayOfWeek = current.getDay()
      
      // Determine status
      let status: DaySegment['status']
      if (dateStr === downloadingDate) {
        status = 'downloading'
      } else if (dayOfWeek === 5 || dayOfWeek === 6) {
        status = 'weekend'
      } else if (holidays.has(dateStr)) {
        status = 'holiday'
      } else if (downloadedDates.has(dateStr)) {
        status = 'downloaded'
      } else {
        status = 'pending'
      }
      
      result.push({
        date: new Date(current),
        dateString: dateStr,
        dayOfWeek,
        dayName: current.toLocaleDateString('en-US', { weekday: 'short' }),
        dayNumber: current.getDate(),
        monthName: current.toLocaleDateString('en-US', { month: 'short' }),
        status
      })
    }
    
    return result
  }, [fromDate, toDate, downloadedFiles, currentFile, metadata?.phase])
  
  // Calculate statistics
  const stats = useMemo(() => {
    const total = segments.length
    const weekends = segments.filter(s => s.status === 'weekend').length
    const holidays = segments.filter(s => s.status === 'holiday').length
    const downloaded = segments.filter(s => s.status === 'downloaded').length
    const downloading = segments.filter(s => s.status === 'downloading').length
    const tradingDays = total - weekends - holidays
    const pending = segments.filter(s => s.status === 'pending').length
    
    // Check if operation is complete
    const isComplete = metadata?.phase === 'completed' || 
                       metadata?.phase === 'complete' ||
                       metadata?.completed === true
    
    // Force 100% if operation is complete
    let progress: number
    if (isComplete) {
      progress = 100
    } else if (tradingDays > 0) {
      progress = Math.round(((downloaded + downloading * 0.5) / tradingDays) * 100)
      // Cap at 99% if not complete to avoid confusion
      if (progress >= 100) {
        progress = 99
      }
    } else {
      progress = 0
    }
    
    return {
      total,
      weekends,
      holidays,
      downloaded,
      downloading,
      tradingDays,
      pending,
      progress
    }
  }, [segments, metadata])
  
  // Don't render if no date range
  if (!fromDate || !toDate || segments.length === 0) {
    return null
  }
  
  // Calculate segment width
  const segmentWidth = 100 / segments.length
  
  return (
    <div className={cn("space-y-2", className)}>
      {/* Progress header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <span className="text-sm font-medium">Progress</span>
          <span className="text-lg font-bold">{stats.progress}%</span>
          <span className="text-xs text-muted-foreground">
            ({stats.downloaded}/{stats.tradingDays} trading days)
          </span>
        </div>
        
        {/* Compact legend */}
        <div className="flex items-center gap-2 text-xs">
          <div className="flex items-center gap-1">
            <div className="h-2 w-2 rounded-sm bg-green-500" />
            <span className="text-muted-foreground">Done</span>
          </div>
          <div className="flex items-center gap-1">
            <div className="h-2 w-2 rounded-sm bg-gray-600" />
            <span className="text-muted-foreground">Weekend</span>
          </div>
          <div className="flex items-center gap-1">
            <div className="h-2 w-2 rounded-sm bg-orange-500" />
            <span className="text-muted-foreground">Holiday</span>
          </div>
        </div>
      </div>
      
      {/* Segmented progress bar */}
      <div className="w-full">
        <TooltipProvider delayDuration={0}>
          <div className="flex h-3 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-gray-800">
            {segments.map((segment, index) => (
              <Tooltip key={segment.dateString}>
                <TooltipTrigger asChild>
                  <div
                    className={cn(
                      "h-full transition-all duration-300",
                      "hover:opacity-80 cursor-default",
                      // Colors based on status
                      segment.status === 'downloaded' && "bg-green-500",
                      segment.status === 'downloading' && "bg-blue-500 animate-pulse",
                      segment.status === 'weekend' && "bg-gray-600",
                      segment.status === 'holiday' && "bg-orange-500",
                      segment.status === 'pending' && "bg-white dark:bg-gray-700",
                      // Add subtle borders between segments
                      index > 0 && "border-l border-gray-300/30"
                    )}
                    style={{ width: `${segmentWidth}%` }}
                  />
                </TooltipTrigger>
                <TooltipContent className="text-xs">
                  <div className="space-y-0.5">
                    <div className="font-medium">
                      {segment.dayName}, {segment.monthName} {segment.dayNumber}
                    </div>
                    <div className="text-muted-foreground capitalize">
                      {segment.status === 'downloading' ? '⏳ Downloading...' :
                       segment.status === 'downloaded' ? '✓ Downloaded' :
                       segment.status === 'weekend' ? 'Weekend' :
                       segment.status === 'holiday' ? '🎉 Holiday' :
                       'Pending'}
                    </div>
                  </div>
                </TooltipContent>
              </Tooltip>
            ))}
          </div>
        </TooltipProvider>
      </div>
      
      {/* Statistics */}
      {(stats.holidays > 0 || stats.pending > 0) && (
        <div className="flex items-center justify-between text-xs text-muted-foreground">
          <div>
            {new Date(toDate).toLocaleDateString('en-US', { month: 'short', day: 'numeric' })}
            {' - '}
            {new Date(fromDate).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })}
          </div>
          <div className="flex items-center gap-3">
            {stats.pending > 0 && (
              <span>{stats.pending} remaining</span>
            )}
            {stats.holidays > 0 && (
              <span>{stats.holidays} holidays detected</span>
            )}
          </div>
        </div>
      )}
    </div>
  )
}

export default SegmentedDayProgress