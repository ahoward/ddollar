# Data Model: Automatic SSL Termination

**Feature**: 002-automatic-ssl-termination
**Date**: 2025-10-18

## Overview

This document defines the data structures and relationships for certificate authority management and SSL certificate operations in ddollar. The model focuses on CA lifecycle, certificate generation, and trust installation state tracking.

---

## Entity: Certificate Authority (CA)

### Description
Represents the local Certificate Authority created and managed by ddollar. The CA is used to sign all leaf certificates for AI provider domains. One CA per user installation.

### Attributes

| Attribute | Type | Description | Validation |
|-----------|------|-------------|------------|
| `rootCAPath` | string (file path) | Path to CA certificate PEM file | Must exist, readable |
| `rootCAKeyPath` | string (file path) | Path to CA private key PEM file | Must exist, 0600 permissions |
| `created` | timestamp | CA creation time | ISO 8601 format |
| `fingerprint` | string (hex) | SHA-256 fingerprint of CA cert | 64 hex characters |
| `commonName` | string | CA certificate common name | "ddollar Local CA" |
| `validFrom` | timestamp | Certificate not-before date | ISO 8601 format |
| `validUntil` | timestamp | Certificate not-after date | ISO 8601 format |

### Storage Location

```
~/.ddollar/ca/
├── rootCA.pem       # CA certificate (public, 0644)
└── rootCA-key.pem   # CA private key (0600)
```

### File Format

**rootCA.pem**: PEM-encoded X.509 certificate
```
-----BEGIN CERTIFICATE-----
MIIDMzCCAhugAwIBAgI...
-----END CERTIFICATE-----
```

**rootCA-key.pem**: PEM-encoded RSA private key (2048-bit)
```
-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA...
-----END RSA PRIVATE KEY-----
```

### State Transitions

```
[Does Not Exist]
    ↓ (GenerateCA)
[Created, Not Trusted]
    ↓ (InstallTrust)
[Created, Trusted]
    ↓ (UninstallTrust)
[Created, Not Trusted]
    ↓ (RemoveCA)
[Does Not Exist]
```

### Relationships
- **Signs**: 0..n Leaf Certificates
- **Trusted By**: 0..n Trust Installations (one per platform trust store)

---

## Entity: Leaf Certificate

### Description
SSL/TLS certificate signed by ddollar's CA, covering all AI provider domains. Used by the HTTPS proxy server to terminate SSL connections.

### Attributes

| Attribute | Type | Description | Validation |
|-----------|------|-------------|------------|
| `certPath` | string (file path) | Path to certificate PEM file | Must exist, readable |
| `keyPath` | string (file path) | Path to private key PEM file | Must exist, 0600 permissions |
| `domains` | []string | Subject Alternative Names (SANs) | Non-empty array |
| `created` | timestamp | Certificate generation time | ISO 8601 format |
| `validFrom` | timestamp | Certificate not-before date | ISO 8601 format |
| `validUntil` | timestamp | Certificate not-after date | ISO 8601 format |
| `fingerprint` | string (hex) | SHA-256 fingerprint of cert | 64 hex characters |
| `issuer` | string | CA common name that signed this cert | Must match CA commonName |

### Storage Location

```
~/.ddollar/certs/
├── cert.pem         # Leaf certificate (public, 0644)
└── key.pem          # Leaf private key (0600)
```

### Covered Domains

Default domains included as SANs:
- `api.openai.com`
- `api.anthropic.com`
- `api.cohere.ai`
- `generativelanguage.googleapis.com`
- `localhost`

### File Format

**cert.pem**: PEM-encoded X.509 certificate with multiple SANs
```
-----BEGIN CERTIFICATE-----
MIIDRjCCAi6gAwIBAgIQF...
-----END CERTIFICATE-----
```

**key.pem**: PEM-encoded RSA private key (2048-bit)
```
-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAx...
-----END RSA PRIVATE KEY-----
```

### State Transitions

```
[Does Not Exist]
    ↓ (GenerateCertificate)
[Generated, Not Used]
    ↓ (ProxyStart)
[Generated, In Use by Proxy]
    ↓ (ProxyStop)
[Generated, Not Used]
    ↓ (Regenerate)
[Generated, Not Used]
```

### Relationships
- **Signed By**: 1 Certificate Authority
- **Used By**: 0..1 Proxy Server (at most one active proxy)

---

## Entity: Trust Installation

### Description
Represents the installation state of the CA certificate in a specific platform trust store. Tracks whether ddollar's CA is trusted by the system or application.

### Attributes

| Attribute | Type | Description | Validation |
|-----------|------|-------------|------------|
| `platform` | enum | Trust store platform | {system, nss} |
| `osType` | enum | Operating system | {macos, linux, windows} |
| `trustStorePath` | string | Path to trust store | Platform-specific |
| `installed` | boolean | Whether CA is currently installed | true/false |
| `installedAt` | timestamp | When trust was installed | ISO 8601 format, null if not installed |
| `lastChecked` | timestamp | Last verification time | ISO 8601 format |
| `error` | string | Last installation error (if any) | Empty if successful |

