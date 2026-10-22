# 🚀 Getting Started: Your First Investigation

Welcome! This guide walks you through a real investigation in SPECTRE — from creating a case to generating reports. It should take about 10 minutes.

---

## The Investigation Process (Simple Version)

Think of an investigation as a journey with 5 steps:

1. **Create a Case** — Give your investigation a name
2. **Collect Evidence** — SPECTRE gathers data about your target
3. **Explore the Graph** — See how all the pieces connect
4. **Chat with the AI** — Ask questions about what you found
5. **Generate a Report** — Export your findings

---

## ⚡ Option A: Interactive Console Workflow (Fastest & Recommended)

Launch the interactive console:
```bash
spectre console
# or simply:
spectre
```

Once inside the console, the header displays `✻ SPECTRE` and the active model (`Model: <name>` or `Model: none`). You can run the entire investigation by typing `?` commands and natural language into the chatbar:

1. **Create and select a case:**
   ```text
   ?case create "suspect-infra-01" "Investigation into phishing infrastructure"
   ```
2. **Run reconnaissance collectors:**
   ```text
   ?collect target.com dns
   ```
3. **Inspect collected evidence:**
   ```text
   ?evidence
   ```
4. **View connections & graph:**
   ```text
   ?graph          # Displays clean ASCII network in your terminal
   ?web            # Opens interactive D3 web graph in browser (localhost:8080)
   ```
5. **Ask questions & synthesize findings with AI:**
   Just type your question directly (no prefix needed):
   ```text
   What domains or mail servers are connected to this target, and what are the main security risks?
   ```
   Or run the full automated analysis pipeline:
   ```text
   ?analyze
   ```
6. **Generate reports:**
   ```text
   ?report md      # Generates Markdown report
   ?report pdf     # Generates executive branded PDF
   ```

---

## 💻 Option B: Direct CLI Commands Workflow (Scripting & Automation)

### Step 1: Create a New Case

A **Case** is like a folder for your investigation. Everything you collect gets organized inside it.

**Create a case:**
```bash
spectre case new "suspect-infra-01"
```

**You'll see:**
```
✓ Successfully created case: suspect-infra-01
  ID: a8f7c9e0-1234-abcd-9876-1a2b3c4d5e6f
```

💡 **Copy and save that ID** — You'll use it for every command going forward.

**Forgot the ID?** List all your cases:
```bash
spectre case list
```

---

## Step 2: Gather Basic Information (Passive Collection)

Let's say you want to investigate `target.com`. SPECTRE can automatically gather basic information without ever directly contacting the website (this is called "passive" collection).

**Collect passive data:**
```bash
spectre collect all target.com --case a8f7c9e0-1234-abcd-9876-1a2b3c4d5e6f
```

**What SPECTRE will find:**
- DNS records (where the domain points)
- WHOIS information (who owns it)
- IP geolocation (where it's hosted)
- Any public GitHub repositories

**Where it's stored:**
- Database: `spectre.db` (the knowledge graph)
- Raw files: `evidence_storage/a8f7c9e0-1234-abcd-9876-1a2b3c4d5e6f/` (proof of what was found)

---

## Step 3: Scan the Website (Active Collection) — *Optional*

Want to find open ports or take screenshots? This requires an extra permission flag (to prevent accidents):

**Scan for open ports:**
```bash
spectre collect ports target.com --case a8f7c9e0-1234-abcd-9876-1a2b3c4d5e6f --active
```

⚠️ **What this does:** Tries to connect to common ports (like 80, 443, 22, etc.) on the target. Only common ports are scanned unless you configure custom ones.

---

## Step 4: Explore Using the AI Chat

Now you have evidence. Ask the AI questions about it.

**Start a chat session:**
```bash
spectre chat --case a8f7c9e0-1234-abcd-9876-1a2b3c4d5e6f
```

**Inside the chat, you can:**

- **Search evidence:** `"Find all email addresses in the data"`
- **Run collectors:** `"Run DNS lookup on example.com"`
- **Get analysis:** `"What risks do you see?"`
- **Get help:** Type `/help`
- **Exit:** Type `exit` or `quit`

**Example conversation:**
```
You: Find all email addresses
AI: I found 3 emails: admin@target.com, support@target.com, ...

You: Run whois on those domains
AI: [runs whois] Here's what I found...
```

---

## Step 5: See How Everything Connects (Graph Visualization)

SPECTRE finds relationships between all the pieces (domains, IPs, emails, etc.) and shows them as a visual graph.

**Start the visualization server:**
```bash
spectre server
```

**In another terminal, generate the graph:**
```bash
spectre visualize --case a8f7c9e0-1234-abcd-9876-1a2b3c4d5e6f
```

**What happens:**
- A web browser opens automatically
- You see all entities (domains, IPs, emails) as nodes
- Lines show how they connect
- You can drag, zoom, and click to explore

---

## Step 6: Export Your Findings

Ready to share your investigation? Export it in multiple formats.

**Generate a text summary:**
```bash
spectre report markdown --case a8f7c9e0-1234-abcd-9876-1a2b3c4d5e6f
```
Outputs: `case_report.md`

**Generate a professional PDF:**
```bash
spectre report pdf --case a8f7c9e0-1234-abcd-9876-1a2b3c4d5e6f
```
Outputs: `case_report.pdf` (includes timeline, entities, risk scores)

---

## 🎯 Common Tasks Quick Reference

**List your cases:**
```bash
spectre case list
```

**Look inside a case:**
```bash
spectre case show a8f7c9e0-1234-abcd-9876-1a2b3c4d5e6f
```

**Delete a case:**
```bash
spectre case delete a8f7c9e0-1234-abcd-9876-1a2b3c4d5e6f
```

**Investigate a username across multiple platforms:**
```bash
spectre collect accounts username123 --case <YOUR_CASE_ID>
```

---

## ✅ Next Steps

- **Troubleshooting?** See [TROUBLESHOOTING.md](TROUBLESHOOTING.md)
- **Want to secure your investigation?** See the "Operational Security" section in [INSTALLATION.md](INSTALLATION.md)
- **Building custom collectors?** See [PLUGIN_DEVELOPMENT.md](PLUGIN_DEVELOPMENT.md)
