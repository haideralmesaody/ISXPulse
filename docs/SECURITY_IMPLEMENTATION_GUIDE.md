# ISX Daily Reports Scrapper - Secure Credential System Implementation Guide

## Overview

This document describes the implementation of a comprehensive secure credential encryption system for the ISX Daily Reports Scrapper. The system replaces hardcoded service account credentials with enterprise-grade encrypted storage following OWASP ASVS requirements.

## Security Architecture

### Core Components

1. **AES-256-GCM Encryption** (`dev/internal/security/encryption.go`)
   - Industry-standard authenticated encryption
   - 96-bit nonces with 128-bit authentication tags
   - SCRYPT key derivation (N=32768, r=8, p=1)
   - Memory protection with secure credential clearing

2. **Binary Integrity Verification** (`dev/internal/security/integrity.go`)
   - SHA-256 binary hash verification
   - Anti-tampering detection mechanisms
   - Runtime integrity checks
   - Debugger and VM detection

3. **Certificate Pinning** (`dev/internal/security/pinning.go`)
   - Google APIs certificate pinning
   - TLS 1.2+ enforcement
   - Secure cipher suite selection
   - Transport layer protection

4. **Secure Credentials Manager** (`dev/internal/security/credentials.go`)
   - Encrypted credential storage and retrieval
   - Access count and timeout limits
   - Comprehensive audit logging
   - Secure cleanup and memory protection

## Implementation Details

### Encryption Standards

- **Algorithm**: AES-256-GCM
- **Key Derivation**: SCRYPT (N=32768, r=8, p=1, keyLen=32)
- **Nonce Size**: 96 bits (12 bytes)
- **Authentication Tag**: 128 bits (16 bytes)
- **Salt**: 256 bits (32 bytes) cryptographically random

### Security Controls

- **Memory Protection**: Multiple-pass secure memory clearing
- **Access Limits**: Maximum 1000 credential accesses per session
- **Timeout Protection**: 1-hour maximum credential lifetime
- **Integrity Verification**: SHA-256 binary hash validation
- **Certificate Pinning**: Google APIs certificate validation
- **Audit Logging**: Comprehensive security event tracking

## Build Process Integration

### 1. Credential Encryption Tool

```bash
# Build the encryption tool
go build -o encrypt-credentials.exe dev/internal/security/tools/encrypt-credentials.go

# Encrypt service account credentials
encrypt-credentials.exe -input credentials.json -output encrypted.dat -verbose
```

### 2. Binary Hash Generation

```go
// Generate binary hash for integrity verification
hash, err := security.GenerateBinaryHash()
if err != nil {
    log.Fatalf("Failed to generate binary hash: %v", err)
}
```

### 3. Credential Embedding

The encrypted credentials and binary hash must be embedded in the source code during build:

```go
// Replace placeholders in dev/internal/security/credentials.go
var embeddedCredentials = `{encrypted_payload_json}`
var expectedBinaryHash = "actual_binary_hash_64_chars"
```

## Usage Examples

### Initializing Secure License Manager

```go
// Create license manager with secure credentials
manager, err := license.NewManager("license.dat")
if err != nil {
    return fmt.Errorf("failed to create license manager: %v", err)
}
defer manager.Close()

// Manager automatically uses encrypted credentials
// with integrity verification and certificate pinning
```

### Manual Credential Access

```go
// Create secure credentials manager
credMgr, err := security.NewSecureCredentialsManager()
if err != nil {
    return fmt.Errorf("failed to create credentials manager: %v", err)
}
defer credMgr.Close()

// Get secure credentials with full validation
ctx := context.Background()
credentials, err := credMgr.GetSecureCredentials(ctx)
if err != nil {
    return fmt.Errorf("failed to get credentials: %v", err)
}
defer credentials.Clear() // Always clear from memory

// Use credentials
credentialsJSON := credentials.Data()
// ... use with Google APIs ...
```

## Security Testing

### Running Security Tests

```bash
# Run all security tests
go test ./internal/security/... -v

# Run integration tests
go test ./internal/security/... -run TestSecurityIntegration -v

# Run benchmarks
go test ./internal/security/... -bench=. -v
```

### Test Coverage

- **Encryption/Decryption**: End-to-end workflow testing
- **Integrity Verification**: Binary hash validation
- **Certificate Pinning**: Google APIs connection security
- **Memory Protection**: Secure credential clearing
- **Tampering Detection**: Anti-tampering mechanisms
- **Audit Logging**: Security event tracking
- **Performance**: Encryption/decryption benchmarks

## Security Compliance

