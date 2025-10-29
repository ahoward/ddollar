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
curl -LO https://github.com/ahoward/ddollar/releases/latest/download/ddollar-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m)
chmod +x ddollar-*
sudo mv ddollar-* /usr/local/bin/ddollar
```

**Build from source**:
```bash
git clone https://github.com/ahoward/ddollar.git
cd ddollar
go build -o ddollar ./src
```

---

## 🚀 Usage

```bash
# Validate your token setup first
ddollar --validate

# Run any long-running CLI
ddollar claude --continue
ddollar python train_model.py
ddollar node agent.js

# Interactive mode (get prompted on limit hit)
ddollar --interactive claude --continue
```

**Multiple tokens** (3 ways):
```bash
# 1. Comma-separated
export ANTHROPIC_API_KEY=sk-ant-primary...
export ANTHROPIC_API_KEYS=sk-ant-1...,sk-ant-2...,sk-ant-3...

# 2. File with one token per line
echo "sk-ant-1..." > ~/.ddollar-keys
echo "sk-ant-2..." >> ~/.ddollar-keys
export ANTHROPIC_API_KEYS_FILE=~/.ddollar-keys

# 3. Mix and match (all get deduplicated)
export ANTHROPIC_API_KEY=sk-ant-primary...
export ANTHROPIC_API_KEYS=sk-ant-1...,sk-ant-2...
export ANTHROPIC_API_KEYS_FILE=~/.ddollar-keys

ddollar claude --continue
# Rotates through ALL discovered tokens
```

---

## 🛠️ How It Works

1. Spawns your command with token in ENV
2. Makes 1-token API call every 60s to check rate limits
3. When >95% used → SIGTERM subprocess → rotate token → restart
4. Your tool's `--continue` flag picks up where it left off

**KISS**: No proxy, no DNS, no config. Just process supervision + token rotation.

---

## 🕵️ Tor Integration (Mask Your IP)

Use ddollar with Tor to anonymize your API requests:

```bash
# Install Tor
sudo apt-get install tor        # Linux
brew install tor                # macOS

# Start Tor daemon
sudo systemctl start tor        # Linux
brew services start tor         # macOS

# Run through Tor
torify ddollar claude --continue
```

**Benefits**:
- 🔒 Hide your IP from AI providers
- 🔄 New IP with each token rotation
- 🌍 Geographic diversity (different exit nodes)
- 🛡️ Multi-account isolation

**Advanced usage**:
```bash
# Multiple tokens + Tor = new IP per rotation
export ANTHROPIC_API_KEYS=key1,key2,key3
torify ddollar claude --continue

# Verify Tor is working
torify curl -s https://api.ipify.org
```

See [docs/TOR_INTEGRATION.md](docs/TOR_INTEGRATION.md) for:
- Per-token Tor circuits
- Auto-renewing circuits on rotation
- Country-specific exit nodes
- Performance optimization
- Security considerations

---

## 🔍 Validate Your Setup

Before running long sessions, validate your tokens:

```bash
ddollar --validate
```

**Example output**:
```
🔍 Validating tokens...

[1/3] Testing Anthropic token...
  ✓ Valid
    Requests: 4850/5000 remaining (3.0% used)
    Tokens:   95234/100000 remaining (4.8% used)
    Reset:    52m 18s

[2/3] Testing Anthropic token...
  ✓ Valid
    Requests: 5000/5000 remaining (0.0% used)
    Tokens:   100000/100000 remaining (0.0% used)

[3/3] Testing OpenAI token...
  ✗ FAILED: HTTP 401: authentication failed or invalid token

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Summary: 2 valid, 1 invalid, 3 total
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

This tests each token with a minimal API call and shows:
- ✓ Token validity
- Current rate limit usage
- Time until rate limits reset

---

## 🐛 Troubleshooting

- **"No tokens found"** → Set `ANTHROPIC_API_KEY` (etc) in shell
- **Token validation fails** → Run `ddollar --validate` to test each token
- **Process won't rotate** → Tool must support `--continue` flag
- **Limit hit before rotation** → Tokens hitting limits faster than 60s check interval

---

## 📦 Platforms

✅ macOS (Intel + Apple Silicon)
✅ Linux (x86_64 + ARM64)
✅ Windows (x86_64)

Single binary. No dependencies. No runtime.

---

## 🤝 Contributing

PRs welcome. Issues welcome. [GitHub](https://github.com/ahoward/ddollar)

---

*max out those tokens* 💸🔥

<sub>an [#n5](https://www.nickel5.com/) joint 🚬</sub>
