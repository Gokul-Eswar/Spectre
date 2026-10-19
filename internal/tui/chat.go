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
	"github.com/Gokul-Eswar/Spectre/internal/analyzer"
)

type chatModel struct {
	viewport      viewport.Model
	textinput     textinput.Model
	spinner       spinner.Model
	thinking      bool
	messages      []analyzer.Message
	caseID        string
	engine        *agent.Engine
	width, height int
}

func newChatModel() chatModel {
	ti := textinput.New()
	ti.Prompt = "❯ "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(ColorClaudeTerracotta).Bold(true)
	ti.Placeholder = "Ask Claude Code / Spectre anything, run recon, or type /help..."
	ti.Focus()
	ti.CharLimit = 512
	ti.Width = 50

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorClaudeTerracotta)

	vp := viewport.New(0, 0)
	welcome := lipgloss.NewStyle().Foreground(ColorClaudeSecondary).Render(
		"✻ Welcome to SPECTRE Investigation Chat\n" +
			"Powered by Claude Code autonomous intelligence architecture.\n\n" +
			"Select a case from [1] Cases or type a question below to begin.\n" +
			"Type '/help' for investigation workflows and slash commands.\n",
	)
	vp.SetContent(welcome)

	return chatModel{
		textinput: ti,
		viewport:  vp,
		spinner:   s,
		messages:  []analyzer.Message{},
	}
}

func (m *chatModel) setCase(caseID string) {
	if m.caseID == caseID {
		return
	}
	m.caseID = caseID
	m.engine = agent.NewEngine(caseID)
	m.messages = []analyzer.Message{}

	banner := lipgloss.NewStyle().Foreground(ColorClaudeTerracotta).Bold(true).Render(fmt.Sprintf("✻ Agent ready for case: %s\n", caseID)) +
		StyleDim.Render("Type a message to begin autonomous investigation.\n")
	m.viewport.SetContent(banner)
}

type agentResponseMsg struct {
	content string
	err     error
}

func (m chatModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m chatModel) Update(msg tea.Msg) (chatModel, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
		sCmd  tea.Cmd
	)

	m.textinput, tiCmd = m.textinput.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	if m.thinking {
		m.spinner, sCmd = m.spinner.Update(msg)
	}

	switch msg := msg.(type) {
	case agentResponseMsg:
		m.thinking = false
		if msg.err != nil {
			m.appendMessage("system", "Error: "+msg.err.Error())
		} else {
			m.appendMessage("assistant", msg.content)
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if m.thinking || strings.TrimSpace(m.textinput.Value()) == "" {
				return m, nil
			}

			input := strings.TrimSpace(m.textinput.Value())
			m.appendMessage("user", input)
			m.textinput.SetValue("")
			m.thinking = true

			return m, tea.Batch(
				m.spinner.Tick,
				func() tea.Msg {
					resp, err := m.engine.Execute(input)
					return agentResponseMsg{resp, err}
				},
			)
		}
	}

	return m, tea.Batch(tiCmd, vpCmd, sCmd)
}

func (m *chatModel) appendMessage(role, content string) {
	m.messages = append(m.messages, analyzer.Message{Role: role, Content: content})

	var sb strings.Builder
	for _, msg := range m.messages {
		switch msg.Role {
		case "user":
			tag := lipgloss.NewStyle().Foreground(ColorClaudeText).Bold(true).Render("❯ You")
			sb.WriteString(tag + "\n")
			sb.WriteString(lipgloss.NewStyle().Foreground(ColorClaudeText).Render(formatIndented(msg.Content)))
			sb.WriteString("\n\n")
		case "assistant":
			tag := lipgloss.NewStyle().Foreground(ColorClaudeTerracotta).Bold(true).Render("✻ Spectre")
			sb.WriteString(tag + "\n")
			sb.WriteString(lipgloss.NewStyle().Foreground(ColorClaudeSecondary).Render(formatIndented(msg.Content)))
			sb.WriteString("\n\n")
		case "system":
			tag := lipgloss.NewStyle().Foreground(ColorClaudeDim).Bold(true).Render("ℹ System")
			sb.WriteString(tag + "\n")
			sb.WriteString(lipgloss.NewStyle().Foreground(ColorClaudeDim).Render(formatIndented(msg.Content)))
			sb.WriteString("\n\n")
		}
	}
	m.viewport.SetContent(sb.String())
	m.viewport.GotoBottom()
}

func formatIndented(text string) string {
	lines := strings.Split(text, "\n")
	for i, l := range lines {
		lines[i] = "  " + l
	}
	return strings.Join(lines, "\n")
}

func (m chatModel) View() string {
	spin := ""
	if m.thinking {
		spin = lipgloss.NewStyle().Foreground(ColorClaudeTerracotta).Render(m.spinner.View()+" Thinking...") + "\n"
	}

	boxStyle := StyleInputBoxFocused.Width(m.width - 2)
	inputView := boxStyle.Render(m.textinput.View())
	hints := StyleDim.Render("Enter: Send  •  Tab: Switch Pane  •  /help: Guide")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.viewport.View(),
		spin,
		inputView,
		hints,
	)
}

func (m *chatModel) setSize(w, h int) {
	m.width = w
	m.height = h
	m.viewport.Width = w
	m.viewport.Height = h - 6
	m.textinput.Width = w - 6
}
