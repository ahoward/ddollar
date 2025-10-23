# Contract: Certificate Generation

**Feature**: 002-automatic-ssl-termination
**Component**: `src/proxy/cert.go` (refactored)
**Date**: 2025-10-18

## Overview

This contract defines the API for generating SSL/TLS leaf certificates signed by ddollar's CA. These certificates are used by the HTTPS proxy to terminate SSL connections for AI provider domains.

---

## Function: GenerateCert

### Purpose
Generate SSL certificate for AI provider domains, signed by ddollar's CA. This is the main entry point called by proxy server initialization.

### Signature

```go
func GenerateCert() (certPath, keyPath string, err error)
```

**Note**: This function signature is preserved from existing `cert.go` for backward compatibility. Internal implementation now uses mkcert.

### Behavior

1. Get or create CA using `EnsureCA()`
2. Define domains to cover (AI provider domains + localhost)
3. Generate certificate using mkcert's `MakeCert()` function
4. Write certificate and private key to `~/.ddollar/certs/`
5. Set appropriate file permissions (cert: 0644, key: 0600)
6. Return paths to generated files

### Input
- None (uses default domain list and paths)

### Output

**Success**:
```go
certPath = "~/.ddollar/certs/cert.pem"
keyPath = "~/.ddollar/certs/key.pem"
return certPath, keyPath, nil
```

**Error Cases**:
- CA doesn't exist/can't be created: `error: "failed to initialize CA: ..."`
- Certificate generation failed: `error: "failed to generate certificate: ..."`
- Filesystem write failed: `error: "failed to write certificate files: ..."`
- Permission denied: `error: "cannot write to ~/.ddollar/certs/: permission denied"`

### Covered Domains

Default domains included as Subject Alternative Names (SANs):
```go
domains := []string{
    "api.openai.com",
    "api.anthropic.com",
    "api.cohere.ai",
    "generativelanguage.googleapis.com",
    "localhost",
}
```

### File Format

**cert.pem**: Multi-domain certificate
```
Certificate:
    Data:
        Version: 3 (0x2)
        Serial Number: [random]
        Signature Algorithm: sha256WithRSAEncryption
        Issuer: CN=ddollar Local CA
        Validity:
            Not Before: [timestamp]
            Not After:  [timestamp + 365 days]
        Subject: CN=api.openai.com
        X509v3 extensions:
            X509v3 Subject Alternative Name:
                DNS:api.openai.com
                DNS:api.anthropic.com
                DNS:api.cohere.ai
                DNS:generativelanguage.googleapis.com
                DNS:localhost
```

**key.pem**: RSA-2048 private key
```
-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA...
-----END RSA PRIVATE KEY-----
```

### Side Effects
- Creates `~/.ddollar/certs/` directory if it doesn't exist
- Writes `cert.pem` (mode 0644) and `key.pem` (mode 0600)
- Overwrites existing certificate files

### Example Usage

```go
// In proxy/server.go initialization:
certPath, keyPath, err := GenerateCert()
if err != nil {
    return fmt.Errorf("failed to setup SSL: %w", err)
}

// Load for TLS config
cert, err := tls.LoadX509KeyPair(certPath, keyPath)
if err != nil {
    return fmt.Errorf("failed to load certificate: %w", err)
}

tlsConfig := &tls.Config{
    Certificates: []tls.Certificate{cert},
}
```

### Testing

**Unit Test**:
```go
func TestGenerateCert_CreatesValidCertificate(t *testing.T) {
    // Setup
    testDir := t.TempDir()
    os.Setenv("HOME", testDir)

    // Execute
    certPath, keyPath, err := GenerateCert()

    // Assert
    assert.NoError(t, err)
    assert.FileExists(t, certPath)
    assert.FileExists(t, keyPath)

    // Verify cert is valid
    cert, err := tls.LoadX509KeyPair(certPath, keyPath)
    assert.NoError(t, err)
    assert.NotNil(t, cert)
}

func TestGenerateCert_CoversAllDomains(t *testing.T) {
    certPath, _, _ := GenerateCert()

    // Load and parse certificate
    certPEM, _ := os.ReadFile(certPath)
    block, _ := pem.Decode(certPEM)
    cert, _ := x509.ParseCertificate(block.Bytes)

    // Assert all domains present
    expectedDomains := []string{
        "api.openai.com",
        "api.anthropic.com",
        "api.cohere.ai",
        "generativelanguage.googleapis.com",
        "localhost",
    }

    for _, domain := range expectedDomains {
        assert.Contains(t, cert.DNSNames, domain)
    }
}
```

