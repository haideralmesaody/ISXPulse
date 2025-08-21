/**
 * Operation Request Builder
 * Centralized request construction for ISX operations
 * Ensures proper structure matching backend expectations
 */

import { z } from 'zod'

// Schema matching backend expectations exactly
export const OperationRequestSchema = z.object({
  mode: z.enum(['full', 'partial']),
  steps: z.array(z.object({
    id: z.string(),
    type: z.string(),
    parameters: z.record(z.any())
  })).min(1, "At least one step is required"),
  parameters: z.record(z.any()).optional()
})

export type OperationRequest = z.infer<typeof OperationRequestSchema>

/**
 * Builder class for creating properly structured operation requests
 * Handles both single-step and multi-step operations
 */
export class OperationRequestBuilder {
  
  /**
   * Build request for immediate execution operations (NO POPUP)
   * Used for: processing, indices, liquidity
   */
  static buildQuickStart(stepId: string): OperationRequest {
    const request = {
      mode: 'full' as const,
      steps: [{
        id: stepId,
        type: stepId,
        parameters: {}
      }],
      parameters: {
        step: stepId  // Backend compatibility - manager.go checks this
      }
    }
    
    // Validate before returning
    return OperationRequestSchema.parse(request)
  }
  
  /**
   * Build request for operations requiring dates (WITH POPUP)
   * Used for: scraping
   */
  static buildWithDates(stepId: string, fromDate: string, toDate: string): OperationRequest {
    const request = {
      mode: 'full' as const,
      steps: [{
        id: stepId,
        type: stepId,
        parameters: {
          from: fromDate,
          to: toDate
        }
      }],
      parameters: {
        step: stepId,
        from: fromDate,
        to: toDate
      }
    }
    
    return OperationRequestSchema.parse(request)
  }
  
  /**
   * Build request for full pipeline (multiple steps)
   * Runs all operations in sequence
   */
  static buildFullPipeline(fromDate: string, toDate: string): OperationRequest {
    const request = {
      mode: 'full' as const,
      steps: [
        { 
          id: 'scraping', 
          type: 'scraping', 
          parameters: { from: fromDate, to: toDate }
        },
        { 
          id: 'processing', 
          type: 'processing', 
          parameters: {}
        },
        { 
          id: 'indices', 
          type: 'indices', 
          parameters: {}
        },
        { 
          id: 'liquidity', 
          type: 'liquidity', 
          parameters: {}
        }
      ],
      parameters: {
        step: 'full_pipeline',
        from: fromDate,
        to: toDate
      }
    }
    
    return OperationRequestSchema.parse(request)
  }
  
  /**
   * Validate a request structure without building
   */
  static validate(request: unknown): OperationRequest {
    return OperationRequestSchema.parse(request)
  }
  
  /**
   * Check if a request is valid
   */
  static isValid(request: unknown): boolean {
    try {
      OperationRequestSchema.parse(request)
      return true
    } catch {
      return false
    }
  }
}

/**
 * Operation configuration defining which operations need user input
 */
export const OPERATIONS_CONFIG = {
  scraping: {
    requiresDates: true,
    quickStart: false,
    name: "Data Collection",
    description: "Download ISX daily reports for specified date range",
    icon: "Download"
  },
  processing: {
    requiresDates: false,
    quickStart: true,
    name: "Data Processing",
    description: "Convert Excel files to CSV format",
    icon: "FileSpreadsheet"
  },
  indices: {
    requiresDates: false,
    quickStart: true,
    name: "Index Extraction",
    description: "Extract ISX60 and ISX15 indices",
    icon: "BarChart3"
  },
  liquidity: {
    requiresDates: false,
    quickStart: true,
    name: "Liquidity Analysis",
    description: "Calculate ISX Hybrid Liquidity Metrics and scoring",
    icon: "Droplets"
  },
  full_pipeline: {
    requiresDates: true,
    quickStart: false,
    name: "Full Pipeline",
    description: "Run complete data processing workflow",
    icon: "Workflow"
  }
} as const

export type OperationType = keyof typeof OPERATIONS_CONFIG