# ISX Pulse - Documentation
## The Heartbeat of Iraqi Markets

This directory contains comprehensive documentation for ISX Pulse, the professional analytics platform for Iraqi Stock Exchange data.

## Documentation Index

### 🔧 Development Guides
- [`DEVELOPER_QUICK_REFERENCE.md`](./DEVELOPER_QUICK_REFERENCE.md) - Quick reference guide for developers
- [`NEXT_JS_ARCHITECTURE.md`](./NEXT_JS_ARCHITECTURE.md) - Frontend architecture and component structure
- [`UI_COMPONENT_ARCHITECTURE.md`](./UI_COMPONENT_ARCHITECTURE.md) - Unified UI component architecture (2025-08-09)
- [`EXECUTION_DEPENDENCY_CHART.md`](./EXECUTION_DEPENDENCY_CHART.md) - System execution flow and dependencies

### 🔒 Security Documentation
- [`SECURITY.md`](./SECURITY.md) - **Primary Security Documentation** (OWASP ASVS Level 3 compliant)
  - Includes comprehensive security architecture
  - Credential management and encryption details
  - Authentication and authorization systems
  - Incident response procedures
- [`SECURITY_IMPLEMENTATION_GUIDE.md`](./SECURITY_IMPLEMENTATION_GUIDE.md) - Security implementation best practices

### 📚 API & Operations
- [`API_REFERENCE.md`](./API_REFERENCE.md) - Complete REST API documentation
  - Endpoint specifications
  - Request/response formats
  - WebSocket integration
  - TypeScript types
- [`OPERATION_FLOWS.md`](./OPERATION_FLOWS.md) - Data operation flow documentation
  - Frontend-backend communication
  - Multi-step operation processing
  - WebSocket real-time updates

### 🚀 Deployment & Production
- [`PRODUCTION_INTEGRATION_GUIDE.md`](./PRODUCTION_INTEGRATION_GUIDE.md) - Production deployment guide
- [`../PRODUCTION_DEPLOYMENT_CHECKLIST.md`](../PRODUCTION_DEPLOYMENT_CHECKLIST.md) - Comprehensive deployment checklist
- [`../DEPLOYMENT_GUIDE.md`](../DEPLOYMENT_GUIDE.md) - General deployment instructions

### 📊 Reports & Audits
- Various validation reports may be generated during development
- One-time reports are excluded from version control per `.gitignore`

## Quick Links

### Essential Files
1. **Getting Started**: Start with [`DEVELOPER_QUICK_REFERENCE.md`](./DEVELOPER_QUICK_REFERENCE.md)
2. **Security Setup**: Review [`SECURITY.md`](./SECURITY.md) for credential management
3. **API Integration**: See [`API_REFERENCE.md`](./API_REFERENCE.md) for endpoint details
4. **Frontend Development**: Check [`NEXT_JS_ARCHITECTURE.md`](./NEXT_JS_ARCHITECTURE.md)

### Related Documentation
- **Main README**: [`../README.md`](../README.md) - Project overview
- **CLAUDE.md**: [`../CLAUDE.md`](../CLAUDE.md) - AI assistant instructions
- **CHANGELOG**: [`../CHANGELOG.md`](../CHANGELOG.md) - Version history

## Change Log

### 2025-08-09 - UI Component Simplification
- **ADDED**: `UI_COMPONENT_ARCHITECTURE.md` - Comprehensive documentation of unified component architecture
- **REMOVED**: `ScrapingProgress` component and related files (5 files, ~800 lines)
- **SIMPLIFIED**: WebSocket flow reduced from 130+ to 20 lines in hub.go
- **CREATED**: Unified `StepProgress` and `MetadataGrid` components
- **IMPACT**: 70% reduction in UI code (1200 → 450 lines) with improved maintainability

### 2025-07-30 - Task 0.1: Documentation Cleanup
- **REMOVED**: `docs/MASTER_DEVELOPMENT_PLAN_V3.0.md` - Replaced by OPERATIONS_IMPLEMENTATION_PLAN.md
- **REMOVED**: `docs/PHASE_4_SERVICE_MIGRATION_SUMMARY.md` - Migration completed
- **REMOVED**: `docs/PHASE_5_HANDLER_MIGRATION_SUMMARY.md` - Migration completed  
- **REMOVED**: `dev/docs/TYPE_CONSOLIDATION_PLAN.md` - Consolidation completed
- **REMOVED**: `LICENSE_SYSTEM_SIMPLIFIED.md` - Outdated license documentation
- **FIXED**: Updated broken reference in DEVELOPER_QUICK_REFERENCE.md to point to pkg/contracts documentation
- **ADDED**: Created docs/README.md for documentation index and change tracking

**Total files removed**: 5
**Rationale**: Cleanup of outdated documentation files that were no longer relevant after completion of migration phases and consolidation tasks.

### 2025-07-30 - Task 0.3: Old Web Interface Removal
- **REMOVED**: `dev/web/` directory - Complete legacy HTML/CSS/JS interface (35+ files)
- **REMOVED**: `release/web/static/` and `release/web/templates/` - Legacy static assets and templates (38+ files)
- **ARCHITECTURE**: Transitioned from 1040+ lines of HTML templates to modern React components
- **MIGRATION**: Legacy web interface replaced by Next.js frontend in `dev/frontend/`
- **BUILD**: Updated build processes to use embedded Next.js static exports

**Total files removed**: 73
**Impact**: Major architectural shift from server-side HTML templates to modern React-based SPA with embedded static assets