# Implementation Plan: Automatic SSL Termination with mkcert

**Branch**: `002-automatic-ssl-termination` | **Date**: 2025-10-18 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/002-automatic-ssl-termination/spec.md`

**Note**: This plan implements Option 1 (Embed mkcert) from `docs/SSL_TERMINATION.md` for zero-friction SSL certificate trust.

## Summary

Embed mkcert functionality into ddollar to automatically create and trust a local Certificate Authority (CA) on first run, eliminating manual certificate trust steps. The system will generate a CA, install it into system trust stores (macOS/Linux/Windows), and generate leaf certificates for AI provider domains. This achieves the "one command, zero config" goal by removing all manual SSL setup friction.

## Technical Context

**Language/Version**: Go 1.21+
**Primary Dependencies**:
- `github.com/FiloSottile/mkcert` (certificate generation and trust installation)
- Go stdlib (crypto/x509, crypto/rsa, crypto/tls)
**Storage**: Filesystem (`~/.ddollar/ca/` for CA certificates, `~/.ddollar/certs/` for leaf certs)
**Testing**: Go testing stdlib (`go test`), integration tests require sudo
**Target Platform**: macOS (Intel/ARM64), Linux (x86_64/ARM64 - Debian/Ubuntu, RHEL/Fedora), Windows (x86_64)
**Project Type**: Single binary CLI tool (extends existing ddollar)
**Performance Goals**: <5 seconds for all certificate operations (CA creation + trust + cert generation)
**Constraints**:
- Must not break existing manual certificate trust users
- Must continue proxy operation even if trust installation fails
- Cannot require internet connectivity
- Must work with existing Go 1.21+ requirement
**Scale/Scope**: 4 AI provider domains, single CA, long-lived certificates (365 days)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**Note**: No formal constitution exists. Applying KISS principles from existing codebase:
- ✅ Single binary - extends existing ddollar binary
- ✅ Minimal dependencies - adds one well-established library (mkcert)
- ✅ Platform-native - leverages mkcert's cross-platform trust store integration
- ✅ Clear error messages - fallback instructions when auto-trust fails
- ✅ No breaking changes - refactors `src/proxy/cert.go`, extends CLI

**Constitution Compliance**: PASS - adheres to existing KISS architecture

## Project Structure

### Documentation (this feature)

```
specs/002-automatic-ssl-termination/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0: mkcert integration patterns, trust store APIs
├── data-model.md        # Phase 1: CA/Certificate structures, trust state model
├── quickstart.md        # Phase 1: User guide for automatic SSL setup
├── contracts/           # Phase 1: Certificate management API contracts
│   ├── ca-management.md
│   ├── cert-generation.md
│   └── trust-installation.md
└── tasks.md             # Phase 2: Actionable implementation tasks (NOT created by /speckit.plan)
```

### Source Code (repository root)

```
src/
├── main.go              # CLI entry point (existing, extends with trust commands)
├── proxy/
│   ├── server.go        # HTTP/HTTPS reverse proxy (existing)
│   ├── cert.go          # Certificate management (REFACTOR: use mkcert)
│   └── mkcert.go        # NEW: mkcert wrapper and CA management
├── tokens/
│   ├── discover.go      # Token discovery (existing, unchanged)
│   ├── pool.go          # Token pool (existing, unchanged)
│   └── providers.go     # Provider configs (existing, unchanged)
├── hosts/
│   ├── manager.go       # /etc/hosts modification (existing, unchanged)
│   ├── backup.go        # Backup/restore (existing, unchanged)
│   └── paths.go         # Platform-specific paths (existing, unchanged)
└── cli/
    └── trust.go         # NEW: trust/untrust certificate commands

tests/
├── integration/
│   ├── mkcert_test.go   # NEW: CA creation and trust installation (requires sudo)
│   └── cert_test.go     # NEW: End-to-end certificate flow
└── unit/
    ├── mkcert_test.go   # NEW: mkcert wrapper unit tests
    └── cert_test.go     # Certificate generation unit tests

~/.ddollar/              # User directory (runtime)
├── ca/
│   ├── rootCA.pem       # CA certificate
│   └── rootCA-key.pem   # CA private key
└── certs/
    ├── cert.pem         # Leaf certificate for AI providers
    └── key.pem          # Leaf private key
```

**Structure Decision**: Single project layout extending existing ddollar architecture. New mkcert integration code added to `src/proxy/mkcert.go`. Existing `src/proxy/cert.go` refactored to use mkcert instead of manual cert generation. New CLI commands added to `src/cli/trust.go` for certificate management. Tests follow existing structure with new mkcert-specific test files.

## Complexity Tracking

*No constitution violations - feature aligns with KISS principles and extends existing architecture minimally.*

