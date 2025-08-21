/**
 * Floating Configuration Panel for Operations
 * Provides spacious, focused configuration experience
 */

'use client'

import React, { useState, useEffect, useRef } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { Card } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import { 
  X, 
  Calendar, 
  Play, 
  Clock,
  ChevronRight,
  Sparkles,
  CalendarDays,
  AlertCircle,
  RefreshCw
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { OPERATION_DATE_DEFAULTS, DATE_VALIDATION } from '@/lib/constants'
import { validateAndUpdateDates, formatDateString } from '@/lib/date-utils'

interface FloatingConfigPanelProps {
  isOpen: boolean
  onClose: () => void
  onStart: (params: any) => void
  operationType: {
    id: string
    name: string
    description: string
  } | null
  isStarting: boolean
  anchorRef?: React.RefObject<HTMLElement>
}

// Date range presets
const DATE_PRESETS = [
  { 
    label: 'Today', 
    icon: Clock,
    getValue: () => ({ 
      from: new Date().toISOString().split('T')[0], 
      to: new Date().toISOString().split('T')[0] 
    }) 
  },
  { 
    label: 'Last 7 days', 
    icon: CalendarDays,
    getValue: () => {
      const to = new Date()
      const from = new Date()
      from.setDate(from.getDate() - 7)
      return { 
        from: from.toISOString().split('T')[0], 
        to: to.toISOString().split('T')[0] 
      }
    }
  },
  { 
    label: 'Last 30 days',
    icon: Calendar, 
    getValue: () => {
      const to = new Date()
      const from = new Date()
      from.setDate(from.getDate() - 30)
      return { 
        from: from.toISOString().split('T')[0], 
        to: to.toISOString().split('T')[0] 
      }
    }
  },
  { 
    label: 'This month',
    icon: Sparkles,
    getValue: () => {
      const now = new Date()
      const from = new Date(now.getFullYear(), now.getMonth(), 1)
      const to = new Date(now.getFullYear(), now.getMonth() + 1, 0)
      return { 
        from: from.toISOString().split('T')[0], 
        to: to.toISOString().split('T')[0] 
      }
    }
  }
]

export function FloatingConfigPanel({
  isOpen,
  onClose,
  onStart,
  operationType,
  isStarting,
  anchorRef
}: FloatingConfigPanelProps) {
  // Initialize with default dates from Single Source of Truth
  const [fromDate, setFromDate] = useState<string>(OPERATION_DATE_DEFAULTS.getFromDateString())
  const [toDate, setToDate] = useState<string>(OPERATION_DATE_DEFAULTS.getToDateString())
  const [selectedPreset, setSelectedPreset] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [dateWasUpdated, setDateWasUpdated] = useState(false)
  const panelRef = useRef<HTMLDivElement>(null)

  // Check if this operation requires dates
  const requiresDates = operationType && 
    (operationType.id === 'scraping' || operationType.id === 'full_pipeline')

  // Set default dates when panel opens with smart date handling
  useEffect(() => {
    if (isOpen && operationType) {
      setDateWasUpdated(false) // Reset the indicator
      // Load saved dates or use defaults
      const savedDates = localStorage.getItem(`operation_dates_${operationType.id}`)
      if (savedDates) {
        try {
          const parsed = JSON.parse(savedDates)
          // Use smart date validation to update old dates
          const validatedDates = validateAndUpdateDates({
            from: parsed.from || OPERATION_DATE_DEFAULTS.getFromDateString(),
            to: parsed.to || OPERATION_DATE_DEFAULTS.getToDateString()
          })
          
          // Check if the "to" date was updated to today
          const originalToDate = new Date(parsed.to)
          const updatedToDate = validatedDates.to
          const today = new Date()
          today.setHours(0, 0, 0, 0)
          
          if (originalToDate < today && updatedToDate.getTime() === today.getTime()) {
            setDateWasUpdated(true)
          }
          
          setFromDate(formatDateString(validatedDates.from))
          setToDate(formatDateString(validatedDates.to))
        } catch {
          setDefaultDates()
        }
      } else {
        setDefaultDates()
      }
      setError(null)
      setSelectedPreset(null)
    }
  }, [isOpen, operationType])

  // Set default dates from Single Source of Truth
  const setDefaultDates = () => {
    setFromDate(OPERATION_DATE_DEFAULTS.getFromDateString())
    setToDate(OPERATION_DATE_DEFAULTS.getToDateString())
  }

  // Handle clicks outside panel
  useEffect(() => {
    if (!isOpen) return

    const handleClickOutside = (event: MouseEvent) => {
      if (panelRef.current && !panelRef.current.contains(event.target as Node)) {
        onClose()
      }
    }

    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose()
      }
    }

    document.addEventListener('mousedown', handleClickOutside)
    document.addEventListener('keydown', handleEscape)

    return () => {
      document.removeEventListener('mousedown', handleClickOutside)
      document.removeEventListener('keydown', handleEscape)
    }
  }, [isOpen, onClose])

  // Handle preset selection
  const handlePresetSelect = (preset: typeof DATE_PRESETS[0]) => {
    const dates = preset.getValue()
    setFromDate(dates.from)
    setToDate(dates.to)
    setSelectedPreset(preset.label)
    setError(null)
  }

  // Validate dates
  const validateDates = (): boolean => {
    if (!requiresDates) return true

    if (!fromDate || !toDate) {
      setError('Please select both from and to dates')
      return false
    }

    const from = new Date(fromDate)
    const to = new Date(toDate)

    if (from > to) {
      setError('From date must be before or equal to to date')
      return false
    }

    const daysDiff = Math.ceil((to.getTime() - from.getTime()) / (1000 * 60 * 60 * 24))
    if (daysDiff > 365) {
      setError('Date range cannot exceed 365 days')
      return false
    }

    return true
  }

  // Handle start operation
  const handleStart = () => {
    if (!operationType) return

    if (!validateDates()) return

    // Save dates for next time
    if (requiresDates) {
      localStorage.setItem(`operation_dates_${operationType.id}`, JSON.stringify({
        from: fromDate,
        to: toDate
      }))
    }

    // Build parameters
    const params: any = {
      mode: 'full',
      steps: [{
        id: operationType.id,
        type: operationType.name,
        parameters: {}
      }]
    }

    // Add dates if required
    if (requiresDates) {
      params.steps[0].parameters.from = fromDate
      params.steps[0].parameters.to = toDate
    }

    onStart(params)
    onClose()
  }

  // Calculate days in range
  const calculateDays = () => {
    if (!fromDate || !toDate) return 0
    const from = new Date(fromDate)
    const to = new Date(toDate)
    return Math.ceil((to.getTime() - from.getTime()) / (1000 * 60 * 60 * 24)) + 1
  }

  const daysInRange = calculateDays()

  if (!operationType) return null

  return (
    <AnimatePresence>
      {isOpen && (
        <>
          {/* Backdrop */}
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.2 }}
            className="fixed inset-0 bg-black/20 backdrop-blur-sm z-40"
          />

          {/* Panel */}
          <motion.div
            ref={panelRef}
            initial={{ opacity: 0, y: -20, scale: 0.95 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: -20, scale: 0.95 }}
            transition={{ 
              type: "spring",
              stiffness: 300,
              damping: 25
            }}
            className="fixed left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 z-50 w-full max-w-3xl"
          >
            <Card className="shadow-2xl border-2 overflow-hidden">
              {/* Header */}
              <div className="bg-gradient-to-r from-primary/10 to-primary/5 border-b px-6 py-4">
                <div className="flex items-center justify-between">
                  <div>
                    <h2 className="text-xl font-semibold flex items-center gap-2">
                      <Calendar className="h-5 w-5 text-primary" />
                      Configure {operationType.name}
                    </h2>
                    <p className="text-sm text-muted-foreground mt-1">
                      {operationType.description}
                    </p>
                  </div>
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={onClose}
                    className="rounded-full hover:bg-white/50"
                  >
                    <X className="h-4 w-4" />
                  </Button>
                </div>
              </div>

              {/* Content */}
              <div className="p-6">
                {/* Auto-update notification */}
                {dateWasUpdated && (
                  <div className="mb-4 p-3 bg-blue-50 dark:bg-blue-950 border border-blue-200 dark:border-blue-800 rounded-lg flex items-center gap-2">
                    <RefreshCw className="h-4 w-4 text-blue-600 dark:text-blue-400" />
                    <span className="text-sm text-blue-800 dark:text-blue-200">
                      The "To Date" has been automatically updated to today for current data
                    </span>
                  </div>
                )}
                
                {requiresDates ? (
                  <div className="grid md:grid-cols-2 gap-6">
                    {/* Left: Quick Presets */}
                    <div className="space-y-4">
                      <Label className="text-sm font-medium">Quick Selection</Label>
                      <div className="grid gap-2">
                        {DATE_PRESETS.map((preset) => {
                          const Icon = preset.icon
                          return (
                            <Button
                              key={preset.label}
                              variant={selectedPreset === preset.label ? "default" : "outline"}
                              className={cn(
                                "justify-start h-auto py-3 px-4",
                                "hover:scale-[1.02] transition-transform"
                              )}
                              onClick={() => handlePresetSelect(preset)}
                            >
                              <Icon className="h-4 w-4 mr-3 opacity-70" />
                              <span className="font-medium">{preset.label}</span>
                              {selectedPreset === preset.label && (
                                <Badge variant="secondary" className="ml-auto">
                                  Selected
                                </Badge>
                              )}
                            </Button>
                          )
                        })}
                      </div>
                    </div>

                    {/* Right: Custom Date Range */}
                    <div className="space-y-4">
                      <Label className="text-sm font-medium">Custom Date Range</Label>
                      
                      <div className="space-y-3">
                        <div>
                          <Label htmlFor="from-date" className="text-xs text-muted-foreground mb-1.5 block">
                            From Date
                          </Label>
                          <Input
                            id="from-date"
                            type="date"
                            value={fromDate}
                            onChange={(e) => {
                              setFromDate(e.target.value)
                              setSelectedPreset(null)
                              setError(null)
                            }}
                            max={new Date().toISOString().split('T')[0]}
                            min="2020-01-01"
                            className="h-11"
                          />
                        </div>

                        <div>
                          <Label htmlFor="to-date" className="text-xs text-muted-foreground mb-1.5 block">
                            To Date
                          </Label>
                          <Input
                            id="to-date"
                            type="date"
                            value={toDate}
                            onChange={(e) => {
                              setToDate(e.target.value)
                              setSelectedPreset(null)
                              setError(null)
                            }}
                            max={new Date().toISOString().split('T')[0]}
                            min={fromDate || "2020-01-01"}
                            className="h-11"
                          />
                        </div>
                      </div>

                      {/* Date Range Preview */}
                      {fromDate && toDate && !error && (
                        <div className="bg-primary/5 rounded-lg p-4 border border-primary/20">
                          <div className="flex items-center justify-between mb-2">
                            <span className="text-sm font-medium">Selected Range</span>
                            <Badge variant="outline" className="font-mono">
                              {daysInRange} {daysInRange === 1 ? 'day' : 'days'}
                            </Badge>
                          </div>
                          <div className="text-xs text-muted-foreground space-y-1">
                            <div className="flex items-center gap-2">
                              <span className="font-medium">From:</span>
                              <span>{new Date(fromDate).toLocaleDateString('en-US', { 
                                weekday: 'short', 
                                year: 'numeric', 
                                month: 'short', 
                                day: 'numeric' 
                              })}</span>
                            </div>
                            <div className="flex items-center gap-2">
                              <span className="font-medium">To:</span>
                              <span>{new Date(toDate).toLocaleDateString('en-US', { 
                                weekday: 'short', 
                                year: 'numeric', 
                                month: 'short', 
                                day: 'numeric' 
                              })}</span>
                            </div>
                          </div>
                        </div>
                      )}
                    </div>
                  </div>
                ) : (
                  <div className="text-center py-8">
                    <div className="inline-flex items-center justify-center w-12 h-12 rounded-full bg-primary/10 mb-4">
                      <Sparkles className="h-6 w-6 text-primary" />
                    </div>
                    <p className="text-sm text-muted-foreground">
                      This operation will process existing data files.
                      <br />
                      No additional configuration is required.
                    </p>
                  </div>
                )}

                {/* Error Display */}
                {error && (
                  <div className="mt-4 p-3 bg-destructive/10 border border-destructive/20 rounded-lg flex items-center gap-2">
                    <AlertCircle className="h-4 w-4 text-destructive" />
                    <span className="text-sm text-destructive">{error}</span>
                  </div>
                )}
              </div>

              {/* Footer */}
              <Separator />
              <div className="px-6 py-4 bg-muted/30 flex items-center justify-between">
                <Button
                  variant="ghost"
                  onClick={onClose}
                  disabled={isStarting}
                >
                  Cancel
                </Button>
                <Button
                  onClick={handleStart}
                  disabled={isStarting || (requiresDates && (!fromDate || !toDate))}
                  className="min-w-[140px]"
                >
                  {isStarting ? (
                    <>
                      <span className="animate-spin mr-2">⏳</span>
                      Starting...
                    </>
                  ) : (
                    <>
                      <Play className="h-4 w-4 mr-2" />
                      Start Operation
                    </>
                  )}
                </Button>
              </div>
            </Card>
          </motion.div>
        </>
      )}
    </AnimatePresence>
  )
}

export default FloatingConfigPanel