### OWASP ASVS Requirements

- **V2.1.1**: Strong authentication mechanisms ✓
- **V3.2.1**: Cryptographically secure random values ✓
- **V6.2.1**: Approved cryptographic algorithms ✓
- **V6.2.2**: Industry-proven cryptographic modes ✓
- **V6.2.3**: Cryptographically secure PRNGs ✓
- **V9.1.2**: Certificate pinning ✓
- **V9.2.1**: TLS security configuration ✓
- **V10.3.2**: Security event logging ✓

### Cryptographic Standards

- **NIST SP 800-132**: SCRYPT for key derivation
- **NIST SP 800-38D**: AES-GCM authenticated encryption
- **RFC 7539**: ChaCha20-Poly1305 cipher suites
- **RFC 5869**: HKDF key derivation (certificate pinning)

## Monitoring and Alerting

### Security Metrics

The system provides comprehensive security metrics:

```go
metrics := credentialsManager.GetSecurityMetrics()
// Returns: access_count, last_access, binary_hash_prefix,
//          encryption_version, certificate_pins, etc.
```

### Audit Events

All security events are logged with structured data:

- **initialization**: System startup and configuration
- **credentials_accessed**: Credential decryption events
- **integrity_check_failed**: Binary tampering detection
- **tampering_detected**: Anti-tampering triggers
- **credentials_rotated**: Credential update events
- **security_validation_failed**: Configuration errors

### Recommended Monitoring

1. **Failed Integrity Checks**: Immediate alert on tampering
2. **Excessive Access Attempts**: Rate limiting triggers
3. **Certificate Pinning Failures**: Google APIs connectivity issues
4. **Debug/VM Detection**: Unusual runtime environments
5. **Credential Rotation Events**: Authorized updates only

## Production Deployment

### Build operation Integration

1. **Credential Preparation**
   - Obtain Google service account credentials
   - Validate JSON structure and permissions
   - Store securely in build environment

2. **Encryption Phase**
   - Run credential encryption tool
   - Generate application-specific salt
   - Create encrypted payload

3. **Binary Preparation**
   - Calculate binary hash (post-compilation)
   - Embed encrypted credentials in source
   - Embed binary hash for verification

4. **Security Validation**
   - Test credential decryption
   - Verify integrity checking
   - Validate certificate pinning

5. **Final Build**
   - Compile with embedded credentials
   - Sign binary (recommended)
   - Package for distribution

### Security Considerations

- **Key Management**: Application salt should be unique per deployment
- **Binary Signing**: Use code signing for additional integrity
- **Secure Distribution**: Protect distribution channels
- **Update Process**: Implement secure credential rotation
- **Monitoring**: Deploy security event monitoring
- **Incident Response**: Plan for security breach scenarios

## Troubleshooting

### Common Issues

1. **Integrity Verification Failures**
   - Binary hash mismatch (rebuild required)
   - File corruption during distribution
   - Antivirus software modification

2. **Credential Decryption Failures**
   - Wrong application salt
   - Corrupted encrypted payload
   - Version mismatch

3. **Certificate Pinning Failures**
   - Google APIs certificate rotation
   - Network proxy interference
   - Firewall blocking connections

4. **Performance Issues**
   - SCRYPT parameters too high
   - Frequent integrity checks
   - Memory pressure

### Debug Mode

For development and troubleshooting:

```go
// Enable verbose logging
manager.SetLogLevel(slog.LevelDebug)

// Validate security configuration
err := manager.ValidateSecurityConfiguration()
if err != nil {
    log.Printf("Security validation failed: %v", err)
}

// Check security metrics
metrics := manager.GetSecurityMetrics()
log.Printf("Security metrics: %+v", metrics)
```

## Maintenance

### Regular Tasks

1. **Certificate Pin Updates**: Monitor Google APIs certificates
2. **Security Testing**: Regular penetration testing
3. **Dependency Updates**: Keep cryptographic libraries current
4. **Log Analysis**: Review security audit logs
5. **Performance Monitoring**: Track encryption/decryption times

### Security Updates

1. **Credential Rotation**: Implement periodic rotation
2. **Algorithm Updates**: Monitor for cryptographic advances
3. **Pin Updates**: Update certificate pins as needed
4. **Binary Re-signing**: Refresh code signatures

---

## Conclusion

This secure credential system provides enterprise-grade protection for embedded service account credentials while maintaining the self-contained deployment model. The implementation follows security best practices and provides comprehensive monitoring and audit capabilities.

For questions or issues, refer to the test suite in `dev/internal/security/*_test.go` for implementation examples and expected behavior.