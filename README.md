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
# Run any long-running CLI
ddollar claude --continue
ddollar python train_model.py
ddollar node agent.js

# Interactive mode (get prompted on limit hit)
ddollar --interactive claude --continue
```

**Multiple tokens**:
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

- **"No tokens found"** → Set `ANTHROPIC_API_KEY` (etc) in shell
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
