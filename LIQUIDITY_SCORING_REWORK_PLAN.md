# Liquidity Scoring System Rework Plan

## Executive Summary
Complete rework of the liquidity scoring system to implement a multi-mode approach with EMA (default), Most Recent, and Average scoring modes. This addresses current issues with HKAR/BTIB showing misleadingly high scores and empty day trading categories.

## Current Issues
1. **HKAR/BTIB Problem**: Stocks with POOR data quality show scores of 90/80 due to early trading days with relative scaling
2. **Day Trading Empty**: Categorization logic has a bug preventing stocks from being added
3. **Inconsistent Scoring**: Using "best" historical score creates misleading representations
4. **Zero Thresholds**: POOR quality stocks show zero safe trading sizes but high scores

## Solution Architecture

### Core Principles
1. **EMA as Primary**: Use 20-period Exponential Moving Average as the default scoring method
2. **Multiple Modes**: Provide flexibility with EMA, Latest, and Average modes
3. **Transparency**: Show all metrics with tooltips for user education
4. **Data Quality Awareness**: POOR quality data automatically results in zero thresholds

## Detailed Implementation Plan

### Phase 1: Backend Data Structure Changes

#### 1.1 New Data Models
Location: `api/internal/services/liquidity_service.go`

```go
// StockMetrics holds all calculated values for a single mode
type StockMetrics struct {
    Score        float64 `json:"score"`
    Conservative float64 `json:"conservative"`
    Moderate     float64 `json:"moderate"`
    Aggressive   float64 `json:"aggressive"`
    Optimal      float64 `json:"optimal"`
    Continuity   float64 `json:"continuity"`
    DailyVolume  float64 `json:"dailyVolume"`
}

// StockRecommendation with multi-mode support
type StockRecommendation struct {
    Symbol      string      `json:"symbol"`
    DataQuality string      `json:"dataQuality"`
    Categories  []string    `json:"categories,omitempty"`
    
    // All calculation modes
    EMA         StockMetrics `json:"ema"`        // 20-period EMA (default)
    Latest      StockMetrics `json:"latest"`     // Most recent values
    Average     StockMetrics `json:"average"`    // Simple average
    Best        StockMetrics `json:"best"`       // Historical best (reference only)
    
    // Derived fields (calculated based on active mode)
    Action      string       `json:"action"`
    Rationale   string       `json:"rationale"`
}

// CategorySet for each scoring mode
type CategorySet struct {
    TopOpportunities   []string `json:"topOpportunities"`
    BestForLargeTrades []string `json:"bestForLargeTrades"`
    BestForDayTrading  []string `json:"bestForDayTrading"`
    HighRisk          []string `json:"highRisk"`
}

// LiquidityInsights with multi-mode categories
type LiquidityInsights struct {
    GeneratedAt        time.Time             `json:"generatedAt"`
    MarketHealthScore  float64               `json:"marketHealthScore"`
    TotalStocks        int                   `json:"totalStocks"`
    HighQualityStocks  int                   `json:"highQualityStocks"`
    
    // Single master list with all metrics
    AllStocks          []StockRecommendation `json:"allStocks"`
    
    // Categories for each scoring mode
    Categories         struct {
        EMA     CategorySet `json:"ema"`
        Latest  CategorySet `json:"latest"`
        Average CategorySet `json:"average"`
    } `json:"categories"`
}
```

#### 1.2 Aggregation Logic Updates

```go
func (s *LiquidityService) aggregateStockData(entries []StockRecommendation) StockRecommendation {
    // Step 1: Filter and validate entries
    validEntries := filterValidEntries(entries)
    if len(validEntries) == 0 {
        return createEmptyStock(entries[0].Symbol)
    }
    
    // Step 2: Extract time series for each metric
    timeSeries := extractTimeSeriesData(validEntries)
    
    // Step 3: Calculate EMA for all metrics
    emaMetrics := calculateEMAMetrics(timeSeries)
    
    // Step 4: Get latest values
    latestMetrics := extractLatestMetrics(validEntries)
    
    // Step 5: Calculate averages
    averageMetrics := calculateAverageMetrics(timeSeries)
    
    // Step 6: Find best historical values
    bestMetrics := findBestMetrics(timeSeries)
    
    // Step 7: Handle POOR quality data
    if lastEntry.DataQuality == "POOR" {
        zeroOutUnreliableMetrics(&emaMetrics, &latestMetrics, &averageMetrics)
    }
    
    return StockRecommendation{
        Symbol:      entries[0].Symbol,
        DataQuality: lastEntry.DataQuality,
        EMA:         emaMetrics,
        Latest:      latestMetrics,
        Average:     averageMetrics,
        Best:        bestMetrics,
    }
}
```

