// ============================================
// ISX LICENSE SYSTEM - COMPLETE GOOGLE APPS SCRIPT
// UPDATED FOR ISX PULSE WITH EMBEDDED CREDENTIALS
// ============================================
// Version: 2.2.0
// Last Updated: 2025-08-20 - Added smart device recognition for license reactivation
// Compatible with: ISX Pulse embedded credential system
// ============================================
// Instructions:
// 1. Copy this ENTIRE file content
// 2. Open your Google Sheet (ID: 1l4jJNNqHZNomjp3wpkL-txDfCjsRr19aJZOZqPHJ6lc)
// 3. Go to Extensions > Apps Script
// 4. DELETE all existing code
// 5. Paste this entire content
// 6. Save the project (Ctrl+S or Cmd+S)
// 7. Run setupAllSheets() first to create/fix sheet structure
// 8. Add test license or generate new licenses
// 9. Deploy as Web App with access set to "Anyone"
// ============================================

// ============================================
// CONFIGURATION - Your Sheet ID and Settings
// ============================================

const SHEET_ID = '1l4jJNNqHZNomjp3wpkL-txDfCjsRr19aJZOZqPHJ6lc';
const MAX_ATTEMPTS_PER_HOUR = 10;
const BLOCK_DURATION_HOURS = 24;

// Shared secret for HMAC verification (must match embedded Go credentials)
const SHARED_SECRET = 'ISX-Pulse-S3cur3-K3y-2024-@lm3s@0dy';

// Test license that should be manually added for testing
const TEST_LICENSE_CODE = 'ISX-FRAC-7Z29-KWFT-QYS2';

// ============================================
// MAIN ACTIVATION ENDPOINT - Handles POST requests
// ============================================

function doPost(e) {
  const lock = LockService.getScriptLock();
  
  try {
    // Acquire lock for atomic operation (wait max 10 seconds)
    lock.waitLock(10000);
    
    // Log incoming request for debugging
    console.log('Incoming POST request:', e.postData.contents);
    
    // Parse request
    const requestData = JSON.parse(e.postData.contents);
    
    // Handle signed requests from ISX Pulse Go backend
    if (requestData.payload) {
      // This is a signed request from the Go backend with embedded credentials
      console.log('Processing signed request from ISX Pulse');
      
      const payload = requestData.payload;
      const fingerprint = requestData.fingerprint || '';
      const signature = requestData.signature || '';
      const timestamp = requestData.timestamp || 0;
      const nonce = requestData.nonce || '';
      const requestId = requestData.request_id || '';
      
      // Extract action from payload, default to 'activate'
      const action = payload.action || 'activate';
      
      // TODO: Implement HMAC signature verification here for production
      // For now, we'll process the request without verification
      
      // Create a compatible request object for our handlers
      // The Go backend sends the license key in 'code' field
      // and device info in 'payload.deviceInfo'
      const compatibleRequest = {
        action: action,
        code: payload.code || payload.license_key,  // Support both 'code' and 'license_key' fields
        license_key: payload.code || payload.license_key,  // Provide both for compatibility
        deviceInfo: payload.deviceInfo || {
          fingerprint: fingerprint,
          ip: e.parameter.ip || '',
          userAgent: e.parameter['user-agent'] || '',
          requestId: requestId
        }
      };
      
      // Route to appropriate handler based on action
      switch(action) {
        case 'activate':
          return handleActivation(compatibleRequest, requestId);
        case 'validate':
          return handleValidation(compatibleRequest, requestId);
        case 'revoke':
          return handleRevocation(compatibleRequest, requestId);
        case 'checkStatus':
          return handleStatusCheck(compatibleRequest, requestId);
        default:
          return createSignedResponse(false, 'Unknown action: ' + action, null, requestId);
      }
    }
    
    // Handle direct requests (not from Go backend)
    const action = requestData.action || 'activate';
    
    // Route to appropriate handler
    switch(action) {
      case 'activate':
        return handleActivation(requestData, '');
      case 'validate':
        return handleValidation(requestData, '');
      case 'revoke':
        return handleRevocation(requestData, '');
      case 'checkStatus':
        return handleStatusCheck(requestData, '');
      default:
        return createSignedResponse(false, 'Unknown action', null, '');
    }
    
  } catch (error) {
    console.error('Error in doPost:', error);
    return createSignedResponse(false, 'Server error: ' + error.toString(), null, '');
  } finally {
    lock.releaseLock();
  }
}

// ============================================
// LICENSE ACTIVATION HANDLER
// ============================================

