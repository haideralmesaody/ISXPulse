[Setup]
; ISX Pulse Scratch Card License System Installer
AppName=ISX Pulse - Scratch Card Edition
AppVersion=0.0.1-alpha
AppPublisher=ISX Digital Solutions
AppPublisherURL=https://github.com/yourusername/ISXDailyReportsScrapper
AppSupportURL=https://github.com/yourusername/ISXDailyReportsScrapper/issues
AppUpdatesURL=https://github.com/yourusername/ISXDailyReportsScrapper/releases
DefaultDirName={autopf}\ISX Pulse Scratch Card
DefaultGroupName=ISX Pulse
AllowNoIcons=yes
LicenseFile=..\LICENSE
InfoBeforeFile=SCRATCH_CARD_INFO.txt
InfoAfterFile=SCRATCH_CARD_SETUP.txt
OutputDir=..\dist\installer
OutputBaseFilename=ISXPulse-ScratchCard-Setup-{#AppVersion}
SetupIconFile=assets\isx-app-icon.ico
Compression=lzma
SolidCompression=yes
WizardStyle=modern
DisableProgramGroupPage=yes
DisableReadyPage=no
PrivilegesRequired=admin
ArchitecturesAllowed=x64
ArchitecturesInstallIn64BitMode=x64

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Types]
Name: "full"; Description: "Full Installation (Recommended)"
Name: "server"; Description: "Server Only"
Name: "generator"; Description: "License Generator Only"
Name: "custom"; Description: "Custom Installation"; Flags: iscustom

[Components]
Name: "core"; Description: "ISX Pulse Server (Scratch Card)"; Types: full server custom; Flags: fixed
Name: "tools"; Description: "License Generator & Tools"; Types: full generator custom
Name: "docs"; Description: "Documentation & Guides"; Types: full custom
Name: "examples"; Description: "Configuration Examples"; Types: full custom
Name: "scripts"; Description: "Deployment Scripts"; Types: full custom

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: unchecked
Name: "quicklaunchicon"; Description: "{cm:CreateQuickLaunchIcon}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: unchecked; OnlyBelowVersion: 6.1
Name: "firewall"; Description: "Configure Windows Firewall"; GroupDescription: "Security"; Flags: unchecked
Name: "autostart"; Description: "Start server automatically"; GroupDescription: "Service"; Flags: unchecked

[Files]
; Core Application Files
Source: "..\dist\ISXPulse.exe"; DestDir: "{app}"; Flags: ignoreversion; Components: core
Source: "..\dist\scraper.exe"; DestDir: "{app}"; Flags: ignoreversion; Components: core
Source: "..\dist\processor.exe"; DestDir: "{app}"; Flags: ignoreversion; Components: core  
Source: "..\dist\indexcsv.exe"; DestDir: "{app}"; Flags: ignoreversion; Components: core

; License Generator
Source: "..\dist\license-generator.exe"; DestDir: "{app}"; Flags: ignoreversion; Components: tools
Source: "..\tools\license-generator\README.md"; DestDir: "{app}\docs"; Flags: ignoreversion; Components: tools docs

; Configuration Files
Source: "..\config\examples\*.example"; DestDir: "{app}\config"; Flags: ignoreversion; Components: examples
Source: "..\dist\.env.example"; DestDir: "{app}"; Flags: ignoreversion; Components: examples

; Documentation
Source: "..\README.md"; DestDir: "{app}\docs"; Flags: ignoreversion; Components: docs
Source: "..\SCRATCH_CARD_LICENSE_IMPLEMENTATION_PLAN.md"; DestDir: "{app}\docs"; Flags: ignoreversion; Components: docs
Source: "..\SECURITY.md"; DestDir: "{app}\docs"; Flags: ignoreversion; Components: docs
Source: "..\docs\DEPLOYMENT_GUIDE.md"; DestDir: "{app}\docs"; Flags: ignoreversion; Components: docs
Source: "..\docs\QUICK_START.md"; DestDir: "{app}\docs"; Flags: ignoreversion; Components: docs
Source: "..\docs\SECURITY.md"; DestDir: "{app}\docs"; Flags: ignoreversion; Components: docs

; Scripts
Source: "..\scripts\deploy-scratch-card.bat"; DestDir: "{app}\scripts"; Flags: ignoreversion; Components: scripts
Source: "..\scripts\setup-scratch-card-credentials.bat"; DestDir: "{app}\scripts"; Flags: ignoreversion; Components: scripts
Source: "..\scripts\package-scratch-card-release.bat"; DestDir: "{app}\scripts"; Flags: ignoreversion; Components: scripts
Source: "..\scripts\start-server.bat"; DestDir: "{app}\scripts"; Flags: ignoreversion; Components: scripts

