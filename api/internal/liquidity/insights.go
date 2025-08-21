package liquidity

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"
)

// TradingRecommendation represents actionable trading advice for a stock
type TradingRecommendation struct {
	Symbol            string
	HybridScore      float64  // Best score achieved
	DataQuality      string
	
	// Moving average scores
	EMA20Score       float64  // 20-period exponential moving average
	LatestScore      float64  // Most recent score
	
	// All trading thresholds with descriptions
	ConservativeTrade float64  // Safe_Trade_0.5% - For risk-averse traders
	ModerateTrade    float64  // Safe_Trade_1% - Balanced approach
	AggressiveTrade  float64  // Safe_Trade_2% - For experienced traders
	OptimalTrade     float64  // Algorithm-recommended size
	
	// Context for decision making
	DailyVolume      float64
	Continuity       float64
	SpreadCost       float64
	TradingDays      int
	
	// Actionable recommendation
	Recommendation   string   // "BUY_LARGE", "DAY_TRADE", "AVOID", etc.
	Rationale       string   // Explanation of recommendation
}

// LiquidityInsights contains analyzed and categorized liquidity information
type LiquidityInsights struct {
	GeneratedAt       time.Time
	MarketHealthScore float64  // Average hybrid score of top 20 stocks
	TotalStocks       int
	HighQualityStocks int      // Stocks with good data quality
	
	// Categorized recommendations
	TopOpportunities  []TradingRecommendation // Top 10 most liquid
	HighRisk         []TradingRecommendation // Data quality POOR or very low scores
	BestForLargeTrades []TradingRecommendation // Highest optimal trade values
	BestForDayTrading []TradingRecommendation // High continuity stocks
	
	// Market analysis
	AverageContinuity float64
	AverageSpread    float64
	MedianDailyVolume float64
}

// GenerateInsights analyzes liquidity scores and creates actionable insights
func GenerateInsights(liquidityCSVPath string, outputDir string) error {
	// Read the liquidity scores CSV
	stocks, err := readLiquidityScores(liquidityCSVPath)
	if err != nil {
		return fmt.Errorf("failed to read liquidity scores: %w", err)
	}
	
	if len(stocks) == 0 {
		return fmt.Errorf("no liquidity data found")
	}
	
	// Generate insights
	insights := analyzeLiquidity(stocks)
	
	// Save insights to CSV
	outputPath := filepath.Join(outputDir, fmt.Sprintf("liquidity_insights_%s.csv", 
		time.Now().Format("2006-01-02")))
	
	if err := saveInsightsToCSV(insights, outputPath); err != nil {
		return fmt.Errorf("failed to save insights: %w", err)
	}
	
	return nil
}

// calculateEMA20 calculates 20-period exponential moving average
func calculateEMA20(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	
	// EMA smoothing factor for 20 periods
	alpha := 2.0 / 21.0
	ema := values[0]
	
	for i := 1; i < len(values); i++ {
		ema = alpha*values[i] + (1-alpha)*ema
	}
	
	return ema
}

// aggregateSymbolData aggregates multiple date entries for a single symbol
func aggregateSymbolData(entries []TradingRecommendation) TradingRecommendation {
	if len(entries) == 0 {
		return TradingRecommendation{}
	}
	
	// Collect scores for EMA calculation
	scores := make([]float64, 0, len(entries))
	totalVolume := 0.0
	totalContinuity := 0.0
	totalSpread := 0.0
	bestIdx := 0
	
	for i, e := range entries {
		scores = append(scores, e.HybridScore)
		totalVolume += e.DailyVolume
		totalContinuity += e.Continuity
		totalSpread += e.SpreadCost
		
		// Find entry with best score
		if e.HybridScore > entries[bestIdx].HybridScore {
			bestIdx = i
		}
	}
	
	// Start with the best scoring entry
	result := entries[bestIdx]
	
	// Calculate aggregated values
	result.EMA20Score = calculateEMA20(scores)
	result.LatestScore = entries[len(entries)-1].HybridScore
	result.HybridScore = entries[bestIdx].HybridScore // Keep best score
	
	// Average the volume and continuity across all dates
	if len(entries) > 0 {
		result.DailyVolume = totalVolume / float64(len(entries))
		result.Continuity = totalContinuity / float64(len(entries))
		result.SpreadCost = totalSpread / float64(len(entries))
	}
	
	// Update trading days to reflect actual data
	result.TradingDays = len(entries)
	
	return result
}

