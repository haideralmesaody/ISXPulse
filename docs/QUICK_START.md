# ISX Daily Reports Scrapper - Quick Start Guide

## Project Structure

```
ISXDailyReportsScrapper/
├── dev/          # All source code
├── docs/         # All documentation  
├── tools/        # Development utilities
├── installer/    # Windows installer
└── build.bat     # Main build command
```

## Essential Commands

### Build Everything
```bash
build.bat
# OR
build.bat -target=all
```

### Build Specific Target
```bash
build.bat -target=web        # Build web server only
build.bat -target=frontend   # Build Next.js frontend only
build.bat -target=release    # Build release package
```

### Setup Credentials
```bash
setup-production-credentials.bat
```

### Run Tests
```bash
cd tools
test.bat
```

### Start Server
```bash
start-server.bat
```

## Key Directories

- **Source Code**: `dev/`
  - Go code: `dev/cmd/`, `dev/internal/`, `dev/pkg/`
  - Frontend: `dev/frontend/`
  
- **Documentation**: `docs/`
  - Security: `docs/SECURITY.md`
  - API Reference: `docs/API_REFERENCE.md`
  - Deployment: `docs/DEPLOYMENT_GUIDE.md`

- **Development Tools**: `tools/`
  - Test runner: `tools/test.bat`
  - Credential setup: `tools/setup-production-credentials.ps1`

## Quick Development Workflow

1. **Clone the repository**
2. **Setup credentials**: Run `setup-production-credentials.bat`
3. **Build everything**: Run `build.bat`
4. **Start server**: Run `start-server.bat`

## Where to Find Things

| What | Where |
|------|-------|
| Build system | `build.go` (use `build.bat` wrapper) |
| Main server | `dev/cmd/web-licensed/` |
| Frontend code | `dev/frontend/` |
| API contracts | `dev/pkg/contracts/` |
| Documentation | `docs/` |
| Dev utilities | `tools/` |

## Next Steps

- Read `docs/DEVELOPER_QUICK_REFERENCE.md` for detailed development guide
- Check `docs/SECURITY.md` for security setup
- See `docs/API_REFERENCE.md` for API documentation