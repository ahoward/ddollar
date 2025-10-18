# Contract: Trust Installation CLI Commands

**Feature**: 002-automatic-ssl-termination
**Component**: `src/cli/trust.go` (new) + `src/main.go` (extended)
**Date**: 2025-10-18

## Overview

This contract defines the user-facing CLI commands for managing certificate trust. These commands provide explicit control over CA trust installation and removal, complementing the automatic trust that occurs during `ddollar start`.

---

## Command: ddollar trust

### Purpose
Manually install ddollar's CA certificate into system trust stores. Useful when automatic trust fails or user wants explicit control.

### Usage

```bash
ddollar trust [--force]
```

### Flags

- `--force`: Reinstall even if already trusted (optional)

### Behavior

1. Ensure CA exists (create if needed via `EnsureCA()`)
2. Check if CA is already trusted (via `VerifyTrust()`)
3. If already trusted and no `--force`: Print message and exit
4. If not trusted or `--force`: Install CA (via `InstallTrust()`)
5. Verify installation succeeded
6. Print success message or fallback instructions

### Output

**Success** (CA already trusted):
```
✓ Certificate authority is already trusted
No action needed.
```

**Success** (CA installed):
```
Installing ddollar CA certificate...
✓ CA installed to system trust store
✓ CA installed to NSS trust store (Firefox)

Your system now trusts ddollar certificates.
```

**Partial Success** (system OK, NSS failed):
```
Installing ddollar CA certificate...
✓ CA installed to system trust store
⚠️  NSS installation failed (Firefox may show warnings)

System trust successful. Most applications will work correctly.
```

**Failure** (permission denied):
```
❌ Failed to install CA certificate

ddollar needs administrator privileges to install certificates.
Please run with sudo:

  sudo ddollar trust

Or manually trust the certificate:
  Certificate location: ~/.ddollar/ca/rootCA.pem

Platform-specific instructions:
  [macOS/Linux/Windows instructions]
```

**Failure** (unsupported platform):
```
❌ Automatic certificate trust not supported on this platform

Your system: freebsd
Supported: macOS, Linux (Ubuntu/Debian, RHEL/Fedora), Windows

Manual trust instructions:
  1. Open certificate: ~/.ddollar/ca/rootCA.pem
  2. Import to system trust store
  3. Mark as trusted for SSL/TLS
  4. Verify with: ddollar status
```

### Exit Codes

- `0`: Success (trusted or already trusted)
- `1`: Failure (permission denied, unsupported platform, or installation error)

### Example Usage

```bash
# Standard usage (requires sudo)
sudo ddollar trust

# Force reinstall
sudo ddollar trust --force

# Check if trust is needed
ddollar status | grep "CA trusted"
```

### Testing

**Unit Test** (`tests/unit/trust_test.go`):
```go
func TestTrustCommand_ParsesFlags(t *testing.T) {
    cmd := &TrustCommand{}

    // Test --force flag
    args := []string{"--force"}
    err := cmd.ParseArgs(args)
    assert.NoError(t, err)
    assert.True(t, cmd.Force)
}

func TestTrustCommand_ChecksIfAlreadyTrusted(t *testing.T) {
    // Mock CA as already trusted
    ca := &CA{}
    mockVerifyTrust(ca, nil) // No error = trusted

    cmd := &TrustCommand{Force: false}
    shouldInstall := cmd.ShouldInstall(ca)

    assert.False(t, shouldInstall) // Skip install
}
```

**Integration Test** (`tests/integration/trust_test.go`):
```go
func TestTrustCommand_ActualInstallation(t *testing.T) {
    if os.Getuid() != 0 {
        t.Skip("Requires sudo")
    }

    // Ensure CA exists but not trusted
    ca, _ := EnsureCA()
    UninstallTrust(ca) // Clean slate

    // Execute trust command
    cmd := exec.Command("ddollar", "trust")
    output, err := cmd.CombinedOutput()

    assert.NoError(t, err)
    assert.Contains(t, string(output), "✓ CA installed")

    // Verify
    err = VerifyTrust(ca)
    assert.NoError(t, err)

    // Cleanup
    UninstallTrust(ca)
}
```

---

## Command: ddollar untrust

### Purpose
Remove ddollar's CA certificate from system trust stores. Used during uninstallation or when user wants to revoke trust.

### Usage

```bash
ddollar untrust
```

### Flags

- None

### Behavior

1. Check if CA exists
2. Check if CA is trusted
3. If not trusted: Print message and exit
4. If trusted: Remove CA from trust stores (via `UninstallTrust()`)
5. Verify removal succeeded
6. Print success message or manual instructions

### Output

