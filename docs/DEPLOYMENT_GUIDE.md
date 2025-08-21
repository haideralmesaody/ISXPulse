# ISX Daily Reports Scrapper - Windows Deployment Guide

**Version**: 2.0 | **Target Platform**: Windows Server 2019/2022, Windows 10/11 | **Last Updated**: 2025-01-31

## Executive Summary

This deployment guide provides comprehensive instructions for deploying the ISX Daily Reports Scrapper on Windows environments. The application is designed as a self-contained system with embedded Next.js frontend, enterprise license management, and secure credential handling.

### Deployment Highlights
- ✅ **Single Binary Deployment** - All components bundled in executables
- ✅ **Embedded Frontend** - Next.js static export embedded in web server
- ✅ **Encrypted Credentials** - Production credentials encrypted and embedded
- ✅ **Health Monitoring** - Comprehensive health check endpoints
- ✅ **Windows Service Support** - Run as Windows Service for production
- ✅ **Zero-Downtime Updates** - Rolling deployment support

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Pre-Deployment Planning](#pre-deployment-planning)
3. [Build Process](#build-process)
4. [Environment Setup](#environment-setup)
5. [Credential Configuration](#credential-configuration)
6. [Application Deployment](#application-deployment)
7. [Windows Service Configuration](#windows-service-configuration)
8. [Network & Firewall Setup](#network--firewall-setup)
9. [Health Check Validation](#health-check-validation)
10. [Performance Tuning](#performance-tuning)
11. [Monitoring & Logging](#monitoring--logging)
12. [Backup & Recovery](#backup--recovery)
13. [Troubleshooting](#troubleshooting)
14. [Security Hardening](#security-hardening)

---

## Prerequisites

### System Requirements

#### Minimum Requirements
- **OS**: Windows Server 2019 or Windows 10 (64-bit)
- **CPU**: 2 cores, 2.4 GHz
- **RAM**: 4 GB
- **Storage**: 2 GB free space
- **Network**: Internet connectivity for Google Sheets API

#### Recommended Requirements
- **OS**: Windows Server 2022 (64-bit)
- **CPU**: 4 cores, 3.0 GHz
- **RAM**: 8 GB
- **Storage**: 10 GB free space (SSD preferred)
- **Network**: Dedicated network interface, 100 Mbps+

### Software Dependencies

#### Build Environment (Development Machine)
```powershell
# Required for building from source
Go 1.21+ (https://golang.org/dl/)
Node.js 18+ (https://nodejs.org/)
Git (https://git-scm.com/)
PowerShell 5.1+ (built into Windows)
```

#### Runtime Environment (Production Server)
```powershell
# Only required for deployment
Windows PowerShell 5.1+ (built-in)
.NET Framework 4.8+ (usually pre-installed)
Windows Defender or enterprise antivirus
```

### Network Prerequisites
- **Outbound HTTPS (443)**: Access to Google Sheets API (sheets.googleapis.com)
- **Inbound HTTP (8080)**: Web interface access (configurable)
- **DNS Resolution**: Internet DNS or internal DNS for Google APIs

---

## Pre-Deployment Planning

### Deployment Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    PRODUCTION SERVER                        │
│                                                             │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────┐ │
│  │   web.exe       │  │   scraper.exe   │  │ processor.exe│ │
│  │  (Port 8080)    │  │  (Scheduled)    │  │ (Scheduled) │ │
│  │  - Next.js UI   │  │  - ISX Download │  │ - CSV Process│ │
│  │  - REST API     │  │  - Data Fetch   │  │ - Index Gen  │ │
│  │  - WebSocket    │  │                 │  │             │ │
│  └─────────────────┘  └─────────────────┘  └─────────────┘ │
│                                                             │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │                 Data Directory                          │ │
│  │  ├── downloads/     (Excel files from ISX)             │ │
│  │  ├── reports/       (Generated CSV reports)            │ │
│  │  └── logs/          (Application logs)                 │ │
│  └─────────────────────────────────────────────────────────┘ │
│                                                             │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │              Configuration Files                        │ │
│  │  ├── license.dat           (Encrypted license data)    │ │
│  │  ├── encrypted_credentials.dat (Google API creds)      │ │
│  │  └── sheets-config.json    (Sheet ID mappings)         │ │
│  └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

### Deployment Strategies

#### 1. Single Server Deployment (Recommended)
- All components on one Windows Server
- Suitable for small to medium workloads
- Simplified management and monitoring
- Resource sharing between components

#### 2. Multi-Server Deployment (Enterprise)
- Web server on dedicated machine
- Scheduled jobs on separate processing server
- Load balancer for high availability
- Shared storage for data synchronization

### Resource Planning

#### Disk Space Requirements
```
release/
├── web.exe                 ~15 MB (includes embedded frontend)
├── scraper.exe            ~8 MB
├── processor.exe          ~8 MB
├── indexcsv.exe           ~6 MB
├── data/
│   ├── downloads/         ~100-500 MB (Excel files, rotating)
│   ├── reports/           ~50-200 MB (CSV files, growing)
│   └── logs/              ~10-50 MB (application logs, rotating)
└── config/                ~1 MB (configuration files)

Total Initial: ~50-100 MB
Total with Data: ~200-800 MB (depends on data retention)
```

#### Memory Usage
- **web.exe**: 50-150 MB (base + concurrent users)
- **scraper.exe**: 20-50 MB (during execution)
- **processor.exe**: 30-100 MB (during processing)
- **indexcsv.exe**: 15-30 MB (during execution)

#### CPU Usage
- **Normal Operations**: 5-15% CPU usage
- **Data Processing**: 20-60% CPU usage (during batch processing)
- **Peak Load**: Up to 80% CPU (multiple concurrent operations)

---

## Build Process

### Development Build (Source Code)

#### 1. Clone Repository
```powershell
# Clone the repository
git clone https://github.com/your-org/ISXDailyReportsScrapper.git
cd ISXDailyReportsScrapper

# Verify repository structure
Get-ChildItem -Path . -Name
```

#### 2. Build with PowerShell Script
```powershell
# Run the complete build process
.\build.ps1

# Output verification
if (Test-Path "release\web.exe") {
    Write-Host "✅ Build successful - web.exe created" -ForegroundColor Green
} else {
    Write-Host "❌ Build failed - check errors above" -ForegroundColor Red
}
```

#### 3. Build Components
The build process creates these executables:
- **web.exe**: Main web server with embedded Next.js frontend
- **scraper.exe**: ISX website scraper for downloading Excel reports
- **processor.exe**: Excel to CSV converter with forward-fill processing
- **indexcsv.exe**: ISX60/ISX15 index value extractor

### Production Build with Encrypted Credentials

#### 1. Setup Production Credentials
```powershell
# Create encrypted credentials for production
.\setup-production-credentials.ps1

# Verify encrypted credentials were created
if (Test-Path "encrypted_credentials.dat") {
    Write-Host "✅ Encrypted credentials created" -ForegroundColor Green
} else {
    Write-Host "❌ Credential encryption failed" -ForegroundColor Red
}
```

#### 2. Build with Embedded Credentials
```powershell
# Build with production credentials embedded
.\build.ps1

# Verify credentials were embedded in build
$webExeSize = (Get-Item "release\web.exe").Length
if ($webExeSize -gt 10MB) {
    Write-Host "✅ Production build with embedded credentials" -ForegroundColor Green
} else {
    Write-Host "⚠️ Build may not include credentials" -ForegroundColor Yellow
}
```

---

## Environment Setup

### Windows Server Configuration

#### 1. Server Hardening
```powershell
# Enable Windows Firewall
Set-NetFirewallProfile -Profile Domain,Public,Private -Enabled True

# Disable unnecessary services
Stop-Service -Name "Themes" -Force
Set-Service -Name "Themes" -StartupType Disabled

Stop-Service -Name "Windows Search" -Force  
Set-Service -Name "Windows Search" -StartupType Disabled

# Enable automatic updates
Set-ItemProperty -Path "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\WindowsUpdate\Auto Update" -Name "AUOptions" -Value 4
```

#### 2. Create Application User (Security Best Practice)
```powershell
# Create dedicated service account
$Password = ConvertTo-SecureString "ComplexPassword123!" -AsPlainText -Force
New-LocalUser -Name "ISXService" -Password $Password -Description "ISX Daily Reports Service Account"

# Assign minimal required permissions
Add-LocalGroupMember -Group "Log on as a service" -Member "ISXService"

# Create application directory with correct permissions
New-Item -Path "C:\ISXReports" -ItemType Directory -Force
$Acl = Get-Acl "C:\ISXReports"
$AccessRule = New-Object System.Security.AccessControl.FileSystemAccessRule("ISXService","FullControl","ContainerInherit,ObjectInherit","None","Allow")
$Acl.SetAccessRule($AccessRule)
Set-Acl -Path "C:\ISXReports" -AclObject $Acl
```

#### 3. Environment Variables
```powershell
# Set system-wide environment variables
[Environment]::SetEnvironmentVariable("ISX_ENVIRONMENT", "production", "Machine")
[Environment]::SetEnvironmentVariable("ISX_LOG_LEVEL", "info", "Machine")
[Environment]::SetEnvironmentVariable("ISX_PORT", "8080", "Machine")
[Environment]::SetEnvironmentVariable("ISX_DATA_DIR", "C:\ISXReports\data", "Machine")

# Restart system to apply environment variables
Restart-Computer -Force
```

### Directory Structure Setup

#### 1. Create Application Directories
```powershell
# Create complete directory structure
$BaseDir = "C:\ISXReports"
$Directories = @(
    "$BaseDir\bin",
    "$BaseDir\data\downloads",
    "$BaseDir\data\reports", 
    "$BaseDir\logs",
    "$BaseDir\config",
    "$BaseDir\backup"
)

foreach ($Dir in $Directories) {
    New-Item -Path $Dir -ItemType Directory -Force
    Write-Host "Created: $Dir" -ForegroundColor Green
}
```

#### 2. Set Permissions
```powershell
# Set restrictive permissions on application directory
$Acl = Get-Acl "$BaseDir"

# Remove inherited permissions
$Acl.SetAccessRuleProtection($true, $false)

# Add specific permissions
$AdminRule = New-Object System.Security.AccessControl.FileSystemAccessRule("Administrators","FullControl","ContainerInherit,ObjectInherit","None","Allow")
$ServiceRule = New-Object System.Security.AccessControl.FileSystemAccessRule("ISXService","Modify","ContainerInherit,ObjectInherit","None","Allow")

$Acl.SetAccessRule($AdminRule)
$Acl.SetAccessRule($ServiceRule)
Set-Acl -Path $BaseDir -AclObject $Acl
```

---

## Credential Configuration

### Google Sheets API Setup

#### 1. Service Account Creation
1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create new project or select existing project
3. Enable Google Sheets API
4. Create service account with minimal permissions:
   - Name: `isx-reports-production`
   - Role: None (permissions granted at sheet level)
5. Generate JSON key file
6. Share Google Sheets with service account email

#### 2. Credential Encryption
```powershell
# Place service account JSON in credentials.json
Copy-Item "path\to\downloaded\service-account.json" "credentials.json"

# Encrypt credentials for production
.\setup-production-credentials.ps1

# Verify encryption
if (Test-Path "encrypted_credentials.dat") {
    $Size = (Get-Item "encrypted_credentials.dat").Length
    Write-Host "✅ Credentials encrypted ($Size bytes)" -ForegroundColor Green
} else {
    Write-Host "❌ Credential encryption failed" -ForegroundColor Red
    exit 1
}

# Clean up plaintext credentials
Remove-Item "credentials.json" -Force
```

### License Configuration

#### 1. License Activation
```powershell
# Copy license file to application directory
Copy-Item "license.dat" "$BaseDir\config\license.dat"

# Test license activation
cd "$BaseDir\bin"
.\web.exe --test-license

# Verify license status
$Response = Invoke-RestMethod -Uri "http://localhost:8080/api/license/status"
if ($Response.is_valid) {
    Write-Host "✅ License activated successfully" -ForegroundColor Green
} else {
    Write-Host "❌ License activation failed: $($Response.message)" -ForegroundColor Red
}
```

#### 2. Sheet Configuration
```powershell
# Create sheets configuration
$SheetsConfig = @{
    "daily_reports" = @{
        "sheet_id" = "your-google-sheet-id-here"
        "range" = "Sheet1!A:Z"
    }
    "company_master" = @{
        "sheet_id" = "your-company-sheet-id-here"
        "range" = "Companies!A:D"
    }
} | ConvertTo-Json -Depth 3

$SheetsConfig | Out-File "$BaseDir\config\sheets-config.json" -Encoding UTF8
```

---

## Application Deployment

### File Deployment

#### 1. Copy Application Files
```powershell
# Copy built executables to production directory
$SourcePath = "release"
$DestPath = "C:\ISXReports\bin"

Copy-Item "$SourcePath\web.exe" "$DestPath\" -Force
Copy-Item "$SourcePath\scraper.exe" "$DestPath\" -Force
Copy-Item "$SourcePath\processor.exe" "$DestPath\" -Force
Copy-Item "$SourcePath\indexcsv.exe" "$DestPath\" -Force

# Verify file integrity
foreach ($Exe in @("web.exe", "scraper.exe", "processor.exe", "indexcsv.exe")) {
    $Size = (Get-Item "$DestPath\$Exe").Length
    Write-Host "✅ $Exe deployed ($([math]::Round($Size/1MB, 1)) MB)" -ForegroundColor Green
}
```

#### 2. Deploy Configuration Files
```powershell
# Copy configuration files
Copy-Item "encrypted_credentials.dat" "$BaseDir\config\" -Force
Copy-Item "sheets-config.json" "$BaseDir\config\" -Force
Copy-Item "license.dat" "$BaseDir\config\" -Force

# Set read-only permissions on sensitive files
Set-ItemProperty "$BaseDir\config\encrypted_credentials.dat" -Name IsReadOnly -Value $true
Set-ItemProperty "$BaseDir\config\license.dat" -Name IsReadOnly -Value $true
```

### Initial Testing

#### 1. Manual Application Test
```powershell
# Test web server startup
cd "C:\ISXReports\bin"
Start-Process -FilePath ".\web.exe" -ArgumentList "--port=8080" -NoNewWindow

# Wait for startup
Start-Sleep 10

# Test health endpoint
try {
    $HealthCheck = Invoke-RestMethod -Uri "http://localhost:8080/api/health" -TimeoutSec 30
    if ($HealthCheck.status -eq "ok") {
        Write-Host "✅ Application started successfully" -ForegroundColor Green
    } else {
        Write-Host "❌ Application health check failed" -ForegroundColor Red
    }
} catch {
    Write-Host "❌ Cannot connect to application: $($_.Exception.Message)" -ForegroundColor Red
}

# Stop test instance
Get-Process -Name "web" | Stop-Process -Force
```

#### 2. Component Testing
```powershell
# Test scraper component
cd "C:\ISXReports\bin"
.\scraper.exe --test-run --output="C:\ISXReports\data\downloads"

# Test processor component  
.\processor.exe --input="C:\ISXReports\data\downloads" --output="C:\ISXReports\data\reports" --test-run

# Test index extractor
.\indexcsv.exe --input="C:\ISXReports\data\reports" --output="C:\ISXReports\data\reports\indexes.csv" --test-run
```

---

## Windows Service Configuration

### Service Installation

#### 1. Create Service Wrapper Script
```powershell
# Create service wrapper script
$ServiceScript = @'
$ProcessName = "web"
$ExePath = "C:\ISXReports\bin\web.exe"
$Arguments = "--port=8080 --data-dir=C:\ISXReports\data --log-dir=C:\ISXReports\logs"

# Check if process is already running
$ExistingProcess = Get-Process -Name $ProcessName -ErrorAction SilentlyContinue
if ($ExistingProcess) {
    Write-Host "Service already running (PID: $($ExistingProcess.Id))"
    exit 0
}

# Start the application
try {
    Start-Process -FilePath $ExePath -ArgumentList $Arguments -WorkingDirectory "C:\ISXReports\bin" -WindowStyle Hidden
    Write-Host "ISX Reports Web Service started successfully"
} catch {
    Write-Host "Failed to start service: $($_.Exception.Message)"
    exit 1
}
'@

$ServiceScript | Out-File "C:\ISXReports\bin\start-service.ps1" -Encoding UTF8
```

#### 2. Install Windows Service
```powershell
# Install using NSSM (Non-Sucking Service Manager)
# Download NSSM from https://nssm.cc/download

# Install NSSM
$NSSMPath = "C:\Tools\nssm\win64\nssm.exe"

# Install service
& $NSSMPath install "ISXReportsWeb" "powershell.exe"
& $NSSMPath set "ISXReportsWeb" Arguments "-ExecutionPolicy Bypass -File C:\ISXReports\bin\start-service.ps1"
& $NSSMPath set "ISXReportsWeb" DisplayName "ISX Daily Reports Web Service"
& $NSSMPath set "ISXReportsWeb" Description "ISX Daily Reports Scrapper Web Interface and API"
& $NSSMPath set "ISXReportsWeb" Start SERVICE_AUTO_START
& $NSSMPath set "ISXReportsWeb" ObjectName "ISXService" "ComplexPassword123!"

# Configure service recovery
& $NSSMPath set "ISXReportsWeb" AppExit Default Restart
& $NSSMPath set "ISXReportsWeb" AppRestartDelay 30000
```

#### 3. Alternative: Native Windows Service
```powershell
# Create service directly (requires Administrator privileges)
New-Service -Name "ISXReportsWeb" `
    -BinaryPathName "C:\ISXReports\bin\web.exe --service --port=8080" `
    -DisplayName "ISX Daily Reports Web Service" `
    -Description "ISX Daily Reports Scrapper Web Interface and API" `
    -StartupType Automatic `
    -Credential (Get-Credential -UserName "ISXService")
```

### Service Management

#### 1. Service Control Commands
```powershell
# Start service
Start-Service -Name "ISXReportsWeb"

# Stop service
Stop-Service -Name "ISXReportsWeb" -Force

# Restart service
Restart-Service -Name "ISXReportsWeb"

# Check service status
Get-Service -Name "ISXReportsWeb"

# View service logs
Get-EventLog -LogName Application -Source "ISXReportsWeb" -Newest 10
```

#### 2. Service Monitoring Script
```powershell
# Create monitoring script
$MonitorScript = @'
param([int]$IntervalSeconds = 300)

while ($true) {
    try {
        # Check service status
        $Service = Get-Service -Name "ISXReportsWeb" -ErrorAction Stop
        
        if ($Service.Status -ne "Running") {
            Write-Host "$(Get-Date): Service not running, attempting restart..."
            Start-Service -Name "ISXReportsWeb"
            Start-Sleep 30
        }
        
        # Check application health
        $Health = Invoke-RestMethod -Uri "http://localhost:8080/api/health" -TimeoutSec 10
        if ($Health.status -eq "ok") {
            Write-Host "$(Get-Date): Service healthy"
        } else {
            Write-Host "$(Get-Date): Health check failed: $($Health.status)"
        }
        
    } catch {
        Write-Host "$(Get-Date): Monitor error: $($_.Exception.Message)"
        
        # Attempt service restart on error
        try {
            Restart-Service -Name "ISXReportsWeb"
            Write-Host "$(Get-Date): Service restarted"
        } catch {
            Write-Host "$(Get-Date): Failed to restart service: $($_.Exception.Message)"
        }
    }
    
    Start-Sleep $IntervalSeconds
}
'@

$MonitorScript | Out-File "C:\ISXReports\bin\monitor-service.ps1" -Encoding UTF8
```

---

## Network & Firewall Setup

### Firewall Configuration

#### 1. Windows Firewall Rules
```powershell
# Allow inbound HTTP traffic on port 8080
New-NetFirewallRule -DisplayName "ISX Reports Web" `
    -Direction Inbound `
    -Protocol TCP `
    -LocalPort 8080 `
    -Action Allow `
    -Description "ISX Daily Reports Web Interface"

# Allow outbound HTTPS to Google APIs
New-NetFirewallRule -DisplayName "ISX Reports Google APIs" `
    -Direction Outbound `
    -Protocol TCP `
    -RemotePort 443 `
    -RemoteAddress sheets.googleapis.com `
    -Action Allow `
    -Description "Google Sheets API Access"

# Block all other outbound traffic from application (optional security)
New-NetFirewallRule -DisplayName "ISX Reports Block Other" `
    -Direction Outbound `
    -Program "C:\ISXReports\bin\web.exe" `
    -Action Block `
    -Description "Block unintended outbound traffic"
```

#### 2. Verify Firewall Rules
```powershell
# List ISX-related firewall rules
Get-NetFirewallRule | Where-Object DisplayName -like "*ISX*" | Select-Object DisplayName, Direction, Action

# Test port accessibility
Test-NetConnection -ComputerName localhost -Port 8080
```

### Network Validation

#### 1. Internal Network Test
```powershell
# Test local accessibility
$Response = Invoke-WebRequest -Uri "http://localhost:8080/api/health" -UseBasicParsing
if ($Response.StatusCode -eq 200) {
    Write-Host "✅ Local access working" -ForegroundColor Green
} else {
    Write-Host "❌ Local access failed" -ForegroundColor Red
}
```

#### 2. External Network Test
```powershell
# Test from another machine on network
# Replace YOUR-SERVER-IP with actual server IP
$ServerIP = "192.168.1.100"
$Response = Invoke-WebRequest -Uri "http://$ServerIP:8080/api/health" -UseBasicParsing -TimeoutSec 10
if ($Response.StatusCode -eq 200) {
    Write-Host "✅ External access working" -ForegroundColor Green
} else {
    Write-Host "❌ External access failed" -ForegroundColor Red
}
```

---

## Health Check Validation

### Health Endpoint Testing

#### 1. Basic Health Check
```powershell
function Test-ISXHealth {
    param([string]$BaseUrl = "http://localhost:8080")
    
    $Endpoints = @(
        @{ Path = "/api/health"; Name = "Basic Health" },
        @{ Path = "/api/health/ready"; Name = "Readiness" },
        @{ Path = "/api/health/live"; Name = "Liveness" },
        @{ Path = "/api/version"; Name = "Version Info" },
        @{ Path = "/api/license/status"; Name = "License Status" }
    )
    
    foreach ($Endpoint in $Endpoints) {
        try {
            $Response = Invoke-RestMethod -Uri "$BaseUrl$($Endpoint.Path)" -TimeoutSec 10
            Write-Host "✅ $($Endpoint.Name): OK" -ForegroundColor Green
            
            if ($Endpoint.Path -eq "/api/health") {
                Write-Host "   Status: $($Response.status)" -ForegroundColor Cyan
                Write-Host "   Version: $($Response.version)" -ForegroundColor Cyan
            }
        } catch {
            Write-Host "❌ $($Endpoint.Name): FAILED - $($_.Exception.Message)" -ForegroundColor Red
        }
    }
}

# Run health checks
Test-ISXHealth
```

#### 2. Comprehensive Health Validation
```powershell
function Test-ISXComprehensive {
    param([string]$BaseUrl = "http://localhost:8080")
    
    Write-Host "=== ISX Daily Reports Health Check ===" -ForegroundColor Cyan
    Write-Host "Server: $BaseUrl"
    Write-Host "Time: $(Get-Date)"
    Write-Host ""
    
    # Test basic connectivity
    try {
        $Health = Invoke-RestMethod -Uri "$BaseUrl/api/health" -TimeoutSec 30
        Write-Host "✅ Application Running" -ForegroundColor Green
        Write-Host "   Status: $($Health.status)" -ForegroundColor Cyan
        Write-Host "   Version: $($Health.version)" -ForegroundColor Cyan
        Write-Host "   Timestamp: $($Health.timestamp)" -ForegroundColor Cyan
    } catch {
        Write-Host "❌ Application Not Responding" -ForegroundColor Red
        return $false
    }
    
    # Test readiness
    try {
        $Ready = Invoke-RestMethod -Uri "$BaseUrl/api/health/ready" -TimeoutSec 10
        Write-Host "✅ Application Ready" -ForegroundColor Green
        
        # Check individual services
        foreach ($ServiceName in $Ready.services.PSObject.Properties.Name) {
            $ServiceStatus = $Ready.services.$ServiceName
            if ($ServiceStatus.status -eq "ready") {
                Write-Host "   $ServiceName: Ready" -ForegroundColor Green
            } else {
                Write-Host "   $ServiceName: $($ServiceStatus.status) - $($ServiceStatus.message)" -ForegroundColor Yellow
            }
        }
    } catch {
        Write-Host "❌ Readiness Check Failed" -ForegroundColor Red
    }
    
    # Test license
    try {
        $License = Invoke-RestMethod -Uri "$BaseUrl/api/license/status" -TimeoutSec 10
        if ($License.is_valid) {
            Write-Host "✅ License Valid" -ForegroundColor Green
            Write-Host "   Days Remaining: $($License.days_left)" -ForegroundColor Cyan
            Write-Host "   Expiry: $($License.expiry_date)" -ForegroundColor Cyan
        } else {
            Write-Host "❌ License Invalid: $($License.message)" -ForegroundColor Red
        }
    } catch {
        Write-Host "❌ License Check Failed" -ForegroundColor Red
    }
    
    # Test WebSocket connectivity
    try {
        # Note: This is a simplified test - full WebSocket testing requires more complex logic
        $WebSocketTest = Test-NetConnection -ComputerName "localhost" -Port 8080
        if ($WebSocketTest.TcpTestSucceeded) {
            Write-Host "✅ WebSocket Port Accessible" -ForegroundColor Green
        } else {
            Write-Host "❌ WebSocket Port Not Accessible" -ForegroundColor Red
        }
    } catch {
        Write-Host "❌ WebSocket Test Failed" -ForegroundColor Red
    }
    
    Write-Host ""
    Write-Host "=== Health Check Complete ===" -ForegroundColor Cyan
    return $true
}

# Run comprehensive health check
Test-ISXComprehensive
```

### Automated Health Monitoring

#### 1. Create Health Check Task
```powershell
# Create scheduled task for health monitoring
$Action = New-ScheduledTaskAction -Execute "powershell.exe" -Argument "-ExecutionPolicy Bypass -File C:\ISXReports\bin\health-check.ps1"
$Trigger = New-ScheduledTaskTrigger -RepetitionInterval (New-TimeSpan -Minutes 5) -RepetitionDuration (New-TimeSpan -Days 365) -At (Get-Date)
$Settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable

Register-ScheduledTask -TaskName "ISXHealthCheck" -Action $Action -Trigger $Trigger -Settings $Settings -User "ISXService" -Description "ISX Reports Health Monitoring"
```

#### 2. Health Check Script
```powershell
# Create health check script
$HealthCheckScript = @'
# ISX Reports Health Check Script
$LogFile = "C:\ISXReports\logs\health-check.log"
$BaseUrl = "http://localhost:8080"

function Write-Log {
    param([string]$Message, [string]$Level = "INFO")
    $Timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $LogEntry = "[$Timestamp] [$Level] $Message"
    Write-Host $LogEntry
    Add-Content -Path $LogFile -Value $LogEntry
}

try {
    # Test basic health
    $Health = Invoke-RestMethod -Uri "$BaseUrl/api/health" -TimeoutSec 30
    if ($Health.status -eq "ok") {
        Write-Log "Health check passed - Application running normally"
    } else {
        Write-Log "Health check warning - Status: $($Health.status)" "WARN"
    }
    
    # Test license validity
    $License = Invoke-RestMethod -Uri "$BaseUrl/api/license/status" -TimeoutSec 10
    if (-not $License.is_valid) {
        Write-Log "License validation failed: $($License.message)" "ERROR"
        
        # Send alert (implement your alerting mechanism)
        # Send-MailMessage or write to event log
    } elseif ($License.days_left -lt 30) {
        Write-Log "License expires in $($License.days_left) days" "WARN"
    }
    
} catch {
    Write-Log "Health check failed: $($_.Exception.Message)" "ERROR"
    
    # Attempt service restart
    try {
        Restart-Service -Name "ISXReportsWeb"
        Write-Log "Service restarted due to health check failure" "INFO"
    } catch {
        Write-Log "Failed to restart service: $($_.Exception.Message)" "ERROR"
    }
}
'@

$HealthCheckScript | Out-File "C:\ISXReports\bin\health-check.ps1" -Encoding UTF8
```

---

## Performance Tuning

### System Optimization

#### 1. Windows Performance Settings
```powershell
# Optimize system for server workload
# Set High Performance power plan
powercfg /setactive 8c5e7fda-e8bf-4a96-9a85-a6e23a8c635c

# Optimize processor scheduling for background services
Set-ItemProperty -Path "HKLM:\SYSTEM\CurrentControlSet\Control\PriorityControl" -Name "Win32PrioritySeparation" -Value 24

# Increase system cache
Set-ItemProperty -Path "HKLM:\SYSTEM\CurrentControlSet\Control\Session Manager\Memory Management" -Name "LargeSystemCache" -Value 1

# Optimize network settings
netsh int tcp set global autotuninglevel=normal
netsh int tcp set global chimney=enabled
netsh int tcp set global rss=enabled
```

#### 2. Application Performance Tuning
```powershell
# Set environment variables for Go runtime optimization
[Environment]::SetEnvironmentVariable("GOGC", "100", "Machine")  # GC target percentage
[Environment]::SetEnvironmentVariable("GOMAXPROCS", "4", "Machine")  # Max CPU cores to use
[Environment]::SetEnvironmentVariable("GOMEMLIMIT", "2GiB", "Machine")  # Memory limit

# Set ISX-specific performance settings
[Environment]::SetEnvironmentVariable("ISX_MAX_CONCURRENT_OPERATIONS", "4", "Machine")
[Environment]::SetEnvironmentVariable("ISX_WEBSOCKET_BUFFER_SIZE", "8192", "Machine")
[Environment]::SetEnvironmentVariable("ISX_HTTP_TIMEOUT", "30s", "Machine")
```

### Resource Monitoring

#### 1. Performance Counter Monitoring
```powershell
# Create performance monitoring script
$PerfMonScript = @'
# Performance monitoring for ISX Reports
$LogFile = "C:\ISXReports\logs\performance.log"
$ProcessName = "web"

function Get-PerformanceMetrics {
    try {
        $Process = Get-Process -Name $ProcessName -ErrorAction Stop
        
        $Metrics = @{
            Timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
            ProcessId = $Process.Id
            CPUUsage = (Get-Counter "\Process($ProcessName)\% Processor Time" -SampleInterval 1 -MaxSamples 2 | Select-Object -ExpandProperty CounterSamples | Select-Object -Last 1).CookedValue
            MemoryMB = [math]::Round($Process.WorkingSet / 1MB, 2)
            HandleCount = $Process.HandleCount
            ThreadCount = $Process.Threads.Count
        }
        
        # System metrics
        $SystemMetrics = @{
            SystemCPU = (Get-Counter "\Processor(_Total)\% Processor Time" -SampleInterval 1 -MaxSamples 1).CounterSamples.CookedValue
            AvailableMemoryMB = [math]::Round((Get-Counter "\Memory\Available MBytes").CounterSamples.CookedValue, 2)
            DiskUsagePercent = (Get-Counter "\LogicalDisk(C:)\% Disk Time" -SampleInterval 1 -MaxSamples 1).CounterSamples.CookedValue
        }
        
        $AllMetrics = $Metrics + $SystemMetrics
        
        # Log metrics
        $LogEntry = ($AllMetrics.GetEnumerator() | ForEach-Object { "$($_.Key)=$($_.Value)" }) -join ", "
        Add-Content -Path $LogFile -Value $LogEntry
        
        # Check for performance issues
        if ($Metrics.CPUUsage -gt 80) {
            Write-EventLog -LogName Application -Source "ISXReports" -EventId 1001 -EntryType Warning -Message "High CPU usage: $($Metrics.CPUUsage)%"
        }
        
        if ($Metrics.MemoryMB -gt 500) {
            Write-EventLog -LogName Application -Source "ISXReports" -EventId 1002 -EntryType Warning -Message "High memory usage: $($Metrics.MemoryMB) MB"
        }
        
    } catch {
        Add-Content -Path $LogFile -Value "$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss') ERROR: $($_.Exception.Message)"
    }
}

# Run performance monitoring
Get-PerformanceMetrics
'@

$PerfMonScript | Out-File "C:\ISXReports\bin\performance-monitor.ps1" -Encoding UTF8

# Schedule performance monitoring
$Action = New-ScheduledTaskAction -Execute "powershell.exe" -Argument "-ExecutionPolicy Bypass -File C:\ISXReports\bin\performance-monitor.ps1"
$Trigger = New-ScheduledTaskTrigger -RepetitionInterval (New-TimeSpan -Minutes 1) -RepetitionDuration (New-TimeSpan -Days 365) -At (Get-Date)
Register-ScheduledTask -TaskName "ISXPerformanceMonitor" -Action $Action -Trigger $Trigger -User "System"
```

---

## Monitoring & Logging

### Application Logging

#### 1. Log Configuration
```powershell
# Configure log rotation and retention
$LogConfig = @{
    "log_level" = "info"
    "log_format" = "json"
    "log_file" = "C:\ISXReports\logs\application.log"
    "max_file_size" = "100MB"
    "max_files" = 10
    "compress_old_files" = $true
}

$LogConfig | ConvertTo-Json | Out-File "C:\ISXReports\config\logging.json" -Encoding UTF8
```

#### 2. Log Monitoring Script
```powershell
# Create log monitoring and alerting script
$LogMonitorScript = @'
# Log monitoring script for ISX Reports
param([int]$TailLines = 100)

$LogFile = "C:\ISXReports\logs\application.log"
$AlertKeywords = @("ERROR", "FATAL", "PANIC", "license.*fail", "authentication.*fail")

function Monitor-Logs {
    if (-not (Test-Path $LogFile)) {
        Write-Host "Log file not found: $LogFile"
        return
    }
    
    $LastLines = Get-Content $LogFile -Tail $TailLines
    
    foreach ($Line in $LastLines) {
        foreach ($Keyword in $AlertKeywords) {
            if ($Line -match $Keyword) {
                Write-Host "ALERT: $Line" -ForegroundColor Red
                
                # Write to Windows Event Log
                try {
                    Write-EventLog -LogName Application -Source "ISXReports" -EventId 2001 -EntryType Error -Message "Critical log entry detected: $Line"
                } catch {
                    # Create event source if it doesn't exist
                    New-EventLog -LogName Application -Source "ISXReports"
                    Write-EventLog -LogName Application -Source "ISXReports" -EventId 2001 -EntryType Error -Message "Critical log entry detected: $Line"
                }
                
                break
            }
        }
    }
}

# Monitor logs continuously
while ($true) {
    Monitor-Logs
    Start-Sleep 60  # Check every minute
}
'@

$LogMonitorScript | Out-File "C:\ISXReports\bin\log-monitor.ps1" -Encoding UTF8
```

### System Integration

#### 1. Windows Event Log Integration
```powershell
# Create custom event log for ISX Reports
try {
    New-EventLog -LogName "ISXReports" -Source "ISXReportsApp"
    Write-Host "✅ Custom event log created" -ForegroundColor Green
} catch {
    Write-Host "ℹ️ Event log may already exist" -ForegroundColor Yellow
}

# Test event logging
Write-EventLog -LogName "ISXReports" -Source "ISXReportsApp" -EventId 1000 -EntryType Information -Message "ISX Reports deployment completed successfully"
```

#### 2. Performance Counter Integration
```powershell
# Create custom performance counters (requires admin privileges)
$CounterCategoryName = "ISX Reports"
$Counters = @(
    @{ Name = "Active WebSocket Connections"; Type = "NumberOfItems32" },
    @{ Name = "License Validations per Second"; Type = "RateOfCountsPerSecond32" },
    @{ Name = "API Requests per Second"; Type = "RateOfCountsPerSecond32" },
    @{ Name = "Processing Operations"; Type = "NumberOfItems32" }
)

try {
    if ([System.Diagnostics.PerformanceCounterCategory]::Exists($CounterCategoryName)) {
        [System.Diagnostics.PerformanceCounterCategory]::Delete($CounterCategoryName)
    }
    
    $CounterCreationDataCollection = New-Object System.Diagnostics.CounterCreationDataCollection
    
    foreach ($Counter in $Counters) {
        $CounterCreationData = New-Object System.Diagnostics.CounterCreationData
        $CounterCreationData.CounterName = $Counter.Name
        $CounterCreationData.CounterType = $Counter.Type
        $CounterCreationDataCollection.Add($CounterCreationData)
    }
    
    [System.Diagnostics.PerformanceCounterCategory]::Create($CounterCategoryName, "ISX Reports Performance Counters", [System.Diagnostics.PerformanceCounterCategoryType]::SingleInstance, $CounterCreationDataCollection)
    
    Write-Host "✅ Performance counters created" -ForegroundColor Green
} catch {
    Write-Host "❌ Failed to create performance counters: $($_.Exception.Message)" -ForegroundColor Red
}
```

---

## Backup & Recovery

### Backup Strategy

#### 1. Data Backup Script
```powershell
# Comprehensive backup script
$BackupScript = @'
# ISX Reports Backup Script
param(
    [string]$BackupPath = "C:\ISXReports\backup",
    [switch]$FullBackup = $false
)

$SourcePath = "C:\ISXReports"
$Date = Get-Date -Format "yyyyMMdd-HHmm"
$BackupName = if ($FullBackup) { "ISXReports-Full-$Date" } else { "ISXReports-Data-$Date" }
$BackupDestination = Join-Path $BackupPath $BackupName

# Create backup directory
New-Item -Path $BackupDestination -ItemType Directory -Force

function Backup-Files {
    param([string]$Source, [string]$Destination, [string[]]$Include, [string[]]$Exclude)
    
    Write-Host "Backing up: $Source -> $Destination"
    
    $RobocopyArgs = @(
        $Source,
        $Destination,
        "/MIR",  # Mirror directory
        "/R:3",  # Retry 3 times
        "/W:5",  # Wait 5 seconds between retries
        "/LOG+:$BackupPath\backup.log"
    )
    
    if ($Include) {
        $RobocopyArgs += $Include
    }
    
    if ($Exclude) {
        foreach ($ExcludeItem in $Exclude) {
            $RobocopyArgs += "/XD"
            $RobocopyArgs += $ExcludeItem
        }
    }
    
    $Result = Start-Process -FilePath "robocopy.exe" -ArgumentList $RobocopyArgs -Wait -PassThru
    
    if ($Result.ExitCode -le 1) {
        Write-Host "✅ Backup completed successfully" -ForegroundColor Green
    } else {
        Write-Host "❌ Backup failed with exit code: $($Result.ExitCode)" -ForegroundColor Red
    }
}

if ($FullBackup) {
    # Full system backup
    Write-Host "Starting full backup..."
    
    # Stop service during full backup
    $ServiceRunning = (Get-Service -Name "ISXReportsWeb" -ErrorAction SilentlyContinue)?.Status -eq "Running"
    if ($ServiceRunning) {
        Stop-Service -Name "ISXReportsWeb" -Force
        Write-Host "Service stopped for backup"
    }
    
    Backup-Files -Source $SourcePath -Destination $BackupDestination -Exclude @("backup", "logs\*.log")
    
    # Restart service
    if ($ServiceRunning) {
        Start-Service -Name "ISXReportsWeb"
        Write-Host "Service restarted"
    }
} else {
    # Data-only backup (can run while service is running)
    Write-Host "Starting data backup..."
    
    # Backup critical data
    $DataFolders = @("data", "config", "logs")
    foreach ($Folder in $DataFolders) {
        $FolderPath = Join-Path $SourcePath $Folder
        if (Test-Path $FolderPath) {
            $FolderDestination = Join-Path $BackupDestination $Folder
            Backup-Files -Source $FolderPath -Destination $FolderDestination
        }
    }
}

# Compress backup
Write-Host "Compressing backup..."
Compress-Archive -Path $BackupDestination -DestinationPath "$BackupDestination.zip" -CompressionLevel Optimal
Remove-Item -Path $BackupDestination -Recurse -Force

# Cleanup old backups (keep last 7 days)
$OldBackups = Get-ChildItem -Path $BackupPath -Filter "ISXReports-*.zip" | Where-Object CreationTime -lt (Get-Date).AddDays(-7)
foreach ($OldBackup in $OldBackups) {
    Remove-Item -Path $OldBackup.FullName -Force
    Write-Host "Deleted old backup: $($OldBackup.Name)"
}

Write-Host "Backup completed: $BackupDestination.zip"
'@

$BackupScript | Out-File "C:\ISXReports\bin\backup.ps1" -Encoding UTF8
```

#### 2. Scheduled Backup Tasks
```powershell
# Daily data backup
$DailyBackupAction = New-ScheduledTaskAction -Execute "powershell.exe" -Argument "-ExecutionPolicy Bypass -File C:\ISXReports\bin\backup.ps1"
$DailyBackupTrigger = New-ScheduledTaskTrigger -Daily -At "02:00"
Register-ScheduledTask -TaskName "ISXReportsBackupDaily" -Action $DailyBackupAction -Trigger $DailyBackupTrigger -User "ISXService" -Description "Daily ISX Reports Data Backup"

# Weekly full backup
$WeeklyBackupAction = New-ScheduledTaskAction -Execute "powershell.exe" -Argument "-ExecutionPolicy Bypass -File C:\ISXReports\bin\backup.ps1 -FullBackup"
$WeeklyBackupTrigger = New-ScheduledTaskTrigger -Weekly -WeeksInterval 1 -DaysOfWeek Sunday -At "01:00"
Register-ScheduledTask -TaskName "ISXReportsBackupWeekly" -Action $WeeklyBackupAction -Trigger $WeeklyBackupTrigger -User "ISXService" -Description "Weekly ISX Reports Full Backup"
```

### Recovery Procedures

#### 1. Data Recovery Script
```powershell
# Recovery script
$RecoveryScript = @'
# ISX Reports Recovery Script
param(
    [Parameter(Mandatory)]
    [string]$BackupFile,
    [string]$RecoveryPath = "C:\ISXReports",
    [switch]$Force = $false
)

if (-not (Test-Path $BackupFile)) {
    Write-Host "❌ Backup file not found: $BackupFile" -ForegroundColor Red
    exit 1
}

# Stop service before recovery
$ServiceRunning = (Get-Service -Name "ISXReportsWeb" -ErrorAction SilentlyContinue)?.Status -eq "Running"
if ($ServiceRunning) {
    Write-Host "Stopping ISX Reports service..."
    Stop-Service -Name "ISXReportsWeb" -Force
}

# Create recovery timestamp
$RecoveryTime = Get-Date -Format "yyyyMMdd-HHmm"

# Backup current installation if it exists
if ((Test-Path $RecoveryPath) -and -not $Force) {
    $CurrentBackup = "${RecoveryPath}-backup-$RecoveryTime"
    Write-Host "Creating backup of current installation: $CurrentBackup"
    Rename-Item -Path $RecoveryPath -NewName $CurrentBackup
}

# Extract backup
Write-Host "Extracting backup: $BackupFile"
try {
    Expand-Archive -Path $BackupFile -DestinationPath $RecoveryPath -Force
    Write-Host "✅ Backup extracted successfully" -ForegroundColor Green
} catch {
    Write-Host "❌ Failed to extract backup: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

# Restore file permissions
Write-Host "Restoring file permissions..."
$Acl = Get-Acl $RecoveryPath
$ServiceRule = New-Object System.Security.AccessControl.FileSystemAccessRule("ISXService","Modify","ContainerInherit,ObjectInherit","None","Allow")
$Acl.SetAccessRule($ServiceRule)
Set-Acl -Path $RecoveryPath -AclObject $Acl

# Start service
if ($ServiceRunning) {
    Write-Host "Starting ISX Reports service..."
    Start-Service -Name "ISXReportsWeb"
    
    # Wait for service to start and test
    Start-Sleep 30
    try {
        $Health = Invoke-RestMethod -Uri "http://localhost:8080/api/health" -TimeoutSec 30
        if ($Health.status -eq "ok") {
            Write-Host "✅ Recovery completed successfully - Service is healthy" -ForegroundColor Green
        } else {
            Write-Host "⚠️ Service started but health check failed" -ForegroundColor Yellow
        }
    } catch {
        Write-Host "⚠️ Service started but not responding to health checks" -ForegroundColor Yellow
    }
}

Write-Host "Recovery completed from: $BackupFile"
'@

$RecoveryScript | Out-File "C:\ISXReports\bin\recovery.ps1" -Encoding UTF8
```

#### 2. Disaster Recovery Plan
```powershell
# Create disaster recovery documentation
$DisasterRecoveryPlan = @'
# ISX Reports Disaster Recovery Plan

## Recovery Time Objective (RTO): 2 hours
## Recovery Point Objective (RPO): 24 hours (daily backup)

## Emergency Contacts
- IT Administrator: admin@company.com
- Application Owner: app-owner@company.com
- Google Cloud Support: (if Google APIs issues)

## Recovery Steps

### 1. Assess the Situation
- Determine scope of failure (hardware, software, data corruption)
- Check if backups are available and recent
- Identify minimum recovery requirements

### 2. Prepare Recovery Environment
```powershell
# Verify system requirements
Get-ComputerInfo | Select-Object WindowsProductName, WindowsVersion, TotalPhysicalMemory

# Check disk space
Get-PSDrive -PSProvider FileSystem | Select-Object Name, @{Name="Size(GB)";Expression={[math]::Round($_.Used/1GB+$_.Free/1GB,2)}}, @{Name="FreeSpace(GB)";Expression={[math]::Round($_.Free/1GB,2)}}
```

### 3. Restore from Backup
```powershell
# List available backups
Get-ChildItem "C:\ISXReports\backup\*.zip" | Sort-Object CreationTime -Descending | Select-Object Name, CreationTime, @{Name="Size(MB)";Expression={[math]::Round($_.Length/1MB,2)}}

# Restore latest backup
.\recovery.ps1 -BackupFile "C:\ISXReports\backup\ISXReports-Full-YYYYMMDD-HHMM.zip"
```

### 4. Verify Recovery
```powershell
# Test all components
.\health-check.ps1

# Verify data integrity
Get-ChildItem "C:\ISXReports\data" -Recurse | Measure-Object -Property Length -Sum
```

### 5. Resume Operations
- Restart scheduled tasks
- Notify users of service restoration
- Monitor system for 24 hours post-recovery

## Alternative Recovery Options

### Option 1: Partial Recovery (Data Only)
If only data is corrupted, restore data directories only:
```powershell
# Extract data from backup
Expand-Archive -Path $BackupFile -DestinationPath "C:\Temp\Recovery"
Copy-Item "C:\Temp\Recovery\data\*" "C:\ISXReports\data\" -Recurse -Force
```

### Option 2: New Server Deployment
If hardware failure, deploy to new server:
1. Follow deployment guide for new server setup
2. Restore data and configuration from backup
3. Update DNS/load balancer to point to new server

### Option 3: Cloud Backup Recovery
If local backups are unavailable:
1. Contact cloud backup provider
2. Download latest backup from cloud storage
3. Follow standard recovery procedures
'@

$DisasterRecoveryPlan | Out-File "C:\ISXReports\docs\disaster-recovery-plan.md" -Encoding UTF8
```

---

## Troubleshooting

### Common Issues and Solutions

#### 1. Service Won't Start
```powershell
# Troubleshooting script for service startup issues
$TroubleshootService = @'
# ISX Reports Service Troubleshooting

Write-Host "=== ISX Reports Service Troubleshooting ===" -ForegroundColor Cyan

# Check service status
$Service = Get-Service -Name "ISXReportsWeb" -ErrorAction SilentlyContinue
if ($Service) {
    Write-Host "Service Status: $($Service.Status)" -ForegroundColor $(if($Service.Status -eq "Running"){"Green"}else{"Red"})
} else {
    Write-Host "❌ Service not found" -ForegroundColor Red
}

# Check if port is in use
$PortTest = Test-NetConnection -ComputerName localhost -Port 8080
Write-Host "Port 8080 Status: $(if($PortTest.TcpTestSucceeded){"In Use"}else{"Available"})" -ForegroundColor $(if($PortTest.TcpTestSucceeded){"Red"}else{"Green"})

# Check process
$Process = Get-Process -Name "web" -ErrorAction SilentlyContinue
if ($Process) {
    Write-Host "Process Running: Yes (PID: $($Process.Id))" -ForegroundColor Green
    Write-Host "Memory Usage: $([math]::Round($Process.WorkingSet/1MB, 2)) MB" -ForegroundColor Cyan
} else {
    Write-Host "Process Running: No" -ForegroundColor Red
}

# Check executable
$ExePath = "C:\ISXReports\bin\web.exe"
if (Test-Path $ExePath) {
    $ExeInfo = Get-Item $ExePath
    Write-Host "Executable: Exists ($([math]::Round($ExeInfo.Length/1MB, 2)) MB)" -ForegroundColor Green
    Write-Host "Modified: $($ExeInfo.LastWriteTime)" -ForegroundColor Cyan
} else {
    Write-Host "❌ Executable not found: $ExePath" -ForegroundColor Red
}

# Check configuration files
$ConfigFiles = @(
    "C:\ISXReports\config\encrypted_credentials.dat",
    "C:\ISXReports\config\license.dat",
    "C:\ISXReports\config\sheets-config.json"
)

foreach ($ConfigFile in $ConfigFiles) {
    if (Test-Path $ConfigFile) {
        Write-Host "Config: $(Split-Path $ConfigFile -Leaf) - Exists" -ForegroundColor Green
    } else {
        Write-Host "❌ Config: $(Split-Path $ConfigFile -Leaf) - Missing" -ForegroundColor Red
    }
}

# Check Windows Event Log
Write-Host "`n=== Recent Event Log Entries ===" -ForegroundColor Cyan
try {
    Get-EventLog -LogName Application -Source "ISXReports*" -Newest 5 -ErrorAction Stop | Format-Table TimeGenerated, EntryType, Message -Wrap
} catch {
    Write-Host "No event log entries found for ISXReports" -ForegroundColor Yellow
}

# Check firewall
Write-Host "`n=== Firewall Rules ===" -ForegroundColor Cyan
Get-NetFirewallRule -DisplayName "*ISX*" | Select-Object DisplayName, Direction, Action, Enabled | Format-Table

# Manual startup test
Write-Host "`n=== Manual Startup Test ===" -ForegroundColor Cyan
Write-Host "Attempting manual startup..." -ForegroundColor Yellow

$StartupTest = Start-Process -FilePath $ExePath -ArgumentList "--port=8081", "--test-mode" -PassThru -WindowStyle Hidden
Start-Sleep 10

if ($StartupTest.HasExited) {
    Write-Host "❌ Process exited with code: $($StartupTest.ExitCode)" -ForegroundColor Red
} else {
    Write-Host "✅ Process started successfully" -ForegroundColor Green
    $StartupTest | Stop-Process -Force
}

Write-Host "`n=== Troubleshooting Complete ===" -ForegroundColor Cyan
'@

$TroubleshootService | Out-File "C:\ISXReports\bin\troubleshoot-service.ps1" -Encoding UTF8
```

#### 2. License Issues
```powershell
# License troubleshooting
$LicenseTroubleshoot = @'
# ISX Reports License Troubleshooting

Write-Host "=== License Troubleshooting ===" -ForegroundColor Cyan

# Check license file
$LicenseFile = "C:\ISXReports\config\license.dat"
if (Test-Path $LicenseFile) {
    $LicenseInfo = Get-Item $LicenseFile
    Write-Host "✅ License file exists" -ForegroundColor Green
    Write-Host "   Size: $($LicenseInfo.Length) bytes" -ForegroundColor Cyan
    Write-Host "   Modified: $($LicenseInfo.LastWriteTime)" -ForegroundColor Cyan
    
    # Check file permissions
    $Acl = Get-Acl $LicenseFile
    Write-Host "   Permissions: $($Acl.Owner)" -ForegroundColor Cyan
} else {
    Write-Host "❌ License file not found: $LicenseFile" -ForegroundColor Red
    Write-Host "   Solution: Copy valid license.dat to config directory" -ForegroundColor Yellow
}

# Test license validation
try {
    Write-Host "`nTesting license validation..." -ForegroundColor Yellow
    $Response = Invoke-RestMethod -Uri "http://localhost:8080/api/license/status" -TimeoutSec 10
    
    if ($Response.is_valid) {
        Write-Host "✅ License is valid" -ForegroundColor Green
        Write-Host "   Status: $($Response.status)" -ForegroundColor Cyan
        Write-Host "   Days remaining: $($Response.days_left)" -ForegroundColor Cyan
        Write-Host "   Expiry date: $($Response.expiry_date)" -ForegroundColor Cyan
    } else {
        Write-Host "❌ License is invalid" -ForegroundColor Red
        Write-Host "   Message: $($Response.message)" -ForegroundColor Yellow
    }
} catch {
    Write-Host "❌ Cannot test license - service may not be running" -ForegroundColor Red
    Write-Host "   Error: $($_.Exception.Message)" -ForegroundColor Yellow
}

# Hardware fingerprint check (if service is running)
try {
    Write-Host "`nChecking hardware fingerprint..." -ForegroundColor Yellow
    # This would typically require access to the license manager API
    # For now, just indicate where to check
    Write-Host "   Hardware fingerprint validation requires service access" -ForegroundColor Cyan
    Write-Host "   Check application logs for hardware fingerprint mismatches" -ForegroundColor Cyan
} catch {
    Write-Host "⚠️ Hardware fingerprint check not available" -ForegroundColor Yellow
}

Write-Host "`n=== License Troubleshooting Complete ===" -ForegroundColor Cyan
'@

$LicenseTroubleshoot | Out-File "C:\ISXReports\bin\troubleshoot-license.ps1" -Encoding UTF8
```

#### 3. Network Connectivity Issues
```powershell
# Network troubleshooting
$NetworkTroubleshoot = @'
# Network Connectivity Troubleshooting

Write-Host "=== Network Connectivity Troubleshooting ===" -ForegroundColor Cyan

# Test local connectivity
Write-Host "`n1. Testing local connectivity..." -ForegroundColor Yellow
$LocalTest = Test-NetConnection -ComputerName "localhost" -Port 8080
if ($LocalTest.TcpTestSucceeded) {
    Write-Host "✅ Local port 8080 is accessible" -ForegroundColor Green
} else {
    Write-Host "❌ Local port 8080 is not accessible" -ForegroundColor Red
}

# Test external connectivity (if server has external IP)
Write-Host "`n2. Testing external connectivity..." -ForegroundColor Yellow
$ExternalIP = (Get-NetIPAddress -AddressFamily IPv4 | Where-Object {$_.IPAddress -like "192.168.*" -or $_.IPAddress -like "10.*" -or ($_.IPAddress -like "172.*" -and $_.IPAddress -notlike "172.16.*")} | Select-Object -First 1).IPAddress
if ($ExternalIP) {
    $ExternalTest = Test-NetConnection -ComputerName $ExternalIP -Port 8080
    if ($ExternalTest.TcpTestSucceeded) {
        Write-Host "✅ External access working from IP: $ExternalIP" -ForegroundColor Green
    } else {
        Write-Host "❌ External access failed from IP: $ExternalIP" -ForegroundColor Red
    }
}

# Test Google APIs connectivity
Write-Host "`n3. Testing Google APIs connectivity..." -ForegroundColor Yellow
$GoogleTest = Test-NetConnection -ComputerName "sheets.googleapis.com" -Port 443
if ($GoogleTest.TcpTestSucceeded) {
    Write-Host "✅ Google Sheets API is reachable" -ForegroundColor Green
} else {
    Write-Host "❌ Cannot reach Google Sheets API" -ForegroundColor Red
    Write-Host "   Check firewall and internet connectivity" -ForegroundColor Yellow
}

# Check firewall rules
Write-Host "`n4. Checking firewall rules..." -ForegroundColor Yellow
$FirewallRules = Get-NetFirewallRule -DisplayName "*ISX*" -ErrorAction SilentlyContinue
if ($FirewallRules) {
    foreach ($Rule in $FirewallRules) {
        $Status = if ($Rule.Enabled -eq "True") { "Enabled" } else { "Disabled" }
        $Color = if ($Rule.Enabled -eq "True") { "Green" } else { "Red" }
        Write-Host "   $($Rule.DisplayName): $Status ($($Rule.Direction) $($Rule.Action))" -ForegroundColor $Color
    }
} else {
    Write-Host "⚠️ No ISX-specific firewall rules found" -ForegroundColor Yellow
}

# Test DNS resolution
Write-Host "`n5. Testing DNS resolution..." -ForegroundColor Yellow
try {
    $DNSTest = Resolve-DnsName -Name "sheets.googleapis.com" -ErrorAction Stop
    Write-Host "✅ DNS resolution working" -ForegroundColor Green
    Write-Host "   Resolved to: $($DNSTest[0].IPAddress)" -ForegroundColor Cyan
} catch {
    Write-Host "❌ DNS resolution failed" -ForegroundColor Red
    Write-Host "   Error: $($_.Exception.Message)" -ForegroundColor Yellow
}

Write-Host "`n=== Network Troubleshooting Complete ===" -ForegroundColor Cyan
'@

$NetworkTroubleshoot | Out-File "C:\ISXReports\bin\troubleshoot-network.ps1" -Encoding UTF8
```

### Log Analysis Tools

#### 1. Log Parser Script
```powershell
# Log analysis script
$LogAnalyzer = @'
# ISX Reports Log Analyzer
param(
    [string]$LogFile = "C:\ISXReports\logs\application.log",
    [int]$LastHours = 24,
    [string]$Filter = ""
)

if (-not (Test-Path $LogFile)) {
    Write-Host "❌ Log file not found: $LogFile" -ForegroundColor Red
    exit 1
}

$StartTime = (Get-Date).AddHours(-$LastHours)
Write-Host "=== Log Analysis for Last $LastHours Hours ===" -ForegroundColor Cyan
Write-Host "Log File: $LogFile"
Write-Host "Start Time: $StartTime"
if ($Filter) {
    Write-Host "Filter: $Filter"
}
Write-Host ""

# Read and parse logs
$LogEntries = Get-Content $LogFile | Where-Object { $_ -match "\d{4}-\d{2}-\d{2}" }

# Filter by time if possible
$RecentEntries = @()
foreach ($Entry in $LogEntries) {
    try {
        # Try to extract timestamp (assuming format: YYYY-MM-DD HH:MM:SS)
        if ($Entry -match "(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})") {
            $Timestamp = [DateTime]$Matches[1]
            if ($Timestamp -gt $StartTime) {
                if (-not $Filter -or $Entry -match $Filter) {
                    $RecentEntries += $Entry
                }
            }
        }
    } catch {
        # If timestamp parsing fails, include entry if no time filter
        if (-not $Filter -or $Entry -match $Filter) {
            $RecentEntries += $Entry
        }
    }
}

# Analyze log levels
$ErrorCount = ($RecentEntries | Where-Object { $_ -match "ERROR" }).Count
$WarnCount = ($RecentEntries | Where-Object { $_ -match "WARN" }).Count
$InfoCount = ($RecentEntries | Where-Object { $_ -match "INFO" }).Count

Write-Host "=== Log Level Summary ===" -ForegroundColor Cyan
Write-Host "ERROR: $ErrorCount" -ForegroundColor $(if($ErrorCount -gt 0){"Red"}else{"Green"})
Write-Host "WARN:  $WarnCount" -ForegroundColor $(if($WarnCount -gt 0){"Yellow"}else{"Green"})
Write-Host "INFO:  $InfoCount" -ForegroundColor Cyan
Write-Host ""

# Show recent errors
if ($ErrorCount -gt 0) {
    Write-Host "=== Recent Errors ===" -ForegroundColor Red
    $RecentEntries | Where-Object { $_ -match "ERROR" } | Select-Object -Last 10 | ForEach-Object {
        Write-Host $_ -ForegroundColor Red
    }
    Write-Host ""
}

# Show recent warnings
if ($WarnCount -gt 0) {
    Write-Host "=== Recent Warnings ===" -ForegroundColor Yellow
    $RecentEntries | Where-Object { $_ -match "WARN" } | Select-Object -Last 5 | ForEach-Object {
        Write-Host $_ -ForegroundColor Yellow
    }
    Write-Host ""
}

# Pattern analysis
Write-Host "=== Pattern Analysis ===" -ForegroundColor Cyan
$Patterns = @{
    "License" = ($RecentEntries | Where-Object { $_ -match "license" }).Count
    "Authentication" = ($RecentEntries | Where-Object { $_ -match "auth" }).Count
    "Database" = ($RecentEntries | Where-Object { $_ -match "database|sql" }).Count
    "Network" = ($RecentEntries | Where-Object { $_ -match "network|connection|timeout" }).Count
    "Performance" = ($RecentEntries | Where-Object { $_ -match "slow|performance|timeout" }).Count
}

foreach ($Pattern in $Patterns.GetEnumerator()) {
    if ($Pattern.Value -gt 0) {
        Write-Host "$($Pattern.Key): $($Pattern.Value) occurrences" -ForegroundColor Cyan
    }
}

Write-Host "`n=== Log Analysis Complete ===" -ForegroundColor Cyan
'@

$LogAnalyzer | Out-File "C:\ISXReports\bin\analyze-logs.ps1" -Encoding UTF8
```

---

## Security Hardening

### Windows Security Configuration

#### 1. Security Hardening Script
```powershell
# Security hardening for ISX Reports deployment
$SecurityHardening = @'
# ISX Reports Security Hardening Script
# Run with Administrator privileges

Write-Host "=== ISX Reports Security Hardening ===" -ForegroundColor Cyan

# 1. Disable unnecessary Windows features
Write-Host "`n1. Disabling unnecessary Windows features..." -ForegroundColor Yellow

$FeaturesToDisable = @(
    "SMB1Protocol",
    "WorkFolders-Client",
    "Microsoft-Windows-Subsystem-Linux"
)

foreach ($Feature in $FeaturesToDisable) {
    try {
        Disable-WindowsOptionalFeature -Online -FeatureName $Feature -NoRestart -ErrorAction SilentlyContinue
        Write-Host "   Disabled: $Feature" -ForegroundColor Green
    } catch {
        Write-Host "   Could not disable: $Feature" -ForegroundColor Yellow
    }
}

# 2. Configure Windows Defender
Write-Host "`n2. Configuring Windows Defender..." -ForegroundColor Yellow

# Exclude ISX Reports directories from real-time scanning for performance
Add-MpPreference -ExclusionPath "C:\ISXReports" -ErrorAction SilentlyContinue
Write-Host "   Added exclusion for C:\ISXReports" -ForegroundColor Green

# Enable real-time monitoring
Set-MpPreference -DisableRealtimeMonitoring $false
Write-Host "   Real-time monitoring enabled" -ForegroundColor Green

# 3. Configure User Account Control (UAC)
Write-Host "`n3. Configuring UAC..." -ForegroundColor Yellow
Set-ItemProperty -Path "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System" -Name "ConsentPromptBehaviorAdmin" -Value 2
Set-ItemProperty -Path "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System" -Name "ConsentPromptBehaviorUser" -Value 3
Write-Host "   UAC configured for admin approval mode" -ForegroundColor Green

# 4. Configure Windows Firewall with advanced rules
Write-Host "`n4. Configuring advanced firewall rules..." -ForegroundColor Yellow

# Remove any existing ISX rules first
Get-NetFirewallRule -DisplayName "*ISX*" | Remove-NetFirewallRule -ErrorAction SilentlyContinue

# Inbound rules - only allow necessary traffic
New-NetFirewallRule -DisplayName "ISX Reports Web (Inbound)" `
    -Direction Inbound -Protocol TCP -LocalPort 8080 `
    -Action Allow -Profile Domain,Private `
    -Description "ISX Reports Web Interface - Inbound HTTP"

# Outbound rules - restrict to necessary connections only
New-NetFirewallRule -DisplayName "ISX Reports Google APIs (Outbound)" `
    -Direction Outbound -Protocol TCP -RemotePort 443 `
    -RemoteAddress "142.250.0.0/15", "172.217.0.0/16", "216.58.192.0/19" `
    -Action Allow -Program "C:\ISXReports\bin\web.exe" `
    -Description "Google Sheets API Access"

New-NetFirewallRule -DisplayName "ISX Reports DNS (Outbound)" `
    -Direction Outbound -Protocol UDP -RemotePort 53 `
    -Action Allow -Program "C:\ISXReports\bin\web.exe" `
    -Description "DNS Resolution"

Write-Host "   Advanced firewall rules configured" -ForegroundColor Green

# 5. Secure file permissions
Write-Host "`n5. Securing file permissions..." -ForegroundColor Yellow

$Paths = @(
    "C:\ISXReports\config",
    "C:\ISXReports\bin",
    "C:\ISXReports\logs"
)

foreach ($Path in $Paths) {
    if (Test-Path $Path) {
        $Acl = Get-Acl $Path
        $Acl.SetAccessRuleProtection($true, $false)  # Remove inherited permissions
        
        # Add necessary permissions
        $AdminRule = New-Object System.Security.AccessControl.FileSystemAccessRule("Administrators", "FullControl", "ContainerInherit,ObjectInherit", "None", "Allow")
        $ServiceRule = New-Object System.Security.AccessControl.FileSystemAccessRule("ISXService", "ReadAndExecute", "ContainerInherit,ObjectInherit", "None", "Allow")
        
        if ($Path -like "*config*") {
            # Config directory - service needs read access only
            $ServiceRule = New-Object System.Security.AccessControl.FileSystemAccessRule("ISXService", "Read", "ContainerInherit,ObjectInherit", "None", "Allow")
        } elseif ($Path -like "*logs*") {
            # Logs directory - service needs write access
            $ServiceRule = New-Object System.Security.AccessControl.FileSystemAccessRule("ISXService", "Modify", "ContainerInherit,ObjectInherit", "None", "Allow")
        }
        
        $Acl.SetAccessRule($AdminRule)
        $Acl.SetAccessRule($ServiceRule)
        Set-Acl -Path $Path -AclObject $Acl
        
        Write-Host "   Secured permissions for: $Path" -ForegroundColor Green
    }
}

# 6. Configure audit logging
Write-Host "`n6. Configuring audit logging..." -ForegroundColor Yellow

# Enable object access auditing
auditpol /set /category:"Object Access" /success:enable /failure:enable
auditpol /set /category:"Logon/Logoff" /success:enable /failure:enable
auditpol /set /category:"Account Management" /success:enable /failure:enable

Write-Host "   Audit logging configured" -ForegroundColor Green

# 7. Disable unnecessary services
Write-Host "`n7. Disabling unnecessary services..." -ForegroundColor Yellow

$ServicesToDisable = @(
    "Fax",
    "WerSvc",  # Windows Error Reporting
    "Themes",  # Themes service
    "TabletInputService"
)

foreach ($ServiceName in $ServicesToDisable) {
    try {
        $Service = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
        if ($Service -and $Service.StartType -ne "Disabled") {
            Stop-Service -Name $ServiceName -Force -ErrorAction SilentlyContinue
            Set-Service -Name $ServiceName -StartupType Disabled
            Write-Host "   Disabled service: $ServiceName" -ForegroundColor Green
        }
    } catch {
        Write-Host "   Could not disable service: $ServiceName" -ForegroundColor Yellow
    }
}

# 8. Configure registry security settings
Write-Host "`n8. Configuring registry security settings..." -ForegroundColor Yellow

# Disable anonymous SID enumeration
Set-ItemProperty -Path "HKLM:\SYSTEM\CurrentControlSet\Control\Lsa" -Name "RestrictAnonymousSAM" -Value 1
Set-ItemProperty -Path "HKLM:\SYSTEM\CurrentControlSet\Control\Lsa" -Name "RestrictAnonymous" -Value 1

# Disable NTLM v1
Set-ItemProperty -Path "HKLM:\SYSTEM\CurrentControlSet\Control\Lsa" -Name "LmCompatibilityLevel" -Value 5

# Configure session security
Set-ItemProperty -Path "HKLM:\SYSTEM\CurrentControlSet\Control\Lsa\MSV1_0" -Name "NTLMMinClientSec" -Value 537395200
Set-ItemProperty -Path "HKLM:\SYSTEM\CurrentControlSet\Control\Lsa\MSV1_0" -Name "NTLMMinServerSec" -Value 537395200

Write-Host "   Registry security settings configured" -ForegroundColor Green

# 9. Configure network security
Write-Host "`n9. Configuring network security..." -ForegroundColor Yellow

# Disable NetBIOS over TCP/IP
$Adapters = Get-WmiObject -Class Win32_NetworkAdapterConfiguration | Where-Object {$_.IPEnabled -eq $true}
foreach ($Adapter in $Adapters) {
    $Adapter.SetTcpipNetbios(2)  # Disable NetBIOS over TCP/IP
}

# Configure SMB security
Set-SmbServerConfiguration -EnableSMB1Protocol $false -Force -Confirm:$false
Set-SmbServerConfiguration -RequireSecuritySignature $true -Force -Confirm:$false

Write-Host "   Network security configured" -ForegroundColor Green

# 10. Final verification
Write-Host "`n10. Performing security verification..." -ForegroundColor Yellow

# Check if ISX service account exists
$ISXService = Get-LocalUser -Name "ISXService" -ErrorAction SilentlyContinue
if ($ISXService) {
    Write-Host "   ✅ ISX service account exists" -ForegroundColor Green
} else {
    Write-Host "   ❌ ISX service account not found" -ForegroundColor Red
}

# Check firewall status
$FirewallProfiles = Get-NetFirewallProfile
$AllEnabled = ($FirewallProfiles | Where-Object { $_.Enabled -eq $false }).Count -eq 0
if ($AllEnabled) {
    Write-Host "   ✅ Windows Firewall enabled on all profiles" -ForegroundColor Green
} else {
    Write-Host "   ❌ Windows Firewall not enabled on all profiles" -ForegroundColor Red
}

# Check Windows Defender status
$DefenderStatus = Get-MpComputerStatus
if ($DefenderStatus.RealTimeProtectionEnabled) {
    Write-Host "   ✅ Windows Defender real-time protection enabled" -ForegroundColor Green
} else {
    Write-Host "   ❌ Windows Defender real-time protection disabled" -ForegroundColor Red
}

Write-Host "`n=== Security Hardening Complete ===" -ForegroundColor Cyan
Write-Host "⚠️  Restart required for all changes to take effect" -ForegroundColor Yellow
'@

$SecurityHardening | Out-File "C:\ISXReports\bin\security-hardening.ps1" -Encoding UTF8
```

### SSL/TLS Configuration (Optional)

#### 1. Self-Signed Certificate Creation
```powershell
# Create self-signed certificate for HTTPS
$CertScript = @'
# SSL Certificate Setup for ISX Reports
param([string]$CertificateName = "ISX-Reports-Local")

Write-Host "=== SSL Certificate Setup ===" -ForegroundColor Cyan

# Create self-signed certificate
$Cert = New-SelfSignedCertificate -DnsName "localhost", "127.0.0.1", $env:COMPUTERNAME `
    -CertStoreLocation "cert:\LocalMachine\My" `
    -FriendlyName $CertificateName `
    -NotAfter (Get-Date).AddYears(5) `
    -KeyUsage DigitalSignature, KeyEncipherment `
    -KeyAlgorithm RSA `
    -KeyLength 2048

Write-Host "✅ Certificate created with thumbprint: $($Cert.Thumbprint)" -ForegroundColor Green

# Export certificate for client trust (optional)
$CertPath = "C:\ISXReports\config\isx-reports-cert.cer"
Export-Certificate -Cert $Cert -FilePath $CertPath -Type CERT
Write-Host "✅ Certificate exported to: $CertPath" -ForegroundColor Green

# Configure HTTPS binding (requires administrative privileges)
try {
    # Remove existing binding if it exists
    netsh http delete sslcert ipport=0.0.0.0:8443 2>$null
    
    # Add new HTTPS binding
    $Command = "netsh http add sslcert ipport=0.0.0.0:8443 certhash=$($Cert.Thumbprint) appid={12345678-1234-1234-1234-123456789012}"
    Invoke-Expression $Command
    
    Write-Host "✅ HTTPS binding configured for port 8443" -ForegroundColor Green
    Write-Host "   Access URL: https://localhost:8443" -ForegroundColor Cyan
} catch {
    Write-Host "⚠️ Could not configure HTTPS binding: $($_.Exception.Message)" -ForegroundColor Yellow
    Write-Host "   Run as Administrator to configure HTTPS" -ForegroundColor Yellow
}

Write-Host "`n=== SSL Certificate Setup Complete ===" -ForegroundColor Cyan
'@

$CertScript | Out-File "C:\ISXReports\bin\setup-ssl.ps1" -Encoding UTF8
```

---

## Summary

This deployment guide provides comprehensive instructions for deploying the ISX Daily Reports Scrapper on Windows environments with enterprise-grade security and monitoring. The deployment includes:

### ✅ Completed Setup
- **Single Binary Deployment** with embedded Next.js frontend
- **Encrypted Credential Management** for production security
- **Windows Service Configuration** for reliable operation
- **Health Check Endpoints** for monitoring and alerts
- **Comprehensive Security Hardening** following best practices
- **Backup and Recovery Procedures** for business continuity
- **Performance Tuning** for optimal resource utilization
- **Troubleshooting Tools** for rapid issue resolution

### 🔧 Post-Deployment Tasks
1. **License Activation**: Activate production license key
2. **Google Sheets Integration**: Configure API credentials and sheet mappings
3. **Monitoring Setup**: Configure health check monitoring and alerting
4. **Security Audit**: Run security hardening script and verify compliance
5. **Backup Testing**: Test backup and recovery procedures
6. **Performance Baseline**: Establish performance metrics baseline
7. **User Training**: Train administrators on operations and troubleshooting

### 📞 Support Information
- **Documentation**: Located in `C:\ISXReports\docs\`
- **Logs**: Application logs in `C:\ISXReports\logs\`
- **Health Checks**: `http://localhost:8080/api/health`
- **Troubleshooting Scripts**: Available in `C:\ISXReports\bin\`

The deployment is now ready for production use with enterprise security controls, comprehensive monitoring, and automated recovery capabilities.

---

**Document Version**: 2.0  
**Last Updated**: 2025-01-31  
**Target Platform**: Windows Server 2019/2022, Windows 10/11  
**Deployment Status**: Production Ready