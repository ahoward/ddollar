# Feature Specification: Automatic SSL Termination with mkcert

**Feature Branch**: `002-automatic-ssl-termination`
**Created**: 2025-10-18
**Status**: Draft
**Input**: User description: "automatic ssl termination. we will use option 1 (mkcert) from ./docs/SSL_TERMINATION so users have a truly 0 friction way to get going, including ssl termination/support."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Zero-Config SSL Setup (Priority: P1)

A developer wants to start ddollar and have it immediately work with HTTPS APIs without any manual certificate trust steps.

**Why this priority**: This is the core value proposition - removing friction. Without automatic SSL trust, users face cryptic SSL errors and must follow platform-specific manual steps, which breaks the "just works" promise.

**Independent Test**: Can be fully tested by running `sudo ddollar start` on a fresh system and immediately making an HTTPS API call. Delivers immediate value by eliminating all certificate trust friction.

**Acceptance Scenarios**:

1. **Given** a fresh installation with no existing certificates, **When** user runs `sudo ddollar start`, **Then** the CA certificate is automatically created and trusted by the system
2. **Given** ddollar is starting for the first time, **When** the automatic trust installation completes, **Then** user sees confirmation message "✓ Certificate authority trusted"
3. **Given** ddollar has trusted certificates installed, **When** user makes HTTPS request to `api.anthropic.com`, **Then** the request succeeds without any SSL warnings or errors
4. **Given** user runs `ddollar start` without sudo, **When** automatic trust fails, **Then** clear fallback instructions are shown with platform-specific commands

---

### User Story 2 - Graceful Fallback for Restricted Environments (Priority: P2)

A developer in a restricted environment (Docker, corporate machine) where automatic cert installation fails can still use ddollar with clear manual instructions.

**Why this priority**: Not all environments allow automatic certificate installation. Users need a clear path forward when automation fails, but this is secondary to making the automatic path work for most users.

**Independent Test**: Can be tested by running ddollar in a Docker container or without sudo privileges. Delivers value by providing clear recovery path when automation isn't possible.

**Acceptance Scenarios**:

1. **Given** automatic certificate trust installation fails, **When** ddollar detects the failure, **Then** it prints platform-specific manual trust instructions
2. **Given** manual trust instructions are displayed, **When** user follows the instructions, **Then** ddollar continues to work correctly with manually trusted certificates
3. **Given** user is in a restricted environment, **When** trust installation fails, **Then** ddollar does not crash or block - it continues with informative warnings

---

### User Story 3 - Clean Certificate Removal (Priority: P3)

A user wants to uninstall ddollar and have all certificates cleanly removed from their system without leaving trust store artifacts.

**Why this priority**: Security hygiene and user trust. Users should be able to completely remove ddollar's certificates, but this is less critical than making installation work smoothly.

**Independent Test**: Can be tested by running `ddollar stop` or uninstall command and verifying no ddollar certificates remain in system trust stores.

**Acceptance Scenarios**:

1. **Given** ddollar certificates are trusted, **When** user runs `ddollar stop` or uninstall, **Then** all ddollar CA certificates are removed from system trust stores
2. **Given** certificates are being removed, **When** removal completes, **Then** user sees confirmation "✓ Certificates removed"
3. **Given** certificate removal fails, **When** error occurs, **Then** manual removal instructions are provided

---

### Edge Cases

- What happens when the system trust store is locked or requires additional permissions beyond sudo?
- How does the system handle certificate regeneration if the CA is compromised or corrupted?
- What happens when mkcert library is unavailable or fails to load?
- How does ddollar behave on systems with custom certificate trust policies (corporate environments)?
- What happens if certificates expire or need rotation?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST automatically generate a local Certificate Authority (CA) on first run if none exists
- **FR-002**: System MUST automatically install the CA certificate into the operating system's trust store when run with appropriate privileges (sudo/admin)
- **FR-003**: System MUST generate SSL certificates for all supported AI provider domains (api.openai.com, api.anthropic.com, api.cohere.ai, generativelanguage.googleapis.com) signed by the local CA
- **FR-004**: System MUST detect when automatic certificate trust installation fails and provide platform-specific manual instructions as fallback
- **FR-005**: System MUST support certificate trust installation on macOS, Linux (Debian/Ubuntu and RHEL/Fedora families), and Windows
- **FR-006**: System MUST provide clear visual feedback during certificate operations (creation, installation, generation) with success/failure indicators
- **FR-007**: System MUST continue to operate (with warnings) when automatic trust installation fails, allowing users to manually trust certificates
- **FR-008**: System MUST provide a command or option to remove ddollar certificates from system trust stores during uninstallation
- **FR-009**: System MUST store CA certificates in a standard user directory (`~/.ddollar/ca/`) to avoid requiring elevated permissions for storage
- **FR-010**: System MUST validate certificate integrity before use and regenerate if corrupted

