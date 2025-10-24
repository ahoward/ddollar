# Create GitHub Release v0.2.0

## Option 1: Using GitHub CLI (gh)

```bash
# Create release with binaries
gh release create v0.2.0 \
  --title "v0.2.0 - KISS Release" \
  --notes-file releases/RELEASE_NOTES.md \
  releases/ddollar-darwin-amd64 \
  releases/ddollar-darwin-arm64 \
  releases/ddollar-linux-amd64 \
  releases/ddollar-linux-arm64 \
  releases/ddollar-windows-amd64.exe
```

## Option 2: Using GitHub Web UI

1. Go to https://github.com/ahoward/ddollar/releases/new
2. Choose tag: `v0.2.0`
3. Release title: `v0.2.0 - KISS Release`
4. Description: Copy from `releases/RELEASE_NOTES.md`
5. Upload binaries:
   - `releases/ddollar-darwin-amd64`
   - `releases/ddollar-darwin-arm64`
   - `releases/ddollar-linux-amd64`
   - `releases/ddollar-linux-arm64`
   - `releases/ddollar-windows-amd64.exe`
6. Click "Publish release"

## Binaries Ready

All binaries are in `releases/` directory:

```
releases/
├── ddollar-darwin-amd64      (5.1M) - macOS Intel
├── ddollar-darwin-arm64      (4.9M) - macOS Apple Silicon
├── ddollar-linux-amd64       (5.0M) - Linux x86_64
├── ddollar-linux-arm64       (4.8M) - Linux ARM64
├── ddollar-windows-amd64.exe (5.1M) - Windows x86_64
└── RELEASE_NOTES.md
```

## After Release

Users can install with:

```bash
# macOS/Linux
curl -LO https://github.com/ahoward/ddollar/releases/download/v0.2.0/ddollar-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m)
chmod +x ddollar-*
sudo mv ddollar-* /usr/local/bin/ddollar
```

## Verify

Test the release:
```bash
ddollar --version
# Should output: ddollar 0.2.0
```