function handleActivation(request, requestId) {
  // Extract license code - support both 'code' and 'license_key' fields
  let code = request.code || request.license_key;
  const deviceInfo = request.deviceInfo || {};
  
  // Additional extraction for nested payload structure
  if (request.payload) {
    code = request.payload.code || request.payload.license_key || code;
  }
  
  console.log('Activating license:', code);
  console.log('Device info:', deviceInfo);
  
  // Validate license format
  if (!code || !code.startsWith('ISX-')) {
    logActivationAttempt(code, deviceInfo, false, 'Invalid format');
    return createSignedResponse(false, 'Invalid license format. Expected: ISX-XXXX-XXXX-XXXX-XXXX', null, requestId);
  }
  
  // Check blacklist first
  if (isBlacklisted(deviceInfo.ip) || isBlacklisted(deviceInfo.fingerprint)) {
    logActivationAttempt(code, deviceInfo, false, 'Blacklisted');
    return createSignedResponse(false, 'Access denied', null, requestId);
  }
  
  // Check rate limiting
  if (!checkRateLimit(deviceInfo.ip)) {
    addToBlacklist(deviceInfo.ip, 'IP', 'Rate limit exceeded');
    logActivationAttempt(code, deviceInfo, false, 'Rate limited');
    return createSignedResponse(false, 'Too many attempts. Try again later.', null, requestId);
  }
  
  // Get licenses sheet
  const sheet = SpreadsheetApp.openById(SHEET_ID).getSheetByName('Licenses');
  if (!sheet) {
    console.error('Licenses sheet not found!');
    return createSignedResponse(false, 'System error: Licenses sheet not found. Run setupAllSheets() first.', null, requestId);
  }
  
  const dataRange = sheet.getDataRange();
  const values = dataRange.getValues();
  
  // Find the license
  for (let i = 1; i < values.length; i++) {
    if (values[i][0] === code) { // Column A: Code
      
      // Check current status
      const currentStatus = values[i][2]; // Column C: Status
      
      if (currentStatus !== 'Available') {
        // License is already activated - check if it's the same device trying to reactivate
        const storedFingerprint = values[i][5]; // Column F: DeviceFingerprint
        const incomingFingerprint = deviceInfo.fingerprint || '';
        
        if (storedFingerprint && incomingFingerprint) {
          // Calculate fingerprint similarity using fuzzy matching
          const similarity = calculateFingerprintSimilarity(storedFingerprint, incomingFingerprint);
          
          if (similarity >= 0.80) {
            // Same device (80%+ similarity) - check reactivation limits
            const reactivationResult = checkReactivationLimits(code, values[i]);
            
            if (reactivationResult.allowed) {
              // Allow reactivation - update activation date and increment reactivation count
              const now = new Date();
              const reactivationCount = (values[i][10] || 0) + 1; // Use CheckCount column for reactivations
              
              // Update the license row for reactivation
              sheet.getRange(i + 1, 4).setValue(now); // D: ActivationDate (update to current time)
              sheet.getRange(i + 1, 10).setValue(now); // J: LastChecked
              sheet.getRange(i + 1, 11).setValue(reactivationCount); // K: CheckCount (reactivation counter)
              
              // Log successful reactivation
              logActivationAttempt(code, deviceInfo, true, 'Reactivation success');
              logAudit('REACTIVATION', code, deviceInfo.ip, `License reactivated successfully (attempt ${reactivationCount}/5, similarity: ${Math.round(similarity * 100)}%)`);
              
              // Return success response for reactivation
              return createSignedResponse(true, 'License reactivated successfully', {
                status: 'reactivated',
                license_key: code,
                activation_id: values[i][8], // Keep existing activation ID
                expires_at: values[i][7] ? new Date(values[i][7]).toISOString() : '',
                device_id: deviceInfo.fingerprint || '',
                duration: values[i][1],
                reactivation_count: reactivationCount,
                similarity_score: Math.round(similarity * 100),
                features: ['all']
              }, requestId);
            } else {
              // Reactivation limit exceeded
              logActivationAttempt(code, deviceInfo, false, `Reactivation limit exceeded: ${reactivationResult.reason}`);
              logAudit('REACTIVATION_BLOCKED', code, deviceInfo.ip, `Reactivation blocked: ${reactivationResult.reason}`);
              
              return createSignedResponse(false, 'Reactivation limit exceeded', {
                status: 'reactivation_blocked',
                reason: reactivationResult.reason,
                attempts_used: reactivationResult.attemptsUsed,
                max_attempts: 5,
                reset_date: reactivationResult.resetDate
              }, requestId);
            }
          }
        }
        
        // Different device or insufficient similarity
        logActivationAttempt(code, deviceInfo, false, 'Already activated on different device');
        logAudit('ACTIVATION_BLOCKED', code, deviceInfo.ip, `License already activated on different device (similarity: ${Math.round((calculateFingerprintSimilarity(storedFingerprint, incomingFingerprint) || 0) * 100)}%)`);
        
        // Return existing activation details for different device
        return createSignedResponse(false, 'License already activated on a different device', {
          status: 'already_activated_different_device',
          activationDate: values[i][3],
          expiryDate: values[i][7],
          currentStatus: currentStatus,
          device_similarity: Math.round((calculateFingerprintSimilarity(storedFingerprint, incomingFingerprint) || 0) * 100)
        }, requestId);
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
      
      // Return success response compatible with Go backend
      return createSignedResponse(true, 'License activated successfully', {
        status: 'activated',
        license_key: code,
        activation_id: activationId,
        expires_at: expiryDate.toISOString(),
        device_id: deviceInfo.fingerprint || '',
        duration: duration,
        features: ['all'] // Grant all features for now
      }, requestId);
    }
  }
  
  // License not found
  logActivationAttempt(code, deviceInfo, false, 'Invalid code');
  return createSignedResponse(false, 'Invalid license code. Please check your license key.', null, requestId);
}

// ============================================
// LICENSE VALIDATION HANDLER
// ============================================

function handleValidation(request, requestId) {
  const code = request.code || request.license_key;
  const activationId = request.activationId || request.activation_id;
  const deviceFingerprint = request.deviceFingerprint || request.fingerprint || '';
  
  const sheet = SpreadsheetApp.openById(SHEET_ID).getSheetByName('Licenses');
  if (!sheet) {
    return createSignedResponse(false, 'System error: Licenses sheet not found', null, requestId);
  }
  
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
        return createSignedResponse(false, 'License expired', {
          status: 'expired',
          expiryDate: expiryDate.toISOString()
        }, requestId);
      }
      
      // Check device binding (optional - log only for now)
      if (storedFingerprint && storedFingerprint !== deviceFingerprint) {
        logAudit('VALIDATION_MISMATCH', code, '', 'Device fingerprint mismatch');
        // For now, we'll allow it but log the mismatch
      }
      
      // Update last checked
      sheet.getRange(i + 1, 10).setValue(new Date()); // J: LastChecked
      sheet.getRange(i + 1, 11).setValue(values[i][10] + 1); // K: CheckCount++
      
      return createSignedResponse(true, 'License valid', {
        status: 'valid',
        license_status: status,
        expires_at: expiryDate.toISOString(),
        checks_remaining: Math.max(0, 1000 - values[i][10])
      }, requestId);
    }
  }
  
  return createSignedResponse(false, 'Invalid license or activation ID', null, requestId);
}

