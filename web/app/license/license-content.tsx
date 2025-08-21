/**
 * Professional License Page Client Component
 * Clean, elegant license activation and validation
 */

'use client'

import React, { useState, useEffect, useCallback, useRef } from 'react'
import { useRouter } from 'next/navigation'
import { Check, Shield, ChevronRight, Zap, Lock, Award } from 'lucide-react'
import { ThemeToggle } from '@/components/ui/theme-toggle'

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Progress } from '@/components/ui/progress'
import { Separator } from '@/components/ui/separator'

import { useApi } from '@/lib/hooks/use-api'
import { apiClient } from '@/lib/api'
import { type LicenseActivationForm } from '@/lib/schemas'
import { useToast } from '@/lib/hooks/use-toast'
import type { LicenseApiResponse, ApiError } from '@/types/index'
import { InvestorHeaderLogo } from '@/components/layout/investor-logo'
import { LicenseActivationForm as ActivationForm } from '@/components/license/LicenseActivationForm'
import { 
  getCachedLicenseStatus, 
  setCachedLicenseStatus, 
  retryWithBackoff,
  trackLicenseEvent,
  hasRedirectedThisSession,
  markRedirectedThisSession
} from '@/lib/utils/license-helpers'
import { parseLicenseError, LicenseErrorType } from '@/lib/utils/license-errors'

