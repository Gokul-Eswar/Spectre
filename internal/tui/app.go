package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/Gokul-Eswar/Spectre/internal/analysis"
	"github.com/Gokul-Eswar/Spectre/internal/analyzer"
	"github.com/Gokul-Eswar/Spectre/internal/config"
	"github.com/Gokul-Eswar/Spectre/internal/core"
	"github.com/Gokul-Eswar/Spectre/internal/report"
	"github.com/spf13/viper"
)

type sessionState int

const (
	ViewCases sessionState = iota
	ViewAnalysis
	ViewEvidence
	ViewGraph
	ViewTimeline
	ViewReports
	ViewChat
	ViewSettings
	ViewDashboard
	ViewCollection
)

type analysisStatus int

const (
	AnalysisIdle analysisStatus = iota
	AnalysisRunning
	AnalysisComplete
	AnalysisError
)

const (
	SettingLLMProvider = iota
	SettingLLMModel
	SettingLLMEndpoint
	SettingLLMKey
	SettingLLMTimeout
	SettingLLMTest
	SettingGhostMode
	SettingCollectorDNS
	SettingCollectorWhois
	SettingCollectorGitHub
	SettingCollectorGeo
	SettingCollectorPorts
	SettingCollectorSocial
	SettingCollectorScreenshot
	TotalSettingsCount
)

type model struct {
	state          sessionState
	cursor         int
	quitting       bool
	width          int
	height         int
	selectedCaseID string
	modelName      string // For status bar and prompt routing
	focusNav       bool   // Focus state: true=Sidebar, false=MainContent
	navCursor      int    // Cursor for sidebar selection

	// LLM Configuration & Settings State
	settingsCursor   int
	llmProvider      string
	llmEndpoint      string
	llmApiKey        string
	llmTimeout       int
	testStatus       string
	testRunning      bool
	isEditingSetting bool
	editSettingIndex int
	settingInput     textinput.Model

	availableModels []string

	// Analysis State
	analysisStatus analysisStatus
	analysisStep   int
	analysisResult string
	analysisError  string

	// Cached Data for Views
	graphData    string
	timelineData string

	// Sub-models
	caseList    list.Model
	entityTable table.Model
	relTable    table.Model
	runner      runnerModel
	chat        chatModel
}

func InitialModel() model {
	l := list.New([]list.Item{}, NewClaudeListDelegate(), 0, 0)
	l.Title = "CASES"
	l.Styles.Title = lipgloss.NewStyle().
		Foreground(ColorClaudeTerracotta).
		Bold(true)

	cfg := analyzer.GetLLMConfig()
	activeModel := viper.GetString("llm.model")
	if activeModel == "" {
		activeModel = analyzer.GetDefaultModelForProvider(cfg.Provider)
	}

	ti := textinput.New()
	ti.Prompt = "❯ "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(ColorClaudeTerracotta).Bold(true)
	ti.CharLimit = 256
	ti.Width = 60

	_, _, presets := analysis.GetPresetsForProvider(cfg.Provider)

	return model{
		state:            ViewCases,
		navCursor:        0,
		settingsCursor:   0,
		caseList:         l,
		entityTable:      NewEntityTable(),
		relTable:         NewRelationshipTable(),
		runner:           NewRunnerModel(),
		chat:             newChatModel(),
		modelName:        activeModel,
		llmProvider:      cfg.Provider,
		llmEndpoint:      cfg.URL,
		llmApiKey:        cfg.APIKey,
		llmTimeout:       cfg.Timeout,
		settingInput:     ti,
		availableModels:  presets,
		isEditingSetting: false,
	}
}

