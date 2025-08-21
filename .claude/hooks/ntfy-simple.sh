#!/bin/bash

# Simple ntfy notification script for Claude Code
# Usage: ntfy-simple.sh [action-required|task-complete]

NTFY_TOPIC="https://ntfy.sh/ClaudeCodeNotifications"
NOTIFICATION_TYPE=$1

# Read JSON input from Claude Code
INPUT=$(cat)

# Extract session ID for tracking (fallback to grep if jq not available)
if command -v jq &> /dev/null; then
    SESSION_ID=$(echo "$INPUT" | jq -r '.session_id // "unknown"')
else
    # Fallback: use grep/sed to extract session_id
    SESSION_ID=$(echo "$INPUT" | grep -o '"session_id"[[:space:]]*:[[:space:]]*"[^"]*"' | sed 's/.*"session_id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')
    [ -z "$SESSION_ID" ] && SESSION_ID="unknown"
fi

case "$NOTIFICATION_TYPE" in
    "action-required")
        # Claude Code needs user input
        curl -X POST "$NTFY_TOPIC" \
            -H "Title: ISX Project - Action Required" \
            -H "Priority: 4" \
            -H "Tags: warning,bell" \
            -d "Claude Code needs your input on ISX Project (Session: ${SESSION_ID:0:8})"
        ;;
        
    "task-complete")
        # Claude Code finished all tasks
        # Try to extract some summary info from the input
        TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')
        
        curl -X POST "$NTFY_TOPIC" \
            -H "Title: ISX Project - Tasks Complete" \
            -H "Priority: 3" \
            -H "Tags: white_check_mark,done" \
            -d "Claude Code completed all tasks on ISX Project at $TIMESTAMP (Session: ${SESSION_ID:0:8})"
        ;;
        
    *)
        # Unknown notification type - log but don't fail
        echo "Unknown notification type: $NOTIFICATION_TYPE" >&2
        exit 0
        ;;
esac

# Always exit successfully to not interrupt Claude Code
exit 0