// ============================================
// STUB HANDLERS (implement as needed)
// ============================================

function handleRevocation(request, requestId) {
  // TODO: Implement license revocation
  return createSignedResponse(false, 'Revocation not implemented', null, requestId);
}

function handleStatusCheck(request, requestId) {
  // TODO: Implement status check
  return createSignedResponse(false, 'Status check not implemented', null, requestId);
}

// ============================================
// RESPONSE CREATION - Compatible with Go backend
// ============================================

function createSignedResponse(success, message, data, requestId) {
  // Create response matching the Go backend's SignedResponse structure
  const response = {
    timestamp: Math.floor(Date.now() / 1000), // Unix timestamp in seconds
    request_id: requestId || Utilities.getUuid(),
    success: success,
    data: data || {},
    error: success ? '' : message,
    signature: '' // Will be calculated below
  };
  
  // Add message to data if successful
  if (success && message) {
    response.data.message = message;
  }
  
  // Generate HMAC signature for the response
  // Create canonical string to sign (must match Go backend's verification)
  let canonical = `${response.timestamp}|${response.request_id}|${response.success}`;
  
  // Add data if present
  if (response.data && Object.keys(response.data).length > 0) {
    const dataJSON = JSON.stringify(response.data);
    canonical += '|' + dataJSON;
  }
  
  // Add error if present
  if (response.error) {
    canonical += '|' + response.error;
  }
  
  // Create HMAC-SHA256 signature
  const signature = Utilities.computeHmacSha256Signature(canonical, SHARED_SECRET);
  response.signature = Utilities.base64Encode(signature);
  
  console.log('Sending signed response:', response);
  
  return ContentService.createTextOutput(
    JSON.stringify(response)
  ).setMimeType(ContentService.MimeType.JSON);
}

// ============================================
// TEST ENDPOINT - Handles GET requests
// ============================================

function doGet(e) {
  return ContentService.createTextOutput(
    JSON.stringify({
      status: 'OK',
      message: 'ISX License Manager API is running',
      version: '2.1.0',
      sheet_id: SHEET_ID,
      test_license: TEST_LICENSE_CODE,
      timestamp: new Date().toISOString()
    })
  ).setMimeType(ContentService.MimeType.JSON);
}

// ============================================
// DEVICE RECOGNITION AND REACTIVATION HELPERS
// ============================================