// readLiquidityScores reads the liquidity CSV and returns parsed recommendations
func readLiquidityScores(csvPath string) ([]TradingRecommendation, error) {
	file, err := os.Open(csvPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	
	if len(records) < 2 {
		return nil, fmt.Errorf("insufficient data in CSV")
	}
	
	// Parse header to find column indices
	header := records[0]
	indices := make(map[string]int)
	for i, col := range header {
		indices[col] = i
	}
	
	// Required columns
	required := []string{"Symbol", "Hybrid_Score", "Data_Quality", "Safe_Trade_0.5%", 
		"Safe_Trade_1%", "Safe_Trade_2%", "Optimal_Trade", "Continuity_Scaled", 
		"Spread_Scaled", "Trading_Days", "Value_Raw"}
	
	for _, col := range required {
		if _, ok := indices[col]; !ok {
			return nil, fmt.Errorf("missing required column: %s", col)
		}
	}
	
	// Parse data rows
	// Group all entries by symbol for aggregation
	symbolData := make(map[string][]TradingRecommendation)
	
	for i := 1; i < len(records); i++ {
		row := records[i]
		if len(row) < len(header) {
			continue
		}
		
		symbol := row[indices["Symbol"]]
		
		// Parse numeric values
		hybridScore, _ := strconv.ParseFloat(row[indices["Hybrid_Score"]], 64)
		conservative, _ := strconv.ParseFloat(row[indices["Safe_Trade_0.5%"]], 64)
		moderate, _ := strconv.ParseFloat(row[indices["Safe_Trade_1%"]], 64)
		aggressive, _ := strconv.ParseFloat(row[indices["Safe_Trade_2%"]], 64)
		optimal, _ := strconv.ParseFloat(row[indices["Optimal_Trade"]], 64)
		continuity, _ := strconv.ParseFloat(row[indices["Continuity_Scaled"]], 64)
		spread, _ := strconv.ParseFloat(row[indices["Spread_Scaled"]], 64)
		tradingDays, _ := strconv.Atoi(row[indices["Trading_Days"]])
		dailyVolume, _ := strconv.ParseFloat(row[indices["Value_Raw"]], 64)
		
		rec := TradingRecommendation{
			Symbol:            symbol,
			HybridScore:      hybridScore,
			DataQuality:      row[indices["Data_Quality"]],
			ConservativeTrade: conservative,
			ModerateTrade:    moderate,
			AggressiveTrade:  aggressive,
			OptimalTrade:     optimal,
			DailyVolume:      dailyVolume,
			Continuity:       continuity / 100.0, // Convert to decimal
			SpreadCost:       spread,
			TradingDays:      tradingDays,
		}
		
		// Collect all entries for this symbol
		symbolData[symbol] = append(symbolData[symbol], rec)
	}
	
	// Aggregate data for each symbol
	var stocks []TradingRecommendation
	for _, entries := range symbolData {
		// Aggregate all date entries for this symbol
		aggregated := aggregateSymbolData(entries)
		
		// Generate recommendation based on aggregated data
		aggregated.Recommendation, aggregated.Rationale = generateRecommendation(aggregated)
		
		stocks = append(stocks, aggregated)
	}
	
	return stocks, nil
}

// generateRecommendation creates actionable advice based on metrics
func generateRecommendation(stock TradingRecommendation) (string, string) {
	// Avoid stocks with poor data quality
	if stock.DataQuality == "POOR" {
		return "AVOID", fmt.Sprintf("Insufficient trading data (%d days), unreliable metrics", stock.TradingDays)
	}
	
	// High liquidity stocks
	if stock.HybridScore >= 85 {
		if stock.OptimalTrade >= 10_000_000 { // 10M IQD
			return "BUY_LARGE", fmt.Sprintf("Excellent liquidity (%.1f score), supports large trades up to %.0fM IQD", 
				stock.HybridScore, stock.OptimalTrade/1_000_000)
		}
		return "BUY", fmt.Sprintf("High liquidity (%.1f score), good for active trading", stock.HybridScore)
	}
	
	// Good for day trading
	if stock.Continuity >= 70 && stock.HybridScore >= 60 {
		return "DAY_TRADE", fmt.Sprintf("High continuity (%.0f%%), suitable for frequent trading", stock.Continuity)
	}
	
	// Medium liquidity
	if stock.HybridScore >= 50 {
		return "HOLD", fmt.Sprintf("Moderate liquidity (%.1f score), trade carefully within %.0fK IQD", 
			stock.HybridScore, stock.OptimalTrade/1000)
	}
	
	// Low liquidity
	if stock.HybridScore >= 30 {
		return "CAUTION", fmt.Sprintf("Low liquidity (%.1f score), limit trades to %.0fK IQD", 
			stock.HybridScore, stock.ConservativeTrade/1000)
	}
	
	// Very low liquidity
	return "AVOID", fmt.Sprintf("Very low liquidity (%.1f score), high price impact risk", stock.HybridScore)
}

// analyzeLiquidity generates insights from stock data
func analyzeLiquidity(stocks []TradingRecommendation) LiquidityInsights {
	insights := LiquidityInsights{
		GeneratedAt: time.Now(),
		TotalStocks: len(stocks),
	}
	
	// Sort by EMA20 score as primary, hybrid score as secondary
	sort.Slice(stocks, func(i, j int) bool {
		// Primary sort by EMA20
		if stocks[i].EMA20Score != stocks[j].EMA20Score {
			return stocks[i].EMA20Score > stocks[j].EMA20Score
		}
		// Secondary sort by best hybrid score
		return stocks[i].HybridScore > stocks[j].HybridScore
	})
	
	// Calculate market health (average of top 20 or all if less)
	topN := 20
	if len(stocks) < topN {
		topN = len(stocks)
	}
	
	var totalScore, totalContinuity, totalSpread float64
	var volumes []float64
	highQualityCount := 0
	
	for i, stock := range stocks {
		if i < topN {
			totalScore += stock.HybridScore
		}
		
		if stock.DataQuality != "POOR" {
			highQualityCount++
			totalContinuity += stock.Continuity
			totalSpread += stock.SpreadCost
			volumes = append(volumes, stock.DailyVolume)
		}
	}
	
	insights.MarketHealthScore = totalScore / float64(topN)
	insights.HighQualityStocks = highQualityCount
	
	if highQualityCount > 0 {
		insights.AverageContinuity = totalContinuity / float64(highQualityCount)
		insights.AverageSpread = totalSpread / float64(highQualityCount)
	}
	
	// Calculate median volume
	if len(volumes) > 0 {
		sort.Float64s(volumes)
		mid := len(volumes) / 2
		if len(volumes)%2 == 0 {
			insights.MedianDailyVolume = (volumes[mid-1] + volumes[mid]) / 2
		} else {
			insights.MedianDailyVolume = volumes[mid]
		}
	}
	
	// Categorize stocks
	for _, stock := range stocks {
		// Top opportunities (top 10)
		if len(insights.TopOpportunities) < 10 && stock.HybridScore >= 50 {
			insights.TopOpportunities = append(insights.TopOpportunities, stock)
		}
		
		// High risk
		if stock.DataQuality == "POOR" || stock.HybridScore < 30 {
			if len(insights.HighRisk) < 10 {
				insights.HighRisk = append(insights.HighRisk, stock)
			}
		}
		
		// Best for large trades (high optimal trade value)
		if stock.OptimalTrade >= 5_000_000 && stock.DataQuality != "POOR" {
			if len(insights.BestForLargeTrades) < 5 {
				insights.BestForLargeTrades = append(insights.BestForLargeTrades, stock)
			}
		}
		
		// Best for day trading (high continuity)
		if stock.Continuity >= 70 && stock.HybridScore >= 50 {
			if len(insights.BestForDayTrading) < 5 {
				insights.BestForDayTrading = append(insights.BestForDayTrading, stock)
			}
		}
	}
	
	return insights
}

// saveInsightsToCSV saves the insights to a CSV file
func saveInsightsToCSV(insights LiquidityInsights, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	writer := csv.NewWriter(file)
	defer writer.Flush()
	
	// Write metadata section
	writer.Write([]string{"ISX Liquidity Insights Report"})
	writer.Write([]string{"Generated:", insights.GeneratedAt.Format("2006-01-02 15:04:05")})
	writer.Write([]string{"Market Health Score:", fmt.Sprintf("%.1f", insights.MarketHealthScore)})
	writer.Write([]string{"Total Stocks Analyzed:", strconv.Itoa(insights.TotalStocks)})
	writer.Write([]string{"High Quality Stocks:", strconv.Itoa(insights.HighQualityStocks)})
	writer.Write([]string{"Average Continuity:", fmt.Sprintf("%.1f%%", insights.AverageContinuity)})
	writer.Write([]string{"Median Daily Volume:", fmt.Sprintf("%.0f IQD", insights.MedianDailyVolume)})
	writer.Write([]string{""}) // Empty line
	
	// Write top opportunities
	writer.Write([]string{"TOP TRADING OPPORTUNITIES"})
	writer.Write([]string{"Symbol", "EMA20", "Latest", "Best", "Conservative (0.5%)", "Moderate (1%)", 
		"Aggressive (2%)", "Optimal", "Continuity", "Daily Vol", "Action", "Rationale"})
	
	for _, stock := range insights.TopOpportunities {
		writer.Write([]string{
			stock.Symbol,
			fmt.Sprintf("%.1f", stock.EMA20Score),
			fmt.Sprintf("%.1f", stock.LatestScore),
			fmt.Sprintf("%.1f", stock.HybridScore),
			fmt.Sprintf("%.0f", stock.ConservativeTrade),
			fmt.Sprintf("%.0f", stock.ModerateTrade),
			fmt.Sprintf("%.0f", stock.AggressiveTrade),
			fmt.Sprintf("%.0f", stock.OptimalTrade),
			fmt.Sprintf("%.1f%%", stock.Continuity*100),
			fmt.Sprintf("%.0f", stock.DailyVolume),
			stock.Recommendation,
			stock.Rationale,
		})
	}
	writer.Write([]string{""})
	
	// Write best for large trades
	writer.Write([]string{"BEST FOR LARGE TRADES"})
	writer.Write([]string{"Symbol", "Score", "Optimal Trade Size", "Daily Volume", "Recommendation"})
	
	for _, stock := range insights.BestForLargeTrades {
		writer.Write([]string{
			stock.Symbol,
			fmt.Sprintf("%.1f", stock.HybridScore),
			fmt.Sprintf("%.0f", stock.OptimalTrade),
			fmt.Sprintf("%.0f", stock.DailyVolume),
			stock.Rationale,
		})
	}
	writer.Write([]string{""})
	
	// Write best for day trading
	writer.Write([]string{"BEST FOR DAY TRADING"})
	writer.Write([]string{"Symbol", "Score", "Continuity", "Spread Cost", "Recommendation"})
	
	for _, stock := range insights.BestForDayTrading {
		writer.Write([]string{
			stock.Symbol,
			fmt.Sprintf("%.1f", stock.HybridScore),
			fmt.Sprintf("%.0f%%", stock.Continuity),
			fmt.Sprintf("%.1f%%", stock.SpreadCost),
			stock.Rationale,
		})
	}
	writer.Write([]string{""})
	
	// Write high risk stocks
	writer.Write([]string{"HIGH RISK - AVOID"})
	writer.Write([]string{"Symbol", "Score", "Data Quality", "Trading Days", "Warning"})
	
	for _, stock := range insights.HighRisk {
		writer.Write([]string{
			stock.Symbol,
			fmt.Sprintf("%.1f", stock.HybridScore),
			stock.DataQuality,
			strconv.Itoa(stock.TradingDays),
			stock.Rationale,
		})
	}
	
	// Write trading guidelines
	writer.Write([]string{""})
	writer.Write([]string{"TRADING GUIDELINES"})
	writer.Write([]string{"Conservative (0.5% impact):", "Minimal market impact, best for large institutional orders"})
	writer.Write([]string{"Moderate (1% impact):", "Balanced approach for active traders"})
	writer.Write([]string{"Aggressive (2% impact):", "Faster execution but higher market impact"})
	writer.Write([]string{"Optimal:", "Algorithm-recommended size considering all factors"})
	
	return nil
}

// GetLatestInsights reads the most recent insights CSV and returns structured data
func GetLatestInsights(reportsDir string) (*LiquidityInsights, error) {
	// Find the most recent insights file
	pattern := filepath.Join(reportsDir, "liquidity_insights_*.csv")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	
	if len(files) == 0 {
		return nil, fmt.Errorf("no insights files found")
	}
	
	// Sort to get the most recent
	sort.Strings(files)
	// latestFile := files[len(files)-1]
	
	// For now, return a simple structure
	// In production, we would parse the CSV back into the struct
	// TODO: Implement parsing of the insights CSV file
	insights := &LiquidityInsights{
		GeneratedAt: time.Now(),
		MarketHealthScore: 72.5, // Placeholder
		TotalStocks: 80,
		HighQualityStocks: 45,
		AverageContinuity: 55.2,
		AverageSpread: 2.3,
		MedianDailyVolume: 5_000_000,
	}
	
	// Parse the actual file if needed
	// This is simplified for now
	
	return insights, nil
}