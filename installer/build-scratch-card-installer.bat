@echo off
:: ============================================
:: ISX Pulse - Scratch Card Installer Builder
:: ============================================
:: Builds the Inno Setup installer for the scratch card edition

setlocal enabledelayedexpansion

echo ==========================================
echo  ISX Pulse - Scratch Card Installer Build
echo     The Heartbeat of Iraqi Markets       
echo ==========================================
echo.

:: Check if we're in the installer directory
if not exist "scratch-card-installer.iss" (
    echo [ERROR] Must run from installer directory
    echo [ERROR] scratch-card-installer.iss not found
    pause
    exit /b 1
)

:: Check for Inno Setup
echo [INFO] Checking for Inno Setup compiler...
set INNO_SETUP_PATH=
if exist "C:\Program Files (x86)\Inno Setup 6\ISCC.exe" (
    set INNO_SETUP_PATH=C:\Program Files (x86)\Inno Setup 6\ISCC.exe
) else if exist "C:\Program Files\Inno Setup 6\ISCC.exe" (
    set INNO_SETUP_PATH=C:\Program Files\Inno Setup 6\ISCC.exe
) else (
    echo [ERROR] Inno Setup not found
    echo [ERROR] Please install Inno Setup 6 from https://jrsoftware.org/isinfo.php
    pause
    exit /b 1
)

echo [SUCCESS] Found Inno Setup: !INNO_SETUP_PATH!

:: Check if dist directory exists with required files
echo [INFO] Checking build prerequisites...
if not exist "..\dist\ISXPulse.exe" (
    echo [ERROR] ISXPulse.exe not found in dist directory
    echo [ERROR] Please build the project first using:
    echo [ERROR]   scripts\deploy-scratch-card.bat
    pause
    exit /b 1
)

if not exist "..\dist\license-generator.exe" (
    echo [ERROR] license-generator.exe not found in dist directory
    echo [ERROR] Please ensure scratch card build completed successfully
    pause
    exit /b 1
)

:: Verify all components are present
set MISSING_FILES=
if not exist "..\dist\scraper.exe" set MISSING_FILES=!MISSING_FILES! scraper.exe
if not exist "..\dist\processor.exe" set MISSING_FILES=!MISSING_FILES! processor.exe
if not exist "..\dist\indexcsv.exe" set MISSING_FILES=!MISSING_FILES! indexcsv.exe

if defined MISSING_FILES (
    echo [WARNING] Some components missing: !MISSING_FILES!
    set /p CONTINUE="Continue with installer build? (y/N): "
    if /i not "!CONTINUE!"=="y" (
        echo [INFO] Installer build cancelled
        pause
        exit /b 1
    )
)

:: Create output directory if it doesn't exist
if not exist "..\dist\installer" (
    echo [INFO] Creating installer output directory...
    mkdir "..\dist\installer"
)

:: Check for icon file
if not exist "assets\isx-app-icon.ico" (
    echo [WARNING] Icon file not found: assets\isx-app-icon.ico
    echo [INFO] Installer will use default icon
)

:: Backup any existing installer
for %%f in (..\dist\installer\ISXPulse-ScratchCard-Setup-*.exe) do (
    set EXISTING_INSTALLER=%%f
    if exist "!EXISTING_INSTALLER!" (
        echo [INFO] Backing up existing installer...
        move "!EXISTING_INSTALLER!" "!EXISTING_INSTALLER!.bak" >nul
    )
)

:: Build the installer
echo [INFO] Building scratch card installer...
echo [INFO] This may take a few minutes...
echo.

"!INNO_SETUP_PATH!" scratch-card-installer.iss

if errorlevel 1 (
    echo [ERROR] Installer build failed!
    echo [ERROR] Check the Inno Setup compiler output above for details
    pause
    exit /b 1
)

:: Check if installer was created
set INSTALLER_FILE=
for %%f in (..\dist\installer\ISXPulse-ScratchCard-Setup-*.exe) do (
    set INSTALLER_FILE=%%f
)

if not defined INSTALLER_FILE (
    echo [ERROR] Installer file was not created
    echo [ERROR] Check for errors in the build process
    pause
    exit /b 1
)

:: Get installer file size
for %%A in ("!INSTALLER_FILE!") do set SIZE=%%~zA
set /a SIZE_MB=!SIZE! / 1048576

echo.
echo ==========================================
echo   SCRATCH CARD INSTALLER BUILD COMPLETE!
echo ==========================================
echo.
echo Installer File: !INSTALLER_FILE!
echo Size: ~!SIZE_MB! MB
echo.
echo Features Included:
echo - ISX Pulse Server with scratch card support
echo - License generator and management tools
echo - Comprehensive documentation
echo - Configuration examples and templates
echo - Deployment and setup scripts
echo - Windows installer with guided setup
echo.
echo Installer Capabilities:
echo - Guided Google Apps Script URL configuration
echo - Automatic .env file generation
echo - Windows Firewall configuration (optional)
echo - Desktop and Start Menu shortcuts
echo - Post-installation setup wizard
echo - Component selection (full/server/generator)
echo.
echo Distribution Notes:
echo - Installer requires Windows 10/11 (64-bit)
echo - Administrative privileges needed for installation
echo - Internet connection required for Apps Script setup
echo - Google account and credentials needed
echo.
echo Testing Recommendations:
echo 1. Test installer on clean Windows system
echo 2. Verify all components install correctly
echo 3. Test Google Apps Script URL configuration
echo 4. Validate license generation and activation
echo 5. Check firewall rule creation (if selected)
echo.
echo ==========================================
echo.

:: Offer to test the installer
set /p TEST_INSTALLER="Run installer now for testing? (y/N): "
if /i "!TEST_INSTALLER!"=="y" (
    echo [INFO] Starting installer in test mode...
    start "ISX Pulse Installer" "!INSTALLER_FILE!"
)

:: Offer to open output directory
set /p OPEN_DIR="Open installer directory? (Y/n): "
if /i not "!OPEN_DIR!"=="n" (
    start "" "..\dist\installer"
)

echo.
echo [SUCCESS] Scratch card installer build completed!
echo [INFO] Installer ready for distribution
echo [INFO] Press any key to exit...
pause >nul