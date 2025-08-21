# Run ISX Daily Reports Scrapper Server
# This script launches the web-licensed server in a new PowerShell window

$scriptPath = Split-Path -Parent $MyInvocation.MyCommand.Path
$exePath = Join-Path $scriptPath "release\web-licensed.exe"
$logPath = Join-Path $scriptPath "release\logs\app.log"
$logsDir = Join-Path $scriptPath "release\logs"

# Check if the executable exists
if (-not (Test-Path $exePath)) {
    Write-Host "Error: web-licensed.exe not found at $exePath" -ForegroundColor Red
    Write-Host "Please run build.bat first to build the application." -ForegroundColor Yellow
    Read-Host "Press Enter to exit"
    exit 1
}

# Clear logs before starting
Write-Host "Clearing previous logs..." -ForegroundColor Yellow
if (Test-Path $logsDir) {
    Remove-Item "$logsDir\*.log" -Force -ErrorAction SilentlyContinue
    Write-Host "Logs cleared successfully." -ForegroundColor Green
} else {
    Write-Host "No logs directory found, skipping log cleanup." -ForegroundColor Gray
}

# Launch the server in a new PowerShell window
Write-Host "Starting ISX Daily Reports Scrapper server..." -ForegroundColor Green
Write-Host "Server will run on http://localhost:8080" -ForegroundColor Cyan

# Start the server in a new PowerShell window that stays open
Start-Process powershell.exe -ArgumentList "-NoExit", "-Command", "& '$exePath'"

Write-Host "Server launched in a new window!" -ForegroundColor Green