# Supervisor-Only KISS Refactor Plan

## Goal
Strip ddollar down to ONLY the supervisor mode. Maximum KISS.

---

## What Gets Removed

### Delete entirely:
- `src/proxy/` - All proxy code (server.go, cert.go, mkcert.go)
- `src/hosts/` - All hosts file manipulation
- `src/watchdog/` - Watchdog process (for proxy cleanup)
- Proxy-related commands from main.go: `start`, `stop`, `trust`, `untrust`
- All SSL/certificate logic
- All /etc/hosts logic

### What Stays:
- `src/supervisor/` - Core supervisor and monitor
- `src/tokens/` - Token discovery and pool management
- `src/main.go` - Simplified to just supervisor command
- Token discovery from ENV vars

---

## New Structure

```
ddollar/
├── src/
│   ├── main.go           (minimal - just supervise)
│   ├── supervisor/
│   │   ├── supervisor.go (subprocess management)
│   │   └── monitor.go    (rate limit monitoring)
│   └── tokens/
│       ├── discover.go   (find tokens in ENV)
│       ├── pool.go       (manage token rotation)
│       └── providers.go  (provider configs)
├── go.mod
└── README.md
```

---

## Simplified main.go

```go
package main

import (
	"fmt"
	"os"

	"github.com/drawohara/ddollar/src/supervisor"
	"github.com/drawohara/ddollar/src/tokens"
)

const version = "0.2.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "version", "--version", "-v":
		fmt.Printf("ddollar %s\n", version)
	case "help", "--help", "-h":
		printUsage()
	default:
		// Everything else is a command to supervise
		superviseCommand(os.Args[1:])
	}
}

func printUsage() {
	fmt.Println(`ddollar - Never hit token limits again

Usage:
  ddollar [--interactive] <command> [args...]

Examples:
  ddollar claude --continue              # All-night AI sessions
  ddollar python train_model.py          # Long-running scripts
  ddollar --interactive node agent.js    # Prompt on limit hit

Flags:
  --interactive, -i    Prompt user when limit hit (default: auto-rotate)
  --help, -h           Show this help
  --version, -v        Show version

How it works:
  1. Monitors rate limits every 60s
  2. When >95% used → SIGTERM → rotate token → restart
  3. Your command's --continue flag picks up where it left off

Supports: Anthropic · OpenAI · Cohere · Google AI`)
}

func superviseCommand(args []string) {
	interactive := false

	// Check for --interactive flag
	if len(args) > 0 && (args[0] == "--interactive" || args[0] == "-i") {
		interactive = true
		args = args[1:]
	}

	if len(args) == 0 {
		fmt.Println("ERROR: No command specified")
		printUsage()
		os.Exit(1)
	}

	// Discover tokens
	fmt.Println("Discovering API tokens...")
	discovered := tokens.Discover()

	if len(discovered) == 0 {
		fmt.Println("ERROR: No API tokens found in environment.")
		fmt.Println("\nSet one or more:")
		for _, p := range tokens.SupportedProviders {
			for _, envVar := range p.EnvVars {
				fmt.Printf("  export %s=your-token-here\n", envVar)
			}
		}
		os.Exit(1)
	}

	// Create token pool
	pool := tokens.NewPool()
	for _, pt := range discovered {
		if err := pool.AddProvider(pt.Provider, pt.Tokens); err != nil {
			fmt.Printf("Warning: Failed to add provider %s: %v\n", pt.Provider.Name, err)
			continue
		}
	}

	if pool.ProviderCount() == 0 {
		fmt.Println("ERROR: No providers configured")
		os.Exit(1)
	}

	// Run supervisor
	sup := supervisor.New(pool, args, interactive)
	if err := sup.Run(); err != nil {
		fmt.Printf("ERROR: %v\n", err)
		os.Exit(1)
	}
}
```

**Lines**: ~100 (down from 470)

---

## Updated README