func (m model) Init() tea.Cmd {
	return func() tea.Msg {
		items, err := FetchCases()
		if err != nil {
			return err
		}
		return items
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

		// Update sub-models sizes
		h := msg.Height - 6 // Space for header and footer
		w := msg.Width - 25 // Space for nav

		m.caseList.SetSize(w, h)
		m.runner.caseList.SetSize(w, h)
		m.runner.collList.SetSize(w, h)
		m.chat.setSize(w, h)

	case []list.Item:
		m.caseList.SetItems(msg)
		m.runner.caseList.SetItems(msg)

	case []table.Row:
		if m.state == ViewEvidence {
			m.entityTable.SetRows(msg)
		}

	case TickMsg:
		if m.state == ViewAnalysis && m.analysisStatus == AnalysisRunning {
			if m.analysisStep < 4 {
				m.analysisStep++
				return m, tickCmd()
			}
			return m, nil
		}

	case AnalysisFinishedMsg:
		m.analysisStatus = AnalysisComplete
		if msg.Result != nil {
			m.analysisResult = FormatAnalysis(msg.Result)
		} else {
			// This was a report generation success
			m.analysisStatus = AnalysisComplete
			m.analysisResult = "Report generated successfully! Check report_<case_id>.md"
		}

	case ModelsFoundMsg:
		m.availableModels = msg
		// If current model is not in list, switch to first
		if len(m.availableModels) > 0 {
			found := false
			for _, mod := range m.availableModels {
				if strings.EqualFold(mod, m.modelName) {
					found = true
					break
				}
			}
			if !found {
				m.modelName = m.availableModels[0]
				viper.Set("llm.model", m.modelName)
				_ = config.SaveConfig()
			}
		}

	case TestConnectionMsg:
		m.testRunning = false
		if msg.Success {
			m.testStatus = lipgloss.NewStyle().Foreground(ColorSuccess).Render("✓ " + msg.Message)
		} else {
			m.testStatus = lipgloss.NewStyle().Foreground(ColorError).Render("✗ " + msg.Message)
		}
		return m, nil

	case AnalysisErrorMsg:
		m.analysisStatus = AnalysisError
		m.analysisError = string(msg)

	case GraphDataMsg:
		m.graphData = string(msg)

	case TimelineDataMsg:
		m.timelineData = string(msg)

	case agentResponseMsg:
		m.chat, cmd = m.chat.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		// When actively editing an LLM setting text field, route keystrokes to textinput
		if m.state == ViewSettings && m.isEditingSetting {
			switch msg.String() {
			case "ctrl+c":
				m.quitting = true
				return m, tea.Quit
			case "esc":
				m.isEditingSetting = false
				return m, nil
			case "enter":
				val := strings.TrimSpace(m.settingInput.Value())
				switch m.editSettingIndex {
				case SettingLLMModel:
					if val != "" {
						m.modelName = val
						viper.Set("llm.model", m.modelName)
						found := false
						for _, mod := range m.availableModels {
							if strings.EqualFold(mod, val) {
								found = true
								break
							}
						}
						if !found {
							m.availableModels = append([]string{val}, m.availableModels...)
						}
					}
				case SettingLLMEndpoint:
					if val != "" {
						m.llmEndpoint = val
						viper.Set("llm.url", m.llmEndpoint)
					}
				case SettingLLMKey:
					m.llmApiKey = val
					viper.Set("llm.api_key", m.llmApiKey)
				}
				_ = config.SaveConfig()
				m.isEditingSetting = false
				return m, nil
			default:
				m.settingInput, cmd = m.settingInput.Update(msg)
				return m, cmd
			}
		}

		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "tab":
			m.focusNav = !m.focusNav
			return m, nil
		case "right":
			if !m.focusNav && m.state != ViewSettings {
				m.focusNav = false
				return m, nil
			}
			if m.focusNav {
				m.focusNav = false
				return m, nil
			}
		case "left":
			if !m.focusNav && m.state != ViewSettings {
				m.focusNav = true
				return m, nil
			}
		}

		if m.focusNav {
			switch msg.String() {
			case "j", "down":
				if m.navCursor < int(ViewCollection) {
					m.navCursor++
				}
			case "k", "up":
				if m.navCursor > 0 {
					m.navCursor--
				}
			case "enter":
				m.state = sessionState(m.navCursor)
				// Trigger specific view logic if needed
				if m.state == ViewAnalysis && m.selectedCaseID != "" && m.analysisStatus == AnalysisIdle {
					m.analysisStatus = AnalysisRunning
					m.analysisStep = 0
					return m, StartAnalysis(m.selectedCaseID, m.modelName)
				}
				if m.state == ViewDashboard {
					openBrowser("http://localhost:8080")
				}
				// Auto-refresh data when entering view via sidebar
				if m.state == ViewGraph && m.selectedCaseID != "" {
					return m, fetchGraphDataCmd(m.selectedCaseID)
				}
				if m.state == ViewTimeline && m.selectedCaseID != "" {
					return m, fetchTimelineDataCmd(m.selectedCaseID)
				}
			}
			return m, nil
		}

		// Navigation logic (j/k or arrows)
		if m.state == ViewCases {
			var cmd2 tea.Cmd
			m.caseList, cmd2 = m.caseList.Update(msg)

			if msg.String() == "enter" {
				if selected, ok := m.caseList.SelectedItem().(item); ok {
					m.selectedCaseID = selected.id
					m.chat.setCase(selected.id)
					m.state = ViewEvidence
					return m, func() tea.Msg {
						rows, err := FetchEntities(selected.id)
						if err != nil {
							return err
						}
						return rows
					}
				}
			}
			return m, cmd2
		}

		if m.state == ViewEvidence {
			var cmd2 tea.Cmd
			m.entityTable, cmd2 = m.entityTable.Update(msg)
			return m, cmd2
		}

		if m.state == ViewChat {
			m.chat, cmd = m.chat.Update(msg)
			return m, cmd
		}

		if m.state == ViewReports {
			switch msg.String() {
			case "p":
				if m.selectedCaseID != "" {
					outPath := fmt.Sprintf("evidence_storage/%s/investigation_report.pdf", m.selectedCaseID)
					_ = report.GeneratePDFReport(m.selectedCaseID, outPath)
				}
			case "v":
				m.state = ViewAnalysis
				if m.selectedCaseID != "" && m.analysisStatus == AnalysisIdle {
					m.analysisStatus = AnalysisRunning
					m.analysisStep = 0
					return m, StartAnalysis(m.selectedCaseID, m.modelName)
				}
			}
		}

		// Global navigation if not in a list
		switch msg.String() {
		case "1":
			m.state = ViewCases
		case "2":
			m.state = ViewAnalysis
			if m.selectedCaseID != "" && m.analysisStatus == AnalysisIdle {
				m.analysisStatus = AnalysisRunning
				m.analysisStep = 0
				return m, StartAnalysis(m.selectedCaseID, m.modelName)
			}
		case "3":
			m.state = ViewEvidence
		case "4":
			m.state = ViewGraph
			if m.selectedCaseID != "" {
				return m, fetchGraphDataCmd(m.selectedCaseID)
			}
		case "5":
			m.state = ViewTimeline
			if m.selectedCaseID != "" {
				return m, fetchTimelineDataCmd(m.selectedCaseID)
			}
		case "6":
			m.state = ViewReports
		case "7":
			m.state = ViewSettings
		case "8":
			m.state = ViewChat
		case "9":
			m.state = ViewDashboard
		case "0":
			m.state = ViewCollection
		}

		if m.state == ViewCollection {
			var cmd2 tea.Cmd
			m.runner, cmd2 = m.runner.Update(msg)
			return m, cmd2
		}

		// Specific View Keybindings for Settings
		if m.state == ViewSettings {
			switch msg.String() {
			case "j", "down":
				if m.settingsCursor < TotalSettingsCount-1 {
					m.settingsCursor++
				}
				return m, nil
			case "k", "up":
				if m.settingsCursor > 0 {
					m.settingsCursor--
				}
				return m, nil
			case "r":
				if m.settingsCursor == SettingLLMModel || m.settingsCursor == SettingLLMTest {
					return m, fetchModelsCmd
				}
			case "e":
				switch m.settingsCursor {
				case SettingLLMModel:
					m.isEditingSetting = true
					m.editSettingIndex = SettingLLMModel
					m.settingInput.SetValue(m.modelName)
					m.settingInput.Placeholder = "Enter model name (e.g. gpt-4o, llama3)"
					m.settingInput.Focus()
					return m, textinput.Blink
				case SettingLLMEndpoint:
					m.isEditingSetting = true
					m.editSettingIndex = SettingLLMEndpoint
					m.settingInput.SetValue(m.llmEndpoint)
					m.settingInput.Placeholder = "http://localhost:11434 or https://api.openai.com/v1"
					m.settingInput.Focus()
					return m, textinput.Blink
				case SettingLLMKey:
					m.isEditingSetting = true
					m.editSettingIndex = SettingLLMKey
					m.settingInput.SetValue(m.llmApiKey)
					m.settingInput.Placeholder = "Enter API Key (leave empty to clear)"
					m.settingInput.Focus()
					return m, textinput.Blink
				}
			case "left", "h":
				if m.settingsCursor == SettingLLMProvider {
					providers := analysis.GetSupportedProviders()
					currIdx := 0
					for i, p := range providers {
						if strings.EqualFold(p, m.llmProvider) {
							currIdx = i
							break
						}
					}
					prevIdx := (currIdx - 1 + len(providers)) % len(providers)
					m.llmProvider = providers[prevIdx]
					viper.Set("llm.provider", m.llmProvider)

					defaultURL, defaultModel, presets := analysis.GetPresetsForProvider(m.llmProvider)
					m.llmEndpoint = defaultURL
					viper.Set("llm.url", m.llmEndpoint)
					m.modelName = defaultModel
					viper.Set("llm.model", m.modelName)
					m.availableModels = presets
					m.testStatus = ""
					_ = config.SaveConfig()
					return m, nil
				}
			case "right", "l":
				if m.settingsCursor == SettingLLMProvider {
					providers := analysis.GetSupportedProviders()
					currIdx := 0
					for i, p := range providers {
						if strings.EqualFold(p, m.llmProvider) {
							currIdx = i
							break
						}
					}
					nextIdx := (currIdx + 1) % len(providers)
					m.llmProvider = providers[nextIdx]
					viper.Set("llm.provider", m.llmProvider)

					defaultURL, defaultModel, presets := analysis.GetPresetsForProvider(m.llmProvider)
					m.llmEndpoint = defaultURL
					viper.Set("llm.url", m.llmEndpoint)
					m.modelName = defaultModel
					viper.Set("llm.model", m.modelName)
					m.availableModels = presets
					m.testStatus = ""
					_ = config.SaveConfig()
					return m, nil
				}
			case "enter", " ":
				switch m.settingsCursor {
				case SettingLLMProvider:
					providers := analysis.GetSupportedProviders()
					currIdx := 0
					for i, p := range providers {
						if strings.EqualFold(p, m.llmProvider) {
							currIdx = i
							break
						}
					}
					nextIdx := (currIdx + 1) % len(providers)
					m.llmProvider = providers[nextIdx]
					viper.Set("llm.provider", m.llmProvider)

					defaultURL, defaultModel, presets := analysis.GetPresetsForProvider(m.llmProvider)
					m.llmEndpoint = defaultURL
					viper.Set("llm.url", m.llmEndpoint)
					m.modelName = defaultModel
					viper.Set("llm.model", m.modelName)
					m.availableModels = presets
					m.testStatus = ""
					_ = config.SaveConfig()

				case SettingLLMModel:
					if len(m.availableModels) > 0 {
						currIdx := 0
						for i, mod := range m.availableModels {
							if strings.EqualFold(mod, m.modelName) {
								currIdx = i
								break
							}
						}
						nextIdx := (currIdx + 1) % len(m.availableModels)
						m.modelName = m.availableModels[nextIdx]
						viper.Set("llm.model", m.modelName)
						_ = config.SaveConfig()
					} else {
						m.isEditingSetting = true
						m.editSettingIndex = SettingLLMModel
						m.settingInput.SetValue(m.modelName)
						m.settingInput.Focus()
						return m, textinput.Blink
					}

				case SettingLLMEndpoint:
					m.isEditingSetting = true
					m.editSettingIndex = SettingLLMEndpoint
					m.settingInput.SetValue(m.llmEndpoint)
					m.settingInput.Focus()
					return m, textinput.Blink

				case SettingLLMKey:
					m.isEditingSetting = true
					m.editSettingIndex = SettingLLMKey
					m.settingInput.SetValue(m.llmApiKey)
					m.settingInput.Focus()
					return m, textinput.Blink

				case SettingLLMTimeout:
					timeouts := []int{30, 60, 120, 180, 300}
					currIdx := 2
					for i, t := range timeouts {
						if t == m.llmTimeout {
							currIdx = i
							break
						}
					}
					nextIdx := (currIdx + 1) % len(timeouts)
					m.llmTimeout = timeouts[nextIdx]
					viper.Set("llm.timeout", m.llmTimeout)
					_ = config.SaveConfig()

				case SettingLLMTest:
					m.testRunning = true
					m.testStatus = "Testing connection..."
					return m, testConnectionCmd

				case SettingGhostMode:
					viper.Set("ghost_mode", !viper.GetBool("ghost_mode"))
					_ = config.SaveConfig()

				case SettingCollectorDNS:
					viper.Set("collectors.dns.enabled", !viper.GetBool("collectors.dns.enabled"))
					config.ApplyEthicsConfig()
					_ = config.SaveConfig()

				case SettingCollectorWhois:
					viper.Set("collectors.whois.enabled", !viper.GetBool("collectors.whois.enabled"))
					config.ApplyEthicsConfig()
					_ = config.SaveConfig()

				case SettingCollectorGitHub:
					viper.Set("collectors.github.enabled", !viper.GetBool("collectors.github.enabled"))
					config.ApplyEthicsConfig()
					_ = config.SaveConfig()

				case SettingCollectorGeo:
					viper.Set("collectors.geo.enabled", !viper.GetBool("collectors.geo.enabled"))
					config.ApplyEthicsConfig()
					_ = config.SaveConfig()

				case SettingCollectorPorts:
					viper.Set("collectors.ports.enabled", !viper.GetBool("collectors.ports.enabled"))
					config.ApplyEthicsConfig()
					_ = config.SaveConfig()

				case SettingCollectorSocial:
					viper.Set("collectors.social.enabled", !viper.GetBool("collectors.social.enabled"))
					config.ApplyEthicsConfig()
					_ = config.SaveConfig()

				case SettingCollectorScreenshot:
					viper.Set("collectors.screenshot.enabled", !viper.GetBool("collectors.screenshot.enabled"))
					config.ApplyEthicsConfig()
					_ = config.SaveConfig()
				}
				return m, nil
			}
		}

		if m.state == ViewReports {
			if msg.String() == "1" && m.selectedCaseID != "" {
				return m, func() tea.Msg {
					content, err := report.GenerateMarkdownReport(m.selectedCaseID)
					if err != nil {
						return AnalysisErrorMsg(err.Error())
					}
					filename := fmt.Sprintf("report_%s.md", m.selectedCaseID)
					_ = os.WriteFile(filename, []byte(content), 0644)
					return AnalysisFinishedMsg{nil}
				}
			}
			if msg.String() == "p" && m.selectedCaseID != "" {
				return m, func() tea.Msg {
					outputDir := filepath.Join("evidence_storage", m.selectedCaseID)
					_ = os.MkdirAll(outputDir, 0755)
					outputPath := filepath.Join(outputDir, "investigation_report.pdf")
					err := report.GeneratePDFReport(m.selectedCaseID, outputPath)
					if err != nil {
						return AnalysisErrorMsg(err.Error())
					}
					return AnalysisFinishedMsg{&core.Analysis{
						Findings: []string{fmt.Sprintf("PDF Report generated: %s", outputPath)},
					}}
				}
			}
		}
	}

	return m, cmd
}

