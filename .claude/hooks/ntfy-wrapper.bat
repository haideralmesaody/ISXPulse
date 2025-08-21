@echo off
REM Enhanced wrapper that passes event type and optional task name/details
REM Usage: ntfy-wrapper.bat <event-type> [task-name] [details]

set EVENT_TYPE=%1
set TASK_NAME=%~2
set DETAILS=%~3

echo [%DATE% %TIME%] ntfy-wrapper called with: Event=%EVENT_TYPE%, Task=%TASK_NAME%, Details=%DETAILS% >> C:\ISXDailyReportsScrapper\.claude\hooks\hook-debug.log

if "%TASK_NAME%"=="" (
    powershell -ExecutionPolicy Bypass -File "C:\ISXDailyReportsScrapper\.claude\hooks\ntfy-powershell.ps1" -EventType "%EVENT_TYPE%"
) else if "%DETAILS%"=="" (
    powershell -ExecutionPolicy Bypass -File "C:\ISXDailyReportsScrapper\.claude\hooks\ntfy-powershell.ps1" -EventType "%EVENT_TYPE%" -TaskName "%TASK_NAME%"
) else (
    powershell -ExecutionPolicy Bypass -File "C:\ISXDailyReportsScrapper\.claude\hooks\ntfy-powershell.ps1" -EventType "%EVENT_TYPE%" -TaskName "%TASK_NAME%" -Details "%DETAILS%"
)

echo [%DATE% %TIME%] ntfy-wrapper finished: %EVENT_TYPE% >> C:\ISXDailyReportsScrapper\.claude\hooks\hook-debug.log
exit /b 0