**Integration Test**:
```go
func TestGenerateCert_WithRealCA(t *testing.T) {
    // Ensure CA exists
    ca, err := EnsureCA()
    assert.NoError(t, err)

    // Generate cert
    certPath, keyPath, err := GenerateCert()
    assert.NoError(t, err)

    // Verify cert is signed by CA
    certPEM, _ := os.ReadFile(certPath)
    caPEM, _ := os.ReadFile(ca.RootCAPath)

    roots := x509.NewCertPool()
    roots.AppendCertsFromPEM(caPEM)

    block, _ := pem.Decode(certPEM)
    cert, _ := x509.ParseCertificate(block.Bytes)

    opts := x509.VerifyOptions{Roots: roots}
    _, err = cert.Verify(opts)
    assert.NoError(t, err) // Cert chain validates
}
```

---

## Function: RegenerateCert

### Purpose
Force regeneration of certificate (e.g., after expiration, domain list change, or CA rotation).

### Signature

```go
func RegenerateCert() error
```

### Behavior

1. Remove existing certificate files if present
2. Call `GenerateCert()` to create new certificate
3. Return error if regeneration fails

### Input
- None

### Output

**Success**:
```go
return nil  // New certificate generated
```

**Error Cases**:
- Same as `GenerateCert()` errors

### Side Effects
- Deletes existing `~/.ddollar/certs/cert.pem` and `key.pem`
- Creates new certificate files

### Example Usage

```go
// Check if cert is expiring soon
if cert.IsExpiringSoon() {
    log.Println("Certificate expiring soon, regenerating...")
    if err := RegenerateCert(); err != nil {
        log.Printf("Warning: Failed to regenerate: %v", err)
    }
}
```

### Testing

```go
func TestRegenerateCert_ReplacesExisting(t *testing.T) {
    // Generate initial cert
    GenerateCert()
    initialPath := filepath.Join(os.UserHomeDir(), ".ddollar/certs/cert.pem")
    initialModTime := fileModTime(initialPath)

    // Wait to ensure timestamp difference
    time.Sleep(100 * time.Millisecond)

    // Regenerate
    err := RegenerateCert()
    assert.NoError(t, err)

    // Verify new file created
    newModTime := fileModTime(initialPath)
    assert.True(t, newModTime.After(initialModTime))
}
```

---

## Function: ValidateCert

### Purpose
Verify existing certificate is valid and covers required domains. Used for health checks.

### Signature

```go
func ValidateCert(certPath string) error
```

### Behavior

1. Load certificate from file
2. Parse X.509 certificate
3. Check validity period (not expired)
4. Verify all required domains are covered
5. Verify certificate is signed by ddollar CA

### Input
- `certPath`: Path to certificate file

### Output

**Success**:
```go
return nil  // Certificate is valid
```

**Error Cases**:
- File not found: `error: "certificate file not found"`
- Invalid PEM: `error: "certificate is not valid PEM format"`
- Expired: `error: "certificate expired on [date]"`
- Missing domains: `error: "certificate does not cover required domains: [list]"`
- Invalid signature: `error: "certificate not signed by ddollar CA"`

### Example Usage

```go
// Before starting proxy, validate cert
certPath := filepath.Join(os.UserHomeDir(), ".ddollar/certs/cert.pem")

if err := ValidateCert(certPath); err != nil {
    log.Printf("Certificate invalid: %v", err)
    log.Println("Regenerating certificate...")
    GenerateCert()
}
```

### Testing

```go
func TestValidateCert_AcceptsValidCert(t *testing.T) {
    certPath, _, _ := GenerateCert()

    err := ValidateCert(certPath)
    assert.NoError(t, err)
}

func TestValidateCert_RejectsExpiredCert(t *testing.T) {
    // Create expired cert (mock)
    expiredCertPath := createExpiredCert(t)

    err := ValidateCert(expiredCertPath)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "expired")
}

func TestValidateCert_RejectsMissingDomains(t *testing.T) {
    // Create cert with incomplete domain list
    incompleteCertPath := createCertWithDomains([]string{"localhost"})

    err := ValidateCert(incompleteCertPath)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "does not cover required domains")
}
```

---

## Function: GetCertInfo

### Purpose
Extract certificate metadata for display (status command, debugging).

