# Liquidity Report Columns Reference
## Complete Guide to Understanding Every Column in ISX Liquidity Reports

### Table of Contents
1. [Report Structure](#report-structure)
2. [Core Identification Columns](#core-identification-columns)
3. [Raw Liquidity Metrics](#raw-liquidity-metrics)
4. [Scaled & Normalized Metrics](#scaled--normalized-metrics)
5. [Trading Recommendations](#trading-recommendations)
6. [Statistical Measures](#statistical-measures)
7. [Column Quick Reference](#column-quick-reference)
8. [How to Read a Report Row](#how-to-read-a-report-row)

---

## Report Structure

The ISX Liquidity Report contains approximately 25 columns organized into logical groups:

```
[Identity] → [Raw Metrics] → [Scaled Values] → [Final Scores] → [Trading Guidance]
```

Each row represents one stock's liquidity metrics for a specific date, calculated over a 60-day rolling window.

---

## Core Identification Columns

### 1. **Symbol** 
- **Type**: Text (e.g., "BBOB", "BIME")
- **What it is**: Stock ticker symbol on ISX
- **Use**: Identify which stock the metrics apply to

### 2. **Date**
- **Type**: Date (YYYY-MM-DD)
- **What it is**: The date these metrics were calculated for
- **Use**: Track liquidity changes over time
- **Note**: Uses rolling 60-day window ending on this date

### 3. **Window**
- **Type**: Text (usually "60d")
- **What it is**: The time period used for calculations
- **Use**: Confirms calculation period (always 60 days for ISX)

---

## Raw Liquidity Metrics

These are the fundamental measurements before any scaling or normalization:

### 4. **ILLIQ (Amihud Illiquidity)**
- **Type**: Decimal number
- **Range**: 0 to 1000+ (lower is better)
- **What it measures**: Price impact per million IQD traded
- **Formula**: Average(|Daily Return| / Daily Value in Millions)
- **Interpretation**:
  - < 0.1: Extremely liquid
  - 0.1-1.0: Good liquidity
  - 1.0-10: Moderate liquidity
  - > 10: Poor liquidity
- **Example**: ILLIQ = 0.5 means each million IQD traded moves price by ~0.5%

### 5. **Value (Average Daily Trading Value)**
- **Type**: Number in IQD
- **Range**: 0 to billions
- **What it measures**: Average money traded daily over 60 days
- **Calculation**: SMA(60) including non-trading days as zeros
- **Interpretation**:
  - > 500M IQD: Very high activity
  - 100-500M IQD: High activity
  - 10-100M IQD: Moderate activity
  - < 10M IQD: Low activity
- **Why it matters**: Higher value = deeper market = easier to trade

### 6. **Continuity**
- **Type**: Percentage (0.0 to 1.0)
- **Range**: 0% to 100%
- **What it measures**: Fraction of days with trading activity
- **Formula**: Trading Days / Total Days in Window
- **Interpretation**:
  - > 0.9 (90%): Excellent - trades almost daily
  - 0.7-0.9: Good - regular trading
  - 0.5-0.7: Fair - some gaps
  - < 0.5: Poor - irregular trading
- **Example**: 0.85 = Stock traded on 51 out of 60 days

### 7. **Continuity_NL (Non-Linear Continuity)**
- **Type**: Decimal (0.0 to 1.0)
- **What it is**: Transformed continuity emphasizing extremes
- **Purpose**: Penalizes very low continuity more severely
- **Formula**: Continuity^0.5 (square root transformation)
- **Use**: Better differentiates between sporadic traders

### 8. **Spread_Proxy**
- **Type**: Percentage
- **Range**: 0% to 5%+
- **What it measures**: Estimated bid-ask spread
- **Based on**: Corwin-Schultz high-low spread estimator
- **Interpretation**:
  - < 0.5%: Excellent - very tight spreads
  - 0.5-1%: Good - reasonable costs
  - 1-2%: Moderate - noticeable costs
  - > 2%: Poor - expensive to trade
- **Impact**: Direct trading cost on round-trip trades

---

## Scaled & Normalized Metrics

These transform raw metrics to 0-100 scale for comparison across stocks:

### 9. **ILLIQ_Scaled**
- **Type**: Score (0-100)
- **Range**: 0 (worst) to 100 (best)
- **What it is**: Normalized and inverted ILLIQ score
- **Transformation**: 
  1. Log transform to handle outliers
  2. Invert (low ILLIQ → high score)
  3. Scale cross-sectionally to 0-100
- **Interpretation**:
  - 80-100: Minimal price impact
  - 60-80: Low price impact
  - 40-60: Moderate price impact
  - 20-40: High price impact
  - 0-20: Severe price impact

### 10. **Value_Scaled**
- **Type**: Score (0-100)
- **Range**: 0 (worst) to 100 (best)
- **What it is**: Normalized trading value score
- **Transformation**: Log transform + robust scaling
- **Interpretation**:
  - 80-100: Very high trading activity
  - 60-80: High activity
  - 40-60: Moderate activity
  - 20-40: Low activity
  - 0-20: Very low activity

### 11. **Continuity_Scaled**
- **Type**: Score (0-100)
- **Range**: 0 (worst) to 100 (best)
- **What it is**: Normalized continuity score
- **Transformation**: Direct scaling of continuity_NL
- **Interpretation**:
  - 80-100: Trades almost every day
  - 60-80: Regular trading
  - 40-60: Moderate gaps
  - 20-40: Significant gaps
  - 0-20: Rare trading

### 12. **Spread_Scaled**
- **Type**: Score (0-100)
- **Range**: 0 (worst) to 100 (best)
- **What it is**: Normalized and inverted spread score
- **Note**: Inverted so higher = better (tighter spreads)
- **Interpretation**:
  - 80-100: Very tight spreads
  - 60-80: Good spreads
  - 40-60: Moderate spreads
  - 20-40: Wide spreads
  - 0-20: Very wide spreads

---

## Final Composite Scores

### 13. **Hybrid_Score** ⭐ (Most Important)
- **Type**: Score (0-100)
- **Range**: 0 (worst) to 100 (best)
- **What it is**: Weighted combination of all scaled metrics
- **Formula**: 
  ```
  0.35 × ILLIQ_Scaled + 
  0.30 × Value_Scaled + 
  0.20 × Continuity_Scaled + 
  0.15 × Spread_Scaled
  ```
- **Interpretation**:
  - 80-100: Excellent liquidity
  - 60-80: Good liquidity
  - 40-60: Moderate liquidity
  - 20-40: Poor liquidity
  - 0-20: Very poor liquidity
- **Use**: Primary metric for stock selection

### 14. **Hybrid_Rank**
- **Type**: Integer (1, 2, 3...)
- **What it is**: Relative ranking among all stocks on that date
- **Interpretation**:
  - 1 = Most liquid stock
  - 2 = Second most liquid
  - etc.
- **Use**: Quick comparison between stocks

---

## Trading Recommendations

### 15-18. **Safe Trading Values**

These columns tell you maximum trade sizes for different risk tolerances:

### 15. **Safe_Value_0_5** (Conservative)
- **Type**: Amount in IQD
- **What it is**: Max trade size for 0.5% price impact
- **Use**: Ultra-safe trading, institutional standards
- **Example**: 15,000,000 = Trade up to 15M IQD with minimal impact

### 16. **Safe_Value_1_0** (Moderate)
- **Type**: Amount in IQD
- **What it is**: Max trade size for 1% price impact
- **Use**: Balanced approach for regular trading
- **Example**: 30,000,000 = Trade up to 30M IQD with acceptable impact

### 17. **Safe_Value_2_0** (Aggressive)
- **Type**: Amount in IQD
- **What it is**: Max trade size for 2% price impact
- **Use**: When willing to accept noticeable price movement
- **Example**: 60,000,000 = Trade up to 60M IQD knowing price will move

### 18. **Optimal_Trade_Size**
- **Type**: Amount in IQD
- **What it is**: Recommended trade size balancing impact and efficiency
- **Calculation**: Based on typical daily patterns and volatility
- **Use**: Default position sizing
- **Usually**: Between conservative and moderate thresholds

---

## Statistical Measures

### 19. **Trading_Days**
- **Type**: Integer (0-60)
- **What it is**: Count of days with actual trading
- **Use**: Assess reliability of other metrics
- **Note**: Low count = less reliable statistics

### 20. **Total_Days**
- **Type**: Integer (always 60)
- **What it is**: Days in calculation window
- **Use**: Denominator for continuity calculation

### 21. **Impact_Penalty**
- **Type**: Multiplier (1.0+)
- **Range**: 1.0 to 10.0
- **What it is**: Penalty applied for low trading activity
- **Purpose**: Adjusts ILLIQ score for inactive stocks
- **Interpretation**:
  - 1.0: No penalty (active stock)
  - 2.0: 2x penalty (moderate gaps)
  - 5.0+: Severe penalty (very inactive)

### 22. **Value_Penalty**
- **Type**: Multiplier (1.0+)
- **What it is**: Similar penalty for value calculations
- **Note**: Usually same as Impact_Penalty

### 23. **Activity_Score**
- **Type**: Score (0.0-1.0)
- **What it is**: Unified measure of trading activity
- **Formula**: Combination of continuity and volume metrics
- **Use**: Quick assessment of overall activity level

### 24. **Avg_Return** (Often 0)
- **Type**: Percentage
- **What it is**: Average daily return over window
- **Note**: Not used in final scoring

### 25. **Return_Volatility** (Often 0)
- **Type**: Percentage
- **What it is**: Standard deviation of returns
- **Note**: Not used in final scoring

---

## Column Quick Reference

| Column | Type | Range | Higher is Better? | Weight in Score |
|--------|------|-------|------------------|-----------------|
| Symbol | Text | - | - | - |
| Date | Date | - | - | - |
| ILLIQ | Number | 0-1000 | ❌ No | 35% (inverted) |
| Value | IQD | 0-∞ | ✅ Yes | 30% |
| Continuity | % | 0-1 | ✅ Yes | 20% |
| Spread_Proxy | % | 0-5 | ❌ No | 15% (inverted) |
| **Hybrid_Score** | Score | 0-100 | ✅ Yes | **FINAL** |
| Hybrid_Rank | Rank | 1-N | ❌ No (1 is best) | - |
| Safe_Value_0_5 | IQD | 0-∞ | ✅ Yes | - |
| Optimal_Trade | IQD | 0-∞ | ✅ Yes | - |

---

## How to Read a Report Row

### Example Row Breakdown:

```
Symbol: BBOB
Date: 2025-08-14
ILLIQ: 0.25
Value: 450,000,000
Continuity: 0.92
Spread_Proxy: 0.4%
ILLIQ_Scaled: 85
Value_Scaled: 88
Continuity_Scaled: 90
Spread_Scaled: 82
Hybrid_Score: 86.5
Hybrid_Rank: 2
Safe_Value_0_5: 20,000,000
Optimal_Trade_Size: 35,000,000
```

**Reading This Row**:
1. **Stock**: BBOB on August 14, 2025
2. **Liquidity**: Excellent (score 86.5, ranked #2)
3. **Why Good**: 
   - Low price impact (ILLIQ 0.25)
   - High daily value (450M IQD)
   - Trades 92% of days
   - Tight spreads (0.4%)
4. **Trading Guidance**:
   - Can safely trade 20M IQD (minimal impact)
   - Optimal trades around 35M IQD
   - Suitable for all trading strategies

---

## Practical Tips

### For Quick Analysis:
1. **Start with Hybrid_Score**: Above 60 is generally good
2. **Check Continuity**: Should be above 70% for active trading
3. **Verify Safe Values**: Match your intended trade size
4. **Compare Ranks**: Lower rank number = better liquidity

### Red Flags in Data:
- 🚩 Hybrid_Score < 30
- 🚩 Continuity < 0.5
- 🚩 ILLIQ > 10
- 🚩 Trading_Days < 30
- 🚩 All safe values < 1M IQD

### For Different Users:

**Day Traders**: Focus on
- Hybrid_Score > 70
- Continuity > 0.85
- Spread_Scaled > 70

**Long-term Investors**: Focus on
- Value_Scaled (market interest)
- Safe_Value_1_0 (exit capability)
- Trend in scores over time

**Institutional Traders**: Focus on
- Safe_Value_0_5 (large positions)
- ILLIQ_Scaled (price impact)
- Hybrid_Rank (relative positioning)

---

*Last Updated: August 2025*
*ISX Pulse - Making Iraqi Markets Transparent*