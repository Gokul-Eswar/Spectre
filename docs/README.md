# 📚 SPECTRE Documentation

Welcome! This page helps you find the right guide for what you want to do.

---

## 🚀 Getting Started

**New to SPECTRE?** Start here:

- **[Getting Started Guide](GETTING_STARTED.md)** ← Begin here! A walk-through of your first investigation in 10 minutes.
- **[Installation Guide](INSTALLATION.md)** ← Complete guide for Windows, Linux, macOS, and Tor setup.

---

## ⚡ Quick Installation

### Windows (Single-Command Silent Install)
Open PowerShell as Administrator and run:
```powershell
powershell -NoProfile -Command "$u='https://github.com/Gokul-Eswar/Spectre/releases/latest/download/Spectre-Setup-x64.exe';$o=\"$env:TEMP\Spectre-Setup.exe\";Invoke-WebRequest $u -OutFile $o; Start-Process -FilePath $o -ArgumentList '/VERYSILENT','/NORESTART' -Wait"
```

#### Recommended Secure Install (with SHA256 Verification):
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

*For Linux, macOS, or building from source, see the full **[Installation Guide](INSTALLATION.md)**.*

---

## 🤔 Understanding SPECTRE

Want to understand how it works?

- **[How SPECTRE Works (Architecture)](ARCHITECTURE.md)** ← Explained simply, what each part does and how data flows.
- **[Feature List](features.md)** ← Complete list of what SPECTRE can do.

---

## 🛠️ Building & Extending

Want to add your own tools or integrate SPECTRE with other systems?

- **[Creating Custom Collectors (Plugins)](PLUGIN_DEVELOPMENT.md)** ← Build your own data collectors in Python, Bash, or Go.
- **[How Plugins Work](plugins.md)** ← How SPECTRE discovers and runs custom collectors safely.
- **[API Documentation](API_DOCUMENTATION.md)** ← Technical details of how Go and Python talk to each other.

---

## 🔒 Advanced Topics

For experienced users and enterprise deployments:

- **[Deployment Guide](DEPLOYMENT.md)** ← Running SPECTRE on servers, multi-user setups, and headless systems.
- **[LLM Integration](llm.md)** ← Setting up local or cloud AI models for analysis.
- **[Performance Tuning](performance.md)** ← Making SPECTRE faster and more efficient.

---

## ❓ Stuck?

- **[Troubleshooting](TROUBLESHOOTING.md)** ← Common problems and how to fix them.

---

## 📋 Quick Reference

| I want to... | See this guide |
|---|---|
| Set up SPECTRE | [Installation Guide](INSTALLATION.md) |
| Run my first investigation | [Getting Started](GETTING_STARTED.md) |
| Understand how it works | [Architecture](ARCHITECTURE.md) |
| See what it can do | [Features](features.md) |
| Build a custom collector | [Plugin Development](PLUGIN_DEVELOPMENT.md) |
| Fix a problem | [Troubleshooting](TROUBLESHOOTING.md) |
| Deploy to a server | [Deployment Guide](DEPLOYMENT.md) |
| Use AI analysis | [LLM Integration](llm.md) |
| Speed things up | [Performance](performance.md)
