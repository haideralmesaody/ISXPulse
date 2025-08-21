@echo off
REM Windows wrapper for ntfy-simple.sh
REM This ensures proper execution on Windows systems

set EVENT_TYPE=%1
set SCRIPT_DIR=%~dp0
set SCRIPT_PATH=%SCRIPT_DIR%ntfy-simple.sh

REM Use bash to execute the script, passing stdin through
bash "%SCRIPT_PATH%" %EVENT_TYPE%

exit /b 0