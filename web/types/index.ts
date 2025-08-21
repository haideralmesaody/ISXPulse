/**
 * Type definitions for ISX Daily Reports Scrapper
 * Central type definitions for the application
 */

// ============================================================================
// API Error Types (RFC 7807 Problem Details)
// ============================================================================

export interface ApiError {
  type: string
  title: string
  status: number
  detail?: string
  instance?: string
  trace_id?: string
  errors?: Record<string, string>
}

// ============================================================================
// License Types
// ============================================================================

export interface LicenseActivationRequest {
  license_key: string
  device_fingerprint?: DeviceFingerprint
  activation_id?: string
}

export interface LicenseResponse {
  status: 'valid' | 'invalid' | 'expired' | 'pending' | 'reactivated'
  message: string
  expiry_date?: string
  days_remaining?: number
  activation_id?: string
  device_info?: DeviceInfo
  features?: string[]
  reactivation_count?: number
  reactivation_limit?: number
  similarity_score?: number
  remaining_attempts?: number
}

export interface LicenseApiResponse {
  license_status?: 'active' | 'inactive' | 'expired' | 'invalid' | 'error' | 'not_activated' | 'warning' | 'critical' | 'reactivated'
  status: 'valid' | 'invalid' | 'expired' | 'pending' | 'reactivated'
  message: string
  expiry_date?: string
  days_remaining?: number
  days_left?: number
  activation_id?: string
  device_info?: DeviceInfo
  features?: string[]
  last_check?: string
  license_info?: {
    expiry_date?: string
    activation_date?: string
    device_info?: DeviceInfo
  }
  reactivation_count?: number
  reactivation_limit?: number
  similarity_score?: number
  remaining_attempts?: number
  trace_id?: string
  timestamp?: string
}

// ============================================================================
// Scratch Card Types
// ============================================================================

export interface ScratchCardData {
  code: string
  format: 'standard' | 'scratch' // ISX1M02LYE1F9QJHR9D7Z vs ISX-XXXX-XXXX-XXXX
  revealed: boolean
  activationId?: string
}

export interface ScratchCardActivationRequest {
  scratch_code: string
  device_fingerprint: DeviceFingerprint
  activation_id: string
}

export interface ScratchCardActivationResponse {
  success: boolean
  license_key: string
  status: 'activated' | 'already_used' | 'invalid' | 'expired'
  activation_id: string
  device_info: DeviceInfo
  expiry_date: string
  features: string[]
  message: string
}

// ============================================================================
// Device Fingerprint Types
// ============================================================================

export interface DeviceFingerprint {
  browser: string
  browserVersion: string
  os: string
  osVersion: string
  platform: string
  screenResolution: string
  timezone: string
  language: string
  userAgent: string
  hash: string
  timestamp: string
}

export interface DeviceInfo {
  fingerprint: string
  browser: string
  os: string
  platform: string
  first_activation: string
  last_seen: string
  trusted: boolean
}

// ============================================================================
// License Status Types
// ============================================================================

export interface LicenseStatus {
  isActive: boolean
  daysRemaining: number
  expiryDate: string
  status: 'active' | 'warning' | 'critical' | 'expired' | 'invalid'
  activationHistory: LicenseActivationHistory[]
  deviceInfo: DeviceInfo
  features: string[]
}

export interface LicenseActivationHistory {
  id: string
  date: string
  action: 'activated' | 'renewed' | 'transferred' | 'deactivated'
  device: string
  ip_address?: string
  success: boolean
  message?: string
}

// ============================================================================
// Operation Types
// ============================================================================

export interface Operation {
  id: string
  type: string
  name: string
  description: string
  status: 'idle' | 'running' | 'completed' | 'failed' | 'cancelled'
  progress: number
  created_at: string
  updated_at: string
  started_at?: string
  completed_at?: string
  config: OperationConfig
  stages: OperationStage[]
  metadata?: Record<string, any>
  results?: OperationResult[]
  error?: string
}

export interface OperationConfig {
  auto_start: boolean
  retry_attempts: number
  timeout_seconds: number
  steps: string[]
  notification_email?: string
  parameters?: Record<string, any>
}

export interface OperationStage {
  id: string
  name: string
  description: string
  status: 'pending' | 'running' | 'completed' | 'failed' | 'skipped'
  progress: number
  started_at?: string
  completed_at?: string
  duration?: number
  error?: string
  results?: Record<string, any>
}

export interface OperationResult {
  stage_id: string
  stage_name: string
  success: boolean
  data?: any
  error?: string
  timestamp: string
}

export interface OperationTypeDefinition {
  id: string
  name: string
  description: string
  category: string
  config_schema: Record<string, any>
  required_features?: string[]
  estimated_duration?: string
}

export interface CreateOperationRequest {
  type: string
  name?: string
  config: Partial<OperationConfig>
  auto_start?: boolean
}