/**
 * Calculate similarity between two device fingerprints using fuzzy matching
 * Uses Jaccard similarity for robust comparison
 * @param {string} fingerprint1 - Stored device fingerprint
 * @param {string} fingerprint2 - Incoming device fingerprint
 * @returns {number} Similarity score between 0.0 and 1.0
 */
function calculateFingerprintSimilarity(fingerprint1, fingerprint2) {
  if (!fingerprint1 || !fingerprint2) {
    return 0.0;
  }
  
  // Normalize fingerprints (remove spaces, convert to lowercase)
  const fp1 = fingerprint1.toLowerCase().replace(/\s+/g, '');
  const fp2 = fingerprint2.toLowerCase().replace(/\s+/g, '');
  
  // Exact match
  if (fp1 === fp2) {
    return 1.0;
  }
  
  // Create character bigrams for better matching
  const getBigrams = (str) => {
    const bigrams = new Set();
    for (let i = 0; i < str.length - 1; i++) {
      bigrams.add(str.substring(i, i + 2));
    }
    return bigrams;
  };
  
  const bigrams1 = getBigrams(fp1);
  const bigrams2 = getBigrams(fp2);
  
  // Calculate Jaccard similarity (intersection over union)
  const intersection = new Set([...bigrams1].filter(x => bigrams2.has(x)));
  const union = new Set([...bigrams1, ...bigrams2]);
  
  if (union.size === 0) {
    return 0.0;
  }
  
  const jaccardSimilarity = intersection.size / union.size;
  
  // Also check substring similarity for partial matches
  const longerLength = Math.max(fp1.length, fp2.length);
  const shorterLength = Math.min(fp1.length, fp2.length);
  const lengthSimilarity = shorterLength / longerLength;
  
  // Combine Jaccard and length similarity with weights
  const combinedSimilarity = (jaccardSimilarity * 0.8) + (lengthSimilarity * 0.2);
  
  return Math.min(1.0, combinedSimilarity);
}

/**
 * Check if reactivation is allowed based on limits and timing
 * @param {string} licenseCode - The license code being reactivated
 * @param {Array} licenseRow - The license row data from the sheet
 * @returns {Object} Result object with allowed status and details
 */
function checkReactivationLimits(licenseCode, licenseRow) {
  const maxReactivationsPerMonth = 5;
  const currentReactivationCount = licenseRow[10] || 0; // Column K: CheckCount (repurposed for reactivations)
  const lastChecked = licenseRow[9]; // Column J: LastChecked
  
  // If no previous reactivations, allow
  if (currentReactivationCount === 0) {
    return {
      allowed: true,
      attemptsUsed: 0,
      maxAttempts: maxReactivationsPerMonth,
      resetDate: null
    };
  }
  
  // Check if we're within the 30-day window
  const now = new Date();
  const thirtyDaysAgo = new Date(now.getTime() - (30 * 24 * 60 * 60 * 1000));
  
  let effectiveReactivationCount = currentReactivationCount;
  
  // If last check was more than 30 days ago, reset the counter
  if (lastChecked && new Date(lastChecked) < thirtyDaysAgo) {
    effectiveReactivationCount = 0;
  }
  
  // Check if limit exceeded
  if (effectiveReactivationCount >= maxReactivationsPerMonth) {
    const resetDate = lastChecked ? new Date(new Date(lastChecked).getTime() + (30 * 24 * 60 * 60 * 1000)) : null;
    
    return {
      allowed: false,
      reason: `Maximum ${maxReactivationsPerMonth} reactivations per 30 days exceeded`,
      attemptsUsed: effectiveReactivationCount,
      maxAttempts: maxReactivationsPerMonth,
      resetDate: resetDate ? resetDate.toISOString() : null
    };
  }
  
  // Allow reactivation
  return {
    allowed: true,
    attemptsUsed: effectiveReactivationCount,
    maxAttempts: maxReactivationsPerMonth,
    resetDate: null
  };
}

// ============================================
// HELPER FUNCTIONS
// ============================================

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
  
  try {
    const sheet = SpreadsheetApp.openById(SHEET_ID).getSheetByName('ActivationAttempts');
    if (!sheet) return true; // Allow if sheet doesn't exist
    
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
  } catch (error) {
    console.error('Error checking rate limit:', error);
    return true; // Allow on error
  }
}

function isBlacklisted(identifier) {
  if (!identifier) return false;
  
  try {
    const sheet = SpreadsheetApp.openById(SHEET_ID).getSheetByName('Blacklist');
    if (!sheet) return false; // Not blacklisted if sheet doesn't exist
    
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
  } catch (error) {
    console.error('Error checking blacklist:', error);
    return false; // Not blacklisted on error
  }
}

