# ISX Daily Reports Scrapper - Production Credentials Setup (PowerShell)
# This script helps you integrate your existing Google service account credentials
# with the new encryption system

param(
    [switch]$Verbose = $false
)

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "ISX Production Credentials Setup" -ForegroundColor Cyan  
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Check if we're in the right directory
if (-not (Test-Path "dev\internal\security")) {
    Write-Host "[ERROR] This script must be run from the ISX project root directory" -ForegroundColor Red
    Write-Host "Current directory: $(Get-Location)" -ForegroundColor Red
    Read-Host "Press Enter to exit"
    exit 1
}

Write-Host "[INFO] Setting up production credentials for ISX Daily Reports Scrapper" -ForegroundColor Blue
Write-Host ""

# Step 1: Check for existing credentials
Write-Host "[STEP 1] Checking for existing credentials..." -ForegroundColor Yellow
Write-Host ""

$foundCredentials = $false
$credentialsFile = ""
$useEnvVar = $false

# Check for credentials.json in dev directory
if (Test-Path "dev\credentials.json") {
    Write-Host "[FOUND] dev\credentials.json" -ForegroundColor Green
    $credentialsFile = "dev\credentials.json"
    $foundCredentials = $true
}

# Check for credentials.json in root directory  
if (Test-Path "credentials.json") {
    Write-Host "[FOUND] credentials.json" -ForegroundColor Green
    $credentialsFile = "credentials.json"
    $foundCredentials = $true
}

# Check for ISX_CREDENTIALS environment variable
$envCredentials = $env:ISX_CREDENTIALS
if ($envCredentials) {
    Write-Host "[FOUND] ISX_CREDENTIALS environment variable" -ForegroundColor Green
    $preview = $envCredentials.Substring(0, [Math]::Min(50, $envCredentials.Length))
    Write-Host "[INFO] Environment variable contains $preview..." -ForegroundColor Blue
    $foundCredentials = $true
    $useEnvVar = $true
}

if (-not $foundCredentials) {
    Write-Host ""
    Write-Host "[ERROR] No Google service account credentials found!" -ForegroundColor Red
    Write-Host ""
    Write-Host "Please ensure you have your credentials in one of these locations:" -ForegroundColor Red
    Write-Host "  1. dev\credentials.json" -ForegroundColor White
    Write-Host "  2. credentials.json" -ForegroundColor White
    Write-Host "  3. ISX_CREDENTIALS environment variable" -ForegroundColor White
    Write-Host ""
    Write-Host "The credentials should be your Google service account JSON file that" -ForegroundColor Yellow
    Write-Host "currently works with your Google Sheet: 1l4jJNNqHZNomjp3wpkL-txDfCjsRr19aJZOZqPHJ6lc" -ForegroundColor Yellow
    Write-Host ""
    Read-Host "Press Enter to exit"
    exit 1
}

Write-Host ""
Write-Host "[STEP 2] Building credential encryption tool..." -ForegroundColor Yellow
Write-Host ""

Push-Location "dev"
try {
    $buildResult = go build -o "..\encrypt-credentials.exe" ".\internal\security\tools\encrypt-credentials.go"
    if ($LASTEXITCODE -ne 0) {
        throw "Build failed with exit code $LASTEXITCODE"
    }
    Write-Host "[SUCCESS] Encryption tool built successfully" -ForegroundColor Green
} catch {
    Write-Host "[ERROR] Failed to build encryption tool: $_" -ForegroundColor Red
    Pop-Location
    Read-Host "Press Enter to exit"
    exit 1
} finally {
    Pop-Location
}

Write-Host ""

# Step 3: Encrypt credentials
Write-Host "[STEP 3] Encrypting your production credentials..." -ForegroundColor Yellow
Write-Host ""

$tempFile = $null
try {
    if ($useEnvVar) {
        # Create temporary file from environment variable
        $tempFile = "temp_credentials.json"
        $envCredentials | Out-File -FilePath $tempFile -Encoding UTF8
        $credentialsFile = $tempFile
    }

    # Encrypt the credentials
    $encryptArgs = @("-input", $credentialsFile, "-output", "encrypted_credentials.dat")
    if ($Verbose) {
        $encryptArgs += "-verbose"
    }
    
    & ".\encrypt-credentials.exe" @encryptArgs
    
    if ($LASTEXITCODE -ne 0) {
        throw "Encryption failed with exit code $LASTEXITCODE"
    }
    
    Write-Host "[SUCCESS] Credentials encrypted successfully" -ForegroundColor Green
    
} catch {
    Write-Host "[ERROR] Failed to encrypt credentials: $_" -ForegroundColor Red
    if ($tempFile -and (Test-Path $tempFile)) {
        Remove-Item $tempFile -Force
    }
    Read-Host "Press Enter to exit"
    exit 1
} finally {
    # Clean up temporary file
    if ($tempFile -and (Test-Path $tempFile)) {
        Remove-Item $tempFile -Force
    }
}

Write-Host ""

# Step 4: Generate integration information
Write-Host "[STEP 4] Validating encrypted credentials..." -ForegroundColor Yellow
Write-Host ""

if (Test-Path "encrypted_credentials.dat") {
    $fileSize = (Get-Item "encrypted_credentials.dat").Length
    if ($fileSize -ge 100) {
        Write-Host "[SUCCESS] Encrypted credentials file created ($fileSize bytes)" -ForegroundColor Green
    } else {
        Write-Host "[WARNING] Encrypted credentials file seems too small ($fileSize bytes)" -ForegroundColor Yellow
    }
} else {
    Write-Host "[ERROR] Encrypted credentials file not found" -ForegroundColor Red
    Read-Host "Press Enter to exit"
    exit 1
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Production Credentials Setup Complete!" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

Write-Host "Next Steps:" -ForegroundColor Green
Write-Host "1. Run the production build: " -NoNewline -ForegroundColor White
Write-Host "build.ps1" -ForegroundColor Yellow
Write-Host "2. The build will automatically embed your encrypted credentials" -ForegroundColor White
Write-Host "3. Test the license activation with your Google Sheet" -ForegroundColor White
Write-Host ""

Write-Host "Configuration Summary:" -ForegroundColor Green
Write-Host "  Sheet ID: " -NoNewline -ForegroundColor White
Write-Host "1l4jJNNqHZNomjp3wpkL-txDfCjsRr19aJZOZqPHJ6lc" -ForegroundColor Yellow
Write-Host "  Sheet Name: " -NoNewline -ForegroundColor White
Write-Host "Licenses" -ForegroundColor Yellow
Write-Host ""

Write-Host "Security Features:" -ForegroundColor Green
Write-Host "  • AES-256-GCM encryption with OWASP compliance" -ForegroundColor White
Write-Host "  • Binary integrity verification" -ForegroundColor White
Write-Host "  • Anti-tampering detection" -ForegroundColor White
Write-Host "  • Memory protection with secure cleanup" -ForegroundColor White
Write-Host "  • Certificate pinning for Google APIs" -ForegroundColor White
Write-Host ""

Write-Host "[SECURITY] The original credentials file remains unchanged." -ForegroundColor Blue
Write-Host "[SECURITY] Only the encrypted version will be embedded in the application." -ForegroundColor Blue
Write-Host ""

Read-Host "Press Enter to continue"