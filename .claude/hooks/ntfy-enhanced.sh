#!/bin/bash

# Enhanced ntfy notification script for Claude Code
# Handles multiple event types with contextual information

NTFY_TOPIC="https://ntfy.sh/ClaudeCodeNotifications"
EVENT_TYPE=$1

# Read JSON input from Claude Code
INPUT=$(cat)

# Extract common fields (with fallbacks for systems without jq)
if command -v jq &> /dev/null; then
    SESSION_ID=$(echo "$INPUT" | jq -r '.session_id // "unknown"')
    TOOL_NAME=$(echo "$INPUT" | jq -r '.tool_name // ""')
    AGENT_TYPE=$(echo "$INPUT" | jq -r '.subagent_type // ""')
    FILE_PATH=$(echo "$INPUT" | jq -r '.file_path // ""')
else
    # Fallback extraction without jq
    SESSION_ID=$(echo "$INPUT" | grep -o '"session_id"[[:space:]]*:[[:space:]]*"[^"]*"' | sed 's/.*"session_id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')
    TOOL_NAME=$(echo "$INPUT" | grep -o '"tool_name"[[:space:]]*:[[:space:]]*"[^"]*"' | sed 's/.*"tool_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')
    AGENT_TYPE=$(echo "$INPUT" | grep -o '"subagent_type"[[:space:]]*:[[:space:]]*"[^"]*"' | sed 's/.*"subagent_type"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')
    FILE_PATH=$(echo "$INPUT" | grep -o '"file_path"[[:space:]]*:[[:space:]]*"[^"]*"' | sed 's/.*"file_path"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')
    
    [ -z "$SESSION_ID" ] && SESSION_ID="unknown"
fi

TIMESTAMP=$(date '+%H:%M:%S')

case "$EVENT_TYPE" in
    "build-complete")
        # Build operation completed
        curl -X POST "$NTFY_TOPIC" \
            -H "Title: ISX Build Complete" \
            -H "Priority: 3" \
            -H "Tags: hammer,package" \
            -d "Build completed at $TIMESTAMP"
        ;;
        
    "test-complete")
        # Test execution completed
        curl -X POST "$NTFY_TOPIC" \
            -H "Title: ISX Tests Complete" \
            -H "Priority: 3" \
            -H "Tags: test_tube,check" \
            -d "Tests completed at $TIMESTAMP"
        ;;
        
    "agent-complete")
        # Specialized agent finished
        AGENT_MSG="Agent completed"
        case "$AGENT_TYPE" in
            "test-architect")
                AGENT_MSG="Test suite created"
                ;;
            "security-auditor")
                AGENT_MSG="Security audit complete"
                ;;
            "frontend-modernizer")
                AGENT_MSG="Frontend updates complete"
                ;;
            "deployment-orchestrator")
                AGENT_MSG="Deployment complete"
                ;;
            *)
                AGENT_MSG="$AGENT_TYPE completed"
                ;;
        esac
        
        curl -X POST "$NTFY_TOPIC" \
            -H "Title: ISX Agent Task Complete" \
            -H "Priority: 2" \
            -H "Tags: robot,sparkles" \
            -d "$AGENT_MSG at $TIMESTAMP"
        ;;
        
    "files-modified")
        # Multiple files were modified
        FILE_COUNT=$(echo "$INPUT" | grep -o '"file_path"' | wc -l)
        curl -X POST "$NTFY_TOPIC" \
            -H "Title: ISX Files Modified" \
            -H "Priority: 2" \
            -H "Tags: pencil2,file_folder" \
            -d "$FILE_COUNT files modified at $TIMESTAMP"
        ;;
        
    "session-start")
        # Session started or resumed
        curl -X POST "$NTFY_TOPIC" \
            -H "Title: ISX Session Started" \
            -H "Priority: 1" \
            -H "Tags: rocket,green_circle" \
            -d "Claude Code session started (${SESSION_ID:0:8})"
        ;;
        
    *)
        # Unknown event - don't fail silently
        echo "Unknown event type: $EVENT_TYPE" >&2
        ;;
esac

# Always exit successfully
exit 0