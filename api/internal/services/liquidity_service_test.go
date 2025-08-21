package services

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLiquidityService_RemoveOutliers(t *testing.T) {
	tests := []struct {
		name  string
		input []float64
		want  []float64
	}{
		{
			name:  "removes high outlier",
			input: []float64{10, 20, 30, 40, 50, 500}, // 500 is outlier
			want:  []float64{10, 20, 30, 40, 50},
		},
		{
			name:  "removes low outlier", 
			input: []float64{1, 50, 60, 70, 80, 90}, // 1 is outlier
			want:  []float64{50, 60, 70, 80, 90},
		},
		{
			name:  "keeps all valid values",
			input: []float64{10, 20, 30, 40, 50},
			want:  []float64{10, 20, 30, 40, 50},
		},
		{
			name:  "handles small dataset",
			input: []float64{10, 20, 30},
			want:  []float64{10, 20, 30}, // Too small for IQR, return as-is
		},
		{
			name:  "handles empty input",
			input: []float64{},
			want:  []float64{},
		},
		{
			name:  "HKAR case - multiple outliers",
			input: []float64{90, 85, 20, 20, 38, 43, 55, 62, 61, 62},
			want:  []float64{20, 20, 38, 43, 55, 62, 61, 62}, // Remove 90, 85
		},
	}

	svc := NewLiquidityService("/tmp", slog.New(slog.NewTextHandler(os.Stdout, nil)))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := svc.removeOutliers(tt.input)
			assert.ElementsMatch(t, tt.want, got)
		})
	}
}

func TestLiquidityService_CalculateEMA20_WithOutliers(t *testing.T) {
	tests := []struct {
		name   string
		values []float64
		want   float64
		delta  float64
	}{
		{
			name:   "HKAR case - high early value should be smoothed",
			values: []float64{90, 20, 20, 38, 43, 55, 62, 61, 62},
			want:   45.0, // Should be much lower than 90 after outlier removal
			delta:  10.0,
		},
		{
			name:   "BTIB case - similar pattern",
			values: []float64{80, 15, 18, 25, 30, 35, 40, 42, 45},
			want:   35.0, // Should be much lower than 80
			delta:  10.0,
		},
		{
			name:   "stable values remain stable",
			values: []float64{50, 52, 51, 53, 50, 52},
			want:   51.5,
			delta:  2.0,
		},
		{
			name:   "empty values",
			values: []float64{},
			want:   0,
			delta:  0.1,
		},
		{
			name:   "single value",
			values: []float64{42},
			want:   42,
			delta:  0.1,
		},
	}

	svc := NewLiquidityService("/tmp", slog.New(slog.NewTextHandler(os.Stdout, nil)))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := svc.calculateEMA20WithOutlierRemoval(tt.values)
			assert.InDelta(t, tt.want, got, tt.delta, "EMA should be within delta")
			
			// Additional assertion: EMA should never be the highest outlier value
			if len(tt.values) > 0 {
				maxVal := tt.values[0]
				for _, v := range tt.values {
					if v > maxVal {
						maxVal = v
					}
				}
				if maxVal > 70 { // If there's a high outlier
					assert.Less(t, got, maxVal*0.7, "EMA should be significantly lower than outliers")
				}
			}
		})
	}
}

func TestLiquidityService_ZeroOutPoorQualityMetrics(t *testing.T) {
	svc := NewLiquidityService("/tmp", slog.New(slog.NewTextHandler(os.Stdout, nil)))

	metrics := &StockMetrics{
		Score:       85.5,
		Continuity:  0.95,
		DailyVolume: 1000000,
		Thresholds: TradingThreshold{
			Conservative: 500000,
			Moderate:     1000000,
			Aggressive:   2000000,
			Optimal:      5000000,
		},
	}

	svc.zeroOutPoorQualityMetrics(metrics)

	// All thresholds should be zero for POOR quality
	assert.Equal(t, 0.0, metrics.Thresholds.Conservative)
	assert.Equal(t, 0.0, metrics.Thresholds.Moderate)
	assert.Equal(t, 0.0, metrics.Thresholds.Aggressive)
	assert.Equal(t, 0.0, metrics.Thresholds.Optimal)
	
	// Score should remain for display but thresholds are zero
	assert.Equal(t, 85.5, metrics.Score)
}

