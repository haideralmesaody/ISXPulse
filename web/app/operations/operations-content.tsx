'use client'

import React, { useState, useEffect, useCallback, useRef } from 'react'
import { apiClient } from '@/lib/api'
import { useAllOperationUpdates } from '@/lib/hooks/use-websocket'
import { OperationRequestBuilder } from '@/lib/api/operation-request-builder'
import { Card, CardContent } from '@/components/ui/card'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { UnifiedOperationProgress } from '@/components/operations/UnifiedOperationProgress'
import { PipelineProgress } from '@/components/operations/PipelineProgress'
import { OperationCompleteCard } from '@/components/operations/OperationCompleteCard'
import { CompactOperationCard } from '@/components/operations/CompactOperationCard'
import { FloatingConfigPanel } from '@/components/operations/FloatingConfigPanel'
import { 
  AlertCircle,
  Loader2,
  Download,
  Zap,
  Database,
  FileSpreadsheet,
  BarChart3,
  Workflow,
  Info
} from 'lucide-react'

// Icon mapping for operation types
const operationIcons = {
  scraping: Download,
  processing: FileSpreadsheet,
  indices: BarChart3,
  liquidity: Zap,
  full_pipeline: Workflow,
  data_processing: Database,
} as const

export default function OperationsContent() {
  // WebSocket hook for real-time updates - use data directly
  const { operations, connected, error: wsError } = useAllOperationUpdates()
  
  // State - minimal and simple
  const [operationTypes, setOperationTypes] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [startingOperation, setStartingOperation] = useState<string | null>(null)
  const [isClient, setIsClient] = useState(false)
  const [selectedOperation, setSelectedOperation] = useState<any>(null)
  const [configPanelOpen, setConfigPanelOpen] = useState(false)
  const selectedCardRef = useRef<HTMLDivElement>(null)
  
  // Set client flag after mount
  useEffect(() => {
    setIsClient(true)
  }, [])
  
  // Fetch available operation types
  useEffect(() => {
    const fetchTypes = async () => {
      try {
        setLoading(true)
        const types = await apiClient.getOperationTypes()
        setOperationTypes(types)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch operation types')
      } finally {
        setLoading(false)
      }
    }
    
    fetchTypes()
  }, [])
  
  // Handle operation configuration
  const handleConfigureOperation = useCallback((type: any) => {
    setSelectedOperation(type)
    setConfigPanelOpen(true)
  }, [])
  
  // Handle direct start for operations that don't need configuration
  const handleDirectStart = useCallback(async (type: any) => {
    if (!isClient) return
    
    try {
      setStartingOperation(type.id)
      setError(null)
      
      // Build proper request structure using the request builder
      const params = OperationRequestBuilder.buildQuickStart(type.id)
      
      console.log('Quick Start operation with validated request:', type.id, params)
      
      await apiClient.createOperation(params)
      // WebSocket will automatically update the operations list
    } catch (err) {
      console.error('Failed to start operation:', err)
      setError(err instanceof Error ? err.message : 'Failed to start operation')
    } finally {
      setStartingOperation(null)
    }
  }, [isClient])

  // Start operation from floating panel
  const handleStartOperation = useCallback(async (params: any) => {
    if (!isClient || !selectedOperation) return
    
    try {
      setStartingOperation(selectedOperation.id)
      setError(null)
      
      // Log for debugging
      console.log('Starting operation with request:', params)
      
      await apiClient.createOperation(params)
      // WebSocket will automatically update the operations list
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start operation')
    } finally {
      setStartingOperation(null)
    }
  }, [isClient, selectedOperation])
  
  // Loading state
  if (loading) {
    return (
      <div className="min-h-screen p-8 flex items-center justify-center">
        <div className="text-center">
          <Loader2 className="h-8 w-8 animate-spin mx-auto mb-4" />
          <p className="text-muted-foreground">Loading operations...</p>
        </div>
      </div>
    )
  }
  
  return (
    <div className="min-h-screen p-8">
      <div className="max-w-7xl mx-auto space-y-8">
        {/* Header */}
        <div>
          <h1 className="text-3xl font-bold mb-2">Operations</h1>
          <p className="text-muted-foreground">
            Manage and monitor data processing operations
          </p>
        </div>
        
        {/* WebSocket Status */}
        <div className="flex items-center gap-2">
          <div className={`h-2 w-2 rounded-full ${connected ? 'bg-green-500' : 'bg-red-500'}`} />
          <span className="text-sm text-muted-foreground">
            {connected ? 'Connected' : 'Disconnected'}
          </span>
        </div>
        
        {/* Error Display */}
        {(error || wsError) && (
          <Alert variant="destructive">
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>{error || wsError}</AlertDescription>
          </Alert>
        )}
        
        {/* Operation Types - Compact Grid with Floating Configuration */}
        <div>
          <h2 className="text-xl font-semibold mb-4">Start New Operation</h2>
          <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-3">
            {operationTypes.map((type) => {
              const Icon = operationIcons[type.id as keyof typeof operationIcons] || Database
              const needsDates = type.id === 'scraping' || type.id === 'full_pipeline'
              
              return (
                <div key={type.id} ref={selectedOperation?.id === type.id ? selectedCardRef : null}>
                  <CompactOperationCard
                    type={{
                      ...type,
                      requiresDates: needsDates
                    }}
                    icon={Icon}
                    onConfigure={() => handleConfigureOperation(type)}
                    onDirectStart={() => handleDirectStart(type)}
                    isStarting={startingOperation === type.id}
                  />
                </div>
              )
            })}
          </div>
        </div>
        
        {/* Active and Completed Operations */}
        {operations && operations.length > 0 && (
          <div>
            <h2 className="text-xl font-semibold mb-4">Operations</h2>
            <div className="space-y-4">
              {operations.map((operation) => {
                // Show completion card for completed operations
                if (operation.status === 'completed') {
                  return (
                    <OperationCompleteCard
                      key={operation.operation_id}
                      operation={{
                        ...operation,
                        metadata: {
                          ...operation.metadata,
                          operation_type: operation.name?.toLowerCase().includes('pipeline') ? 'full_pipeline' :
                                         operation.name?.toLowerCase().includes('scrap') ? 'scraping' :
                                         operation.name?.toLowerCase().includes('process') ? 'processing' :
                                         operation.name?.toLowerCase().includes('indic') ? 'indices' : 'scraping'
                        }
                      }}
                      onNextOperation={(type) => {
                        const nextType = operationTypes.find(t => t.id === type)
                        if (nextType) {
                          handleConfigureOperation(nextType)
                        }
                      }}
                    />
                  )
                }
                
                // Use PipelineProgress for multi-step operations, UnifiedOperationProgress for single-step
                if (operation.steps && operation.steps.length > 1) {
                  return (
                    <PipelineProgress
                      key={operation.operation_id}
                      operation={operation}
                    />
                  )
                } else {
                  return (
                    <UnifiedOperationProgress
                      key={operation.operation_id}
                      operation={operation}
                      onNextOperation={(type) => {
                        const nextType = operationTypes.find(t => t.id === type)
                        if (nextType) {
                          handleConfigureOperation(nextType)
                        }
                      }}
                    />
                  )
                }
              })}
            </div>
          </div>
        )}
        
        {/* Empty State */}
        {(!operations || operations.length === 0) && !loading && (
          <Card className="p-8">
            <div className="text-center">
              <Info className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
              <h3 className="text-lg font-medium mb-2">No Active Operations</h3>
              <p className="text-muted-foreground">
                Start a new operation to begin processing data
              </p>
            </div>
          </Card>
        )}
      </div>
      
      {/* Floating Configuration Panel */}
      <FloatingConfigPanel
        isOpen={configPanelOpen}
        onClose={() => {
          setConfigPanelOpen(false)
          setSelectedOperation(null)
        }}
        onStart={handleStartOperation}
        operationType={selectedOperation}
        isStarting={startingOperation === selectedOperation?.id}
        anchorRef={selectedCardRef}
      />
    </div>
  )
}