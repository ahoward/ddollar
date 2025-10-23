# 💸 ddollar

> **DDoS for tokens - burn them to the ground** 🔥

**Two badass modes to never hit rate limits again:**

**Proxy mode**: Intercepts all apps at network layer
**Supervisor mode**: Runs CLI tools all night with auto token rotation

```bash
# Proxy: Works with everything
export ANTHROPIC_API_KEY=sk-ant-...
sudo -E ddollar start

# Supervisor: All-night AI sessions
ddollar supervise -- claude --continue
# go to bed, wake up to finished task
```

## 🎯 Two Badass Modes

**Proxy Mode**:
- 🔀 Rotates tokens round-robin on every request
- 🌐 Intercepts ALL apps via `/etc/hosts` hack
- 🤖 Auto-discovers tokens from ENV
- 🚀 Zero config - just works

**Supervisor Mode**:
- 🔁 Monitors rate limits, auto-rotates tokens
- 🌙 Run AI agents all night without babysitting
- ⚡ Gracefully restarts on limit hit with `--continue`
- 💤 No interruptions, no token limit errors ever

**Supported**: OpenAI · Anthropic · Cohere · Google AI

---

## 🎬 Quick Start

### Proxy Mode

```bash
export ANTHROPIC_API_KEY=sk-ant-...
sudo -E ddollar start
# Done. Every app rotates tokens now.
```

### Supervisor Mode

```bash
export ANTHROPIC_API_KEY=sk-ant-...
ddollar supervise -- claude --continue
# Go to bed. Wake up to finished task.
```

**How supervisor works**:
```
6pm:  Start task with token 1
...
2am:  95% of token 1 used
      → SIGTERM → rotate to token 2 → restart with --continue
      → Session resumes seamlessly
...
4am:  95% of token 2 used
      → Rotate to token 3
...
8am:  You wake up. Task is done. 🎉
```

---

## ⚡ Install

**macOS/Linux**:
```bash
# Grab binary (swap arch if needed: x86_64, arm64)
curl -LO https://github.com/drawohara/ddollar/releases/latest/download/ddollar-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m)
chmod +x ddollar-*
sudo mv ddollar-* /usr/local/bin/ddollar
```

**Windows**: [Download exe](https://github.com/drawohara/ddollar/releases) → Drop in `C:\Windows\System32`

**Build from source**:
```bash
git clone https://github.com/drawohara/ddollar.git
cd ddollar
go build -o ddollar ./src
```

---

## 🚀 Usage

### Proxy Mode

```bash
# Set tokens
export OPENAI_API_KEY=sk-proj-...
export ANTHROPIC_API_KEY=sk-ant-...

# Check discovered tokens
ddollar status

# Start proxy (use -E to preserve env vars)
sudo -E ddollar start

# OR pass inline
sudo ANTHROPIC_API_KEY=$ANTHROPIC_API_KEY ddollar start

# Every app now rotates tokens
python my_script.py
curl https://api.openai.com/v1/models

# Stop
sudo ddollar stop
```

### Supervisor Mode

```bash
# Run any long-running CLI with auto rotation
ddollar supervise -- claude --continue
ddollar supervise -- python train_model.py
ddollar supervise -- node agent.js

# Get prompted when limit hit
ddollar supervise --interactive -- claude --continue
```

**Supervisor handles**:
- Monitors rate limits every 60s
- Auto-rotates at 95% usage
- Gracefully stops subprocess (SIGTERM)
- Restarts with new token
- Works with any tool that reads tokens from ENV

---

## 🛠️ How It Works

### Proxy Mode

1. Modifies `/etc/hosts` → `api.openai.com` points to `127.0.0.1`
2. HTTPS proxy on port 443 with auto-trusted SSL certs
3. Intercepts requests → injects rotated token → forwards to real API
4. Round-robin rotation on every request

### Supervisor Mode

1. Spawns your command with token in ENV
2. Makes 1-token API call every 60s to check rate limits
3. When >95% used → SIGTERM subprocess → rotate token → restart
4. Your tool's `--continue` flag picks up where it left off

**KISS**: No DNS, no daemons, no config. Just a proxy, some forking, and `/etc/hosts` magic.

---

## 🤔 Which Badass Mode Should I Use?

| Use Case | Mode | Why |
|----------|------|-----|
| Multiple apps/tools at once | **Proxy** | Intercepts everything, zero per-app config |
| GUIs, browsers, etc. | **Proxy** | Works at network layer |
| All-night AI agent sessions | **Supervisor** | Monitors limits, auto-rotates, never stops |
| Long-running CLI tools | **Supervisor** | Graceful rotation with `--continue` |
| Want detailed rate limit visibility | **Supervisor** | Logs usage every 60s |
| Just want it to work everywhere | **Proxy** | Set and forget |

**Pro tip**: Use both badass modes. Proxy for day-to-day, supervisor for overnight agents.

---

## 🔒 SSL Certificates (Proxy Mode Only)

**Auto-configured** - `sudo ddollar start` creates and trusts SSL certs. Done.

Manual control if needed:
```bash
sudo ddollar trust    # Install cert trust
sudo ddollar untrust  # Remove cert trust
ddollar status        # Check cert status
```

---

## 🐛 Troubleshooting

**Proxy mode**:
- "Permission denied" → Need `sudo` for port 443 + `/etc/hosts`
- "No tokens found" → Use `sudo -E` to preserve env vars
- macOS Gatekeeper → `xattr -d com.apple.quarantine /usr/local/bin/ddollar`

**Supervisor mode**:
- "No tokens found" → Set `ANTHROPIC_API_KEY` (etc) in shell
- Process won't rotate → Tool must support `--continue` flag
- Limit hit before rotation → Reduce check interval (TODO: add flag)

**Both**:
- "Command not found" → Add `/usr/local/bin` to `$PATH`
- Wrong arch → Check `uname -m`, download correct binary

---

## 📦 Platforms

✅ macOS (Intel + Apple Silicon)
✅ Linux (x86_64 + ARM64)
✅ Windows (x86_64)

Single binary. No dependencies. No runtime.

---

## 🤝 Contributing

PRs welcome. Issues welcome. [GitHub](https://github.com/drawohara/ddollar)

---

## 💰 Sponsor

**an n5 joint 🚬**

---

*max out those tokens* 💸🔥
