# Release Notes - v3.0.4

## Release Date: 2025-08-03

## Overview
This release fixes critical issues with data collection operations and provides comprehensive progress reporting for better user experience.

## Fixed Issues

### 1. Operation Failed - Executable Name Mismatch
- **Issue**: Operations failed with "process.exe not found" error
- **Root Cause**: Code was looking for `process.exe` but the actual executable is named `processor.exe`
- **Solution**: Updated stages.go to use the correct executable name

### 2. React Hydration Errors (418 & 423)
- **Issue**: Console errors in production build causing UI flickering
- **Root Cause**: Client-only state and dynamic date handling causing server/client mismatch
- **Solution**: Removed `isClient` state and fixed date initialization for SSR compatibility

### 3. No Progress Reporting During Data Collection
- **Issue**: Users couldn't see download progress or expected file count
- **Solution**: Implemented comprehensive progress tracking:
  - Calculates expected files based on date range (excluding weekends)
  - Shows "Downloading file X of Y" with real-time updates
  - Displays total progress summary

## New Features

### Enhanced Progress Reporting
1. **Expected File Calculation**
   - Automatically calculates expected files based on date range
   - Accounts for ISX working days (Sunday-Thursday)
   - Shows total at start: "Total expected files: 250 (from 2024-01-01 to 2024-12-31)"

2. **Real-Time Progress Updates**
   - Individual file progress: "Downloading file 112 of 250"
   - Existing file tracking: "File 113 of 250 already exists, skipping"
   - Page summaries: "Progress: 150 of 250 files processed (100 downloaded, 50 existing)"

3. **Detailed WebSocket Metadata**
   ```json
   {
     "progress": 45,
     "message": "Downloading file 112 of 250",
     "stage": "scraping",
     "total_expected": 250,
     "files_downloaded": 100,
     "files_existing": 12,
     "current_file": 112,
     "current_page": 5
   }
   ```

## Technical Improvements

### Scraper Enhancements
- Added `calculateExpectedFiles()` function
- Improved console output formatting
- Better tracking of downloaded vs existing files
- Final summary with all statistics

### Operations Stage Updates
- Enhanced parsing of scraper output
- Accurate progress calculation based on expected files
- Rich metadata in WebSocket broadcasts
- Better error handling and logging

## How It Works Now

1. **Start Operation**: Select date range and start scraping
2. **See Expected Files**: "Total expected files: 250" appears immediately
3. **Track Progress**: Watch as files are downloaded with accurate percentage
4. **Know Status**: See which files exist vs newly downloaded
5. **Complete Summary**: Final report shows all statistics

## Testing Notes

The operation flow now works correctly:
1. Scraping starts and shows expected files
2. Progress updates in real-time
3. Processing stage runs with correct executable
4. All stages complete successfully

## Package Contents
- `web.exe` - Main application server
- `scraper.exe` - ISX data scraper (with progress reporting)
- `processor.exe` - Data processor (renamed from process.exe)
- `indexcsv.exe` - Index extractor
- Configuration files and documentation