func TestLiquidityService_AggregateStockData_WithMultipleModes(t *testing.T) {
	svc := NewLiquidityService("/tmp", slog.New(slog.NewTextHandler(os.Stdout, nil)))

	entries := []StockRecommendation{
		{
			Symbol:      "TEST",
			Score:       90, // High outlier
			Continuity:  0.5,
			DailyVolume: 500000,
			DataQuality: "GOOD",
			Thresholds: TradingThreshold{
				Conservative: 100000,
				Moderate:     200000,
				Aggressive:   400000,
				Optimal:      800000,
			},
		},
		{
			Symbol:      "TEST",
			Score:       45,
			Continuity:  0.7,
			DailyVolume: 600000,
			DataQuality: "GOOD",
			Thresholds: TradingThreshold{
				Conservative: 150000,
				Moderate:     300000,
				Aggressive:   600000,
				Optimal:      1200000,
			},
		},
		{
			Symbol:      "TEST",
			Score:       50,
			Continuity:  0.8,
			DailyVolume: 700000,
			DataQuality: "GOOD",
			Thresholds: TradingThreshold{
				Conservative: 175000,
				Moderate:     350000,
				Aggressive:   700000,
				Optimal:      1400000,
			},
		},
	}

	result := svc.aggregateStockDataWithModes(entries)

	// Check that we have all metrics
	require.NotNil(t, result.EMAMetrics)
	require.NotNil(t, result.LatestMetrics)
	require.NotNil(t, result.AverageMetrics)

	// EMA should smooth out the high outlier
	assert.Less(t, result.EMAMetrics.Score, 70.0, "EMA should reduce outlier impact")
	
	// Latest should be the last entry's score
	assert.Equal(t, 50.0, result.LatestMetrics.Score)
	
	// Average should be (90+45+50)/3 = 61.67
	assert.InDelta(t, 61.67, result.AverageMetrics.Score, 1.0)
	
	// Best score should remain as the highest
	assert.Equal(t, 90.0, result.Score)
}

func TestLiquidityService_Categorization_DayTrading(t *testing.T) {
	// Test that stocks with continuity >= 0.7 and EMA score >= 50 go to day trading
	stocks := []StockRecommendation{
		{
			Symbol:      "GOOD_DAY_TRADER",
			Score:       90, // Best score (misleading)
			Continuity:  0.75,
			DataQuality: "GOOD",
			EMAMetrics: &StockMetrics{
				Score:       55, // EMA score above 50
				Continuity:  0.75,
			},
		},
		{
			Symbol:      "BAD_SCORE",
			Score:       80, // Best score high
			Continuity:  0.8,
			DataQuality: "GOOD",
			EMAMetrics: &StockMetrics{
				Score:       45, // EMA score below 50
				Continuity:  0.8,
			},
		},
		{
			Symbol:      "LOW_CONTINUITY",
			Score:       85,
			Continuity:  0.6, // Below 0.7
			DataQuality: "GOOD",
			EMAMetrics: &StockMetrics{
				Score:       60,
				Continuity:  0.6,
			},
		},
	}

	// Only GOOD_DAY_TRADER should qualify for day trading
	dayTradingStocks := filterDayTradingStocks(stocks)
	
	assert.Len(t, dayTradingStocks, 1)
	assert.Equal(t, "GOOD_DAY_TRADER", dayTradingStocks[0])
}

// Helper function to simulate categorization logic
func filterDayTradingStocks(stocks []StockRecommendation) []string {
	var dayTrading []string
	for _, stock := range stocks {
		// Use EMA metrics for categorization
		if stock.EMAMetrics != nil && 
		   stock.EMAMetrics.Continuity >= 0.7 && 
		   stock.EMAMetrics.Score >= 50 &&
		   len(dayTrading) < 5 {
			dayTrading = append(dayTrading, stock.Symbol)
		}
	}
	return dayTrading
}