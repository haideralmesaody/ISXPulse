# ISX Liquidity Dashboard Guide
## Visual Interface User Manual

### Table of Contents
1. [Dashboard Overview](#dashboard-overview)
2. [Navigation and Layout](#navigation-and-layout)
3. [Understanding the Visual Elements](#understanding-the-visual-elements)
4. [Interactive Features](#interactive-features)
5. [Reading the Trading Cards](#reading-the-trading-cards)
6. [Market Health Indicators](#market-health-indicators)
7. [Using Recommendations](#using-recommendations)
8. [Customization and Filters](#customization-and-filters)
9. [Quick Start Tutorial](#quick-start-tutorial)

---

## Dashboard Overview

The ISX Liquidity Dashboard is your command center for understanding market liquidity at a glance. It transforms complex liquidity calculations into actionable visual insights.

### Key Features:
- 📊 Real-time liquidity scores for all ISX stocks
- 🎯 AI-powered trading recommendations
- 📈 Market health monitoring
- 💡 Smart position sizing guidance
- 🔍 Advanced filtering and search

### Access Path:
```
ISX Pulse → Navigation Menu → Liquidity → Dashboard Opens
```

---

## Navigation and Layout

### Main Sections:

```
┌─────────────────────────────────────────────┐
│            HEADER METRICS BAR               │
│  Market Health | Active Stocks | Avg Volume │
├─────────────────────────────────────────────┤
│                                             │
│            RECOMMENDATION TABS              │
│  [Top Opportunities] [Large Trades] [Day]  │
│                                             │
├─────────────────────────────────────────────┤
│                                             │
│            STOCK CARDS GRID                 │
│   ┌──────┐  ┌──────┐  ┌──────┐           │
│   │Card 1│  │Card 2│  │Card 3│           │
│   └──────┘  └──────┘  └──────┘           │
│                                             │
└─────────────────────────────────────────────┘
```

### Tab Categories:

1. **Top Opportunities** 🌟
   - Best overall liquidity scores
   - Balanced for all strategies

2. **Best for Large Trades** 🏢
   - Can handle institutional size
   - Minimal market impact

3. **Best for Day Trading** ⚡
   - High frequency capability
   - Tight spreads

4. **High Risk** ⚠️
   - Low liquidity warnings
   - Require special care

---

## Understanding the Visual Elements

### Color Coding System

The dashboard uses intuitive colors to convey information quickly:

#### Score Colors:
- 🟢 **Green (80-100)**: Excellent liquidity
- 🔵 **Blue (60-79)**: Good liquidity
- 🟡 **Yellow (40-59)**: Moderate liquidity
- 🟠 **Orange (20-39)**: Poor liquidity
- 🔴 **Red (0-19)**: Very poor liquidity

#### Action Badge Colors:
- **Green Badge**: BUY/BUY_LARGE - Strong opportunity
- **Blue Badge**: DAY_TRADE - Good for active trading
- **Gray Badge**: HOLD - Neutral position
- **Red Badge**: CAUTION/AVOID - Warning signal

### Icons and Their Meanings

| Icon | Meaning | Indicates |
|------|---------|-----------|
| 🛡️ Shield | Conservative threshold | Safest trade size |
| ⚡ Lightning | Aggressive threshold | Higher risk/reward |
| 🎯 Target | Optimal trade size | Recommended amount |
| 💧 Droplet | Liquidity level | Overall market depth |
| 📊 Chart | Trading activity | Volume patterns |
| ⚠️ Warning | Risk alert | Requires caution |
| ✅ Check | Good data quality | Reliable metrics |

---

## Reading the Trading Cards

Each stock is displayed as an interactive card with multiple data layers:

### Card Anatomy:

```
┌─────────────────────────────────┐
│ BBOB                  [BUY_LARGE]│  ← Symbol & Action
│                                  │
│ 87.5 (Score)         Good Data ✓ │  ← Main Score & Quality
│ EMA20: 85.3 | Latest: 83.2      │  ← Trend Indicators
│                                  │
│ "Excellent liquidity, supports   │  ← AI Rationale
│  large trades up to 60M IQD"     │
│                                  │
│ Safe Trading Sizes:              │
│ 🛡️ Conservative: 15M IQD         │  ← Position Sizing
│ ⚡ Aggressive: 60M IQD           │
│ 🎯 Optimal: 35M IQD              │
│                                  │
│ Continuity: 95% | Daily: 250M    │  ← Key Metrics
└─────────────────────────────────┘
```

### Understanding Each Element:

1. **Main Score (Large Number)**
   - The EMA20 score (20-day average)
   - Your primary decision metric
   - Color-coded for quick assessment

2. **Trend Indicators**
   - **EMA20**: 20-day exponential moving average
   - **Latest**: Most recent score
   - **Best**: Peak score achieved
   - Shows if liquidity is improving/declining

3. **AI Rationale**
   - Plain English explanation
   - Why this recommendation
   - Key factors considered

4. **Safe Trading Sizes**
   - Three risk levels
   - In Iraqi Dinars (IQD)
   - Directly actionable

5. **Bottom Metrics**
   - **Continuity %**: How often it trades
   - **Daily Volume**: Average daily value

---

## Market Health Indicators

### Top Header Metrics:

#### Market Health Score (0-100)
```
72.5 = Healthy Market
```
**Interpretation**:
- 80-100: Excellent market conditions
- 60-80: Good trading environment
- 40-60: Mixed conditions
- 20-40: Poor liquidity overall
- 0-20: Severe liquidity crisis

#### Total Active Stocks
```
45 of 80 stocks actively trading
```
**What it means**:
- Shows market participation
- Higher = more opportunities
- Lower = selective trading needed

#### Average Continuity
```
55.2% average trading frequency
```
**Indicates**:
- Market-wide trading patterns
- Holiday/event impacts
- Overall market stability

#### Median Daily Volume
```
5M IQD median daily trading
```
**Reveals**:
- Typical stock liquidity
- Market depth
- Money flow patterns

---

## Using Recommendations

### Understanding Action Recommendations:

#### BUY_LARGE 💚
**What it means**: Stock can handle institutional-size positions
**When to act**: Building large positions, fund allocations
**Typical score**: 80+

#### BUY 🟢
**What it means**: Good for standard position sizes
**When to act**: Regular investment, portfolio additions
**Typical score**: 60-80

#### DAY_TRADE 🔵
**What it means**: Suitable for frequent in/out trading
**When to act**: Short-term strategies, quick profits
**Requirements**: High continuity, tight spreads

#### HOLD ⚪
**What it means**: Maintain current position
**When to act**: No clear advantage to buying/selling
**Typical score**: 40-60

#### CAUTION 🟡
**What it means**: Elevated risk, trade carefully
**When to act**: Only with clear strategy
**Typical score**: 20-40

#### AVOID 🔴
**What it means**: High risk of liquidity trap
**When to act**: Exit if possible, don't enter
**Typical score**: Below 20

---

## Interactive Features

### 1. Search Function 🔍
Located at top of dashboard:
```
[🔍 Search stocks...]
```
- Type symbol or partial name
- Real-time filtering
- Case-insensitive

### 2. Tab Navigation
Click tabs to switch categories:
- Updates cards instantly
- Maintains search filters
- Shows relevant stocks only

### 3. Card Interactions
**Hover Effects**:
- Card enlarges slightly
- Shadow effect appears
- Indicates clickability

**Click Actions** (when implemented):
- Opens detailed view
- Shows historical charts
- Provides deep analytics

### 4. Refresh Button 🔄
- Updates all data
- Fetches latest calculations
- Shows last update time

### 5. Export Function 📥
- Download current view
- CSV format available
- Includes all visible metrics

---

## Customization and Filters

### Available Filters:

#### Score Range Slider
```
Min [0]────────●────────●────[100] Max
           30         80
```
Shows only stocks between selected scores

#### Continuity Filter
```
[ ] > 90% (Daily traders)
[ ] > 70% (Regular)
[ ] > 50% (Occasional)
[ ] All stocks
```

#### Volume Filter
```
[ ] > 500M IQD (Very High)
[ ] > 100M IQD (High)
[ ] > 50M IQD (Medium)
[ ] > 10M IQD (Low)
[ ] All volumes
```

#### Sector Filter
```
[ ] Banking
[ ] Industry
[ ] Services
[ ] Agriculture
[ ] All sectors
```

---

## Quick Start Tutorial

### For New Users - Your First 5 Minutes:

#### Minute 1: Check Market Health
1. Look at top header
2. Is Market Health > 60? Good day to trade
3. Note active stock count

#### Minute 2: Find Your Opportunities
1. Click "Top Opportunities" tab
2. Look for green scores (80+)
3. Read the AI rationale

#### Minute 3: Check Position Sizes
1. Find a stock you like
2. Look at "Optimal" trade size
3. Ensure it fits your budget

#### Minute 4: Verify Continuity
1. Check continuity percentage
2. Above 85%? Good for short-term
3. Above 60%? OK for longer holds

#### Minute 5: Make Your Decision
1. Note 2-3 best candidates
2. Compare their scores
3. Start with highest score

---

## Common Use Cases

### Case 1: "I have 100M IQD to invest"
1. Go to "Best for Large Trades"
2. Find stocks with Optimal > 30M
3. Select 3-4 stocks
4. Diversify across them

### Case 2: "I want to day trade"
1. Click "Best for Day Trading"
2. Look for continuity > 90%
3. Check spreads are tight
4. Use Conservative threshold

### Case 3: "I need to exit positions"
1. Check your stocks' current scores
2. If score < 40, plan gradual exit
3. If score > 70, can exit quickly
4. Prioritize low score exits

### Case 4: "Market seems weak"
1. Check Market Health Score
2. If below 50, reduce activity
3. Focus on highest scores only
4. Keep more cash ready

---

## Understanding Data Quality Indicators

### "Good Data" Badge ✅
**Shows when**:
- Consistent daily updates
- No data gaps
- Reliable price feeds
- Accurate calculations

### Missing Badge ⚠️
**Indicates**:
- Some data gaps exist
- Less reliable metrics
- Use with caution
- Verify before large trades

---

## Pro Tips for Power Users

### 1. The Morning Routine
- Check Market Health first
- Review score changes from yesterday
- Identify movers (up/down significantly)
- Plan your day's trades

### 2. The Liquidity Ladder
- Start with highest score stocks
- Work your way down as needed
- Never go below score 40 for day trades
- Keep 50% in score 70+ stocks

### 3. The Safety Check
- Always verify Optimal trade size
- Check continuity before entering
- Note spread costs
- Plan exit before entry

### 4. The Trend Spotter
- Compare EMA20 vs Latest
- Rising = improving liquidity
- Falling = deteriorating conditions
- Act accordingly

---

## Troubleshooting

### "No data available"
- Run operations pipeline first
- Check data downloads
- Ensure calculations completed

### "Scores seem wrong"
- Check calculation date
- Verify 60-day window
- Look for data gaps

### "Can't find my stock"
- May be below filter threshold
- Check if delisted/suspended
- Try removing filters

---

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `/` | Focus search box |
| `Tab` | Next section |
| `1-4` | Switch tabs |
| `R` | Refresh data |
| `E` | Export view |
| `Esc` | Clear search |

---

## Mobile Responsiveness

The dashboard adapts to screen size:

### Desktop (Full View)
- 3-4 cards per row
- All details visible
- Side-by-side comparisons

### Tablet (Medium View)
- 2 cards per row
- Compressed metrics
- Touch-friendly

### Mobile (Compact View)
- 1 card per row
- Stacked layout
- Swipe navigation

---

## Summary - Your Daily Workflow

### Morning (Market Open):
1. Check Market Health Score
2. Review Top Opportunities
3. Note any score changes
4. Plan day's trades

### Midday (Active Trading):
1. Monitor your positions
2. Check liquidity trends
3. Adjust if needed
4. Watch for opportunities

### Afternoon (Planning):
1. Review day's performance
2. Check tomorrow's outlook
3. Set alerts if available
4. Plan next day

---

*The Liquidity Dashboard is your window into market dynamics. Use it daily for better trading decisions.*

*Last Updated: August 2025*
*ISX Pulse - See the Market Clearly*