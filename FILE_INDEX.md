# ISX Pulse (ISX Daily Reports Scrapper) - File Index

Generated: 2025-08-07
Updated: 2025-08-21
Purpose: Up-to-date index of project files and directories to aid navigation and onboarding.

Overview
- **Project Name**: ISX Pulse (Professional Financial Data Processing System)
- **Languages**: Go 1.21+ (backend), TypeScript/React with Next.js 14 (frontend)
- **Architecture**: Clean architecture with embedded frontend in single binary
- **Key areas**: api/ (Go backend), web/ (Next.js frontend), dist/ (build outputs), docs/ (documentation)
- **CI**: .github/workflows/ci.yml
- **Build System**: build.go with build.bat wrapper (MUST use from root only)
- **Installers**: installer/*.iss with assets/

Top-level layout (Clean Structure - 12 Essential Files)
- .claude/                 AI agent config and specialized agents
- .editorconfig            Editor configuration
- .env.example             Environment variable template
- .github/workflows/       CI/CD workflows
- .gitignore              Git ignore rules
- .mcp.json               MCP configuration
- BUILD_RULES.md          Critical build rules (MUST READ)
- CHANGELOG.md            Version history
- CLAUDE.md               AI assistant guidelines
- CONTRIBUTING.md         Contribution guidelines
- FILE_INDEX.md           This index
- README.md               Project overview
- RELEASE_NOTES.md        Aggregate release notes
- SECURITY.md             Security policy
- build.bat               Windows build wrapper (PRIMARY BUILD METHOD)
- build.go                Go build system (577 lines)
- api/                    Go backend source code
- config/                 Configuration examples
- dist/                   Build outputs (executables, data, logs)
- docs/                   Comprehensive documentation
- installer/              Windows installer scripts
- logs/                   Application logs
- monitoring/             Monitoring configurations
- scripts/                Build and deployment scripts
- tools/                  Development utilities
- web/                    Next.js frontend source code

Documentation (docs/)
- API_REFERENCE.md - Complete API documentation
- DEPLOYMENT_GUIDE.md - Production deployment instructions
- ENHANCED_SECURITY_IMPLEMENTATION.md - Security features
- FREQTRADE_IMPLEMENTATION_PLAN.md - Trading bot integration
- GOOGLE_SHEETS_SETUP_INSTRUCTIONS.md - Sheets API setup
- LICENSE_REACTIVATION_GUIDE.md - License management
- LIQUIDITY_*.md - Liquidity analysis documentation suite
- MCP_SETUP.md - MCP configuration guide
- MIGRATION_GUIDE.md - Version migration guide
- OPERATION_FLOWS.md - Operation pipeline documentation
- PRODUCTION_DEPLOYMENT_CHECKLIST.md - Deployment checklist
- QUICK_START.md - Quick start guide
- REACT_HYDRATION_GUIDE.md - React SSR/hydration best practices
- README.md - Documentation overview
- SECURITY_*.md - Security documentation and audits
- UI_COMPONENT_ARCHITECTURE.md - Frontend architecture
- archive/ - Historical documentation
- releases/ - Version-specific release notes

Backend Structure (api/)
- go.mod, go.sum - Go module dependencies
- cmd/ - Application entry points
  - indexcsv/ - Extract ISX index values
  - liquidity-report/ - Liquidity analysis tool
  - processor/ - Process Excel to CSV
  - scraper/ - Download ISX reports
  - web-licensed/ - Primary licensed web server with embedded frontend
- internal/ - Private application packages
  - app/ - Application initialization
  - config/ - Configuration management
  - dataprocessing/ - Data parsing and processing
  - errors/ - RFC 7807 compliant error handling
  - exporter/ - Data export formats
  - files/ - File management
  - infrastructure/ - Logging, metrics, observability
  - integration/ - Integration tests
  - license/ - License management system
  - liquidity/ - Liquidity scoring and analysis
  - middleware/ - HTTP middleware (Chi v5)
  - operations/ - Pipeline operations management
  - performance/ - Performance tests
  - security/ - Encryption, authentication
  - services/ - Business logic layer
  - shared/ - Shared utilities
  - transport/http/ - HTTP handlers
  - updater/ - Auto-update functionality
  - validation/ - Input validation
  - websocket/ - WebSocket handlers
- pkg/contracts/ - Shared types and interfaces
  - api/v1/ - API request/response contracts
  - domain/ - Domain entities
  - events/ - Event contracts
- tests/ - Test suites
  - e2e/ - End-to-end tests
  - integration/ - Integration tests
  - testutil/ - Test utilities

Frontend Structure (web/)
- package.json, package-lock.json - NPM dependencies
- next.config.js - Next.js configuration
- tailwind.config.js - Tailwind CSS configuration
- tsconfig.json - TypeScript configuration
- app/ - Next.js app router pages
  - analysis/ - Analysis page
  - dashboard/ - Dashboard page
  - license/ - License activation
  - liquidity/ - Liquidity analysis
  - operations/ - Operations management
  - reports/ - Reports viewer
  - layout.tsx - Root layout
  - page.tsx - Home page
- components/ - React components
  - ui/ - Shadcn/ui components
  - operations/ - Operation-specific components
  - layout/ - Layout components
  - license/ - License components
  - reports/ - Report components
  - analysis/ - Analysis components
- lib/ - Utilities and hooks
  - api/ - API client
  - hooks/ - React hooks (including useHydration)
  - schemas/ - Zod schemas
  - utils/ - Utility functions
  - constants/ - Application constants
  - observability/ - Monitoring utilities
- public/ - Static assets
- styles/ - Global styles
- tests/ - Test suites
- types/ - TypeScript type definitions

Build Outputs (dist/)
- ISXPulse.exe - Main web server with embedded frontend
- scraper.exe - ISX data scraper
- processor.exe - Data processor
- indexcsv.exe - Index extractor
- data/ - Runtime data storage
  - downloads/ - Excel files from ISX
  - reports/ - Generated CSV reports
- logs/ - Application logs
- web/ - Static assets backup
- license.dat - Activated license file

Scripts and Tools
- scripts/ - Build and deployment scripts
  - deploy-scratch-card.bat - Scratch card deployment
  - migrate-reports.bat/go - Report migration tools
  - package-release.bat - Release packaging
  - setup-*.bat - Various setup scripts
- tools/ - Development utilities
  - verify-no-dev-builds.bat - Build compliance checker
  - license-generator/ - License generation tool
  - test-harnesses/ - HTML test interfaces

Critical Build Rules (BUILD_RULES.md)
1. NEVER run 'npm run build' directly in web/
2. NEVER run 'go build' directly in api/
3. ALWAYS use ./build.bat from project root
4. All builds output to dist/ directory only
5. No .next/ or out/ directories allowed in web/
6. Logs are automatically cleared before each build

Notable Updates (2025-08-21)
- Project rebranded to ISX Pulse
- Migrated from dev/ structure to api/ and web/
- Added liquidity analysis features
- Enhanced security with embedded credentials
- Improved build system compliance
- 392 files modified in current branch (fix/embedded-credentials-system)

How to keep this file updated
- Update after major structural changes
- Run `find . -type d -maxdepth 2` to check directory structure
- Use `git status --short | wc -l` to count changed files
- Verify build compliance with `./tools/verify-no-dev-builds.bat`
