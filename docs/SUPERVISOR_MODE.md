# Supervisor Mode: Uninterrupted All-Night Sessions

**Goal**: Run Claude Code (or any AI CLI) all night without hitting token limits.

## The Problem

```bash
# You at 6pm:
$ claude --continue
> "Implement this massive refactoring..."

# Claude at 2am (you're asleep):
HTTP 429: Rate limit exceeded
Session terminated.

# You at 8am:
$ cat progress.log
# 😡 Stopped at 2am. 10 hours wasted.
```

## The Solution: ddollar supervise

```bash
# Run Claude Code under supervision (headless mode - default)
$ ddollar supervise -- claude --continue

Starting supervision mode...
✓ Loaded 3 tokens for Anthropic
✓ Monitor started (checking limits every 60s)
✓ Launching: claude --continue

[Claude Code runs normally...]

[2:34am - Monitor detects: 95% of token limit used]
⚠️  Token limit approaching (95% used)
▶  Pausing claude process...
▶  Rotating to next token (2/3)
▶  Resuming claude with: claude --continue
✓ Session resumed seamlessly

[Your agent keeps working all night]
```

---

## Architecture

### Simple Master/Subprocess Model

```
ddollar supervise
├── Token Pool (manages 3 tokens)
├── Monitor (checks limits every 60s)
└── Subprocess: claude --continue
    └── ENV: ANTHROPIC_API_KEY=current-token
```

**No IPC complexity. No shared state. Just:**
1. Fork `claude --continue` with token in ENV
2. Monitor makes API call every 60s, checks headers
3. When limit hit (>95% used):
   - Send SIGTSTP to pause Claude (or SIGTERM to kill gracefully)
   - Rotate token in pool
   - Restart: `claude --continue` with new token in ENV
4. Claude Code's `--continue` picks up where it left off

---

## Usage Modes

### Headless Mode (Default)

**Use case:** Walk away, let it run all night

```bash
# Auto-rotate, never prompt
$ ddollar supervise -- claude --continue

# Runs unattended:
# - Monitor detects limit
# - Auto-rotates token
# - Resumes with --continue
# - Keeps going until task done or all tokens exhausted
```

**When all tokens hit limits:**
```
⚠️  All tokens exhausted. Waiting for limits to reset...
   Token 1 resets in: 23 minutes
   Token 2 resets in: 41 minutes
   Token 3 resets in: 18 minutes
▶  Sleeping for 18 minutes, will resume with Token 3...
```

### Interactive Mode

**Use case:** You're at the keyboard, want control

```bash
$ ddollar supervise --interactive -- claude --continue

[Monitor detects limit]
⚠️  Token limit hit (98% used)

What would you like to do?
  1) Rotate to next token and continue
  2) Wait for limit to reset (18 minutes)
  3) Exit and save state
  4) Keep going anyway (may hit 429 errors)

Choice [1]: _
```

---

## Implementation

### Phase 1: Core Supervisor (Week 1)

File: `src/supervisor/supervisor.go`

