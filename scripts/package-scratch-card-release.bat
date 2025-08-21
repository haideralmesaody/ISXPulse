@echo off
:: ============================================
:: ISX Pulse - Scratch Card Release Packaging
:: ============================================
:: Creates a distributable package with scratch card license system

setlocal enabledelayedexpansion

set VERSION=0.0.1-alpha
set RELEASE_NAME=ISXPulse-ScratchCard-v%VERSION%-win64

echo ==========================================
echo    Packaging ISX Pulse Scratch Card      
echo          v%VERSION% Release               
echo     The Heartbeat of Iraqi Markets       
echo ==========================================
echo.

:: Check if dist directory exists
if not exist dist (
    echo [ERROR] Dist directory not found. 
    echo [ERROR] Run scripts\deploy-scratch-card.bat first to build
    pause
    exit /b 1
)

:: Check for scratch card executables
set MISSING_EXES=
if not exist "dist\ISXPulse.exe" set MISSING_EXES=!MISSING_EXES! ISXPulse.exe
if not exist "dist\scraper.exe" set MISSING_EXES=!MISSING_EXES! scraper.exe
if not exist "dist\processor.exe" set MISSING_EXES=!MISSING_EXES! processor.exe
if not exist "dist\indexcsv.exe" set MISSING_EXES=!MISSING_EXES! indexcsv.exe
if not exist "dist\license-generator.exe" set MISSING_EXES=!MISSING_EXES! license-generator.exe

if defined MISSING_EXES (
    echo [ERROR] Missing executables: !MISSING_EXES!
    echo [ERROR] Please build with scratch card features first
    pause
    exit /b 1
)

:: Check if PowerShell is available for zipping
where powershell >nul 2>nul
if errorlevel 1 (
    echo [ERROR] PowerShell not found. Cannot create ZIP file.
    echo Please create ZIP manually from the dist directory.
    pause
    exit /b 1
)

echo [INFO] Creating scratch card release package...

:: Create temporary package directory
set TEMP_PKG_DIR=%RELEASE_NAME%_temp
if exist "%TEMP_PKG_DIR%" rd /s /q "%TEMP_PKG_DIR%"
mkdir "%TEMP_PKG_DIR%"

:: Copy dist contents
echo [INFO] Copying distribution files...
xcopy "dist\*" "%TEMP_PKG_DIR%\" /E /I /Q >nul

:: Create scratch card specific documentation
echo [INFO] Creating scratch card documentation...