export default function LicenseContent(): JSX.Element {
  const [licenseData, setLicenseData] = useState<LicenseApiResponse | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [redirectCountdown, setRedirectCountdown] = useState<number | null>(null)
  const [activationError, setActivationError] = useState<ApiError | null>(null)
  const [bannerVisible, setBannerVisible] = useState(false)
  
  // Use refs to avoid stale closures
  const mountedRef = useRef(false)
  const countdownInitializedRef = useRef(false)
  
  const { toast } = useToast()
  const router = useRouter()
  const { execute: activateLicense, loading: activating } = useApi(apiClient.activateLicense.bind(apiClient))

  // Check license status - no dependencies to avoid stale closures
  const checkLicenseStatus = useCallback(async () => {
    try {
      setIsLoading(true)
      
      // Fetch actual status with retry logic
      const status = await retryWithBackoff(
        () => apiClient.getLicenseStatus(),
        3,
        1000
      )
      
      setLicenseData(status)
      
      // Cache if active
      if (status.license_status === 'active' || status.license_status === 'warning') {
        setCachedLicenseStatus(status.license_status, status.license_info?.expiry_date)
        
        // Start redirect countdown only if not already redirected this session
        if (mountedRef.current && !countdownInitializedRef.current && !hasRedirectedThisSession()) {
          countdownInitializedRef.current = true
          setRedirectCountdown(5)
          setBannerVisible(true) // Smooth banner appearance
          trackLicenseEvent('redirect', { source: 'api' })
        }
      }
    } catch (error) {
      console.error('Failed to check license status:', error)
      
      // Use cached status if available
      const cached = getCachedLicenseStatus()
      if (cached) {
        setLicenseData({
          license_status: cached.status,
          message: 'Using cached license status',
          status: 'valid',
          trace_id: 'cache',
          timestamp: new Date().toISOString()
        } as LicenseApiResponse)
        
        if (mountedRef.current && cached.status === 'active' && !countdownInitializedRef.current && !hasRedirectedThisSession()) {
          countdownInitializedRef.current = true
          setRedirectCountdown(5)
          setBannerVisible(true)
        }
      }
    } finally {
      setIsLoading(false)
    }
  }, []) // No dependencies - uses refs instead

  // Single effect for mounting and initial check
  useEffect(() => {
    mountedRef.current = true
    checkLicenseStatus()
    
    return () => {
      mountedRef.current = false
    }
  }, [checkLicenseStatus])

  // Countdown timer with cleanup
  useEffect(() => {
    if (redirectCountdown === null || redirectCountdown < 0) {
      // Hide banner when countdown is stopped
      if (redirectCountdown === null) {
        setBannerVisible(false)
      }
      return
    }
    
    if (redirectCountdown === 0) {
      markRedirectedThisSession() // Mark that we've redirected
      router.push('/operations') // Redirect to operations
      return
    }
    
    const timer = setTimeout(() => {
      setRedirectCountdown(prev => prev !== null ? prev - 1 : null)
    }, 1000)
    
    return () => clearTimeout(timer)
  }, [redirectCountdown, router])

  // Handle license activation
  const handleActivation = useCallback(async (data: LicenseActivationForm) => {
    try {
      setActivationError(null)
      trackLicenseEvent('activation_attempt')
      
      // Add timeout wrapper for the activation request
      const activationPromise = activateLicense({ license_key: data.license_key })
      const timeoutPromise = new Promise((_, reject) => 
        setTimeout(() => reject(new Error('Activation request timed out. Please try again.')), 30000)
      )
      
      const result = await Promise.race([activationPromise, timeoutPromise])
      
      // Parse the result to check if it's a reactivation
      const resultObj = result as any
      const isReactivation = resultObj?.status === 'reactivated' || 
                            (typeof resultObj === 'object' && resultObj?.message?.toLowerCase().includes('reactivated'))
      
      trackLicenseEvent(isReactivation ? 'reactivation_success' : 'activation_success')
      
      // Clear any cached status
      setCachedLicenseStatus('active')
      
      // Show appropriate success message
      if (mountedRef.current) {
        toast({
          title: isReactivation ? "License Reactivated" : "License Activated",
          description: isReactivation 
            ? "Your license has been successfully reactivated on this device" 
            : "Welcome to ISX Pulse Professional",
        })
      }
      
      // Start redirect only if not already initialized and not already redirected this session
      if (!countdownInitializedRef.current && !hasRedirectedThisSession()) {
        countdownInitializedRef.current = true
        setRedirectCountdown(5)
        setBannerVisible(true)
      }
      
      // Refresh status in background
      checkLicenseStatus()
      
    } catch (err) {
      const error = err as ApiError
      
      // Parse the error to check for reactivation success that came through as an error
      const errorDetails = parseLicenseError(error)
      
      // Handle reactivation success that may come through as an "error"
      if (errorDetails.type === LicenseErrorType.REACTIVATION_SUCCESS) {
        trackLicenseEvent('reactivation_success')
        
        // Clear any cached status and set as active
        setCachedLicenseStatus('active')
        
        // Show success message
        if (mountedRef.current) {
          toast({
            title: "License Reactivated",
            description: "Your license has been successfully reactivated on this device",
          })
        }
        
        // Start redirect countdown
        if (!countdownInitializedRef.current && !hasRedirectedThisSession()) {
          countdownInitializedRef.current = true
          setRedirectCountdown(5)
          setBannerVisible(true)
        }
        
        // Refresh status in background
        checkLicenseStatus()
        return
      }
      
      // Handle normal errors
      // Ensure we have a proper error object
      if (typeof err === 'string') {
        setActivationError({ 
          type: 'error',
          title: 'Activation Error',
          message: err,
          detail: err,
          status: 500
        } as ApiError)
      } else if (err instanceof Error) {
        setActivationError({ 
          type: 'error',
          title: 'Activation Error',
          message: err.message,
          detail: err.message,
          status: 500
        } as ApiError)
      } else {
        setActivationError(error)
      }
      
      const errorType = errorDetails.type === LicenseErrorType.REACTIVATION_LIMIT_EXCEEDED ? 'reactivation_limit_exceeded' :
                       errorDetails.type === LicenseErrorType.ALREADY_ACTIVATED_DIFFERENT_DEVICE ? 'different_device' :
                       error?.type || 'unknown'
      
      trackLicenseEvent('activation_failure', { error: errorType })
      
      if (mountedRef.current) {
        toast({
          title: "Activation Failed",
          description: error.detail || "Please check your license key",
          variant: "destructive",
        })
      }
    }
  }, [activateLicense, toast, checkLicenseStatus])

  // Determine license state
  const licenseState = licenseData?.license_status || 'checking'
  const isActive = ['active', 'warning', 'critical', 'reactivated'].includes(licenseState)
  const needsActivation = ['not_activated', 'expired', 'invalid', 'error'].includes(licenseState)

  // Redirect countdown state
  const showCountdown = redirectCountdown !== null && redirectCountdown >= 0
  const countdownProgress = showCountdown ? ((5 - redirectCountdown) / 5) * 100 : 0

  // Show loading state initially to prevent flickering
  if (!mountedRef.current || (isLoading && !licenseData)) {
    return (
      <div className="min-h-screen bg-gradient-to-b from-background to-background/95 dark:from-background dark:to-background">
        {/* Theme Toggle for License Page */}
        <div className="absolute top-4 right-4 z-50">
          <ThemeToggle />
        </div>
        <div className="investor-container py-12">
          <div className="text-center space-y-4 mb-12">
            <InvestorHeaderLogo size="xl" />
            <div className="space-y-2">
              <h1 className="text-4xl font-bold bg-gradient-to-r from-foreground to-foreground/80 bg-clip-text text-transparent">
                Professional License
              </h1>
              <p className="text-lg text-muted-foreground max-w-2xl mx-auto">
                Verifying license status...
              </p>
            </div>
          </div>
          <div className="max-w-4xl mx-auto">
            <Card className="shadow-lg border">
              <CardContent className="py-12">
                <div className="flex items-center justify-center space-x-3">
                  <div className="h-2 w-2 rounded-full bg-blue-500 animate-pulse" />
                  <span className="text-muted-foreground">Checking license status...</span>
                </div>
              </CardContent>
            </Card>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gradient-to-b from-background to-background/95 dark:from-background dark:to-background">
      {/* Theme Toggle for License Page */}
      <div className="absolute top-4 right-4 z-50">
        <ThemeToggle />
      </div>
      {/* Professional Success Banner - Always rendered but visibility controlled */}
      <div 
        className={`w-full transition-all duration-500 ease-in-out overflow-hidden ${
          bannerVisible && showCountdown 
            ? 'max-h-32 opacity-100' 
            : 'max-h-0 opacity-0'
        }`}
      >
        <div className="w-full bg-gradient-to-r from-green-50 via-green-50/80 to-green-50 dark:from-green-900/20 dark:via-green-900/10 dark:to-green-900/20 border-b border-green-200 dark:border-green-800">
          <div className="max-w-4xl mx-auto px-6 py-6">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-4">
                <div className="h-12 w-12 rounded-full bg-gradient-to-br from-green-100 to-emerald-100 dark:from-green-900/40 dark:to-emerald-900/40 flex items-center justify-center shadow-sm">
                  <Check className="h-6 w-6 text-green-700 dark:text-green-400" />
                </div>
                <div>
                  <h3 className="text-lg font-semibold text-foreground">
                    {licenseState === 'reactivated' ? 'License Reactivated Successfully' : 'License Verified Successfully'}
                  </h3>
                  <p className="text-sm text-muted-foreground mt-0.5">
                    Preparing your operations workspace • {redirectCountdown} seconds remaining
                  </p>
                </div>
              </div>
              <div className="flex gap-2">
                <Button 
                  variant="outline" 
                  size="sm"
                  className="border-green-300 dark:border-green-700 hover:bg-green-50 dark:hover:bg-green-900/20"
                  onClick={() => {
                    setRedirectCountdown(null)
                    setBannerVisible(false)
                  }}
                >
                  Stay on Page
                </Button>
                <Button 
                  size="sm"
                  className="bg-green-600 hover:bg-green-700 text-white"
                  onClick={() => router.push('/operations')}
                >
                  Go to Operations
                  <ChevronRight className="ml-1 h-3 w-3" />
                </Button>
              </div>
            </div>
            <Progress 
              value={countdownProgress} 
              className="mt-4 h-1 bg-green-100 dark:bg-green-900/30"
            />
          </div>
        </div>
      </div>

      <div className="investor-container py-12">
        {/* Professional Header */}
        <div className="text-center space-y-4 mb-12">
          <InvestorHeaderLogo size="xl" />
          <div className="space-y-2">
            <h1 className="text-4xl font-bold bg-gradient-to-r from-foreground to-foreground/80 bg-clip-text text-transparent">
              Professional License
            </h1>
            <p className="text-lg text-muted-foreground max-w-2xl mx-auto">
              Unlock comprehensive market intelligence for the Iraqi Stock Exchange
            </p>
          </div>
        </div>

        {/* Main Content Area */}
        <div className="max-w-4xl mx-auto">
          <div className="grid gap-6 lg:grid-cols-3">
            {/* Left Column - Status & Actions */}
            <div className="lg:col-span-2 space-y-6">
              {/* Enhanced Status Card */}
              <Card className="shadow-lg border overflow-hidden">
                {isActive && (
                  <div className="h-1 bg-gradient-to-r from-green-500 to-emerald-500" />
                )}
                <CardHeader className="bg-gradient-to-br from-card to-card/80">
                  <div className="flex items-center justify-between">
                    <CardTitle className="flex items-center gap-3">
                      <div className={`p-2 rounded-lg ${isActive ? 'bg-green-100 dark:bg-green-900/30' : 'bg-muted'}`}>
                        <Shield className={`h-5 w-5 ${isActive ? 'text-green-700 dark:text-green-400' : 'text-muted-foreground'}`} />
                      </div>
                      <span className="text-xl">License Status</span>
                    </CardTitle>
                    {isActive && (
                      <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-400 border-green-200 dark:border-green-800 px-3 py-1">
                        <Check className="h-3 w-3 mr-1.5" />
                        ACTIVE
                      </Badge>
                    )}
                  </div>
                  <CardDescription className="mt-2">
                    {isLoading ? (
                      <span className="flex items-center gap-2">
                        <div className="h-2 w-2 rounded-full bg-blue-500 animate-pulse" />
                        Verifying license status...
                      </span>
                    ) : isActive ? (
                      'Your professional license is active and ready to use'
                    ) : (
                      'Activate your license to access professional features'
                    )}
                  </CardDescription>
                </CardHeader>
                
                {!isLoading && (
                  <CardContent className="pt-6">
                    <div className="grid grid-cols-2 gap-4">
                      <div className="space-y-1">
                        <p className="text-sm text-muted-foreground">Status</p>
                        <p className="font-medium flex items-center gap-2">
                          <div className={`h-2 w-2 rounded-full ${
                            isActive ? 'bg-green-500 dark:bg-green-400' : 'bg-muted-foreground'
                          }`} />
                          {licenseState === 'reactivated' ? 'Reactivated' : licenseState.replace('_', ' ').replace(/\b\w/g, (l: string) => l.toUpperCase())}
                        </p>
                      </div>
                      <div className="space-y-1">
                        <p className="text-sm text-muted-foreground">License Type</p>
                        <p className="font-medium">
                          {isActive ? 'Professional' : 'Not Activated'}
                        </p>
                      </div>
                      {licenseData?.days_left !== undefined && (
                        <>
                          <div className="space-y-1">
                            <p className="text-sm text-muted-foreground">Days Remaining</p>
                            <p className="font-medium">{licenseData.days_left} days</p>
                          </div>
                          <div className="space-y-1">
                            <p className="text-sm text-muted-foreground">Expires</p>
                            <p className="font-medium text-sm">
                              {licenseData.license_info?.expiry_date?.split('T')[0] || 'N/A'}
                            </p>
                          </div>
                        </>
                      )}
                    </div>
                    
                    {isActive && !showCountdown && (
                      <>
                        <Separator className="my-6" />
                        <Button 
                          className="w-full bg-gradient-to-r from-blue-600 to-blue-700 hover:from-blue-700 hover:to-blue-800" 
                          size="lg"
                          onClick={() => router.push('/operations')}
                        >
                          Open Operations
                          <ChevronRight className="ml-2 h-4 w-4" />
                        </Button>
                      </>
                    )}
                  </CardContent>
                )}
              </Card>

              {/* Activation Form */}
              {!isLoading && needsActivation && (
                <ActivationForm
                  onSubmit={handleActivation}
                  loading={activating}
                  error={activationError}
                  activationProgress={0}
                  licenseState={licenseState as 'invalid' | 'expired'}
                />
              )}
            </div>

            {/* Right Column - Features */}
            <div className="space-y-6">
              {/* Quick Features Card */}
              <Card className="shadow-lg border">
                <CardHeader className="pb-4">
                  <CardTitle className="text-lg flex items-center gap-2">
                    <Award className="h-5 w-5 text-blue-600 dark:text-blue-400" />
                    Professional Features
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-3">
                  <div className="flex items-start gap-3">
                    <div className="h-8 w-8 rounded-lg bg-green-100 dark:bg-green-900/30 flex items-center justify-center flex-shrink-0 mt-0.5">
                      <Zap className="h-4 w-4 text-green-700 dark:text-green-400" />
                    </div>
                    <div>
                      <p className="font-medium text-sm">Real-time Data</p>
                      <p className="text-xs text-muted-foreground">Live ISX market updates</p>
                    </div>
                  </div>
                  
                  <div className="flex items-start gap-3">
                    <div className="h-8 w-8 rounded-lg bg-blue-100 dark:bg-blue-900/30 flex items-center justify-center flex-shrink-0 mt-0.5">
                      <Lock className="h-4 w-4 text-blue-700 dark:text-blue-400" />
                    </div>
                    <div>
                      <p className="font-medium text-sm">Advanced Analytics</p>
                      <p className="text-xs text-muted-foreground">Professional insights & reports</p>
                    </div>
                  </div>

                  <div className="flex items-start gap-3">
                    <div className="h-8 w-8 rounded-lg bg-purple-100 dark:bg-purple-900/30 flex items-center justify-center flex-shrink-0 mt-0.5">
                      <Shield className="h-4 w-4 text-purple-700 dark:text-purple-400" />
                    </div>
                    <div>
                      <p className="font-medium text-sm">Enterprise Security</p>
                      <p className="text-xs text-muted-foreground">Bank-grade encryption</p>
                    </div>
                  </div>
                </CardContent>
              </Card>

              {/* Support Card */}
              <Card className="shadow-lg border bg-gradient-to-br from-card to-card/80">
                <CardContent className="pt-6">
                  <div className="text-center space-y-3">
                    <div className="h-12 w-12 rounded-full bg-muted flex items-center justify-center mx-auto">
                      <span className="text-xl">?</span>
                    </div>
                    <div>
                      <p className="font-medium text-sm">Need Help?</p>
                      <p className="text-xs text-muted-foreground mt-1">
                        Contact our support team
                      </p>
                    </div>
                    <Button variant="outline" size="sm" className="w-full" asChild>
                      <a href="mailto:support@isxpulse.com">
                        support@isxpulse.com
                      </a>
                    </Button>
                  </div>
                </CardContent>
              </Card>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}