export interface CreateOperationResponse {
  operation: Operation
  job_id?: string
  websocket_url?: string
}

// ============================================================================
// Job Types
// ============================================================================

export interface JobStatus {
  id: string
  operation_id: string
  stage_id?: string
  status: 'queued' | 'running' | 'completed' | 'failed' | 'cancelled'
  progress: number
  created_at: string
  started_at?: string
  completed_at?: string
  error?: string
  result?: any
}

export interface JobListResponse {
  jobs: JobStatus[]
  total: number
  page: number
  page_size: number
}

// ============================================================================
// Market Data Types
// ============================================================================

export interface Ticker {
  symbol: string
  company_name: string
  last_price: number
  change: number
  change_percent: number
  volume: number
  value: number
  high: number
  low: number
  open: number
  trades: number
  last_update: string
}

export interface Report {
  id: string
  ticker: string
  type: 'daily' | 'summary' | 'liquidity'
  date: string
  file_path: string
  file_size: number
  created_at: string
  metadata?: Record<string, any>
}

export interface MarketSummary {
  date: string
  total_volume: number
  total_value: number
  total_trades: number
  advancing: number
  declining: number
  unchanged: number
  top_gainers: Ticker[]
  top_losers: Ticker[]
  most_active: Ticker[]
  indices: MarketIndex[]
}

export interface MarketIndex {
  name: string
  value: number
  change: number
  change_percent: number
  date: string
}

// ============================================================================
// UI State Types
// ============================================================================

export interface LoadingState {
  isLoading: boolean
  message?: string
  progress?: number
}

export interface ErrorState {
  hasError: boolean
  error?: ApiError | Error | string
  timestamp?: string
}

export interface PaginationState {
  page: number
  pageSize: number
  total: number
  totalPages: number
}

export interface SortState {
  column: string
  direction: 'asc' | 'desc'
}

export interface FilterState {
  [key: string]: any
}

// ============================================================================
// WebSocket Types
// ============================================================================

export interface WebSocketMessage {
  type: string
  data: any
  timestamp: string
  id?: string
}

export interface OperationStatusMessage extends WebSocketMessage {
  type: 'operation_status'
  data: {
    operation_id: string
    status: Operation['status']
    progress: number
    stage?: OperationStage
    error?: string
  }
}

export interface LicenseStatusMessage extends WebSocketMessage {
  type: 'license_status'
  data: {
    status: LicenseApiResponse['status']
    days_remaining?: number
    expiry_date?: string
    message?: string
  }
}

// ============================================================================
// Chart and Analysis Types
// ============================================================================

export interface ChartDataPoint {
  x: number
  y: number
  [key: string]: any
}

export interface TimeSeriesData {
  timestamp: number
  value: number
  volume?: number
  open?: number
  high?: number
  low?: number
  close?: number
}

// ============================================================================
// Utility Types
// ============================================================================

export type Status = 'idle' | 'loading' | 'success' | 'error'

export type Theme = 'light' | 'dark' | 'system'

export type ToastType = 'success' | 'error' | 'warning' | 'info'

export interface ToastMessage {
  id: string
  type: ToastType
  title: string
  description?: string
  duration?: number
  action?: {
    label: string
    onClick: () => void
  }
}

// ============================================================================
// Form Types
// ============================================================================

export interface FormField {
  name: string
  label: string
  type: 'text' | 'email' | 'password' | 'number' | 'select' | 'checkbox' | 'textarea'
  placeholder?: string
  required?: boolean
  options?: { label: string; value: string }[]
  validation?: any
}

export interface FormState {
  values: Record<string, any>
  errors: Record<string, string>
  touched: Record<string, boolean>
  isSubmitting: boolean
  isDirty: boolean
  isValid: boolean
}

// ============================================================================
// Component Props Types
// ============================================================================

export interface BaseComponentProps {
  className?: string
  children?: React.ReactNode
}

export interface PageProps {
  params?: Record<string, string>
  searchParams?: Record<string, string>
}

// ============================================================================
// API Response Wrappers
// ============================================================================

export interface ApiResponse<T = any> {
  data: T
  message?: string
  timestamp: string
  request_id?: string
}

export interface PaginatedResponse<T = any> {
  data: T[]
  pagination: {
    page: number
    page_size: number
    total: number
    total_pages: number
  }
  message?: string
  timestamp: string
}

// ============================================================================
// Configuration Types
// ============================================================================

export interface AppConfig {
  api_base_url: string
  websocket_url: string
  version: string
  features: string[]
  debug: boolean
}

export interface UserPreferences {
  theme: Theme
  language: string
  timezone: string
  notifications: {
    email: boolean
    browser: boolean
    sounds: boolean
  }
  dashboard: {
    auto_refresh: boolean
    refresh_interval: number
    default_view: string
  }
}

// ============================================================================
// Export All Types
// ============================================================================

// Re-export reports types
export * from './reports'

export type {
  // Re-export common types for convenience
  React,
} from 'react'