; Directory Structure
Source: "..\dist\data\*"; DestDir: "{app}\data"; Flags: ignoreversion createallsubdirs recursesubdirs; Components: core
Source: "..\dist\logs\*"; DestDir: "{app}\logs"; Flags: ignoreversion createallsubdirs recursesubdirs; Components: core

; License and Legal
Source: "..\LICENSE"; DestDir: "{app}"; Flags: ignoreversion; Components: core

[Dirs]
Name: "{app}\data\downloads"; Permissions: users-full
Name: "{app}\data\reports"; Permissions: users-full
Name: "{app}\logs"; Permissions: users-full
Name: "{app}\config"; Permissions: users-modify
Name: "{app}\backup"; Permissions: users-full

[Icons]
Name: "{group}\ISX Pulse Server"; Filename: "{app}\ISXPulse.exe"; WorkingDir: "{app}"; IconFilename: "{app}\assets\isx-app-icon.ico"
Name: "{group}\License Generator"; Filename: "{app}\license-generator.exe"; WorkingDir: "{app}"; Parameters: "--help"; IconFilename: "{app}\assets\isx-app-icon.ico"; Components: tools
Name: "{group}\Setup Credentials"; Filename: "{app}\scripts\setup-scratch-card-credentials.bat"; WorkingDir: "{app}"; IconFilename: "{app}\assets\isx-app-icon.ico"; Components: scripts
Name: "{group}\Deploy System"; Filename: "{app}\scripts\deploy-scratch-card.bat"; WorkingDir: "{app}"; IconFilename: "{app}\assets\isx-app-icon.ico"; Components: scripts
Name: "{group}\Documentation"; Filename: "{app}\docs\README.md"; WorkingDir: "{app}\docs"; Components: docs
Name: "{group}\{cm:UninstallProgram,ISX Pulse}"; Filename: "{uninstallexe}"
Name: "{autodesktop}\ISX Pulse Scratch Card"; Filename: "{app}\ISXPulse.exe"; WorkingDir: "{app}"; IconFilename: "{app}\assets\isx-app-icon.ico"; Tasks: desktopicon
Name: "{userappdata}\Microsoft\Internet Explorer\Quick Launch\ISX Pulse"; Filename: "{app}\ISXPulse.exe"; WorkingDir: "{app}"; IconFilename: "{app}\assets\isx-app-icon.ico"; Tasks: quicklaunchicon

[Registry]
; Register file associations for ISX license files
Root: HKCR; Subkey: ".isxlic"; ValueType: string; ValueName: ""; ValueData: "ISXLicenseFile"; Flags: uninsdeletekey
Root: HKCR; Subkey: "ISXLicenseFile"; ValueType: string; ValueName: ""; ValueData: "ISX License File"; Flags: uninsdeletekey
Root: HKCR; Subkey: "ISXLicenseFile\DefaultIcon"; ValueType: string; ValueName: ""; ValueData: "{app}\assets\isx-app-icon.ico"
Root: HKCR; Subkey: "ISXLicenseFile\shell\open\command"; ValueType: string; ValueName: ""; ValueData: """{app}\ISXPulse.exe"" ""%1"""

; Application settings
Root: HKCU; Subkey: "Software\ISX Pulse\Scratch Card"; ValueType: string; ValueName: "InstallPath"; ValueData: "{app}"; Flags: uninsdeletekey
Root: HKCU; Subkey: "Software\ISX Pulse\Scratch Card"; ValueType: string; ValueName: "Version"; ValueData: "{#AppVersion}"; Flags: uninsdeletekey
Root: HKCU; Subkey: "Software\ISX Pulse\Scratch Card"; ValueType: dword; ValueName: "ScratchCardMode"; ValueData: 1; Flags: uninsdeletekey

[Run]
; Post-installation configuration
Filename: "{app}\scripts\setup-scratch-card-credentials.bat"; Description: "Setup Scratch Card Credentials"; Flags: postinstall unchecked shellexec
Filename: "{app}\ISXPulse.exe"; Parameters: "--version"; Description: "Verify Installation"; Flags: postinstall unchecked shellexec
Filename: "notepad.exe"; Parameters: "{app}\docs\README.md"; Description: "View Documentation"; Flags: postinstall unchecked shellexec
Filename: "{sys}\netsh.exe"; Parameters: "advfirewall firewall add rule name=""ISX Pulse Server"" dir=in action=allow protocol=TCP localport=8080"; Description: "Configure Windows Firewall"; Flags: postinstall runhidden; Tasks: firewall

[UninstallRun]
; Cleanup firewall rules
Filename: "{sys}\netsh.exe"; Parameters: "advfirewall firewall delete rule name=""ISX Pulse Server"""; Flags: runhidden

[Code]
var
  AppsScriptURLPage: TInputQueryWizardPage;
  ConfigSummaryPage: TOutputMsgWizardPage;

