# 💾 How to Install SPECTRE

This guide walks you through installing SPECTRE step-by-step on Windows, Linux, or macOS.

---

## What You'll Need

Before starting, make sure you have these installed:

| What | Minimum Version | Check with |
|------|---|---|
| **Go** | 1.21+ | `go version` |
| **Python** | 3.10+ | `python --version` |
| **Git** | 2.x+ | `git --version` |

- **Local LLM:** Ollama ([ollama.ai](https://ollama.ai)) or LM Studio ([lmstudio.ai](https://lmstudio.ai)) for private offline AI.
- **Cloud LLM:** API key for OpenAI, Anthropic Claude, Google Gemini, Groq, or OpenRouter.

---

## Installation Methods

You can install SPECTRE using either:
1. **Source Installation (Recommended for Developers & Contributors)** — Available on Windows, Linux, and macOS.
2. **Pre-built Silent Installer (Windows Only)** — Single-command automated installation.

---

## Method 1: Build from Source (All Platforms)

### Windows (PowerShell)

**Step 1:** Clone the repository:
```powershell
git clone https://github.com/Gokul-Eswar/Spectre.git
cd Spectre
```

**Step 2:** Build the Go binary:
```powershell
go build -o spectre.exe cmd/spectre/main.go
```

**Step 3:** Set up the Python virtual environment and analyzer dependencies:
```powershell
python -m venv .venv
.\.venv\Scripts\Activate.ps1
pip install --upgrade pip
pip install -r analyzer/requirements.txt
```

*(Optional) Using Make on Windows:*
If you have `make` installed:
```powershell
make install
```

---

### Linux & macOS (Bash)

**Step 1:** Clone the repository:
```bash
git clone https://github.com/Gokul-Eswar/Spectre.git
cd Spectre
```

**Step 2:** Build the Go executable:
```bash
go build -o spectre cmd/spectre/main.go
chmod +x spectre
```

**Step 3:** Set up the Python virtual environment and analyzer dependencies:
```bash
python3 -m venv .venv
source .venv/bin/activate
pip install --upgrade pip
pip install -r analyzer/requirements.txt
```

*(Optional) Using Make:*
```bash
make install
```

---

## Method 2: Windows Pre-built Installer (Automated)

### One-Line Install
Open PowerShell as Administrator and run:
```powershell
powershell -NoProfile -Command "$u='https://github.com/Gokul-Eswar/Spectre/releases/latest/download/Spectre-Setup-x64.exe';$o=\"$env:TEMP\Spectre-Setup.exe\";Invoke-WebRequest $u -OutFile $o; Start-Process -FilePath $o -ArgumentList '/VERYSILENT','/NORESTART' -Wait"
```

### Recommended Secure Install (with Checksum Verification)
Replace `<SHA256_HASH>` with the checksum listed on the release page:
```powershell
$url = 'https://github.com/Gokul-Eswar/Spectre/releases/latest/download/Spectre-Setup-x64.exe'
$out = "$env:TEMP\Spectre-Setup.exe"
Invoke-WebRequest -Uri $url -OutFile $out
$expected = '<SHA256_HASH>'
if ((Get-FileHash -Path $out -Algorithm SHA256).Hash -ne $expected) {
    Write-Error 'Checksum mismatch — aborting'
    exit 1
}
Start-Process -FilePath $out -ArgumentList '/VERYSILENT','/NORESTART' -Wait
```

This will:
- ✓ Install SPECTRE to `C:\Program Files\SPECTRE`
- ✓ Automatically add SPECTRE to your system `PATH`
- ✓ Create Start Menu shortcuts
- ✓ Register uninstaller in Windows Settings / Add or Remove Programs

---

## Verify Installation

Check the installed version:

```bash
# Windows
.\spectre.exe version

# Linux/macOS
./spectre version
```

You should see: `SPECTRE v1.0.0` ✓

### Launch the Modern Horizontal Console

To launch the full-width interactive console:

```bash
# Windows
.\spectre.exe console
# or simply:
.\spectre.exe

# Linux/macOS
./spectre console
# or simply:
./spectre
```

When the console opens:
- The top header will display **`✻ SPECTRE`** on the left and **`Model: <configured_model>`** (or **`Model: none`** if unconfigured) on the right.
- Type `?` into the chatbar at any time to inspect all available commands (`?cases`, `?model`, `?analyze`, `?evidence`, `?graph`, `?report`, etc.).

---

## Run the Test Suite

Verify that all backend Go modules and Python analyzer tests pass:

```bash
# Test Go modules
go test ./...

# Test Python analyzer (Linux/macOS)
.venv/bin/python -m pytest analyzer/

# Test Python analyzer (Windows)
.venv\Scripts\python -m pytest analyzer/

# Alternatively, run via Make
make test
```

---

## Optional: Ghost Mode (Tor / Proxy Hardening)

If you need to route OSINT traffic through a local proxy (Tor, SOCKS5, or VPN):

### 1. Install Tor

**Windows:**
- Download [Tor Browser](https://www.torproject.org/) (creates proxy at `127.0.0.1:9150`)

**Linux (Ubuntu/Debian):**
```bash
sudo apt install tor
# Default SOCKS5 proxy runs at 127.0.0.1:9050
```

**macOS:**
```bash
brew install tor
brew services start tor
# Default SOCKS5 proxy runs at 127.0.0.1:9050
```

### 2. Configure SPECTRE to Use the Proxy

In `configs/default.yaml`:
```yaml
http:
  proxy: "socks5://127.0.0.1:9050"
  insecure_skip_verify: false
```

### 3. Run with Security Enforcement

```bash
spectre collect all target.com --strict
```

The `--strict` flag ensures that if the proxy goes down, all reconnaissance probes fail-closed immediately to prevent inadvertent data leakage.

---

## Troubleshooting Installation

**"Python not found"**
- Ensure Python 3.10+ is installed: `python --version`
- On Linux/macOS, verify `python3` is symlinked or present in `PATH`.

**"Go build failed"**
- Verify your Go version: `go version` (1.21+ required).
- Ensure your working directory is the root of the cloned repository.

**"Port 8080 is already in use"**
- Another application is using the visualization server port.
- Change the port in `configs/default.yaml` under `server.port: 8081`.

**Still stuck?** See [TROUBLESHOOTING.md](TROUBLESHOOTING.md).
