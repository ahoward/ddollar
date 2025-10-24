# Bash Supervisor Exploration - Maximum KISS

## Current Go Implementation

**What we have**:
- Go supervisor (268 lines)
- Go monitor (179 lines)
- Token pool management
- Rate limit parsing from API responses
- Total: ~450 lines of Go

**Complexity**:
- Subprocess management with goroutines
- HTTP client for rate limit checks
- JSON parsing
- IPC via channels

---

## Proposed Bash Implementation

### The Absolute KISS Version

```bash
#!/bin/bash
# ddollar-supervise - Maximum KISS token rotation

# Config
TOKENS=($ANTHROPIC_API_KEY $ANTHROPIC_API_KEY_2 $ANTHROPIC_API_KEY_3)
CURRENT_TOKEN_INDEX=0
CHECK_INTERVAL=60
THRESHOLD=95

# Get current token
current_token() {
    echo "${TOKENS[$CURRENT_TOKEN_INDEX]}"
}

# Rotate to next token
rotate_token() {
    CURRENT_TOKEN_INDEX=$(( (CURRENT_TOKEN_INDEX + 1) % ${#TOKENS[@]} ))
    echo "▶ Rotated to token $((CURRENT_TOKEN_INDEX + 1))/${#TOKENS[@]}"
}

# Check rate limits (minimal API call)
check_limits() {
    local token="$1"
    local response=$(curl -s -w "\n%{http_code}" \
        -X POST https://api.anthropic.com/v1/messages \
        -H "x-api-key: $token" \
        -H "anthropic-version: 2023-06-01" \
        -H "content-type: application/json" \
        -d '{"model":"claude-3-5-sonnet-20241022","max_tokens":1,"messages":[{"role":"user","content":"."}]}' \
        2>/dev/null)

    # Extract headers (would need to use -i flag and parse)
    # For now, simplified: assume we get headers somehow
    local requests_remaining=$(echo "$response" | grep -i "anthropic-ratelimit-requests-remaining" | cut -d: -f2 | tr -d ' ')
    local requests_limit=$(echo "$response" | grep -i "anthropic-ratelimit-requests-limit" | cut -d: -f2 | tr -d ' ')

    if [ -n "$requests_remaining" ] && [ -n "$requests_limit" ]; then
        local percent=$(( (requests_limit - requests_remaining) * 100 / requests_limit ))
        echo "$percent"
    else
        echo "0"
    fi
}

# Monitor loop (runs in background)
monitor() {
    local child_pid="$1"

    while kill -0 "$child_pid" 2>/dev/null; do
        sleep $CHECK_INTERVAL

        local token=$(current_token)
        local usage=$(check_limits "$token")

        echo "Monitor: Token usage at ${usage}%"

        if [ "$usage" -ge "$THRESHOLD" ]; then
            echo "⚠️  Token limit approaching (${usage}% used)"

            # Kill child process
            kill -TERM "$child_pid" 2>/dev/null
            wait "$child_pid" 2>/dev/null

            # Rotate token
            rotate_token

            # Restart command with new token
            export ANTHROPIC_API_KEY=$(current_token)
            "$@" &
            child_pid=$!
        fi
    done
}

# Main supervisor
main() {
    if [ $# -eq 0 ]; then
        echo "Usage: ddollar-supervise <command> [args...]"
        exit 1
    fi

    echo "Starting supervision mode..."
    echo "✓ Loaded ${#TOKENS[@]} token(s)"
    echo "✓ Monitor checks every ${CHECK_INTERVAL}s"

    # Start command with first token
    export ANTHROPIC_API_KEY=$(current_token)
    "$@" &
    local child_pid=$!

    echo "▶ Launched: $*"

    # Start monitor in background
    monitor "$child_pid" "$@" &
    local monitor_pid=$!

    # Wait for child to finish
    wait "$child_pid"

    # Kill monitor
    kill "$monitor_pid" 2>/dev/null

    echo "✓ Process completed"
}

main "$@"
```

---

## Problems with Bash Approach

### 1. **Header Parsing is Painful**

curl can't easily give you response headers AND body separately without temp files:

```bash
# Option A: Write headers to file
curl -D /tmp/headers.txt -o /tmp/body.txt ...
# Annoying: temp file management

# Option B: Include headers in output
curl -i ...
# Annoying: need to parse HTTP/1.1 200 OK\r\n format

# Option C: Use -w for specific headers
curl -w "%{header_json}" ...  # Not available in older curl
```

**Reality**: You'd need to do something ugly like:

