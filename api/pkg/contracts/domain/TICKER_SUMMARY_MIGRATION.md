# TickerSummary SSOT Migration Guide

This document provides guidance for migrating existing ticker summary implementations to use the new Single Source of Truth (SSOT) contract defined in `ticker_summary.go`.

## Overview

The new `TickerSummary` contract in `pkg/contracts/domain/ticker_summary.go` is now the authoritative definition for all ticker summary data across the ISX Daily Reports Scrapper system. This replaces the previous implementation in `internal/dataprocessing/summarizer.go`.

## Migration Steps

### 1. Update Import Statements

**Before:**
```go
import "isxcli/internal/dataprocessing"

// Using internal type
summary := dataprocessing.TickerSummary{...}
```

**After:**
```go
import "isxcli/pkg/contracts/domain"

// Using SSOT contract
summary := domain.TickerSummary{...}
```

### 2. Replace Constructor Calls

**Before:**
```go
summarizer := dataprocessing.NewSummarizer(logger, config)
summaries, err := summarizer.GenerateFromRecords(ctx, records)
```

**After:**
```go
// Use the new constructor for individual summaries
summary, err := domain.NewTickerSummary("BBOB", "Bank of Baghdad", 1.250, "2024-01-15", 120)
if err != nil {
    return fmt.Errorf("create summary: %w", err)
}

// Or create manually and validate
summary := &domain.TickerSummary{
    Ticker:      "BBOB",
    CompanyName: "Bank of Baghdad",
    LastPrice:   1.250,
    LastDate:    "2024-01-15",
    TradingDays: 120,
    Last10Days:  []float64{1.200, 1.210, 1.220, 1.240, 1.250},
}

if err := domain.ValidateTickerSummary(summary); err != nil {
    return fmt.Errorf("validation failed: %w", err)
}
```

### 3. Update CSV Handling

**Before (internal implementation):**
```go
csvString := summarizer.formatLast10Days(summary.Last10Days)
```

**After (SSOT contract):**
```go
csvString := summary.FormatLast10DaysForCSV()

// For parsing from CSV
err := summary.ParseLast10DaysFromCSV("1.200,1.210,1.220")
if err != nil {
    return fmt.Errorf("parse prices: %w", err)
}
```

### 4. Update Validation

**Before:**
```go
// No standardized validation
```

**After:**
```go
if err := domain.ValidateTickerSummary(summary); err != nil {
    return fmt.Errorf("invalid ticker summary: %w", err)
}

// Or validate just the ticker
if !domain.IsValidTicker("BBOB") {
    return errors.New("invalid ticker format")
}
```

### 5. Update Field Names and Tags

The new SSOT contract includes updated field names and tags. Ensure your code uses the correct field names:

**Updated Field Mapping:**
```go
// Core fields (same as before)
Ticker      string    `json:"ticker" csv:"Ticker"`
CompanyName string    `json:"company_name" csv:"CompanyName"`
LastPrice   float64   `json:"last_price" csv:"LastPrice"`
LastDate    string    `json:"last_date" csv:"LastDate"`
TradingDays int       `json:"trading_days" csv:"TradingDays"`
Last10Days  []float64 `json:"last_10_days" csv:"Last10Days"`

// Extended fields (new/enhanced)
TotalVolume   int64   `json:"total_volume,omitempty" csv:"TotalVolume,omitempty"`
TotalValue    float64 `json:"total_value,omitempty" csv:"TotalValue,omitempty"`
AveragePrice  float64 `json:"average_price,omitempty" csv:"AveragePrice,omitempty"`
HighestPrice  float64 `json:"highest_price,omitempty" csv:"HighestPrice,omitempty"`
LowestPrice   float64 `json:"lowest_price,omitempty" csv:"LowestPrice,omitempty"`
ChangePercent float64 `json:"change_percent,omitempty" csv:"ChangePercent,omitempty"`
```

## Key Differences from Old Implementation

### 1. Enhanced Validation
- Comprehensive validation rules for all fields
- Clear error messages with field-specific information
- Business rule enforcement (e.g., minimum price, ticker format)

### 2. Better CSV Support
- Dedicated methods for CSV formatting and parsing
- Proper error handling for malformed CSV data
- Support for empty values and whitespace trimming

### 3. Extensible Design
- Optional extended metrics with `omitempty` tags
- Version tracking for backward compatibility
- Metadata fields for audit and debugging