```go
package supervisor

import (
    "fmt"
    "log"
    "os"
    "os/exec"
    "syscall"
    "time"

    "github.com/drawohara/ddollar/src/tokens"
)

type Supervisor struct {
    pool        *tokens.Pool
    monitor     *Monitor
    command     []string
    interactive bool
    subprocess  *exec.Cmd
}

func New(pool *tokens.Pool, command []string, interactive bool) *Supervisor {
    return &Supervisor{
        pool:        pool,
        command:     command,
        interactive: interactive,
        monitor:     NewMonitor(60 * time.Second), // Check every 60s
    }
}

func (s *Supervisor) Run() error {
    fmt.Println("Starting supervision mode...")
    fmt.Printf("✓ Loaded %d tokens\n", s.pool.TotalTokens())
    fmt.Println("✓ Monitor started (checking limits every 60s)")

    // Start subprocess with first token
    if err := s.startSubprocess(); err != nil {
        return err
    }

    // Monitor in background
    limitChan := make(chan *RateLimitStatus)
    go s.monitor.Watch(s.pool.Current(), limitChan)

    // Wait for limit events
    for status := range limitChan {
        if status.ShouldRotate() {
            s.handleRotation(status)
        }
    }

    return nil
}

func (s *Supervisor) startSubprocess() error {
    token := s.pool.Current()

    fmt.Printf("▶  Launching: %s\n", s.command)

    s.subprocess = exec.Command(s.command[0], s.command[1:]...)
    s.subprocess.Env = append(os.Environ(),
        fmt.Sprintf("ANTHROPIC_API_KEY=%s", token.Value))
    s.subprocess.Stdin = os.Stdin
    s.subprocess.Stdout = os.Stdout
    s.subprocess.Stderr = os.Stderr

    return s.subprocess.Start()
}

func (s *Supervisor) handleRotation(status *RateLimitStatus) {
    fmt.Printf("\n⚠️  Token limit approaching (%d%% used)\n", status.PercentUsed())

    if s.interactive {
        s.promptUser(status)
    } else {
        s.autoRotate()
    }
}

func (s *Supervisor) autoRotate() {
    fmt.Println("▶  Auto-rotating to next token...")

    // Gracefully stop subprocess
    s.subprocess.Process.Signal(syscall.SIGTERM)
    s.subprocess.Wait()

    // Rotate token
    s.pool.Next()
    fmt.Printf("▶  Switched to token %d/%d\n", s.pool.CurrentIndex()+1, s.pool.TotalTokens())

    // Restart subprocess with new token
    s.startSubprocess()
    fmt.Println("✓ Session resumed")
}

func (s *Supervisor) promptUser(status *RateLimitStatus) {
    fmt.Println("\nWhat would you like to do?")
    fmt.Println("  1) Rotate to next token and continue")
    fmt.Printf("  2) Wait for limit to reset (%s)\n", status.TimeUntilReset())
    fmt.Println("  3) Exit and save state")
    fmt.Println("  4) Keep going (may hit 429 errors)")

    var choice int
    fmt.Print("\nChoice [1]: ")
    fmt.Scanf("%d", &choice)

    switch choice {
    case 1, 0: // Default to rotate
        s.autoRotate()
    case 2:
        s.waitForReset(status)
    case 3:
        s.gracefulExit()
    case 4:
        fmt.Println("▶  Continuing with current token...")
    }
}

func (s *Supervisor) waitForReset(status *RateLimitStatus) {
    duration := status.TimeUntilReset()
    fmt.Printf("▶  Pausing subprocess for %s...\n", duration)

    // Send SIGTSTP to pause (like Ctrl+Z)
    s.subprocess.Process.Signal(syscall.SIGTSTP)

    // Wait
    time.Sleep(duration)

    // Resume with SIGCONT
    fmt.Println("▶  Resuming subprocess...")
    s.subprocess.Process.Signal(syscall.SIGCONT)
}

func (s *Supervisor) gracefulExit() {
    fmt.Println("▶  Stopping subprocess gracefully...")
    s.subprocess.Process.Signal(syscall.SIGTERM)
    s.subprocess.Wait()
    fmt.Println("✓ Session saved. Run 'claude --continue' to resume.")
    os.Exit(0)
}
```

### Phase 2: Rate Limit Monitor

File: `src/supervisor/monitor.go`

```go
package supervisor

import (
    "bytes"
    "encoding/json"
    "net/http"
    "strconv"
    "time"

    "github.com/drawohara/ddollar/src/tokens"
)

type Monitor struct {
    interval time.Duration
}

type RateLimitStatus struct {
    RequestsLimit     int
    RequestsRemaining int
    TokensLimit       int
    TokensRemaining   int
    ResetTime         time.Time
}

func NewMonitor(interval time.Duration) *Monitor {
    return &Monitor{interval: interval}
}

func (m *Monitor) Watch(token *tokens.Token, statusChan chan *RateLimitStatus) {
    ticker := time.NewTicker(m.interval)
    defer ticker.Stop()

    for range ticker.C {
        status, err := m.checkLimits(token)
        if err != nil {
            log.Printf("Monitor error: %v", err)
            continue
        }

        // Send status if rotation needed
        if status.ShouldRotate() {
            statusChan <- status
        }
    }
}

func (m *Monitor) checkLimits(token *tokens.Token) (*RateLimitStatus, error) {
    // Make minimal API call (1 token)
    reqBody := map[string]interface{}{
        "model":      "claude-3-5-sonnet-20241022",
        "max_tokens": 1,
        "messages": []map[string]string{
            {"role": "user", "content": "."},
        },
    }

    body, _ := json.Marshal(reqBody)
    req, _ := http.NewRequest("POST", token.Provider.APIEndpoint, bytes.NewReader(body))
    req.Header.Set("x-api-key", token.Value)
    req.Header.Set("anthropic-version", "2023-06-01")
    req.Header.Set("content-type", "application/json")

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    // Parse rate limit headers
    status := &RateLimitStatus{
        RequestsLimit:     parseInt(resp.Header.Get("anthropic-ratelimit-requests-limit")),
        RequestsRemaining: parseInt(resp.Header.Get("anthropic-ratelimit-requests-remaining")),
        TokensLimit:       parseInt(resp.Header.Get("anthropic-ratelimit-tokens-limit")),
        TokensRemaining:   parseInt(resp.Header.Get("anthropic-ratelimit-tokens-remaining")),
    }

    // Parse reset time
    resetStr := resp.Header.Get("anthropic-ratelimit-requests-reset")
    if resetTime, err := time.Parse(time.RFC3339, resetStr); err == nil {
        status.ResetTime = resetTime
    }

    return status, nil
}

func (s *RateLimitStatus) ShouldRotate() bool {
    // Rotate if >95% used on requests OR tokens
    requestPercent := float64(s.RequestsLimit-s.RequestsRemaining) / float64(s.RequestsLimit) * 100
    tokenPercent := float64(s.TokensLimit-s.TokensRemaining) / float64(s.TokensLimit) * 100

    return requestPercent > 95 || tokenPercent > 95
}

func (s *RateLimitStatus) PercentUsed() int {
    requestPercent := float64(s.RequestsLimit-s.RequestsRemaining) / float64(s.RequestsLimit) * 100
    tokenPercent := float64(s.TokensLimit-s.TokensRemaining) / float64(s.TokensLimit) * 100

    if requestPercent > tokenPercent {
        return int(requestPercent)
    }
    return int(tokenPercent)
}

func (s *RateLimitStatus) TimeUntilReset() time.Duration {
    return time.Until(s.ResetTime)
}

func parseInt(s string) int {
    i, _ := strconv.Atoi(s)
    return i
}
```

