# ddollar v0.2.0 - KISS Release 🔥

**Never hit token limits again** - supervisor-only, maximum KISS

## What's New

### 🎯 Supervisor-Only Focus
- Stripped to essentials: **92% less code** (900 lines vs 12,000)
- **38% smaller binary** (5MB vs 12MB)
- One command: `ddollar <command>` - that's it
- No sudo required, no system modification

### 🔧 Flexible Token Input
```bash
# Comma-separated
export ANTHROPIC_API_KEYS=key1,key2,key3

# File (one per line)
export ANTHROPIC_API_KEYS_FILE=~/.ddollar-keys

# Mix and match (auto-deduplicated)
export ANTHROPIC_API_KEY=primary
export ANTHROPIC_API_KEYS=key1,key2
export ANTHROPIC_API_KEYS_FILE=~/keys.txt
```

### ✨ Features
- 🔁 Monitors rate limits every 60 seconds
- 🌙 Auto-rotates tokens when >95% used
- ⚡ Gracefully restarts with `--continue`
- 💤 Run agents all night, zero babysitting
- 🎛️ Interactive mode with `--interactive`

## Quick Start

```bash
# Download for your platform
curl -LO https://github.com/ahoward/ddollar/releases/download/v0.2.0/ddollar-linux-amd64
chmod +x ddollar-linux-amd64
sudo mv ddollar-linux-amd64 /usr/local/bin/ddollar

# Run
export ANTHROPIC_API_KEY=sk-ant-...
ddollar claude --continue
```

## Platforms

- ✅ macOS (Intel + Apple Silicon)
- ✅ Linux (x86_64 + ARM64)
- ✅ Windows (x86_64)

## Breaking Changes

**v0.2.0 is a complete rewrite** - proxy mode removed

If you were using v0.1.x proxy mode (`ddollar start`):
- ❌ Proxy mode removed (too complex)
- ✅ Use supervisor mode instead: `ddollar <command>`
- No more sudo, no /etc/hosts modification

**Rationale**: Supervisor mode is the KISS solution. One job, done well.

## What Was Removed

- ❌ Proxy server (HTTPS interception)
- ❌ SSL certificate generation
- ❌ /etc/hosts file manipulation
- ❌ Commands: `start`, `stop`, `status`, `trust`, `untrust`

## What Remains

- ✅ Supervisor with rate limit monitoring
- ✅ Token rotation (round-robin)
- ✅ Multi-provider support (Anthropic, OpenAI, Cohere, Google AI)
- ✅ Single binary, zero dependencies

## Supported Providers

- Anthropic Claude
- OpenAI GPT
- Cohere
- Google AI

## Examples

```bash
# All-night AI sessions
ddollar claude --continue

# Long-running scripts
ddollar python train_model.py

# Interactive mode (prompts on limit hit)
ddollar --interactive node agent.js

# Multiple tokens (rotates through all)
export ANTHROPIC_API_KEYS=key1,key2,key3
ddollar claude --continue
```

## Installation

### macOS (Intel)
```bash
curl -LO https://github.com/ahoward/ddollar/releases/download/v0.2.0/ddollar-darwin-amd64
chmod +x ddollar-darwin-amd64
sudo mv ddollar-darwin-amd64 /usr/local/bin/ddollar
```

### macOS (Apple Silicon)
```bash
curl -LO https://github.com/ahoward/ddollar/releases/download/v0.2.0/ddollar-darwin-arm64
chmod +x ddollar-darwin-arm64
sudo mv ddollar-darwin-arm64 /usr/local/bin/ddollar
```

### Linux (x86_64)
```bash
curl -LO https://github.com/ahoward/ddollar/releases/download/v0.2.0/ddollar-linux-amd64
chmod +x ddollar-linux-amd64
sudo mv ddollar-linux-amd64 /usr/local/bin/ddollar
```

### Linux (ARM64)
```bash
curl -LO https://github.com/ahoward/ddollar/releases/download/v0.2.0/ddollar-linux-arm64
chmod +x ddollar-linux-arm64
sudo mv ddollar-linux-arm64 /usr/local/bin/ddollar
```

### Windows (x86_64)
Download `ddollar-windows-amd64.exe` and add to PATH

## Build from Source

```bash
git clone https://github.com/ahoward/ddollar.git
cd ddollar
go build -o ddollar ./src
```

## Full Changelog

- feat: KISS refactor - supervisor-only mode
- feat: flexible token input (comma-separated, file)
- feat: auto-deduplication of tokens
- feat: Windows cross-platform compatibility
- remove: proxy mode (too complex)
- remove: SSL certificate generation
- remove: /etc/hosts manipulation
- reduce: 92% less code, 38% smaller binary

---

*max out those tokens* 💸🔥

<sub>an [#n5](https://www.nickel5.com/) joint 🚬</sub>
