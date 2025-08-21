/**
 * Operation Complete Card Component
 * Clear completion status with next step suggestions
 */

'use client'

import React from 'react'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { 
  CheckCircle2,
  ArrowRight,
  FileSearch,
  Zap,
  Database,
  BarChart3,
  FileText,
  Download,
  Sparkles
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { motion } from 'framer-motion'

interface OperationCompleteCardProps {
  operation: {
    operation_id: string
    name?: string
    metadata?: {
      files_processed?: number
      duration?: number
      operation_type?: string
    }
    completed_at?: string
  }
  onNextOperation?: (type: string) => void
  className?: string
}

const NEXT_OPERATIONS = {
  scraping: [
    {
      id: 'processing',
      label: 'Process Data',
      description: 'Transform Excel files to CSV',
      icon: Database,
      variant: 'default' as const,
      recommended: true
    },
    {
      id: 'indices',
      label: 'Extract Indices',
      description: 'Get ISX60 & ISX15 data',
      icon: BarChart3,
      variant: 'outline' as const
    },
    {
      id: 'liquidity',
      label: 'Liquidity Analysis',
      description: 'Calculate liquidity metrics',
      icon: Zap,
      variant: 'ghost' as const
    }
  ],
  processing: [
    {
      id: 'indices',
      label: 'Extract Indices',
      description: 'Get market indices',
      icon: BarChart3,
      variant: 'default' as const,
      recommended: true
    },
    {
      id: 'liquidity',
      label: 'Liquidity Analysis',
      description: 'Calculate liquidity scores',
      icon: FileText,
      variant: 'outline' as const
    }
  ],
  indices: [
    {
      id: 'liquidity',
      label: 'Liquidity Analysis',
      description: 'Generate liquidity metrics',
      icon: FileText,
      variant: 'default' as const,
      recommended: true
    },
    {
      id: 'export',
      label: 'Export Data',
      description: 'Download results',
      icon: Download,
      variant: 'outline' as const
    }
  ],
  full_pipeline: [] // No next operations for completed full pipeline
} as const

export function OperationCompleteCard({
  operation,
  onNextOperation,
  className
}: OperationCompleteCardProps) {
  
  // Determine operation type
  const operationType = operation.metadata?.operation_type || 
    (operation.name?.toLowerCase().includes('pipeline') ? 'full_pipeline' :
     operation.name?.toLowerCase().includes('scraping') ? 'scraping' :
     operation.name?.toLowerCase().includes('process') ? 'processing' :
     operation.name?.toLowerCase().includes('indic') ? 'indices' : 'scraping')
  
  const nextOps = NEXT_OPERATIONS[operationType as keyof typeof NEXT_OPERATIONS] || []
  const filesCount = operation.metadata?.files_processed || 0
  
  return (
    <motion.div
      initial={{ opacity: 0, y: 20, scale: 0.95 }}
      animate={{ opacity: 1, y: 0, scale: 1 }}
      transition={{ 
        type: "spring",
        stiffness: 200,
        damping: 20
      }}
    >
      <Card className={cn(
        "relative overflow-hidden",
        "bg-gradient-to-br from-green-50 to-emerald-50 dark:from-green-950 dark:to-emerald-950",
        "border-green-200 dark:border-green-800",
        className
      )}>
        {/* Success accent bar */}
        <div className="absolute top-0 left-0 right-0 h-1 bg-gradient-to-r from-green-500 via-emerald-500 to-green-500" />
        
        <CardHeader className="pb-4">
          <div className="flex items-start justify-between">
            <div className="flex items-center gap-3">
              <motion.div
                initial={{ scale: 0 }}
                animate={{ scale: 1 }}
                transition={{ 
                  type: "spring",
                  delay: 0.2
                }}
                className="p-2.5 rounded-full bg-green-100 dark:bg-green-900"
              >
                <CheckCircle2 className="h-6 w-6 text-green-700 dark:text-green-300" />
              </motion.div>
              <div>
                <h3 className="font-semibold text-lg text-green-900 dark:text-green-100">
                  Operation Complete!
                </h3>
                <p className="text-sm text-green-700 dark:text-green-300 mt-0.5">
                  {operation.name || 'Data Collection'}
                </p>
              </div>
            </div>
            
            <Badge 
              variant="outline" 
              className="bg-green-100 text-green-800 border-green-300 dark:bg-green-900 dark:text-green-100 dark:border-green-700"
            >
              <Sparkles className="h-3 w-3 mr-1" />
              Success
            </Badge>
          </div>
        </CardHeader>
        
        <CardContent className="space-y-4">
          {/* Success summary */}
          <div className="bg-white/60 dark:bg-gray-900/30 rounded-lg p-4 space-y-2">
            <div className="flex items-center justify-between text-sm">
              <span className="text-muted-foreground">Files Processed</span>
              <span className="font-semibold text-green-700 dark:text-green-300">
                {filesCount} files
              </span>
            </div>
            {operation.completed_at && (
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">Completed At</span>
                <span className="font-medium">
                  {new Date(operation.completed_at).toLocaleTimeString()}
                </span>
              </div>
            )}
            {operation.metadata?.duration && (
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">Duration</span>
                <span className="font-medium">
                  {Math.round(operation.metadata.duration / 1000)}s
                </span>
              </div>
            )}
          </div>
          
          {/* Next steps */}
          {onNextOperation && nextOps.length > 0 && (
            <div className="space-y-3">
              <div className="flex items-center gap-2">
                <ArrowRight className="h-4 w-4 text-green-600" />
                <p className="text-sm font-medium text-green-800 dark:text-green-200">
                  What would you like to do next?
                </p>
              </div>
              
              <div className="grid gap-2">
                {nextOps.map((op, index) => {
                  const Icon = op.icon
                  return (
                    <motion.div
                      key={op.id}
                      initial={{ opacity: 0, x: -20 }}
                      animate={{ opacity: 1, x: 0 }}
                      transition={{ delay: 0.3 + index * 0.1 }}
                    >
                      <Button
                        variant={op.variant}
                        className={cn(
                          "w-full justify-start h-auto py-3 px-4",
                          op.recommended && "ring-2 ring-green-500/30"
                        )}
                        onClick={() => onNextOperation(op.id)}
                      >
                        <Icon className="h-4 w-4 mr-3 flex-shrink-0" />
                        <div className="flex-1 text-left">
                          <div className="font-medium flex items-center gap-2">
                            {op.label}
                            {op.recommended && (
                              <Badge variant="secondary" className="text-xs px-1.5 py-0">
                                Recommended
                              </Badge>
                            )}
                          </div>
                          <div className="text-xs text-muted-foreground font-normal mt-0.5">
                            {op.description}
                          </div>
                        </div>
                        <ArrowRight className="h-4 w-4 ml-2 opacity-50" />
                      </Button>
                    </motion.div>
                  )
                })}
              </div>
            </div>
          )}
          
          {/* Alternative: Simple success message if no next operations */}
          {(!onNextOperation || nextOps.length === 0) && (
            <div className="text-center py-2">
              <p className="text-sm text-green-700 dark:text-green-300 font-medium">
                {operationType === 'full_pipeline' 
                  ? 'All pipeline stages completed successfully! Reports are ready for viewing.'
                  : 'All operations completed successfully!'}
              </p>
            </div>
          )}
        </CardContent>
      </Card>
    </motion.div>
  )
}

export default OperationCompleteCard