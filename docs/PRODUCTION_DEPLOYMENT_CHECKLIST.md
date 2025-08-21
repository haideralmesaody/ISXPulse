# ISX Daily Reports Scrapper - Production Deployment Checklist

**Version**: 2.0 | **Security Level**: A+ | **OWASP ASVS**: Level 3 Compliant | **Date**: 2025-01-31

## Overview

This comprehensive checklist ensures secure and reliable deployment of the ISX Daily Reports Scrapper in production environments. Each item must be verified and signed off before go-live.

**Deployment Type**: Windows Server Production Environment  
**Target Audience**: System Administrators, DevOps Engineers, Security Teams

---

## Pre-Deployment Phase

### [ ] 1. Infrastructure Requirements

#### System Requirements
- [ ] **Operating System**: Windows Server 2019/2022 (64-bit) or Windows 10/11 Pro
- [ ] **CPU**: Minimum 2 cores, 2.4 GHz (Recommended: 4 cores, 3.0 GHz)
- [ ] **Memory**: Minimum 4 GB RAM (Recommended: 8 GB RAM)
- [ ] **Storage**: Minimum 2 GB free space (Recommended: 10 GB SSD)
- [ ] **Network**: Internet connectivity for Google Sheets API access
- [ ] **PowerShell**: Version 5.1 or higher installed

#### Network Requirements
- [ ] **Outbound HTTPS (443)**: Access to sheets.googleapis.com verified
- [ ] **Inbound HTTP (8080)**: Web interface port available
- [ ] **DNS Resolution**: Google APIs accessible via DNS
- [ ] **Firewall Configuration**: Windows Firewall enabled and configured
- [ ] **Proxy Settings**: Corporate proxy configured if required

### [ ] 2. Security Prerequisites

#### Windows Server Hardening
- [ ] **Latest Updates**: All Windows updates installed
- [ ] **Antivirus**: Windows Defender or enterprise antivirus enabled
- [ ] **User Accounts**: Service account `ISXService` created with minimal privileges
- [ ] **Password Policy**: Strong password policy enforced
- [ ] **Audit Logging**: Windows audit logging enabled
- [ ] **Remote Access**: RDP secured or disabled as per security policy

#### SSL/TLS Configuration (Optional but Recommended)
- [ ] **SSL Certificate**: Valid SSL certificate obtained for HTTPS
- [ ] **Certificate Store**: Certificate installed in Windows certificate store
- [ ] **HTTPS Binding**: Port 8443 configured for HTTPS access
- [ ] **Certificate Chain**: Full certificate chain validated

### [ ] 3. Google Cloud Setup

#### Service Account Configuration
- [ ] **Google Cloud Project**: Project created or identified
- [ ] **Sheets API**: Google Sheets API enabled in project
- [ ] **Service Account**: Production service account created
  - Name: `isx-reports-production`
  - Description: `ISX Daily Reports Production Service Account`
  - Role: None (permissions granted at sheet level)
- [ ] **JSON Key**: Service account JSON key generated and downloaded
- [ ] **Key Security**: JSON key stored securely and not committed to version control

#### Google Sheets Access
- [ ] **Sheet Sharing**: Required Google Sheets shared with service account email
- [ ] **Permission Level**: Service account granted "Editor" access to sheets
- [ ] **Sheet IDs**: All required sheet IDs documented
- [ ] **Access Testing**: Service account access to sheets verified

---

## Build and Deployment Phase

### [ ] 4. Application Build

#### Source Code Preparation
- [ ] **Source Code**: Latest stable code pulled from repository
- [ ] **Branch**: Production branch (usually `main` or `release`) checked out
- [ ] **Dependencies**: All Go modules updated with `go mod tidy`
- [ ] **Version Tag**: Appropriate version tag applied to build

#### Development Credential Setup
- [ ] **Service Account JSON**: Placed in `dev/credentials.json`
- [ ] **Credential Validation**: JSON structure and content verified
- [ ] **API Testing**: Google Sheets API access tested with development credentials
- [ ] **Sheets Config**: Development sheets-config.json created for testing

#### Production Credential Encryption
- [ ] **Encryption Script**: `setup-production-credentials.ps1` executed successfully
- [ ] **Encrypted File**: `encrypted_credentials.dat` created
- [ ] **File Size**: Encrypted file size is reasonable (typically 1-10 KB)
- [ ] **Validation**: Encrypted credentials can be decrypted successfully
- [ ] **Cleanup**: Development `credentials.json` removed from system