function addToBlacklist(identifier, type, reason) {
  try {
    const sheet = SpreadsheetApp.openById(SHEET_ID).getSheetByName('Blacklist');
    if (!sheet) return; // Skip if sheet doesn't exist
    
    const now = new Date();
    const expiryDate = new Date(now.getTime() + BLOCK_DURATION_HOURS * 60 * 60 * 1000);
    
    sheet.appendRow([identifier, type, reason, now, 'System', expiryDate]);
  } catch (error) {
    console.error('Error adding to blacklist:', error);
  }
}

function logActivationAttempt(code, deviceInfo, success, error) {
  try {
    const sheet = SpreadsheetApp.openById(SHEET_ID).getSheetByName('ActivationAttempts');
    if (!sheet) return; // Skip if sheet doesn't exist
    
    sheet.appendRow([
      new Date(),
      code || 'unknown',
      deviceInfo.ip || '',
      success,
      error || '',
      deviceInfo.fingerprint || '',
      deviceInfo.userAgent || ''
    ]);
  } catch (error) {
    console.error('Error logging activation attempt:', error);
  }
}

function logAudit(action, code, performer, details) {
  try {
    const sheet = SpreadsheetApp.openById(SHEET_ID).getSheetByName('AuditLog');
    if (!sheet) return; // Skip if sheet doesn't exist
    
    sheet.appendRow([
      new Date(),
      action,
      code,
      performer,
      details,
      'Success'
    ]);
  } catch (error) {
    console.error('Error logging audit:', error);
  }
}

// ============================================
// SHEET SETUP AND MANAGEMENT
// ============================================

// Menu in Google Sheets
function onOpen() {
  const ui = SpreadsheetApp.getUi();
  ui.createMenu('🎫 ISX License System')
    .addItem('🔧 Setup All Sheets', 'setupAllSheets')
    .addItem('➕ Add Test License', 'addTestLicense')
    .addItem('🎲 Generate 100 Licenses (1 month)', 'generate100Licenses')
    .addItem('📊 Show Statistics', 'showStats')
    .addSeparator()
    .addItem('🧪 Test Fingerprint Similarity', 'testFingerprintSimilarity')
    .addItem('🗑️ Clear All Data', 'clearAllData')
    .addToUi();
}

// MAIN SETUP FUNCTION - RUN THIS FIRST!
function setupAllSheets() {
  const spreadsheet = SpreadsheetApp.openById(SHEET_ID);
  
  // Setup each sheet with correct structure
  setupLicensesSheet(spreadsheet);
  setupActivationAttemptsSheet(spreadsheet);
  setupBlacklistSheet(spreadsheet);
  setupAuditLogSheet(spreadsheet);
  
  SpreadsheetApp.getUi().alert(
    '✅ Setup Complete!',
    'All 4 sheets have been set up with correct column structure.\n\n' +
    'Sheets ready:\n' +
    '• Licenses (14 columns)\n' +
    '• ActivationAttempts (7 columns)\n' +
    '• Blacklist (6 columns)\n' +
    '• AuditLog (6 columns)\n\n' +
    'You can now:\n' +
    '1. Add the test license\n' +
    '2. Generate new licenses\n' +
    '3. Deploy as Web App',
    SpreadsheetApp.getUi().ButtonSet.OK
  );
}

// Add test license for ISX Pulse testing
function addTestLicense() {
  const sheet = SpreadsheetApp.openById(SHEET_ID).getSheetByName('Licenses');
  
  if (!sheet) {
    SpreadsheetApp.getUi().alert('Error', 'Please run Setup All Sheets first!', SpreadsheetApp.getUi().ButtonSet.OK);
    return;
  }
  
  // Check if test license already exists
  const data = sheet.getDataRange().getValues();
  for (let i = 1; i < data.length; i++) {
    if (data[i][0] === TEST_LICENSE_CODE) {
      SpreadsheetApp.getUi().alert(
        'Test License Exists',
        `The test license ${TEST_LICENSE_CODE} already exists in row ${i + 1}.\n` +
        `Status: ${data[i][2]}`,
        SpreadsheetApp.getUi().ButtonSet.OK
      );
      return;
    }
  }
  
  // Add test license
  const createdDate = new Date();
  sheet.appendRow([
    TEST_LICENSE_CODE,          // A: Code
    '1m',                      // B: Duration (1 month)
    'Available',               // C: Status
    '',                        // D: ActivationDate
    '',                        // E: ActivationIP
    '',                        // F: DeviceFingerprint
    '',                        // G: Email
    '',                        // H: ExpiryDate
    '',                        // I: ActivationID
    '',                        // J: LastChecked
    0,                         // K: CheckCount
    createdDate,               // L: CreatedDate
    'TEST-BATCH',              // M: BatchID
    'Test license for ISX Pulse' // N: Notes
  ]);
  
  SpreadsheetApp.getUi().alert(
    '✅ Test License Added!',
    `Test license ${TEST_LICENSE_CODE} has been added.\n\n` +
    'You can now test activation with this license code.',
    SpreadsheetApp.getUi().ButtonSet.OK
  );
}