#### 1.3 EMA Calculation Enhancement

```go
func (s *LiquidityService) calculateEMA20(values []float64) float64 {
    if len(values) == 0 {
        return 0
    }
    
    // Remove outliers first (optional)
    cleanedValues := removeOutliers(values)
    
    // Standard EMA calculation
    alpha := 2.0 / 21.0  // Smoothing factor for 20 periods
    ema := cleanedValues[0]
    
    for i := 1; i < len(cleanedValues); i++ {
        ema = alpha*cleanedValues[i] + (1-alpha)*ema
    }
    
    return ema
}

// Helper to handle POOR quality entries
func removeOutliers(values []float64) []float64 {
    // Implement IQR-based outlier removal
    // This helps with the HKAR/BTIB early high scores
}
```

### Phase 2: Categorization Logic

#### 2.1 Multi-Mode Categorization

```go
func (s *LiquidityService) categorizeStocks(stocks []StockRecommendation) Categories {
    categories := Categories{}
    
    // Generate categories for each mode
    categories.EMA = s.categorizeByMode(stocks, "ema")
    categories.Latest = s.categorizeByMode(stocks, "latest")
    categories.Average = s.categorizeByMode(stocks, "average")
    
    return categories
}

func (s *LiquidityService) categorizeByMode(stocks []StockRecommendation, mode string) CategorySet {
    // Create a copy for sorting
    sorted := make([]StockRecommendation, len(stocks))
    copy(sorted, stocks)
    
    // Sort based on the selected mode's score
    sort.Slice(sorted, func(i, j int) bool {
        switch mode {
        case "latest":
            return sorted[i].Latest.Score > sorted[j].Latest.Score
        case "average":
            return sorted[i].Average.Score > sorted[j].Average.Score
        default: // ema
            return sorted[i].EMA.Score > sorted[j].EMA.Score
        }
    })
    
    var categories CategorySet
    
    for i, stock := range sorted {
        metrics := getMetricsByMode(stock, mode)
        
        // Top Opportunities (top 10 with score >= 50)
        if i < 10 && metrics.Score >= 50 && stock.DataQuality != "POOR" {
            categories.TopOpportunities = append(categories.TopOpportunities, stock.Symbol)
        }
        
        // Day Trading (continuity >= 0.7 AND score >= 50)
        if metrics.Continuity >= 0.7 && metrics.Score >= 50 && len(categories.BestForDayTrading) < 5 {
            categories.BestForDayTrading = append(categories.BestForDayTrading, stock.Symbol)
        }
        
        // Large Trades (optimal >= 5M AND not POOR)
        if metrics.Optimal >= 5_000_000 && stock.DataQuality != "POOR" && len(categories.BestForLargeTrades) < 5 {
            categories.BestForLargeTrades = append(categories.BestForLargeTrades, stock.Symbol)
        }
        
        // High Risk (score < 30 OR POOR quality)
        if (metrics.Score < 30 || stock.DataQuality == "POOR") && len(categories.HighRisk) < 10 {
            categories.HighRisk = append(categories.HighRisk, stock.Symbol)
        }
    }
    
    return categories
}
```

### Phase 3: API Updates

#### 3.1 Enhanced Endpoint
Location: `api/internal/transport/http/liquidity_handler.go`

```go
// GetInsights with optional mode parameter
func (h *LiquidityHandler) GetInsights(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    
    // Parse optional mode parameter
    mode := r.URL.Query().Get("mode")
    if mode == "" {
        mode = "ema" // Default
    }
    
    // Validate mode
    if mode != "ema" && mode != "latest" && mode != "average" {
        h.errorHandler.HandleError(w, r, apierrors.New(
            http.StatusBadRequest,
            "INVALID_MODE",
            "Invalid scoring mode. Use: ema, latest, or average",
        ))
        return
    }
    
    // Get insights (always returns all modes)
    insights, err := h.service.GetLatestInsights(ctx)
    if err != nil {
        h.errorHandler.HandleError(w, r, err)
        return
    }
    
    // Optional: Add active mode hint in response header
    w.Header().Set("X-Scoring-Mode", mode)
    
    render.JSON(w, r, insights)
}
```

### Phase 4: Frontend Implementation

#### 4.1 New Toggle Component
Location: `web/components/liquidity/ScoreModeToggle.tsx`

