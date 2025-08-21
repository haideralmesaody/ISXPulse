# 🎯 Scratch Card License System - Complete Implementation Plan

## Executive Summary
Transform the current license system into a true one-time activation scratch card system while maintaining Google Sheets as the backend. This plan details every step, who implements it, and how.

---

## 📋 Table of Contents
1. [Prerequisites & Preparation](#prerequisites--preparation)
2. [Phase 1: Google Infrastructure Setup](#phase-1-google-infrastructure-setup)
3. [Phase 2: Backend Enhancement](#phase-2-backend-enhancement)
4. [Phase 3: Frontend Updates](#phase-3-frontend-updates)
5. [Phase 4: Security Implementation](#phase-4-security-implementation)
6. [Phase 5: Testing & Validation](#phase-5-testing--validation)
7. [Phase 6: Migration & Deployment](#phase-6-migration--deployment)
8. [Phase 7: Monitoring & Maintenance](#phase-7-monitoring--maintenance)

---

## Prerequisites & Preparation

### Required Access & Tools
- [ ] Google Account with access to current Sheets
- [ ] Google Apps Script Editor access
- [ ] Google Cloud Console (for Apps Script API)
- [ ] Postman or similar for API testing
- [ ] Backup of current license sheet

### Initial Backup (MANUAL - 30 minutes)
```bash
# 1. Export current Google Sheet
#    - Open https://docs.google.com/spreadsheets/d/1l4jJNNqHZNomjp3wpkL-txDfCjsRr19aJZOZqPHJ6lc
#    - File → Download → Microsoft Excel (.xlsx)
#    - Save as: ISX_Licenses_Backup_[DATE].xlsx

# 2. Backup current license.dat files
#    - Copy from all deployed instances
#    - Store in secure location
```

---

## Phase 1: Google Infrastructure Setup (MANUAL - Outside Project)

### 1.1 Create Enhanced Google Sheets Structure (MANUAL - 1 hour)

**Performer**: You (Manual)
**Location**: Google Sheets Web Interface

1. **Open the existing sheet**:
   ```
   https://docs.google.com/spreadsheets/d/1l4jJNNqHZNomjp3wpkL-txDfCjsRr19aJZOZqPHJ6lc
   ```

2. **Rename current "Licenses" sheet to "Licenses_Old"** (backup)

3. **Create new "Licenses" sheet with columns**:
   ```
   A: Code (Text, UNIQUE constraint)
   B: Duration (Text: "1m", "3m", "6m", "1y")
   C: Status (Text: "Available", "Activated", "Expired", "Revoked")
   D: ActivationDate (DateTime)
   E: ActivationIP (Text)
   F: DeviceFingerprint (Text)
   G: Email (Text)
   H: ExpiryDate (DateTime)
   I: ActivationID (Text, UNIQUE)
   J: LastChecked (DateTime)
   K: CheckCount (Number)
   L: Notes (Text)
   M: CreatedDate (DateTime)
   N: BatchID (Text)
   ```

4. **Create "ActivationAttempts" sheet**:
   ```
   A: Timestamp (DateTime)
   B: Code (Text)
   C: IP (Text)
   D: Success (Boolean)
   E: Error (Text)
   F: DeviceFingerprint (Text)
   G: UserAgent (Text)
   ```

5. **Create "Blacklist" sheet**:
   ```
   A: Identifier (Text)
   B: Type (Text: "IP", "Device", "Email")
   C: Reason (Text)
   D: BlockedDate (DateTime)
   E: BlockedBy (Text)
   F: ExpiryDate (DateTime, optional)
   ```

6. **Create "AuditLog" sheet**:
   ```
   A: Timestamp (DateTime)
   B: Action (Text)
   C: LicenseCode (Text)
   D: PerformedBy (Text)
   E: Details (Text)
   F: Result (Text)
   ```

### 1.2 Google Apps Script Implementation (MANUAL - 3 hours)

**Performer**: You (Manual)
**Location**: Google Apps Script Editor

1. **Open Apps Script Editor**:
   - From Google Sheets: Extensions → Apps Script
   - Name project: "ISX_License_Manager"

2. **Create main script file** (`Code.gs`):

```javascript
// ============================================
// ISX License Manager - Atomic Operations
// ============================================

const SHEET_ID = '1l4jJNNqHZNomjp3wpkL-txDfCjsRr19aJZOZqPHJ6lc';
const MAX_ATTEMPTS_PER_HOUR = 10;
const BLOCK_DURATION_HOURS = 24;

// Main activation endpoint
function doPost(e) {
  const lock = LockService.getScriptLock();
  
  try {
    // Acquire lock for atomic operation (wait max 10 seconds)
    lock.waitLock(10000);
    
    // Parse request
    const request = JSON.parse(e.postData.contents);
    const action = request.action;
    
    // Route to appropriate handler
    switch(action) {
      case 'activate':
        return handleActivation(request);
      case 'validate':
        return handleValidation(request);
      case 'revoke':
        return handleRevocation(request);
      case 'checkStatus':
        return handleStatusCheck(request);
      default:
        return createResponse(false, 'Unknown action', null);
    }
    
  } catch (error) {
    console.error('Error in doPost:', error);
    return createResponse(false, 'Server error: ' + error.toString(), null);
  } finally {
    lock.releaseLock();
  }
}

// Atomic activation handler
function handleActivation(request) {
  const code = request.code;
  const deviceInfo = request.deviceInfo || {};
  
  // Check blacklist first
  if (isBlacklisted(deviceInfo.ip) || isBlacklisted(deviceInfo.fingerprint)) {
    logActivationAttempt(code, deviceInfo, false, 'Blacklisted');
    return createResponse(false, 'Access denied', null);
  }
  
  // Check rate limiting
  if (!checkRateLimit(deviceInfo.ip)) {
    addToBlacklist(deviceInfo.ip, 'IP', 'Rate limit exceeded');
    logActivationAttempt(code, deviceInfo, false, 'Rate limited');
    return createResponse(false, 'Too many attempts. Try again later.', null);
  }
  
  // Get licenses sheet
  const sheet = SpreadsheetApp.openById(SHEET_ID).getSheetByName('Licenses');
  const dataRange = sheet.getDataRange();
  const values = dataRange.getValues();
  
  // Find the license
  for (let i = 1; i < values.length; i++) {
    if (values[i][0] === code) { // Column A: Code
      
      // Check current status
      const currentStatus = values[i][2]; // Column C: Status
      
      if (currentStatus !== 'Available') {
        logActivationAttempt(code, deviceInfo, false, 'Already activated');
        
        // Return activation details if already activated
        return createResponse(false, 'License already activated', {
          activationDate: values[i][3],
          status: currentStatus
        });
      }
      
      // Generate unique activation ID
      const activationId = Utilities.getUuid();
      const now = new Date();
      const duration = values[i][1]; // Column B: Duration
      const expiryDate = calculateExpiryDate(duration);
      
      // Update the license row atomically
      const updates = [
        [3, 'Activated'],                    // C: Status
        [4, now],                            // D: ActivationDate
        [5, deviceInfo.ip || ''],           // E: ActivationIP
        [6, deviceInfo.fingerprint || ''],   // F: DeviceFingerprint
        [7, deviceInfo.email || ''],        // G: Email
        [8, expiryDate],                     // H: ExpiryDate
        [9, activationId],                   // I: ActivationID
        [10, now],                           // J: LastChecked
        [11, 1]                              // K: CheckCount
      ];
      
      // Apply all updates
      updates.forEach(([col, value]) => {
        sheet.getRange(i + 1, col).setValue(value);
      });
      
      // Log successful activation
      logActivationAttempt(code, deviceInfo, true, 'Success');
      logAudit('ACTIVATION', code, deviceInfo.ip, 'License activated successfully');
      
      return createResponse(true, 'License activated successfully', {
        activationId: activationId,
        expiryDate: expiryDate,
        duration: duration
      });
    }
  }
  
  // License not found
  logActivationAttempt(code, deviceInfo, false, 'Invalid code');
  return createResponse(false, 'Invalid license code', null);
}

// Validation handler
function handleValidation(request) {
  const code = request.code;
  const activationId = request.activationId;
  const deviceFingerprint = request.deviceFingerprint;
  
  const sheet = SpreadsheetApp.openById(SHEET_ID).getSheetByName('Licenses');
  const dataRange = sheet.getDataRange();
  const values = dataRange.getValues();
  
  for (let i = 1; i < values.length; i++) {
    if (values[i][0] === code && values[i][8] === activationId) {
      const status = values[i][2];
      const expiryDate = new Date(values[i][7]);
      const storedFingerprint = values[i][5];
      
      // Check expiry
      if (new Date() > expiryDate) {
        // Update status to expired
        sheet.getRange(i + 1, 3).setValue('Expired');
        return createResponse(false, 'License expired', {
          expiryDate: expiryDate
        });
      }
      
      // Check device binding (optional)
      if (storedFingerprint && storedFingerprint !== deviceFingerprint) {
        logAudit('VALIDATION_MISMATCH', code, '', 'Device fingerprint mismatch');
        // Decide: strict (return false) or lenient (log only)
        // return createResponse(false, 'License bound to different device', null);
      }
      
      // Update last checked
      sheet.getRange(i + 1, 10).setValue(new Date()); // J: LastChecked
      sheet.getRange(i + 1, 11).setValue(values[i][10] + 1); // K: CheckCount++
      
      return createResponse(true, 'License valid', {
        status: status,
        expiryDate: expiryDate,
        checksRemaining: Math.max(0, 1000 - values[i][10])
      });
    }
  }
  
  return createResponse(false, 'Invalid license or activation ID', null);
}

// Helper Functions
function calculateExpiryDate(duration) {
  const now = new Date();
  let months = 1;
  
  switch(duration) {
    case '1m': months = 1; break;
    case '3m': months = 3; break;
    case '6m': months = 6; break;
    case '1y': months = 12; break;
    default: months = 1;
  }
  
  // Add months
  now.setMonth(now.getMonth() + months);
  
  // Set to midnight of next day
  now.setDate(now.getDate() + 1);
  now.setHours(0, 0, 0, 0);
  
  return now;
}

function checkRateLimit(ip) {
  if (!ip) return true;
  
  const sheet = SpreadsheetApp.openById(SHEET_ID).getSheetByName('ActivationAttempts');
  const now = new Date();
  const oneHourAgo = new Date(now.getTime() - 60 * 60 * 1000);
  
  const data = sheet.getDataRange().getValues();
  let recentAttempts = 0;
  
  for (let i = 1; i < data.length; i++) {
    if (data[i][2] === ip && new Date(data[i][0]) > oneHourAgo) {
      recentAttempts++;
    }
  }
  
  return recentAttempts < MAX_ATTEMPTS_PER_HOUR;
}

function isBlacklisted(identifier) {
  if (!identifier) return false;
  
  const sheet = SpreadsheetApp.openById(SHEET_ID).getSheetByName('Blacklist');
  const data = sheet.getDataRange().getValues();
  
  for (let i = 1; i < data.length; i++) {
    if (data[i][0] === identifier) {
      // Check if expired
      if (data[i][5] && new Date(data[i][5]) < new Date()) {
        continue;
      }
      return true;
    }
  }
  
  return false;
}

function addToBlacklist(identifier, type, reason) {
  const sheet = SpreadsheetApp.openById(SHEET_ID).getSheetByName('Blacklist');
  const now = new Date();
  const expiryDate = new Date(now.getTime() + BLOCK_DURATION_HOURS * 60 * 60 * 1000);
  
  sheet.appendRow([identifier, type, reason, now, 'System', expiryDate]);
}

function logActivationAttempt(code, deviceInfo, success, error) {
  const sheet = SpreadsheetApp.openById(SHEET_ID).getSheetByName('ActivationAttempts');
  sheet.appendRow([
    new Date(),
    code,
    deviceInfo.ip || '',
    success,
    error || '',
    deviceInfo.fingerprint || '',
    deviceInfo.userAgent || ''
  ]);
}

function logAudit(action, code, performer, details) {
  const sheet = SpreadsheetApp.openById(SHEET_ID).getSheetByName('AuditLog');
  sheet.appendRow([
    new Date(),
    action,
    code,
    performer,
    details,
    'Success'
  ]);
}

function createResponse(success, message, data) {
  return ContentService.createTextOutput(
    JSON.stringify({
      success: success,
      message: message,
      data: data,
      timestamp: new Date().toISOString()
    })
  ).setMimeType(ContentService.MimeType.JSON);
}

// Test endpoint for GET requests
function doGet(e) {
  return ContentService.createTextOutput(
    JSON.stringify({
      status: 'OK',
      message: 'ISX License Manager API is running',
      version: '1.0.0'
    })
  ).setMimeType(ContentService.MimeType.JSON);
}
```

3. **Deploy as Web App**:
   - Click "Deploy" → "New Deployment"
   - Type: "Web app"
   - Description: "ISX License Manager API v1"
   - Execute as: "Me"
   - Who has access: "Anyone"
   - Click "Deploy"
   - **SAVE THE WEB APP URL** (format: https://script.google.com/macros/s/[SCRIPT_ID]/exec)

4. **Set up time-based triggers** (optional):
   - Triggers → Add Trigger
   - Function: `cleanupExpiredBlacklist`
   - Time-based → Hour timer → Every hour

### 1.3 Google Cloud API Setup (MANUAL - 30 minutes)

1. **Enable Google Sheets API**:
   - Go to https://console.cloud.google.com
   - Select your project
   - APIs & Services → Enable APIs
   - Search "Google Sheets API" → Enable

2. **Update Service Account Permissions**:
   - IAM & Admin → Service Accounts
   - Find your existing service account
   - Ensure it has "Sheets API" access

---

## Phase 2: Backend Enhancement (AGENT: license-system-engineer)

### 2.1 Core License Manager Updates

**Performer**: `license-system-engineer` agent
**Files to modify**: 
- `api/internal/license/manager.go`
- `api/internal/license/types.go`
- `api/internal/license/validation.go`

**Implementation Instructions for Agent**:

```markdown
Task: Enhance license manager for scratch card system

1. Add new types in api/internal/license/types.go:
   - DeviceInfo struct with IP, Fingerprint, Email, UserAgent
   - ActivationResult struct with ActivationID, ExpiryDate
   - Add ActivationID field to LicenseInfo struct

2. Update api/internal/license/manager.go:
   - Add Google Apps Script URL as constant
   - Modify performActivation() to call Apps Script endpoint
   - Add device fingerprinting function
   - Update validation to check ActivationID
   - Add retry logic with exponential backoff

3. Implement device fingerprinting:
   - Combine MAC address, hostname, CPU info
   - Create SHA256 hash for privacy
   - Store in license.dat

4. Add atomic operations support:
   - Replace direct Sheets API calls with Apps Script calls
   - Implement proper error handling for network failures
   - Add timeout handling (30 seconds max)
```

### 2.2 Security Enhancements

**Performer**: `security-auditor` agent
**Files to modify**:
- `api/internal/security/fingerprint.go` (new)
- `api/internal/security/integrity.go`
- `api/internal/license/security.go`

**Implementation Instructions for Agent**:

```markdown
Task: Implement security enhancements for scratch card system

1. Create api/internal/security/fingerprint.go:
   - Implement GetMACAddress() using net package
   - Implement GetCPUID() with OS-specific implementations
   - Create GenerateFingerprint() combining all factors
   - Add FingerprintCache for performance

2. Update license security:
   - Add HMAC signatures to license.dat
   - Implement tamper detection
   - Add license file encryption option
   - Implement secure deletion of old licenses

3. Add rate limiting enhancements:
   - Track attempts by IP and fingerprint
   - Implement exponential backoff
   - Add automatic blacklisting after threshold
```

### 2.3 Code Generation Tool

**Performer**: `license-system-engineer` agent
**Files to create**:
- `tools/license-generator/main.go`
- `tools/license-generator/README.md`

**Implementation Instructions for Agent**:

```markdown
Task: Create scratch card code generation tool

1. Create tools/license-generator/main.go:
   - Command-line tool for batch code generation
   - Format: ISX-XXXX-XXXX-XXXX (alphanumeric, no confusing chars)
   - Use crypto/rand for security
   - Batch upload to Google Sheets
   - Export to CSV for printing

2. Features to implement:
   - Generate N codes with specified duration
   - Check uniqueness against existing codes
   - Add batch ID for tracking
   - Generate QR codes (optional)
   - Export in multiple formats (CSV, JSON, PDF)

3. Usage:
   ./license-generator -count 100 -duration 1m -batch "BATCH001"
```

---

## Phase 3: Frontend Updates (AGENT: frontend-modernizer)

### 3.1 Enhanced Activation UI

**Performer**: `frontend-modernizer` agent
**Files to modify**:
- `web/components/license/LicenseActivationFormComponent.tsx`
- `web/lib/api.ts`
- `web/types/index.ts`

**Implementation Instructions for Agent**:

```markdown
Task: Update frontend for scratch card activation

1. Update LicenseActivationFormComponent.tsx:
   - Add scratch card input format (XXX-XXXX-XXXX-XXXX)
   - Auto-format input with dashes
   - Show visual feedback during activation
   - Display remaining days clearly
   - Add "reveal code" animation for scratch card feel

2. Update API client:
   - Add device fingerprinting (using browser APIs)
   - Include activation ID in requests
   - Add retry logic for network failures
   - Store activation ID in localStorage as backup

3. Add new components:
   - ScratchCardInput component with formatting
   - LicenseStatusCard showing detailed info
   - ActivationSuccessModal with confetti animation
```

### 3.2 User Experience Improvements

**Performer**: `frontend-modernizer` agent
**Files to create/modify**:
- `web/components/license/ScratchCard.tsx` (new)
- `web/components/license/LicenseStatus.tsx` (new)

**Implementation Instructions for Agent**:

```markdown
Task: Create engaging scratch card UX

1. Create ScratchCard component:
   - Visual scratch card representation
   - Scratch-off animation effect
   - Code reveal animation
   - Copy-to-clipboard functionality

2. Create comprehensive status display:
   - Days remaining with visual indicator
   - Activation history
   - Device information
   - Quick actions (extend, transfer, support)

3. Add helpful error messages:
   - "Code already used on [date]"
   - "Invalid code format - check your typing"
   - "Network error - please try again"
   - "Rate limited - wait X minutes"
```

---

## Phase 4: Security Implementation (AGENT: security-auditor)

### 4.1 Security Audit & Hardening

**Performer**: `security-auditor` agent
**Tasks**:

```markdown
Task: Security audit and hardening for scratch card system

1. Audit current implementation:
   - Check for timing attacks in validation
   - Verify HMAC implementation
   - Test rate limiting effectiveness
   - Check for SQL injection in Apps Script

2. Implement additional security:
   - Add request signing for API calls
   - Implement certificate pinning for Google APIs
   - Add anomaly detection for suspicious patterns
   - Create security event logging

3. Create security tests:
   - Brute force attempt simulation
   - Replay attack tests
   - Tampered license file tests
   - Rate limiting bypass attempts
```

---

## Phase 5: Testing & Validation (AGENT: test-architect)

### 5.1 Comprehensive Test Suite

**Performer**: `test-architect` agent
**Files to create**:
- `api/internal/license/activation_test.go`
- `api/internal/license/validation_test.go`
- `api/internal/security/fingerprint_test.go`
- `tools/license-generator/generator_test.go`

**Implementation Instructions for Agent**:

```markdown
Task: Create comprehensive test suite for scratch card system

1. Unit tests for activation:
   - Test one-time activation enforcement
   - Test duplicate activation prevention
   - Test invalid code handling
   - Test rate limiting
   - Test device fingerprinting

2. Integration tests:
   - Test full activation flow
   - Test Apps Script integration
   - Test network failure handling
   - Test cache behavior

3. Security tests:
   - Test brute force protection
   - Test blacklisting
   - Test tamper detection
   - Test timing attack resistance

4. Performance tests:
   - Benchmark activation speed
   - Test concurrent activations
   - Load test Apps Script endpoint
```

### 5.2 End-to-End Testing

**Performer**: `integration-test-orchestrator` agent

```markdown
Task: Create E2E tests for complete activation flow

1. Test scenarios:
   - Fresh installation → Activation → Validation
   - Multiple activation attempts
   - Device change scenarios
   - Network failure recovery
   - License expiry flow

2. Create test data:
   - Generate test licenses in sandbox sheet
   - Create test device fingerprints
   - Simulate various network conditions
```

---

## Phase 6: Migration & Deployment

### 6.1 Data Migration (MANUAL - 2 hours)

**Performer**: You (Manual)
**Steps**:

1. **Export existing licenses**:
   ```javascript
   // Google Apps Script to migrate data
   function migrateLicenses() {
     const oldSheet = SpreadsheetApp.openById(SHEET_ID).getSheetByName('Licenses_Old');
     const newSheet = SpreadsheetApp.openById(SHEET_ID).getSheetByName('Licenses');
     
     const oldData = oldSheet.getDataRange().getValues();
     
     for (let i = 1; i < oldData.length; i++) {
       const oldRow = oldData[i];
       const newRow = [
         oldRow[0], // Code
         oldRow[1], // Duration
         oldRow[3], // Status
         oldRow[5], // ActivationDate
         '',        // ActivationIP (new)
         '',        // DeviceFingerprint (new)
         oldRow[2], // Email
         oldRow[2], // ExpiryDate
         Utilities.getUuid(), // Generate ActivationID
         oldRow[6], // LastChecked
         0,         // CheckCount (new)
         'Migrated from old system'
       ];
       
       newSheet.appendRow(newRow);
     }
   }
   ```

2. **Generate new scratch cards**:
   ```bash
   cd tools/license-generator
   go run main.go -count 100 -duration 1m -batch "LAUNCH001"
   ```

### 6.2 Deployment Steps (AGENT: deployment-orchestrator)

**Performer**: `deployment-orchestrator` agent

```markdown
Task: Deploy enhanced scratch card system

1. Pre-deployment checklist:
   - Backup current system
   - Test Apps Script endpoint
   - Verify Google Sheets structure
   - Test activation flow in staging

2. Build and package:
   - Update version numbers
   - Build with embedded Apps Script URL
   - Include fingerprinting libraries
   - Test binary on target platforms

3. Deployment sequence:
   - Deploy Apps Script (already done)
   - Update backend API
   - Update frontend
   - Run smoke tests
   - Monitor error rates

4. Rollback plan:
   - Keep old Apps Script version
   - Maintain backward compatibility
   - Have database restore ready
   - Document rollback procedures
```

### 6.3 Configuration Updates (MANUAL)

**Update these files**:

1. `.env`:
   ```env
   GOOGLE_APPS_SCRIPT_URL=https://script.google.com/macros/s/[YOUR_SCRIPT_ID]/exec
   ENABLE_DEVICE_FINGERPRINT=true
   ENABLE_SCRATCH_CARD_MODE=true
   ```

2. `config/production.json`:
   ```json
   {
     "license": {
       "mode": "scratch_card",
       "apps_script_url": "...",
       "enable_fingerprint": true,
       "max_attempts": 10
     }
   }
   ```

---

## Phase 7: Monitoring & Maintenance

### 7.1 Monitoring Setup (AGENT: observability-engineer)

**Performer**: `observability-engineer` agent

```markdown
Task: Implement monitoring for scratch card system

1. Add metrics:
   - Activation success/failure rate
   - Average activation time
   - Validation cache hit rate
   - Apps Script response time
   - Blacklist size growth

2. Add alerts:
   - High failure rate (>10%)
   - Slow activation (>5 seconds)
   - Apps Script errors
   - Unusual activation patterns

3. Create dashboards:
   - Real-time activation status
   - License inventory levels
   - Geographic distribution
   - Device type breakdown
```

### 7.2 Documentation (AGENT: documentation-enforcer)

**Performer**: `documentation-enforcer` agent

```markdown
Task: Create comprehensive documentation

1. Create docs/SCRATCH_CARD_SYSTEM.md:
   - Architecture overview
   - Activation flow diagram
   - Security features
   - Troubleshooting guide

2. Update README.md:
   - Add scratch card section
   - Update activation instructions
   - Add code generation guide

3. Create admin guide:
   - How to generate codes
   - How to monitor usage
   - How to handle support issues
   - Blacklist management
```

---

## Timeline & Milestones

### Week 1: Foundation
- Day 1-2: Google Sheets & Apps Script setup (MANUAL)
- Day 3-4: Backend core updates (license-system-engineer)
- Day 5: Security implementation (security-auditor)

### Week 2: Implementation
- Day 6-7: Frontend updates (frontend-modernizer)
- Day 8-9: Testing suite (test-architect)
- Day 10: Integration testing (integration-test-orchestrator)

### Week 3: Deployment
- Day 11: Data migration (MANUAL)
- Day 12: Staging deployment (deployment-orchestrator)
- Day 13: Production deployment
- Day 14: Monitoring & documentation
- Day 15: Buffer & fixes

---

## Success Criteria

### Functional Requirements
- [ ] One-time activation enforced
- [ ] No race conditions in activation
- [ ] Device fingerprinting working
- [ ] Rate limiting active
- [ ] Blacklisting functional

### Performance Requirements
- [ ] Activation < 3 seconds
- [ ] Validation < 100ms (cached)
- [ ] 99.9% uptime
- [ ] Support 1000 concurrent users

### Security Requirements
- [ ] No duplicate activations possible
- [ ] Brute force protection active
- [ ] Tamper detection working
- [ ] Audit logging complete

---

## Risk Mitigation

### Identified Risks
1. **Google Apps Script quota limits**
   - Mitigation: Implement caching, rate limiting
   - Backup: Consider Google Cloud Functions

2. **Network dependency**
   - Mitigation: Aggressive caching, offline grace period
   - Backup: Local validation with periodic sync

3. **Device fingerprint changes**
   - Mitigation: Grace period, support override
   - Backup: Email-based recovery

4. **Data loss**
   - Mitigation: Regular backups, audit logs
   - Backup: Recovery procedures

---

## Support & Maintenance

### Day 1 Support Plan
- Monitor activation success rate
- Watch for Apps Script errors
- Check blacklist growth
- Review failed activation logs

### Ongoing Maintenance
- Weekly: Review activation metrics
- Monthly: Clean old attempt logs
- Quarterly: Security audit
- Yearly: License cleanup

---

## Appendix A: Code Templates

### A.1 Batch Code Generation Script
```go
// tools/license-generator/templates/batch.go
func GenerateBatch(count int, duration string) []string {
    // Implementation provided by license-system-engineer
}
```

### A.2 Device Fingerprinting
```go
// api/internal/security/fingerprint.go
func GenerateFingerprint() string {
    // Implementation provided by security-auditor
}
```

### A.3 Frontend Scratch Card Component
```tsx
// web/components/license/ScratchCard.tsx
export function ScratchCard({ onReveal }) {
    // Implementation provided by frontend-modernizer
}
```

---

## Appendix B: Testing Checklist

### Pre-Launch Testing
- [ ] Generate 10 test codes
- [ ] Test activation on 3 different devices
- [ ] Test duplicate activation prevention
- [ ] Test rate limiting (11 attempts)
- [ ] Test device change scenario
- [ ] Test expiry flow
- [ ] Test revocation
- [ ] Test offline validation
- [ ] Load test with 100 concurrent activations
- [ ] Security scan for vulnerabilities

---

## Appendix C: Emergency Procedures

### If Apps Script fails:
1. Check Google Cloud status
2. Verify script deployment
3. Check quota limits
4. Roll back to previous version
5. Enable emergency offline mode

### If mass activation attempts detected:
1. Check blacklist effectiveness
2. Temporarily increase rate limits
3. Block suspicious IP ranges
4. Review recent activation patterns
5. Consider CAPTCHA implementation

---

## Final Notes

This implementation plan transforms your license system into a true scratch card system while maintaining Google Sheets as the backend. The atomic operations via Google Apps Script ensure one-time activation, while the comprehensive security measures protect against abuse.

**Total Estimated Time**: 3 weeks
**Complexity**: Medium-High
**Risk Level**: Low (with proper testing)

**Key Success Factors**:
1. Proper Apps Script implementation
2. Thorough testing
3. Gradual rollout
4. Monitoring from day 1

---

*Document Version: 1.0*
*Created: [Current Date]*
*Last Updated: [Current Date]*
*Status: READY FOR IMPLEMENTATION*