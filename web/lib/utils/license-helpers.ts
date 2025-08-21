/**
 * License utility functions for ISX Pulse
 * Provides formatting, validation, and rate limiting for license operations
 */

/**
 * Clean license key for submission (remove spaces, uppercase)
 * Keep dashes for scratch card format
 */
export function cleanLicenseKey(value: string): string {
  const trimmed = value.trim().toUpperCase()
  // If it has dashes in the right pattern, keep them (scratch card format)
  if (trimmed.match(/^ISX-[A-Z0-9]{4}-[A-Z0-9]{4}-[A-Z0-9]{4}-[A-Z0-9]{4}$/)) {
    return trimmed
  }
  // Otherwise remove spaces and dashes for standard format
  return trimmed.replace(/[\s-]/g, '')
}

/**
 * Auto-format license key as user types
 * Supports both standard (ISX1M02LYE1F9QJHR9D7Z) and scratch card (ISX-XXXX-XXXX-XXXX-XXXX) formats
 */
export function formatLicenseKey(value: string, format: 'standard' | 'scratch' = 'standard'): string {
  const cleaned = value.trim().toUpperCase().replace(/[\s]/g, '') // Only remove spaces, not dashes
  
  if (format === 'scratch') {
    // Remove all non-alphanumeric for formatting
    const alphanum = cleaned.replace(/[^A-Z0-9]/g, '')
    
    // Format as ISX-XXXX-XXXX-XXXX-XXXX (20 chars total with dashes, 16 without)
    if (alphanum.length <= 3) return alphanum
    if (alphanum.length <= 7) return `${alphanum.slice(0, 3)}-${alphanum.slice(3)}`
    if (alphanum.length <= 11) return `${alphanum.slice(0, 3)}-${alphanum.slice(3, 7)}-${alphanum.slice(7)}`
    if (alphanum.length <= 15) return `${alphanum.slice(0, 3)}-${alphanum.slice(3, 7)}-${alphanum.slice(7, 11)}-${alphanum.slice(11)}`
    return `${alphanum.slice(0, 3)}-${alphanum.slice(3, 7)}-${alphanum.slice(7, 11)}-${alphanum.slice(11, 15)}-${alphanum.slice(15, 19)}`
  }
  
  // Standard format - remove all dashes
  return cleaned.replace(/-/g, '')
}

/**
 * Detect license key format based on content
 */
export function detectLicenseFormat(key: string): 'standard' | 'scratch' {
  const upperKey = key.toUpperCase()
  
  // Check if it matches scratch card pattern (with or without complete dashes)
  if (upperKey.includes('-') || (upperKey.startsWith('ISX') && !upperKey.match(/^ISX[136]M/))) {
    return 'scratch'
  }
  
  // Check for standard format prefixes
  if (upperKey.match(/^ISX[136]M/) || upperKey.match(/^ISX1Y/)) {
    return 'standard'
  }
  
  // Default based on length (scratch cards are 16 chars without dashes, standard are 15+)
  const cleaned = upperKey.replace(/[^A-Z0-9]/g, '')
  if (cleaned.length === 19 && cleaned.startsWith('ISX')) {
    return 'scratch'
  }
  
  return 'standard'
}

/**
 * Validate license key format (supports both standard and scratch card formats)
 */
export function isValidLicenseFormat(key: string): boolean {
  const upperKey = key.trim().toUpperCase()
  
  if (!upperKey.startsWith('ISX')) return false
  
  // Check scratch card format with dashes: ISX-XXXX-XXXX-XXXX-XXXX
  if (upperKey.match(/^ISX-[A-Z0-9]{4}-[A-Z0-9]{4}-[A-Z0-9]{4}-[A-Z0-9]{4}$/)) {
    return true
  }
  
  // Check scratch card format without dashes (19 chars total)
  const noDashes = upperKey.replace(/-/g, '')
  if (noDashes.match(/^ISX[A-Z0-9]{16}$/)) {
    return true
  }
  
  // Check standard format: ISX1M/3M/6M/1Y followed by alphanumeric
  if (noDashes.match(/^ISX(1M|3M|6M|1Y)[A-Z0-9]{5,}$/)) {
    return true
  }
  
  return false
}

/**
 * Validate scratch card specific format
 */
export function isValidScratchCardFormat(key: string): boolean {
  const upperKey = key.trim().toUpperCase()
  
  // Match exact scratch card format: ISX-XXXX-XXXX-XXXX-XXXX or without dashes
  if (upperKey.match(/^ISX-[A-Z0-9]{4}-[A-Z0-9]{4}-[A-Z0-9]{4}-[A-Z0-9]{4}$/)) {
    return true
  }
  
  const noDashes = upperKey.replace(/-/g, '')
  return noDashes.match(/^ISX[A-Z0-9]{16}$/) !== null
}

/**
 * Normalize license key input for consistent processing
 */
export function normalizeLicenseKey(value: string): string {
  return cleanLicenseKey(value)
}

/**
 * Rate limiting for activation attempts
 */
const RATE_LIMIT_KEY = 'license_activation_attempts'
const MAX_ATTEMPTS = 10  // Increased from 5 to be more lenient
const TIME_WINDOW = 300000 // 5 minutes (increased from 1 minute)

export function canAttemptActivation(): { allowed: boolean; remainingAttempts: number; resetTime?: number } {
  try {
    const now = Date.now()
    const attempts = JSON.parse(localStorage.getItem(RATE_LIMIT_KEY) || '[]') as number[]
    
    // Filter attempts within the time window
    const recentAttempts = attempts.filter(timestamp => now - timestamp < TIME_WINDOW)
    
    if (recentAttempts.length >= MAX_ATTEMPTS && recentAttempts.length > 0) {
      const oldestAttempt = recentAttempts[0]
      const resetTime = (oldestAttempt || now) + TIME_WINDOW
      
      return {
        allowed: false,
        remainingAttempts: 0,
        resetTime
      }
    }
    
    return {
      allowed: true,
      remainingAttempts: MAX_ATTEMPTS - recentAttempts.length
    }
  } catch {
    // If localStorage is unavailable, allow the attempt
    return { allowed: true, remainingAttempts: MAX_ATTEMPTS }
  }
}

