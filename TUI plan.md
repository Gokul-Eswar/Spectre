# SPECTRE Modern Horizontal TUI & Unified Command Architecture Plan

## 1. Executive Summary & Design Transformation

This plan outlines the end-to-end redesign and modernization of the SPECTRE Terminal User Interface (TUI). 

### Current Pain Points (Legacy TUI)
- **Cluttered Vertical Layout**: A 22-character vertical left sidebar permanently consumed horizontal screen space, making tabular data, ASCII graphs, and LLM code output feel cramped and difficult to read.
- **Visual Distraction & Noise**: The legacy header contained multiple redundant badges (Ghost Mode, case ID, protocol pills, shortcuts lists, hotkey reminders) which violated modern minimalist CLI aesthetics.
- **Fragmented Interaction Model**: Users had to switch mental modes between number keys (`1`-`9`, `0`), arrow/vim keys (`j`/`k`), and chat input. App features were segregated into rigid panels instead of being accessible directly from the prompt.
- **Static Model Assumption**: Model status was not dynamically determined—if unconfigured or invalid, it displayed outdated hardcoded defaults rather than accurately identifying unconfigured states (`none`).

### The Modernized Solution
- **100% Horizontal Orientation**: The vertical sidebar is eliminated. The UI adopts a full-width, clean top-to-bottom layout inspired by modern developer CLI tools (Claude Code, GitHub Copilot CLI, Gemini CLI).
- **Minimalist Header**: Stripped of all unnecessary noise. The top bar displays strictly:
  1. **Application Name**: `SPECTRE`
  2. **Active Model**: Dynamically identified LLM model name (e.g. `llama3`, `gpt-4o`, `claude-3-5-sonnet`) or dynamically falls back to `none` if nothing is configured.
- **Unified `?` Command Bar**: The chat input bar at the bottom acts as the universal command center. All app capabilities (Cases, Analysis, Evidence, Graph, Timeline, Reports, Collectors, Settings, Web Dashboard) are converted into `?` commands (e.g., `?cases`, `?analyze`, `?graph`, `?report`, `?collect`, `?help`). Freeform text is automatically routed to the autonomous AI agent.
- **Responsive & Consistent Backend Coupling**: Full-width viewport resizing with `tea.WindowSizeMsg`, animated background task execution indicators, clear feedback banners, and asynchronous streaming.

---

## 2. Horizontal Architecture & Visual Wireframe

### Layout Wireframe (Full Terminal Width)

```text
┌──────────────────────────────────────────────────────────────────────────────────────────────┐
│  ✻ SPECTRE                                                            Model: claude-3-7-sonnet│
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                              │
│  ✻ Welcome to SPECTRE Intelligence Console                                                   │
│  Type natural language queries to investigate, or type '?' for the command directory.       │
│                                                                                              │
│  ❯ ?cases                                                                                    │
│                                                                                              │
│  ID            NAME                 TARGET             STATUS       ENTITIES   CREATED       │
│  ─────────────────────────────────────────────────────────────────────────────────────────   │
│  case_8f1a2    Operation Blackout   target-corp.com    ACTIVE       42         2026-10-20    │
│  case_4c9d1    Shadow Audit         198.51.100.24      ARCHIVED     18         2026-10-18    │
│                                                                                              │
│  ❯ ?analyze                                                                                  │
│  ✻ Running analysis pipeline on case_8f1a2...                                                │
│  ✔ Ingested 42 entities and 19 relationships                                                 │
│  ✔ Synthesizing risk scoring & attack surface vectors                                        │
│                                                                                              │
│  [Intelligence Findings]                                                                     │
│  - Critical: Exposed Jenkins server on port 8080 with anonymous read access                  │
│  - Medium: WHOIS registrant email linked to 3 historical credential leaks                   │
│                                                                                              │
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│  [?]: ?cases  ?case <id>  ?analyze  ?evidence  ?graph  ?timeline  ?report  ?collect  ?help   │
│  ❯ Ask Spectre or type '?' for commands...                                                  │
└──────────────────────────────────────────────────────────────────────────────────────────────┘
```

### Layout Elements
1. **Top Bar (`renderHeader`)**:
   - Left: `✻ SPECTRE` (Bold terracotta accent)
   - Right: `Model: <name>` (Emerald green if configured, Amber/Dim `Model: none` if unconfigured)
   - Height: 1 line + 1 line separator border.
2. **Main Viewport (`renderViewport`)**:
   - Uses `viewport.Model` from `bubbles/viewport`.
   - Occupies `Width = terminal_width - 2`, `Height = terminal_height - 5`.
   - Displays all conversational history, analytical tables, ASCII network graphs, timelines, and command outputs with smooth scroll support (`PgUp`/`PgDn` or mouse wheel).
