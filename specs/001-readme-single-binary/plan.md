# Implementation Plan: Single Binary with DNS Auto-Configuration

**Branch**: `001-readme-single-binary` | **Date**: 2025-10-18 | **Spec**: README.md
**Input**: Feature requirements from README.md

## Summary

Build `ddollar` as a single Go binary that transparently proxies AI provider requests by modifying `/etc/hosts`, rotating API tokens discovered from environment variables and config files. The tool prioritizes simplicity: `/etc/hosts` modification works identically on macOS, Linux, and Windows, eliminating platform-specific DNS complexity.

## Technical Context

**Language/Version**: Go 1.21+
**Primary Dependencies**: Go stdlib only (net/http, crypto/tls, os/exec)
**Storage**: In-memory token pool, filesystem for hosts backup
**Testing**: Go testing stdlib (`go test`)
**Target Platform**: macOS (Intel/ARM64), Linux (x86_64/ARM64), Windows (x86_64)
**Project Type**: Single binary CLI tool
**Performance Goals**: <10ms proxy latency, handle 100+ concurrent requests
**Constraints**: Must run with sudo/admin (port 443 + /etc/hosts), zero external dependencies
**Scale/Scope**: 3-5 AI providers, 10-50 tokens per provider, single user workstation

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**Note**: No constitution file exists yet. Default to KISS principles:
- Single binary (no services/daemons beyond the proxy process)
- No external dependencies
- Platform-native packaging (single executable)
- Clear error messages

## Project Structure

### Documentation (this feature)

```
specs/001-readme-single-binary/
├── plan.md              # This file
├── research.md          # Phase 0: Go proxy patterns, /etc/hosts safety
├── data-model.md        # Phase 1: Token pool structure, hosts file format
├── quickstart.md        # Phase 1: User setup guide
├── contracts/           # Phase 1: CLI command contracts
└── tasks.md             # Phase 2: Actionable implementation tasks
```

### Source Code (repository root)

```
src/
├── main.go              # CLI entry point (start, stop, status, version)
├── proxy/
│   ├── server.go        # HTTP/HTTPS reverse proxy
│   ├── cert.go          # Self-signed cert generation
│   └── router.go        # Request interception and token injection
├── tokens/
│   ├── discover.go      # Scan env vars and config files
│   ├── pool.go          # Round-robin token rotation
│   └── providers.go     # Provider-specific token formats
├── hosts/
│   ├── manager.go       # /etc/hosts modification (cross-platform)
│   ├── backup.go        # Backup/restore original hosts file
│   └── paths.go         # Platform-specific hosts file paths
└── cli/
    ├── start.go         # Start proxy, modify hosts
    ├── stop.go          # Stop proxy, restore hosts
    └── status.go        # Check proxy state

tests/
├── integration/
│   ├── hosts_test.go    # Verify hosts modification (requires sudo)
│   ├── proxy_test.go    # End-to-end proxy flow
│   └── tokens_test.go   # Token discovery and rotation
└── unit/
    ├── pool_test.go     # Token pool logic
    └── router_test.go   # Request routing logic
```

**Structure Decision**: Single project layout. This is a standalone CLI tool with no web frontend or mobile app. The `src/` directory contains all Go packages, organized by responsibility (proxy, tokens, hosts, cli). Tests follow Go conventions with `_test.go` suffix.

## KISS Architecture Decisions

### 1. DNS Auto-Configuration: `/etc/hosts` Approach

**Why `/etc/hosts` instead of DNS servers:**
- ✅ Identical path on macOS and Linux: `/etc/hosts`
- ✅ Known path on Windows: `C:\Windows\System32\drivers\etc\hosts`
- ✅ No distro-specific code (systemd-resolved, NetworkManager, scutil)
- ✅ Simple append/restore operations
- ✅ Takes precedence over DNS on all platforms
- ✅ No DNS server needed

