# Quick Start: Automatic SSL Termination

**Feature**: 002-automatic-ssl-termination
**Date**: 2025-10-18

## Overview

This guide shows how to use ddollar's automatic SSL certificate trust feature. After this feature is implemented, users will enjoy zero-friction SSL setup - no manual certificate trust steps required.

---

## 🚀 Quick Start (Post-Implementation)

### One Command Setup

```bash
# Set your API tokens
export ANTHROPIC_API_KEY=sk-ant-api03-...

# Start ddollar (requires sudo for automatic trust)
sudo ddollar start
```

**That's it!** SSL certificates are automatically trusted. No manual steps.

### What Happens Automatically

When you run `sudo ddollar start`:

1. ✓ Creates local Certificate Authority (CA) if needed
2. ✓ Installs CA to system trust stores
3. ✓ Generates SSL certificate for AI providers
4. ✓ Starts HTTPS proxy on port 443

**Result**: API calls work immediately with no SSL warnings.

---

## 🎯 User Scenarios

### Scenario 1: First-Time Setup (Ideal Path)

**User**: Developer installing ddollar for the first time

**Steps**:
```bash
# 1. Install ddollar
curl -LO https://github.com/ahoward/ddollar/releases/latest/download/ddollar-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m)
chmod +x ddollar-*
sudo mv ddollar-* /usr/local/bin/ddollar

# 2. Set API tokens
export ANTHROPIC_API_KEY=sk-ant-...

# 3. Start with sudo (for automatic trust)
sudo ddollar start
```

**Output**:
```
Starting ddollar...
Discovering API tokens...
  ✓ Loaded 1 token(s) for Anthropic
Setting up SSL certificates...
  Creating certificate authority...
  ✓ CA installed to system trust store
  ✓ CA installed to NSS trust store (Firefox)
  ✓ Certificate authority trusted
  ✓ Generated certificates
Starting proxy on port 443...
✓ ddollar running
```

**Verification**:
```bash
# Make an API call - should work with no SSL warnings
curl https://api.anthropic.com/v1/messages \
  -H "content-type: application/json" \
  -H "anthropic-version: 2023-06-01" \
  -d '{"model": "claude-3-5-sonnet-20241022", "max_tokens": 16, "messages": [{"role": "user", "content": "hi"}]}'
```

**Expected**: Response with no certificate errors.

---

### Scenario 2: Running Without Sudo (Fallback Path)

**User**: Developer who forgets sudo or doesn't have admin privileges

**Steps**:
```bash
# Start without sudo
ddollar start
```

**Output**:
```
Starting ddollar...
Discovering API tokens...
  ✓ Loaded 1 token(s) for Anthropic
Setting up SSL certificates...
  Creating certificate authority...
  ⚠️  Automatic certificate trust failed: permission denied

Manual trust instructions for macOS:
  sudo security add-trusted-cert -d -r trustRoot \
      -k /Library/Keychains/System.keychain \
      ~/.ddollar/ca/rootCA.pem

Proxy is starting anyway. API calls will fail until certificates are trusted.
To manually trust certificates, run: sudo ddollar trust

Starting proxy on port 443...
✓ ddollar running (certificates not trusted yet)
```

**User Action**:
```bash
# Stop proxy
ddollar stop

# Trust certificates manually
sudo ddollar trust

# Start again
sudo ddollar start
```

**Verification**: API calls now work.

---

### Scenario 3: Status Check

**User**: Developer wants to see SSL certificate status

**Steps**:
```bash
ddollar status
```

**Output**:
```
ddollar status:
  Hosts file modified: true
  Tokens discovered: 1
  Providers configured: 1

Certificates:
  CA certificate: ~/.ddollar/ca/rootCA.pem
  CA trusted: ✓ Yes (system + NSS)
  Leaf certificate: ~/.ddollar/certs/cert.pem
  Valid until: 2026-10-18 (342 days remaining)
  Domains: api.anthropic.com, api.openai.com, api.cohere.ai, generativelanguage.googleapis.com

Configured providers:
  - Anthropic: 1 token(s)
```