/**
 * Clear rate limit data (useful for debugging or reset)
 */
export function clearRateLimitData(): void {
  try {
    localStorage.removeItem(RATE_LIMIT_KEY)
  } catch {
    // Ignore errors
  }
}

export function recordActivationAttempt(): void {
  try {
    const now = Date.now()
    const attempts = JSON.parse(localStorage.getItem(RATE_LIMIT_KEY) || '[]') as number[]
    
    // Keep only recent attempts and add new one
    const recentAttempts = attempts.filter(timestamp => now - timestamp < TIME_WINDOW)
    recentAttempts.push(now)
    
    localStorage.setItem(RATE_LIMIT_KEY, JSON.stringify(recentAttempts))
  } catch {
    // Ignore localStorage errors
  }
}

/**
 * Cache license status locally for resilience
 */
const LICENSE_CACHE_KEY = 'license_status_cache'
const CACHE_DURATION = 300000 // 5 minutes

export interface CachedLicenseStatus {
  status: string
  expiryDate: string | undefined
  cachedAt: number
}

export function getCachedLicenseStatus(): CachedLicenseStatus | null {
  try {
    const cached = localStorage.getItem(LICENSE_CACHE_KEY)
    if (!cached) return null
    
    const data = JSON.parse(cached) as CachedLicenseStatus
    const now = Date.now()
    
    // Check if cache is still valid
    if (now - data.cachedAt > CACHE_DURATION) {
      localStorage.removeItem(LICENSE_CACHE_KEY)
      return null
    }
    
    return data
  } catch {
    return null
  }
}

export function setCachedLicenseStatus(status: string, expiryDate?: string): void {
  try {
    const data: CachedLicenseStatus = {
      status,
      expiryDate: expiryDate,
      cachedAt: Date.now()
    }
    localStorage.setItem(LICENSE_CACHE_KEY, JSON.stringify(data))
  } catch {
    // Ignore localStorage errors
  }
}

export function clearLicenseCache(): void {
  try {
    localStorage.removeItem(LICENSE_CACHE_KEY)
  } catch {
    // Ignore localStorage errors
  }
}

/**
 * Format time remaining for countdown
 */
export function formatCountdown(seconds: number): string {
  if (seconds <= 0) return 'now'
  if (seconds === 1) return '1 second'
  return `${seconds} seconds`
}

/**
 * Sleep utility for retry logic
 */
export function sleep(ms: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms))
}

/**
 * Exponential backoff retry wrapper
 */
export async function retryWithBackoff<T>(
  fn: () => Promise<T>,
  maxRetries = 3,
  baseDelay = 1000
): Promise<T> {
  let lastError: Error | undefined
  
  for (let i = 0; i < maxRetries; i++) {
    try {
      // Check if online before attempting
      if (!navigator.onLine) {
        throw new Error('No internet connection')
      }
      
      return await fn()
    } catch (error) {
      lastError = error as Error
      
      // Don't retry on client errors (4xx)
      if (error instanceof Error && error.message.includes('4')) {
        throw error
      }
      
      // If this is the last attempt, throw
      if (i === maxRetries - 1) {
        throw error
      }
      
      // Calculate delay with exponential backoff and jitter
      const delay = baseDelay * Math.pow(2, i) + Math.random() * 1000
      await sleep(delay)
    }
  }
  
  throw lastError || new Error('Retry failed')
}

/**
 * Copy text to clipboard with fallback
 */
export async function copyToClipboard(text: string): Promise<boolean> {
  try {
    // Modern API
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(text)
      return true
    }
    
    // Fallback for older browsers
    const textArea = document.createElement('textarea')
    textArea.value = text
    textArea.style.position = 'fixed'
    textArea.style.left = '-999999px'
    textArea.style.top = '-999999px'
    document.body.appendChild(textArea)
    textArea.focus()
    textArea.select()
    
    const successful = document.execCommand('copy')
    document.body.removeChild(textArea)
    
    return successful
  } catch {
    return false
  }
}

/**
 * Session-based redirect tracking to prevent duplicate redirects
 */
const REDIRECT_SESSION_KEY = 'license_redirect_session'

export function hasRedirectedThisSession(): boolean {
  try {
    return sessionStorage.getItem(REDIRECT_SESSION_KEY) === 'true'
  } catch {
    return false
  }
}

export function markRedirectedThisSession(): void {
  try {
    sessionStorage.setItem(REDIRECT_SESSION_KEY, 'true')
  } catch {
    // Ignore sessionStorage errors
  }
}

export function clearRedirectSession(): void {
  try {
    sessionStorage.removeItem(REDIRECT_SESSION_KEY)
  } catch {
    // Ignore sessionStorage errors
  }
}

/**
 * Analytics tracking (placeholder - implement with your analytics provider)
 */
export function trackLicenseEvent(
  event: 'activation_attempt' | 'activation_success' | 'activation_failure' | 'reactivation_success' | 'redirect',
  data?: Record<string, any>
): void {
  // Log to console in development
  if (process.env.NODE_ENV === 'development') {
    console.log('[License Analytics]', event, data)
  }
  
  // TODO: Send to analytics service
  // window.gtag?.('event', event, data)
  // window.plausible?.('License', { props: { event, ...data } })
}