3. **Command Suggestion Strip (`renderSuggestions`)**:
   - Dynamically appears or highlights available `?` commands whenever the user types `?` in the input bar.
4. **Chatbar / Command Center (`renderInput`)**:
   - `textinput.Model` with `❯ ` prompt.
   - Immediate dispatch on `Enter`.

---

## 3. Dynamic Model Resolution Engine

### Logic Specification
The application must dynamically inspect the LLM configuration in Viper, environment variables, and active connection parameters:
1. Read `viper.GetString("llm.model")`.
2. Inspect `viper.GetString("llm.provider")`.
3. If provider requires an API Key (e.g., `openai`, `anthropic`, `gemini`, `groq`, `openrouter`) and neither `llm.api_key` nor `keys.<provider>` is set: the model is unconfigured (`none`).
4. If `model` is empty, `"none"`, or whitespace: resolves to `"none"`.
5. If valid: returns clean model identifier.
6. Reactive synchronization: Updating the model via `?model <name>` or `?set model <name>` immediately updates the top bar display.

```go
// Dynamic model resolver rule:
// 1. If explicit "none" or empty string -> "none"
// 2. If provider requires API key but key is missing -> "none"
// 3. If valid model string -> return sanitized model name
```

---

## 4. Complete `?` Command Registry & Mapping

Every application feature is mapped to a dedicated `?` command:

| Command | Arguments | Description | Backend Connection |
|---|---|---|---|
| `?` / `?help` | None | Displays full interactive command directory and usage | In-memory Help Catalog |
| `?cases` | None | Lists all investigation cases with ID, target, status | `storage.ListCases()` |
| `?case` | `<id>` | Switches active investigation case context | `m.selectedCaseID`, `chat.setCase()` |
| `?case create` | `<name> [desc]` | Creates a new case and switches context to it | `storage.CreateCase()` |
| `?case info` | None | Shows active case metadata and dossier summary | `storage.GetCase(m.selectedCaseID)` |
| `?model` | `[name]` | View active model or switch to a new model (or `none`) | `viper.Set("llm.model", ...)` |
| `?analyze` | None | Executes automated AI synthesis on active case | `analysis.StartAnalysis()` |
| `?evidence` | None | Renders extracted entities, credentials, and indicators | `storage.ListEntities()`, `table.Model` |
| `?graph` | None | Renders full-width ASCII entity relationship network | `tui.RenderASCIIGraph()` |
| `?timeline` | None | Renders chronological event sequence of evidence | `tui.RenderTimeline()` |
| `?report` | `[md \| pdf]` | Generates structured Markdown or branded PDF report | `report.GenerateMarkdownReport()`, `report.GeneratePDFReport()` |
| `?collect` | `<target> [coll]` | Runs passive/active reconnaissance collectors | `collector.RunAndSave()` |
| `?collectors` | None | Lists all available reconnaissance collectors | `collector.List()` |
| `?settings` | None | Displays current configuration parameters | Viper configuration reader |
| `?set` | `<key> <val>` | Live updates configuration parameter | `viper.Set()`, `config.SaveConfig()` |
| `?test` | None | Tests network latency & authorization to LLM endpoint | `analysis.TestLLMConnection()` |
| `?scan` | None | Scans and detects local/remote models (Ollama, LM Studio) | `analysis.FetchAvailableModels()` |
| `?web` | None | Launches local browser web dashboard (localhost:8080) | `browser.OpenURL()` |
| `?clear` | None | Clears the viewport output buffer | `m.viewport.SetContent("")` |
| `?quit` / `?exit` | None | Exits the TUI application | `tea.Quit` |
| *`<any text>`* | *`<prompt>`* | Sent to Autonomous Agent for investigation chat | `agent.Engine.Execute()` |

---

## 5. Detailed Component Implementation with Full Code Snippets

### 5.1 Minimalist Theme & Horizontal Styles (`internal/tui/theme.go`)