```typescript
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group"
import { TrendingUp, Clock, Calculator, Info } from "lucide-react"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"

export type ScoringMode = 'ema' | 'latest' | 'average'

interface ScoreModeToggleProps {
  mode: ScoringMode
  onChange: (mode: ScoringMode) => void
}

export function ScoreModeToggle({ mode, onChange }: ScoreModeToggleProps) {
  return (
    <div className="flex items-center gap-2">
      <span className="text-sm text-muted-foreground">Scoring:</span>
      <ToggleGroup type="single" value={mode} onValueChange={onChange}>
        <Tooltip>
          <TooltipTrigger asChild>
            <ToggleGroupItem value="ema" aria-label="EMA scoring">
              <TrendingUp className="h-4 w-4 mr-1" />
              EMA
            </ToggleGroupItem>
          </TooltipTrigger>
          <TooltipContent>
            <p>20-period Exponential Moving Average</p>
            <p className="text-xs text-muted-foreground">Smooths out anomalies, most stable</p>
          </TooltipContent>
        </Tooltip>
        
        <Tooltip>
          <TooltipTrigger asChild>
            <ToggleGroupItem value="latest" aria-label="Latest scoring">
              <Clock className="h-4 w-4 mr-1" />
              Recent
            </ToggleGroupItem>
          </TooltipTrigger>
          <TooltipContent>
            <p>Most recent trading day values</p>
            <p className="text-xs text-muted-foreground">Real-time but more volatile</p>
          </TooltipContent>
        </Tooltip>
        
        <Tooltip>
          <TooltipTrigger asChild>
            <ToggleGroupItem value="average" aria-label="Average scoring">
              <Calculator className="h-4 w-4 mr-1" />
              Average
            </ToggleGroupItem>
          </TooltipTrigger>
          <TooltipContent>
            <p>Simple average of all historical data</p>
            <p className="text-xs text-muted-foreground">Balanced view of performance</p>
          </TooltipContent>
        </Tooltip>
      </ToggleGroup>
      
      <Tooltip>
        <TooltipTrigger>
          <Info className="h-4 w-4 text-muted-foreground" />
        </TooltipTrigger>
        <TooltipContent className="max-w-xs">
          <p className="font-semibold mb-1">Scoring Modes Explained:</p>
          <ul className="text-xs space-y-1">
            <li><strong>EMA:</strong> Best for identifying trends, reduces impact of outliers</li>
            <li><strong>Recent:</strong> Best for current market conditions</li>
            <li><strong>Average:</strong> Best for long-term perspective</li>
          </ul>
        </TooltipContent>
      </Tooltip>
    </div>
  )
}
```

#### 4.2 Enhanced Trading Card
Location: `web/components/liquidity/TradingCard.tsx`

