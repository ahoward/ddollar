# Token Limit Monitoring Architecture - Feasibility Analysis

**Date**: 2025-10-19
**Status**: Feasibility Study

## Executive Summary

**Verdict: FEASIBLE but with significant trade-offs**

The proposed approach of using a monitoring process to check token limits and gracefully rotate is technically feasible but introduces complexity and latency that may not align with ddollar's "zero friction" philosophy.

---

## Proposed Architecture

### Master Process
- Spawns and manages two subprocesses
- Handles token rotation logic
- Manages graceful shutdown/restart

### Subprocess A: Claude Code (or any AI tool)
- Runs with single token in `ANTHROPIC_API_KEY` env var
- Unaware of monitoring/rotation
- Can be gracefully stopped and restarted with `--continue`

### Subprocess B: Monitoring Process
- Makes lightweight API calls every minute to check rate limit headers
- Parses response headers for remaining capacity
- Triggers rotation when limits are hit
- Communicates with master to shutdown/restart Subprocess A

---

## Feasibility Analysis

### ✅ What Works

#### 1. Rate Limit Detection via Headers

Both major providers expose rate limits in response headers:

**Anthropic (2025):**
```
anthropic-ratelimit-requests-limit: 1000
anthropic-ratelimit-requests-remaining: 847
anthropic-ratelimit-requests-reset: 2025-10-19T00:15:00Z
anthropic-ratelimit-tokens-limit: 50000
anthropic-ratelimit-tokens-remaining: 32451
anthropic-ratelimit-tokens-reset: 2025-10-19T00:15:00Z
retry-after: 60
```

**OpenAI:**
```
x-ratelimit-limit-requests: 500
x-ratelimit-remaining-requests: 342
x-ratelimit-reset-requests: 1m23s
x-ratelimit-limit-tokens: 40000
x-ratelimit-remaining-tokens: 28934
x-ratelimit-reset-tokens: 1m23s
```

**Monitoring approach:**
- Make minimal API call (e.g., 1 token request every 60 seconds)
- Parse headers to track remaining capacity
- Trigger rotation at threshold (e.g., <5% remaining)

**Cost:** ~1440 minimal API calls/day = negligible cost

#### 2. Graceful Process Management

Go has excellent support for subprocess management:

```go
// Master process
cmd := exec.Command("claude", "--continue")
cmd.Env = append(os.Environ(), "ANTHROPIC_API_KEY="+currentToken)

// Start process
cmd.Start()

// Graceful shutdown on rotation
cmd.Process.Signal(syscall.SIGTERM)
cmd.Wait()

// Rotate token
currentToken = pool.Next()

// Restart with --continue
cmd = exec.Command("claude", "--continue")
cmd.Env = append(os.Environ(), "ANTHROPIC_API_KEY="+currentToken)
cmd.Start()
```

#### 3. Multi-Provider Support

The approach works for:
- ✅ Anthropic (rate limit headers documented)
- ✅ OpenAI (rate limit headers documented)
- ✅ Cohere (similar header pattern)
- ✅ Google AI (similar header pattern)

---

### ⚠️ Challenges & Trade-offs

#### 1. Reactive vs Proactive

**Current ddollar (proxy):** Proactive
- Rotates tokens round-robin on every request
- Distributes load evenly across all tokens
- No single token hits limits

**Proposed (monitoring):** Reactive
- Uses one token until it hits limits
- Only rotates when limit detected
- Introduces "waiting period" while limits reset

**Impact:** May hit 429 errors before rotation triggers

#### 2. Detection Latency

**Monitoring interval:** 60 seconds (per your spec)
- If Claude Code makes 1000 requests in 30 seconds, monitor won't detect until next check
- Could hit hard limit and get 429 errors before rotation
- Tighter intervals (every 10s) reduce latency but increase monitoring cost

**Mitigation:**
- Reduce check interval to 10-15 seconds
- Parse 429 responses and trigger immediate rotation
- Implement exponential backoff during rotation

#### 3. State Management Complexity

**Challenge:** Coordinating state between 3 processes
- Master must track current token
- Monitor must communicate limit status to master
- Claude Code must support `--continue` reliably

**Inter-process communication options:**
- Shared file (simple but slower)
- Unix socket (faster, more complex)
- Signals (limited data transfer)

