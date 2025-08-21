@echo off
:: ============================================
:: ISX Pulse - Scratch Card Credentials Setup
:: ============================================
:: This script helps set up credentials for the scratch card license system

setlocal enabledelayedexpansion

echo ==========================================
echo  ISX Pulse - Scratch Card Credentials    
echo    The Heartbeat of Iraqi Markets        
echo ==========================================
echo.

:: Check if we're in the correct directory
if not exist "build.go" (
    echo [ERROR] Must run from project root directory
    echo [ERROR] build.go not found
    pause
    exit /b 1
)

echo [INFO] This script will help you set up credentials for the scratch card system
echo [INFO] You will need:
echo [INFO]   1. Google Apps Script Web App URL
echo [INFO]   2. Google Sheets API credentials (credentials.json)
echo [INFO]   3. Environment configuration
echo.

:: ============================================
:: Step 1: Google Apps Script URL
:: ============================================

echo ============================================
echo Step 1: Google Apps Script Configuration
echo ============================================
echo.

echo [INFO] You need to deploy the ISX License Manager as a Google Apps Script web app
echo [INFO] and provide the deployment URL here.
echo.
echo [INFO] Instructions:
echo [INFO]   1. Open Google Apps Script: https://script.google.com
echo [INFO]   2. Create new project named "ISX_License_Manager"
echo [INFO]   3. Paste the license management code from SCRATCH_CARD_LICENSE_IMPLEMENTATION_PLAN.md
echo [INFO]   4. Deploy as web app with "Execute as: Me" and "Access: Anyone"
echo [INFO]   5. Copy the web app URL
echo.

set /p APPS_SCRIPT_URL="Enter Google Apps Script URL: "

:: Validate URL format
echo !APPS_SCRIPT_URL! | findstr /C:"script.google.com/macros/s/" >nul
if errorlevel 1 (
    echo [ERROR] Invalid Apps Script URL format
    echo [ERROR] Expected format: https://script.google.com/macros/s/SCRIPT_ID/exec
    pause
    exit /b 1
)

echo [SUCCESS] Apps Script URL validated: !APPS_SCRIPT_URL!
echo.

:: ============================================
:: Step 2: Test Apps Script Connectivity
:: ============================================

echo ============================================
echo Step 2: Testing Apps Script Connectivity
echo ============================================
echo.

echo [INFO] Testing connection to Apps Script...
curl -s -o temp_response.txt -w "%%{http_code}" -X GET "!APPS_SCRIPT_URL!"
set /p HTTP_CODE=<temp_response.txt

if exist temp_response.txt (
    echo [INFO] Response content:
    type temp_response.txt
    del temp_response.txt
    echo.
)

if "!HTTP_CODE!"=="200" (
    echo [SUCCESS] Apps Script is accessible and responding
) else (
    echo [WARNING] Apps Script returned HTTP !HTTP_CODE!
    echo [WARNING] This might be normal if the script expects POST requests
    set /p CONTINUE="Continue with setup? (y/N): "
    if /i not "!CONTINUE!"=="y" (
        echo [INFO] Setup cancelled
        pause
        exit /b 1
    )
)
echo.

:: ============================================
:: Step 3: Google Sheets Credentials
:: ============================================

echo ============================================
echo Step 3: Google Sheets API Credentials
echo ============================================
echo.

if exist "credentials.json" (
    echo [INFO] Found existing credentials.json
    set /p REPLACE_CREDS="Replace with new credentials? (y/N): "
    if /i not "!REPLACE_CREDS!"=="y" (
        echo [INFO] Keeping existing credentials.json
        goto :skip_credentials
    )
)

echo [INFO] You need to provide Google Sheets API credentials
echo [INFO] These should be service account credentials in JSON format
echo.
echo [INFO] Instructions:
echo [INFO]   1. Go to Google Cloud Console: https://console.cloud.google.com
echo [INFO]   2. Enable Google Sheets API
echo [INFO]   3. Create service account credentials
echo [INFO]   4. Download JSON key file
echo [INFO]   5. Place it in this directory as 'credentials.json'
echo.

if not exist "credentials.json" (
    echo [ERROR] credentials.json not found
    echo [ERROR] Please place your Google service account credentials file here
    pause
    exit /b 1
)

:skip_credentials
echo [SUCCESS] Google Sheets credentials ready
echo.

:: ============================================
:: Step 4: Environment Configuration
:: ============================================

echo ============================================
echo Step 4: Environment Configuration
echo ============================================
echo.

:: Create .env file with scratch card configuration
echo [INFO] Creating .env file with scratch card configuration...