**Success** (CA removed):
```
Removing ddollar CA certificate...
✓ CA removed from system trust store
✓ CA removed from NSS trust store

ddollar certificates are no longer trusted by your system.
```

**Success** (CA not trusted):
```
✓ CA certificate is not installed
No action needed.
```

**Partial Success** (system removed, NSS not found):
```
Removing ddollar CA certificate...
✓ CA removed from system trust store
⚠️  NSS trust store not found (skipped)

System trust removed successfully.
```

**Failure** (permission denied):
```
❌ Failed to remove CA certificate

ddollar needs administrator privileges to modify trust stores.
Please run with sudo:

  sudo ddollar untrust

Or manually remove the certificate:
  [platform-specific instructions]
```

**Warning** (CA files still exist):
```
✓ CA removed from trust stores

Note: CA files still exist at ~/.ddollar/ca/
To completely remove ddollar:
  1. Run: rm -rf ~/.ddollar
  2. Uninstall ddollar binary
```

### Exit Codes

- `0`: Success (removed or not installed)
- `1`: Failure (permission denied or removal error)

### Example Usage

```bash
# Standard usage (requires sudo)
sudo ddollar untrust

# Verify removal
ddollar status | grep "CA trusted"
```

### Testing

```go
func TestUntrustCommand_RemovesTrust(t *testing.T) {
    if os.Getuid() != 0 {
        t.Skip("Requires sudo")
    }

    // Setup: Install CA first
    ca, _ := EnsureCA()
    InstallTrust(ca)

    // Execute untrust
    cmd := exec.Command("ddollar", "untrust")
    output, err := cmd.CombinedOutput()

    assert.NoError(t, err)
    assert.Contains(t, string(output), "✓ CA removed")

    // Verify removal
    err = VerifyTrust(ca)
    assert.Error(t, err) // Should not be trusted
}

func TestUntrustCommand_HandlesNotInstalled(t *testing.T) {
    // Ensure CA not installed
    ca, _ := EnsureCA()
    UninstallTrust(ca)

    cmd := exec.Command("ddollar", "untrust")
    output, err := cmd.CombinedOutput()

    assert.NoError(t, err) // No error for already uninstalled
    assert.Contains(t, string(output), "not installed")
}
```

---

## Integration with `ddollar start`

### Automatic Trust on Start

The `ddollar start` command automatically attempts to install trust as part of initialization:

```go
// In src/main.go startCommand()

func startCommand() {
    log.Println("Starting ddollar...")

    // ... token discovery, etc. ...

    // Certificate setup with automatic trust
    log.Println("Setting up SSL certificates...")

    ca, err := EnsureCA()
    if err != nil {
        log.Fatalf("Failed to create CA: %v", err)
    }

    // Attempt automatic trust installation
    err = InstallTrust(ca)
    if err != nil {
        // Non-fatal - continue with warning
        log.Printf("⚠️  Automatic certificate trust failed: %v", err)
        log.Println("\nManual trust instructions:")
        PrintManualInstructions()
        log.Println("\nProxy is starting anyway. API calls will fail until certificates are trusted.")
        log.Println("To manually trust certificates, run: sudo ddollar trust")
    } else {
        log.Println("✓ Certificate authority trusted")
    }

    // Generate leaf certificate
    certPath, keyPath, err := GenerateCert()
    if err != nil {
        log.Fatalf("Failed to generate certificate: %v", err)
    }

    // ... continue with proxy startup ...
}
```

### User Flow

**Scenario 1: First run with sudo**
```bash
$ sudo ddollar start
Starting ddollar...
Discovering API tokens...
  ✓ Loaded 2 token(s) for Anthropic
Setting up SSL certificates...
  Creating certificate authority...
  ✓ CA installed to system trust store
  ✓ CA installed to NSS trust store
  ✓ Certificate authority trusted
  ✓ Generated certificates
Starting proxy on port 443...
✓ ddollar running
```

**Scenario 2: First run without sudo**
```bash
$ ddollar start
Starting ddollar...
Discovering API tokens...
  ✓ Loaded 2 token(s) for Anthropic
Setting up SSL certificates...
  Creating certificate authority...
  ⚠️  Automatic certificate trust failed: permission denied

Manual trust instructions:
  Run with administrator privileges:
    sudo ddollar start

  Or manually trust the certificate:
    sudo security add-trusted-cert -d -r trustRoot \
        -k /Library/Keychains/System.keychain \
        ~/.ddollar/ca/rootCA.pem

Proxy is starting anyway. API calls will fail until certificates are trusted.
To manually trust certificates, run: sudo ddollar trust

Starting proxy on port 443...
✓ ddollar running (certificates not trusted yet)
```

