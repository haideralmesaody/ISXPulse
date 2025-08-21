param(
    [string]$EventType,
    [string]$TaskName = "",
    [string]$Details = ""
)

$ntfyTopic = "https://ntfy.sh/ClaudeCodeNotifications"
$timestamp = Get-Date -Format "HH:mm:ss"
$dayOfWeek = (Get-Date).DayOfWeek
$timeOfDay = switch ((Get-Date).Hour) {
    {$_ -lt 12} { "morning" }
    {$_ -lt 17} { "afternoon" }
    {$_ -lt 21} { "evening" }
    default { "night" }
}

# Try to extract task info from environment or file if not provided
if (-not $TaskName) {
    # Check if there's a current task file (Claude might write this)
    $taskFile = "C:\ISXDailyReportsScrapper\.claude\current-task.txt"
    if (Test-Path $taskFile) {
        $TaskName = Get-Content $taskFile -First 1 -ErrorAction SilentlyContinue
    }
}

# Initialize notification parameters
$title = ""
$message = ""
$priority = 3
$tags = ""

switch ($EventType) {
    "session-start" {
        $greeting = switch ($timeOfDay) {
            "morning" { "Good morning!" }
            "afternoon" { "Good afternoon!" }
            "evening" { "Good evening!" }
            "night" { "Working late?" }
        }
        $title = "ISX Pulse Development Started"
        $message = "$greeting Your Claude Code session started at $timestamp. Ready to work on ISX Pulse!"
        $priority = 1
        $tags = "rocket,green_circle"
    }
    "build-complete" {
        $buildType = if ($TaskName) { " for: $TaskName" } else { "" }
        $title = "Build Successful!"
        $message = "Great news! ISX Pulse build completed$buildType at $timestamp. Application is ready to run."
        $priority = 3
        $tags = "hammer,package,white_check_mark"
    }
    "test-complete" {
        $testContext = if ($TaskName) { " for: $TaskName" } else { "" }
        $title = "Tests Finished"
        $message = "All tests completed$testContext at $timestamp. Check terminal for results!"
        $priority = 3
        $tags = "test_tube,checkered_flag"
    }
    "agent-complete" {
        $agentTask = if ($TaskName) { "Task: '$TaskName'" } else { "Agent task" }
        $title = "AI Assistant Finished"
        $message = "$agentTask completed successfully at $timestamp. Results ready for review!"
        $priority = 2
        $tags = "robot,sparkles"
    }
    "files-modified" {
        $modContext = if ($TaskName) { " for: $TaskName" } else { "" }
        $fileCount = if ($Details) { " ($Details)" } else { "" }
        $title = "Code Updated"
        $message = "Files modified$modContext at $timestamp$fileCount. Your codebase is evolving!"
        $priority = 2
        $tags = "pencil2,file_folder"
    }
    "action-required" {
        $actionContext = if ($TaskName) { "Task: '$TaskName' - " } else { "" }
        $actionDetails = if ($Details) { ": $Details" } else { "" }
        $title = "Your Input Needed!"
        $message = "${actionContext}Claude needs your help at $timestamp$actionDetails. Please check the terminal."
        $priority = 4
        $tags = "warning,bell,eyes"
    }
    "task-complete" {
        $completedTask = if ($TaskName) { "'$TaskName'" } else { "Task" }
        $taskDetails = if ($Details) { " - $Details" } else { "" }
        $encouragement = @(
            "Excellent work!",
            "Another one done!",
            "Great progress!",
            "You're on fire!",
            "Nicely done!",
            "Well executed!",
            "Mission accomplished!",
            "Keep it up!",
            "Fantastic job!"
        ) | Get-Random
        $title = "Task Completed!"
        $message = "$encouragement $completedTask finished at $timestamp$taskDetails. Ready for the next challenge?"
        $priority = 2
        $tags = "white_check_mark,tada,trophy"
    }
}

if ($message) {
    try {
        # Log the notification for debugging
        $logEntry = "[$timestamp] Event: $EventType"
        if ($TaskName) { $logEntry += ", Task: $TaskName" }
        if ($Details) { $logEntry += ", Details: $Details" }
        Add-Content -Path "C:\ISXDailyReportsScrapper\.claude\hooks\notification.log" -Value $logEntry -ErrorAction SilentlyContinue
        
        # Create headers for ntfy
        $headers = @{
            "Title" = $title
            "Priority" = $priority.ToString()
            "Tags" = $tags
        }
        
        # Send notification using headers format (not JSON)
        # This sends the message as plain text with metadata in headers
        Invoke-RestMethod -Uri $ntfyTopic -Method Post -Headers $headers -Body $message
        
        Write-Host "Notification sent: $EventType - $title"
    } catch {
        Write-Host "Failed to send notification: $_"
        
        # Fallback to simple format if headers fail
        try {
            $fallbackMessage = "$title - $message"
            Invoke-RestMethod -Uri $ntfyTopic -Method Post -Body $fallbackMessage
            Write-Host "Notification sent (fallback): $EventType"
        } catch {
            Write-Host "Fallback also failed: $_"
        }
    }
}

exit 0