# Tor Integration Guide

## Overview

ddollar can be combined with Tor (`torify`, `torsocks`, or `proxychains-ng`) to mask your IP address when making API requests. This is useful for:

- **Privacy**: Prevent AI providers from tracking your location/IP
- **Multi-account isolation**: Different Tor circuits for different tokens
- **Rate limit circumvention**: Fresh IP with each token rotation
- **Geographic diversity**: Appear from different locations

## 🎯 Quick Start

### Option 1: torify (simplest)

```bash
# Install Tor
sudo apt-get install tor  # Debian/Ubuntu
brew install tor          # macOS

# Start Tor daemon
sudo systemctl start tor  # Linux
brew services start tor   # macOS

# Run ddollar through Tor
export ANTHROPIC_API_KEY=sk-ant-...
torify ddollar claude --continue
```

**How it works**: `torify` wraps the entire process and routes all TCP through Tor's SOCKS proxy (localhost:9050).

### Option 2: torsocks (more control)

```bash
# Install torsocks
sudo apt-get install torsocks  # Debian/Ubuntu
brew install torsocks          # macOS

# Run with custom Tor config
torsocks ddollar claude --continue
```

### Option 3: proxychains-ng (advanced)

```bash
# Install proxychains-ng
sudo apt-get install proxychains4  # Debian/Ubuntu
brew install proxychains-ng        # macOS

# Configure /etc/proxychains.conf
echo "socks5 127.0.0.1 9050" >> /etc/proxychains.conf

# Run through proxychains
proxychains4 -q ddollar claude --continue
```

---

## 🔄 Tor + Token Rotation = Maximum Privacy

The killer combo: **new IP with every token rotation**

```bash
# Setup: Multiple tokens + Tor circuit renewal
export ANTHROPIC_API_KEYS=key1,key2,key3

# Create script to renew Tor circuit on rotation
cat > ~/ddollar-tor-rotate.sh <<'EOF'
#!/bin/bash
# Signal Tor to get new circuit
echo -e 'AUTHENTICATE ""\r\nSIGNAL NEWNYM\r\nQUIT' | nc 127.0.0.1 9051
EOF
chmod +x ~/ddollar-tor-rotate.sh

# Run ddollar with Tor
torify ddollar claude --continue
```

**What happens**:
```
6pm:  Token 1 → Tor exit node in Germany
2am:  95% used → rotate to Token 2 → restart
      Token 2 → New Tor circuit → exit node in France
4am:  95% used → rotate to Token 3 → restart
      Token 3 → New Tor circuit → exit node in Netherlands
8am:  Task done, 3 different IPs used 🎉
```

---

## 🛡️ Advanced: Per-Token Tor Circuits

Want a dedicated Tor circuit per token? Use multiple Tor instances:

```bash
# Start 3 Tor instances on different ports
tor --SOCKSPort 9050 --ControlPort 9051 &  # Token 1
tor --SOCKSPort 9052 --ControlPort 9053 &  # Token 2
tor --SOCKSPort 9054 --ControlPort 9055 &  # Token 3

# Create wrapper script per token
cat > ~/ddollar-multi-tor.sh <<'EOF'
#!/bin/bash
TOKEN_INDEX=${1:-0}
SOCKS_PORT=$((9050 + TOKEN_INDEX * 2))

ALL_PROXY=socks5h://127.0.0.1:$SOCKS_PORT \
ANTHROPIC_API_KEY=$(sed -n "$((TOKEN_INDEX+1))p" ~/.ddollar-keys) \
  ddollar claude --continue
EOF
chmod +x ~/ddollar-multi-tor.sh

# Run with dedicated circuit per token
~/ddollar-multi-tor.sh 0  # Uses port 9050
```

---

## 🧪 Verify Tor is Working

Test that your IP is being masked:

```bash
# Check your current IP (without Tor)
curl -s https://api.ipify.org
# Output: 203.0.113.45 (your real IP)

# Check IP through Tor
torify curl -s https://api.ipify.org
# Output: 185.220.101.35 (Tor exit node IP)

# Check IP during ddollar operation
torify ddollar bash -c 'curl -s https://api.ipify.org'
# Output: 185.220.101.35 (Tor exit node IP)
```