### Platform-Specific Trust Store Paths

| OS | Trust Store | Path | Command |
|----|-------------|------|---------|
| macOS | System Keychain | `/Library/Keychains/System.keychain` | `security add-trusted-cert` |
| Linux (Debian/Ubuntu) | ca-certificates | `/usr/local/share/ca-certificates/` | `update-ca-certificates` |
| Linux (RHEL/Fedora) | ca-trust | `/etc/pki/ca-trust/source/anchors/` | `update-ca-trust` |
| Windows | Cert Store (ROOT) | `Cert:\LocalMachine\Root` | `certutil -addstore ROOT` |
| Firefox/NSS | NSS DB | `~/.pki/nssdb/` | `certutil -A -n ddollar -t "C,,"` |

### State Transitions

```
[Not Installed]
    ↓ (Install)
[Installed Successfully]
    ↓ (Verify)
[Verified Trusted]
    ↓ (Uninstall)
[Not Installed]

[Not Installed]
    ↓ (Install Fails)
[Installation Failed]
    ↓ (Retry/Manual)
[Installed Successfully]
```

### Relationships
- **Trusts**: 1 Certificate Authority
- **Managed By**: 1 System (OS or application like Firefox)

---

## Enumerated Types

### TrustStoreType

```go
type TrustStoreType string

const (
    TrustStoreSystem TrustStoreType = "system"  // OS system trust store
    TrustStoreNSS    TrustStoreType = "nss"     // NSS (Firefox, Chromium snap)
)
```

### OSType

```go
type OSType string

const (
    OSTypeMacOS   OSType = "macos"
    OSTypeLinux   OSType = "linux"
    OSTypeWindows OSType = "windows"
)
```

### InstallationStatus

```go
type InstallationStatus string

const (
    StatusNotInstalled InstallationStatus = "not_installed"
    StatusInstalled    InstallationStatus = "installed"
    StatusFailed       InstallationStatus = "failed"
    StatusVerified     InstallationStatus = "verified"
)
```

---

## Struct Definitions (Go Implementation)

### CA Struct

```go
package proxy

import (
    "crypto/sha256"
    "crypto/x509"
    "encoding/hex"
    "encoding/pem"
    "os"
    "path/filepath"
    "time"
)

type CA struct {
    RootCAPath    string
    RootCAKeyPath string
    Created       time.Time
    Fingerprint   string
    CommonName    string
    ValidFrom     time.Time
    ValidUntil    time.Time
}

// NewCA loads or creates CA from standard location
func NewCA() (*CA, error) {
    caDir := filepath.Join(os.UserHomeDir(), ".ddollar", "ca")
    certPath := filepath.Join(caDir, "rootCA.pem")
    keyPath := filepath.Join(caDir, "rootCA-key.pem")

    // Check if CA already exists
    if _, err := os.Stat(certPath); err == nil {
        return loadCA(certPath, keyPath)
    }

    // Generate new CA
    return generateCA(caDir)
}

// Fingerprint calculates SHA-256 fingerprint
func (ca *CA) Fingerprint() (string, error) {
    certPEM, err := os.ReadFile(ca.RootCAPath)
    if err != nil {
        return "", err
    }

    block, _ := pem.Decode(certPEM)
    hash := sha256.Sum256(block.Bytes)
    return hex.EncodeToString(hash[:]), nil
}

// IsValid checks if CA certificate is within validity period
func (ca *CA) IsValid() bool {
    now := time.Now()
    return now.After(ca.ValidFrom) && now.Before(ca.ValidUntil)
}
```

### Certificate Struct

```go
package proxy

type Certificate struct {
    CertPath     string
    KeyPath      string
    Domains      []string
    Created      time.Time
    ValidFrom    time.Time
    ValidUntil   time.Time
    Fingerprint  string
    Issuer       string
}

// NewCertificate generates leaf certificate from CA
func NewCertificate(ca *CA, domains []string) (*Certificate, error) {
    certsDir := filepath.Join(os.UserHomeDir(), ".ddollar", "certs")
    certPath := filepath.Join(certsDir, "cert.pem")
    keyPath := filepath.Join(certsDir, "key.pem")

    // Generate certificate using mkcert
    certPEM, keyPEM, err := ca.MakeCert(domains)
    if err != nil {
        return nil, err
    }

    // Write to files
    if err := os.WriteFile(certPath, certPEM, 0644); err != nil {
        return nil, err
    }
    if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
        return nil, err
    }

    return &Certificate{
        CertPath:  certPath,
        KeyPath:   keyPath,
        Domains:   domains,
        Created:   time.Now(),
        Issuer:    ca.CommonName,
    }, nil
}
```

### TrustInstallation Struct

