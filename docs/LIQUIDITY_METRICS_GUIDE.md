# ISX Pulse Liquidity Metrics Guide
## Understanding the ISX Hybrid Liquidity Scoring System

### Table of Contents
1. [Overview](#overview)
2. [What is Liquidity?](#what-is-liquidity)
3. [The Four Pillars of ISX Liquidity](#the-four-pillars-of-isx-liquidity)
4. [Understanding the Hybrid Score](#understanding-the-hybrid-score)
5. [How to Use These Metrics](#how-to-use-these-metrics)
6. [Quick Reference](#quick-reference)

---

## Overview

The ISX Pulse Liquidity Scoring System is a sophisticated framework designed specifically for the Iraqi Stock Exchange (ISX). It provides traders and investors with a comprehensive assessment of how easily they can buy or sell stocks without significantly affecting the price.

### Why This Matters
- **Better Trading Decisions**: Know which stocks can handle your trade size
- **Risk Management**: Avoid stocks that might trap your capital
- **Cost Optimization**: Minimize price impact and trading costs
- **Market Timing**: Identify the best stocks for different market conditions

---

## What is Liquidity?

**Liquidity** refers to how quickly and easily you can convert a stock into cash (by selling) or cash into stock (by buying) without causing a significant price movement.

### High Liquidity Means:
- ✅ You can buy/sell quickly
- ✅ Minimal price impact from your trades
- ✅ Narrow bid-ask spreads
- ✅ Consistent daily trading
- ✅ Many buyers and sellers

### Low Liquidity Means:
- ❌ Difficult to buy/sell without moving the price
- ❌ Wide bid-ask spreads
- ❌ Irregular trading (gaps between trading days)
- ❌ Few market participants
- ❌ Risk of being "stuck" in a position

---

## The Four Pillars of ISX Liquidity

Our hybrid scoring system evaluates liquidity across four key dimensions:

### 1. 📊 Price Impact (35% weight)
**What it measures**: How much the stock price moves when you trade

**Based on**: Amihud ILLIQ measure - the relationship between price changes and trading volume

**What it tells you**:
- Lower score = Your trades move the price significantly
- Higher score = You can trade without disturbing the price
- Critical for large trades or institutional investors

**Example**: 
- Stock A with high price impact score (80): Trading 10M IQD moves price by ~0.5%
- Stock B with low price impact score (20): Trading 10M IQD moves price by ~3%

### 2. 💰 Trading Value (30% weight)
**What it measures**: Average daily trading value in Iraqi Dinars (IQD)

**Calculated as**: 60-day Simple Moving Average including non-trading days

**What it tells you**:
- Higher values = More money flowing through the stock daily
- Lower values = Limited trading interest
- Indicates market depth and institutional interest

**Example**:
- Bank stock: 500M IQD daily average = High liquidity
- Small industrial: 5M IQD daily average = Low liquidity

### 3. 📅 Trading Continuity (20% weight)
**What it measures**: How consistently the stock trades

**Calculated as**: Percentage of days with trading activity in the 60-day window

**What it tells you**:
- 90%+ continuity = Trades almost every day
- 50% continuity = Trades only half the days
- <30% continuity = Sporadic, unreliable trading

**Why it matters**:
- Continuous trading = Easy entry and exit
- Gaps in trading = Risk of being unable to sell when needed

### 4. 📈 Spread Proxy (15% weight)
**What it measures**: Estimated bid-ask spread using high-low price data

**Based on**: Corwin-Schultz spread estimator

**What it tells you**:
- Lower spreads = Lower transaction costs
- Higher spreads = More expensive to trade
- Indicates market maker activity and competition

---

## Understanding the Hybrid Score

The **ISX Hybrid Liquidity Score** combines all four pillars into a single number from 0-100:

### Score Interpretation:

#### 🟢 Excellent Liquidity (80-100)
- **Characteristics**: 
  - Can handle large trades (>50M IQD)
  - Minimal price impact
  - Trades every day
  - Tight spreads
- **Best for**: Institutional investors, large trades, active trading
- **Examples**: Major banks like BBOB, BIME

#### 🔵 Good Liquidity (60-79)
- **Characteristics**:
  - Can handle moderate trades (10-50M IQD)
  - Acceptable price impact
  - Regular trading (>80% of days)
  - Reasonable spreads
- **Best for**: Active traders, medium-sized positions
- **Examples**: Large industrial companies

#### 🟡 Moderate Liquidity (40-59)
- **Characteristics**:
  - Can handle small trades (5-10M IQD)
  - Noticeable price impact
  - Some trading gaps
  - Wider spreads
- **Best for**: Small investors, long-term positions
- **Examples**: Mid-cap companies

#### 🟠 Low Liquidity (20-39)
- **Characteristics**:
  - Limited to very small trades (<5M IQD)
  - Significant price impact
  - Frequent trading gaps
  - Wide spreads
- **Best for**: Very patient investors, speculative positions
- **Caution**: Exit strategy crucial

#### 🔴 Very Low Liquidity (0-19)
- **Characteristics**:
  - Extremely difficult to trade
  - Severe price impact
  - Rare trading days
  - Very wide spreads
- **Best for**: Avoid unless special situation
- **Warning**: High risk of being trapped

---

## How to Use These Metrics

### For Different Trading Strategies:

#### 1. Day Trading
**Required Score**: 70+
**Focus on**: 
- High continuity (>90%)
- Low spread proxy
- High daily value

#### 2. Swing Trading (Few Days to Weeks)
**Required Score**: 50+
**Focus on**:
- Moderate continuity (>70%)
- Acceptable price impact
- Consistent patterns

#### 3. Long-term Investment
**Required Score**: 30+ (can tolerate lower)
**Focus on**:
- Company fundamentals over liquidity
- Plan exit strategy carefully
- Consider accumulating slowly

#### 4. Large Block Trades
**Required Score**: 80+
**Focus on**:
- Price impact score
- Safe trading thresholds
- Consider splitting orders

### Reading Safe Trading Values

Each stock provides four trading thresholds:

1. **Conservative (0.5% impact)**: Safest trade size
2. **Moderate (1% impact)**: Balanced risk/reward
3. **Aggressive (2% impact)**: Higher impact acceptable
4. **Optimal**: Recommended size based on typical patterns

**Example for BBOB (Score: 89)**:
- Conservative: 15M IQD (virtually no price movement)
- Moderate: 30M IQD (slight price movement)
- Aggressive: 60M IQD (noticeable but acceptable movement)
- Optimal: 35M IQD (best balance)

---

## Quick Reference

### Liquidity Score Cheat Sheet

| Score Range | Quality | Max Trade Size | Trading Frequency | Best Use Case |
|------------|---------|---------------|-------------------|---------------|
| 80-100 | Excellent | >50M IQD | Daily | Institutional/Active |
| 60-79 | Good | 10-50M IQD | 90% of days | Regular Trading |
| 40-59 | Moderate | 5-10M IQD | 70% of days | Small Positions |
| 20-39 | Low | <5M IQD | 50% of days | Long-term Only |
| 0-19 | Very Low | <1M IQD | Sporadic | Avoid |

### Red Flags to Watch 🚩
- Score below 30
- Continuity below 50%
- Sudden drops in score (>20 points)
- Wide spread proxy (>2%)
- Daily value below 10M IQD

### Green Flags to Look For ✅
- Score above 70
- Continuity above 85%
- Stable or improving scores
- Narrow spreads (<0.5%)
- Daily value above 100M IQD

---

## Practical Examples

### Example 1: Choosing Between Two Bank Stocks

**Stock A: BBOB**
- Hybrid Score: 89
- Continuity: 95%
- Daily Value: 250M IQD
- **Decision**: Excellent for any trading strategy

**Stock B: BANK**
- Hybrid Score: 45
- Continuity: 60%
- Daily Value: 25M IQD
- **Decision**: Only for small, patient positions

### Example 2: Planning a 100M IQD Investment

**Approach**:
1. Filter stocks with score >70
2. Check safe trading values
3. Split across 3-4 stocks
4. Enter positions over 2-3 days
5. Monitor liquidity scores for changes

---

## Key Takeaways

1. **Liquidity = Flexibility**: Higher scores mean easier entry/exit
2. **Match Strategy to Score**: Day traders need 70+, investors can accept 30+
3. **Respect the Thresholds**: Stay within safe trading values
4. **Monitor Changes**: Liquidity can improve or deteriorate
5. **Diversify by Liquidity**: Mix high and moderate liquidity stocks

---

## Frequently Asked Questions

**Q: Can I trade stocks with scores below 30?**
A: Yes, but only with small amounts and a long-term horizon. Have an exit strategy.

**Q: Why do some stocks have 0 continuity but still trade?**
A: They may have just resumed trading after suspension or are newly listed.

**Q: Should I only buy stocks with 80+ scores?**
A: No, but understand the trade-offs. Lower liquidity might offer value opportunities.

**Q: How often are scores updated?**
A: Daily, based on a rolling 60-day window.

**Q: What causes liquidity scores to change?**
A: Market interest, news, earnings, regulatory changes, or broader market conditions.

---

*Last Updated: August 2025*
*ISX Pulse - Professional Financial Intelligence Platform*