```typescript
interface TradingCardProps {
  stock: StockRecommendation
  mode: ScoringMode
}

export function TradingCard({ stock, mode }: TradingCardProps) {
  // Select metrics based on active mode
  const metrics = mode === 'ema' ? stock.ema : 
                  mode === 'latest' ? stock.latest : 
                  stock.average
  
  // Generate action and rationale based on mode
  const { action, rationale } = generateRecommendation(metrics, stock.dataQuality)
  
  return (
    <Card className="hover:shadow-lg transition-all">
      <CardHeader>
        <div className="flex items-start justify-between">
          <div>
            <CardTitle className="text-lg font-mono">{stock.symbol}</CardTitle>
            <div className="flex items-center gap-2 mt-1">
              <Badge variant={getActionVariant(action)}>
                {action.replace('_', ' ')}
              </Badge>
              {stock.dataQuality === 'POOR' && (
                <Badge variant="destructive" className="text-xs">
                  Unreliable Data
                </Badge>
              )}
            </div>
          </div>
          
          {/* Score with comparison tooltip */}
          <Tooltip>
            <TooltipTrigger>
              <div className={`text-2xl font-bold ${getScoreColor(metrics.score)}`}>
                {metrics.score.toFixed(1)}
              </div>
            </TooltipTrigger>
            <TooltipContent>
              <div className="space-y-2">
                <p className="font-semibold">Score Comparison</p>
                <div className="grid grid-cols-2 gap-2 text-xs">
                  <div className={mode === 'ema' ? 'font-bold' : ''}>
                    EMA: {stock.ema.score.toFixed(1)}
                  </div>
                  <div className={mode === 'latest' ? 'font-bold' : ''}>
                    Latest: {stock.latest.score.toFixed(1)}
                  </div>
                  <div className={mode === 'average' ? 'font-bold' : ''}>
                    Average: {stock.average.score.toFixed(1)}
                  </div>
                  <div className="text-muted-foreground">
                    Best: {stock.best.score.toFixed(1)}
                  </div>
                </div>
              </div>
            </TooltipContent>
          </Tooltip>
        </div>
      </CardHeader>
      
      <CardContent className="space-y-3">
        <p className="text-sm text-muted-foreground">{rationale}</p>
        
        {/* Trading Thresholds */}
        <div className="space-y-2">
          <h4 className="text-xs font-semibold uppercase text-muted-foreground">
            Safe Trading Sizes ({mode.toUpperCase()})
          </h4>
          <div className="grid grid-cols-2 gap-2 text-sm">
            <Tooltip>
              <TooltipTrigger className="flex items-center gap-1">
                <Shield className="h-3 w-3 text-green-600" />
                <span className="text-xs">Conservative:</span>
                <span className="font-medium">{formatCurrency(metrics.conservative)}</span>
              </TooltipTrigger>
              <TooltipContent>
                <ComparisonTooltip 
                  label="Conservative" 
                  ema={stock.ema.conservative}
                  latest={stock.latest.conservative}
                  average={stock.average.conservative}
                  current={metrics.conservative}
                  mode={mode}
                />
              </TooltipContent>
            </Tooltip>
            {/* Repeat for other thresholds */}
          </div>
        </div>
        
        {/* Metrics */}
        <div className="flex items-center justify-between text-xs pt-2 border-t">
          <Tooltip>
            <TooltipTrigger>
              <span>Continuity: {(metrics.continuity * 100).toFixed(0)}%</span>
            </TooltipTrigger>
            <TooltipContent>
              <div className="text-xs">
                <p>EMA: {(stock.ema.continuity * 100).toFixed(0)}%</p>
                <p>Latest: {(stock.latest.continuity * 100).toFixed(0)}%</p>
                <p>Average: {(stock.average.continuity * 100).toFixed(0)}%</p>
              </div>
            </TooltipContent>
          </Tooltip>
          
          <Tooltip>
            <TooltipTrigger>
              <span>Volume: {formatCurrency(metrics.dailyVolume)}</span>
            </TooltipTrigger>
            <TooltipContent>
              <div className="text-xs">
                <p>EMA: {formatCurrency(stock.ema.dailyVolume)}</p>
                <p>Latest: {formatCurrency(stock.latest.dailyVolume)}</p>
                <p>Average: {formatCurrency(stock.average.dailyVolume)}</p>
              </div>
            </TooltipContent>
          </Tooltip>
        </div>
      </CardContent>
    </Card>
  )
}
```

#### 4.3 Updated Dashboard
Location: `web/app/liquidity/liquidity-dashboard.tsx`

```typescript
export default function LiquidityDashboard() {
  const [insights, setInsights] = useState<LiquidityInsights | null>(null)
  const [scoringMode, setScoringMode] = useState<ScoringMode>(() => {
    // Load saved preference
    if (typeof window !== 'undefined') {
      return (localStorage.getItem('liquidityScoringMode') as ScoringMode) || 'ema'
    }
    return 'ema'
  })
  
  // Save mode preference
  useEffect(() => {
    if (typeof window !== 'undefined') {
      localStorage.setItem('liquidityScoringMode', scoringMode)
    }
  }, [scoringMode])
  
  // Get active categories based on mode
  const getActiveCategories = useCallback((): CategorySet | null => {
    if (!insights) return null
    return insights.categories[scoringMode]
  }, [insights, scoringMode])
  
  // Get stocks for a category
  const getCategoryStocks = useCallback((categoryName: keyof CategorySet): StockRecommendation[] => {
    if (!insights) return []
    const categories = getActiveCategories()
    if (!categories) return []
    
    const symbols = categories[categoryName]
    return symbols.map(symbol => 
      insights.allStocks.find(s => s.symbol === symbol)
    ).filter(Boolean) as StockRecommendation[]
  }, [insights, getActiveCategories])
  
  return (
    <main className="min-h-screen p-8">
      <div className="max-w-7xl mx-auto space-y-8">
        {/* Header with Toggle */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold">Liquidity Analysis Dashboard</h1>
            <p className="text-muted-foreground mt-2">
              Real-time ISX liquidity insights with safe trading recommendations
            </p>
          </div>
          <div className="flex items-center gap-4">
            <ScoreModeToggle mode={scoringMode} onChange={setScoringMode} />
            <Button onClick={loadInsights} variant="outline">
              <RefreshCw className="h-4 w-4 mr-2" />
              Refresh
            </Button>
          </div>
        </div>
        
        {/* Market Overview Cards */}
        <MarketOverview insights={insights} mode={scoringMode} />
        
        {/* Main Content Tabs */}
        <Tabs defaultValue="opportunities" className="space-y-4">
          <TabsList className="grid w-full grid-cols-4">
            <TabsTrigger value="opportunities">
              Top Opportunities
              <Badge className="ml-2" variant="secondary">
                {getActiveCategories()?.topOpportunities.length || 0}
              </Badge>
            </TabsTrigger>
            <TabsTrigger value="large">
              Large Trades
              <Badge className="ml-2" variant="secondary">
                {getActiveCategories()?.bestForLargeTrades.length || 0}
              </Badge>
            </TabsTrigger>
            <TabsTrigger value="daytrading">
              Day Trading
              <Badge className="ml-2" variant="secondary">
                {getActiveCategories()?.bestForDayTrading.length || 0}
              </Badge>
            </TabsTrigger>
            <TabsTrigger value="risk">
              High Risk
              <Badge className="ml-2" variant="destructive">
                {getActiveCategories()?.highRisk.length || 0}
              </Badge>
            </TabsTrigger>
          </TabsList>
          
          {/* Tab Contents */}
          <TabsContent value="opportunities">
            <CategoryTab
              title="Top Trading Opportunities"
              description="Stocks with the best overall liquidity scores"
              stocks={getCategoryStocks('topOpportunities')}
              mode={scoringMode}
              emptyMessage="No stocks currently meet opportunity criteria"
            />
          </TabsContent>
          
          {/* ... other tabs ... */}
        </Tabs>
      </div>
    </main>
  )
}
```