---

## 🔧 Tor Configuration

### Optimize for ddollar

Create `~/.torrc`:

```
# Faster circuit building
CircuitBuildTimeout 10
LearnCircuitBuildTimeout 0

# Use bridges if Tor is blocked
# UseBridges 1
# Bridge obfs4 ...

# Control port for circuit renewal
ControlPort 9051
CookieAuthentication 0

# Prefer fast exit nodes
ExitNodes {DE},{FR},{NL},{SE}
StrictNodes 0
```

Start Tor with custom config:
```bash
tor -f ~/.torrc
```

### Auto-renew circuits on token rotation

ddollar restarts the supervised process on rotation. Use this to renew Tor circuits:

```bash
# Create pre-launch hook
cat > ~/ddollar-renew-tor.sh <<'EOF'
#!/bin/bash
# Renew Tor circuit before each launch
echo -e 'AUTHENTICATE ""\r\nSIGNAL NEWNYM\r\nQUIT' | nc 127.0.0.1 9051
sleep 5  # Wait for new circuit
EOF
chmod +x ~/ddollar-renew-tor.sh

# Wrap ddollar command
cat > ~/ddollar-tor-wrapped.sh <<'EOF'
#!/bin/bash
~/ddollar-renew-tor.sh
exec "$@"
EOF
chmod +x ~/ddollar-tor-wrapped.sh

# Run with auto-renewal
torify ddollar ~/ddollar-tor-wrapped.sh claude --continue
```

---

## 🌍 Geographic Diversity

Force specific countries for exit nodes:

```bash
# Use only US exit nodes
cat > ~/.torrc <<EOF
ExitNodes {US}
StrictNodes 1
EOF

# Or rotate through specific countries
ExitNodes {US},{GB},{CA},{AU},{DE},{FR},{NL},{SE},{CH}
StrictNodes 0
```

Country codes: https://en.wikipedia.org/wiki/ISO_3166-1_alpha-2

---

## 🚨 Limitations & Considerations

### Performance Impact
- **Latency**: Tor adds 1-3 seconds per request (3-hop circuit)
- **Throughput**: Limited by Tor exit node bandwidth
- **Not suitable for**: Real-time applications, high-frequency APIs

### API Provider Policies
- Some APIs block Tor exit nodes
- Rate limits may be stricter from Tor IPs
- Check provider ToS before using Tor

### Tor Network Health
- Don't abuse Tor bandwidth for bulk operations
- Consider running a Tor relay to give back
- Use `--interactive` mode to minimize unnecessary traffic

---

## 💡 Use Cases

### 1. Privacy-Conscious Development
```bash
# Develop AI apps without leaking your location
torify ddollar claude --continue
```

### 2. Multi-Account Management
```bash
# Different IP per token = better account isolation
export ANTHROPIC_API_KEYS=account1,account2,account3
torify ddollar claude --continue
```

### 3. Testing Geographic Restrictions
```bash
# Test how your app behaves from different countries
# Set ExitNodes in ~/.torrc to target country
torify ddollar python test_geo_features.py
```

### 4. Research & Analysis
```bash
# Scrape/analyze without IP-based tracking
torify ddollar python research_script.py
```

---

## 🔗 Resources

- **Tor Project**: https://www.torproject.org/
- **Tor Manual**: https://2019.www.torproject.org/docs/tor-manual.html.en
- **torsocks**: https://gitlab.torproject.org/tpo/core/torsocks
- **proxychains-ng**: https://github.com/rofl0r/proxychains-ng

---

## ⚠️ Disclaimer

This guide is for **legitimate privacy protection** only. ddollar + Tor should be used responsibly:

- ✅ Protecting your privacy during development
- ✅ Testing geographic features
- ✅ Isolating multi-account usage
- ❌ Violating API provider Terms of Service
- ❌ Evading legitimate rate limits through abuse
- ❌ Unauthorized access or malicious activity

Always comply with your API provider's Terms of Service and Acceptable Use Policy.

---

*anonymize responsibly* 🕵️‍♂️🔥