:: Create main README for scratch card release
(
echo # ISX Pulse - Scratch Card License System
echo The Heartbeat of Iraqi Markets
echo.
echo Version: %VERSION%
echo Release Date: %date%
echo License System: Scratch Card ^(One-Time Activation^)
echo.
echo ## Quick Start
echo.
echo 1. **Setup Credentials**
echo    - Run setup-scratch-card-credentials.bat to configure
echo    - Ensure you have Google Apps Script deployed
echo    - Place your credentials.json file in this directory
echo.
echo 2. **Generate Scratch Cards**
echo    ```
echo    license-generator.exe -count 100 -duration 1m -batch "BATCH001"
echo    ```
echo.
echo 3. **Start Server**
echo    ```
echo    ISXPulse.exe
echo    ```
echo.
echo 4. **Activate License**
echo    - Open web interface at http://localhost:8080
echo    - Enter scratch card code ^(format: ISX-XXXX-XXXX-XXXX^)
echo    - Code can only be used once
echo.
echo ## Included Components
echo.
echo ### Core Applications
echo - **ISXPulse.exe** - Main server with scratch card support
echo - **scraper.exe** - ISX data scraper
echo - **processor.exe** - Data processor
echo - **indexcsv.exe** - CSV index extractor
echo - **license-generator.exe** - Scratch card generator
echo.
echo ### Configuration Files
echo - **.env.example** - Environment configuration template
echo - **sheets-config.json.example** - Google Sheets configuration
echo - **credentials.json.example** - Google API credentials template
echo.
echo ### Documentation
echo - **SCRATCH_CARD_DEPLOYMENT.txt** - Deployment information
echo - **SECURITY.md** - Security considerations
echo - **README.md** - This file
echo.
echo ## Scratch Card Features
echo.
echo - **One-Time Activation**: Each code can only be used once
echo - **Device Binding**: License tied to device fingerprint
echo - **Rate Limiting**: Protection against brute force attacks
echo - **Blacklisting**: Automatic blocking of suspicious IPs
echo - **Audit Logging**: Complete activation history
echo - **Offline Grace Period**: Works temporarily without internet
echo.
echo ## License Generator Usage
echo.
echo Generate scratch cards in batches:
echo ```
echo license-generator.exe -count 100 -duration 1m -batch "BATCH001"
echo license-generator.exe -count 50 -duration 3m -batch "PREMIUM001"
echo license-generator.exe -count 25 -duration 1y -batch "ENTERPRISE001"
echo ```
echo.
echo Duration options: 1m, 3m, 6m, 1y
echo.
echo ## Security Notes
echo.
echo - Keep your Google Apps Script URL private
echo - Monitor activation logs for suspicious activity
echo - Regular backup of Google Sheets data recommended
echo - Use HTTPS in production deployments
echo - Protect credentials.json file
echo.
echo ## Troubleshooting
echo.
echo ### Common Issues
echo.
echo 1. **"Invalid license code"**
echo    - Check code format: ISX-XXXX-XXXX-XXXX
echo    - Ensure code hasn't been used already
echo    - Verify Apps Script is accessible
echo.
echo 2. **"Network error"**
echo    - Check internet connection
echo    - Verify Apps Script URL is correct
echo    - Check firewall settings
echo.
echo 3. **"Rate limited"**
echo    - Too many activation attempts
echo    - Wait and try again later
echo    - Check for IP blacklisting
echo.
echo ### Log Files
echo - Server logs: logs\server.log
echo - Activation logs: logs\activation.log
echo - Error logs: logs\error.log
echo.
echo ### Support
echo For technical support, check the logs directory and deployment documentation.
echo.
echo ---
echo **ISX Pulse** - Professional financial data processing for the Iraqi Stock Exchange
echo Scratch Card License System - Secure, reliable, one-time activation
) > "%TEMP_PKG_DIR%\README.md"

:: Create deployment checklist
(
echo # Scratch Card Deployment Checklist
echo.
echo ## Pre-Deployment
echo - [ ] Google Apps Script deployed and tested
echo - [ ] Google Sheets API credentials configured
echo - [ ] Apps Script URL added to configuration
echo - [ ] Service account has access to license sheet
echo.
echo ## Deployment
echo - [ ] Run setup-scratch-card-credentials.bat
echo - [ ] Configure .env file with production values
echo - [ ] Test build with scratch card features
echo - [ ] Deploy to production environment
echo.
echo ## Post-Deployment
echo - [ ] Generate initial batch of scratch cards
echo - [ ] Test activation flow with sample card
echo - [ ] Monitor Apps Script logs
echo - [ ] Set up activation metrics monitoring
echo - [ ] Backup Google Sheets data
echo.
echo ## Security Checklist
echo - [ ] Apps Script URL is private
echo - [ ] credentials.json is secure
echo - [ ] Rate limiting is enabled
echo - [ ] Blacklisting is functional
echo - [ ] Audit logging is working
echo - [ ] HTTPS enabled in production
echo.
echo ## Ongoing Maintenance
echo - [ ] Monitor activation success rate
echo - [ ] Review blacklist entries
echo - [ ] Clean up old attempt logs
echo - [ ] Update license batches as needed
echo - [ ] Regular security audits
) > "%TEMP_PKG_DIR%\DEPLOYMENT_CHECKLIST.md"

