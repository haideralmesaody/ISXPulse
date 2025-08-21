@echo off
echo ==========================================
echo Sanitizing Credentials for Git Commit
echo ==========================================
echo.

:: Backup current manager.go
echo [1/3] Backing up current manager.go...
if not exist scripts\credentials (
    mkdir scripts\credentials
)
copy dev\internal\license\manager.go scripts\credentials\manager.go.real >nul 2>&1
if errorlevel 1 (
    echo [ERROR] Failed to backup manager.go
    exit /b 1
)
echo [SUCCESS] Backed up to scripts\credentials\manager.go.real

:: Create sanitized version
echo.
echo [2/3] Creating sanitized version...
powershell -Command "(Get-Content 'dev\internal\license\manager.go') -replace '\"private_key_id\": \"[^\"]+\"', '\"private_key_id\": \"PRIVATE_KEY_ID_PLACEHOLDER\"' -replace '\"private_key\": \"[^\"]+\"', '\"private_key\": \"PRIVATE_KEY_PLACEHOLDER\"' -replace '\"client_id\": \"[^\"]+\"', '\"client_id\": \"CLIENT_ID_PLACEHOLDER\"' -replace '\"client_email\": \"[^\"]+\"', '\"client_email\": \"CLIENT_EMAIL_PLACEHOLDER\"' -replace 'sheetID := \"[^\"]+\"', 'sheetID := \"SHEET_ID_PLACEHOLDER\"' | Set-Content 'dev\internal\license\manager.go'"

if errorlevel 1 (
    echo [ERROR] Failed to sanitize manager.go
    exit /b 1
)
echo [SUCCESS] Created sanitized version

:: Verify sanitization
echo.
echo [3/3] Verifying sanitization...
findstr /C:"PLACEHOLDER" dev\internal\license\manager.go >nul
if errorlevel 1 (
    echo [ERROR] Sanitization failed - no placeholders found
    echo         Restoring original file...
    copy scripts\credentials\manager.go.real dev\internal\license\manager.go >nul 2>&1
    exit /b 1
)

:: Check for any remaining sensitive data
findstr /C:"4d17ff4" dev\internal\license\manager.go >nul
if not errorlevel 1 (
    echo [ERROR] Sanitization incomplete - sensitive data still found
    echo         Restoring original file...
    copy scripts\credentials\manager.go.real dev\internal\license\manager.go >nul 2>&1
    exit /b 1
)

echo [SUCCESS] Sanitization complete
echo.
echo ==========================================
echo Ready for Git commit
echo After push, run restore-credentials.bat
echo ==========================================