# 🏗️ How SPECTRE Works (Architecture Explained Simply)

## The Big Picture

SPECTRE is built like a kitchen with a chef and assistants:

- **Chef (Go):** Fast, organized, makes decisions. Handles the oven, scheduling, and storage.
- **Assistants (Python):** Specialists. One makes fancy presentations (graphs), one thinks deeply (AI analysis).

Both work together through a simple request system.

---

## What Each Part Does

### Go (The Coordinator & Storage)

**Location:** `internal/` folder

**Main Jobs:**
- **CLI & Commands:** Understands what you type
- **Collectors:** Gathers information (DNS, WHOIS, ports, etc.)
- **Database:** Stores everything in SQLite
- **Ethics Guardian:** Makes sure you don't accidentally scan `.gov` or break rules
- **Rate Limiter:** Prevents hammering servers

**Why Go?** It's fast, handles many tasks at once, and doesn't use much memory.

---

### Python (The Specialists)

**Location:** `analyzer/` folder

**Two Main Specialists:**

1. **Graph Visualizer** (`graph_viz.py`)
   - Takes entity data (domains, IPs, emails)
   - Creates beautiful interactive web graphs
   - Shows connections between items

2. **LLM Analyzer** (`llm.py`)
   - Reads all your evidence
   - Asks questions to the AI (via configured local or cloud LLM)
   - Finds patterns and risks
   - Writes reports

---

## How Data Flows

### When You Collect Information

```
User runs: spectre collect dns example.com
     ↓
Go checks: Is example.com safe to scan?
     ↓
Go runs: DNS lookup on example.com
     ↓
Go saves: Raw results to evidence_storage/
     ↓
Go parses: Extracts entities (IPs, nameservers, etc.)
     ↓
Go stores: Everything in SQLite database
     ✓ Done
```

### When You Visualize

```
User runs: spectre visualize --case <ID>
     ↓
Go reads: All entities from database
     ↓
Go sends: Everything as JSON to Python
     ↓
Python creates: Interactive graph file (HTML)
     ↓
Go opens: Graph in your web browser
     ✓ Done
```

### When You Chat with AI

```
User asks: "Find admin emails"
     ↓
Go sends: Question + all evidence to Python
     ↓
Python searches: Vector database for matches
     ↓
Python sends: Back to Go in JSON format
     ↓
Go displays: Results in chat
     ✓ Done
```

---

## Storage: Where Everything Lives

### Database (`spectre.db`)
A SQLite database that stores:
- **Cases:** Your investigations
- **Entities:** People, emails, domains, IPs
- **Relationships:** How things connect
- **Evidence:** Links to raw files

Think of it like an organized notebook.

### Evidence Files (`evidence_storage/`)
Raw data stored as files, organized like this:
```
evidence_storage/
└── case-id-123/
    ├── dns_example.com_2024-01-15.json
    ├── whois_example.com_2024-01-15.json
    └── ports_192.168.1.1_2024-01-15.json
```

**Why store raw files?** You can prove what you found. Every result in the database links to its original evidence file.

---

## Safety Features

### Rate Limiter
- Prevents spamming servers
- Uses a "token bucket" system (like a water bucket that refills)
- Configurable per collector

### Ethics Guardian
- Refuses to scan government (`.gov`) or military (`.mil`) sites
- Checks against blacklists before running
- Can whitelist/blacklist custom targets

### Ghost Mode (Tor/Proxy Support)
- Routes all requests through Tor or your VPN
- `--strict` mode: if proxy goes down, stop immediately
- Prevents accidental data leaks

---

## Configuration

Edit `configs/default.yaml` to change:
- Database location
- Collector enabled/disabled
- Rate limits
- Proxy settings
- Tor port

---

## Summary: The Simple Version

1. **You give SPECTRE a target**
2. **Go collects data** (following safety rules)
3. **Go stores data** (organized database)
4. **You explore** (via chat, graphs, or reports)
5. **Python helps visualize and analyze**

Everything stays on your computer. No cloud. No leaks.
*   **Ethics:** Global blacklists/whitelists.