:: Create license generator quick start
(
echo # License Generator Quick Start
echo.
echo ## Basic Usage
echo.
echo Generate 100 one-month licenses:
echo ```
echo license-generator.exe -count 100 -duration 1m -batch "STANDARD001"
echo ```
echo.
echo ## Duration Options
echo - `1m` - One month
echo - `3m` - Three months  
echo - `6m` - Six months
echo - `1y` - One year
echo.
echo ## Batch Management
echo.
echo Use batch IDs to track different license sets:
echo - STANDARD001 - Standard monthly licenses
echo - PREMIUM001 - Premium quarterly licenses
echo - ENTERPRISE001 - Enterprise annual licenses
echo.
echo ## Output Files
echo.
echo The generator creates:
echo - licenses.csv - License codes for import
echo - batch_summary.txt - Generation summary
echo - qr_codes.pdf - QR codes for printing ^(optional^)
echo.
echo ## Import to Google Sheets
echo.
echo 1. Open your license management spreadsheet
echo 2. Go to the Licenses sheet
echo 3. Import the CSV file
echo 4. Verify all codes are marked as "Available"
echo.
echo ## Security Notes
echo.
echo - Generate codes on secure, offline machine when possible
echo - Secure transport of license files
echo - Delete temporary files after import
echo - Track batch distribution carefully
) > "%TEMP_PKG_DIR%\LICENSE_GENERATOR_GUIDE.md"

:: Create troubleshooting guide
(
echo # Scratch Card Troubleshooting Guide
echo.
echo ## Activation Issues
echo.
echo ### "Invalid license code"
echo **Causes:**
echo - Code format incorrect ^(should be ISX-XXXX-XXXX-XXXX^)
echo - Code already activated
echo - Code not in Google Sheets
echo - Apps Script not accessible
echo.
echo **Solutions:**
echo 1. Verify code format and typing
echo 2. Check activation history in Google Sheets
echo 3. Test Apps Script URL directly
echo 4. Check internet connectivity
echo.
echo ### "Rate limited"
echo **Causes:**
echo - Too many activation attempts from same IP
echo - IP address blacklisted
echo - Apps Script rate limits hit
echo.
echo **Solutions:**
echo 1. Wait before retrying
echo 2. Check blacklist in Google Sheets
echo 3. Monitor Apps Script quotas
echo 4. Consider IP whitelist for legitimate users
echo.
echo ### "Network error"
echo **Causes:**
echo - Internet connection issues
echo - Apps Script endpoint down
echo - Firewall blocking requests
echo - DNS resolution problems
echo.
echo **Solutions:**
echo 1. Check internet connectivity
echo 2. Verify Apps Script URL
echo 3. Test from different network
echo 4. Check firewall settings
echo.
echo ## Configuration Issues
echo.
echo ### Apps Script Not Responding
echo 1. Check deployment status
echo 2. Verify permissions ^(Execute as: Me, Access: Anyone^)
echo 3. Check Apps Script logs
echo 4. Redeploy if necessary
echo.
echo ### Google Sheets Access Denied
echo 1. Verify service account has access
echo 2. Check credentials.json format
echo 3. Ensure sheet ID is correct
echo 4. Test API access manually
echo.
echo ## Performance Issues
echo.
echo ### Slow Activation
echo **Causes:**
echo - Apps Script performance
echo - Large sheets with many licenses
echo - Network latency
echo.
echo **Solutions:**
echo 1. Optimize Apps Script code
echo 2. Archive old activation attempts
echo 3. Use CDN for Apps Script
echo 4. Implement local caching
echo.
echo ### High Memory Usage
echo **Causes:**
echo - Large license batches
echo - Cache not clearing
echo - Memory leaks in validation
echo.
echo **Solutions:**
echo 1. Process licenses in smaller batches
echo 2. Clear cache regularly
echo 3. Monitor for memory leaks
echo 4. Restart server periodically
echo.
echo ## Log Analysis
echo.
echo ### Important Log Locations
echo - `logs\activation.log` - All activation attempts
echo - `logs\error.log` - Error details
echo - `logs\server.log` - General server operations
echo - Google Apps Script logs - Apps Script execution
echo.
echo ### Log Patterns to Watch
echo - High failure rates ^(^>10%%^)
echo - Repeated attempts from same IP
echo - Apps Script timeout errors
echo - Memory or performance warnings
echo.
echo ## Emergency Procedures
echo.
echo ### Apps Script Failure
echo 1. Enable offline validation mode
echo 2. Use backup Apps Script deployment
echo 3. Contact Google Apps Script support
echo 4. Consider migration to Cloud Functions
echo.
echo ### Mass Invalid Activations
echo 1. Check for data corruption
echo 2. Verify Google Sheets integrity
echo 3. Review recent code changes
echo 4. Restore from backup if needed
echo.
echo ### Security Breach Suspected
echo 1. Immediately blacklist suspicious IPs
echo 2. Review activation logs
echo 3. Change Apps Script URL if compromised
echo 4. Audit all recent activations
echo 5. Generate new license batch if needed
) > "%TEMP_PKG_DIR%\TROUBLESHOOTING.md"

