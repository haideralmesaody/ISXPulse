'use client'

import { usePathname } from 'next/navigation'
import Link from 'next/link'
import { 
  Settings, 
  BarChart3, 
  FileText,
  Menu, 
  X,
  Shield,
  ShieldCheck,
  Clock,
  TrendingUp
} from 'lucide-react'
import { useState, useEffect } from 'react'

import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { ThemeToggle } from '@/components/ui/theme-toggle'
import { useConnectionStatus, useSystemStatus } from '@/lib/hooks/use-websocket'
import { InvestorLogo, InvestorLogoCompact } from '@/components/layout/investor-logo'
import { ErrorBoundary } from '@/components/error-boundary'
import { cn } from '@/lib/utils'
import { apiClient } from '@/lib/api'
import type { LicenseApiResponse } from '@/types/index'

// Simple hook for license status API
function useSimpleLicenseStatus() {
  const [response, setResponse] = useState<LicenseApiResponse | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const fetchLicenseStatus = async () => {
      try {
        const data = await apiClient.getLicenseStatus()
        setResponse(data)
      } catch (error) {
        console.error('License status fetch failed:', error)
        setResponse(null)
      } finally {
        setLoading(false)
      }
    }

    fetchLicenseStatus()
    // Poll every 30 seconds for updates
    const interval = setInterval(fetchLicenseStatus, 30000)
    return () => clearInterval(interval)
  }, [])

  return { response, loading }
}

// Navigation items configuration
const navigationItems = [
  {
    name: 'Operations',
    href: '/operations',
    icon: Settings,
    description: 'Manage data collection and processing'
  },
  {
    name: 'Liquidity',
    href: '/liquidity',
    icon: BarChart3,
    description: 'ISX Hybrid Liquidity Metrics and scoring'
  },
  {
    name: 'Analysis',
    href: '/analysis',
    icon: TrendingUp,
    description: 'Technical analysis with advanced charting'
  },
  {
    name: 'Reports',
    href: '/reports',
    icon: FileText,
    description: 'Generated reports and data exports'
  }
]

interface NavigationProps {
  currentPath: string
  isMobileMenuOpen: boolean
  onMobileMenuToggle: () => void
}

function Navigation({ currentPath, isMobileMenuOpen, onMobileMenuToggle }: NavigationProps) {
  return (
    <>
      {/* Desktop Navigation */}
      <nav className="hidden md:flex items-center space-x-6">
        {navigationItems.map((item) => {
          const isActive = currentPath === item.href || 
                          (item.href !== '/' && currentPath.startsWith(item.href))
          
          return (
            <Link
              key={item.href}
              href={item.href}
              className={cn(
                "text-sm font-medium transition-colors relative group",
                isActive 
                  ? "text-foreground" 
                  : "text-muted-foreground hover:text-foreground"
              )}
            >
              <div className="flex items-center space-x-2">
                <item.icon className="h-4 w-4" />
                <span>{item.name}</span>
              </div>
              {isActive && (
                <div className="absolute -bottom-6 left-0 right-0 h-0.5 bg-primary" />
              )}
              
              {/* Tooltip */}
              <div className="absolute top-full left-1/2 transform -translate-x-1/2 mt-2 px-2 py-1 bg-background border rounded shadow-lg text-xs whitespace-nowrap opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none z-50">
                {item.description}
              </div>
            </Link>
          )
        })}
      </nav>

      {/* Mobile Menu Button */}
      <div className="md:hidden">
        <Button
          variant="ghost"
          size="sm"
          onClick={onMobileMenuToggle}
          aria-label="Toggle navigation menu"
        >
          {isMobileMenuOpen ? (
            <X className="h-5 w-5" />
          ) : (
            <Menu className="h-5 w-5" />
          )}
        </Button>
      </div>

      {/* Mobile Navigation */}
      {isMobileMenuOpen && (
        <div className="md:hidden absolute top-16 left-0 right-0 bg-background border-b shadow-lg z-40">
          <div className="px-4 py-4 space-y-3">
            {navigationItems.map((item) => {
              const isActive = currentPath === item.href || 
                              (item.href !== '/' && currentPath.startsWith(item.href))
              
              return (
                <Link
                  key={item.href}
                  href={item.href}
                  onClick={onMobileMenuToggle}
                  className={cn(
                    "flex items-center space-x-3 p-3 rounded-lg transition-colors",
                    isActive 
                      ? "bg-primary/10 text-foreground" 
                      : "text-muted-foreground hover:bg-muted hover:text-foreground"
                  )}
                >
                  <item.icon className="h-5 w-5" />
                  <div>
                    <div className="font-medium">{item.name}</div>
                    <div className="text-xs text-muted-foreground">{item.description}</div>
                  </div>
                </Link>
              )
            })}
          </div>
        </div>
      )}
    </>
  )
}

interface StatusIndicatorProps {
  isConnected: boolean
  isHealthy: boolean
}