```bash
response=$(curl -i -s https://api.anthropic.com/v1/messages \
    -H "x-api-key: $token" \
    -H "anthropic-version: 2023-06-01" \
    -d '{"model":"...","max_tokens":1,"messages":[...]}')

# Split headers and body
headers=$(echo "$response" | sed '/^\r$/q')
remaining=$(echo "$headers" | grep -i "anthropic-ratelimit-requests-remaining:" | cut -d: -f2 | tr -d ' \r')
limit=$(echo "$headers" | grep -i "anthropic-ratelimit-requests-limit:" | cut -d: -f2 | tr -d ' \r')

percent=$(( (limit - remaining) * 100 / limit ))
```

**Ugly factor**: 7/10

### 2. **Process Management Gets Tricky**

```bash
# Start child
"$@" &
child_pid=$!

# What if child exits while monitor is sleeping?
# What if we get SIGTERM during monitor sleep?
# What if monitor needs to communicate with parent?
```

**Need to handle**:
- Child exits unexpectedly → monitor should detect and exit
- Parent gets killed → monitor should cleanup child
- Restart child with new token → need to update child_pid

**Bash solution**:
```bash
# Trap signals
trap cleanup SIGTERM SIGINT

cleanup() {
    kill "$child_pid" "$monitor_pid" 2>/dev/null
    exit
}

# Check if child is alive
while kill -0 "$child_pid" 2>/dev/null; do
    # monitor logic
done
```

**Complexity factor**: Medium (doable but not KISS)

### 3. **Token Array Management**

```bash
# How do users specify multiple tokens?

# Option 1: Separate env vars
ANTHROPIC_API_KEY=sk-ant-1...
ANTHROPIC_API_KEY_2=sk-ant-2...
ANTHROPIC_API_KEY_3=sk-ant-3...

# Build array (ugly)
TOKENS=()
[ -n "$ANTHROPIC_API_KEY" ] && TOKENS+=("$ANTHROPIC_API_KEY")
[ -n "$ANTHROPIC_API_KEY_2" ] && TOKENS+=("$ANTHROPIC_API_KEY_2")
[ -n "$ANTHROPIC_API_KEY_3" ] && TOKENS+=("$ANTHROPIC_API_KEY_3")

# Option 2: Comma-separated
ANTHROPIC_API_KEYS="sk-ant-1,sk-ant-2,sk-ant-3"
IFS=',' read -ra TOKENS <<< "$ANTHROPIC_API_KEYS"

# Option 3: Config file
# Not KISS
```

**Ugly factor**: 5/10

### 4. **Multi-Provider Support**

Current Go version handles:
- Anthropic
- OpenAI
- Cohere
- Google AI

Each has different:
- Header names (`anthropic-ratelimit-*` vs `x-ratelimit-*`)
- Auth formats (`x-api-key` vs `Authorization: Bearer`)
- Endpoint paths

**Bash solution**:
```bash
check_limits() {
    local provider="$1"
    local token="$2"

    case "$provider" in
        anthropic)
            curl -i -s https://api.anthropic.com/v1/messages \
                -H "x-api-key: $token" ...
            # Parse anthropic-ratelimit-* headers
            ;;
        openai)
            curl -i -s https://api.openai.com/v1/chat/completions \
                -H "Authorization: Bearer $token" ...
            # Parse x-ratelimit-* headers
            ;;
        *)
            echo "Unsupported provider: $provider"
            return 1
            ;;
    esac
}
```

**Complexity**: Grows quickly with each provider

---

## Hybrid Approach: Bash + jq

Use `jq` for JSON parsing:

```bash
# Check limits with jq
check_limits() {
    local token="$1"

    local response=$(curl -i -s https://api.anthropic.com/v1/messages \
        -H "x-api-key: $token" \
        -H "anthropic-version: 2023-06-01" \
        -H "content-type: application/json" \
        -d '{"model":"claude-3-5-sonnet-20241022","max_tokens":1,"messages":[{"role":"user","content":"."}]}')

    # Extract headers section
    headers=$(echo "$response" | sed '/^\r$/q')

    # Parse with grep
    local remaining=$(echo "$headers" | grep -i "anthropic-ratelimit-requests-remaining:" | sed 's/.*: *//' | tr -d '\r')
    local limit=$(echo "$headers" | grep -i "anthropic-ratelimit-requests-limit:" | sed 's/.*: *//' | tr -d '\r')

    # Calculate percentage
    if [ -n "$remaining" ] && [ -n "$limit" ] && [ "$limit" -gt 0 ]; then
        echo $(( (limit - remaining) * 100 / limit ))
    else
        echo "0"
    fi
}
```

