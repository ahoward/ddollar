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

## 🎬 TL;DR - See It Work

```bash
# 1. Set your token(s)
export ANTHROPIC_API_KEY=sk-ant-api03-...

# 2. Start ddollar (run in background or separate terminal)
sudo ddollar start &

# 3. Hit Claude API - ddollar intercepts and injects your token
curl https://api.anthropic.com/v1/messages \
  -H "content-type: application/json" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "claude-3-5-sonnet-20241022",
    "max_tokens": 16,
    "messages": [{"role": "user", "content": "Say hello"}]
  }'
```

**Output you'll see:**
```
# ddollar logs:
[POST] api.anthropic.com /v1/messages
Injected token for api.anthropic.com (provider: Anthropic)

# API response:
{"id":"msg_...","content":[{"text":"Hello!","type":"text"}],...}
```

**That's it.** No SDK config. No manual headers. Just works. 🔥

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

## 🔒 SSL Certificates (Automatic!)

**Zero config needed** - `sudo ddollar start` automatically creates and trusts SSL certificates. Just run it and go! 🎉

### What Happens Automatically

1. ✓ Creates local Certificate Authority (CA)
2. ✓ Installs CA to system trust stores
3. ✓ Generates SSL certificate for AI providers
4. ✓ HTTPS requests work immediately - no SSL warnings

**Supported**: macOS, Linux (Ubuntu/Debian, RHEL/Fedora), Windows, Firefox (NSS)

### Manual Control (Optional)

If automatic trust fails or you want explicit control:

```bash
# Manually install trust
sudo ddollar trust

# Remove trust
sudo ddollar untrust

# Check certificate status
ddollar status
```

### Fallback: Manual Trust (If Needed)

Only needed if running without sudo or in restricted environments:

**macOS**:
```bash
sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain ~/.ddollar/ca/rootCA.pem
```

**Linux (Debian/Ubuntu)**:
```bash
sudo cp ~/.ddollar/ca/rootCA.pem /usr/local/share/ca-certificates/ddollar.crt
sudo update-ca-certificates
```

**Linux (RHEL/Fedora)**:
```bash
sudo cp ~/.ddollar/ca/rootCA.pem /etc/pki/ca-trust/source/anchors/ddollar.pem
sudo update-ca-trust
```

**Windows** (PowerShell as Admin):
```powershell
certutil -addstore -f "ROOT" $env:USERPROFILE\.ddollar\ca\rootCA.pem
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

**an n5 joint 🚬**

---

*max out those tokens* 💸🔥