**Key Info**:
- ✓ "CA trusted: Yes" means SSL is working
- ✗ "CA trusted: No" means manual trust needed
- "Valid until" shows certificate expiration (auto-renewed if needed)

---

### Scenario 4: Restricted Environment (Docker)

**User**: Developer running ddollar in Docker container

**Context**: Docker containers often can't modify host trust stores

**Steps**:
```bash
# In Dockerfile
FROM ubuntu:22.04
RUN apt-get update && apt-get install -y ca-certificates
COPY ddollar /usr/local/bin/

# Run (won't have sudo in container)
CMD ["ddollar", "start"]
```

**Output**:
```
Starting ddollar...
...
Setting up SSL certificates...
  ⚠️  Automatic certificate trust failed: trust store not writable

Manual trust instructions:
  1. Copy CA certificate from container:
       docker cp <container>:~/.ddollar/ca/rootCA.pem ./ddollar-ca.pem
  2. Trust on host system:
       sudo security add-trusted-cert ... (macOS)
       sudo cp ... && sudo update-ca-certificates (Linux)
  3. Restart container

Proxy is starting anyway. API calls will fail until certificates are trusted.
```

**User Action**:
```bash
# Extract CA from container
docker cp ddollar-container:/root/.ddollar/ca/rootCA.pem ./ddollar-ca.pem

# Trust on host (macOS example)
sudo security add-trusted-cert -d -r trustRoot \
    -k /Library/Keychains/System.keychain \
    ./ddollar-ca.pem
```

**Result**: Requests from container now trusted by host browser/tools.

---

### Scenario 5: Uninstallation

**User**: Developer removing ddollar from system

**Steps**:
```bash
# 1. Remove certificate trust
sudo ddollar untrust

# 2. Remove ddollar files
rm -rf ~/.ddollar

# 3. Remove binary
sudo rm /usr/local/bin/ddollar
```

**Output (step 1)**:
```
Removing ddollar CA certificate...
✓ CA removed from system trust store
✓ CA removed from NSS trust store

ddollar certificates are no longer trusted by your system.
```

**Result**: Complete cleanup, no trust store artifacts remain.

---

## 🔧 Advanced Usage

### Manual Trust Commands

For explicit control over certificate trust:

```bash
# Install trust (alternative to automatic on start)
sudo ddollar trust

# Force reinstall trust
sudo ddollar trust --force

# Remove trust
sudo ddollar untrust

# Check trust status
ddollar status | grep "CA trusted"
```

### Certificate Locations

All certificates stored in `~/.ddollar/`:

```
~/.ddollar/
├── ca/
│   ├── rootCA.pem       # CA certificate (public)
│   └── rootCA-key.pem   # CA private key
└── certs/
    ├── cert.pem         # Leaf certificate
    └── key.pem          # Leaf private key
```

**To inspect CA**:
```bash
# View certificate details
openssl x509 -in ~/.ddollar/ca/rootCA.pem -text -noout

# Check fingerprint
openssl x509 -in ~/.ddollar/ca/rootCA.pem -fingerprint -noout
```

---

## 🐛 Troubleshooting

### Issue: "Automatic trust failed"

**Symptoms**:
```
⚠️  Automatic certificate trust failed: permission denied
```

**Causes**:
- Ran without sudo/admin privileges
- Trust store is locked or read-only
- Running in restricted environment (Docker, corporate machine)

**Solutions**:
1. Run with sudo: `sudo ddollar start`
2. Manually trust: `sudo ddollar trust`
3. Follow platform-specific instructions printed by ddollar

---

### Issue: "SSL certificate warnings in browser"

**Symptoms**:
- Browser shows "Not secure" or certificate error
- curl shows `SSL certificate problem`

**Causes**:
- CA not trusted by system
- Using Firefox with NSS (may need separate trust)

**Solutions**:
```bash
# Check trust status
ddollar status

# If "CA trusted: No":
sudo ddollar trust

# For Firefox specifically:
certutil -A -n "ddollar Local CA" -t "C,," \
    -d sql:$HOME/.pki/nssdb \
    -i ~/.ddollar/ca/rootCA.pem
```

