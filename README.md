# ISX Pulse
## The Heartbeat of Iraqi Markets

A professional Go-based analytics platform for real-time monitoring, processing, and analysis of Iraq Stock Exchange (ISX) trading data, featuring a modern Next.js dashboard and enterprise-grade security.

## Overview

This application provides automated data collection and processing for ISX trading data, featuring:
- Automated Excel report downloading from ISX
- Data transformation and CSV export
- Real-time WebSocket updates
- Hardware-based license activation
- Modern React/Next.js frontend with TypeScript
- Comprehensive observability with OpenTelemetry

## Architecture

### Backend (Go)
- **Web Server**: Chi router v5 with embedded frontend
- **Operations System**: Multi-step concurrent data processing
- **WebSocket**: Real-time updates using Gorilla WebSocket
- **License Management**: Hardware fingerprinting and activation
- **Observability**: Structured logging (slog) and OpenTelemetry

### Frontend (Next.js)
- **Framework**: Next.js 14 with TypeScript
- **UI Components**: Shadcn/ui with Tailwind CSS
- **Real-time Updates**: WebSocket integration
- **Static Export**: Embedded in Go binary

### Executables
- `ISXPulse`: Main analytics server with embedded dashboard (port 8080)
- `ISXScraper`: Automated ISX report downloader
- `ISXProcessor`: Data transformation and CSV export engine
- `ISXIndexer`: Market index extractor (ISX60/ISX15)

## Quick Start

### Prerequisites
- Go 1.21+
- Node.js 18+
- Windows OS (for production builds)

### Building

```bash
# Complete build (backend + frontend)
./build.bat        # Windows

# Development
cd api
go mod tidy
go test ./... -race
```

### Running

```bash
# Start ISX Pulse server
./dist/ISXPulse.exe

# Run data collection
./dist/ISXScraper.exe

# Process market data
./dist/ISXProcessor.exe

# Extract market indices
./dist/ISXIndexer.exe
```

## Configuration

### Required Files
- `credentials.json`: Google Sheets API credentials
- `sheets-config.json`: Sheet ID mappings
- `encrypted_credentials.dat`: Production credentials (encrypted)

### Environment Variables
- `ISX_PORT`: Server port (default: 8080)
- `ISX_LOG_LEVEL`: Logging level (default: info)
- `ISX_LICENSE_SERVER`: License activation server URL

## Development

See [CLAUDE.md](CLAUDE.md) for comprehensive development guidelines, including:
- Code standards and best practices
- Architecture patterns
- Testing requirements
- Documentation standards

## License

ISX Pulse requires a valid license for operation. Features include:
- Hardware-locked activation for security
- Smart device recognition for seamless reactivation
- Automatic same-device license reactivation after reinstalls
- Up to 5 reactivations per 30-day period

For licensing information, contact support. For technical details on the reactivation system, see [License Reactivation Guide](docs/LICENSE_REACTIVATION_GUIDE.md).

## Change Log

- 2025-08-19: Added smart device recognition and automatic license reactivation
- 2025-08-05: Rebranded to ISX Pulse - "The Heartbeat of Iraqi Markets"
- 2025-07-30: Added operations system replacing legacy operation architecture
- 2025-01-26: Initial release with embedded frontend and license management