**Scenario 3: Subsequent runs**
```bash
$ sudo ddollar start
Starting ddollar...
Discovering API tokens...
  ✓ Loaded 2 token(s) for Anthropic
Setting up SSL certificates...
  ✓ Certificate authority trusted (already installed)
  ✓ Certificates ready
Starting proxy on port 443...
✓ ddollar running
```

---

## Status Command Integration

### Extended Output

The `ddollar status` command shows certificate trust state:

```bash
$ ddollar status
ddollar status:
  Hosts file modified: true
  Tokens discovered: 2
  Providers configured: 1

Certificates:
  CA certificate: ~/.ddollar/ca/rootCA.pem
  CA trusted: ✓ Yes (system + NSS)
  Leaf certificate: ~/.ddollar/certs/cert.pem
  Valid until: 2026-10-18 (342 days remaining)
  Domains: api.anthropic.com, api.openai.com, api.cohere.ai, generativelanguage.googleapis.com

Configured providers:
  - Anthropic: 2 token(s)
```

**When not trusted**:
```bash
Certificates:
  CA certificate: ~/.ddollar/ca/rootCA.pem
  CA trusted: ✗ No
  Action required: Run 'sudo ddollar trust' to install

Leaf certificate: ~/.ddollar/certs/cert.pem
  Valid until: 2026-10-18 (342 days remaining)
  ⚠️  Certificate will not be trusted until CA is installed
```

---

## Manual Trust Instructions

### Platform-Specific Instructions

**macOS**:
```
Manual trust instructions for macOS:

  sudo security add-trusted-cert -d -r trustRoot \
      -k /Library/Keychains/System.keychain \
      ~/.ddollar/ca/rootCA.pem

Verify with:
  security find-certificate -c "ddollar Local CA" \
      /Library/Keychains/System.keychain
```

**Linux (Debian/Ubuntu)**:
```
Manual trust instructions for Linux (Debian/Ubuntu):

  sudo cp ~/.ddollar/ca/rootCA.pem \
      /usr/local/share/ca-certificates/ddollar.crt
  sudo update-ca-certificates

Verify with:
  ls /usr/local/share/ca-certificates/ddollar.crt
```

**Linux (RHEL/Fedora)**:
```
Manual trust instructions for Linux (RHEL/Fedora):

  sudo cp ~/.ddollar/ca/rootCA.pem \
      /etc/pki/ca-trust/source/anchors/ddollar.pem
  sudo update-ca-trust

Verify with:
  ls /etc/pki/ca-trust/source/anchors/ddollar.pem
```

**Windows**:
```
Manual trust instructions for Windows:

  certutil -addstore -f "ROOT" %USERPROFILE%\.ddollar\ca\rootCA.pem

Verify with:
  certutil -store "ROOT" | findstr "ddollar"
```

**Firefox (NSS)**:
```
Manual trust for Firefox:

  certutil -A -n "ddollar Local CA" -t "C,," \
      -d sql:$HOME/.pki/nssdb \
      -i ~/.ddollar/ca/rootCA.pem

Verify with:
  certutil -L -d sql:$HOME/.pki/nssdb | grep ddollar
```

---

## Error Handling

### Permission Denied

```go
if err := InstallTrust(ca); err != nil {
    if errors.Is(err, os.ErrPermission) {
        fmt.Println("❌ Permission denied")
        fmt.Println("\nRun with sudo:")
        fmt.Println("  sudo ddollar trust")
        os.Exit(1)
    }
    // ... other error handling
}
```

### Unsupported Platform

```go
if err := InstallTrust(ca); err != nil {
    if errors.Is(err, ErrUnsupportedPlatform) {
        fmt.Printf("❌ Platform not supported: %s\n", runtime.GOOS)
        fmt.Println("\nManual trust instructions:")
        PrintManualInstructions()
        os.Exit(1)
    }
}
```

### Partial Failure (System OK, NSS Failed)

```go
// InstallTrust returns success if at least system trust succeeds
err := InstallTrust(ca)
if err == nil {
    fmt.Println("✓ CA installed to system trust store")
    // Check NSS separately
    if nssErr := InstallTrustNSS(ca); nssErr != nil {
        fmt.Println("⚠️  NSS installation failed (Firefox may show warnings)")
    } else {
        fmt.Println("✓ CA installed to NSS trust store")
    }
}
```

---

## Contract Summary

**Commands**: 2 (`trust`, `untrust`)
**Integration**: Automatic trust on `ddollar start`, status reporting in `ddollar status`
**Permission Requirements**: sudo/admin for trust installation and removal
**Error Strategy**: Graceful fallback with manual instructions
**Platforms**: macOS, Linux (Debian/Ubuntu, RHEL/Fedora), Windows, NSS (optional)
**User Experience**: Clear success/failure messages, platform-specific guidance
