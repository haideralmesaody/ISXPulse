@echo off
echo Testing notification...
curl -X POST "https://ntfy.sh/ClaudeCodeNotifications" -H "Title: Test from Batch" -d "If you see this, hooks can send notifications"
echo Done!