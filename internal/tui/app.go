package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/Gokul-Eswar/Spectre/internal/agent"
	"github.com/Gokul-Eswar/Spectre/internal/analysis"
	"github.com/Gokul-Eswar/Spectre/internal/core"
	"github.com/Gokul-Eswar/Spectre/internal/storage"
)

type model struct {
	width, height  int
	quitting       bool
	selectedCaseID string
	modelStatus    DynamicModelStatus
	engine         *agent.Engine
	viewport       viewport.Model
	input          textinput.Model
	spinner        spinner.Model
	isWorking      bool
	workingMessage string
	commandHistory []string
	historyIndex   int
}

// InitialModel initializes the modern horizontal TUI with dynamic model identification
func InitialModel() model {
	ti := textinput.New()
	ti.Prompt = StylePromptSymbol.Render("❯ ")
	ti.Placeholder = "Ask Spectre or type '?' for commands..."
	ti.Focus()
	ti.CharLimit = 1024

	vp := viewport.New(0, 0)
	welcomeMsg := lipgloss.NewStyle().Foreground(ColorClaudeSecondary).Render(
		"✻ Welcome to SPECTRE Intelligence Platform\n" +
			"Modern horizontal command console with local-first AI synthesis.\n\n" +
			"Type natural language to investigate, or type '?' for available commands.\n",
	)
	vp.SetContent(welcomeMsg)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorClaudeTerracotta)

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

		// Full-width horizontal responsive layout
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
			m.appendContent(lipgloss.NewStyle().Foreground(ColorClaudeError).Bold(true).Render("✖ Agent Error: ") + msg.err.Error())
		} else {
			m.appendContent(lipgloss.NewStyle().Foreground(ColorClaudeTerracotta).Bold(true).Render("✻ Spectre:\n") + msg.content)
		}
		return m, nil

	case analysisAsyncMsg:
		m.isWorking = false
		if msg.err != nil {
			m.appendContent(lipgloss.NewStyle().Foreground(ColorClaudeError).Bold(true).Render("✖ Analysis Error: ") + msg.err.Error())
		} else {
			m.appendContent(lipgloss.NewStyle().Foreground(ColorClaudeSuccess).Bold(true).Render("✔ Analysis Complete:\n") + FormatAnalysis(msg.result))
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
			m.appendContent(StylePromptSymbol.Render("❯ ") + lipgloss.NewStyle().Foreground(ColorClaudeText).Bold(true).Render(val))

			// Check for ? commands
			if strings.HasPrefix(val, "?") {
				if val == "?analyze" {
					if m.selectedCaseID == "" {
						m.appendContent(lipgloss.NewStyle().Foreground(ColorClaudeError).Render("○ No case selected. Use '?case <id>' first."))
						return m, nil
					}
					m.isWorking = true
					m.workingMessage = "Running AI analysis pipeline..."
					return m, func() tea.Msg {
						res, err := analysis.AnalyzeCase(m.selectedCaseID, m.modelStatus.Name)
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
				// Handle case context change
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
					m.appendContent(lipgloss.NewStyle().Foreground(ColorClaudeWarning).Render("○ Please select a case first with '?cases' before starting analysis."))
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
		Background(ColorClaudeSurface).
		Render(appName + strings.Repeat(" ", gap) + modelPill)
}

func (m model) renderInputBar() string {
	var sb strings.Builder

	// Dynamic activity indicator
	if m.isWorking {
		sb.WriteString(lipgloss.NewStyle().Foreground(ColorClaudeTerracotta).Render(m.spinner.View()+" "+m.workingMessage) + "\n")
	} else if strings.HasPrefix(m.input.Value(), "?") {
		// Responsive command suggestions strip
		sb.WriteString(StyleSuggestions.Render("[?]: ?cases  ?case <id>  ?model  ?analyze  ?evidence  ?graph  ?timeline  ?report  ?collect  ?help") + "\n")
	}

	inputBox := StyleInputBoxActive.Width(m.width - 4).Render(m.input.View())
	sb.WriteString(inputBox)

	return sb.String()
}

// formatHost extracts host and port from URL string
func formatHost(urlStr string) string {
	if urlStr == "" {
		return "localhost"
	}
	s := strings.TrimPrefix(urlStr, "http://")
	s = strings.TrimPrefix(s, "https://")
	parts := strings.Split(s, "/")
	return parts[0]
}

// maskAPIKey masks secret key for safe display
func maskAPIKey(key string) string {
	if key == "" {
		return "[Not set / None]"
	}
	if len(key) <= 8 {
		return "••••••••"
	}
	return key[:3] + "••••" + key[len(key)-4:]
}

type TestConnectionMsg struct {
	Success bool
	Message string
}
