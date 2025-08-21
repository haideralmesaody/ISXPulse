@echo off
echo.
echo ========================================
echo   ISX Reports Migration Tool
echo   Reorganizing reports into new structure
echo ========================================
echo.

REM Check if reports directory is provided
if "%1"=="" (
    echo Usage: migrate-reports.bat ^<reports-directory^>
    echo Example: migrate-reports.bat dist\data\reports
    echo.
    exit /b 1
)

REM Run the migration script
echo Starting migration for: %1
echo.
go run scripts\migrate-reports.go %1

if %ERRORLEVEL% NEQ 0 (
    echo.
    echo Migration failed with error code %ERRORLEVEL%
    exit /b %ERRORLEVEL%
)

echo.
echo Migration completed successfully!
echo.
pause