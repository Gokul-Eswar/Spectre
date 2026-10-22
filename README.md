# 🕵️ SPECTRE — Local-First OSINT Intelligence Platform

> **Turn raw internet noise into structured, verifiable intelligence — fast, repeatable, and 100% local.**

"SPECTRE is like having a smart detective assistant that gathers all online clues about a target in one place, figures out connections automatically, and keeps everything private and locked down—all without needing the internet or cloud services."

---

## 🎯 The Problem SPECTRE Solves

Imagine you're investigating a suspicious website or tracking down who's behind an online account. Right now, you'd have to use 10+ different tools—check who owns the domain, look up the server location, scan for open ports, take screenshots, search social media manually. Then you'd copy-paste everything into a spreadsheet and try to connect the dots yourself. It's slow, messy, and your data gets scattered across cloud services you don't fully trust.

SPECTRE brings everything together in one place. Tell it a website or username, and it automatically gathers all the information you need—who owns it, where the servers are, what services are running, and what else it's connected to. Then it uses AI to connect the dots for you and show you what matters. Everything stays on your computer (no cloud), and every finding is locked down so it can be used as evidence in court or investigations. What used to take hours now takes minutes.

---

## 🚀 Setup & Installation

### 1. Prerequisites
- **Go:** `1.21+` ([Download Go](https://go.dev/dl/))
- **Python:** `3.10+` with `pip` ([Download Python](https://www.python.org/downloads/))
- **AI / LLM (Optional):** Local (Ollama, LM Studio) or Cloud (OpenAI, Anthropic Claude, Google Gemini, Groq, OpenRouter).

---

### 2. Install from Repository (Developer / Source)

Clone the repository and build the binary directly:

#### Windows (PowerShell):
```powershell
# 1. Clone repository
git clone https://github.com/Gokul-Eswar/Spectre.git
cd Spectre

# 2. Build Go binary
go build -o spectre.exe cmd/spectre/main.go

# 3. Create Python virtual environment and install analyzer dependencies
python -m venv .venv
.\.venv\Scripts\Activate.ps1
pip install --upgrade pip
pip install -r analyzer/requirements.txt
```

#### Linux & macOS (Bash):
```bash
# 1. Clone repository
git clone https://github.com/Gokul-Eswar/Spectre.git
cd Spectre

# 2. Build Go binary
go build -o spectre cmd/spectre/main.go
chmod +x spectre

# 3. Create Python virtual environment and install analyzer dependencies
python3 -m venv .venv
source .venv/bin/activate
pip install --upgrade pip
pip install -r analyzer/requirements.txt
```

---

### 3. Windows Pre-built Installer (Single-Command Silent Install)

Open PowerShell as Administrator and run the one-line installer:

```powershell
powershell -NoProfile -Command "$u='https://github.com/Gokul-Eswar/Spectre/releases/latest/download/Spectre-Setup-x64.exe';$o=\"$env:TEMP\Spectre-Setup.exe\";Invoke-WebRequest $u -OutFile $o; Start-Process -FilePath $o -ArgumentList '/VERYSILENT','/NORESTART' -Wait"
```

#### Recommended Secure Install (with Checksum Verification):
Replace `<SHA256_HASH>` with the checksum listed on the [Releases page](https://github.com/Gokul-Eswar/Spectre/releases):

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

*Installed to `C:\Program Files\SPECTRE` and automatically added to system `PATH`.*

---

## 🖥️ Modern Horizontal TUI Console

Launch the unified interactive console by simply typing:

```bash
spectre
# or
spectre console
```

### Clean, Clutter-Free Horizontal Interface
- **Minimal Header**: Displays strictly the application name (**`✻ SPECTRE`**) on the left and the active model on the right (**`Model: <name>`**).
- **Dynamic Model Identification**: If no model or API credentials are configured, the header dynamically updates to **`Model: none`**.
- **Universal Chat & Command Center**: The bottom prompt (`❯ `) accepts both freeform natural language queries for the autonomous AI agent and `?` commands for all app features.

### Quick Reference: `?` Commands

Type any of the following into the chatbar at any time:

| Command | Description |
|---|---|
| `?` or `?help` | Display the interactive command directory |
| `?cases` | List all investigation cases with ID, status, and creation date |
| `?case <id>` | Switch active investigation case context |
| `?case create <name> [desc]` | Create a new case and switch to it |
| `?case info` | Display active case metadata and dossier details |
| `?model [name]` | View active model or dynamically switch (e.g. `?model gpt-4o`, `?model none`) |
| `?analyze` | Run automated AI intelligence synthesis pipeline with live step feedback |
| `?evidence` | View extracted entities, IPs, domains, and indicators table |
| `?graph` | Display ASCII entity relationship network graph |
| `?timeline` | Display chronological event sequence of evidence |
| `?report [md\|pdf]` | Generate structured Markdown dossier or executive branded PDF |
| `?collect <target> [module]` | Run reconnaissance probes (DNS, Whois, Ports, etc.) |
| `?collectors` | List all available reconnaissance collectors |
| `?settings` | Display active configuration parameters |
| `?set <key> <value>` | Live-update configuration settings (e.g. `?set ghost_mode true`) |
| `?test` | Test network latency and authorization to the active LLM |
| `?scan` | Probe and discover locally installed / remote LLM models |
| `?web` | Open the interactive D3/vis.js network graph dashboard in your browser |
| `?clear` | Clear the viewport text buffer |
| `?quit` | Cleanly exit the console |

*Note: Entering any prompt without `?` sends your query directly to the Autonomous Investigation Agent.*

---

## 📚 Documentation

Detailed documentation is available in the [`docs/`](docs/) directory:
- [Getting Started Guide](docs/GETTING_STARTED.md)
- [Installation Guide](docs/INSTALLATION.md)
- [Multi-Provider LLM Guide](docs/llm.md)
- [Feature Guide](docs/features.md)
- [Architecture](docs/ARCHITECTURE.md)
- [API Documentation](docs/API_DOCUMENTATION.md)
- [Troubleshooting](docs/TROUBLESHOOTING.md)

---

## 📄 License

This project is licensed under the **MIT License** — see the [LICENSE](LICENSE) file for details.