### Signature

```go
func GetCertInfo(certPath string) (*CertInfo, error)
```

### Return Type

```go
type CertInfo struct {
    Domains      []string
    ValidFrom    time.Time
    ValidUntil   time.Time
    Issuer       string
    Fingerprint  string
    DaysRemaining int
}
```

### Behavior

1. Load and parse certificate
2. Extract relevant metadata
3. Calculate days until expiration
4. Return structured info

### Input
- `certPath`: Path to certificate file

### Output

**Success**:
```go
info := &CertInfo{
    Domains: []string{"api.openai.com", "api.anthropic.com", ...},
    ValidFrom: time.Parse(...),
    ValidUntil: time.Parse(...),
    Issuer: "ddollar Local CA",
    Fingerprint: "abc123...",
    DaysRemaining: 342,
}
return info, nil
```

**Error Cases**:
- Same as `ValidateCert()` errors

### Example Usage

```go
// In `ddollar status` command
certPath := filepath.Join(os.UserHomeDir(), ".ddollar/certs/cert.pem")

info, err := GetCertInfo(certPath)
if err != nil {
    fmt.Printf("Certificate: Not found\n")
    return
}

fmt.Printf("Certificate:\n")
fmt.Printf("  Domains: %s\n", strings.Join(info.Domains, ", "))
fmt.Printf("  Valid until: %s (%d days remaining)\n",
    info.ValidUntil.Format("2006-01-02"), info.DaysRemaining)
fmt.Printf("  Issuer: %s\n", info.Issuer)
fmt.Printf("  Fingerprint: %s\n", info.Fingerprint[:16]+"...")
```

### Testing

```go
func TestGetCertInfo_ReturnsCorrectMetadata(t *testing.T) {
    certPath, _, _ := GenerateCert()

    info, err := GetCertInfo(certPath)
    assert.NoError(t, err)

    assert.Contains(t, info.Domains, "api.openai.com")
    assert.Equal(t, "ddollar Local CA", info.Issuer)
    assert.True(t, info.DaysRemaining > 0 && info.DaysRemaining <= 365)
}
```

---

## Integration with Proxy Server

### Initialization Sequence

```go
// In src/proxy/server.go Start() function

func (s *Server) Start() error {
    // 1. Generate or load certificate
    certPath, keyPath, err := GenerateCert()
    if err != nil {
        return fmt.Errorf("SSL setup failed: %w", err)
    }

    // 2. Validate certificate before use
    if err := ValidateCert(certPath); err != nil {
        log.Printf("Certificate validation failed: %v", err)
        log.Println("Attempting to regenerate...")
        if err := RegenerateCert(); err != nil {
            return fmt.Errorf("certificate regeneration failed: %w", err)
        }
        certPath, keyPath, _ = GenerateCert()
    }

    // 3. Load certificate for TLS
    cert, err := tls.LoadX509KeyPair(certPath, keyPath)
    if err != nil {
        return fmt.Errorf("failed to load certificate: %w", err)
    }

    // 4. Configure TLS
    s.httpServer.TLSConfig = &tls.Config{
        Certificates: []tls.Certificate{cert},
    }

    // 5. Start server
    return s.httpServer.ListenAndServeTLS("", "")
}
```

---

## Error Handling Strategy

### Certificate Generation Failure

```go
certPath, keyPath, err := GenerateCert()
if err != nil {
    // This is critical - proxy can't start without cert
    return fmt.Errorf("cannot start proxy: %w", err)
}
```

### Certificate Validation Failure (Recoverable)

```go
if err := ValidateCert(certPath); err != nil {
    // Non-critical - attempt regeneration
    log.Printf("Certificate invalid: %v", err)

    if err := RegenerateCert(); err != nil {
        // Still critical if regeneration fails
        return fmt.Errorf("certificate regeneration failed: %w", err)
    }

    log.Println("✓ Certificate regenerated successfully")
}
```

---

## Contract Summary

**Functions**: 4 (GenerateCert, RegenerateCert, ValidateCert, GetCertInfo)
**Primary Entry Point**: `GenerateCert()` (called by proxy server)
**Certificate Validity**: 365 days (mkcert default)
**Covered Domains**: 4 AI providers + localhost
**File Permissions**: cert.pem (0644), key.pem (0600)
**Error Strategy**: Critical failures block proxy start, validation failures trigger regeneration
**Backward Compatibility**: Preserves existing `GenerateCert()` signature