function StatusIndicator({ isConnected, isHealthy }: StatusIndicatorProps) {
  const getOverallStatus = () => {
    if (isConnected && isHealthy) return 'optimal'
    if (isConnected || isHealthy) return 'good'
    return 'inactive'
  }

  const status = getOverallStatus()

  const statusConfig = {
    optimal: { 
      color: 'bg-green-500', 
      text: 'Connected', 
      textColor: 'text-green-600',
      icon: ShieldCheck,
      description: 'System fully operational'
    },
    good: { 
      color: 'bg-blue-500', 
      text: 'Partial', 
      textColor: 'text-blue-600',
      icon: Shield,
      description: 'System partially connected'
    },
    limited: { 
      color: 'bg-yellow-500', 
      text: 'Limited', 
      textColor: 'text-yellow-600',
      icon: Clock,
      description: 'Limited connectivity'
    },
    inactive: { 
      color: 'bg-red-500', 
      text: 'Offline', 
      textColor: 'text-red-600',
      icon: X,
      description: 'System disconnected'
    }
  }

  const config = statusConfig[status]

  return (
    <div className="flex items-center space-x-2">
      {/* System Status */}
      <div 
        className="flex items-center space-x-2 relative group cursor-help" 
        title={config.description}
      >
        <config.icon className="h-4 w-4" />
        <span className={`text-xs font-medium ${config.textColor} hidden sm:inline`}>
          {config.text}
        </span>
        
        {/* Enhanced Tooltip */}
        <div className="absolute top-full right-0 mt-2 px-3 py-2 bg-background border rounded-lg shadow-lg text-xs whitespace-nowrap opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none z-50">
          <div className="font-semibold mb-1">System Status</div>
          <div className="text-muted-foreground">{config.description}</div>
          <div className="text-muted-foreground text-[10px] mt-1">
            WebSocket: {isConnected ? 'Connected' : 'Disconnected'}
          </div>
        </div>
      </div>
    </div>
  )
}

function AppHeader() {
  const pathname = usePathname()
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false)

  // Real-time status integration
  const { isConnected } = useConnectionStatus()
  const { isHealthy } = useSystemStatus()
  // License status is now handled in footer only
  
  // Close mobile menu on route change - hook must be called before any returns
  useEffect(() => {
    setIsMobileMenuOpen(false)
  }, [pathname])

  // Close mobile menu on outside click - hook must be called before any returns
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      const target = event.target as Element
      if (isMobileMenuOpen && !target.closest('header')) {
        setIsMobileMenuOpen(false)
      }
    }

    document.addEventListener('click', handleClickOutside)
    return () => document.removeEventListener('click', handleClickOutside)
  }, [isMobileMenuOpen])
  
  // Don't render header on license page for clean experience
  if (pathname === '/license') {
    return null
  }

  return (
    <header className="sticky top-0 z-50 border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="px-6">
        <div className="flex h-16 items-center justify-between">
          {/* Logo and Brand */}
          <Link href="/" className="hover:opacity-80 transition-opacity">
            <div className="hidden sm:block">
              <InvestorLogo size="lg" />
            </div>
            <div className="sm:hidden">
              <InvestorLogoCompact />
            </div>
          </Link>
          
          {/* Navigation */}
          <Navigation 
            currentPath={pathname} 
            isMobileMenuOpen={isMobileMenuOpen}
            onMobileMenuToggle={() => setIsMobileMenuOpen(!isMobileMenuOpen)}
          />
          
          {/* Status Indicators and Theme Toggle */}
          <div className="flex items-center space-x-2">
            <StatusIndicator 
              isConnected={isConnected}
              isHealthy={isHealthy}
            />
            <ThemeToggle />
          </div>
        </div>
      </div>
    </header>
  )
}

function AppFooter() {
  const pathname = usePathname()
  const [currentYear, setCurrentYear] = useState(2025)
  
  // Simple license status from API
  const { response } = useSimpleLicenseStatus()
  const [isClient, setIsClient] = useState(false)
  
  useEffect(() => {
    setIsClient(true)
    // Only set current year on client side to avoid hydration mismatch
    if (typeof window !== 'undefined') {
      setCurrentYear(new Date().getFullYear())
    }
  }, [])
  
  // Don't render footer on license page for clean experience
  if (pathname === '/license') {
    return null
  }

  // Simple license logic - handle both 'active' and 'warning' as licensed states
  const isLicensed = response?.license_status === 'active' || response?.license_status === 'warning'
  const daysLeft = response?.days_left || 0
  const displayText = isLicensed 
    ? `Licensed (${daysLeft} days remaining)` 
    : 'Unlicensed'
  const statusColor = response?.license_status === 'active' ? 'text-green-500' : 
                      response?.license_status === 'warning' ? 'text-yellow-500' : 'text-red-500'
  
  return (
    <footer className="sticky bottom-0 z-40 border-t bg-card/95 backdrop-blur supports-[backdrop-filter]:bg-card/60">
      <div className="px-6">
        <div className="flex h-12 items-center justify-between text-xs text-muted-foreground">
          <div className="flex items-center space-x-4">
            <span suppressHydrationWarning>© {currentYear} ISX Pulse</span>
            <Badge variant="secondary" className="text-xs">
              Professional v1.0.0
            </Badge>
            {/* License status display - only show client-side */}
            {isClient && (
              <div className="flex items-center space-x-1">
                <span>License:</span>
                <span className={cn("font-medium", statusColor)}>
                  {displayText}
                </span>
              </div>
            )}
          </div>
          <div className="hidden sm:flex items-center space-x-4">
            <span>Market Intelligence Platform</span>
            <span>•</span>
            <span>Enterprise Grade</span>
            <span>•</span>
            <span>Real-time Processing</span>
          </div>
        </div>
      </div>
    </footer>
  )
}

interface AppContentProps {
  children: React.ReactNode
}

export default function AppContentClient({ children }: AppContentProps) {
  const pathname = usePathname()
  
  // All pages use the same full-width layout
  return (
    <ErrorBoundary>
      <div className="min-h-screen bg-background flex flex-col">
        <AppHeader />
        
        <main className="flex-1">
          {children}
        </main>
        
        <AppFooter />
      </div>
    </ErrorBoundary>
  )
}