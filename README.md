# 💸 ddollar

> **DDoS for tokens - burn them to the ground** 🔥

Transparent HTTPS proxy that rotates AI provider tokens automatically. Install → Set env vars → Run. Zero config.

```bash
export OPENAI_API_KEY=sk-proj-...
export ANTHROPIC_API_KEY=sk-ant-...
sudo ddollar start
# every app now rotates tokens automatically
```

## 🎯 What It Does

- 🔀 **Rotates tokens**: Round-robin across all your API keys
- 🌐 **Intercepts everything**: Modifies `/etc/hosts` - ALL apps use the proxy
- 🤖 **Auto-discovers tokens**: Scans `OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, etc.
- 🚀 **Zero config**: No files, no setup - just works

**Supported**: OpenAI · Anthropic · Cohere · Google AI

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

```bash
# Set your tokens
export OPENAI_API_KEY=sk-proj-abc123
export ANTHROPIC_API_KEY=sk-ant-xyz789

# Check discovered tokens
ddollar status

# Start proxy (requires sudo for /etc/hosts)
sudo ddollar start

# Use any app normally - tokens rotate automatically
curl https://api.openai.com/v1/models
python my_openai_script.py

# Stop and cleanup
sudo ddollar stop
```

**That's it.** Apps hit the proxy, tokens rotate, requests go through.

---

## 🛠️ How It Works

1. Modifies `/etc/hosts` to point `api.openai.com` → `127.0.0.1`
2. Starts HTTPS proxy on port 443
3. Intercepts requests, injects rotated token
4. Forwards to real API

**KISS**: No DNS servers, no daemons, no config files. Just a proxy + hosts file.

---

## ⚠️ First Run

Self-signed cert needed for HTTPS interception. Trust it once:

**macOS**:
```bash
sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain ~/.ddollar/cert.pem
```

**Linux**:
```bash
sudo cp ~/.ddollar/cert.pem /usr/local/share/ca-certificates/ddollar.crt
sudo update-ca-certificates
```

**Windows** (PowerShell as Admin):
```powershell
certutil -addstore -f "ROOT" $env:USERPROFILE\.ddollar\cert.pem
```

---

## 🐛 Troubleshooting

**"Permission denied"** → Need `sudo` (port 443 + `/etc/hosts`)

**"Command not found"** → Add `/usr/local/bin` to `$PATH`

**macOS Gatekeeper** → `xattr -d com.apple.quarantine /usr/local/bin/ddollar`

**Wrong arch** → Check with `uname -m`, download correct binary

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

**n5**

---

*max out those tokens* 💸🔥
