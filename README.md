# ddollar

**DDoS for tokens** - A high-performance proxy that intercepts requests to AI providers and rotates multiple tokens to maximize API usage across accounts.

ddollar creates a transparent proxy that:
- Intercepts requests to various AI providers (OpenAI, Anthropic, etc.)
- Alters local DNS so ALL traffic from ANY program goes through the proxy
- Scans configuration files or environment variables to discover available tokens
- Automatically rotates tokens across multiple accounts to distribute load and maximize throughput

**Simple**: Install, start, let it auto-configure
**Advanced**: Full control over token rotation, rate limiting, and provider configuration

---

## Installation

### System Requirements

- **Supported Platforms**: macOS (Intel/Apple Silicon), Linux (x86_64/ARM64), Windows (x86_64)
- **Prerequisites**: None (standalone binary)
- **Network**: May require sudo/administrator privileges for DNS modifications

Download the latest release from [GitHub Releases](https://github.com/drawohara/ddollar/releases).

### macOS

**Intel (x86_64)**:
```bash
# Download ddollar binary for macOS Intel
curl -LO https://github.com/drawohara/ddollar/releases/latest/download/ddollar-macos-x86_64

# Make executable
chmod +x ddollar-macos-x86_64

# Move to PATH
sudo mv ddollar-macos-x86_64 /usr/local/bin/ddollar
```

**Apple Silicon (ARM64)**:
```bash
# Download ddollar binary for macOS ARM
curl -LO https://github.com/drawohara/ddollar/releases/latest/download/ddollar-macos-arm64

# Make executable
chmod +x ddollar-macos-arm64

# Move to PATH
sudo mv ddollar-macos-arm64 /usr/local/bin/ddollar
```

### Linux

**x86_64**:
```bash
# Download ddollar binary for Linux x86_64
curl -LO https://github.com/drawohara/ddollar/releases/latest/download/ddollar-linux-x86_64

# Make executable
chmod +x ddollar-linux-x86_64

# Move to PATH
sudo mv ddollar-linux-x86_64 /usr/local/bin/ddollar
```

**ARM64**:
```bash
# Download ddollar binary for Linux ARM64
curl -LO https://github.com/drawohara/ddollar/releases/latest/download/ddollar-linux-arm64

# Make executable
chmod +x ddollar-linux-arm64

# Move to PATH
sudo mv ddollar-linux-arm64 /usr/local/bin/ddollar
```

### Windows

**x86_64**:

Download [ddollar-windows-x86_64.exe](https://github.com/drawohara/ddollar/releases/latest/download/ddollar-windows-x86_64.exe) and place it in your PATH.

Or using PowerShell:
```powershell
# Download ddollar binary for Windows
Invoke-WebRequest -Uri "https://github.com/drawohara/ddollar/releases/latest/download/ddollar-windows-x86_64.exe" -OutFile "ddollar.exe"

# Move to a directory in your PATH (example: C:\Windows\System32)
# Or create a custom directory and add it to your PATH
Move-Item ddollar.exe C:\Windows\System32\ddollar.exe
```

### Alternative Installation Methods

**Homebrew (macOS/Linux)**:
```bash
# Coming soon - homebrew tap in development
brew install drawohara/tap/ddollar
```

**Cargo (Rust)**:
```bash
# If published to crates.io
cargo install ddollar
```

**Build from Source**:
```bash
# Clone the repository
git clone https://github.com/drawohara/ddollar.git
cd ddollar

# Build with Cargo (requires Rust toolchain)
cargo build --release

# Install binary
sudo mv target/release/ddollar /usr/local/bin/
```

---

## Quick Start

Verify installation:
```bash
ddollar --version
```

Expected output:
```
ddollar x.y.z
```

Start the proxy with auto-configuration:
```bash
ddollar start
```

The proxy will:
1. Scan for API tokens in environment variables and config files
2. Configure DNS to route AI provider traffic through the proxy
3. Begin rotating tokens automatically

Check proxy status:
```bash
ddollar status
```

Stop the proxy:
```bash
ddollar stop
```

For detailed configuration options:
```bash
ddollar --help
```

---

## Troubleshooting

### "Permission denied" when running ddollar

Ensure the binary is executable:
```bash
chmod +x /usr/local/bin/ddollar
```

### "Command not found" after installation

Verify `/usr/local/bin` is in your PATH:
```bash
echo $PATH | grep /usr/local/bin
```

If not present, add to your shell profile (`~/.bashrc`, `~/.zshrc`, `~/.config/fish/config.fish`):
```bash
export PATH="/usr/local/bin:$PATH"
```

Then reload your shell:
```bash
source ~/.bashrc  # or ~/.zshrc
```

### macOS: "Cannot be opened because the developer cannot be verified"

This is macOS Gatekeeper. Two options:

1. Right-click the binary, select "Open", click "Open" in the dialog
2. Remove quarantine attribute:
```bash
xattr -d com.apple.quarantine /usr/local/bin/ddollar
```

### Downloaded the wrong architecture

Check your system architecture:
```bash
# macOS/Linux
uname -m
```

Output indicates architecture:
- `x86_64`: Intel/AMD 64-bit
- `arm64` or `aarch64`: ARM 64-bit

Download the matching binary from [releases](https://github.com/drawohara/ddollar/releases).

### DNS modifications require sudo/administrator privileges

The proxy modifies local DNS to intercept AI provider traffic. On Unix systems:
```bash
# Run with sudo when starting
sudo ddollar start
```

On Windows, run PowerShell or Command Prompt as Administrator.

---

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## Contributing

Issues and pull requests are welcome! Please visit the [GitHub repository](https://github.com/drawohara/ddollar) to:
- Report bugs or issues
- Request features
- Submit pull requests

---

## Sponsor

**n5**

---

*DDoS for tokens - burn them to the ground* 🔥