**Implementation:**
```
# ddollar appends to /etc/hosts:
127.0.0.1 api.anthropic.com
127.0.0.1 api.openai.com
127.0.0.1 api.cohere.ai
# ... additional providers
```

### 2. HTTPS Interception: Self-Signed Certificate

**Problem**: AI providers use HTTPS (443), we need to decrypt/modify requests.

**Solution**:
1. Generate self-signed cert on first run (stored in `~/.ddollar/cert.pem`)
2. User runs `ddollar trust-cert` once (platform-specific instructions)
3. Proxy serves on port 443 with this cert

**Platform cert trust commands:**
- macOS: `sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain ~/.ddollar/cert.pem`
- Linux: Copy to `/usr/local/share/ca-certificates/` + `update-ca-certificates`
- Windows: `certutil -addstore -f "ROOT" cert.pem`

### 3. Token Discovery: Environment Variables First

**Priority order:**
1. Environment variables: `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, etc.
2. Config files:
   - `~/.config/anthropic/api_key`
   - `~/.openai/api_key`
   - `.env` files in common locations (`~/.env`, current dir)

**Rationale**: Env vars are universal and trivial to parse. Config files are provider-specific and may change format.

### 4. Token Rotation: Round-Robin

**Algorithm**: Simple round-robin (not rate-limit aware initially).

```go
tokens := []string{"sk-ant-...", "sk-ant-...", "sk-ant-..."}
currentIndex := 0

func nextToken() string {
    token := tokens[currentIndex]
    currentIndex = (currentIndex + 1) % len(tokens)
    return token
}
```

**Future enhancement**: Track rate limits (HTTP 429 responses), skip exhausted tokens.

### 5. Provider Support: Hardcoded List

**Initial providers:**
- OpenAI: `api.openai.com`
- Anthropic: `api.anthropic.com`
- Cohere: `api.cohere.ai`
- Google AI: `generativelanguage.googleapis.com`

**Why hardcoded**: Each provider has different auth headers (`Authorization: Bearer` vs `x-api-key`). Start simple, expand later.

### 6. Process Management: No Daemon

**`ddollar start`**: Starts proxy in foreground (or with `&` for background).

**Why no systemd/launchd**: Adds complexity. Users can background the process themselves (`ddollar start &` or `nohup ddollar start`).

**Alternative (if users demand it)**: Phase 2 can add daemon mode.

### 7. Cross-Compilation: Go Build Tags

```bash
# Build for all platforms:
GOOS=darwin GOARCH=amd64 go build -o ddollar-macos-x86_64
GOOS=darwin GOARCH=arm64 go build -o ddollar-macos-arm64
GOOS=linux GOARCH=amd64 go build -o ddollar-linux-x86_64
GOOS=linux GOARCH=arm64 go build -o ddollar-linux-arm64
GOOS=windows GOARCH=amd64 go build -o ddollar-windows-x86_64.exe
```

## Phase Breakdown

### Phase 0: Research
**Output**: `research.md`

**Questions to answer:**
1. Go reverse proxy patterns (net/http/httputil.ReverseProxy)
2. Safe /etc/hosts modification (atomic writes, backup strategy)
3. Self-signed cert generation (crypto/x509, crypto/rsa)
4. Cross-platform file paths (runtime.GOOS switch)
5. Provider auth header formats (test with real APIs)

### Phase 1: Design
**Output**: `data-model.md`, `quickstart.md`, `contracts/`

**Deliverables:**
1. **data-model.md**: Token pool struct, hosts file entry format, config schema
2. **quickstart.md**: User setup (install, trust cert, start proxy, verify)
3. **contracts/**:
   - `start.md`: `ddollar start` behavior, exit codes, error messages
   - `stop.md`: `ddollar stop` behavior, cleanup guarantees
   - `status.md`: `ddollar status` output format
   - `trust-cert.md`: `ddollar trust-cert` platform instructions

### Phase 2: Tasks
**Output**: `tasks.md` (generated by `/speckit.tasks`)

**High-level task groups:**
1. Hosts file manager (backup, append, restore)
2. Proxy server (HTTPS listener, reverse proxy logic)
3. Token discovery (env vars, config files)
4. Token rotation (round-robin pool)
5. CLI commands (start, stop, status, trust-cert)
6. Cross-platform build scripts
7. Integration tests (requires sudo)

## Risk Mitigation

### Risk 1: Hosts File Corruption
**Impact**: System DNS breaks if /etc/hosts is malformed.

**Mitigation**:
- Always create backup before modification (`/etc/hosts.ddollar.backup`)
- Atomic writes (write to temp file, rename)
- Validate syntax before replacing
- `ddollar stop` restores backup even if proxy crashes

### Risk 2: Port 443 Already in Use
**Impact**: Proxy can't start if another service uses port 443.

**Mitigation**:
- Check port availability before starting (`net.Listen` test)
- Clear error message: "Port 443 in use. Stop other services or use --port flag"
- Optional: `--port` flag to run on alternate port (requires manual app config)

### Risk 3: Certificate Trust
**Impact**: Users may struggle with cert trust (security warnings).

**Mitigation**:
- `ddollar trust-cert` command with platform-specific instructions
- Clear first-run message: "Run 'ddollar trust-cert' to avoid security warnings"
- Document troubleshooting steps in README

### Risk 4: Token Leakage
**Impact**: Tokens stored in memory could be exposed.

**Mitigation**:
- No token persistence (only in-memory)
- Clear tokens on `ddollar stop`
- Future: Encrypt tokens in memory (overkill for v1)

## Success Criteria

**Minimum Viable Product (MVP):**
1. `ddollar start` modifies `/etc/hosts` and starts HTTPS proxy on port 443
2. Proxy intercepts requests to OpenAI and Anthropic
3. Tokens discovered from `OPENAI_API_KEY` and `ANTHROPIC_API_KEY` env vars
4. Round-robin rotation works (verified with logs or status command)
5. `ddollar stop` restores original `/etc/hosts`
6. Builds run on macOS (Intel/ARM), Linux (x86_64), Windows (x86_64)

**Success Metrics:**
- User runs 3 commands: `ddollar trust-cert`, `ddollar start`, `curl https://api.openai.com/v1/models`
- No manual proxy configuration needed
- Works with official OpenAI/Anthropic SDKs (Python, JavaScript, etc.)

