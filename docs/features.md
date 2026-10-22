# SPECTRE Feature Guide

## 🕵️ Collectors

Spectre includes a suite of powerful collectors to gather intelligence across different vectors.

### Passive Collectors (Low Risk)
These collectors interact with public registries and APIs. They rarely touch the target infrastructure directly.

- **DNS (`dns`):**
  - Resolves A, AAAA, MX, NS, TXT, and CNAME records.
  - Useful for mapping infrastructure and finding mail servers.

- **Whois (`whois`):**
  - Queries registrar databases for domain ownership info.
  - *Note:* Often redacted for privacy, but historical data can be useful.

- **GeoIP (`geo`):**
  - Maps resolved IP addresses to physical locations (City, Country, ISP).
  - Helps in attribution and identifying hosting providers.

- **GitHub (`github`):**
  - Scans public repositories for occurrences of the target domain or keywords.
  - Good for finding leaked credentials or source code references.

### Active Collectors (Moderate Risk)
These collectors send traffic directly to the target. Use with caution and authorization.

- **Port Scanner (`ports`):**
  - Scans TCP ports to identify running services.
  - **Modes:**
    - `default`: Scans ~20 common ports (HTTP, SSH, FTP, etc.).
    - `top-100`: Scans the most frequent 100 ports from Nmap services.
    - `custom`: Scans a specific list defined in config.

- **Screenshot (`screenshot`):**
  - Uses a headless browser (`chromedp`) to render the target URL and capture a PNG.
  - **Features:**
    - Full-page capture.
    - Respects proxy settings (Ghost Mode).

- **Social Media (`social`):**
  - Checks username availability across 50+ platforms (GitHub, Twitter, Reddit, etc.).
  - Useful for "Persona Investigation" when the target is a handle (e.g., `hacker_one`).

---

## 🌐 Web Dashboard

The Web Dashboard offers an advanced, interactive view of your investigation.

### Features
- **Force-Directed Graph:** Physics-based rendering where nodes repel/attract based on relationships.
- **Interactive:** Drag nodes to rearrange the view. Scroll to zoom.
- **Details Panel:** Hover over any node to see metadata (Source, Time Discovered).
- **Live Updates:** The graph refreshes automatically as new evidence is collected in the background.

### How to Use
- **Inside the TUI Console:** Simply type `?web` in the chatbar. SPECTRE automatically launches `http://localhost:8080` in your default browser.
- **From CLI:** Run `spectre server` and open `http://localhost:8080`.
- **In-Terminal ASCII Graph:** If you prefer staying in the terminal without opening a browser, type `?graph` in the chatbar.

---

## 📄 Reporting

### Markdown Report
- **Best for:** Technical archiving, sharing with other analysts, or importing into wikis.
- **Content:** Raw structured data, lists of assets, and full analysis text.
- **How to Generate:**
  - **In Console:** Type `?report md` (or simply `?report`).
  - **From CLI:** Run `spectre report markdown --case <case_id>`.

### PDF Report
- **Best for:** Executive summaries, client deliverables, and formal documentation.
- **Content:**
  - **Cover Page:** Case ID, Date, Status.
  - **Executive Summary:** AI-synthesized Findings and Risks.
  - **Entity Table:** Cleanly formatted list of discovered assets.
  - **Timeline:** Chronological log of investigation events.
- **How to Generate:**
  - **In Console:** Type `?report pdf`.
  - **From CLI:** Run `spectre report pdf --case <case_id>`.