```go
package tui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Modern Zinc & Terracotta Palette
	ColorTerracotta  = lipgloss.Color("#D97757")
	ColorPeach       = lipgloss.Color("#E8987E")
	ColorDarkBg      = lipgloss.Color("#121214") // Sleek deep graphite
	ColorSurface     = lipgloss.Color("#18181B") // Zinc-900
	ColorSurfaceHigh = lipgloss.Color("#27272A") // Zinc-800
	ColorBorder      = lipgloss.Color("#3F3F46") // Zinc-700
	ColorBorderMuted = lipgloss.Color("#27272A") // Zinc-800 separator
	ColorText        = lipgloss.Color("#F4F4F5") // Zinc-100
	ColorTextMuted   = lipgloss.Color("#A1A1AA") // Zinc-400
	ColorTextDim     = lipgloss.Color("#71717A") // Zinc-500
	ColorSuccess     = lipgloss.Color("#10B981") // Emerald-500
	ColorWarning     = lipgloss.Color("#F59E0B") // Amber-500
	ColorError       = lipgloss.Color("#EF4444") // Red-500
	ColorCyan        = lipgloss.Color("#06B6D4") // Cyan-500

	// Header Styles (Clean single-line horizontal)
	StyleAppLogo = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorTerracotta)

	StyleModelPillConfigured = lipgloss.NewStyle().
					Foreground(ColorSuccess).
					Bold(true)

	StyleModelPillNone = lipgloss.NewStyle().
				Foreground(ColorWarning).
				Bold(true)

	// Horizontal Separator
	StyleDivider = lipgloss.NewStyle().
			Foreground(ColorBorderMuted)

	// Viewport & Output Styles
	StyleViewport = lipgloss.NewStyle().
			Padding(0, 1).
			Background(ColorDarkBg)

	// Input Bar Styles
	StylePromptSymbol = lipgloss.NewStyle().
				Foreground(ColorTerracotta).
				Bold(true)

	StyleInputBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Background(ColorSurface).
			Padding(0, 1)

	StyleInputBoxActive = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorTerracotta).
				Background(ColorSurface).
				Padding(0, 1)

	StyleSuggestions = lipgloss.NewStyle().
				Foreground(ColorTextDim).
				Background(ColorDarkBg).
				Padding(0, 1)

	// Command Output Cards
	StyleCard = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Background(ColorSurface).
			Padding(1, 2)
)
```

---

### 5.2 Dynamic Model Resolution Engine (`internal/tui/model_detector.go`)

```go
package tui

import (
	"strings"

	"github.com/spf13/viper"
)

// DynamicModelStatus encapsulates detected model information.
type DynamicModelStatus struct {
	Name         string
	IsConfigured bool
	DisplayTag   string
}

// DetectActiveModel dynamically inspects runtime configuration and determines
// whether a valid model is chosen, or automatically resolves to "none".
func DetectActiveModel() DynamicModelStatus {
	rawModel := strings.TrimSpace(viper.GetString("llm.model"))
	rawProvider := strings.ToLower(strings.TrimSpace(viper.GetString("llm.provider")))
	apiKey := strings.TrimSpace(viper.GetString("llm.api_key"))

	if rawModel == "" || strings.EqualFold(rawModel, "none") {
		return DynamicModelStatus{
			Name:         "none",
			IsConfigured: false,
			DisplayTag:   "Model: none",
		}
	}

	// Providers that require an API key to function
	cloudProviders := map[string]bool{
		"openai":     true,
		"anthropic":  true,
		"gemini":     true,
		"groq":       true,
		"openrouter": true,
	}

	if cloudProviders[rawProvider] {
		// Also check keys.<provider> fallback
		if apiKey == "" {
			apiKey = strings.TrimSpace(viper.GetString("keys." + rawProvider))
		}
		if apiKey == "" {
			return DynamicModelStatus{
				Name:         "none",
				IsConfigured: false,
				DisplayTag:   "Model: none (API key missing)",
			}
		}
	}

	return DynamicModelStatus{
		Name:         rawModel,
		IsConfigured: true,
		DisplayTag:   "Model: " + rawModel,
	}
}

// SetActiveModel updates the active model and synchronizes configuration.
func SetActiveModel(modelName string) DynamicModelStatus {
	clean := strings.TrimSpace(modelName)
	if clean == "" || strings.EqualFold(clean, "none") {
		viper.Set("llm.model", "")
	} else {
		viper.Set("llm.model", clean)
	}
	return DetectActiveModel()
}
```

---

### 5.3 Unified `?` Command Dispatcher (`internal/tui/commands.go`)

