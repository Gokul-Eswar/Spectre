# 🕵️ SPECTRE — Local-First OSINT Intelligence Platform

> **Turn raw internet noise into structured, verifiable intelligence — fast, repeatable, and 100% local.**

"SPECTRE is like having a smart detective assistant that gathers all online clues about a target in one place, figures out connections automatically, and keeps everything private and locked down—all without needing the internet or cloud services."


## 🎯 The Problem SPECTRE Solves

Imagine you're investigating a suspicious website or tracking down who's behind an online account. Right now, you'd have to use 10+ different tools—check who owns the domain, look up the server location, scan for open ports, take screenshots, search social media manually. Then you'd copy-paste everything into a spreadsheet and try to connect the dots yourself. It's slow, messy, and your data gets scattered across cloud services you don't fully trust.

SPECTRE brings everything together in one place. Tell it a website or username, and it automatically gathers all the information you need—who owns it, where the servers are, what services are running, and what else it's connected to. Then it uses AI to connect the dots for you and show you what matters. Everything stays on your computer (no cloud), and every finding is locked down so it can be used as evidence in court or investigations. What used to take hours now takes minutes.

## 🚀 Setup

### 1. Prerequisites
- **Go:** `1.21+` ([Download Go](https://go.dev/dl/))
- **Python:** `3.10+` with `pip` ([Download Python](https://www.python.org/downloads/))
- **AI / LLM (Optional):** Local (Ollama, LM Studio) or Cloud (OpenAI, Anthropic Claude, Google Gemini, Groq, OpenRouter).

---

### 2. Installation

#### Windows (One-Click Setup)
Run the automated installation script in PowerShell:
```powershell
.\install.ps1
```
To enable global command line access from any terminal:
```powershell
.\setup_global.ps1
```

#### Linux & macOS / Manual Setup
```bash
# 1. Clone repository
git clone https://github.com/Gokul-Eswar/Spectre.git
cd Spectre

# 2. Build the Go binary
go build -o spectre cmd/spectre/main.go

# 3. Setup Python virtual environment & dependencies
python3 -m venv .venv
source .venv/bin/activate
pip install -r analyzer/requirements.txt
```

## 📄 License

This project is licensed under the **MIT License** — see the [LICENSE](LICENSE) file for details.
