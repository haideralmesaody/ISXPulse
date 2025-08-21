/**
 * Segmented File Progress Component
 * Professional progress bar where each segment represents one file
 * Used for processing and index extraction operations
 */

'use client'

import React, { useMemo } from 'react'
import { cn } from '@/lib/utils'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'

interface FileSegment {
  filename: string
  index: number
  status: 'processed' | 'processing' | 'pending'
}

interface SegmentedFileProgressProps {
  totalFiles: number
  processedFiles: number
  currentFile?: string
  fileList?: string[]  // Optional list of all files
  className?: string
}

export function SegmentedFileProgress({
  totalFiles,
  processedFiles,
  currentFile,
  fileList,
  className
}: SegmentedFileProgressProps) {
  
  // Generate file segments
  const segments = useMemo(() => {
    const result: FileSegment[] = []
    
    // If we have a file list, use it
    if (fileList && fileList.length > 0) {
      fileList.forEach((filename, index) => {
        let status: FileSegment['status'] = 'pending'
        
        // Check if this file is currently being processed
        if (currentFile && (filename === currentFile || filename.includes(currentFile) || currentFile.includes(filename))) {
          status = 'processing'
        } else if (index < processedFiles) {
          status = 'processed'
        }
        
        result.push({
          filename,
          index,
          status
        })
      })
    } else {
      // Fallback: Generate generic segments based on counts
      for (let i = 0; i < totalFiles; i++) {
        let status: FileSegment['status'] = 'pending'
        
        if (i < processedFiles) {
          status = 'processed'
        } else if (i === processedFiles && currentFile) {
          status = 'processing'
        }
        
        result.push({
          filename: `File ${i + 1}`,
          index: i,
          status
        })
      }
    }
    
    return result
  }, [totalFiles, processedFiles, currentFile, fileList])
  
  // Calculate progress percentage
  const progress = useMemo(() => {
    if (totalFiles === 0) return 0
    const actualProgress = Math.round((processedFiles / totalFiles) * 100)
    return Math.min(actualProgress, 100)
  }, [processedFiles, totalFiles])
  
  // Don't render if no files
  if (totalFiles === 0 || segments.length === 0) {
    return null
  }
  
  // Calculate segment width
  const segmentWidth = 100 / segments.length
  
  // Limit segments display if too many files (show max 50 segments)
  const displaySegments = segments.length > 50 
    ? segments.filter((_, i) => {
        // Show first 20, last 20, and some in middle including current
        if (i < 20 || i >= segments.length - 20) return true
        if (segments[i].status === 'processing') return true
        // Sample middle segments
        return i % Math.floor(segments.length / 50) === 0
      })
    : segments
    
  const displaySegmentWidth = 100 / displaySegments.length
  
  return (
    <div className={cn("space-y-2", className)}>
      {/* Progress header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <span className="text-sm font-medium">Progress</span>
          <span className="text-lg font-bold">{progress}%</span>
          <span className="text-xs text-muted-foreground">
            ({processedFiles}/{totalFiles} files)
          </span>
        </div>
        
        {/* Current file indicator */}
        {currentFile && (
          <div className="text-xs text-muted-foreground truncate max-w-[300px]">
            Processing: {currentFile.split('/').pop()?.split('\\').pop() || currentFile}
          </div>
        )}
      </div>
      
      {/* Segmented progress bar */}
      <div className="w-full">
        <TooltipProvider delayDuration={0}>
          <div className="flex h-3 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-gray-800">
            {displaySegments.map((segment) => (
              <Tooltip key={`${segment.index}-${segment.filename}`}>
                <TooltipTrigger asChild>
                  <div
                    className={cn(
                      "h-full transition-all duration-300",
                      "hover:opacity-80 cursor-default",
                      // Colors based on status
                      segment.status === 'processed' && "bg-green-500",
                      segment.status === 'processing' && "bg-blue-500 animate-pulse",
                      segment.status === 'pending' && "bg-white dark:bg-gray-700",
                      // Add subtle borders between segments
                      segment.index > 0 && "border-l border-gray-300/30"
                    )}
                    style={{ width: `${displaySegmentWidth}%` }}
                  />
                </TooltipTrigger>
                <TooltipContent side="top" className="text-xs">
                  <div className="space-y-1">
                    <div className="font-medium">
                      {segment.filename.split('/').pop()?.split('\\').pop() || segment.filename}
                    </div>
                    <div className="text-muted-foreground">
                      {segment.status === 'processed' && 'Completed'}
                      {segment.status === 'processing' && 'Processing...'}
                      {segment.status === 'pending' && 'Pending'}
                    </div>
                  </div>
                </TooltipContent>
              </Tooltip>
            ))}
          </div>
        </TooltipProvider>
      </div>
      
      {/* Summary stats */}
      <div className="flex items-center gap-4 text-xs text-muted-foreground">
        <div className="flex items-center gap-1">
          <div className="h-2 w-2 rounded-sm bg-green-500" />
          <span>Processed: {processedFiles}</span>
        </div>
        {currentFile && (
          <div className="flex items-center gap-1">
            <div className="h-2 w-2 rounded-sm bg-blue-500 animate-pulse" />
            <span>Processing: 1</span>
          </div>
        )}
        <div className="flex items-center gap-1">
          <div className="h-2 w-2 rounded-sm bg-white dark:bg-gray-700 border border-gray-300" />
          <span>Remaining: {Math.max(0, totalFiles - processedFiles - (currentFile ? 1 : 0))}</span>
        </div>
      </div>
    </div>
  )
}