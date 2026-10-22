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
	"github.com/Gokul-Eswar/Spectre/internal/core"
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
			newCase := &core.Case{
				Name:        name,
				Description: desc,
			}
			err := storage.CreateCase(newCase)
			if err != nil {
				return CommandResult{Output: fmt.Sprintf("Error creating case: %v", err), IsError: true}
			}
			return CommandResult{Output: fmt.Sprintf("✔ Case created successfully: %s (ID: %s)", newCase.Name, newCase.ID)}
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
		entities, err := storage.ListEntitiesByCase(currentCaseID)
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
		sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ColorClaudeTerracotta).Render("RECONNAISSANCE COLLECTORS\n\n"))
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
		_ = openBrowser("http://localhost:8080")
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
	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ColorClaudeTerracotta).Render(
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
	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ColorClaudeTerracotta).Render("SPECTRE CONFIGURATION\n\n"))
	sb.WriteString(fmt.Sprintf("  • LLM Provider : %s\n", viper.GetString("llm.provider")))
	sb.WriteString(fmt.Sprintf("  • LLM Model    : %s\n", viper.GetString("llm.model")))
	sb.WriteString(fmt.Sprintf("  • Endpoint URL : %s\n", viper.GetString("llm.url")))
	sb.WriteString(fmt.Sprintf("  • Ghost Mode   : %v\n", viper.GetBool("ghost_mode")))
	sb.WriteString(fmt.Sprintf("  • HTTP Strict  : %v\n", viper.GetBool("http.strict")))
	sb.WriteString("\nUse '?set <key> <value>' to update configuration.")
	return CommandResult{Output: sb.String()}
}

// RenderEntitiesTable formats a slice of entities into a clean readable table
func RenderEntitiesTable(entities []*core.Entity) string {
	if len(entities) == 0 {
		return "No entities found for this case."
	}
	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ColorClaudeTerracotta).Render(
		fmt.Sprintf("%-16s  %-36s  %s\n", "TYPE", "VALUE", "SOURCE"),
	))
	sb.WriteString(strings.Repeat("─", 70) + "\n")
	for _, e := range entities {
		sb.WriteString(fmt.Sprintf("%-16s  %-36s  %s\n",
			truncate(e.Type, 14),
			truncate(e.Value, 34),
			truncate(e.Source, 16),
		))
	}
	return sb.String()
}

// RenderHelpDirectory formats the complete interactive command guide
func RenderHelpDirectory() string {
	header := lipgloss.NewStyle().Bold(true).Foreground(ColorClaudeTerracotta).Render("✻ SPECTRE COMMAND DIRECTORY (? COMMANDS)")
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
