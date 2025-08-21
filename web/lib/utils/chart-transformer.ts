/**
 * Chart data transformation utilities
 * Converts ISX ticker data to Highcharts format
 */

import type { TickerHistoricalData } from '@/types/analysis'

/**
 * Transform ticker historical data to Highcharts OHLC and Volume format
 */
export function transformToHighchartsData(data: TickerHistoricalData[]) {
  const ohlc: number[][] = []
  const volume: number[][] = []
  let hasDataIssue = false
  
  // Sort data by date to ensure chronological order
  const sortedData = [...data].sort((a, b) => {
    const dateA = new Date(a.date).getTime()
    const dateB = new Date(b.date).getTime()
    return dateA - dateB
  })
  
  sortedData.forEach(item => {
    // Parse date and convert to timestamp
    const timestamp = new Date(item.date).getTime()
    
    // Ensure we have valid OHLC values - they're already rounded from API
    // Use close as fallback for missing values, or a default of 1.00
    const close = item.close > 0 ? item.close : (item.open > 0 ? item.open : 1.00)
    const open = item.open > 0 ? item.open : close
    const high = item.high > 0 ? item.high : Math.max(open, close)
    const low = item.low > 0 ? item.low : Math.min(open, close)
    
    // Warn once if data had to be corrected
    if (!hasDataIssue && (item.close <= 0 || item.open <= 0 || item.high <= 0 || item.low <= 0)) {
      console.warn('Chart data contains invalid prices (zeros), using fallback values')
      hasDataIssue = true
    }
    
    // OHLC format: [timestamp, open, high, low, close]
    ohlc.push([
      timestamp,
      open,
      high,
      low,
      close
    ])
    
    // Volume format: [timestamp, volume]
    // Color based on price movement (green for up, red for down)
    const color = close >= open ? '#10b981' : '#ef4444'
    volume.push([timestamp, item.volume || 0])
  })
  
  return { ohlc, volume }
}

/**
 * Calculate technical indicators from OHLC data
 */
export function calculateIndicators(data: TickerHistoricalData[]) {
  const indicators = {
    sma20: calculateSMA(data, 20),
    sma50: calculateSMA(data, 50),
    ema20: calculateEMA(data, 20),
    ema50: calculateEMA(data, 50),
    rsi: calculateRSI(data, 14),
    macd: calculateMACD(data),
    bollingerBands: calculateBollingerBands(data, 20, 2)
  }
  
  return indicators
}

/**
 * Calculate Simple Moving Average
 */
function calculateSMA(data: TickerHistoricalData[], period: number): number[][] {
  const sma: number[][] = []
  
  for (let i = period - 1; i < data.length; i++) {
    let sum = 0
    for (let j = 0; j < period; j++) {
      sum += data[i - j].close
    }
    const timestamp = new Date(data[i].date).getTime()
    sma.push([timestamp, Math.round((sum / period) * 100) / 100])
  }
  
  return sma
}

/**
 * Calculate Exponential Moving Average
 */
function calculateEMA(data: TickerHistoricalData[], period: number): number[][] {
  const ema: number[][] = []
  const multiplier = 2 / (period + 1)
  
  // Start with SMA for the first value
  let sum = 0
  for (let i = 0; i < period; i++) {
    sum += data[i].close
  }
  let previousEMA = sum / period
  
  const firstTimestamp = new Date(data[period - 1].date).getTime()
  ema.push([firstTimestamp, previousEMA])
  
  // Calculate EMA for remaining values
  for (let i = period; i < data.length; i++) {
    const currentEMA = (data[i].close - previousEMA) * multiplier + previousEMA
    const timestamp = new Date(data[i].date).getTime()
    ema.push([timestamp, currentEMA])
    previousEMA = currentEMA
  }
  
  return ema
}

/**
 * Calculate Relative Strength Index
 */
function calculateRSI(data: TickerHistoricalData[], period: number = 14): number[][] {
  const rsi: number[][] = []
  
  if (data.length < period + 1) return rsi
  
  // Calculate price changes
  const changes: number[] = []
  for (let i = 1; i < data.length; i++) {
    changes.push(data[i].close - data[i - 1].close)
  }
  
  // Calculate initial average gain and loss
  let avgGain = 0
  let avgLoss = 0
  
  for (let i = 0; i < period; i++) {
    if (changes[i] > 0) {
      avgGain += changes[i]
    } else {
      avgLoss += Math.abs(changes[i])
    }
  }
  
  avgGain /= period
  avgLoss /= period
  
  // Calculate RSI values
  for (let i = period; i < changes.length; i++) {
    const change = changes[i]
    
    if (change > 0) {
      avgGain = (avgGain * (period - 1) + change) / period
      avgLoss = (avgLoss * (period - 1)) / period
    } else {
      avgGain = (avgGain * (period - 1)) / period
      avgLoss = (avgLoss * (period - 1) + Math.abs(change)) / period
    }
    
    const rs = avgLoss === 0 ? 100 : avgGain / avgLoss
    const rsiValue = 100 - (100 / (1 + rs))
    const timestamp = new Date(data[i + 1].date).getTime()
    
    rsi.push([timestamp, rsiValue])
  }
  
  return rsi
}

