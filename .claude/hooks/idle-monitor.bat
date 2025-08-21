@echo off
setlocal enabledelayedexpansion

REM Idle session monitor for Claude Code
REM This would need to be triggered by a timer/scheduler

set NTFY_TOPIC=https://ntfy.sh/ClaudeCodeNotifications
set EVENT_TYPE=%1
set IDLE_MINUTES=%2

if "%EVENT_TYPE%"=="idle-warning" (
    curl -X POST "%NTFY_TOPIC%" ^
        -H "Title: ISX Session Idle" ^
        -H "Priority: 3" ^
        -H "Tags: hourglass,warning" ^
        -d "Session has been idle for %IDLE_MINUTES% minutes"
    goto :end
)

if "%EVENT_TYPE%"=="idle-timeout" (
    curl -X POST "%NTFY_TOPIC%" ^
        -H "Title: ISX Session Timeout Warning" ^
        -H "Priority: 4" ^
        -H "Tags: alarm_clock,x" ^
        -d "Session will timeout soon - %IDLE_MINUTES% minutes idle"
    goto :end
)

if "%EVENT_TYPE%"=="session-resume" (
    curl -X POST "%NTFY_TOPIC%" ^
        -H "Title: ISX Session Resumed" ^
        -H "Priority: 2" ^
        -H "Tags: play_button,green_circle" ^
        -d "Session activity resumed after idle period"
    goto :end
)

:end
exit /b 0