**Dependencies**: curl, grep, sed
**KISS factor**: Better, but still ugly header parsing

---

## The REAL KISS Approach: Single File Bash Script

What if we DON'T monitor rate limits, and just rotate on a fixed schedule?

```bash
#!/bin/bash
# ddollar-supervise-simple - Rotate tokens every N requests/time

TOKENS=("$ANTHROPIC_API_KEY" "$ANTHROPIC_API_KEY_2" "$ANTHROPIC_API_KEY_3")
CURRENT=0
ROTATE_INTERVAL=3600  # Rotate every hour

main() {
    [ $# -eq 0 ] && { echo "Usage: $0 <command>"; exit 1; }

    echo "Supervision mode: Rotating tokens every ${ROTATE_INTERVAL}s"

    while true; do
        export ANTHROPIC_API_KEY="${TOKENS[$CURRENT]}"
        echo "▶ Running with token $((CURRENT + 1))/${#TOKENS[@]}"

        # Run command with timeout
        timeout $ROTATE_INTERVAL "$@"
        local exit_code=$?

        if [ $exit_code -eq 124 ]; then
            # Timeout - rotate token
            CURRENT=$(( (CURRENT + 1) % ${#TOKENS[@]} ))
            echo "⏰ Rotating to next token..."
        elif [ $exit_code -eq 0 ]; then
            # Success - exit
            echo "✓ Process completed"
            exit 0
        else
            # Error - exit
            echo "✗ Process failed: $exit_code"
            exit $exit_code
        fi
    done
}

main "$@"
```

**Lines**: 30
**Dependencies**: bash, timeout (coreutils)
**KISS factor**: 10/10
**Problem**: Doesn't actually check rate limits, just rotates blindly

---

## Comparison Matrix

| Approach | Lines | Dependencies | Rate Limit Aware | Multi-Provider | KISS Factor |
|----------|-------|--------------|------------------|----------------|-------------|
| **Current Go** | ~450 | None (single binary) | ✅ Yes | ✅ Yes | 6/10 |
| **Bash + curl parsing** | ~150 | bash, curl, grep, sed | ✅ Yes | ⚠️ Hard | 4/10 |
| **Bash + jq** | ~100 | bash, curl, jq | ✅ Yes | ⚠️ Medium | 5/10 |
| **Bash simple (time-based)** | ~30 | bash, timeout | ❌ No | ✅ Easy | 9/10 |
| **Bash simple (request count)** | ~50 | bash | ❌ No | ✅ Easy | 8/10 |

---

## Recommendation

### Keep Go, Add Bash Wrapper for Simple Use Case

**Option A: Pure Go (current)**
- Best for production use
- Single binary, no dependencies
- Rate limit aware
- Multi-provider support

**Option B: Add simple bash script for quick use**
```bash
#!/bin/bash
# ddollar-supervise-simple
# Rotate every hour or N requests (no rate limit checking)

TOKENS=("$ANTHROPIC_API_KEY" "$ANTHROPIC_API_KEY_2" "$ANTHROPIC_API_KEY_3")
CURRENT=0

while true; do
    export ANTHROPIC_API_KEY="${TOKENS[$CURRENT]}"
    echo "▶ Token $((CURRENT+1))/${#TOKENS[@]}"

    timeout 3600 "$@" || {
        CURRENT=$(( (CURRENT + 1) % ${#TOKENS[@]} ))
        echo "⏰ Rotating..."
        continue
    }
    break
done
```

**Use cases**:
- Go version: Production, rate-limit aware, all-night sessions
- Bash version: Quick hacks, time-based rotation, no rate limit checking needed

---

## Verdict

**Don't drop Go supervisor**. Here's why:

1. **Rate limit parsing in bash is ugly**: Header parsing, no structured data
2. **Multi-provider support gets messy**: Different headers, auth, endpoints
3. **Error handling complexity**: Process management, signal handling, restart logic
4. **The Go version IS already KISS**: Single binary, no config, works

**The Go supervisor (~450 lines) is the RIGHT level of complexity for the job.**

A bash script would be:
- Simpler IF you ignore rate limits (just rotate on time)
- MORE complex IF you want rate limit awareness (ugly header parsing)
- Less maintainable (bash process management is tricky)
- More fragile (depends on curl, grep, sed versions)

**Final answer**: Keep the Go supervisor. It's already KISS for what it does. A bash script would either be too simple (no rate limits) or too complex (ugly parsing).
