@echo off
:: ============================================
:: ISX Pulse - Scratch Card Deployment Script
:: ============================================
:: This script deploys ISX Pulse with scratch card license system enabled
:: Ensures proper configuration validation and health checks

setlocal enabledelayedexpansion

echo ==========================================
echo    ISX Pulse - Scratch Card Deployment   
echo    The Heartbeat of Iraqi Markets        
echo ==========================================
echo.

:: Set deployment configuration
set DEPLOYMENT_TYPE=scratch-card
set VALIDATION_REQUIRED=true
set HEALTH_CHECK_TIMEOUT=30

:: Check if we're in the correct directory
if not exist "build.go" (
    echo [ERROR] Must run from project root directory
    echo [ERROR] build.go not found
    pause
    exit /b 1
)

:: ============================================
:: Phase 1: Pre-Deployment Validation
:: ============================================

echo [INFO] Phase 1: Pre-Deployment Validation
echo.

:: Check for required environment variables
echo [INFO] Checking environment configuration...
set MISSING_VARS=
if not defined GOOGLE_APPS_SCRIPT_URL (
    set MISSING_VARS=!MISSING_VARS! GOOGLE_APPS_SCRIPT_URL
)

if defined MISSING_VARS (
    echo [ERROR] Missing required environment variables: !MISSING_VARS!
    echo [ERROR] Please set these variables before deployment:
    echo.
    echo   GOOGLE_APPS_SCRIPT_URL - Your Google Apps Script web app URL
    echo.
    echo   Example:
    echo   set GOOGLE_APPS_SCRIPT_URL=https://script.google.com/macros/s/YOUR_SCRIPT_ID/exec
    echo.
    pause
    exit /b 1
)

:: Validate Apps Script URL format
echo [INFO] Validating Apps Script URL format...
echo !GOOGLE_APPS_SCRIPT_URL! | findstr /C:"script.google.com/macros/s/" >nul
if errorlevel 1 (
    echo [WARNING] Apps Script URL doesn't match expected format
    echo [WARNING] Expected: https://script.google.com/macros/s/SCRIPT_ID/exec
    echo [WARNING] Current: !GOOGLE_APPS_SCRIPT_URL!
    echo.
    set /p CONTINUE="Continue anyway? (y/N): "
    if /i not "!CONTINUE!"=="y" (
        echo [INFO] Deployment cancelled
        pause
        exit /b 1
    )
)

:: Check for .env file or create from template
if not exist ".env" (
    if exist ".env.example" (
        echo [INFO] Creating .env from template...
        copy ".env.example" ".env" >nul
        echo [WARNING] Please configure .env file with your specific values
        echo [WARNING] Especially GOOGLE_APPS_SCRIPT_URL and other scratch card settings
        pause
    ) else (
        echo [ERROR] No .env file found and no .env.example to copy from
        pause
        exit /b 1
    )
)

:: Check for Go environment
echo [INFO] Checking Go environment...
go version >nul 2>&1
if errorlevel 1 (
    echo [ERROR] Go is not installed or not in PATH
    echo [ERROR] Please install Go from https://golang.org/dl/
    pause
    exit /b 1
)

:: Check for Node.js (for frontend build)
echo [INFO] Checking Node.js environment...
npm --version >nul 2>&1
if errorlevel 1 (
    echo [ERROR] Node.js/npm is not installed or not in PATH
    echo [ERROR] Please install Node.js from https://nodejs.org/
    pause
    exit /b 1
)

echo [SUCCESS] Pre-deployment validation completed
echo.

:: ============================================
:: Phase 2: Test Apps Script Connectivity
:: ============================================

echo [INFO] Phase 2: Testing Apps Script Connectivity
echo.

:: Test the Apps Script endpoint
echo [INFO] Testing Apps Script endpoint: !GOOGLE_APPS_SCRIPT_URL!
curl -s -o nul -w "%%{http_code}" -X GET "!GOOGLE_APPS_SCRIPT_URL!" > temp_response.txt
set /p HTTP_CODE=<temp_response.txt
del temp_response.txt >nul 2>&1

if "!HTTP_CODE!"=="200" (
    echo [SUCCESS] Apps Script endpoint is accessible
) else (
    echo [WARNING] Apps Script endpoint returned HTTP !HTTP_CODE!
    echo [WARNING] This might be normal if the script requires POST requests
    echo [INFO] Continuing with deployment...
)
echo.