procedure InitializeWizard;
begin
  // Create Apps Script URL input page
  AppsScriptURLPage := CreateInputQueryPage(wpSelectComponents,
    'Google Apps Script Configuration',
    'Configure your Google Apps Script URL for license management',
    'Enter the Google Apps Script web app URL that you deployed for license management. ' +
    'This URL will be used for scratch card activation and validation.');
  
  AppsScriptURLPage.Add('Apps Script URL:', False);
  AppsScriptURLPage.Values[0] := 'https://script.google.com/macros/s/YOUR_SCRIPT_ID/exec';

  // Create configuration summary page
  ConfigSummaryPage := CreateOutputMsgPage(wpReady,
    'Installation Summary',
    'Ready to install ISX Pulse Scratch Card Edition',
    'The installer will now install ISX Pulse with the following configuration:' + #13#10#13#10 +
    'Features:' + #13#10 +
    '• One-time activation scratch card system' + #13#10 +
    '• Device fingerprinting and binding' + #13#10 +
    '• Rate limiting and blacklisting' + #13#10 +
    '• Complete audit logging' + #13#10 +
    '• Google Apps Script integration' + #13#10#13#10 +
    'Post-installation, you will need to:' + #13#10 +
    '1. Configure your Google Apps Script URL' + #13#10 +
    '2. Set up Google Sheets API credentials' + #13#10 +
    '3. Generate initial scratch cards' + #13#10 +
    '4. Deploy the system');
end;

function NextButtonClick(CurPageID: Integer): Boolean;
begin
  Result := True;
  
  if CurPageID = AppsScriptURLPage.ID then
  begin
    // Validate Apps Script URL format
    if (AppsScriptURLPage.Values[0] = '') or 
       (Pos('script.google.com/macros/s/', AppsScriptURLPage.Values[0]) = 0) then
    begin
      MsgBox('Please enter a valid Google Apps Script URL in the format:' + #13#10 +
             'https://script.google.com/macros/s/YOUR_SCRIPT_ID/exec', 
             mbError, MB_OK);
      Result := False;
    end;
  end;
end;

procedure CurStepChanged(CurStep: TSetupStep);
var
  EnvFile: string;
  ConfigLines: TArrayOfString;
begin
  if CurStep = ssPostInstall then
  begin
    // Create .env file with user's configuration
    EnvFile := ExpandConstant('{app}\.env');
    
    SetArrayLength(ConfigLines, 20);
    ConfigLines[0] := '# ISX Pulse - Scratch Card Configuration';
    ConfigLines[1] := '# Generated during installation';
    ConfigLines[2] := '';
    ConfigLines[3] := '# Server Configuration';
    ConfigLines[4] := 'PORT=8080';
    ConfigLines[5] := 'HOST=0.0.0.0';
    ConfigLines[6] := 'ENV=production';
    ConfigLines[7] := '';
    ConfigLines[8] := '# Google Apps Script Configuration';
    ConfigLines[9] := 'GOOGLE_APPS_SCRIPT_URL=' + AppsScriptURLPage.Values[0];
    ConfigLines[10] := '';
    ConfigLines[11] := '# Scratch Card Features';
    ConfigLines[12] := 'ENABLE_SCRATCH_CARD_MODE=true';
    ConfigLines[13] := 'ENABLE_DEVICE_FINGERPRINTING=true';
    ConfigLines[14] := 'ENABLE_ONE_TIME_ACTIVATION=true';
    ConfigLines[15] := '';
    ConfigLines[16] := '# Security Configuration';
    ConfigLines[17] := 'SCRATCH_CARD_MAX_ATTEMPTS_PER_HOUR=10';
    ConfigLines[18] := 'BLACKLIST_CHECK_ENABLED=true';
    ConfigLines[19] := 'AUDIT_LOG_ENABLED=true';
    
    SaveStringsToFile(EnvFile, ConfigLines, False);
  end;
end;

function ShouldSkipPage(PageID: Integer): Boolean;
begin
  Result := False;
  
  // Skip Apps Script URL page if generator-only installation
  if (PageID = AppsScriptURLPage.ID) and (WizardSelectedComponents(False) = 'generator') then
    Result := True;
end;

[Messages]
WelcomeLabel2=This will install [name/ver] with scratch card license system on your computer.%n%nThis edition features one-time activation scratch cards with device binding, rate limiting, and comprehensive audit logging.%n%nIt is recommended that you close all other applications before continuing.

[CustomMessages]
ComponentsDescription=Select the components to install:
FullInstallation=Complete installation with server, tools, and documentation
ServerOnly=ISX Pulse server with scratch card support only  
GeneratorOnly=License generator and management tools only
CustomInstallation=Choose which components to install

CreateDesktopIcon=Create a &desktop icon
CreateQuickLaunchIcon=Create a &Quick Launch icon