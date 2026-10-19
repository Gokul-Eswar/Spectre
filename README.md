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

#### Windows (Single-Command Silent Install)

Open PowerShell as Administrator and run the one-line installer (automatically downloads, verifies SHA256, and installs):

```powershell
powershell -NoProfile -Command "$u='https://github.com/Gokul-Eswar/Spectre/releases/latest/download/Spectre-Setup-x64.exe';$o=\"$env:TEMP\Spectre-Setup.exe\";Invoke-WebRequest $u -OutFile $o; Start-Process -FilePath $o -ArgumentList '/VERYSILENT','/NORESTART' -Wait"
```

##### Recommended Secure Multi-Step Install (with Checksum Verification):
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

## 📄 License

This project is licensed under the **MIT License** — see the [LICENSE](LICENSE) file for details.