### Phase 3: CLI Integration

File: `src/main.go` (add command)

```go
case "supervise":
    superviseCommand()
```

```go
func superviseCommand() {
    // Parse flags
    interactive := false
    args := os.Args[2:]

    if len(args) > 0 && (args[0] == "--interactive" || args[0] == "-i") {
        interactive = true
        args = args[1:]
    }

    // Skip "--" separator
    if len(args) > 0 && args[0] == "--" {
        args = args[1:]
    }

    if len(args) == 0 {
        fmt.Println("ERROR: No command specified")
        fmt.Println("Usage: ddollar supervise [--interactive] -- <command>")
        os.Exit(1)
    }

    // Discover tokens
    discovered := tokens.Discover()
    if len(discovered) == 0 {
        fmt.Println("ERROR: No API tokens found")
        os.Exit(1)
    }

    // Create token pool
    pool := tokens.NewPool()
    for _, pt := range discovered {
        pool.AddProvider(pt.Provider, pt.Tokens)
    }

    // Start supervisor
    sup := supervisor.New(pool, args, interactive)
    if err := sup.Run(); err != nil {
        fmt.Printf("ERROR: %v\n", err)
        os.Exit(1)
    }
}
```

---

## Usage Examples

### Example 1: All-Night Refactoring

```bash
$ ddollar supervise -- claude --continue
Starting supervision mode...
✓ Loaded 3 tokens for Anthropic
✓ Monitor started (checking limits every 60s)
▶  Launching: claude --continue

[You give Claude a huge task, go to bed]

[2:15am] ⚠️  Token 1 limit approaching (96% used)
[2:15am] ▶  Auto-rotating to next token...
[2:15am] ▶  Switched to token 2/3
[2:15am] ✓ Session resumed

[4:47am] ⚠️  Token 2 limit approaching (95% used)
[4:47am] ▶  Auto-rotating to next token...
[4:47am] ▶  Switched to token 3/3
[4:47am] ✓ Session resumed

[8:00am] You wake up - task is complete! 🎉
```

### Example 2: Interactive Control

```bash
$ ddollar supervise --interactive -- claude --continue

[Working on task...]

[1:00pm] ⚠️  Token limit approaching (97% used)

What would you like to do?
  1) Rotate to next token and continue
  2) Wait for limit to reset (23 minutes)
  3) Exit and save state
  4) Keep going (may hit 429 errors)

Choice [1]: 2

▶  Pausing subprocess for 23 minutes...
[Go grab lunch]
▶  Resuming subprocess...
✓ Session continued
```

---

## Why This is KISS

1. **No proxy complexity**: Just fork/exec with ENV
2. **No IPC**: Monitor runs in goroutine, uses channel
3. **Graceful shutdown**: SIGTERM, wait, restart
4. **Leverages `--continue`**: Claude Code already handles state
5. **One file each**: supervisor.go, monitor.go, done

**Total implementation: ~300 lines of Go**

This is exactly what you described: support all-night sessions without babysitting. Let's build it.