:: ============================================
:: Phase 3: Backup Current System
:: ============================================

echo [INFO] Phase 3: Creating system backup
echo.

:: Create backup directory with timestamp
for /f "tokens=2 delims==" %%a in ('wmic OS Get localdatetime /value') do set "dt=%%a"
set "YY=%dt:~2,2%" & set "YYYY=%dt:~0,4%" & set "MM=%dt:~4,2%" & set "DD=%dt:~6,2%"
set "HH=%dt:~8,2%" & set "Min=%dt:~10,2%" & set "Sec=%dt:~12,2%"
set "datestamp=%YYYY%%MM%%DD%_%HH%%Min%%Sec%"

set BACKUP_DIR=backup_%datestamp%
echo [INFO] Creating backup in %BACKUP_DIR%...

if not exist "%BACKUP_DIR%" mkdir "%BACKUP_DIR%"

:: Backup current dist directory if exists
if exist "dist" (
    echo [INFO] Backing up current dist directory...
    xcopy "dist" "%BACKUP_DIR%\dist" /E /I /Q >nul
)

:: Backup current license files
if exist "license.dat" (
    echo [INFO] Backing up license.dat...
    copy "license.dat" "%BACKUP_DIR%\" >nul
)

if exist "dist\license.dat" (
    echo [INFO] Backing up dist\license.dat...
    copy "dist\license.dat" "%BACKUP_DIR%\" >nul
)

echo [SUCCESS] Backup created in %BACKUP_DIR%
echo.

:: ============================================
:: Phase 4: Build with Scratch Card Features
:: ============================================

echo [INFO] Phase 4: Building with scratch card features enabled
echo.

:: Set environment variables for build
set ENABLE_SCRATCH_CARD_MODE=true
set ENABLE_DEVICE_FINGERPRINTING=true
set ENABLE_ONE_TIME_ACTIVATION=true

echo [INFO] Building with configuration:
echo   - Scratch Card Mode: ENABLED
echo   - Device Fingerprinting: ENABLED  
echo   - One-Time Activation: ENABLED
echo   - Apps Script URL: !GOOGLE_APPS_SCRIPT_URL!
echo.

:: Clean previous build
echo [INFO] Cleaning previous build artifacts...
call build.bat -target=clean

:: Build all components with scratch card features
echo [INFO] Building all components with scratch card features...
call build.bat -target=all -scratch-card -apps-script-url="!GOOGLE_APPS_SCRIPT_URL!"

if errorlevel 1 (
    echo [ERROR] Build failed!
    echo [ERROR] Check the error messages above
    echo [INFO] Backup is available in %BACKUP_DIR%
    pause
    exit /b 1
)

echo [SUCCESS] Build completed successfully
echo.

:: ============================================
:: Phase 5: Deployment Validation
:: ============================================

echo [INFO] Phase 5: Deployment validation
echo.

:: Check if all required executables were built
set MISSING_EXES=
if not exist "dist\ISXPulse.exe" set MISSING_EXES=!MISSING_EXES! ISXPulse.exe
if not exist "dist\scraper.exe" set MISSING_EXES=!MISSING_EXES! scraper.exe
if not exist "dist\processor.exe" set MISSING_EXES=!MISSING_EXES! processor.exe
if not exist "dist\indexcsv.exe" set MISSING_EXES=!MISSING_EXES! indexcsv.exe
if not exist "dist\license-generator.exe" set MISSING_EXES=!MISSING_EXES! license-generator.exe

if defined MISSING_EXES (
    echo [ERROR] Missing executables: !MISSING_EXES!
    echo [ERROR] Build may have failed
    pause
    exit /b 1
)

:: Check file sizes (basic sanity check)
echo [INFO] Validating executable sizes...
for %%f in (dist\*.exe) do (
    set "size=0"
    for /f "usebackq" %%s in (`powershell "(Get-Item '%%f').length"`) do set "size=%%s"
    if !size! LSS 1048576 (
        echo [WARNING] %%f seems unusually small (!size! bytes)
    ) else (
        echo [INFO] %%f: !size! bytes
    )
)

:: Copy configuration files to dist
echo [INFO] Copying configuration files...
if exist ".env.example" copy ".env.example" "dist\" >nul
if exist "sheets-config.json.example" copy "sheets-config.json.example" "dist\" >nul