func (m model) View() string {
	if m.quitting {
		return "Goodbye, Agent.\n"
	}

	header := m.renderHeader()
	nav := m.renderNav()
	content := m.renderContent()
	footer := m.renderFooter()

	mainArea := lipgloss.JoinHorizontal(lipgloss.Top, nav, content)
	appView := lipgloss.JoinVertical(lipgloss.Left, header, mainArea, footer)
	return lipgloss.NewStyle().Background(ColorBG).Render(appView)
}

func (m model) renderHeader() string {
	logo := lipgloss.NewStyle().Foreground(ColorClaudeTerracotta).Bold(true).Render("✻ SPECTRE")
	ver := StylePillSubtle.Render("v1.0")
	brand := fmt.Sprintf("%s %s", logo, ver)

	casePill := StyleDim.Render("○ No Case Selected")
	if m.selectedCaseID != "" {
		casePill = lipgloss.NewStyle().
			Foreground(ColorClaudeText).
			Background(ColorClaudeSurfaceHigh).
			Bold(true).
			Padding(0, 1).
			Render(fmt.Sprintf("⬡ %s", m.selectedCaseID))
	}

	modelPill := lipgloss.NewStyle().
		Foreground(ColorClaudeSecondary).
		Background(ColorClaudeSurfaceHigh).
		Padding(0, 1).
		Render(fmt.Sprintf("⚡ [%s] %s", m.llmProvider, m.modelName))

	left := lipgloss.JoinHorizontal(lipgloss.Center, brand, "  ", casePill)
	right := modelPill

	gapSize := m.width - lipgloss.Width(left) - lipgloss.Width(right) - 4
	if gapSize < 0 {
		gapSize = 0
	}

	header := lipgloss.JoinHorizontal(lipgloss.Center, left, strings.Repeat(" ", gapSize), right)
	return StyleHeader.Width(m.width).Render(header)
}