// Setup Licenses sheet with CORRECT column order
function setupLicensesSheet(spreadsheet) {
  let sheet = spreadsheet.getSheetByName('Licenses');
  if (!sheet) {
    sheet = spreadsheet.insertSheet('Licenses');
  }
  
  // Clear existing content
  sheet.clear();
  
  // CORRECT headers matching activation handler expectations
  const headers = [
    'Code',               // A: License code (ISX-XXXX-XXXX-XXXX-XXXX)
    'Duration',           // B: Duration (1m, 3m, 6m, 1y)
    'Status',             // C: Available/Activated/Expired
    'ActivationDate',     // D: When activated
    'ActivationIP',       // E: IP address that activated
    'DeviceFingerprint',  // F: Device ID
    'Email',              // G: User email
    'ExpiryDate',         // H: When expires (calculated on activation)
    'ActivationID',       // I: Unique activation ID
    'LastChecked',        // J: Last validation check
    'CheckCount',         // K: Number of checks
    'CreatedDate',        // L: When license was generated
    'BatchID',            // M: Batch identifier
    'Notes'               // N: Optional notes
  ];
  
  sheet.getRange(1, 1, 1, headers.length).setValues([headers]);
  
  // Format headers
  const headerRange = sheet.getRange(1, 1, 1, headers.length);
  headerRange.setBackground('#4285f4');
  headerRange.setFontColor('#ffffff');
  headerRange.setFontWeight('bold');
  headerRange.setHorizontalAlignment('center');
  
  // Set column widths for better visibility
  sheet.setColumnWidth(1, 200); // Code
  sheet.setColumnWidth(2, 80);  // Duration
  sheet.setColumnWidth(3, 100); // Status
  sheet.setColumnWidth(4, 150); // ActivationDate
  sheet.setColumnWidth(5, 120); // ActivationIP
  sheet.setColumnWidth(6, 200); // DeviceFingerprint
  sheet.setColumnWidth(7, 150); // Email
  sheet.setColumnWidth(8, 150); // ExpiryDate
  sheet.setColumnWidth(9, 200); // ActivationID
  sheet.setColumnWidth(10, 150); // LastChecked
  sheet.setColumnWidth(11, 100); // CheckCount
  sheet.setColumnWidth(12, 150); // CreatedDate
  sheet.setColumnWidth(13, 200); // BatchID
  sheet.setColumnWidth(14, 200); // Notes
  
  sheet.setFrozenRows(1);
}

// Setup ActivationAttempts sheet
function setupActivationAttemptsSheet(spreadsheet) {
  let sheet = spreadsheet.getSheetByName('ActivationAttempts');
  if (!sheet) {
    sheet = spreadsheet.insertSheet('ActivationAttempts');
  }
  
  sheet.clear();
  
  const headers = [
    'Timestamp',
    'Code',
    'IP',
    'Success',
    'Error',
    'DeviceFingerprint',
    'UserAgent'
  ];
  
  sheet.getRange(1, 1, 1, headers.length).setValues([headers]);
  
  const headerRange = sheet.getRange(1, 1, 1, headers.length);
  headerRange.setBackground('#ea4335');
  headerRange.setFontColor('#ffffff');
  headerRange.setFontWeight('bold');
  headerRange.setHorizontalAlignment('center');
  sheet.setFrozenRows(1);
}

// Setup Blacklist sheet
function setupBlacklistSheet(spreadsheet) {
  let sheet = spreadsheet.getSheetByName('Blacklist');
  if (!sheet) {
    sheet = spreadsheet.insertSheet('Blacklist');
  }
  
  sheet.clear();
  
  const headers = [
    'Identifier',
    'Type',
    'Reason',
    'AddedDate',
    'AddedBy',
    'ExpiryDate'
  ];
  
  sheet.getRange(1, 1, 1, headers.length).setValues([headers]);
  
  const headerRange = sheet.getRange(1, 1, 1, headers.length);
  headerRange.setBackground('#fbbc04');
  headerRange.setFontColor('#000000');
  headerRange.setFontWeight('bold');
  headerRange.setHorizontalAlignment('center');
  sheet.setFrozenRows(1);
}