### Phase 5: Testing Strategy

#### 5.1 Unit Tests
- Test EMA calculation with various data patterns
- Test categorization logic for all modes
- Test POOR quality data handling
- Test outlier removal

#### 5.2 Integration Tests
- Test full pipeline with real CSV data
- Verify HKAR/BTIB scores are properly handled
- Verify day trading category is populated
- Test mode switching in API

#### 5.3 Frontend Tests
- Test mode toggle functionality
- Test localStorage persistence
- Test tooltip comparisons
- Test category switching

### Phase 6: Migration & Deployment

#### 6.1 Data Migration
- No database changes required
- Existing CSV files remain compatible
- API is backward compatible (defaults to EMA)

#### 6.2 Deployment Steps
1. Deploy backend changes
2. Deploy frontend changes
3. Monitor for any issues
4. Document mode preferences for users

## Success Criteria

1. **HKAR/BTIB Issue Resolved**: 
   - EMA scores should be low (~0-30) due to POOR quality
   - Thresholds should be 0 for POOR quality stocks

2. **Day Trading Populated**:
   - Stocks with continuity >= 0.7 appear in day trading
   - At least 3-5 stocks should qualify

3. **User Experience**:
   - Mode toggle works smoothly
   - Tooltips provide clear comparisons
   - Categories update based on selected mode

4. **Performance**:
   - API response time < 100ms
   - Frontend renders smoothly with mode switches
   - No memory leaks with repeated refreshes

## Risk Mitigation

1. **Backward Compatibility**: API defaults to EMA mode if not specified
2. **Data Quality**: POOR quality always results in zero thresholds
3. **Performance**: Calculate all modes once, cache results
4. **User Education**: Comprehensive tooltips explain each mode

## Timeline

- **Day 1**: Backend implementation (Phases 1-2)
- **Day 2**: API updates and testing (Phase 3)
- **Day 3**: Frontend implementation (Phase 4)
- **Day 4**: Testing and bug fixes (Phase 5)
- **Day 5**: Deployment and monitoring (Phase 6)

## Appendix: Sample Calculations

### EMA Calculation Example
```
Given scores: [90, 20, 20, 38, 43, 55, 62, 61, 62]
Alpha = 2/21 = 0.0952

EMA[0] = 90
EMA[1] = 0.0952 * 20 + 0.9048 * 90 = 83.33
EMA[2] = 0.0952 * 20 + 0.9048 * 83.33 = 77.28
...
Final EMA ≈ 45.2 (much lower than best score of 90)
```

### Categorization Example
```
Stock: IMAP
EMA Score: 90, Continuity: 0.94
Latest Score: 86.5, Continuity: 0.92
Average Score: 88, Continuity: 0.91

EMA Mode: ✓ Top Opportunities, ✓ Day Trading
Latest Mode: ✓ Top Opportunities, ✓ Day Trading  
Average Mode: ✓ Top Opportunities, ✓ Day Trading
```

## Notes

- This plan prioritizes simplicity while providing flexibility
- EMA naturally handles the outlier problem (HKAR/BTIB)
- Mode selection allows users to choose their trading style
- All calculations happen server-side for consistency