```go
package proxy

type TrustInstallation struct {
    Platform       TrustStoreType
    OSType         OSType
    TrustStorePath string
    Installed      bool
    InstalledAt    time.Time
    LastChecked    time.Time
    Error          string
}

// Install attempts to install CA to trust store
func (ti *TrustInstallation) Install(ca *CA) error {
    // Platform-specific installation logic
    switch ti.OSType {
    case OSTypeMacOS:
        return ti.installMacOS(ca)
    case OSTypeLinux:
        return ti.installLinux(ca)
    case OSTypeWindows:
        return ti.installWindows(ca)
    }
    return fmt.Errorf("unsupported OS: %s", ti.OSType)
}

// Verify checks if CA is actually trusted
func (ti *TrustInstallation) Verify(ca *CA) error {
    // Verify CA fingerprint exists in trust store
    ti.LastChecked = time.Now()
    // ... platform-specific verification
    return nil
}
```

---

## Data Validation Rules

### CA Validation

1. **File Existence**: Both `rootCA.pem` and `rootCA-key.pem` must exist
2. **File Permissions**: Private key must have 0600 permissions
3. **PEM Format**: Both files must be valid PEM-encoded
4. **Certificate Validity**: Must be within valid date range
5. **Key Pair Match**: Public key in cert must match private key

### Certificate Validation

1. **File Existence**: Both `cert.pem` and `key.pem` must exist
2. **File Permissions**: Private key must have 0600 permissions
3. **Issuer Match**: Certificate must be signed by ddollar CA
4. **Domain Coverage**: Must include all required AI provider domains
5. **Validity Period**: Must be within valid date range

### Trust Installation Validation

1. **Platform Detection**: Must correctly identify OS and trust store type
2. **Permission Check**: Must verify sudo/admin before attempting install
3. **Idempotency**: Installing already-trusted CA should be no-op
4. **Verification**: After install, must verify CA is actually trusted

---

## Persistence Strategy

### Filesystem-Based

All data persisted as files in user directory:

```
~/.ddollar/
├── ca/
│   ├── rootCA.pem       # CA certificate
│   └── rootCA-key.pem   # CA private key
├── certs/
│   ├── cert.pem         # Leaf certificate
│   └── key.pem          # Leaf private key
└── .trust-state.json    # Trust installation state (optional)
```

### Optional State File

Trust state can be cached in JSON for faster status checks:

```json
{
  "ca_fingerprint": "abc123...",
  "trust_installations": [
    {
      "platform": "system",
      "os_type": "linux",
      "installed": true,
      "installed_at": "2025-10-18T12:00:00Z",
      "last_checked": "2025-10-18T12:00:00Z"
    },
    {
      "platform": "nss",
      "os_type": "linux",
      "installed": true,
      "installed_at": "2025-10-18T12:00:05Z",
      "last_checked": "2025-10-18T12:00:05Z"
    }
  ]
}
```

**Note**: State file is optional - trust can always be verified by checking actual trust stores.

---

## Error States and Recovery

### CA Generation Failure

**Causes**: Filesystem permissions, disk full, crypto library error
**Recovery**:
- Check `~/.ddollar/ca/` permissions
- Verify disk space
- Retry CA generation
- If persists, report error with diagnostic info

### Trust Installation Failure

**Causes**: Insufficient permissions, locked trust store, unsupported platform
**Recovery**:
- Detect failure reason (permission vs platform)
- Provide platform-specific manual instructions
- Continue proxy operation with warning
- Allow retry with `ddollar trust` command

### Certificate Generation Failure

**Causes**: Missing CA, CA private key inaccessible, crypto error
**Recovery**:
- Verify CA exists and is valid
- Check CA private key permissions (0600)
- Regenerate CA if corrupted
- Retry certificate generation

### Certificate Expiration

**Causes**: Certificate validity period (365 days) expired
**Recovery**:
- Detect expiration on proxy startup
- Automatically regenerate certificate
- No user action required (transparent renewal)

---

## Relationships Diagram

```
┌─────────────────────┐
│ Certificate         │
│ Authority (CA)      │
│ ~/.ddollar/ca/      │
└──────────┬──────────┘
           │
           │ signs
           │
           ↓
┌─────────────────────┐
│ Leaf Certificate    │
│ ~/.ddollar/certs/   │
│ (AI provider domains│
└─────────────────────┘
           │
           │ used by
           │
           ↓
┌─────────────────────┐
│ HTTPS Proxy Server  │
│ (src/proxy/server)  │
└─────────────────────┘

┌─────────────────────┐
│ Trust Installation  │
│ (system store)      │
└──────────┬──────────┘
           │
           │ trusts
           │
           ↓
┌─────────────────────┐
│ Certificate         │
│ Authority (CA)      │
└─────────────────────┘

┌─────────────────────┐
│ Trust Installation  │
│ (NSS store)         │
└──────────┬──────────┘
           │
           │ trusts
           │
           ↓
┌─────────────────────┐
│ Certificate         │
│ Authority (CA)      │
└─────────────────────┘
```

---

## Data Model Summary

**Entities**: 3 (CA, Certificate, TrustInstallation)
**Storage**: Filesystem-based (PEM files)
**State Tracking**: Optional JSON file + trust store verification
**Validation**: File permissions, PEM format, validity periods, signature verification
**Relationships**: CA signs Certificates, TrustInstallations trust CA
**Error Recovery**: Graceful degradation with clear error messages and retry paths
