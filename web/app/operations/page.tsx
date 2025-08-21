import dynamic from 'next/dynamic'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { 
  Loader2,
  Download,
  Zap,
  Database,
  FileSpreadsheet,
  BarChart3,
  Workflow,
  Info
} from 'lucide-react'

// Dynamic import with SSR disabled to prevent hydration issues
const OperationsContent = dynamic(() => import('./operations-content'), {
  ssr: false,
  loading: () => <OperationsPageSkeleton />
})

// SEO metadata
export const metadata = {
  title: 'Operations - ISX Pulse',
  description: 'Manage and monitor ISX data processing operations with real-time WebSocket updates.',
  robots: { index: false, follow: false }
}

// Loading skeleton that matches the operations page layout
function OperationsPageSkeleton() {
  const mockOperationTypes = [
    { id: 'scraping', name: 'Data Scraping', Icon: Download },
    { id: 'processing', name: 'Data Processing', Icon: FileSpreadsheet },
    { id: 'indices', name: 'Index Analysis', Icon: BarChart3 },
    { id: 'liquidity', name: 'Liquidity Analysis', Icon: Zap },
    { id: 'full_pipeline', name: 'Full Pipeline', Icon: Workflow }
  ]

  return (
    <div className="min-h-screen p-8">
      <div className="max-w-7xl mx-auto space-y-8">
        {/* Header Skeleton */}
        <div>
          <div className="h-9 w-48 bg-muted rounded-md animate-pulse mb-2" />
          <div className="h-5 w-96 bg-muted rounded-md animate-pulse" />
        </div>
        
        {/* WebSocket Status Skeleton */}
        <div className="flex items-center gap-2">
          <div className="h-2 w-2 rounded-full bg-muted animate-pulse" />
          <div className="h-4 w-24 bg-muted rounded-md animate-pulse" />
        </div>
        
        {/* Operation Types Section Skeleton */}
        <div>
          <div className="h-7 w-48 bg-muted rounded-md animate-pulse mb-4" />
          <div className="grid grid-cols-1 md:grid-cols-3 lg:grid-cols-5 gap-4">
            {mockOperationTypes.map((type, index) => (
              <Card key={index} className="hover:shadow-lg transition-shadow">
                <CardHeader className="pb-3">
                  <div className="flex items-center justify-between">
                    <type.Icon className="h-8 w-8 text-muted-foreground animate-pulse" />
                    <Badge variant="outline" className="animate-pulse">
                      <div className="h-3 w-8 bg-muted rounded" />
                    </Badge>
                  </div>
                  <div className="h-5 w-32 bg-muted rounded-md animate-pulse" />
                </CardHeader>
                <CardContent>
                  <div className="space-y-2">
                    <div className="h-3 w-full bg-muted rounded-md animate-pulse" />
                    <div className="h-3 w-3/4 bg-muted rounded-md animate-pulse" />
                  </div>
                  <div className="h-8 w-full bg-muted rounded-md animate-pulse mt-3" />
                </CardContent>
              </Card>
            ))}
          </div>
        </div>
        
        {/* Active Operations Skeleton */}
        <div>
          <div className="h-7 w-48 bg-muted rounded-md animate-pulse mb-4" />
          <Card className="p-8">
            <div className="text-center">
              <Info className="h-12 w-12 text-muted-foreground mx-auto mb-4 animate-pulse" />
              <div className="h-6 w-48 bg-muted rounded-md animate-pulse mx-auto mb-2" />
              <div className="h-4 w-64 bg-muted rounded-md animate-pulse mx-auto" />
            </div>
          </Card>
        </div>
        
        {/* Loading Indicator */}
        <div className="fixed bottom-8 right-8">
          <div className="bg-background border rounded-lg p-4 shadow-lg">
            <div className="flex items-center gap-3">
              <Loader2 className="h-5 w-5 animate-spin text-primary" />
              <span className="text-sm text-muted-foreground">
                Initializing operations...
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

export default function OperationsPage() {
  return <OperationsContent />
}