### Key Entities

- **Certificate Authority (CA)**: Local CA created by ddollar, stored in `~/.ddollar/ca/`, used to sign all leaf certificates
  - Attributes: CA certificate (PEM), private key (PEM), creation timestamp
  - Relationships: Signs all leaf certificates for AI provider domains

- **Leaf Certificate**: Domain-specific SSL certificate signed by ddollar's CA
  - Attributes: Certificate (PEM), private key (PEM), domains covered, expiration
  - Relationships: Signed by CA, used by HTTPS proxy server

- **Trust Installation**: Platform-specific process to add CA to system trust stores
  - Attributes: Platform (macOS/Linux/Windows), trust store path, installation status
  - Relationships: Requires CA certificate, may require elevated permissions

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can start ddollar with one command (`sudo ddollar start`) and make HTTPS API calls without any manual certificate steps
- **SC-002**: Automatic certificate trust succeeds on 90%+ of standard desktop/server installations (macOS, Ubuntu, Fedora, Windows)
- **SC-003**: When automatic trust fails, users receive clear fallback instructions and can manually trust certificates within 2 minutes
- **SC-004**: Certificate operations (CA creation, trust installation, cert generation) complete in under 5 seconds total
- **SC-005**: Zero SSL certificate warnings or errors appear in browsers or HTTP clients after successful ddollar installation
- **SC-006**: Certificate removal during uninstall succeeds on 95%+ of systems where automatic installation succeeded

## Scope *(mandatory)*

### In Scope

- Automatic generation of local CA certificate
- Automatic installation of CA into system trust stores (macOS, Linux, Windows)
- Automatic generation of leaf certificates for AI provider domains
- Detection and graceful handling of trust installation failures
- Platform-specific fallback instructions for manual trust
- Certificate cleanup during uninstallation
- Support for NSS trust stores (Firefox, Chromium snap)
- Clear user feedback during all certificate operations

### Out of Scope

- Public certificate authorities or Let's Encrypt integration
- Certificate rotation or automatic renewal (certificates are long-lived for local development)
- Support for custom certificate paths or user-provided certificates
- Certificate revocation or CRL/OCSP
- Support for non-standard Linux distributions without standard trust store locations
- Java keystore integration (can be added later if needed)
- Certificate pinning or advanced security features

## Assumptions *(mandatory)*

1. Users will run ddollar with sudo/administrator privileges for the initial setup (required for trust store modification)
2. Standard system trust store locations are accessible and writable with appropriate permissions
3. The mkcert library (`github.com/FiloSottile/mkcert`) is compatible with all target platforms and Go versions
4. Users in highly restricted environments (corporate, containerized) understand they may need manual certificate trust steps
5. AI provider domains remain stable and do not require frequent certificate regeneration
6. Certificates with 1-year expiration are acceptable for local development use
7. Users trust ddollar enough to allow it to install a CA certificate in their system trust store

## Dependencies

### External Dependencies

- `github.com/FiloSottile/mkcert` Go library for certificate generation and trust installation
- System trust store binaries:
  - macOS: `security` command-line tool
  - Linux: `update-ca-certificates` (Debian/Ubuntu) or `update-ca-trust` (RHEL/Fedora)
  - Windows: `certutil` command

### Internal Dependencies

- Existing proxy server implementation (`src/proxy/server.go`)
- Existing certificate generation code (`src/proxy/cert.go`) - will be refactored to use mkcert
- Hosts file management (`src/hosts/`) - unchanged

## Constraints

- Must work with Go 1.21+ (ddollar's current minimum version)
- Must not break existing functionality for users who have already manually trusted certificates
- Certificate operations must not block proxy startup - if trust fails, proxy should still start with warnings
- Cannot require internet connectivity for certificate operations (all operations are local)
- Must respect platform security policies (no bypassing of system security controls)

## Non-Functional Requirements

### Performance

- Certificate generation and trust installation must complete in under 5 seconds
- No measurable impact on proxy request latency (certificate operations happen at startup only)

### Security

- CA private keys must be stored with restrictive file permissions (0600)
- CA certificates must have appropriate key usage extensions (cert signing only)
- Leaf certificates must only cover specified AI provider domains (no wildcards)
- Clear warnings when trust installation fails to prevent false sense of security

### Usability

- Zero configuration required for standard installations
- Clear, actionable error messages with platform-specific guidance
- Visual feedback (✓/✗ indicators) for all certificate operations
- No technical jargon in user-facing messages

### Maintainability

- Leverage mkcert library to avoid reimplementing trust store logic
- Platform-specific code isolated to fallback instructions only
- Clear code structure separating CA management, cert generation, and trust installation