func (m model) renderNav() string {
	views := []struct {
		state sessionState
		icon  string
		label string
	}{
		{ViewCases, IconFolder, "Cases"},
		{ViewAnalysis, IconBrain, "Analysis"},
		{ViewEvidence, IconEvidence, "Evidence"},
		{ViewGraph, IconGraph, "Graph"},
		{ViewTimeline, IconTimeline, "Timeline"},
		{ViewReports, IconReport, "Reports"},
		{ViewChat, IconChat, "Chat"},
		{ViewSettings, IconSettings, "Settings"},
		{ViewDashboard, IconWeb, "Web Dashboard"},
		{ViewCollection, IconRunner, "Collection"},
	}

	var s strings.Builder
	s.WriteString("\n")
	for i, v := range views {
		line := fmt.Sprintf("%s %s", v.icon, v.label)

		prefix := "  "
		if m.navCursor == i {
			prefix = lipgloss.NewStyle().Foreground(ColorClaudeTerracotta).Bold(true).Render("❯ ")
		} else if m.state == v.state {
			prefix = lipgloss.NewStyle().Foreground(ColorClaudePeach).Render("● ")
		}

		var content string
		if m.navCursor == i {
			content = StyleSelectedNav.Render(line)
		} else if m.state == v.state {
			content = lipgloss.NewStyle().Foreground(ColorClaudeText).Bold(true).Render(line)
		} else {
			content = StyleMuted.Render(line)
		}

		s.WriteString(prefix)
		s.WriteString(content)
		s.WriteString("\n")
	}

	s.WriteString("\n")
	s.WriteString(StyleDim.Render(" ── Shortcuts ──\n"))
	s.WriteString(StyleDim.Render(" [Tab] Focus\n"))
	s.WriteString(StyleDim.Render(" [j/k] Navigate\n"))
	s.WriteString(StyleDim.Render(" [↵]   Select\n"))

	style := StyleNav.Height(m.height - 4).Width(22)
	if m.focusNav {
		style = style.BorderForeground(ColorClaudeTerracotta)
	}
	return style.Render(s.String())
}

