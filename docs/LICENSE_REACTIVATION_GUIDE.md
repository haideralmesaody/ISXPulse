# License Reactivation Guide

## Overview

The ISX Pulse license system now supports smart device recognition and automatic reactivation for users reinstalling the application on the same device. This guide explains the feature's implementation, configuration, and usage.

## Table of Contents

1. [Feature Overview](#feature-overview)
2. [How It Works](#how-it-works)
3. [Technical Implementation](#technical-implementation)
4. [Configuration](#configuration)
5. [Security Considerations](#security-considerations)
6. [Troubleshooting](#troubleshooting)
7. [API Reference](#api-reference)

## Feature Overview

### Problem Solved
Previously, users who reinstalled ISX Pulse on the same machine needed to contact support to reactivate their license. This created friction and support overhead.

### Solution
Smart device recognition with fuzzy fingerprint matching allows automatic reactivation when:
- Same device is detected (80% similarity threshold)
- License is still valid (not expired)
- Reactivation limit not exceeded (5 per 30 days)

### Benefits
- **User Experience**: Seamless reactivation without support intervention
- **Security**: Prevents unauthorized device transfers while allowing legitimate reinstalls
- **Support Efficiency**: Reduces support tickets for reinstallation issues

## How It Works

### 1. Device Fingerprinting
When a user activates a license, the system captures:
- Browser type and version
- Operating system and version
- Screen resolution
- Timezone
- Hardware concurrency
- WebGL renderer information
- Canvas fingerprint

### 2. Fingerprint Comparison
During reactivation attempts:
1. Generate current device fingerprint
2. Compare with stored fingerprint using Jaccard similarity
3. Calculate similarity score (0-100%)
4. Allow reactivation if score ≥ 80%

### 3. Reactivation Limits
To prevent abuse:
- Maximum 5 reactivations per license per 30 days
- Each reactivation is logged with timestamp
- Automatic cleanup of old reactivation records

## Technical Implementation

### Frontend Components

#### Device Fingerprint Generation
```typescript
// web/lib/utils/device-fingerprint.ts
export async function generateDeviceFingerprint(): Promise<DeviceFingerprint> {
  // Collects browser, OS, hardware information
  // Generates SHA-256 hash for consistency
  // Returns comprehensive device profile
}
```

#### License Activation Flow
```typescript
// web/app/license/license-content.tsx
const handleActivation = async (data: LicenseActivationForm) => {
  // Generate device fingerprint
  // Submit activation request
  // Handle reactivation success specially
  // Display appropriate user feedback
}
```

### Backend Implementation

#### Reactivation Handler
```go
// api/internal/transport/http/license_handler.go
func (h *Handler) ActivateLicense(w http.ResponseWriter, r *http.Request) {
    // Validate license key
    // Check Google Sheets for existing activation
    // Compare device fingerprints
    // Allow reactivation if same device
}
```

#### Error Handling
```go
// api/internal/errors/license_errors.go
var (
    ErrLicenseReactivated = errors.New("license successfully reactivated")
    ErrReactivationLimitExceeded = errors.New("reactivation limit exceeded")
)
```

### Google Sheets Integration

#### Fuzzy Matching Algorithm
```javascript
// GOOGLE_SHEETS_FIX.js
function calculateFingerprintSimilarity(stored, current) {
  // Uses Jaccard similarity with character bigrams
  // Returns score 0-100
  // 80% threshold for same-device detection
}
```

#### Reactivation Tracking
```javascript
function updateReactivationTracking(sheet, rowIndex) {
  // Increments reactivation count
  // Records timestamp
  // Enforces 30-day rolling window
  // Maximum 5 reactivations per period
}
```

## Configuration

### Environment Variables
```bash
# Enable device fingerprinting
ENABLE_DEVICE_FINGERPRINT=true

# Reactivation settings
LICENSE_REACTIVATION_ENABLED=true
LICENSE_REACTIVATION_THRESHOLD=80  # Similarity percentage
LICENSE_REACTIVATION_LIMIT=5       # Max per 30 days
LICENSE_REACTIVATION_WINDOW=30     # Days
```

### Google Sheets Structure
The license tracking sheet requires these columns:
- **Column N**: Reactivation Count
- **Column O**: Reactivation History (JSON array)
- **Column P**: Last Reactivation Date

### Frontend Configuration
```typescript
// web/lib/config.ts
export const LICENSE_CONFIG = {
  enableFingerprinting: true,
  reactivationEnabled: true,
  similarityThreshold: 0.8,
  rateLimit: {
    maxAttempts: 10,
    windowMs: 300000 // 5 minutes
  }
}
```

## Security Considerations

### 1. Fingerprint Privacy
- No personally identifiable information collected
- Fingerprints are hashed before storage
- Data stored only in Google Sheets (not transmitted elsewhere)

### 2. Similarity Threshold
- 80% threshold balances security and usability
- Accounts for minor browser/OS updates
- Prevents completely different devices from reactivating

### 3. Rate Limiting
- Client-side: 10 attempts per 5 minutes
- Server-side: Validates against Google Sheets
- Prevents brute force attempts

### 4. Audit Trail
- All reactivations logged with timestamps
- Device information recorded for security review
- Support can review reactivation history

## Troubleshooting

### Common Issues

#### "Device Not Recognized" Error
**Cause**: Similarity score below 80% threshold
**Solutions**:
1. Check if major browser update occurred
2. Verify same physical device is being used
3. Contact support if legitimate reinstall

#### "Reactivation Limit Exceeded"
**Cause**: More than 5 reactivations in 30 days
**Solutions**:
1. Wait for older reactivations to expire
2. Contact support for manual review
3. Consider if license is being shared improperly

#### "Too Many Attempts" Error
**Cause**: Rate limiting triggered
**Solutions**:
1. Wait 5 minutes before retrying
2. Clear browser cache and cookies
3. Restart browser if issue persists

### Debug Mode

Enable debug logging in browser console:
```javascript
localStorage.setItem('DEBUG_LICENSE', 'true')
```

View device fingerprint:
```javascript
const fp = await generateDeviceFingerprint()
console.log('Device Fingerprint:', fp)
```

## API Reference

### License Activation Endpoint

**POST** `/api/v1/license/activate`

#### Request Body
```json
{
  "license_key": "ISX1M02LYE1F9QJHR9D7Z",
  "device_fingerprint": {
    "browser": "Chrome",
    "browserVersion": "120.0.0",
    "os": "Windows",
    "osVersion": "Windows 10/11",
    "platform": "Win32",
    "screenResolution": "1920x1080x24",
    "timezone": "Asia/Baghdad",
    "language": "en-US",
    "userAgent": "Mozilla/5.0...",
    "hash": "a1b2c3d4e5f6...",
    "timestamp": "2024-01-19T08:00:00Z"
  }
}
```

#### Success Response (New Activation)
```json
{
  "status": "valid",
  "message": "License activated successfully",
  "activation_id": "ACT-123456",
  "expiry_date": "2024-02-19T23:59:59Z",
  "days_remaining": 30,
  "features": ["professional", "real-time", "analytics"]
}
```

#### Success Response (Reactivation)
```json
{
  "status": "reactivated",
  "message": "License reactivated on same device",
  "activation_id": "ACT-123456",
  "expiry_date": "2024-02-19T23:59:59Z",
  "days_remaining": 30,
  "similarity_score": 92,
  "reactivation_count": 2,
  "reactivation_limit": 5
}
```

#### Error Responses

**409 Conflict** - Already activated on different device
```json
{
  "type": "/errors/license-already-activated",
  "title": "License Already Activated",
  "status": 409,
  "detail": "This license has been activated on a different device",
  "similarity_score": 45
}
```

**429 Too Many Requests** - Reactivation limit exceeded
```json
{
  "type": "/errors/reactivation-limit-exceeded",
  "title": "Reactivation Limit Exceeded",
  "status": 429,
  "detail": "Maximum reactivations (5) reached in the past 30 days",
  "reactivation_count": 5,
  "next_available": "2024-01-25T10:30:00Z"
}
```

### License Status Endpoint

**GET** `/api/v1/license/status`

#### Response
```json
{
  "license_status": "active",
  "message": "License is valid and active",
  "days_left": 25,
  "license_info": {
    "expiry_date": "2024-02-19T23:59:59Z",
    "activation_date": "2024-01-19T08:00:00Z",
    "device_info": {
      "fingerprint": "a1b2c3d4...",
      "browser": "Chrome",
      "os": "Windows",
      "platform": "Win32",
      "first_activation": "2024-01-19T08:00:00Z",
      "last_seen": "2024-01-19T12:00:00Z",
      "trusted": true
    }
  }
}
```

## Best Practices

### For Developers

1. **Always Include Device Fingerprint**: Ensure fingerprint is generated and sent with activation requests
2. **Handle Reactivation Success**: Display different message for reactivations vs new activations
3. **Implement Retry Logic**: Use exponential backoff for transient failures
4. **Cache License Status**: Reduce API calls by caching status for 5 minutes

### For Support Teams

1. **Review Reactivation History**: Check Column O in Google Sheets for patterns
2. **Monitor Similarity Scores**: Low scores may indicate device changes
3. **Validate Legitimate Cases**: Some users may have valid reasons for multiple reactivations
4. **Security Alerts**: Flag licenses with suspicious reactivation patterns

### For System Administrators

1. **Regular Audits**: Review reactivation logs monthly
2. **Threshold Tuning**: Adjust similarity threshold based on false positive/negative rates
3. **Backup Google Sheets**: Regular backups of license data
4. **Monitor API Performance**: Track activation/reactivation response times

## Migration Guide

### Upgrading from Previous Version

1. **Update Google Sheets**:
   - Add columns N, O, P for reactivation tracking
   - Run migration script to initialize existing licenses

2. **Deploy Backend Changes**:
   - Update license handler with reactivation logic
   - Add new error types for reactivation scenarios

3. **Update Frontend**:
   - Implement device fingerprint generation
   - Update activation flow for reactivation handling

4. **Test Thoroughly**:
   - Test new activation on clean device
   - Test reactivation on same device
   - Test rejection on different device
   - Test rate limiting and error handling

## Monitoring and Metrics

### Key Metrics to Track

1. **Reactivation Success Rate**: Percentage of successful same-device reactivations
2. **False Positive Rate**: Legitimate users blocked incorrectly
3. **False Negative Rate**: Different devices allowed incorrectly
4. **Support Ticket Reduction**: Decrease in reinstallation-related tickets
5. **Average Similarity Score**: For successful reactivations

### Logging

All reactivation attempts are logged with:
- Timestamp
- License key (hashed)
- Device fingerprint
- Similarity score
- Success/failure status
- Error details if failed

### Alerts

Configure alerts for:
- Unusually high reactivation rates (potential abuse)
- Low similarity scores for successful reactivations
- Reactivation limit frequently exceeded
- API errors during activation

## Conclusion

The smart device recognition feature significantly improves user experience while maintaining security. By allowing automatic reactivation on the same device, we reduce support burden and user friction without compromising license integrity.

For additional support or questions, contact the development team or refer to the main documentation.