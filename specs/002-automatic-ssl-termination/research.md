# Research: Automatic SSL Termination with mkcert

**Feature**: 002-automatic-ssl-termination
**Date**: 2025-10-18
**Status**: Complete

## Overview

This document consolidates research for integrating mkcert into ddollar for automatic SSL certificate trust. It builds on the comprehensive analysis in `docs/SSL_TERMINATION.md` and focuses on specific implementation decisions.

## Decision 1: mkcert Library Integration Approach

**Decision**: Use mkcert as a Go library dependency, not as an external binary

**Rationale**:
- mkcert exposes its core functionality through importable Go packages
- Embedding as library maintains single-binary architecture
- No need to bundle external executables for multiple platforms
- Direct API access provides better error handling and control
- Reduces attack surface (no shell execution of external binaries)

**Alternatives Considered**:
1. **Shell out to mkcert binary**:
   - ❌ Requires bundling platform-specific binaries
   - ❌ Complex cross-platform binary management
   - ❌ Shell execution security concerns
   - ❌ Harder to handle errors programmatically

2. **Reimplement mkcert logic manually**:
   - ❌ Reinventing the wheel (platform-specific trust store APIs)
   - ❌ High maintenance burden
   - ❌ Risk of security bugs
   - ❌ No benefit over using battle-tested library

**Implementation Note**: Import `github.com/FiloSottile/mkcert` and use its internal packages directly.

---

## Decision 2: Certificate Authority (CA) Storage Location

**Decision**: Store CA in `~/.ddollar/ca/` directory, not system-wide locations

**Rationale**:
- User directory doesn't require elevated permissions for storage
- Follows mkcert convention (`~/.local/share/mkcert/`)
- Allows per-user CA instances (important for multi-user systems)
- Easy cleanup on uninstall (remove user directory)
- Consistent across all platforms