func (m model) renderContent() string {
	var content string
	switch m.state {
	case ViewCases:
		content = m.caseList.View()
	case ViewEvidence:
		var b strings.Builder
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ColorClaudeTerracotta).Render("✦ EVIDENCE DOSSIER"))
		if m.selectedCaseID != "" {
			b.WriteString(StyleMuted.Render(fmt.Sprintf("  •  Case: %s", m.selectedCaseID)))
		}
		b.WriteString("\n" + StyleDim.Render(strings.Repeat("─", 45)) + "\n\n")
		b.WriteString(m.entityTable.View())
		content = b.String()

	case ViewAnalysis:
		if m.selectedCaseID == "" {
			content = StyleCard.Render("○ No case selected. Please select a case first (Press 1).")
		} else {
			switch m.analysisStatus {
			case AnalysisIdle:
				content = StyleCard.Render(fmt.Sprintf(
					"%s\n%s\n\nActive Case : %s\nEngine      : %s (%s)\n\nPress %s to start analysis.",
					StyleTitle.Render("✻ AI INVESTIGATION ENGINE"),
					StyleDim.Render(strings.Repeat("─", 46)),
					m.selectedCaseID,
					m.modelName,
					m.llmProvider,
					lipgloss.NewStyle().Foreground(ColorClaudeTerracotta).Bold(true).Render("'2'"),
				))
			case AnalysisRunning:
				steps := []string{
					"Collecting evidence...",
					"Cross-checking sources...",
					"Reasoning over timeline...",
					"Synthesizing conclusion...",
				}
				var s strings.Builder
				s.WriteString(StyleTitle.Render(fmt.Sprintf("✻ ANALYSIS — Case %s", m.selectedCaseID)) + "\n")
				s.WriteString(StyleDim.Render(strings.Repeat("─", 46)) + "\n\n")
				s.WriteString(lipgloss.NewStyle().Foreground(ColorClaudeTerracotta).Render("⠋ Running analysis pipeline...\n\n"))

				for i, step := range steps {
					if m.analysisStep > i {
						s.WriteString(fmt.Sprintf("  %s %s\n", lipgloss.NewStyle().Foreground(ColorClaudeSuccess).Render("✔"), step))
					} else if m.analysisStep == i {
						s.WriteString(fmt.Sprintf("  %s %s\n", lipgloss.NewStyle().Foreground(ColorClaudeTerracotta).Render("✻"), lipgloss.NewStyle().Bold(true).Foreground(ColorClaudeText).Render(step)))
					} else {
						s.WriteString(fmt.Sprintf("  %s %s\n", StyleDim.Render("○"), StyleDim.Render(step)))
					}
				}
				content = StyleCard.Render(s.String())
			case AnalysisComplete:
				content = m.analysisResult
			case AnalysisError:
				content = StyleCard.BorderForeground(ColorClaudeError).Render(fmt.Sprintf("✖ Analysis Pipeline Error:\n\n%s", m.analysisError))
			}
		}
	case ViewGraph:
		if m.graphData == "" {
			content = StyleCard.Render("Loading graph data...")
		} else {
			content = m.graphData
		}
	case ViewTimeline:
		if m.timelineData == "" {
			content = StyleCard.Render("Loading timeline data...")
		} else {
			content = m.timelineData
		}
	case ViewReports:
		var b strings.Builder
		b.WriteString(StyleTitle.Render("✦ INVESTIGATION REPORTS") + "\n")
		b.WriteString(StyleDim.Render(strings.Repeat("─", 48)) + "\n\n")

		card1 := StyleCard.Width(52).Render(fmt.Sprintf("%s\n%s",
			lipgloss.NewStyle().Bold(true).Foreground(ColorClaudeTerracotta).Render("[ 1 ] Generate Markdown Report"),
			StyleMuted.Render("Export structured markdown dossier into workspace.")))

		card2 := StyleCard.Width(52).Render(fmt.Sprintf("%s\n%s",
			lipgloss.NewStyle().Bold(true).Foreground(ColorClaudePeach).Render("[ p ] Generate Professional PDF"),
			StyleMuted.Render("Compile an executive branded investigation brief PDF.")))

		card3 := StyleCard.Width(52).Render(fmt.Sprintf("%s\n%s",
			lipgloss.NewStyle().Bold(true).Foreground(ColorClaudeInfo).Render("[ v ] View Latest Analysis"),
			StyleMuted.Render("Inspect automated AI findings, risks, and next steps.")))

		b.WriteString(lipgloss.JoinVertical(lipgloss.Left, card1, card2, card3))

		if m.analysisStatus == AnalysisComplete {
			b.WriteString("\n\n" + StyleCard.BorderForeground(ColorClaudeSuccess).Render(lipgloss.NewStyle().Foreground(ColorClaudeSuccess).Render("✔ "+m.analysisResult)))
		}
		content = b.String()

	case ViewChat:
		if m.selectedCaseID == "" {
			content = StyleCard.Render("○ No case selected. Please select a case first (Press 1).")
		} else {
			content = m.chat.View()
		}

	case ViewSettings:
		if m.isEditingSetting {
			title := "Edit Setting"
			hint := "Press Enter to save, Esc to cancel"
			switch m.editSettingIndex {
			case SettingLLMModel:
				title = "Enter Model Name"
				hint = "e.g. claude-3-7-sonnet, gpt-4o, llama3, deepseek-r1"
			case SettingLLMEndpoint:
				title = "Enter LLM Endpoint Base URL"
				hint = "e.g. https://api.anthropic.com, http://localhost:11434, https://api.openai.com/v1"
			case SettingLLMKey:
				title = "Enter API Key"
				hint = "Paste your secret API key (leave empty to clear)"
			}

			boxStyle := StyleCardFocused.Width(65)

			var b strings.Builder
			b.WriteString(StyleTitle.Render(fmt.Sprintf("✻ %s", title)) + "\n\n")
			b.WriteString(m.settingInput.View())
			b.WriteString("\n\n")
			b.WriteString(StyleDim.Render(hint) + "\n")
			b.WriteString(StyleDim.Render("[Enter: Save  •  Esc: Cancel]"))

			content = "SETTINGS — EDITING\n──────────────────\n\n" + boxStyle.Render(b.String())
			break
		}

		keyLabel := maskAPIKey(m.llmApiKey)
		testLabel := "[ Test Connection ]"
		if m.testRunning {
			testLabel = "[ Testing... ]"
		} else if m.testStatus != "" {
			testLabel = fmt.Sprintf("[ Test Connection ] %s", m.testStatus)
		}

		modelLabel := m.modelName
		if len(m.availableModels) > 1 {
			modelLabel += fmt.Sprintf(" (%d preset/detected)", len(m.availableModels))
		}

		type settingItem struct {
			section string
			label   string
			val     string
			hint    string
		}

		items := []settingItem{
			{section: "LLM CONFIGURATION", label: "Provider", val: m.llmProvider, hint: "Enter/Space: cycle | Left/Right: change"},
			{section: "", label: "Model", val: modelLabel, hint: "Enter: cycle | e: edit | r: scan"},
			{section: "", label: "Endpoint URL", val: m.llmEndpoint, hint: "Enter/e: edit"},
			{section: "", label: "API Key", val: keyLabel, hint: "Enter/e: edit"},
			{section: "", label: "Timeout", val: fmt.Sprintf("%ds", m.llmTimeout), hint: "Enter: cycle"},
			{section: "", label: "Test Connection", val: testLabel, hint: "Enter: run connection test"},

			{section: "GENERAL & COLLECTORS", label: "Ghost Mode", val: formatBool(viper.GetBool("ghost_mode")), hint: "Enter: toggle"},
			{section: "", label: "DNS Collector", val: formatBool(viper.GetBool("collectors.dns.enabled")), hint: "Enter: toggle"},
			{section: "", label: "Whois Collector", val: formatBool(viper.GetBool("collectors.whois.enabled")), hint: "Enter: toggle"},
			{section: "", label: "GitHub Collector", val: formatBool(viper.GetBool("collectors.github.enabled")), hint: "Enter: toggle"},
			{section: "", label: "Geo Collector", val: formatBool(viper.GetBool("collectors.geo.enabled")), hint: "Enter: toggle"},
			{section: "", label: "Ports Collector", val: formatBool(viper.GetBool("collectors.ports.enabled")), hint: "Enter: toggle"},
			{section: "", label: "Social Collector", val: formatBool(viper.GetBool("collectors.social.enabled")), hint: "Enter: toggle"},
			{section: "", label: "Screenshot Collector", val: formatBool(viper.GetBool("collectors.screenshot.enabled")), hint: "Enter: toggle"},
		}

		var s strings.Builder
		s.WriteString(StyleTitle.Render("✦ SETTINGS & CONFIGURATION") + "\n")
		s.WriteString(StyleDim.Render(strings.Repeat("─", 50)) + "\n\n")

		for i, item := range items {
			if item.section != "" {
				s.WriteString("\n" + lipgloss.NewStyle().Bold(true).Foreground(ColorClaudeTerracotta).Render(fmt.Sprintf("── ✦ %s ──", item.section)) + "\n")
			}

			cursor := "  "
			if m.settingsCursor == i {
				cursor = lipgloss.NewStyle().Foreground(ColorClaudeTerracotta).Bold(true).Render("❯ ")
			}

			lblStyle := StyleMuted
			valStyle := lipgloss.NewStyle().Foreground(ColorClaudeText)
			if m.settingsCursor == i {
				lblStyle = StyleSelectedNav
				valStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF"))
			}

			line := fmt.Sprintf("%s%-22s : %s", cursor, lblStyle.Render(item.label), valStyle.Render(item.val))
			s.WriteString(line)

			if m.settingsCursor == i && item.hint != "" {
				s.WriteString("  " + StyleDim.Render("("+item.hint+")"))
			}
			s.WriteString("\n")
		}

		s.WriteString("\n" + StyleDim.Render("Navigation: 'j'/'k' (Up/Down)  |  'Enter' Select/Cycle  |  'e' Edit Text  |  'r' Scan Models"))
		content = s.String()

	case ViewDashboard:
		content = StyleCard.Render(fmt.Sprintf(
			"%s\n%s\n\nInteractive Browser Interface: %s\n\nVisual interactive network graphs, entity manipulation,\nand advanced investigation management dashboard.",
			StyleTitle.Render("✻ WEB DASHBOARD"),
			StyleDim.Render(strings.Repeat("─", 50)),
			lipgloss.NewStyle().Foreground(ColorClaudeInfo).Underline(true).Render("http://localhost:8080"),
		))

	case ViewCollection:
		content = m.runner.View()

	default:
		content = StyleCard.Render(fmt.Sprintf(
			"%s\n%s\n\nActive Case  : %s\nModel Engine : %s (%s)\nStatus       : Ready",
			StyleTitle.Render("✻ INTELLIGENCE SUMMARY"),
			StyleDim.Render(strings.Repeat("─", 45)),
			m.selectedCaseID,
			m.modelName,
			m.llmProvider,
		))
	}

	return StyleMain.Width(m.width - 25).Height(m.height - 4).Render(content)
}

