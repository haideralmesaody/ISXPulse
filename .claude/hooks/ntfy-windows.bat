@echo off
REM Add debug output to a log file
echo [%DATE% %TIME%] Hook called with: %1 >> C:\ISXDailyReportsScrapper\.claude\hooks\hook-debug.log
setlocal enabledelayedexpansion

REM Native Windows notification script for Claude Code
REM Sends notifications via ntfy.sh using curl

set NTFY_TOPIC=https://ntfy.sh/ClaudeCodeNotifications
set EVENT_TYPE=%1
set TIMESTAMP=%TIME:~0,8%

REM Simple event handling
if "%EVENT_TYPE%"=="session-start" (
    curl -X POST "%NTFY_TOPIC%" ^
        -H "Title: ISX Session Started" ^
        -H "Priority: 1" ^
        -H "Tags: rocket,green_circle" ^
        -d "Claude Code session started at %TIMESTAMP%"
    goto :end
)

if "%EVENT_TYPE%"=="build-complete" (
    curl -X POST "%NTFY_TOPIC%" ^
        -H "Title: ISX Build Complete" ^
        -H "Priority: 3" ^
        -H "Tags: hammer,package" ^
        -d "Build completed at %TIMESTAMP%"
    goto :end
)

if "%EVENT_TYPE%"=="test-complete" (
    curl -X POST "%NTFY_TOPIC%" ^
        -H "Title: ISX Tests Complete" ^
        -H "Priority: 3" ^
        -H "Tags: test_tube,check" ^
        -d "Tests completed at %TIMESTAMP%"
    goto :end
)

if "%EVENT_TYPE%"=="agent-complete" (
    curl -X POST "%NTFY_TOPIC%" ^
        -H "Title: ISX Agent Task Complete" ^
        -H "Priority: 2" ^
        -H "Tags: robot,sparkles" ^
        -d "Agent task completed at %TIMESTAMP%"
    goto :end
)

if "%EVENT_TYPE%"=="files-modified" (
    curl -X POST "%NTFY_TOPIC%" ^
        -H "Title: ISX Files Modified" ^
        -H "Priority: 2" ^
        -H "Tags: pencil2,file_folder" ^
        -d "Files modified at %TIMESTAMP%"
    goto :end
)

if "%EVENT_TYPE%"=="action-required" (
    curl -X POST "%NTFY_TOPIC%" ^
        -H "Title: ISX Action Required" ^
        -H "Priority: 4" ^
        -H "Tags: warning,bell" ^
        -d "Your attention is needed at %TIMESTAMP%"
    goto :end
)

if "%EVENT_TYPE%"=="task-complete" (
    curl -X POST "%NTFY_TOPIC%" ^
        -H "Title: ISX Task Complete" ^
        -H "Priority: 2" ^
        -H "Tags: white_check_mark,tada" ^
        -d "Task completed at %TIMESTAMP%"
    goto :end
)

:end
exit /b 0