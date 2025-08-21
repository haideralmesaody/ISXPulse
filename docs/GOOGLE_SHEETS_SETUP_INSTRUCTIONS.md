# Google Sheets Setup Instructions for ISX Pulse Scratch Card System

## Quick Fix Guide for Your Current Issue

Your Google Sheets currently has corrupted data with misaligned columns. Follow these steps to fix it:

## Step 1: Setup Sheet Headers Using Apps Script

1. **Open your Google Sheet**
2. **Go to Extensions → Apps Script**
3. **Delete any existing code** in the editor
4. **Copy the entire contents** of `scripts/setup-sheets-headers.gs`
5. **Paste it** into the Apps Script editor
6. **Save** (Ctrl+S or Cmd+S)
7. **Run the function**:
   - Click the dropdown that says "Select function"
   - Choose `setupAllSheets`
   - Click the "Run" button (▶️)
   - Grant permissions when prompted

This will:
- Create/update all 4 sheets (Licenses, ActivationAttempts, Blacklist, AuditLog)
- Set proper headers with formatting
- Configure column widths
- Add a custom menu to your sheet

## Step 2: Import the Generated Licenses

### Option A: Direct CSV Import (Recommended)

1. **In your Google Sheet**, go to the `Licenses` sheet
2. **File → Import**
3. **Upload** the file `dist/licenses_for_sheets.csv`
4. **Import settings**:
   - Import location: **Replace current sheet**
   - Separator type: **Comma**
   - Convert text: **No**
5. Click **Import data**

### Option B: Copy-Paste Method

1. Open `dist/licenses_for_sheets.csv` in Excel or a text editor
2. Select all data **except the header row** (row 2 onwards)
3. In Google Sheets, click on cell A2 in the Licenses sheet
4. Paste (Ctrl+V or Cmd+V)

## Step 3: Verify the Structure

After import, your Licenses sheet should look like this:

| licenseKey | duration | status | activatedBy | activationDate | expiryDate | deviceFingerprint | activationID | createdDate | batchID | qrCode | notes |
|------------|----------|--------|-------------|----------------|------------|-------------------|--------------|-------------|---------|--------|-------|
| ISX-AMNS-J6SP-CNGM-896X | 1m | Available | | | | | | 2025-08-18 11:12:55 | PRODUCTION-001 | | |
| ISX-MNE3-C8CU-36EE-R9GB | 1m | Available | | | | | | 2025-08-18 11:12:55 | PRODUCTION-001 | | |

## Complete Sheet Structure Reference

### 1. Licenses Sheet
- **Purpose**: Store all generated scratch card licenses
- **Key Fields**:
  - `licenseKey`: The scratch card code (ISX-XXXX-XXXX-XXXX)
  - `duration`: How long the license is valid (1m, 3m, 6m, 12m)
  - `status`: Available, Activated, Expired, Blocked
  - Empty fields are filled during activation

### 2. ActivationAttempts Sheet
- **Purpose**: Log all activation attempts for security monitoring
- **Automatically populated** by the Apps Script endpoint

### 3. Blacklist Sheet
- **Purpose**: Block suspicious IPs, devices, or license keys
- **Manual entries** for security management

### 4. AuditLog Sheet  
- **Purpose**: Complete audit trail of all operations
- **Automatically populated** for compliance

## Generating New Licenses

### Method 1: Using License Generator Tool

```bash
# From project root
cd dist

# Generate 100 licenses for 1 month
./license-generator.exe -count 100 -duration "1m" -batch "BATCH-001" -export csv

# Convert to Google Sheets format
python ../scripts/convert-licenses-for-sheets.py licenses_BATCH-001_*.csv licenses_for_import.csv
```

### Method 2: Using Apps Script Generator

After setting up the sheets, you can use the built-in generator:

1. In Google Sheets, look for the **ISX License System** menu
2. Click **Generate Sample Licenses**
3. This will add 10 test licenses directly to your sheet

### Method 3: Batch Script in Google Sheets

Add this function to your Apps Script:

```javascript
function generateBatchLicenses() {
  const COUNT = 100;  // Number of licenses to generate
  const DURATION = '1m';  // Duration (1m, 3m, 6m, 12m)
  
  const sheet = SpreadsheetApp.getActiveSpreadsheet().getSheetByName('Licenses');
  const batchID = 'BATCH-' + Utilities.formatDate(new Date(), 'GMT', 'yyyyMMdd-HHmmss');
  const createdDate = Utilities.formatDate(new Date(), 'GMT', 'yyyy-MM-dd HH:mm:ss');
  
  const licenses = [];
  for (let i = 0; i < COUNT; i++) {
    licenses.push([
      generateLicenseKey(),  // Auto-generates ISX-XXXX-XXXX-XXXX
      DURATION,
      'Available',
      '', '', '', '', '',  // Empty activation fields
      createdDate,
      batchID,
      '',  // QR code
      ''   // Notes
    ]);
  }
  
  // Find last row and append
  const lastRow = sheet.getLastRow();
  sheet.getRange(lastRow + 1, 1, licenses.length, 12).setValues(licenses);
  
  SpreadsheetApp.getUi().alert(`Generated ${COUNT} licenses with batch ID: ${batchID}`);
}
```

## Troubleshooting

### Problem: Columns are misaligned
**Solution**: Run `setupAllSheets()` to reset the structure, then re-import data

### Problem: Can't import CSV
**Solution**: Make sure you're importing to the correct sheet and the CSV has proper headers

### Problem: License format is wrong (not ISX-XXXX-XXXX-XXXX)
**Solution**: The system now requires the extended format with 4 groups of 4 characters

### Problem: Apps Script permissions error
**Solution**: 
1. Go to your Google Account settings
2. Security → Third-party apps
3. Ensure Apps Script has necessary permissions

## Data Format Requirements

### License Key Format
- **Pattern**: `ISX-XXXX-XXXX-XXXX` (where X is A-Z or 0-9)
- **Example**: `ISX-AMNS-J6SP-CNGM`
- **Case**: Always uppercase

### Duration Values
- `1m` = 1 month
- `3m` = 3 months
- `6m` = 6 months
- `12m` = 12 months

### Status Values
- `Available` - Ready for activation
- `Activated` - Currently in use
- `Expired` - Past expiry date
- `Blocked` - Manually blocked

### Date Format
- Use ISO format: `YYYY-MM-DD HH:MM:SS`
- Example: `2025-08-18 11:12:55`

## Security Notes

1. **Never share** your Google Sheets publicly
2. **Restrict access** to authorized personnel only
3. **Regular backups** - use File → Make a copy weekly
4. **Monitor** the ActivationAttempts sheet for suspicious activity
5. **Use the Blacklist** sheet to block suspicious IPs or devices

## Integration with ISX Pulse

Once your sheets are set up:

1. The ISX Pulse application will use the Google Apps Script endpoint
2. All activations will be atomic operations through the script
3. The system will automatically:
   - Validate license keys
   - Check device fingerprints
   - Update activation status
   - Log all attempts

## Next Steps

1. ✅ Fix your current sheet structure using `setupAllSheets()`
2. ✅ Import the 100 generated licenses
3. ✅ Test activation with a sample license
4. ✅ Monitor the ActivationAttempts sheet
5. ✅ Set up regular backups

## Support Files

All necessary files are in your project:
- `scripts/setup-sheets-headers.gs` - Sheet setup script
- `scripts/convert-licenses-for-sheets.py` - CSV converter
- `dist/licenses_for_sheets.csv` - Ready-to-import licenses
- `tools/license-generator/` - License generation tool

For any issues, check the AuditLog sheet for detailed error messages.