:: Copy security documentation if it exists
if exist "SECURITY.md" (
    copy "SECURITY.md" "%TEMP_PKG_DIR%\" >nul
)

:: Create installation scripts in package
(
echo @echo off
echo :: Quick setup script for scratch card system
echo echo Setting up ISX Pulse Scratch Card System...
echo if not exist ".env" copy ".env.example" ".env"
echo echo Please edit .env file with your Apps Script URL
echo echo Then run ISXPulse.exe to start the server
echo pause
) > "%TEMP_PKG_DIR%\INSTALL.bat"

:: Remove old zip if exists
if exist "%RELEASE_NAME%.zip" del "%RELEASE_NAME%.zip"

:: Create zip using PowerShell
echo [INFO] Creating ZIP package...
powershell -Command "Compress-Archive -Path '%TEMP_PKG_DIR%\*' -DestinationPath '%RELEASE_NAME%.zip' -Force"

if errorlevel 1 (
    echo [ERROR] Failed to create ZIP file
    pause
    exit /b 1
)

:: Clean up temporary directory
rd /s /q "%TEMP_PKG_DIR%"

:: Get file size
for %%A in (%RELEASE_NAME%.zip) do set SIZE=%%~zA
set /a SIZE_MB=%SIZE% / 1048576

echo.
echo ==========================================
echo   Scratch Card Release Package Created!
echo ==========================================
echo.
echo File: %RELEASE_NAME%.zip
echo Size: ~%SIZE_MB% MB
echo.
echo Package Contents:
echo.
echo Core Applications:
echo - ISXPulse.exe ^(Main server with scratch card support^)
echo - scraper.exe ^(ISX data scraper^)
echo - processor.exe ^(Data processor^)
echo - indexcsv.exe ^(CSV index extractor^)
echo - license-generator.exe ^(Scratch card generator^)
echo.
echo Configuration:
echo - .env.example ^(Environment template^)
echo - sheets-config.json.example ^(Sheets configuration^)
echo - credentials.json.example ^(API credentials template^)
echo.
echo Documentation:
echo - README.md ^(Quick start guide^)
echo - LICENSE_GENERATOR_GUIDE.md ^(License generation^)
echo - TROUBLESHOOTING.md ^(Problem solving^)
echo - DEPLOYMENT_CHECKLIST.md ^(Deployment steps^)
echo - SCRATCH_CARD_DEPLOYMENT.txt ^(Build info^)
echo.
echo Installation:
echo - INSTALL.bat ^(Quick setup script^)
echo.
echo Features:
echo - One-time activation scratch card system
echo - Device fingerprinting and binding
echo - Rate limiting and blacklisting
echo - Complete audit logging
echo - Offline validation grace period
echo - Google Apps Script integration
echo.
echo ==========================================
echo.
echo Ready for distribution!
echo This package contains the complete scratch card
echo license system for ISX Pulse.
echo.
echo Recipients should:
echo 1. Run INSTALL.bat for quick setup
echo 2. Configure Apps Script URL in .env
echo 3. Generate scratch cards with license-generator.exe
echo 4. Start server with ISXPulse.exe
echo.
pause