:: Create scratch card specific readme
echo [INFO] Creating deployment documentation...
echo # ISX Pulse - Scratch Card Deployment > "dist\SCRATCH_CARD_DEPLOYMENT.txt"
echo. >> "dist\SCRATCH_CARD_DEPLOYMENT.txt"
echo Deployment Date: %date% %time% >> "dist\SCRATCH_CARD_DEPLOYMENT.txt"
echo Apps Script URL: !GOOGLE_APPS_SCRIPT_URL! >> "dist\SCRATCH_CARD_DEPLOYMENT.txt"
echo. >> "dist\SCRATCH_CARD_DEPLOYMENT.txt"
echo Configuration: >> "dist\SCRATCH_CARD_DEPLOYMENT.txt"
echo - Scratch Card Mode: ENABLED >> "dist\SCRATCH_CARD_DEPLOYMENT.txt"
echo - Device Fingerprinting: ENABLED >> "dist\SCRATCH_CARD_DEPLOYMENT.txt"
echo - One-Time Activation: ENABLED >> "dist\SCRATCH_CARD_DEPLOYMENT.txt"
echo. >> "dist\SCRATCH_CARD_DEPLOYMENT.txt"
echo For troubleshooting, see backup in: %BACKUP_DIR% >> "dist\SCRATCH_CARD_DEPLOYMENT.txt"

echo [SUCCESS] Deployment validation completed
echo.

:: ============================================
:: Phase 6: Health Check
:: ============================================

echo [INFO] Phase 6: System health check
echo.

:: Test if the built executable can start (basic check)
echo [INFO] Testing executable startup...
pushd dist
timeout %HEALTH_CHECK_TIMEOUT% ISXPulse.exe --version >nul 2>&1
set HEALTH_RESULT=!errorlevel!
popd

if %HEALTH_RESULT% == 0 (
    echo [SUCCESS] Health check passed
) else (
    echo [WARNING] Health check returned code %HEALTH_RESULT%
    echo [WARNING] This might be normal if --version is not implemented
)

:: Test license generator
echo [INFO] Testing license generator...
pushd dist
license-generator.exe --help >nul 2>&1
set GENERATOR_RESULT=!errorlevel!
popd

if %GENERATOR_RESULT% == 0 (
    echo [SUCCESS] License generator is working
) else (
    echo [WARNING] License generator test failed with code %GENERATOR_RESULT%
)

echo.

:: ============================================
:: Phase 7: Final Summary
:: ============================================

echo ==========================================
echo        DEPLOYMENT SUMMARY
echo ==========================================
echo.
echo [INFO] Scratch card deployment completed successfully!
echo.
echo Build Configuration:
echo   - Target Directory: dist\
echo   - Scratch Card Mode: ENABLED
echo   - Apps Script URL: !GOOGLE_APPS_SCRIPT_URL!
echo   - Backup Location: %BACKUP_DIR%
echo.
echo Built Components:
echo   - ISXPulse.exe (Main server with scratch card support)
echo   - scraper.exe (Data scraper)
echo   - processor.exe (Data processor)  
echo   - indexcsv.exe (CSV indexer)
echo   - license-generator.exe (Scratch card generator)
echo.
echo Next Steps:
echo   1. Test the system with a sample scratch card
echo   2. Generate initial batch of scratch cards using license-generator.exe
echo   3. Monitor the Apps Script logs for any issues
echo   4. Set up monitoring for activation metrics
echo.
echo Configuration Files:
echo   - .env (update with production values)
echo   - dist\.env.example (template for reference)
echo   - dist\SCRATCH_CARD_DEPLOYMENT.txt (deployment record)
echo.
echo Troubleshooting:
echo   - Backup available in: %BACKUP_DIR%
echo   - Check logs in: dist\logs\
echo   - Apps Script URL: !GOOGLE_APPS_SCRIPT_URL!
echo.
echo ==========================================
echo.

:: Offer to start the server
set /p START_SERVER="Start ISX Pulse server now? (y/N): "
if /i "!START_SERVER!"=="y" (
    echo [INFO] Starting ISX Pulse server...
    pushd dist
    start ISXPulse.exe
    popd
    echo [INFO] Server started in background
    echo [INFO] Check dist\logs\ for server logs
)

echo.
echo [SUCCESS] Scratch card deployment completed!
echo [INFO] Press any key to exit...
pause >nul