```go
package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/Gokul-Eswar/Spectre/internal/analysis"
	"github.com/Gokul-Eswar/Spectre/internal/collector"
	"github.com/Gokul-Eswar/Spectre/internal/config"
	"github.com/Gokul-Eswar/Spectre/internal/report"
	"github.com/Gokul-Eswar/Spectre/internal/storage"
	"github.com/spf13/viper"
)

// CommandResult represents the output of a ? command execution.
type CommandResult struct {
	Output      string
	IsError     bool
	ModelChange bool
	Quit        bool
}

// ExecuteQuestionCommand parses and runs any command starting with '?'
func ExecuteQuestionCommand(input string, currentCaseID string) CommandResult {
	trimmed := strings.TrimSpace(input)
	if !strings.HasPrefix(trimmed, "?") {
		return CommandResult{Output: "Not a ? command", IsError: true}
	}

	content := strings.TrimSpace(strings.TrimPrefix(trimmed, "?"))
	if content == "" || content == "help" {
		return CommandResult{Output: RenderHelpDirectory()}
	}

	parts := strings.Fields(content)
	cmd := strings.ToLower(parts[0])
	args := parts[1:]

	switch cmd {
	case "cases", "case-list":
		return handleCasesList()

	case "case":
		if len(args) == 0 {
			return handleCaseInfo(currentCaseID)
		}
		subCmd := strings.ToLower(args[0])
		if subCmd == "create" {
			if len(args) < 2 {
				return CommandResult{Output: "Usage: ?case create <name> [description]", IsError: true}
			}
			name := args[1]
			desc := ""
			if len(args) > 2 {
				desc = strings.Join(args[2:], " ")
			}
			c, err := storage.CreateCase(name, desc)
			if err != nil {
				return CommandResult{Output: fmt.Sprintf("Error creating case: %v", err), IsError: true}
			}
			return CommandResult{Output: fmt.Sprintf("✔ Case created successfully: %s (ID: %s)", c.Name, c.ID)}
		}
		if subCmd == "info" {
			return handleCaseInfo(currentCaseID)
		}
		// Treat argument as case ID to switch to
		c, err := storage.GetCase(args[0])
		if err != nil || c == nil {
			return CommandResult{Output: fmt.Sprintf("Error: Case '%s' not found.", args[0]), IsError: true}
		}
		return CommandResult{Output: fmt.Sprintf("✔ Switched active case to: %s (%s)", c.Name, c.ID)}

	case "model":
		if len(args) == 0 {
			st := DetectActiveModel()
			presets, _ := analysis.FetchAvailableModels()
			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("Current Model: %s\n\nAvailable detected presets:\n", st.Name))
			for _, p := range presets {
				sb.WriteString(fmt.Sprintf("  • %s\n", p))
			}
			sb.WriteString("\nType '?model <name>' or '?model none' to update.")
			return CommandResult{Output: sb.String()}
		}
		newModel := args[0]
		st := SetActiveModel(newModel)
		_ = config.SaveConfig()
		return CommandResult{
			Output:      fmt.Sprintf("✔ Active model set to: %s", st.Name),
			ModelChange: true,
		}

	case "evidence", "entities":
		if currentCaseID == "" {
			return CommandResult{Output: "○ No case selected. Use '?case <id>' first.", IsError: true}
		}
		entities, err := storage.ListEntities(currentCaseID)
		if err != nil {
			return CommandResult{Output: fmt.Sprintf("Error fetching evidence: %v", err), IsError: true}
		}
		if len(entities) == 0 {
			return CommandResult{Output: fmt.Sprintf("○ No evidence records found for case %s.", currentCaseID)}
		}
		return CommandResult{Output: RenderEntitiesTable(entities)}

	case "graph":
		if currentCaseID == "" {
			return CommandResult{Output: "○ No case selected. Use '?case <id>' first.", IsError: true}
		}
		return CommandResult{Output: RenderASCIIGraph(currentCaseID)}

	case "timeline":
		if currentCaseID == "" {
			return CommandResult{Output: "○ No case selected. Use '?case <id>' first.", IsError: true}
		}
		return CommandResult{Output: RenderTimeline(currentCaseID)}

	case "report":
		if currentCaseID == "" {
			return CommandResult{Output: "○ No case selected. Use '?case <id>' first.", IsError: true}
		}
		format := "md"
		if len(args) > 0 {
			format = strings.ToLower(args[0])
		}
		if format == "pdf" {
			outDir := filepath.Join("evidence_storage", currentCaseID)
			_ = os.MkdirAll(outDir, 0755)
			pdfPath := filepath.Join(outDir, "investigation_report.pdf")
			if err := report.GeneratePDFReport(currentCaseID, pdfPath); err != nil {
				return CommandResult{Output: fmt.Sprintf("PDF generation failed: %v", err), IsError: true}
			}
			return CommandResult{Output: fmt.Sprintf("✔ Branded PDF report generated at: %s", pdfPath)}
		}
		content, err := report.GenerateMarkdownReport(currentCaseID)
		if err != nil {
			return CommandResult{Output: fmt.Sprintf("Markdown report failed: %v", err), IsError: true}
		}
		mdPath := fmt.Sprintf("report_%s.md", currentCaseID)
		_ = os.WriteFile(mdPath, []byte(content), 0644)
		return CommandResult{Output: fmt.Sprintf("✔ Markdown report saved to %s\n\n%s", mdPath, truncateReport(content, 600))}

	case "collectors":
		var sb strings.Builder
		sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ColorTerracotta).Render("RECONNAISSANCE COLLECTORS\n\n"))
		for _, c := range collector.List() {
			sb.WriteString(fmt.Sprintf("  • %-16s : %s\n", c.Name(), c.Description()))
		}
		sb.WriteString("\nUsage: ?collect <target> [collector_name]")
		return CommandResult{Output: sb.String()}

	case "collect":
		if len(args) < 1 {
			return CommandResult{Output: "Usage: ?collect <target> [collector_name]\nExample: ?collect example.com dns", IsError: true}
		}
		if currentCaseID == "" {
			return CommandResult{Output: "○ No case selected. Please select or create a case first with '?cases'.", IsError: true}
		}
		target := args[0]
		collName := "dns"
		if len(args) > 1 {
			collName = args[1]
		}
		go collector.RunAndSave(collName, currentCaseID, target, false, nil)
		return CommandResult{Output: fmt.Sprintf("✔ Triggered collector '%s' on '%s' for case %s in background.", collName, target, currentCaseID)}

	case "settings", "config":
		return handleSettingsDisplay()

	case "set":
		if len(args) < 2 {
			return CommandResult{Output: "Usage: ?set <key> <value>\nExamples: ?set ghost_mode true, ?set llm.provider openai", IsError: true}
		}
		key, val := args[0], args[1]
		viper.Set(key, val)
		_ = config.SaveConfig()
		return CommandResult{Output: fmt.Sprintf("✔ Setting updated: %s = %s", key, val), ModelChange: strings.Contains(key, "model")}

	case "test":
		ok, latency, msg, _ := analysis.TestLLMConnection()
		if ok {
			return CommandResult{Output: fmt.Sprintf("✔ Connection Verified! Latency: %s\n%s", latency, msg)}
		}
		return CommandResult{Output: fmt.Sprintf("✖ Connection Failed:\n%s", msg), IsError: true}

	case "scan":
		models, err := analysis.FetchAvailableModels()
		if err != nil {
			return CommandResult{Output: fmt.Sprintf("Scan error: %v", err), IsError: true}
		}
		return CommandResult{Output: fmt.Sprintf("✔ Discovered Models (%d):\n%s", len(models), strings.Join(models, ", "))}

	case "web", "dashboard":
		openBrowser("http://localhost:8080")
		return CommandResult{Output: "✔ Opened interactive web dashboard at http://localhost:8080"}

	case "clear":
		return CommandResult{Output: "CLEAR_BUFFER"}

	case "exit", "quit":
		return CommandResult{Quit: true}

	default:
		return CommandResult{
			Output:  fmt.Sprintf("Unknown command '?%s'. Type '?' or '?help' to view the command directory.", cmd),
			IsError: true,
		}
	}
}

func handleCasesList() CommandResult {
	cases, err := storage.ListCases()
	if err != nil {
		return CommandResult{Output: fmt.Sprintf("Error fetching cases: %v", err), IsError: true}
	}
	if len(cases) == 0 {
		return CommandResult{Output: "No cases exist. Type '?case create <name>' to create your first case."}
	}

	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ColorTerracotta).Render(
		fmt.Sprintf("%-16s  %-24s  %-12s  %s\n", "CASE ID", "NAME", "STATUS", "CREATED"),
	))
	sb.WriteString(strings.Repeat("─", 72) + "\n")
	for _, c := range cases {
		sb.WriteString(fmt.Sprintf("%-16s  %-24s  %-12s  %s\n",
			c.ID,
			truncate(c.Name, 22),
			c.Status,
			c.CreatedAt.Format("2006-01-02"),
		))
	}
	sb.WriteString("\nType '?case <id>' to switch active context.")
	return CommandResult{Output: sb.String()}
}

func handleCaseInfo(caseID string) CommandResult {
	if caseID == "" {
		return CommandResult{Output: "No active case. Use '?cases' to select one."}
	}
	c, err := storage.GetCase(caseID)
	if err != nil || c == nil {
		return CommandResult{Output: fmt.Sprintf("Case '%s' not found.", caseID), IsError: true}
	}
	return CommandResult{Output: fmt.Sprintf(
		"Active Case Dossier:\n  • Name:        %s\n  • ID:          %s\n  • Status:      %s\n  • Description: %s",
		c.Name, c.ID, c.Status, c.Description,
	)}
}

func handleSettingsDisplay() CommandResult {
	cfg := viper.AllSettings()
	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ColorTerracotta).Render("SPECTRE CONFIGURATION\n\n"))
	sb.WriteString(fmt.Sprintf("  • LLM Provider : %s\n", viper.GetString("llm.provider")))
	sb.WriteString(fmt.Sprintf("  • LLM Model    : %s\n", viper.GetString("llm.model")))
	sb.WriteString(fmt.Sprintf("  • Endpoint URL : %s\n", viper.GetString("llm.url")))
	sb.WriteString(fmt.Sprintf("  • Ghost Mode   : %v\n", viper.GetBool("ghost_mode")))
	sb.WriteString(fmt.Sprintf("  • HTTP Strict  : %v\n", viper.GetBool("http.strict")))
	sb.WriteString("\nUse '?set <key> <value>' to update configuration.")
	return CommandResult{Output: sb.String()}
}

func RenderHelpDirectory() string {
	header := lipgloss.NewStyle().Bold(true).Foreground(ColorTerracotta).Render("✻ SPECTRE COMMAND DIRECTORY (? COMMANDS)")
	guide := `
  COMMAND                    DESCRIPTION
  ──────────────────────────────────────────────────────────────────────────
  ?                          Show this help directory
  ?cases                     List all cases with status and timestamps
  ?case <id>                 Switch active investigation case
  ?case create <name> [desc] Create a new investigation case
  ?case info                 Display current case metadata
  ?model [name]              View or switch active LLM (e.g. ?model gpt-4o, ?model none)
  ?analyze                   Execute AI analysis and threat synthesis on active case
  ?evidence                  Show extracted intelligence entities and indicators
  ?graph                     Display ASCII entity relation network
  ?timeline                  Display chronological event timeline
  ?report [md|pdf]           Generate Markdown dossier or branded PDF brief
  ?collect <target> [coll]   Execute reconnaissance (DNS, WHOIS, Ports, etc.)
  ?collectors                List all available collectors
  ?settings                  Display active configuration parameters
  ?set <key> <val>           Update configuration key
  ?test                      Test connection and latency to active LLM
  ?scan                      Probe local/remote endpoints for available models
  ?web                       Open interactive web graph dashboard (localhost:8080)
  ?clear                     Clear viewport text
  ?quit                      Exit SPECTRE console

  Note: Any prompt without '?' is dispatched to the Autonomous Agent directly.
