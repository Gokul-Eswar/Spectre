# 💾 How to Install SPECTRE

This guide walks you through installing SPECTRE step-by-step on Windows, Linux, or macOS.

---

## What You'll Need

Before starting, make sure you have these installed:

| What | Minimum Version | Check with |
|------|---|---|
| **Go** | 1.22+ | `go version` |
| **Python** | 3.10+ | `python --version` |
| **Git** | 2.x+ | `git --version` |

- **Local LLM:** Ollama ([ollama.ai](https://ollama.ai)) or LM Studio ([lmstudio.ai](https://lmstudio.ai)) for private offline AI.
- **Cloud LLM:** API key for OpenAI, Anthropic Claude, Google Gemini, Groq, or OpenRouter.

---

## Windows Users (Easiest Way: Single-Command Installer)

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


## Linux & macOS (Manual Setup)

**Step 1:** Clone and enter the folder:
```bash
git clone https://github.com/Gokul-Eswar/Spectre.git
cd Spectre
```

**Step 2:** Build the program:
```bash
go build -o spectre cmd/spectre/main.go
chmod +x spectre
```

**Step 3:** Set up Python:
```bash
python3 -m venv .venv
source .venv/bin/activate
pip install --upgrade pip
pip install -r analyzer/requirements.txt
```

---

## Check If Everything Works

Run this to verify your installation:

```bash
# Windows
.\spectre.exe version

# Linux/macOS
./spectre version
```

You should see: `SPECTRE v1.0.0` ✓

---

## Run the Tests

Want to make sure everything is working correctly?

```bash
# Test the Go part
go test ./...

# Test the Python part (Linux/macOS)
.venv/bin/python -m pytest analyzer/

# Test the Python part (Windows)
.venv\Scripts\python -m pytest analyzer/
```

---

## Optional: Private & Secure Mode (Ghost Mode)

If you want to hide your investigations through a proxy (Tor, VPN, etc.), follow these steps:

### 1. Install Tor

**Windows:**
- Download [Tor Browser](https://www.torproject.org/)
- It creates a proxy at `127.0.0.1:9150`

**Linux (Ubuntu/Debian):**
```bash
sudo apt install tor
# Runs at 127.0.0.1:9050
```

**macOS:**
```bash
brew install tor
brew services start tor
# Runs at 127.0.0.1:9050
```

### 2. Configure SPECTRE to Use the Proxy

Edit `configs/default.yaml`:
```yaml
http:
  proxy: "socks5://127.0.0.1:9050"
  insecure_skip_verify: false
```

### 3. Run with Security Mode On

```bash
spectre collect all target.com --strict
```

The `--strict` flag means: if the proxy goes down, stop immediately instead of continuing without it. This protects you from accidentally leaking data.

---

## Troubleshooting Installation

**"Python not found"**
- Make sure Python 3.10+ is installed: `python --version`
- On some systems, use `python3` instead of `python`

**"Go build failed"**
- Check your Go version: `go version` (need 1.22+)
- Make sure you're in the SPECTRE folder

**"Port 8080 is already in use"**
- Something else is using that port
- Change it in `configs/default.yaml` under the `server:` section

**Still stuck?** See [TROUBLESHOOTING.md](TROUBLESHOOTING.md)