/**
 * Calculate MACD (Moving Average Convergence Divergence)
 */
function calculateMACD(data: TickerHistoricalData[]) {
  const ema12 = calculateEMAValues(data, 12)
  const ema26 = calculateEMAValues(data, 26)
  
  const macdLine: number[][] = []
  const signalLine: number[][] = []
  const histogram: number[][] = []
  
  // Calculate MACD line (EMA12 - EMA26)
  const macdValues: number[] = []
  for (let i = 25; i < data.length; i++) {
    const macd = ema12[i] - ema26[i]
    macdValues.push(macd)
    const timestamp = new Date(data[i].date).getTime()
    macdLine.push([timestamp, macd])
  }
  
  // Calculate Signal line (9-period EMA of MACD)
  const signalMultiplier = 2 / (9 + 1)
  let previousSignal = macdValues.slice(0, 9).reduce((a, b) => a + b, 0) / 9
  
  for (let i = 8; i < macdValues.length; i++) {
    const currentSignal = (macdValues[i] - previousSignal) * signalMultiplier + previousSignal
    const timestamp = new Date(data[i + 25].date).getTime()
    signalLine.push([timestamp, currentSignal])
    histogram.push([timestamp, macdValues[i] - currentSignal])
    previousSignal = currentSignal
  }
  
  return { macdLine, signalLine, histogram }
}

/**
 * Helper function to calculate EMA values (not formatted for Highcharts)
 */
function calculateEMAValues(data: TickerHistoricalData[], period: number): number[] {
  const ema: number[] = new Array(data.length).fill(0)
  const multiplier = 2 / (period + 1)
  
  // Start with SMA
  let sum = 0
  for (let i = 0; i < period; i++) {
    sum += data[i].close
  }
  ema[period - 1] = sum / period
  
  // Calculate EMA
  for (let i = period; i < data.length; i++) {
    ema[i] = (data[i].close - ema[i - 1]) * multiplier + ema[i - 1]
  }
  
  return ema
}

/**
 * Calculate Bollinger Bands
 */
function calculateBollingerBands(data: TickerHistoricalData[], period: number = 20, stdDev: number = 2) {
  const upper: number[][] = []
  const middle: number[][] = []
  const lower: number[][] = []
  
  for (let i = period - 1; i < data.length; i++) {
    // Calculate SMA (middle band)
    let sum = 0
    for (let j = 0; j < period; j++) {
      sum += data[i - j].close
    }
    const sma = sum / period
    
    // Calculate standard deviation
    let squaredDifferences = 0
    for (let j = 0; j < period; j++) {
      squaredDifferences += Math.pow(data[i - j].close - sma, 2)
    }
    const standardDeviation = Math.sqrt(squaredDifferences / period)
    
    const timestamp = new Date(data[i].date).getTime()
    
    upper.push([timestamp, sma + (standardDeviation * stdDev)])
    middle.push([timestamp, sma])
    lower.push([timestamp, sma - (standardDeviation * stdDev)])
  }
  
  return { upper, middle, lower }
}

/**
 * Format large numbers for display
 */
export function formatLargeNumber(num: number): string {
  if (num >= 1e9) return `${(num / 1e9).toFixed(2)}B`
  if (num >= 1e6) return `${(num / 1e6).toFixed(2)}M`
  if (num >= 1e3) return `${(num / 1e3).toFixed(2)}K`
  return num.toFixed(2)
}

/**
 * Calculate percentage change
 */
export function calculatePercentageChange(oldValue: number, newValue: number): number {
  if (oldValue === 0) return 0
  return ((newValue - oldValue) / oldValue) * 100
}

/**
 * Determine trend direction
 */
export function determineTrend(data: TickerHistoricalData[], period: number = 10): 'up' | 'down' | 'sideways' {
  if (data.length < period) return 'sideways'
  
  const recentData = data.slice(-period)
  const firstPrice = recentData[0].close
  const lastPrice = recentData[recentData.length - 1].close
  const change = calculatePercentageChange(firstPrice, lastPrice)
  
  if (change > 2) return 'up'
  if (change < -2) return 'down'
  return 'sideways'
}