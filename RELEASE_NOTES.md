# ISX Pulse v0.0.1-alpha - Release Notes
## The Heartbeat of Iraqi Markets

**Release Date**: August 5, 2025  
**Package**: ISXPulse-v0.0.1-alpha-win64.zip (22 MB)

## 🎉 Release Highlights

This release represents a major milestone with a completely professionalized project structure and clean distribution package.

## 📦 Package Contents

### Executables (4)
- **ISXPulse.exe** (21 MB) - Main analytics server with REST API and embedded dashboard
- **ISXScraper.exe** (20 MB) - Automated ISX report downloader
- **ISXProcessor.exe** (9.2 MB) - Data transformation and CSV export engine
- **ISXIndexer.exe** (9.1 MB) - Market index extractor (ISX60/ISX15)

### Configuration Files
- **credentials.json.example** - Template for Google Sheets API service account
- **sheets-config.json.example** - Template for Google Sheets configuration

### Scripts
- **start-server.bat** - Quick start for the web server
- **run-scraper-workflow.bat** - Automated workflow for data collection

### Directory Structure
```
dist/
├── data/
│   ├── downloads/    # Excel files from ISX
│   └── reports/      # Generated CSV reports
├── logs/             # Application logs
└── README.md         # Quick start guide
```

## 🚀 What's New in v0.0.1-alpha

### Alpha Release - ISX Pulse
- **New Identity**: Introducing "ISX Pulse - The Heartbeat of Iraqi Markets"
- **Professional Naming**: All executables now use ISX-prefixed professional names
- **Modern Build System**: Updated to use `dist/` directory for cleaner organization
- **Alpha Features**: This is an early alpha release for testing and feedback

### Project Structure Improvements
- **40% reduction** in root directory files (20 → 12 files)
- **Single build system** using `build.go` (577 lines)
- **Professional organization**:
  - All source in `dev/`
  - All docs in `docs/`
  - All tools in `tools/`
- **Removed redundant scripts**: 4 build scripts consolidated to 1

### Technical Improvements
- **Optimized executables** with `-ldflags="-s -w"` for smaller size
- **Clean architecture** following Go best practices
- **Embedded frontend** in ISXPulse.exe for single-file deployment
- **Structured logging** with slog throughout

### Documentation
- Comprehensive `FILE_INDEX.md` documenting entire structure
- Updated `CLAUDE.md` with project structure reference
- Reorganized all documentation in `docs/` directory
- Added `QUICK_START.md` for easy onboarding

## 💻 System Requirements

- **OS**: Windows 10 or later (64-bit)
- **RAM**: 4 GB minimum
- **Storage**: 100 MB for application + space for data
- **Network**: Internet connection for ISX data download

## 🔧 Installation

1. Download `ISXPulse-v0.0.1-alpha-win64.zip`
2. Extract to desired location (e.g., `C:\ISXPulse`)
3. Configure credentials (see README.md in package)
4. Run `start-server.bat` to start

## 📝 Configuration

1. Copy `credentials.json.example` to `credentials.json`
2. Add your Google service account credentials
3. Copy `sheets-config.json.example` to `sheets-config.json`
4. Update with your Google Sheet IDs

## 🏃 Quick Start

```cmd
# Start ISX Pulse server
start-server.bat

# Run complete data collection workflow
run-scraper-workflow.bat

# Or run individual components
ISXScraper.exe    # Download market data
ISXProcessor.exe  # Transform and analyze
ISXIndexer.exe    # Extract market indices
```

## 🔒 Security Notes

- Credentials are never included in the release
- Support for encrypted credentials via `encrypted_credentials.dat`
- All sensitive data protected with AES-256-GCM encryption

## 📊 API Endpoints

Web server runs on http://localhost:8080 with:
- Health check: `GET /health`
- API routes: `/api/v1/*`
- WebSocket: `/ws` for real-time updates

## 🐛 Known Issues

- Frontend requires Node.js to build from source (pre-built included)
- Some antivirus may flag Go executables (false positive)

## 📚 Documentation

Full documentation available in the source repository:
- API Reference: `docs/API_REFERENCE.md`
- Security Guide: `docs/SECURITY.md`
- Development Guide: `docs/DEVELOPER_QUICK_REFERENCE.md`

## 🤝 Support

For issues or questions:
- Check README.md in the release package
- Review documentation in source repository
- Report issues on project repository

---

**Note**: This is a clean release build with optimized executables. The total package size of 22 MB includes all four executables with embedded resources.