### 4. Type Safety
- Strong typing with proper validation
- Immutable validation rules structure
- Clear separation of core vs. extended fields

## Common Migration Patterns

### Pattern 1: Batch Processing
**Before:**
```go
summarizer := dataprocessing.NewSummarizer(logger, config)
summaries, err := summarizer.GenerateFromRecords(ctx, records)
if err != nil {
    return err
}

// Write to CSV
err = summarizer.WriteCSV(ctx, "output.csv", summaries)
```

**After:**
```go
summaries := make([]domain.TickerSummary, 0, len(tickerGroups))

for ticker, tickerRecords := range tickerGroups {
    summary, err := generateTickerSummaryFromRecords(ctx, ticker, tickerRecords)
    if err != nil {
        return fmt.Errorf("generate summary for %s: %w", ticker, err)
    }
    
    if err := domain.ValidateTickerSummary(summary); err != nil {
        return fmt.Errorf("invalid summary for %s: %w", ticker, err)
    }
    
    summaries = append(summaries, *summary)
}

// Write to CSV using your own implementation or existing CSV writer
err = writeTickerSummariesToCSV(ctx, "output.csv", summaries)
```

### Pattern 2: API Responses
**Before:**
```go
type APIResponse struct {
    Summaries []dataprocessing.TickerSummary `json:"summaries"`
}
```

**After:**
```go
type APIResponse struct {
    Summaries []domain.TickerSummary `json:"summaries"`
    // Use the built-in response type for better structure
}

// Or use the provided response type
response := &domain.TickerSummaryResponse{
    Summaries:   summaries,
    TotalCount:  len(summaries),
    GeneratedAt: time.Now(),
}
```

### Pattern 3: Configuration
**Before:**
```go
config := dataprocessing.SummarizerConfig{
    IncludeExtendedMetrics: true,
    MaxLast10Days:         10,
    DateFormat:            "2006-01-02",
}
```

**After:**
```go
// Configuration is now embedded in the contract validation rules
// Use domain.TickerSummaryValidationRules for constraints
maxDays := domain.TickerSummaryValidationRules.MaxLast10Days
dateFormat := domain.TickerSummaryValidationRules.RequiredDateFormat
```

## Testing Your Migration

1. **Validation Testing:**
```go
// Test that your summaries pass validation
for _, summary := range summaries {
    if err := domain.ValidateTickerSummary(&summary); err != nil {
        t.Errorf("Summary validation failed: %v", err)
    }
}
```

2. **CSV Round-Trip Testing:**
```go
// Test CSV formatting and parsing
original := summary.Last10Days
csvString := summary.FormatLast10DaysForCSV()

newSummary := &domain.TickerSummary{}
err := newSummary.ParseLast10DaysFromCSV(csvString)
require.NoError(t, err)

assert.Equal(t, original, newSummary.Last10Days)
```

3. **JSON Serialization Testing:**
```go
// Test JSON marshaling/unmarshaling
data, err := json.Marshal(summary)
require.NoError(t, err)

var unmarshaled domain.TickerSummary
err = json.Unmarshal(data, &unmarshaled)
require.NoError(t, err)

assert.Equal(t, summary, unmarshaled)
```

## Deprecated Code to Remove

After migration, you can safely remove:

1. `internal/dataprocessing/summarizer.go` - Replace with SSOT contract
2. Local `TickerSummary` type definitions - Use `domain.TickerSummary`
3. Custom validation functions - Use `domain.ValidateTickerSummary`
4. Manual CSV formatting - Use contract methods

## Benefits of Migration

1. **Consistency:** Single definition used across all components
2. **Validation:** Comprehensive validation with clear error messages
3. **Type Safety:** Strong typing with proper field validation
4. **Maintainability:** Centralized contract reduces duplication
5. **Extensibility:** Easy to add new fields without breaking changes
6. **Testing:** Built-in test coverage for all contract operations

## Support

For questions about the migration:

1. Review the comprehensive test suite in `ticker_summary_test.go`
2. Check the validation rules in `TickerSummaryValidationRules`
3. Refer to the detailed field documentation in the contract
4. Use the provided constructor `NewTickerSummary` for safe initialization

The SSOT contract is designed to be backward-compatible while providing enhanced functionality and validation. Take advantage of the new features while maintaining your existing business logic.