func (m model) renderFooter() string {
	status := lipgloss.NewStyle().Foreground(ColorClaudeSuccess).Bold(true).Render("● Ready")
	if viper.GetBool("ghost_mode") {
		status = lipgloss.NewStyle().Foreground(ColorClaudeWarning).Bold(true).Render("👻 Ghost Mode Active")
	}

	host := formatHost(m.llmEndpoint)
	info := fmt.Sprintf("%s @ %s", m.modelName, host)
	hints := StyleDim.Render("? Help  •  Tab Switch  •  q Quit")

	left := status
	right := fmt.Sprintf("%s   %s", StyleMuted.Render(info), hints)

	gapSize := m.width - lipgloss.Width(left) - lipgloss.Width(right) - 4
	if gapSize < 0 {
		gapSize = 0
	}

	footer := lipgloss.JoinHorizontal(lipgloss.Center,
		left,
		strings.Repeat(" ", gapSize),
		right,
	)

	return StyleStatus.Width(m.width).Render(footer)
}

func formatBool(b bool) string {
	if b {
		return lipgloss.NewStyle().Foreground(ColorClaudeSuccess).Bold(true).Render("[● ON]")
	}
	return StyleDim.Render("[○ OFF]")
}

func formatHost(urlStr string) string {
	if urlStr == "" {
		return "localhost"
	}
	s := strings.TrimPrefix(urlStr, "http://")
	s = strings.TrimPrefix(s, "https://")
	parts := strings.Split(s, "/")
	return parts[0]
}

func maskAPIKey(key string) string {
	if key == "" {
		return "[Not set / None]"
	}
	if len(key) <= 8 {
		return "••••••••"
	}
	return key[:3] + "••••" + key[len(key)-4:]
}

type ModelsFoundMsg []string
type GraphDataMsg string
type TimelineDataMsg string

type TestConnectionMsg struct {
	Success bool
	Message string
}

func testConnectionCmd() tea.Msg {
	success, _, msg, _ := analysis.TestLLMConnection()
	return TestConnectionMsg{
		Success: success,
		Message: msg,
	}
}

func fetchModelsCmd() tea.Msg {
	models, err := analysis.FetchAvailableModels()
	if err != nil {
		return AnalysisErrorMsg("Scan failed: " + err.Error())
	}
	return ModelsFoundMsg(models)
}

func fetchGraphDataCmd(caseID string) tea.Cmd {
	return func() tea.Msg {
		return GraphDataMsg(RenderASCIIGraph(caseID))
	}
}

func fetchTimelineDataCmd(caseID string) tea.Cmd {
	return func() tea.Msg {
		return TimelineDataMsg(RenderTimeline(caseID))
	}
}