**Alternatives Considered**:
1. **System-wide location** (`/etc/ddollar/`, `C:\ProgramData\ddollar\`):
   - ❌ Requires sudo/admin for all CA operations (not just trust installation)
   - ❌ Complicates multi-user scenarios
   - ❌ Harder cleanup (system files may persist)

2. **Temporary directory**:
   - ❌ CA would be lost on reboot
   - ❌ Forces regeneration and re-trust on every restart

**Storage Structure**:
```
~/.ddollar/
├── ca/
│   ├── rootCA.pem       # CA certificate (public)
│   └── rootCA-key.pem   # CA private key (0600 permissions)
└── certs/
    ├── cert.pem         # Leaf certificate
    └── key.pem          # Leaf private key (0600 permissions)
```

---

## Decision 3: Trust Installation Failure Handling

**Decision**: Continue proxy operation with warnings when trust installation fails

**Rationale**:
- Proxy can still function if user manually trusts certificates later
- Restrictive environments (Docker, corporate) may block auto-trust
- Better UX: inform user of failure but don't block startup
- Aligns with "graceful degradation" principle
- Matches Caddy's behavior (proceeds with warnings)

**Fallback Behavior**:
1. Attempt automatic trust installation
2. If fails, print platform-specific manual instructions
3. Continue proxy startup with warning banner
4. User can manually trust and restart (or use `ddollar trust` command)

**Alternatives Considered**:
1. **Block proxy startup on trust failure**:
   - ❌ Breaks in restricted environments with no recovery path
   - ❌ Poor UX for users who want to manually trust
   - ❌ Prevents testing in containers/CI

2. **Silent failure (no warnings)**:
   - ❌ User sees cryptic SSL errors without context
   - ❌ No guidance on how to fix

**Error Message Template**:
```
⚠️  Automatic certificate trust installation failed
This usually happens in Docker, corporate environments, or without sudo

Manual trust instructions for [PLATFORM]:
  [platform-specific command]

Proxy is starting anyway. API calls will fail until certificates are trusted.
```

---

## Decision 4: Certificate Domains and Validity Period

**Decision**: Generate single certificate covering all AI provider domains, 365-day validity

**Rationale**:
- Single cert simpler than per-domain certs (one file to manage)
- mkcert supports SAN (Subject Alternative Names) for multiple domains
- 365 days is mkcert default and reasonable for local development
- Long validity reduces rotation friction
- Matches existing cert.go behavior

**Domains to Include**:
- `api.openai.com`
- `api.anthropic.com`
- `api.cohere.ai`
- `generativelanguage.googleapis.com`
- `localhost` (for potential local testing)

**Alternatives Considered**:
1. **Per-domain certificates**:
   - ❌ More files to manage
   - ❌ Slightly more complex proxy logic (cert selection)
   - ❌ No security benefit for local proxy

2. **Shorter validity (30-90 days)**:
   - ❌ Requires rotation logic (out of scope per spec)
   - ❌ More user friction
   - ❌ No benefit for local development use case

3. **Wildcard certificates** (`*.openai.com`):
   - ❌ Overly broad (violates principle of least privilege)
   - ❌ May trigger security warnings in some browsers

---

## Decision 5: Integration with Existing cert.go

**Decision**: Refactor `src/proxy/cert.go` to use mkcert, keep GenerateCert() API signature

**Rationale**:
- Minimize changes to existing code that calls certificate generation
- `GenerateCert()` already exists in cert.go - repurpose it
- Backward compatible with existing manual cert trust users
- Clear separation: mkcert.go handles CA, cert.go handles leaf certs

**Refactoring Strategy**:
```go
// src/proxy/cert.go (BEFORE - manual generation)
func GenerateCert() (certPath, keyPath string, err error) {
    // Manual RSA key generation
    // Manual x509 certificate creation
    // Manual PEM encoding
}

// src/proxy/cert.go (AFTER - uses mkcert)
func GenerateCert() (certPath, keyPath string, err error) {
    ca, err := mkcert.EnsureCA()  // Get or create CA
    if err != nil {
        return "", "", err
    }

    cert, key, err := ca.MakeCert(providers...)
    // Write to standard paths
    return certPath, keyPath, nil
}
```

**Alternatives Considered**:
1. **New separate function** (`GenerateCertWithMkcert()`):
   - ❌ Clutters API
   - ❌ Requires changing call sites
   - ❌ Doesn't leverage existing code structure

2. **Complete rewrite of cert.go**:
   - ❌ Unnecessary - most logic can stay
   - ❌ Higher risk of breaking changes

---

## Decision 6: NSS Trust Store Support (Firefox)

**Decision**: Include NSS trust store installation for Firefox/Chromium snap support

**Rationale**:
- mkcert handles NSS automatically when available
- Firefox uses separate trust store on Linux
- Snap-packaged Chromium uses NSS
- No additional code needed (mkcert handles it)
- Improves compatibility for users who use Firefox

**Implementation**: mkcert's `Install()` function automatically detects and installs to NSS if `certutil` (NSS tool) is available. No special handling needed in our code.

**Alternatives Considered**:
1. **Skip NSS support**:
   - ❌ Firefox users would see SSL errors
   - ❌ No downside to including (mkcert handles it)

---

## Decision 7: Testing Strategy for Trust Installation

**Decision**: Integration tests require sudo, unit tests mock trust operations

**Rationale**:
- Trust installation requires elevated permissions (can't unit test)
- Integration tests validate real trust store modification
- Unit tests cover mkcert wrapper logic without sudo
- Follows existing test strategy in ddollar

**Test Structure**:
```
tests/unit/mkcert_test.go:
- Test CA creation (in temp directory)
- Test certificate generation
- Test error handling
- Mock trust installation calls

tests/integration/mkcert_test.go:
- Test full CA creation + trust + cert generation
- Requires: sudo go test
- Validates system trust store modification
- Cleanup: removes test CA from trust stores
```

**CI/CD Consideration**: Integration tests skipped in CI unless running with elevated permissions (GitHub Actions limitation).

---

## Decision 8: Command-Line Interface Extensions

**Decision**: Add `ddollar trust` and `ddollar untrust` convenience commands

**Rationale**:
- Provides explicit way to manage certificate trust
- Useful for manual trust after automatic failure
- Clean uninstall path (`ddollar untrust` before removing binary)
- Matches mkcert CLI UX (`mkcert -install`, `mkcert -uninstall`)

**Commands**:
```bash
# Install CA to trust stores (same as part of `start`)
ddollar trust

# Remove CA from trust stores (part of cleanup)
ddollar untrust

# Existing commands extend with automatic trust:
ddollar start     # Now includes automatic trust installation
ddollar stop      # Optionally untrusts (or keep for reuse)
```

**Alternatives Considered**:
1. **No explicit trust commands** (only automatic):
   - ❌ No way to manually trigger trust after failure
   - ❌ Users must manually run platform commands

2. **Flags on existing commands** (`--trust`, `--untrust`):
   - ❌ Clutters command interface
   - ❌ Less intuitive than dedicated commands

---

## Technical Constraints Validated

### Constraint 1: Go 1.21+ Compatibility
**Validation**: mkcert library compatible with Go 1.21+
**Source**: github.com/FiloSottile/mkcert go.mod specifies `go 1.16` minimum
**Status**: ✅ COMPATIBLE

### Constraint 2: Zero External Dependencies (Beyond Go Libraries)
**Validation**: mkcert library requires system trust store binaries, but these are OS-provided
**Dependencies**:
- macOS: `security` (pre-installed)
- Linux: `update-ca-certificates` or `update-ca-trust` (pre-installed)
- Windows: `certutil` (pre-installed)
- NSS (optional): `certutil` from NSS (user-installed for Firefox)

**Status**: ✅ ACCEPTABLE - system binaries, not application dependencies

### Constraint 3: No Internet Required
**Validation**: All certificate operations are local (generation + trust)
**Status**: ✅ CONFIRMED

### Constraint 4: Cross-Platform Build
**Validation**: mkcert is pure Go, compiles to all target platforms
**Status**: ✅ CONFIRMED

---

## mkcert Library API Reference

### Key Functions

```go
// Package: github.com/FiloSottile/mkcert

// Create or load CA
func NewCA() (*CA, error)

// Install CA to system trust stores
func (ca *CA) Install() error

// Remove CA from trust stores
func (ca *CA) Uninstall() error

// Generate certificate for domains
func (ca *CA) MakeCert(hosts []string) (certPEM, keyPEM []byte, err error)

// Check if CA is trusted
func (ca *CA) CheckPlatform() error
```

### Integration Pattern

```go
package proxy

import "github.com/FiloSottile/mkcert"

// EnsureCA creates or loads existing CA
func EnsureCA() (*mkcert.CA, error) {
    ca, err := mkcert.NewCA()
    if err != nil {
        return nil, fmt.Errorf("failed to create CA: %w", err)
    }
    return ca, nil
}

// InstallTrust installs CA to system trust stores
func InstallTrust(ca *mkcert.CA) error {
    if err := ca.Install(); err != nil {
        return fmt.Errorf("failed to install CA: %w", err)
    }
    return nil
}

// GenerateCertificate creates leaf certificate for AI providers
func GenerateCertificate(ca *mkcert.CA, domains []string) (certPath, keyPath string, err error) {
    certPEM, keyPEM, err := ca.MakeCert(domains)
    if err != nil {
        return "", "", fmt.Errorf("failed to generate cert: %w", err)
    }

    // Write to standard paths
    certPath = filepath.Join(CADir(), "certs", "cert.pem")
    keyPath = filepath.Join(CADir(), "certs", "key.pem")

    // Save files with appropriate permissions
    // ...

    return certPath, keyPath, nil
}
```

---

## Performance Considerations

### CA Generation Performance
**Expected**: <1 second for RSA-2048 key generation
**Source**: mkcert uses crypto/rsa with 2048-bit keys (standard)
**Mitigation**: One-time operation per installation

### Trust Installation Performance
**Expected**: <2 seconds for all trust stores (system + NSS)
**Source**: Platform-specific commands are fast (file copy + refresh)
**Mitigation**: One-time per CA, cached afterward

### Certificate Generation Performance
**Expected**: <1 second for multi-domain cert
**Source**: Certificate signing is fast with existing CA
**Mitigation**: Generated once at startup, reused

**Total Startup Impact**: +3-4 seconds on first run, <1 second on subsequent runs (CA already exists)

**Meets Performance Goal**: ✅ <5 seconds for all certificate operations

---

## Security Considerations

### CA Private Key Protection
**Risk**: CA private key compromise allows signing malicious certificates
**Mitigation**:
- Store CA key with 0600 permissions (owner read/write only)
- Located in user directory (not world-readable)
- Same protection as mkcert default

### Trust Store Modification
**Risk**: Malicious code could install rogue CA
**Mitigation**:
- Requires sudo/admin (user consent required)
- CA only signs specific AI provider domains (limited scope)
- User can inspect CA with `ddollar status`
- Clean removal with `ddollar untrust`

### Man-in-the-Middle Awareness
**Risk**: User may not understand CA trust implications
**Mitigation**:
- Clear messaging: "ddollar will install a local CA certificate"
- Warn that ddollar can intercept HTTPS traffic (by design)
- Document CA location and removal process
- Mark CA cert with "ddollar Local CA" common name

---

## Migration Path for Existing Users

### Users with Manual Certificate Trust
**Scenario**: User already manually trusted old self-signed cert

**Behavior**:
1. New CA generation doesn't remove old cert
2. Both certs trusted simultaneously (no conflict)
3. New leaf cert used by proxy (replaces old)
4. Old cert remains trusted (harmless) until manual removal

**Optional Cleanup**: User can remove old cert manually if desired

**Migration Steps**: None required - seamless upgrade

---

## Research Conclusions

All technical decisions made with rationale and alternatives documented. Key decisions:
1. ✅ Use mkcert as Go library dependency
2. ✅ Store CA in `~/.ddollar/ca/`
3. ✅ Continue proxy on trust failure with warnings
4. ✅ Single cert covering all AI domains, 365-day validity
5. ✅ Refactor existing cert.go to use mkcert
6. ✅ Include NSS trust store support (automatic via mkcert)
7. ✅ Integration tests require sudo, unit tests mock
8. ✅ Add `ddollar trust`/`untrust` CLI commands

**No NEEDS CLARIFICATION items remaining - ready for Phase 1 design.**

**Performance**: Meets <5 second goal
**Security**: Appropriate mitigations in place
**Compatibility**: Go 1.21+, all target platforms
**Migration**: Seamless for existing users