// Setup AuditLog sheet
function setupAuditLogSheet(spreadsheet) {
  let sheet = spreadsheet.getSheetByName('AuditLog');
  if (!sheet) {
    sheet = spreadsheet.insertSheet('AuditLog');
  }
  
  sheet.clear();
  
  const headers = [
    'Timestamp',
    'Action',
    'LicenseCode',
    'PerformedBy',
    'Details',
    'Result'
  ];
  
  sheet.getRange(1, 1, 1, headers.length).setValues([headers]);
  
  const headerRange = sheet.getRange(1, 1, 1, headers.length);
  headerRange.setBackground('#34a853');
  headerRange.setFontColor('#ffffff');
  headerRange.setFontWeight('bold');
  headerRange.setHorizontalAlignment('center');
  sheet.setFrozenRows(1);
}

// Generate 100 licenses
function generate100Licenses() {
  const sheet = SpreadsheetApp.openById(SHEET_ID).getSheetByName('Licenses');
  
  if (!sheet) {
    SpreadsheetApp.getUi().alert('Error', 'Please run Setup All Sheets first!', SpreadsheetApp.getUi().ButtonSet.OK);
    return;
  }
  
  // Generate batch ID with timestamp
  const batchId = 'BATCH-' + Utilities.formatDate(new Date(), 'GMT', 'yyyyMMdd-HHmmss');
  const createdDate = new Date();
  
  // Get existing codes to check uniqueness
  const existingData = sheet.getDataRange().getValues();
  const existingCodes = new Set();
  for (let i = 1; i < existingData.length; i++) {
    if (existingData[i][0]) existingCodes.add(existingData[i][0]);
  }
  
  // Generate 100 unique codes
  const newLicenses = [];
  const chars = '23456789ABCDEFGHJKMNPQRSTUVWXYZ'; // No confusing characters
  let generated = 0;
  
  while (generated < 100) {
    // Generate code format: ISX-XXXX-XXXX-XXXX-XXXX
    let code = 'ISX';
    for (let segment = 0; segment < 4; segment++) {
      code += '-';
      for (let i = 0; i < 4; i++) {
        code += chars.charAt(Math.floor(Math.random() * chars.length));
      }
    }
    
    // Check if unique
    if (!existingCodes.has(code)) {
      // CORRECT COLUMN ORDER - Matching activation handler
      newLicenses.push([
        code,                    // A: Code
        '1m',                   // B: Duration (1 month)
        'Available',            // C: Status
        '',                     // D: ActivationDate
        '',                     // E: ActivationIP
        '',                     // F: DeviceFingerprint
        '',                     // G: Email
        '',                     // H: ExpiryDate
        '',                     // I: ActivationID
        '',                     // J: LastChecked
        0,                      // K: CheckCount
        createdDate,            // L: CreatedDate
        batchId,                // M: BatchID
        ''                      // N: Notes
      ]);
      existingCodes.add(code);
      generated++;
    }
  }
  
  // Add all licenses to sheet
  if (newLicenses.length > 0) {
    const lastRow = sheet.getLastRow();
    sheet.getRange(lastRow + 1, 1, newLicenses.length, 14).setValues(newLicenses);
  }
  
  // Show success message
  SpreadsheetApp.getUi().alert(
    '✅ Success!',
    `Generated 100 licenses\n` +
    `Batch ID: ${batchId}\n` +
    `Duration: 1 month\n\n` +
    `Total licenses in sheet: ${sheet.getLastRow() - 1}`,
    SpreadsheetApp.getUi().ButtonSet.OK
  );
  
  // Log to audit sheet
  logAudit('GENERATE_BATCH', `Batch: ${batchId}`, Session.getActiveUser().getEmail(), 'Generated 100 licenses (1 month)');
}

// Show statistics
function showStats() {
  const sheet = SpreadsheetApp.openById(SHEET_ID).getSheetByName('Licenses');
  
  if (!sheet || sheet.getLastRow() <= 1) {
    SpreadsheetApp.getUi().alert('No Data', 'No licenses found in the sheet.', SpreadsheetApp.getUi().ButtonSet.OK);
    return;
  }
  
  const data = sheet.getDataRange().getValues();
  
  let stats = {
    total: 0,
    available: 0,
    activated: 0,
    expired: 0
  };
  
  // Count licenses by status (skip header row)
  for (let i = 1; i < data.length; i++) {
    if (data[i][0]) { // If code exists
      stats.total++;
      const status = data[i][2]; // Column C: Status
      if (status === 'Available') stats.available++;
      else if (status === 'Activated') stats.activated++;
      else if (status === 'Expired') stats.expired++;
    }
  }
  
  SpreadsheetApp.getUi().alert(
    '📊 License Statistics',
    `Total Licenses: ${stats.total}\n\n` +
    `✅ Available: ${stats.available} (${Math.round((stats.available/stats.total)*100)}%)\n` +
    `🔒 Activated: ${stats.activated} (${Math.round((stats.activated/stats.total)*100)}%)\n` +
    `⏰ Expired: ${stats.expired} (${Math.round((stats.expired/stats.total)*100)}%)\n\n` +
    `Usage Rate: ${Math.round((stats.activated/stats.total)*100)}%`,
    SpreadsheetApp.getUi().ButtonSet.OK
  );
}