#### Build Process
- [ ] **Frontend Build**: Next.js frontend built successfully (`npm run build`)
- [ ] **Frontend Embedding**: Frontend assets embedded in Go binary
- [ ] **Go Build**: All executables built successfully
  - [ ] `web.exe` (main server with embedded frontend)
  - [ ] `scraper.exe` (ISX website scraper)
  - [ ] `processor.exe` (Excel to CSV processor)
  - [ ] `indexcsv.exe` (index value extractor)
- [ ] **Build Verification**: All executables created and have reasonable file sizes
- [ ] **Dependency Check**: No external dependencies required at runtime

### [ ] 5. File Deployment

#### Directory Structure Creation
- [ ] **Base Directory**: `C:\ISXReports` created with correct permissions
- [ ] **Subdirectories**: All required subdirectories created:
  - [ ] `C:\ISXReports\bin` (executables)
  - [ ] `C:\ISXReports\config` (configuration files)
  - [ ] `C:\ISXReports\data\downloads` (Excel files)
  - [ ] `C:\ISXReports\data\reports` (CSV reports)
  - [ ] `C:\ISXReports\logs` (application logs)
  - [ ] `C:\ISXReports\backup` (backups)

#### File Deployment
- [ ] **Executables**: All `.exe` files copied to `C:\ISXReports\bin\`
- [ ] **Configuration**: `encrypted_credentials.dat` copied to config directory
- [ ] **Sheets Config**: `sheets-config.json` created in config directory
- [ ] **License File**: `license.dat` copied to config directory (if available)
- [ ] **File Permissions**: Correct permissions set on all files and directories
- [ ] **File Integrity**: File hashes verified against build artifacts

#### Permission Configuration
- [ ] **Service Account Access**: `ISXService` has appropriate access to all directories
- [ ] **Admin Access**: Administrators have full control
- [ ] **User Access**: Standard users have no access to sensitive files
- [ ] **Executable Permissions**: Service account can execute all binaries
- [ ] **Config Protection**: Configuration files are read-only for service account
- [ ] **Log Permissions**: Service account can write to logs directory

---

## Configuration Phase

### [ ] 6. Application Configuration

#### License Configuration
- [ ] **License File**: Valid `license.dat` file in config directory
- [ ] **License Format**: License file format validated
- [ ] **Hardware Binding**: License compatible with target hardware
- [ ] **Expiration**: License expiration date verified (should not expire soon)
- [ ] **Activation**: License activation tested successfully

#### Google Sheets Configuration
- [ ] **Production Config**: `sheets-config.json` configured for production environment
- [ ] **Sheet IDs**: All production sheet IDs entered correctly
- [ ] **Sheet Ranges**: Correct cell ranges specified for each sheet
- [ ] **Environment**: Production environment section populated
- [ ] **Validation**: Configuration file syntax validated

#### Environment Variables
- [ ] **System Variables**: Required environment variables set at system level:
  - [ ] `ISX_ENVIRONMENT=production`
  - [ ] `ISX_LOG_LEVEL=info`
  - [ ] `ISX_PORT=8080`
  - [ ] `ISX_DATA_DIR=C:\ISXReports\data`
- [ ] **Variable Persistence**: Environment variables persist across reboots
- [ ] **Variable Access**: Service account can access environment variables

### [ ] 7. Security Configuration

#### Windows Security Hardening
- [ ] **Security Script**: `security-hardening.ps1` executed successfully
- [ ] **Firewall Rules**: Appropriate firewall rules created and enabled
- [ ] **Service Account**: ISXService account configured with minimal privileges
- [ ] **UAC Configuration**: User Account Control properly configured
- [ ] **Audit Logging**: Windows audit logging enabled for security events
- [ ] **Antivirus Exclusions**: ISX directories excluded from real-time scanning (performance)

#### Application Security
- [ ] **Security Headers**: Security headers enabled and configured
- [ ] **CORS Policy**: CORS configured for production domains only
- [ ] **Rate Limiting**: Rate limiting enabled with appropriate thresholds
- [ ] **Input Validation**: Input validation active on all endpoints
- [ ] **Error Handling**: Error messages don't expose sensitive information
- [ ] **Session Management**: Secure session management configured

#### Network Security
- [ ] **Inbound Rules**: Only necessary inbound ports (8080, optional 8443) allowed
- [ ] **Outbound Rules**: Outbound access restricted to Google APIs only
- [ ] **Certificate Pinning**: Google APIs certificate pinning enabled
- [ ] **TLS Configuration**: Strong TLS settings configured if HTTPS enabled
- [ ] **DNS Security**: Secure DNS resolution configured

---

## Service Installation Phase

### [ ] 8. Windows Service Setup

#### Service Installation
- [ ] **Service Wrapper**: Service wrapper script created if using NSSM
- [ ] **Service Installation**: Windows service installed successfully
  - Service Name: `ISXReportsWeb`
  - Display Name: `ISX Daily Reports Web Service`
  - Description: `ISX Daily Reports Scrapper Web Interface and API`
- [ ] **Service Account**: Service configured to run as `ISXService` account
- [ ] **Startup Type**: Service set to "Automatic" startup
- [ ] **Recovery Options**: Service recovery actions configured (restart on failure)

#### Service Configuration
- [ ] **Service Parameters**: Correct command line parameters configured
- [ ] **Working Directory**: Service working directory set correctly
- [ ] **Environment Variables**: Service inherits system environment variables
- [ ] **Log On Permissions**: Service account has "Log on as a service" right
- [ ] **Dependencies**: Any service dependencies configured if required

### [ ] 9. Initial Testing

#### Service Startup Testing
- [ ] **Manual Start**: Service can be started manually without errors
- [ ] **Automatic Start**: Service starts automatically after system reboot
- [ ] **Process Validation**: Correct process starts when service runs
- [ ] **Port Binding**: Service successfully binds to configured port (8080)
- [ ] **Resource Usage**: Memory and CPU usage within expected ranges

#### Application Testing
- [ ] **Health Endpoints**: All health check endpoints responding correctly:
  - [ ] `GET /api/health` returns `{"status": "ok"}`
  - [ ] `GET /api/health/ready` returns readiness status
  - [ ] `GET /api/health/live` returns liveness status
  - [ ] `GET /api/version` returns version information
- [ ] **License Validation**: `GET /api/license/status` returns valid license info
- [ ] **Web Interface**: Frontend loads correctly in web browser
- [ ] **API Endpoints**: Core API endpoints respond appropriately

#### Integration Testing
- [ ] **Google Sheets Access**: Application can read from configured Google Sheets
- [ ] **Data Processing**: Sample data processing workflow completes successfully
- [ ] **File Operations**: Application can read/write to data directories
- [ ] **Logging**: Application logs are written to expected location
- [ ] **Error Handling**: Application handles errors gracefully

---

## Monitoring and Alerting Phase

### [ ] 10. Monitoring Setup

#### Health Check Monitoring
- [ ] **Health Check Script**: `health-check.ps1` scheduled task created
- [ ] **Monitoring Frequency**: Health checks run every 5 minutes
- [ ] **Alert Thresholds**: Appropriate thresholds set for alerting
- [ ] **Escalation Procedures**: Alert escalation procedures documented
- [ ] **Response Procedures**: Incident response procedures prepared

#### Performance Monitoring
- [ ] **Performance Counters**: Custom performance counters created if required
- [ ] **Resource Monitoring**: CPU, memory, and disk usage monitoring enabled
- [ ] **Application Metrics**: Application-specific metrics collection configured
- [ ] **Log Monitoring**: Critical log patterns monitored for alerts
- [ ] **Trend Analysis**: Performance trending and analysis capabilities enabled

#### Security Monitoring
- [ ] **Security Events**: Security event monitoring configured
- [ ] **Audit Logging**: Comprehensive audit logging enabled
- [ ] **Intrusion Detection**: Basic intrusion detection measures in place
- [ ] **Log Analysis**: Log analysis scripts available and tested
- [ ] **Incident Response**: Security incident response procedures prepared

### [ ] 11. Backup and Recovery

#### Backup Configuration
- [ ] **Backup Script**: `backup.ps1` script configured and tested
- [ ] **Backup Schedule**: Automated backup schedule configured:
  - [ ] Daily data backups at 2:00 AM
  - [ ] Weekly full backups on Sundays at 1:00 AM
- [ ] **Backup Location**: Secure backup location configured
- [ ] **Backup Retention**: Appropriate backup retention policy (7 days for daily, 4 weeks for weekly)
- [ ] **Backup Verification**: Backup integrity verification enabled

#### Recovery Procedures
- [ ] **Recovery Scripts**: `recovery.ps1` and `system-recovery.ps1` scripts available
- [ ] **Recovery Testing**: Recovery procedures tested successfully
- [ ] **Recovery Documentation**: Recovery procedures documented
- [ ] **Recovery Time Objective**: RTO (2 hours) and RPO (24 hours) objectives validated
- [ ] **Disaster Recovery Plan**: Disaster recovery plan created and accessible

---

## Security Validation Phase

### [ ] 12. Security Testing

#### Vulnerability Assessment
- [ ] **Dependency Scanning**: Go module vulnerability scan completed
- [ ] **Security Scan**: Basic security scan performed (gosec or similar)
- [ ] **Configuration Review**: Security configuration reviewed and validated
- [ ] **Access Control Testing**: Access control mechanisms tested
- [ ] **Input Validation Testing**: Input validation tested with edge cases

#### Penetration Testing (If Required)
- [ ] **External Testing**: External penetration test scheduled or completed
- [ ] **Internal Testing**: Internal security assessment performed
- [ ] **Vulnerability Remediation**: Identified vulnerabilities addressed
- [ ] **Security Documentation**: Security findings documented and addressed
- [ ] **Compliance Verification**: OWASP ASVS Level 3 compliance verified

#### Security Controls Validation
- [ ] **Encryption**: AES-256-GCM encryption working correctly
- [ ] **Authentication**: JWT authentication and session management working
- [ ] **Authorization**: License-based authorization functioning properly
- [ ] **Rate Limiting**: Rate limiting and account lockout working as expected
- [ ] **Security Headers**: All security headers present and correctly configured
- [ ] **Audit Logging**: Security events properly logged and accessible

---

## Production Readiness Phase

### [ ] 13. Performance Validation

#### Load Testing
- [ ] **Concurrent Users**: Application handles expected concurrent user load
- [ ] **API Performance**: API response times within acceptable limits (<500ms p95)
- [ ] **Memory Usage**: Memory usage stable under normal load
- [ ] **Resource Scaling**: Resource usage scales appropriately with load
- [ ] **WebSocket Performance**: WebSocket connections stable under load

#### Stress Testing
- [ ] **High Load**: Application gracefully handles high load conditions
- [ ] **Resource Limits**: Application respects configured resource limits
- [ ] **Error Handling**: Appropriate error responses during stress conditions
- [ ] **Recovery**: Application recovers properly after stress conditions
- [ ] **Performance Degradation**: Graceful performance degradation under extreme load

### [ ] 14. Documentation and Training

#### Technical Documentation
- [ ] **Deployment Guide**: Deployment guide updated and accurate
- [ ] **Security Documentation**: Security documentation complete and current
- [ ] **Operational Procedures**: Operational procedures documented
- [ ] **Troubleshooting Guide**: Troubleshooting procedures documented
- [ ] **Configuration Reference**: Configuration options documented

#### Operations Documentation
- [ ] **Runbooks**: Operational runbooks created for common tasks
- [ ] **Monitoring Procedures**: Monitoring and alerting procedures documented
- [ ] **Incident Response**: Incident response procedures prepared
- [ ] **Emergency Procedures**: Emergency response procedures documented
- [ ] **Contact Information**: Emergency contact information current and accessible

#### Training and Knowledge Transfer
- [ ] **Administrator Training**: System administrators trained on operations
- [ ] **Support Training**: Support staff trained on application functionality
- [ ] **Security Training**: Security procedures communicated to relevant staff
- [ ] **Documentation Access**: All documentation accessible to appropriate personnel
- [ ] **Knowledge Base**: Internal knowledge base updated with deployment information

---

## Go-Live Phase

### [ ] 15. Final Pre-Production Checks

#### System Validation
- [ ] **Final Health Check**: Complete system health check passed
- [ ] **All Services Running**: All required services running and healthy
- [ ] **Database Connectivity**: Database connections working (if applicable)
- [ ] **External Integrations**: All external integrations functioning
- [ ] **Scheduled Tasks**: All scheduled tasks configured and tested

#### Security Final Checks
- [ ] **Security Posture**: Final security posture review completed
- [ ] **Access Controls**: All access controls verified and working
- [ ] **Audit Trail**: Audit trail functioning and complete
- [ ] **Encryption Status**: All encryption working as expected
- [ ] **Security Monitoring**: Security monitoring active and alerting

#### Business Readiness
- [ ] **User Acceptance**: User acceptance testing completed successfully
- [ ] **Business Validation**: Business stakeholders approve go-live
- [ ] **Support Ready**: Support team ready for production issues
- [ ] **Communication Plan**: Go-live communication plan executed
- [ ] **Rollback Plan**: Rollback plan prepared and understood

### [ ] 16. Go-Live Execution

#### Deployment Execution
- [ ] **Change Window**: Deployment executed during approved change window
- [ ] **Deployment Steps**: All deployment steps executed successfully
- [ ] **Verification Tests**: Post-deployment verification tests passed
- [ ] **Monitoring Active**: All monitoring and alerting systems active
- [ ] **Support Available**: Technical support available during go-live

#### Post Go-Live Validation
- [ ] **User Acceptance**: End users can access and use the system
- [ ] **Functionality Check**: All key functionality working as expected
- [ ] **Performance Validation**: System performance within expected parameters
- [ ] **Security Validation**: Security controls functioning properly
- [ ] **Monitoring Confirmation**: All monitoring systems reporting correctly

---

## Post-Production Phase

### [ ] 17. Post-Deployment Activities

#### Immediate Post-Production (0-24 hours)
- [ ] **System Monitoring**: Continuous monitoring for first 24 hours
- [ ] **Performance Tracking**: Performance metrics tracked and analyzed
- [ ] **Error Monitoring**: Error rates monitored and investigated
- [ ] **User Feedback**: User feedback collected and addressed
- [ ] **Issue Resolution**: Any issues identified and resolved quickly

#### Short-term Post-Production (1-7 days)
- [ ] **Stability Assessment**: System stability assessed over first week
- [ ] **Performance Baseline**: Performance baseline established
- [ ] **Optimization**: Initial performance optimizations implemented
- [ ] **Training Issues**: Training gaps identified and addressed
- [ ] **Documentation Updates**: Documentation updated based on deployment experience

#### Long-term Post-Production (1-4 weeks)
- [ ] **Performance Review**: Comprehensive performance review completed
- [ ] **Security Review**: Post-deployment security review performed
- [ ] **Process Improvement**: Deployment process improvements identified
- [ ] **Lessons Learned**: Lessons learned documented and shared
- [ ] **Future Planning**: Future enhancement planning initiated

---

## Sign-Off Section

### Deployment Team Sign-Off

| Role | Name | Signature | Date |
|------|------|-----------|------|
| **System Administrator** | | | |
| **Security Engineer** | | | |
| **DevOps Engineer** | | | |
| **Application Owner** | | | |
| **Business Stakeholder** | | | |

### Quality Gates

- [ ] **QG1**: All infrastructure requirements met and verified
- [ ] **QG2**: All security requirements implemented and tested
- [ ] **QG3**: All application functionality tested and validated
- [ ] **QG4**: All monitoring and alerting systems operational
- [ ] **QG5**: All documentation complete and accessible
- [ ] **QG6**: All stakeholders trained and ready
- [ ] **QG7**: Go-live approval received from all stakeholders

### Final Approval

**Production Deployment Approved**: ⬜ Yes ⬜ No

**Approved By**: _________________________________ **Date**: _____________

**Title**: _________________________________

**Notes**:
```
_________________________________________________________________
_________________________________________________________________
_________________________________________________________________
```

---

## Emergency Contacts

### Technical Support
- **Primary**: _________________________________ (Phone: _____________)
- **Secondary**: _______________________________ (Phone: _____________)
- **On-Call**: _________________________________ (Phone: _____________)

### Business Contacts
- **Business Owner**: ___________________________ (Phone: _____________)
- **Project Manager**: __________________________ (Phone: _____________)
- **Executive Sponsor**: ________________________ (Phone: _____________)

### Vendor Support
- **Google Cloud Support**: https://cloud.google.com/support
- **Microsoft Support**: https://support.microsoft.com
- **Emergency Vendor Contact**: ________________________ (Phone: _____________)

---

**Document Version**: 2.0  
**Last Updated**: 2025-01-31  
**Next Review**: 2025-04-30  
**Document Owner**: DevOps Team