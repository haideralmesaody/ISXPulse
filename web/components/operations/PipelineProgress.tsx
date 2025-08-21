/**
 * Pipeline Progress Component
 * Stacks existing UnifiedOperationProgress components vertically for pipeline view
 */

'use client'

import React from 'react'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { UnifiedOperationProgress } from './UnifiedOperationProgress'
import { cn } from '@/lib/utils'

interface PipelineProgressProps {
  operation: {
    operation_id: string
    name?: string
    status: string
    progress: number
    message?: string
    error?: string
    metadata?: any
    steps?: Array<{
      id: string
      name: string
      status: string
      progress: number
      message?: string
      error?: string
      metadata?: any
    }>
  }
}

export function PipelineProgress({ operation }: PipelineProgressProps) {
  const { steps = [] } = operation
  
  // Calculate overall stats
  const completedCount = steps.filter(s => s.status === 'completed').length
  const runningStep = steps.find(s => s.status === 'running')
  const failedStep = steps.find(s => s.status === 'failed')
  
  // Determine overall status
  const overallStatus = failedStep ? 'failed' : 
                       completedCount === steps.length ? 'completed' :
                       runningStep ? 'running' : 'pending'
  
  return (
    <Card className={cn(
      "w-full overflow-hidden transition-all duration-300",
      overallStatus === 'completed' && "ring-2 ring-green-500/20",
      overallStatus === 'failed' && "ring-2 ring-red-500/20"
    )}>
      {/* Status bar at top */}
      <div className={cn(
        "h-1 w-full",
        overallStatus === 'completed' && "bg-gradient-to-r from-green-500 to-green-400",
        overallStatus === 'failed' && "bg-gradient-to-r from-red-500 to-red-400",
        overallStatus === 'running' && "bg-gradient-to-r from-blue-500 to-blue-400",
        overallStatus === 'pending' && "bg-gradient-to-r from-gray-400 to-gray-300"
      )} />
      
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-lg font-semibold">
              Full Pipeline Operation
            </h3>
            <p className="text-sm text-muted-foreground">
              {completedCount} of {steps.length} stages complete
            </p>
          </div>
          <Badge 
            variant={overallStatus === 'completed' ? "success" : 
                    overallStatus === 'failed' ? "destructive" : 
                    overallStatus === 'running' ? "default" : "secondary"}
            className="font-medium"
          >
            {overallStatus}
          </Badge>
        </div>
      </CardHeader>
      
      <CardContent className="space-y-4">
        {steps.map((step, index) => {
          // Wrap each step as its own operation for UnifiedOperationProgress
          const stepAsOperation = {
            operation_id: `${operation.operation_id}-${step.id}`,
            name: step.name,
            status: step.status,
            progress: step.progress,
            message: step.message,
            error: step.error,
            metadata: step.metadata,
            // Pass the step as a single-item array so UnifiedOperationProgress works correctly
            steps: [{
              ...step,
              metadata: step.metadata
            }]
          }
          
          return (
            <div key={step.id} className="relative">
              {/* Visual connector between stages */}
              {index < steps.length - 1 && (
                <div className={cn(
                  "absolute left-8 top-full h-4 w-0.5 z-10",
                  step.status === 'completed' ? "bg-green-500" :
                  step.status === 'failed' ? "bg-red-500" :
                  step.status === 'running' ? "bg-blue-500" :
                  "bg-gray-300"
                )} />
              )}
              
              {/* Stage number indicator */}
              <div className="absolute -left-2 top-6 flex items-center justify-center">
                <div className={cn(
                  "w-6 h-6 rounded-full text-xs font-semibold flex items-center justify-center",
                  step.status === 'completed' && "bg-green-100 text-green-700",
                  step.status === 'failed' && "bg-red-100 text-red-700",
                  step.status === 'running' && "bg-blue-100 text-blue-700",
                  step.status === 'pending' && "bg-gray-100 text-gray-500",
                  step.status === 'skipped' && "bg-yellow-100 text-yellow-700"
                )}>
                  {index + 1}
                </div>
              </div>
              
              {/* Use existing UnifiedOperationProgress for each stage */}
              <div className="ml-6">
                <UnifiedOperationProgress
                  operation={stepAsOperation}
                  onNextOperation={undefined} // No next actions in pipeline view
                />
              </div>
            </div>
          )
        })}
        
        {/* Overall completion message */}
        {overallStatus === 'completed' && (
          <div className="mt-4 p-3 bg-green-50 dark:bg-green-950 rounded-lg">
            <p className="text-sm font-medium text-green-800 dark:text-green-200">
              ✅ All pipeline stages completed successfully!
            </p>
          </div>
        )}
        
        {overallStatus === 'failed' && failedStep && (
          <div className="mt-4 p-3 bg-red-50 dark:bg-red-950 rounded-lg">
            <p className="text-sm font-medium text-red-800 dark:text-red-200">
              ❌ Pipeline failed at stage: {failedStep.name}
            </p>
            {failedStep.error && (
              <p className="text-xs mt-1 text-red-700 dark:text-red-300">
                {failedStep.error}
              </p>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

export default PipelineProgress