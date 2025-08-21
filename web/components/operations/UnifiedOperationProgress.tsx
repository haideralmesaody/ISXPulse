/**
 * Unified Operation Progress Component
 * Clean, single source of truth for operation status display
 */

'use client'

import React, { useMemo } from 'react'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import { 
  CheckCircle2,
  Clock,
  Download,
  FileSearch,
  Loader2,
  AlertCircle,
  ArrowRight,
  Package,
  FileText,
  Zap
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { SegmentedDayProgress } from './SegmentedDayProgress'
import { SegmentedFileProgress } from './SegmentedFileProgress'

// Operation phases with clear progression
const OPERATION_PHASES = {
  scraping: [
    { id: 'preparing', label: 'Preparing', icon: Clock, progress: 0 },
    { id: 'scanning', label: 'Scanning', icon: FileSearch, progress: 25 },
    { id: 'downloading', label: 'Collecting', icon: Download, progress: 50 },
    { id: 'verifying', label: 'Verifying', icon: Package, progress: 75 },
    { id: 'complete', label: 'Complete', icon: CheckCircle2, progress: 100 }
  ],
  processing: [
    { id: 'preparing', label: 'Initializing', icon: Clock, progress: 0 },
    { id: 'reading', label: 'Reading Excel Files', icon: FileText, progress: 25 },
    { id: 'transforming', label: 'Converting to CSV', icon: Zap, progress: 50 },
    { id: 'writing', label: 'Writing Reports', icon: FileText, progress: 75 },
    { id: 'complete', label: 'Complete', icon: CheckCircle2, progress: 100 }
  ],
  indices: [
    { id: 'preparing', label: 'Preparing', icon: Clock, progress: 0 },
    { id: 'extracting', label: 'Extracting Indices', icon: FileSearch, progress: 50 },
    { id: 'complete', label: 'Complete', icon: CheckCircle2, progress: 100 }
  ],
  liquidity: [
    { id: 'preparing', label: 'Initializing', icon: Clock, progress: 0 },
    { id: 'reading', label: 'Loading Data', icon: FileText, progress: 20 },
    { id: 'calculating', label: 'Calculating Scores', icon: Zap, progress: 40 },
    { id: 'scaling', label: 'Cross-sectional Scaling', icon: Package, progress: 70 },
    { id: 'writing', label: 'Writing Results', icon: FileText, progress: 90 },
    { id: 'complete', label: 'Complete', icon: CheckCircle2, progress: 100 }
  ]
} as const

interface UnifiedOperationProgressProps {
  operation: {
    operation_id: string
    name?: string
    status: string
    progress: number
    message?: string
    error?: string
    metadata?: {
      phase?: string
      files_processed?: number
      total_files?: number
      current_file?: string
      speed?: number
      from_date?: string
      to_date?: string
      downloaded_files?: string[]
    }
    steps?: Array<{
      id: string
      name: string
      status: string
      progress: number
      metadata?: {
        phase?: string
        files_processed?: number
        total_files?: number
        current_file?: string
        speed?: number
        from_date?: string
        to_date?: string
        downloaded_files?: string[]
      }
    }>
  }
  onNextOperation?: (type: string) => void
}

export function UnifiedOperationProgress({ 
  operation,
  onNextOperation 
}: UnifiedOperationProgressProps) {
  
  // Get metadata from steps array (backend sends it in steps[0].metadata)
  const stepMetadata = useMemo(() => {
    if (operation.steps && operation.steps.length > 0) {
      return operation.steps[0].metadata || {}
    }
    return operation.metadata || {}
  }, [operation.steps, operation.metadata])
  
  // Determine operation type
  const operationType = useMemo(() => {
    if (operation.name?.toLowerCase().includes('scraping') || 
        operation.name?.toLowerCase().includes('collection')) {
      return 'scraping'
    }
    if (operation.name?.toLowerCase().includes('process')) {
      return 'processing'
    }
    if (operation.name?.toLowerCase().includes('index') || 
        operation.name?.toLowerCase().includes('indices')) {
      return 'indices'
    }
    if (operation.name?.toLowerCase().includes('liquidity') || 
        operation.name?.toLowerCase().includes('liquid')) {
      return 'liquidity'
    }
    return 'scraping' // default
  }, [operation.name])

  const phases = OPERATION_PHASES[operationType]
  
  // Determine current phase
  const currentPhase = useMemo(() => {
    // Check if completed - single source of truth: operation.status
    if (operation.status === 'completed') {
      return phases[phases.length - 1]
    }
    
    // Check metadata for phase hint
    if (operation.metadata?.phase) {
      const found = phases.find(p => p.id === operation.metadata?.phase)
      if (found) return found
    }
    
    // Determine by progress based on operation type
    if (operationType === 'processing') {
      // Processing has 5 phases: 0%, 25%, 50%, 75%, 100%
      if (operation.progress <= 10) return phases[0]  // Initializing
      if (operation.progress <= 35) return phases[1]  // Reading Excel Files
      if (operation.progress <= 60) return phases[2]  // Converting to CSV
      if (operation.progress <= 85) return phases[3]  // Writing Reports
      return phases[4]  // Complete
    } else if (operationType === 'indices') {
      // Indices has 3 phases: 0%, 50%, 100%
      if (operation.progress <= 25) return phases[0]  // Preparing
      if (operation.progress <= 75) return phases[1]  // Extracting
      return phases[2]  // Complete
    } else if (operationType === 'liquidity') {
      // Liquidity has 6 phases: 0%, 20%, 40%, 70%, 90%, 100%
      if (operation.progress <= 10) return phases[0]  // Initializing
      if (operation.progress <= 25) return phases[1]  // Loading Data
      if (operation.progress <= 50) return phases[2]  // Calculating Scores
      if (operation.progress <= 75) return phases[3]  // Cross-sectional Scaling
      if (operation.progress <= 95) return phases[4]  // Writing Results
      return phases[5]  // Complete
    } else {
      // Scraping has 5 phases: 0%, 25%, 50%, 75%, 100%
      if (operation.progress <= 10) return phases[0]
      if (operation.progress <= 35) return phases[1]
      if (operation.progress <= 60) return phases[2]
      if (operation.progress <= 85) return phases[3]
      return phases[4]
    }
  }, [operation, phases])

  const CurrentIcon = currentPhase.icon
  const isComplete = operation.status === 'completed'  // Single source of truth
  const isFailed = operation.status === 'failed'
  
  // Generate status message
  const statusMessage = useMemo(() => {
    if (operation.message) return operation.message
    
    if (isFailed) {
      return operation.error || 'Operation failed. Please try again.'
    }
    
    if (isComplete) {
      const fileCount = stepMetadata?.files_processed || 0
      return `✅ Operation completed successfully - ${fileCount} file${fileCount !== 1 ? 's' : ''} processed`
    }
    
    // For scraping with segmented progress, only show meaningful messages
    if (operationType === 'scraping' && stepMetadata?.from_date && stepMetadata?.to_date) {
      const current = stepMetadata?.current_file
      if (current && !isComplete) {
        const fileName = current.split('/').pop() || current.split('\\').pop() || current
        return `Downloading: ${fileName}`
      }
      if (currentPhase.id === 'scanning') {
        return 'Scanning for files...'
      }
      if (currentPhase.id === 'preparing') {
        return 'Initializing...'
      }
      // Return null for segmented progress to avoid redundant messages
      return null
    }
    
    // For other operations, use default messages
    switch (currentPhase.id) {
      case 'preparing':
        return 'Setting up operation environment...'
      case 'scanning':
        return `Scanning for files to process...`
      case 'downloading':
      case 'collecting':
        const current = stepMetadata?.current_file
        const processed = stepMetadata?.files_processed || 0
        const total = stepMetadata?.total_files
        if (current) {
          return `Downloading: ${current.split('/').pop()}`
        }
        if (total) {
          return `Processing files: ${processed} of ${total}`
        }
        return `Processing files... (${processed} completed)`
      case 'verifying':
        return 'Verifying data integrity...'
      case 'reading':
        return 'Reading input files...'
      case 'transforming':
        return 'Transforming data...'
      case 'extracting':
        return 'Extracting market indices...'
      case 'calculating':
        return 'Calculating liquidity scores (Impact, Volume, Continuity)...'
      case 'scaling':
        return 'Applying cross-sectional scaling for relative ranking...'
      default:
        return 'Processing...'
    }
  }, [operation, currentPhase, isComplete, isFailed, operationType, stepMetadata])

  // Calculate adjusted progress for smooth visual
  const visualProgress = useMemo(() => {
    if (isComplete) return 100
    if (operation.progress > 0) return Math.min(operation.progress, 95)
    return currentPhase.progress
  }, [operation.progress, currentPhase, isComplete])

  return (
    <Card className={cn(
      "w-full overflow-hidden transition-all duration-300",
      isComplete && "ring-2 ring-green-500/20",
      isFailed && "ring-2 ring-red-500/20"
    )}>
      {/* Status bar at top */}
      <div className={cn(
        "h-1 w-full",
        isComplete && "bg-gradient-to-r from-green-500 to-green-400",
        isFailed && "bg-gradient-to-r from-red-500 to-red-400",
        !isComplete && !isFailed && "bg-gradient-to-r from-blue-500 to-blue-400"
      )} />
      
      <CardHeader className="pb-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className={cn(
              "p-2 rounded-lg",
              isComplete && "bg-green-100",
              isFailed && "bg-red-100",
              !isComplete && !isFailed && "bg-blue-100"
            )}>
              <CurrentIcon className={cn(
                "h-5 w-5",
                isComplete && "text-green-700",
                isFailed && "text-red-700",
                !isComplete && !isFailed && "text-blue-700",
                currentPhase.id === 'preparing' && "animate-pulse",
                (currentPhase.id === 'scanning' || currentPhase.id === 'downloading') && "animate-bounce"
              )} />
            </div>
            <div>
              <h3 className="font-semibold text-base">
                {operation.name || 'Operation'}
              </h3>
              {/* Only show phase text for non-scraping operations or when no date range */}
              {!(operationType === 'scraping' && stepMetadata?.from_date && stepMetadata?.to_date) && (
                <p className="text-sm text-muted-foreground">
                  Phase {phases.findIndex(p => p.id === currentPhase.id) + 1} of {phases.length}
                </p>
              )}
            </div>
          </div>
          
          <Badge 
            variant={isComplete ? "success" : isFailed ? "destructive" : "default"}
            className="font-medium"
          >
            {currentPhase.label}
          </Badge>
        </div>
      </CardHeader>
      
      <CardContent className="space-y-4">
        {/* Use SegmentedDayProgress for scraping operations with date ranges */}
        {operationType === 'scraping' && stepMetadata?.from_date && stepMetadata?.to_date ? (
          <>
            <SegmentedDayProgress
              fromDate={stepMetadata.from_date}
              toDate={stepMetadata.to_date}
              downloadedFiles={stepMetadata.downloaded_files}
              currentFile={stepMetadata.current_file}
              metadata={stepMetadata}
            />
            
            {/* Simplified status message only - no redundant phase indicators */}
            {statusMessage && (
              <div className={cn(
                "p-3 rounded-lg",
                isComplete && "bg-green-50 dark:bg-green-950",
                isFailed && "bg-red-50 dark:bg-red-950",
                !isComplete && !isFailed && "bg-blue-50 dark:bg-blue-950"
              )}>
                <p className={cn(
                  "text-sm font-medium",
                  isComplete && "text-green-800 dark:text-green-200",
                  isFailed && "text-red-800 dark:text-red-200",
                  !isComplete && !isFailed && "text-blue-800 dark:text-blue-200"
                )}>
                  {statusMessage}
                </p>
                
                {/* File processing speed if available */}
                {stepMetadata?.speed && !isComplete && (
                  <div className="flex items-center gap-4 mt-2 text-xs opacity-70">
                    <span>Speed: {stepMetadata.speed.toFixed(1)} files/min</span>
                  </div>
                )}
              </div>
            )}
          </>
        ) : operationType === 'processing' || operationType === 'indices' || operationType === 'liquidity' ? (
          <>
            {/* Use SegmentedFileProgress for processing and index operations */}
            <SegmentedFileProgress
              totalFiles={stepMetadata?.total_files || 0}
              processedFiles={stepMetadata?.files_processed || 0}
              currentFile={stepMetadata?.current_file}
              fileList={stepMetadata?.file_list as string[] | undefined}
            />
            
            {/* Status message */}
            {statusMessage && (
              <div className={cn(
                "p-3 rounded-lg",
                isComplete && "bg-green-50 dark:bg-green-950",
                isFailed && "bg-red-50 dark:bg-red-950",
                !isComplete && !isFailed && "bg-blue-50 dark:bg-blue-950"
              )}>
                <p className={cn(
                  "text-sm font-medium",
                  isComplete && "text-green-800 dark:text-green-200",
                  isFailed && "text-red-800 dark:text-red-200",
                  !isComplete && !isFailed && "text-blue-800 dark:text-blue-200"
                )}>
                  {statusMessage}
                </p>
              </div>
            )}
          </>
        ) : (
          <>
            {/* Phase progress indicators for other operations (analysis, etc.) */}
            <div className="flex items-center justify-between px-2">
              {phases.map((phase, index) => {
                const isPast = phases.findIndex(p => p.id === currentPhase.id) > index
                const isCurrent = phase.id === currentPhase.id
                const PhaseIcon = phase.icon
                
                return (
                  <React.Fragment key={phase.id}>
                    <div className="flex flex-col items-center gap-1">
                      <div className={cn(
                        "p-1.5 rounded-full transition-all",
                        isPast && "bg-green-100",
                        isCurrent && "bg-blue-100 ring-2 ring-blue-300",
                        !isPast && !isCurrent && "bg-gray-100"
                      )}>
                        <PhaseIcon className={cn(
                          "h-3.5 w-3.5",
                          isPast && "text-green-600",
                          isCurrent && "text-blue-600",
                          !isPast && !isCurrent && "text-gray-400"
                        )} />
                      </div>
                      <span className={cn(
                        "text-xs",
                        isCurrent && "font-medium text-foreground",
                        !isCurrent && "text-muted-foreground"
                      )}>
                        {phase.label}
                      </span>
                    </div>
                    {index < phases.length - 1 && (
                      <div className={cn(
                        "flex-1 h-0.5 -mt-5",
                        isPast && "bg-green-300",
                        !isPast && "bg-gray-200"
                      )} />
                    )}
                  </React.Fragment>
                )
              })}
            </div>
            
            {/* Main progress bar for non-scraping operations */}
            <div className="space-y-2">
              <div className="flex justify-between text-sm">
                <span className="text-muted-foreground">Overall Progress</span>
                <span className="font-medium">{visualProgress}%</span>
              </div>
              <Progress 
                value={visualProgress} 
                className="h-3"
              />
            </div>
            
            {/* Status message */}
            <div className={cn(
              "p-3 rounded-lg",
              isComplete && "bg-green-50 dark:bg-green-950",
              isFailed && "bg-red-50 dark:bg-red-950",
              !isComplete && !isFailed && "bg-blue-50 dark:bg-blue-950"
            )}>
              <p className={cn(
                "text-sm font-medium",
                isComplete && "text-green-800 dark:text-green-200",
                isFailed && "text-red-800 dark:text-red-200",
                !isComplete && !isFailed && "text-blue-800 dark:text-blue-200"
              )}>
                {statusMessage}
              </p>
              
              {/* File stats if available */}
              {stepMetadata && (stepMetadata.files_processed || stepMetadata.speed) && (
                <div className="flex items-center gap-4 mt-2 text-xs opacity-70">
                  {stepMetadata.files_processed && (
                    <span>Files: {stepMetadata.files_processed}</span>
                  )}
                  {stepMetadata.speed && (
                    <span>Speed: {stepMetadata.speed.toFixed(1)}/min</span>
                  )}
                </div>
              )}
            </div>
          </>
        )}
        
        {/* Completion actions */}
        {isComplete && onNextOperation && (
          <>
            <Separator />
            <div className="space-y-3">
              <p className="text-sm font-medium text-green-700 dark:text-green-300">
                ✨ Ready for next step!
              </p>
              <div className="grid grid-cols-1 sm:grid-cols-3 gap-2">
                {operationType === 'scraping' && (
                  <>
                    <Button
                      variant="default"
                      size="sm"
                      onClick={() => onNextOperation('processing')}
                      className="justify-start"
                    >
                      <Zap className="h-4 w-4 mr-2" />
                      Process Data
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => onNextOperation('indices')}
                      className="justify-start"
                    >
                      <FileSearch className="h-4 w-4 mr-2" />
                      Extract Indices
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => onNextOperation('full_pipeline')}
                      className="justify-start"
                    >
                      <ArrowRight className="h-4 w-4 mr-2" />
                      Run All
                    </Button>
                  </>
                )}
                {operationType === 'processing' && (
                  <>
                    <Button
                      variant="default"
                      size="sm"
                      onClick={() => onNextOperation('indices')}
                      className="justify-start"
                    >
                      <FileSearch className="h-4 w-4 mr-2" />
                      Extract Indices
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => onNextOperation('liquidity')}
                      className="justify-start"
                    >
                      <Zap className="h-4 w-4 mr-2" />
                      Liquidity Analysis
                    </Button>
                  </>
                )}
              </div>
            </div>
          </>
        )}
        
        {/* Error actions */}
        {isFailed && (
          <div className="flex items-center gap-2">
            <AlertCircle className="h-4 w-4 text-red-600" />
            <span className="text-sm text-red-600 font-medium">
              Operation failed - please check logs and try again
            </span>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

export default UnifiedOperationProgress