(
echo # ISX Pulse - Scratch Card Configuration
echo # Generated on %date% %time%
echo.
echo # Server Configuration
echo PORT=8080
echo HOST=0.0.0.0
echo ENV=production
echo.
echo # Logging
echo ISX_LOGGING_OUTPUT=file
echo LOG_LEVEL=info
echo.
echo # Google Apps Script Configuration
echo GOOGLE_APPS_SCRIPT_URL=!APPS_SCRIPT_URL!
echo.
echo # Google Sheets API
echo GOOGLE_APPLICATION_CREDENTIALS=credentials.json
echo.
echo # Scratch Card Features
echo ENABLE_SCRATCH_CARD_MODE=true
echo ENABLE_DEVICE_FINGERPRINTING=true
echo ENABLE_ONE_TIME_ACTIVATION=true
echo.
echo # Fingerprinting Configuration
echo FINGERPRINT_CACHE_TTL=1h
echo FINGERPRINT_INCLUDE_MAC=true
echo FINGERPRINT_INCLUDE_CPU=true
echo FINGERPRINT_INCLUDE_HOSTNAME=true
echo.
echo # Security & Rate Limiting
echo SCRATCH_CARD_MAX_ATTEMPTS_PER_HOUR=10
echo SCRATCH_CARD_BLOCK_DURATION_HOURS=24
echo SCRATCH_CARD_RATE_LIMIT_PER_IP=10
echo SCRATCH_CARD_VALIDATION_TIMEOUT=30s
echo.
echo # Apps Script Integration
echo APPS_SCRIPT_TIMEOUT=30s
echo APPS_SCRIPT_RETRY_COUNT=3
echo APPS_SCRIPT_RETRY_BACKOFF=2s
echo.
echo # Validation & Caching
echo VALIDATION_CACHE_ENABLED=true
echo VALIDATION_CACHE_TTL=5m
echo VALIDATION_CACHE_SIZE=1000
echo OFFLINE_VALIDATION_GRACE_PERIOD=24h
echo.
echo # Blacklist & Security
echo BLACKLIST_CHECK_ENABLED=true
echo BLACKLIST_AUTO_BAN_THRESHOLD=10
echo BLACKLIST_IP_TRACKING=true
echo BLACKLIST_DEVICE_TRACKING=true
echo.
echo # Audit & Monitoring
echo AUDIT_LOG_ENABLED=true
echo AUDIT_LOG_LEVEL=info
echo AUDIT_LOG_MAX_ENTRIES=10000
echo ACTIVATION_METRICS_ENABLED=true
echo.
echo # License Generator Configuration
echo LICENSE_GENERATOR_BATCH_SIZE=100
echo LICENSE_GENERATOR_FORMAT=ISX-XXXX-XXXX-XXXX
echo LICENSE_GENERATOR_ALPHABET=ABCDEFGHIJKLMNPQRSTUVWXYZ23456789
echo LICENSE_GENERATOR_DEFAULT_DURATION=1m
echo.
echo # Feature Flags
echo ENABLE_WEBSOCKET=true
echo ENABLE_METRICS=true
echo ENABLE_TRACING=false
echo.
echo # Production Security
echo DEBUG=false
echo SKIP_LICENSE_CHECK=false
) > .env

echo [SUCCESS] Created .env file with scratch card configuration
echo.

:: ============================================
:: Step 5: Validation
:: ============================================

echo ============================================
echo Step 5: Configuration Validation
echo ============================================
echo.

echo [INFO] Validating configuration...

:: Check credentials.json format
echo [INFO] Validating credentials.json format...
powershell -Command "try { Get-Content 'credentials.json' | ConvertFrom-Json | Out-Null; Write-Host '[SUCCESS] credentials.json is valid JSON' } catch { Write-Host '[ERROR] credentials.json is not valid JSON'; exit 1 }"
if errorlevel 1 (
    echo [ERROR] credentials.json validation failed
    pause
    exit /b 1
)

:: Check if .env was created successfully
if not exist ".env" (
    echo [ERROR] Failed to create .env file
    pause
    exit /b 1
)

echo [INFO] Configuration files validated successfully
echo.

:: ============================================
:: Step 6: Test Build
:: ============================================

echo ============================================
echo Step 6: Test Build
echo ============================================
echo.

set /p TEST_BUILD="Perform test build with new configuration? (Y/n): "
if /i "!TEST_BUILD!"=="n" goto :skip_build

echo [INFO] Performing test build...
call build.bat -target=clean >nul 2>&1
call build.bat -target=web -scratch-card

if errorlevel 1 (
    echo [ERROR] Test build failed
    echo [ERROR] Please check the configuration and try again
    pause
    exit /b 1
) else (
    echo [SUCCESS] Test build completed successfully
)

:skip_build
echo.

:: ============================================
:: Final Summary
:: ============================================

echo ==========================================
echo        SETUP SUMMARY
echo ==========================================
echo.
echo [SUCCESS] Scratch card credentials setup completed!
echo.
echo Configuration:
echo   - Apps Script URL: !APPS_SCRIPT_URL!
echo   - Credentials: credentials.json
echo   - Environment: .env (scratch card mode enabled)
echo.
echo Files Created/Updated:
echo   - .env (with scratch card configuration)
echo.
echo Next Steps:
echo   1. Deploy using: scripts\deploy-scratch-card.bat
echo   2. Generate initial scratch cards using license-generator.exe
echo   3. Test activation flow with a sample card
echo   4. Monitor Apps Script logs for any issues
echo.
echo Important Notes:
echo   - Keep credentials.json secure and private
echo   - Apps Script URL should not be shared publicly
echo   - Monitor activation attempts for suspicious activity
echo   - Regular backup of Google Sheets data recommended
echo.
echo ==========================================
echo.

echo [INFO] Setup completed successfully!
echo [INFO] Press any key to exit...
pause >nul