## Out of Scope (Future Enhancements)

**Not in v1:**
- Rate limit tracking (HTTP 429 handling) → v2
- Web UI for token management → v2
- Token usage analytics → v2
- Custom provider configuration (config file) → v2
- Daemon mode (systemd/launchd) → v2
- Homebrew tap (can add after v1 release)

## Implementation Notes

**Go version**: Require Go 1.21+ for standard library improvements (structured logging with `slog`).

**Dependencies**: ZERO external dependencies. Use only Go stdlib:
- `net/http` and `net/http/httputil` for proxy
- `crypto/tls`, `crypto/x509`, `crypto/rsa` for certs
- `os`, `os/exec` for file and process operations
- `encoding/json` for structured output

**Testing strategy**:
- Unit tests: No sudo required (test token pool logic, routing)
- Integration tests: Require sudo (test hosts modification, end-to-end proxy)
- CI: Run unit tests only (integration tests in manual QA)

**Build automation**:
- GitHub Actions for cross-platform builds
- Upload artifacts to GitHub Releases
- Tag releases with semantic versioning (v1.0.0, v1.1.0, etc.)

## Complexity Tracking

*No constitution violations - this is the first feature and adheres to KISS principles.*

---

**Next Steps:**
1. Run `/speckit.plan` to generate `research.md` (Phase 0)
2. After research, generate `data-model.md` and contracts (Phase 1)
3. Run `/speckit.tasks` to break down Phase 2 into actionable tasks
4. Implement with TDD (write tests first, then code)