**Example architecture:**
```
Master Process
├── Token Pool (manages rotation)
├── IPC Server (Unix socket)
│   ├── Listens for "LIMIT_HIT" from monitor
│   └── Sends "ROTATE" command
├── Subprocess A: Claude Code
│   └── ENV: ANTHROPIC_API_KEY=token-1
└── Subprocess B: Monitor
    ├── Makes API call every 60s
    ├── Parses headers
    └── Sends "LIMIT_HIT" via IPC when threshold reached
```

#### 4. Graceful Shutdown is Not Instant

**Problem:** Claude Code may be mid-request when shutdown signal arrives
- Request could fail with connection error
- User sees interrupted operation
- State may be inconsistent

**Mitigation:**
- Implement drain period (wait for current request to finish)
- Buffer requests during rotation
- Requires cooperation from Claude Code (may not be controllable)

#### 5. Multi-Tool Support

**ddollar's value prop:** Works with ANY tool
- Current proxy approach: tool-agnostic (intercepts at network layer)
- Proposed approach: requires spawning the tool as subprocess

**Compatibility:**
- ✅ CLI tools that read env vars (claude, python scripts)
- ✅ Tools with `--continue` flag (Claude Code)
- ❌ GUI apps (can't be spawned as subprocess)
- ❌ Browser-based tools (need proxy approach)
- ⚠️ Tools without `--continue` (lose session state on rotation)

---

## Architecture Design

### Option 1: Hybrid Approach (RECOMMENDED)

Combine proxy + monitoring for best of both worlds:

**Proxy layer (current ddollar):**
- Intercepts requests at network level
- Injects tokens round-robin (default behavior)
- Works with any tool

**Optional monitoring layer:**
- Tracks usage via response headers
- Adjusts rotation strategy based on real-time limits
- Provides usage insights to user

**Benefits:**
- Zero config still works (proxy mode)
- Advanced users can enable monitoring for optimization
- Tool-agnostic
- Graceful degradation

### Option 2: Pure Monitoring (Your Proposal)

Replace proxy entirely with process management:

**Architecture:**
```go
// src/supervisor/master.go
type Master struct {
    tokenPool *tokens.Pool
    monitor   *Monitor
    claude    *exec.Cmd
    ipc       *IPCServer
}

func (m *Master) Start() {
    // Start monitor
    m.monitor = NewMonitor(m.tokenPool.Current())
    go m.monitor.Run(m.ipc)

    // Start Claude Code
    m.claude = m.spawnClaude(m.tokenPool.Current())

    // Listen for rotation events
    for event := range m.ipc.Events {
        if event == "LIMIT_HIT" {
            m.rotateClaude()
        }
    }
}

func (m *Master) rotateClaude() {
    // Graceful shutdown
    m.claude.Process.Signal(syscall.SIGTERM)
    m.claude.Wait()

    // Rotate token
    nextToken := m.tokenPool.Next()

    // Restart with new token
    m.claude = m.spawnClaude(nextToken)
}

func (m *Master) spawnClaude(token string) *exec.Cmd {
    cmd := exec.Command("claude", "--continue")
    cmd.Env = append(os.Environ(), "ANTHROPIC_API_KEY="+token)
    cmd.Start()
    return cmd
}
```

```go
// src/supervisor/monitor.go
type Monitor struct {
    token          string
    checkInterval  time.Duration
    threshold      float64  // Rotate when remaining < 5%
}

func (m *Monitor) Run(ipc *IPCServer) {
    ticker := time.NewTicker(m.checkInterval)
    for range ticker.C {
        limits, err := m.checkLimits()
        if err != nil {
            log.Printf("Failed to check limits: %v", err)
            continue
        }

        // Check if any limit is near threshold
        if limits.RequestsRemaining < limits.RequestsLimit * m.threshold {
            ipc.Send("LIMIT_HIT")
        }
        if limits.TokensRemaining < limits.TokensLimit * m.threshold {
            ipc.Send("LIMIT_HIT")
        }
    }
}

func (m *Monitor) checkLimits() (*RateLimits, error) {
    // Make minimal API call
    req, _ := http.NewRequest("POST", "https://api.anthropic.com/v1/messages",
        strings.NewReader(`{
            "model": "claude-3-5-sonnet-20241022",
            "max_tokens": 1,
            "messages": [{"role": "user", "content": "Hi"}]
        }`))
    req.Header.Set("x-api-key", m.token)
    req.Header.Set("anthropic-version", "2023-06-01")

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    // Parse rate limit headers
    return &RateLimits{
        RequestsLimit:     parseInt(resp.Header.Get("anthropic-ratelimit-requests-limit")),
        RequestsRemaining: parseInt(resp.Header.Get("anthropic-ratelimit-requests-remaining")),
        TokensLimit:       parseInt(resp.Header.Get("anthropic-ratelimit-tokens-limit")),
        TokensRemaining:   parseInt(resp.Header.Get("anthropic-ratelimit-tokens-remaining")),
    }, nil
}
```

**Benefits:**
- Reactive rotation based on actual usage
- Detailed usage visibility
- Potentially more efficient (only rotate when needed)

**Drawbacks:**
- More complex than proxy
- Requires tool cooperation (`--continue` support)
- Not tool-agnostic
- Rotation has latency (60s check interval)
- May hit 429s before rotation kicks in

---

## Comparison Matrix

| Feature | Current Proxy | Pure Monitoring | Hybrid |
|---------|---------------|-----------------|--------|
| **Tool Support** | Any tool | CLI tools only | Any tool |
| **Rotation Strategy** | Round-robin per request | Reactive on limits | Both |
| **Setup Complexity** | Simple | Complex | Medium |
| **Rotation Latency** | None (instant) | 60s (monitoring interval) | None |
| **429 Error Risk** | Very low | Medium-High | Very low |
| **State Continuity** | Seamless | Interrupted on rotation | Seamless |
| **Usage Visibility** | None | Detailed | Detailed |
| **Code Complexity** | Low | High | Medium |

---

## Recommendation

### For ddollar: Use **Hybrid Approach**

**Phase 1 (Current):** Keep proxy as primary mechanism
- Simple, tool-agnostic, zero friction
- Round-robin prevents hitting limits

**Phase 2 (Enhancement):** Add optional monitoring layer
- Passive monitoring (no process management)
- Proxy intercepts responses, extracts rate limit headers
- Displays usage stats: `ddollar status --usage`
- Adjusts rotation strategy if patterns detected

**Example:**
```bash
$ ddollar status --usage

Token Usage (last hour):
  Token 1 (sk-ant-...): 342/1000 requests, 28934/50000 tokens
  Token 2 (sk-ant-...): 289/1000 requests, 21445/50000 tokens
  Token 3 (sk-ant-...): 401/1000 requests, 35221/50000 tokens

Rotation: Round-robin (default)
Next rotation in: 3 requests
```

**Benefits:**
- Preserves simplicity and tool-agnostic nature
- Adds visibility without complexity
- No subprocess management needed
- No graceful shutdown issues
- Still works with GUI tools, browsers, etc.

---

## Alternative: If You Really Want Process Supervision

If the goal is NOT token rotation but rather:
- **Supervisor pattern for robustness** (auto-restart on crash)
- **Resource monitoring** (CPU, memory)
- **Log aggregation** from multiple processes

Then the master/subprocess approach makes sense, but use it for **reliability**, not token rotation.

**Use case:**
```bash
ddollar supervise -- claude --continue
# Master supervises Claude Code, restarts on crash, aggregates logs
# Token rotation still happens at proxy layer
```

---

## Implementation Effort

### Current Proxy (Done)
- ✅ Hosts file modification
- ✅ HTTPS proxy with SSL termination
- ✅ Token injection and round-robin rotation
- ✅ Multi-provider support

### Pure Monitoring Approach (Estimated 2-3 weeks)
- Week 1: Master process, subprocess spawning, IPC
- Week 2: Monitor with API polling, header parsing, rotation logic
- Week 3: Testing, edge cases, graceful shutdown

### Hybrid Approach (Estimated 3-5 days)
- Day 1: Intercept response headers in proxy
- Day 2: Parse and store rate limit data
- Day 3: CLI command for usage display
- Day 4-5: Testing, multi-provider support

---

## Conclusion

**For ddollar's "zero friction" philosophy:**
- ✅ Stick with proxy approach
- ✅ Enhance with passive monitoring (headers)
- ❌ Avoid process supervision for token rotation
- ✅ Consider supervision as separate feature (reliability, not rotation)

**The monitoring approach is feasible but better suited for:**
- Single-tool scenarios (e.g., Claude Code wrapper)
- Reactive rotation requirements
- Deep usage analytics

**ddollar's strength is simplicity:** Install → Set env vars → Run. The proxy achieves this. Process supervision adds friction.