---

### Issue: "Certificate expired"

**Symptoms**:
```
Certificate validation failed: certificate expired on [date]
```

**Causes**:
- Certificate past 365-day validity period

**Solutions**:
```bash
# Automatic regeneration on next start
sudo ddollar start

# Or force regeneration
ddollar stop
rm -rf ~/.ddollar/certs
sudo ddollar start
```

**Note**: Certificate auto-renews on startup if expired.

---

### Issue: "Port 443 already in use"

**Symptoms**:
```
Failed to start server: listen tcp :443: bind: address already in use
```

**Causes**:
- Another service using port 443
- Previous ddollar instance still running

**Solutions**:
```bash
# Find process using port 443
sudo lsof -i :443

# Kill if it's ddollar
pkill ddollar

# Or kill specific process
sudo kill -9 <PID>

# Restart
sudo ddollar start
```

---

## 📋 Checklist for Success

After implementation, users should be able to:

- [ ] Run `sudo ddollar start` once with zero manual steps
- [ ] Make HTTPS API calls immediately with no SSL warnings
- [ ] See ✓ "CA trusted: Yes" in `ddollar status`
- [ ] Use Firefox without additional configuration (NSS auto-trusted)
- [ ] Get clear fallback instructions if automatic trust fails
- [ ] Manually control trust with `ddollar trust`/`untrust` commands
- [ ] Completely uninstall with `sudo ddollar untrust && rm -rf ~/.ddollar`

---

## 🎓 User Education

### What Changed

**Before this feature**:
```bash
# Old workflow (manual trust)
sudo ddollar start
# See error about untrusted certificates
# Copy long platform-specific command
sudo security add-trusted-cert -d -r trustRoot \
    -k /Library/Keychains/System.keychain \
    ~/.ddollar/cert.pem
# Restart browser/apps to pick up trust
# Test API call
```

**After this feature**:
```bash
# New workflow (automatic)
sudo ddollar start
# Done! API calls work immediately
```

### Why Sudo Required

Clear explanation for users:

> **Why does ddollar need sudo?**
>
> ddollar modifies system security settings in two ways:
> 1. **Port 443**: Requires root to bind to privileged ports
> 2. **Trust stores**: System trust stores require admin/root to modify
>
> This is the same requirement as other local development tools (mkcert, Caddy).
>
> **Alternative**: Run without sudo and manually trust certificates afterward.

### Security Implications

Users should understand:

> **What ddollar can do with CA access**:
> - ddollar installs a Certificate Authority (CA) that your system trusts
> - This CA can sign certificates for any domain
> - ddollar only signs certificates for AI provider domains (api.anthropic.com, etc.)
> - The CA private key is stored locally (~/.ddollar/ca/)
> - You can remove trust at any time with: `sudo ddollar untrust`
>
> **This is safe because**:
> - CA is only used for local development
> - CA is isolated to your machine (not shared)
> - ddollar is open source (auditable)
> - Same approach used by mkcert (trusted by 48k+ developers)

---

## 📊 Success Metrics

After implementation, track:

1. **Automatic Trust Success Rate**: % of `ddollar start` runs where trust succeeds
   - Target: >90% on standard desktop/server installations

2. **User Friction**: Average time from install to first successful API call
   - Target: <2 minutes (vs ~5 minutes with manual trust)

3. **Support Requests**: Certificate-related issues reported
   - Target: <10% of users need manual intervention

4. **Browser Compatibility**: % of users reporting SSL warnings
   - Target: <5% (only Firefox users without NSS auto-trust)

---

## 🚀 Next Steps

After reading this guide, users should:

1. ✅ Understand that SSL trust is now automatic
2. ✅ Know to run `sudo ddollar start` for zero-friction setup
3. ✅ Understand fallback path if automatic trust fails
4. ✅ Know how to check trust status (`ddollar status`)
5. ✅ Understand how to uninstall cleanly (`sudo ddollar untrust`)

**Ready to use ddollar with automatic SSL!** 💸🔥