// Clear all data (keep headers)
function clearAllData() {
  const sheetNames = ['Licenses', 'ActivationAttempts', 'Blacklist', 'AuditLog'];
  
  sheetNames.forEach(name => {
    const sheet = SpreadsheetApp.openById(SHEET_ID).getSheetByName(name);
    if (sheet && sheet.getLastRow() > 1) {
      sheet.getRange(2, 1, sheet.getLastRow() - 1, sheet.getLastColumn()).clear();
    }
  });
  
  SpreadsheetApp.getUi().alert(
    '✅ Data Cleared',
    'All data has been cleared from all sheets (headers preserved).',
    SpreadsheetApp.getUi().ButtonSet.OK
  );
}

// Test function to demonstrate fingerprint similarity calculation
function testFingerprintSimilarity() {
  // Sample test cases for fingerprint similarity
  const testCases = [
    // Exact match
    {
      fp1: 'Mozilla/5.0-Windows-Chrome-1920x1080-America/New_York',
      fp2: 'Mozilla/5.0-Windows-Chrome-1920x1080-America/New_York',
      expected: '100% (exact match)'
    },
    // Very similar (same browser, different version)
    {
      fp1: 'Mozilla/5.0-Windows-Chrome-1920x1080-America/New_York',
      fp2: 'Mozilla/5.0-Windows-Chrome-1920x1080-America/Chicago',
      expected: '~85-95% (same device, different timezone)'
    },
    // Moderately similar (same OS, different browser)
    {
      fp1: 'Mozilla/5.0-Windows-Chrome-1920x1080-America/New_York',
      fp2: 'Mozilla/5.0-Windows-Firefox-1920x1080-America/New_York',
      expected: '~70-85% (same device, different browser)'
    },
    // Different devices
    {
      fp1: 'Mozilla/5.0-Windows-Chrome-1920x1080-America/New_York',
      fp2: 'Mozilla/5.0-macOS-Safari-2560x1600-Europe/London',
      expected: '~20-40% (completely different device)'
    }
  ];
  
  let results = '🧪 FINGERPRINT SIMILARITY TEST RESULTS:\n\n';
  
  testCases.forEach((testCase, index) => {
    const similarity = calculateFingerprintSimilarity(testCase.fp1, testCase.fp2);
    const percentage = Math.round(similarity * 100);
    const status = similarity >= 0.80 ? '✅ ALLOW REACTIVATION' : '❌ BLOCK (different device)';
    
    results += `Test ${index + 1}:\n`;
    results += `Expected: ${testCase.expected}\n`;
    results += `Actual: ${percentage}% ${status}\n`;
    results += `Fingerprint 1: ${testCase.fp1}\n`;
    results += `Fingerprint 2: ${testCase.fp2}\n\n`;
  });
  
  results += 'REACTIVATION RULES:\n';
  results += '• ≥80% similarity = Same device (allow reactivation)\n';
  results += '• <80% similarity = Different device (block)\n';
  results += '• Max 5 reactivations per 30 days per license\n';
  results += '• Counter resets automatically after 30 days';
  
  SpreadsheetApp.getUi().alert(
    '🧪 Fingerprint Similarity Test',
    results,
    SpreadsheetApp.getUi().ButtonSet.OK
  );
}

// ============================================
// END OF COMPLETE GOOGLE APPS SCRIPT
// ============================================
// After copying this to Google Apps Script:
// 1. Save the project (Ctrl+S or Cmd+S)
// 2. Run setupAllSheets() to create sheets
// 3. Run addTestLicense() to add the test license
// 4. Deploy > New Deployment > Web App
// 5. Set access to "Anyone"
// 6. Copy the deployment URL
// 7. Update your Go embedded credentials with the URL
// ============================================
//
// NEW IN VERSION 2.2.0 - SMART DEVICE RECOGNITION:
// ============================================
// • Fuzzy matching for device fingerprints (80% similarity threshold)
// • Allow same device reactivation up to 5 times per 30 days
// • Jaccard similarity algorithm for robust fingerprint comparison
// • Automatic reactivation counter reset after 30 days
// • Enhanced audit logging for reactivations and blocked attempts
// • Backward compatible with existing license data
// • Distinct responses for reactivation vs different device activation
// 
// Response Types:
// • 'reactivated' - Same device successfully reactivated
// • 'already_activated_different_device' - Different device attempted
// • 'reactivation_blocked' - Same device exceeded 5/30 day limit
// 
// Security Features:
// • Device fingerprint normalization and bigram analysis
// • Rate limiting still applies for all activation attempts
// • Comprehensive audit trail for all reactivation events
// • Similarity scores logged for forensic analysis
// ============================================