`
	return header + "\n" + guide
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max-3] + "..."
	}
	return s
}

func truncateReport(s string, max int) string {
	if len(s) > max {
		return s[:max] + "\n\n... [Report continues on disk]"
	}
	return s
}
```

---

### 5.4 Refactored Horizontal Application Model (`internal/tui/app.go`)

```go
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/Gokul-Eswar/Spectre/internal/agent"
	"github.com/Gokul-Eswar/Spectre/internal/analysis"
	"github.com/Gokul-Eswar/Spectre/internal/analyzer"
	"github.com/Gokul-Eswar/Spectre/internal/storage"
)

type model struct {
	width, height   int
	quitting        bool
	selectedCaseID  string
	modelStatus     DynamicModelStatus
	engine          *agent.Engine
	viewport        viewport.Model
	input           textinput.Model
	spinner         spinner.Model
	isWorking       bool
	workingMessage  string
	commandHistory  []string
	historyIndex    int
}

func InitialModel() model {
	ti := textinput.New()
	ti.Prompt = StylePromptSymbol.Render("❯ ")
	ti.Placeholder = "Ask Spectre or type '?' for commands..."
	ti.Focus()
	ti.CharLimit = 1024

	vp := viewport.New(0, 0)
	welcomeMsg := lipgloss.NewStyle().Foreground(ColorTextMuted).Render(
		"✻ Welcome to SPECTRE Intelligence Platform\n" +
			"Modern horizontal command console with local-first AI synthesis.\n\n" +
			"Type natural language to investigate, or type '?' for available commands.\n",
	)
	vp.SetContent(welcomeMsg)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorTerracotta)

	activeModel := DetectActiveModel()

	m := model{
		input:          ti,
		viewport:       vp,
		spinner:        s,
		modelStatus:    activeModel,
		commandHistory: make([]string, 0),
	}

	// Auto-select latest case if available
	cases, err := storage.ListCases()
	if err == nil && len(cases) > 0 {
		m.selectedCaseID = cases[0].ID
		m.engine = agent.NewEngine(m.selectedCaseID)
	}

	return m
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.spinner.Tick)
}

type agentResultMsg struct {
	content string
	err     error
}

type analysisAsyncMsg struct {
	result *core.Analysis
	err    error
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
		sCmd  tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		
		// Full width responsive allocation (horizontal orientation)
		headerH := 2
		inputH := 4
		viewportH := m.height - headerH - inputH
		if viewportH < 5 {
			viewportH = 5
		}

		m.viewport.Width = m.width - 2
		m.viewport.Height = viewportH
		m.input.Width = m.width - 6

	case agentResultMsg:
		m.isWorking = false
		if msg.err != nil {
			m.appendContent(lipgloss.NewStyle().Foreground(ColorError).Bold(true).Render("✖ Agent Error: ") + msg.err.Error())
		} else {
			m.appendContent(lipgloss.NewStyle().Foreground(ColorTerracotta).Bold(true).Render("✻ Spectre:\n") + msg.content)
		}
		return m, nil

	case analysisAsyncMsg:
		m.isWorking = false
		if msg.err != nil {
			m.appendContent(lipgloss.NewStyle().Foreground(ColorError).Bold(true).Render("✖ Analysis Error: ") + msg.err.Error())
		} else {
			m.appendContent(lipgloss.NewStyle().Foreground(ColorSuccess).Bold(true).Render("✔ Analysis Complete:\n") + FormatAnalysis(msg.result))
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			val := strings.TrimSpace(m.input.Value())
			if val == "" || m.isWorking {
				return m, nil
			}

			// Add to history and reset input
			m.commandHistory = append(m.commandHistory, val)
			m.historyIndex = len(m.commandHistory)
			m.input.SetValue("")

			// Display user command in viewport
			m.appendContent(StylePromptSymbol.Render("❯ ") + lipgloss.NewStyle().Foreground(ColorText).Bold(true).Render(val))

			// Check for ? commands
			if strings.HasPrefix(val, "?") {
				if val == "?analyze" {
					if m.selectedCaseID == "" {
						m.appendContent(lipgloss.NewStyle().Foreground(ColorError).Render("○ No case selected. Use '?case <id>' first."))
						return m, nil
					}
					m.isWorking = true
					m.workingMessage = "Running AI analysis pipeline..."
					return m, func() tea.Msg {
						res, err := analysis.RunFullPipeline(m.selectedCaseID, m.modelStatus.Name)
						return analysisAsyncMsg{result: res, err: err}
					}
				}

				res := ExecuteQuestionCommand(val, m.selectedCaseID)
				if res.Quit {
					m.quitting = true
					return m, tea.Quit
				}
				if res.Output == "CLEAR_BUFFER" {
					m.viewport.SetContent("")
					return m, nil
				}
				if res.ModelChange {
					m.modelStatus = DetectActiveModel()
				}
				// Handle case change
				if strings.HasPrefix(val, "?case ") && !strings.Contains(val, "create") && !strings.Contains(val, "info") {
					parts := strings.Fields(val)
					if len(parts) > 1 {
						m.selectedCaseID = parts[1]
						m.engine = agent.NewEngine(m.selectedCaseID)
					}
				}

				m.appendContent(res.Output)
				return m, nil
			}

			// Natural language query to autonomous agent
			if m.engine == nil {
				if m.selectedCaseID != "" {
					m.engine = agent.NewEngine(m.selectedCaseID)
				} else {
					m.appendContent(lipgloss.NewStyle().Foreground(ColorWarning).Render("○ Please select a case first with '?cases' before starting analysis."))
					return m, nil
				}
			}

			m.isWorking = true
			m.workingMessage = "Spectre is reasoning..."
			return m, func() tea.Msg {
				resp, err := m.engine.Execute(val)
				return agentResultMsg{content: resp, err: err}
			}

		case "up":
			if len(m.commandHistory) > 0 && m.historyIndex > 0 {
				m.historyIndex--
				m.input.SetValue(m.commandHistory[m.historyIndex])
				m.input.SetCursor(len(m.input.Value()))
			}
			return m, nil

		case "down":
			if len(m.commandHistory) > 0 && m.historyIndex < len(m.commandHistory)-1 {
				m.historyIndex++
				m.input.SetValue(m.commandHistory[m.historyIndex])
				m.input.SetCursor(len(m.input.Value()))
			} else if m.historyIndex >= len(m.commandHistory)-1 {
				m.historyIndex = len(m.commandHistory)
				m.input.SetValue("")
			}
			return m, nil
		}
	}

	if m.isWorking {
		m.spinner, sCmd = m.spinner.Update(msg)
	}

	m.input, tiCmd = m.input.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	return m, tea.Batch(tiCmd, vpCmd, sCmd)
}

func (m *model) appendContent(text string) {
	current := m.viewport.View()
	var updated string
	if current == "" {
		updated = text + "\n"
	} else {
		updated = current + "\n\n" + text
	}
	m.viewport.SetContent(updated)
	m.viewport.GotoBottom()
}

func (m model) View() string {
	if m.quitting {
		return "Goodbye, Agent.\n"
	}

	header := m.renderHeader()
	divider := StyleDivider.Render(strings.Repeat("─", m.width))
	content := StyleViewport.Width(m.width).Render(m.viewport.View())
	inputBar := m.renderInputBar()

	return lipgloss.JoinVertical(lipgloss.Left, header, divider, content, inputBar)
}

func (m model) renderHeader() string {
	appName := StyleAppLogo.Render("✻ SPECTRE")

	var modelPill string
	if m.modelStatus.IsConfigured {
		modelPill = StyleModelPillConfigured.Render("Model: " + m.modelStatus.Name)
	} else {
		modelPill = StyleModelPillNone.Render("Model: none")
	}

	gap := m.width - lipgloss.Width(appName) - lipgloss.Width(modelPill) - 2
	if gap < 0 {
		gap = 0
	}

	return lipgloss.NewStyle().
		Padding(0, 1).
		Background(ColorSurface).
		Render(appName + strings.Repeat(" ", gap) + modelPill)
}

func (m model) renderInputBar() string {
	var sb strings.Builder

	// Dynamic work indicator
	if m.isWorking {
		sb.WriteString(lipgloss.NewStyle().Foreground(ColorTerracotta).Render(m.spinner.View()+" "+m.workingMessage) + "\n")
	} else if strings.HasPrefix(m.input.Value(), "?") {
		// Responsive command suggestions strip
		sb.WriteString(StyleSuggestions.Render("[?]: ?cases  ?case <id>  ?model  ?analyze  ?evidence  ?graph  ?timeline  ?report  ?collect  ?help") + "\n")
	}

	inputBox := StyleInputBoxActive.Width(m.width - 4).Render(m.input.View())
	sb.WriteString(inputBox)

	return sb.String()
}
```

