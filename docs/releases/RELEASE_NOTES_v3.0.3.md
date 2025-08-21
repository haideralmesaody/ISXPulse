# Release Notes - v3.0.3

## Release Date: 2025-08-03

## Overview
This release restores the full ISX Daily Reports Scrapper application with the complete Next.js frontend, including the dashboard and operations interface that was previously missing.

## Fixed Issues

### 1. Missing Dashboard and Operations Interface
- **Issue**: After license activation, users were stuck in a redirect loop to `/dashboard` which didn't exist
- **Root Cause**: The Next.js application was replaced with a simple HTML file
- **Solution**: Properly built and embedded the full Next.js application

### 2. License Activation Flow
- **Issue**: Redirect loop after successful license activation
- **Solution**: Dashboard page now exists and properly handles licensed users

### 3. Logo Display
- **Issue**: Iraqi Investor logo was not displaying
- **Solution**: All assets are now properly embedded in the build

## What's Included

### Frontend Pages
- **Home** (`/`) - Landing page
- **Dashboard** (`/dashboard`) - Main control panel
- **Operations** (`/operations`) - Manage scraping, processing, and analysis operations
- **Reports** (`/reports`) - View and export reports
- **Analysis** (`/analysis`) - Market analysis tools
- **License** (`/license`) - License activation and status

### Features Restored
1. **Operations Interface**
   - Start/stop scraping operations
   - Monitor progress with real-time updates
   - Configure operation parameters
   - View operation history

2. **WebSocket Integration**
   - Real-time operation status updates
   - Progress monitoring
   - Error notifications

3. **Professional UI**
   - Iraqi Investor branding
   - Responsive design
   - Modern Shadcn/ui components

## Technical Details

### Build Information
- Frontend: Next.js 14.2.30 with static export
- Backend: Go with embedded frontend
- Total size: ~22MB (includes embedded UI)

### API Endpoints
All API routes are correctly configured:
- License: `/api/license/*`
- Operations: `/api/operations/*`
- Data: `/api/data/*`

## How to Use

1. Start the application:
   ```batch
   web.exe
   ```

2. Navigate to `http://localhost:8080`

3. If not licensed:
   - You'll see the license activation page
   - Enter your license key
   - After activation, you'll be redirected to the dashboard

4. If already licensed:
   - You'll be automatically redirected to the dashboard
   - Access all features from the navigation menu

## Package Contents
- `web.exe` - Main application server with embedded UI
- `scraper.exe` - ISX data scraper
- `processor.exe` - Data processor
- `indexcsv.exe` - Index extractor
- Configuration examples
- Documentation

## Notes
- Ensure you have proper credentials configured
- License activation requires internet connection
- All previous features are now fully functional