```markdown
# 💸 ddollar

> **Never hit token limits again** 🔥

Run AI CLI tools all night with automatic token rotation. Zero config.

```bash
export ANTHROPIC_API_KEY=sk-ant-...
ddollar claude --continue
# Go to bed. Wake up to finished task.
```

## 🎯 What It Does

- 🔁 Monitors rate limits every 60 seconds
- 🌙 Auto-rotates tokens when >95% used
- ⚡ Gracefully restarts with `--continue`
- 💤 Run agents all night, zero babysitting

**Supported**: OpenAI · Anthropic · Cohere · Google AI

---

## 🎬 Quick Start

```bash
# Set tokens
export ANTHROPIC_API_KEY=sk-ant-...
export ANTHROPIC_API_KEY_2=sk-ant-...

# Run any CLI tool with auto rotation
ddollar claude --continue

# Interactive mode (prompts on limit hit)
ddollar --interactive python long_script.py
```

**What happens**:
```
6pm:  Start with token 1
2am:  95% of token 1 used → rotate to token 2 → restart
4am:  95% of token 2 used → rotate to token 3 → restart
8am:  Task done 🎉
```

---

## ⚡ Install

**macOS/Linux**:
```bash
curl -LO https://github.com/drawohara/ddollar/releases/latest/download/ddollar-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m)
chmod +x ddollar-*
sudo mv ddollar-* /usr/local/bin/ddollar
```

**Build from source**:
```bash
git clone https://github.com/drawohara/ddollar.git
cd ddollar
go build -o ddollar ./src
```

---

## 🚀 Usage

```bash
# Run any long-running CLI
ddollar claude --continue
ddollar python train_model.py
ddollar node agent.js

# Interactive mode (get prompted on limit hit)
ddollar --interactive claude --continue
```

**Multiple tokens** (comma-separated or numbered env vars):
```bash
export ANTHROPIC_API_KEY=sk-ant-1...
export ANTHROPIC_API_KEY_2=sk-ant-2...
export ANTHROPIC_API_KEY_3=sk-ant-3...

ddollar claude --continue
# Rotates through all 3 tokens
```

---

## 🛠️ How It Works

1. Spawns your command with token in ENV
2. Makes 1-token API call every 60s to check rate limits
3. When >95% used → SIGTERM subprocess → rotate token → restart
4. Your tool's `--continue` flag picks up where it left off

**KISS**: No proxy, no DNS, no config. Just process supervision + token rotation.

---

## 🐛 Troubleshooting

- "No tokens found" → Set `ANTHROPIC_API_KEY` (etc) in shell
- Process won't rotate → Tool must support `--continue` flag
- Limit hit before rotation → Tokens hitting limits faster than 60s check interval

---

## 📦 Platforms

✅ macOS (Intel + Apple Silicon)
✅ Linux (x86_64 + ARM64)
✅ Windows (x86_64)

Single binary. No dependencies. No runtime.

---

*max out those tokens* 💸🔥

<sub>an [#n5](https://www.nickel5.com/) joint 🚬</sub>
```

---

## Migration Notes

**What users lose**:
- Proxy mode (intercepts all apps)
- Auto-SSL certificate generation
- /etc/hosts manipulation
- Works-with-everything capability

**What users gain**:
- Simpler mental model (just one thing: supervise)
- No sudo required
- Smaller binary
- Clearer purpose: "Run CLI tools all night"

**Breaking change**: Yes - proxy mode completely removed

---

## Implementation Steps

1. Delete proxy/, hosts/, watchdog/ directories
2. Simplify main.go (remove all proxy commands)
3. Update README (supervisor-only)
4. Bump version to 0.2.0
5. Test: `ddollar claude --continue` (if you have claude)
6. Test: `ddollar bash -c 'sleep 10'`
7. Build and commit

---

## File Sizes

**Before** (two modes):
- Binary: ~12 MB
- Source: 56 files, 12,541 lines

**After** (supervisor only):
- Binary: ~4 MB (estimate)
- Source: ~20 files, ~1,500 lines

**Reduction**: 66% smaller binary, 88% fewer lines

---

## Verdict

This is the KISS version. One job: **Run CLI tools all night without hitting token limits.**

No proxy complexity. No sudo. No /etc/hosts hacks. Just supervision.