---

### 5.5 Unit Test Suite (`internal/tui/app_test.go`)

```go
package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestDetectActiveModel_Configured(t *testing.T) {
	viper.Set("llm.model", "llama3")
	viper.Set("llm.provider", "ollama")

	status := DetectActiveModel()
	assert.True(t, status.IsConfigured)
	assert.Equal(t, "llama3", status.Name)
	assert.Equal(t, "Model: llama3", status.DisplayTag)
}

func TestDetectActiveModel_NoneFallback(t *testing.T) {
	viper.Set("llm.model", "")
	status := DetectActiveModel()
	assert.False(t, status.IsConfigured)
	assert.Equal(t, "none", status.Name)

	viper.Set("llm.model", "none")
	status2 := DetectActiveModel()
	assert.False(t, status2.IsConfigured)
	assert.Equal(t, "none", status2.Name)
}

func TestDetectActiveModel_MissingCloudKey(t *testing.T) {
	viper.Set("llm.provider", "openai")
	viper.Set("llm.model", "gpt-4o")
	viper.Set("llm.api_key", "")
	viper.Set("keys.openai", "")

	status := DetectActiveModel()
	assert.False(t, status.IsConfigured)
	assert.Equal(t, "none", status.Name)
}

func TestExecuteQuestionCommand_Help(t *testing.T) {
	res := ExecuteQuestionCommand("?", "")
	assert.False(t, res.IsError)
	assert.Contains(t, res.Output, "SPECTRE COMMAND DIRECTORY")
	assert.Contains(t, res.Output, "?cases")
	assert.Contains(t, res.Output, "?analyze")
}

func TestExecuteQuestionCommand_ModelSwitch(t *testing.T) {
	res := ExecuteQuestionCommand("?model gpt-4o-mini", "")
	assert.False(t, res.IsError)
	assert.True(t, res.ModelChange)
	assert.Contains(t, res.Output, "gpt-4o-mini")

	// Set back to none
	res2 := ExecuteQuestionCommand("?model none", "")
	assert.True(t, res2.ModelChange)
	assert.Equal(t, "none", DetectActiveModel().Name)
}

func TestExecuteQuestionCommand_Unknown(t *testing.T) {
	res := ExecuteQuestionCommand("?unknownxyz", "")
	assert.True(t, res.IsError)
	assert.Contains(t, res.Output, "Unknown command")
}

func TestHorizontalResizeResponsiveness(t *testing.T) {
	m := InitialModel()
	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	mod := newModel.(model)

	assert.Equal(t, 120, mod.width)
	assert.Equal(t, 40, mod.height)
	assert.Equal(t, 118, mod.viewport.Width)
	assert.Equal(t, 114, mod.input.Width)
}
```

---

## 6. Verification & Step-by-Step Execution Plan

| Step | Action | Files Impacted | Verification Method |
|---|---|---|---|
| **1** | Backup legacy TUI files | `internal/tui/app.go.bak` | Inspect backup copy |
| **2** | Create Dynamic Model Detection Engine | `internal/tui/model_detector.go` | `go test ./internal/tui -run TestDetectActiveModel` |
| **3** | Implement Unified `?` Command Registry | `internal/tui/commands.go` | `go test ./internal/tui -run TestExecuteQuestionCommand` |
| **4** | Refactor Theme to Horizontal Standards | `internal/tui/theme.go` | Style build verification |
| **5** | Implement Horizontal Full-Width TUI Model | `internal/tui/app.go` | `go build ./cmd/spectre` |
| **6** | Update & Expand Test Suite | `internal/tui/app_test.go` | `go test ./internal/tui/... -v` |
| **7** | End-to-End Terminal Smoke Test | `spectre console` | Interactive manual verification of `?cases`, `?model`, `?analyze`, `?graph` |
