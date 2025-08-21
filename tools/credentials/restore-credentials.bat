@echo off
echo ==========================================
echo Restoring Credentials from Backup
echo ==========================================
echo.

:: Check if backup exists
if not exist scripts\credentials\manager.go.real (
    echo [ERROR] No backup found at scripts\credentials\manager.go.real
    echo         Run sanitize-credentials.bat first to create backup
    exit /b 1
)

:: Restore from backup
echo [1/2] Restoring manager.go from backup...
copy scripts\credentials\manager.go.real dev\internal\license\manager.go >nul 2>&1
if errorlevel 1 (
    echo [ERROR] Failed to restore manager.go
    exit /b 1
)
echo [SUCCESS] Restored manager.go

:: Verify restoration
echo.
echo [2/2] Verifying restoration...
findstr /C:"isxportfolio" dev\internal\license\manager.go >nul
if errorlevel 1 (
    echo [ERROR] Restoration failed - credentials not found
    exit /b 1
)

:: Check that placeholders are gone
findstr /C:"PLACEHOLDER" dev\internal\license\manager.go >nul
if not errorlevel 1 (
    echo [ERROR] Restoration incomplete - placeholders still present
    exit /b 1
)

echo [SUCCESS] Credentials restored successfully
echo.
echo ==========================================
echo Ready for development and testing
echo ==========================================