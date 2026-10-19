package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/Gokul-Eswar/Spectre/internal/collector"
)

type runnerState int

const (
	selectCase runnerState = iota
	selectCollector
	inputTarget
	inputOptions
	executing
	result
)

type runnerModel struct {
	state          runnerState
	caseList       list.Model
	collList       list.Model
	textInput      textinput.Model
	optionsInput   textinput.Model
	selectedCaseID string
	selectedColl   string
	activeAllowed  bool
	err            error
	message        string
}

func NewRunnerModel() runnerModel {
	ti := textinput.New()
	ti.Prompt = "❯ "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(ColorClaudeTerracotta).Bold(true)
	ti.Placeholder = "example.com"
	ti.Focus()

	oi := textinput.New()
	oi.Prompt = "❯ "
	oi.PromptStyle = lipgloss.NewStyle().Foreground(ColorClaudeTerracotta).Bold(true)
	oi.Placeholder = "80, 443, 8080 (optional)"

	// Dynamic Collector list with Claude Code Delegate
	var collectors []list.Item
	for _, c := range collector.List() {
		collectors = append(collectors, item{
			title: c.Name(),
			desc:  c.Description(),
		})
	}

	cl := list.New(collectors, NewClaudeListDelegate(), 0, 0)
	cl.Title = "Select Collector (Space to toggle Active Mode) [SAFE]"
	cl.Styles.Title = lipgloss.NewStyle().Foreground(ColorClaudeTerracotta).Bold(true)
	cl.SetShowHelp(false)

	// Case list with Claude Code Delegate
	l := list.New([]list.Item{}, NewClaudeListDelegate(), 0, 0)
	l.Title = "Select Case for Collection"
	l.Styles.Title = lipgloss.NewStyle().Foreground(ColorClaudeTerracotta).Bold(true)
	l.SetShowHelp(false)

	return runnerModel{
		state:        selectCase,
		caseList:     l,
		collList:     cl,
		textInput:    ti,
		optionsInput: oi,
	}
}

func (m runnerModel) Update(msg tea.Msg) (runnerModel, tea.Cmd) {
	var cmd tea.Cmd

	switch m.state {
	case selectCase:
		if km, ok := msg.(tea.KeyMsg); ok && km.String() == "enter" {
			if i := m.caseList.SelectedItem(); i != nil {
				m.selectedCaseID = i.(item).id
				m.state = selectCollector
				return m, nil
			}
		}
		m.caseList, cmd = m.caseList.Update(msg)
		return m, cmd

	case selectCollector:
		if km, ok := msg.(tea.KeyMsg); ok {
			switch km.String() {
			case "enter":
				if i := m.collList.SelectedItem(); i != nil {
					m.selectedColl = i.(item).title
					m.state = inputTarget
					return m, nil
				}
			case " ":
				m.activeAllowed = !m.activeAllowed
				status := "[SAFE]"
				if m.activeAllowed {
					status = "[ACTIVE/DANGEROUS]"
				}
				m.collList.Title = fmt.Sprintf("Select Collector (Space to toggle) %s", status)
				return m, nil
			}
		}
		m.collList, cmd = m.collList.Update(msg)
		return m, cmd

	case inputTarget:
		if km, ok := msg.(tea.KeyMsg); ok && km.String() == "enter" {
			target := strings.TrimSpace(m.textInput.Value())
			if target == "" {
				return m, nil
			}

			if m.selectedColl == "ports" {
				m.state = inputOptions
				m.optionsInput.Focus()
				return m, nil
			}

			m.state = executing
			return m, func() tea.Msg {
				_, err := collector.RunAndSave(m.selectedColl, m.selectedCaseID, target, m.activeAllowed, nil)
				if err != nil {
					return err
				}
				return "Collection complete!"
			}
		}
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd

	case inputOptions:
		if km, ok := msg.(tea.KeyMsg); ok && km.String() == "enter" {
			target := strings.TrimSpace(m.textInput.Value())
			optionsStr := strings.TrimSpace(m.optionsInput.Value())

			var options map[string]interface{}
			if optionsStr != "" {
				parts := strings.Split(optionsStr, ",")
				var ports []int
				for _, p := range parts {
					p = strings.TrimSpace(p)
					var port int
					fmt.Sscanf(p, "%d", &port)
					if port > 0 {
						ports = append(ports, port)
					}
				}
				if len(ports) > 0 {
					options = map[string]interface{}{"ports": ports}
				}
			}

			m.state = executing
			return m, func() tea.Msg {
				_, err := collector.RunAndSave(m.selectedColl, m.selectedCaseID, target, m.activeAllowed, options)
				if err != nil {
					return err
				}
				return "Collection complete!"
			}
		}
		m.optionsInput, cmd = m.optionsInput.Update(msg)
		return m, cmd

	case result, executing:
		if km, ok := msg.(tea.KeyMsg); ok && km.String() == "enter" {
			m.state = selectCase
			m.message = ""
			m.err = nil
			return m, nil
		}
	}

	switch msg := msg.(type) {
	case error:
		m.err = msg
		m.state = result
	case string:
		m.message = msg
		m.state = result
	}

	return m, nil
}

func (m runnerModel) View() string {
	switch m.state {
	case selectCase:
		return m.caseList.View()
	case selectCollector:
		return m.collList.View()
	case inputTarget:
		statusBadge := lipgloss.NewStyle().Foreground(ColorClaudeSuccess).Bold(true).Render("[● SAFE PASSIVE]")
		if m.activeAllowed {
			statusBadge = lipgloss.NewStyle().Foreground(ColorClaudeWarning).Bold(true).Render("[▲ ACTIVE/DANGEROUS]")
		}
		box := StyleCardFocused.Width(62).Render(fmt.Sprintf(
			"%s\n%s\n\nCollector : %s\nCase      : %s\nMode      : %s\n\nTarget Specification:\n%s\n\n%s",
			StyleTitle.Render("✻ RECONNAISSANCE TARGET"),
			StyleDim.Render(strings.Repeat("─", 45)),
			lipgloss.NewStyle().Foreground(ColorClaudeTerracotta).Bold(true).Render(m.selectedColl),
			m.selectedCaseID,
			statusBadge,
			m.textInput.View(),
			StyleDim.Render("[Enter: Next  •  Esc: Cancel]"),
		))
		return box

	case inputOptions:
		box := StyleCardFocused.Width(62).Render(fmt.Sprintf(
			"%s\n%s\n\nCollector : %s\nTarget    : %s\n\nEnter custom ports (comma separated):\n%s\n\n%s",
			StyleTitle.Render("✻ PROBE CONFIGURATION"),
			StyleDim.Render(strings.Repeat("─", 45)),
			lipgloss.NewStyle().Foreground(ColorClaudeTerracotta).Bold(true).Render(m.selectedColl),
			lipgloss.NewStyle().Foreground(ColorClaudeText).Bold(true).Render(m.textInput.Value()),
			m.optionsInput.View(),
			StyleDim.Render("(leave empty for defaults  •  Enter: Run  •  Esc: Cancel)"),
		))
		return box

	case executing:
		box := StyleCard.Width(62).Render(fmt.Sprintf(
			"%s\n%s\n\nRunning %s against %s...\n\n%s",
			StyleTitle.Render("✻ EXECUTING COLLECTOR"),
			StyleDim.Render(strings.Repeat("─", 45)),
			lipgloss.NewStyle().Foreground(ColorClaudeTerracotta).Bold(true).Render(m.selectedColl),
			lipgloss.NewStyle().Foreground(ColorClaudeText).Bold(true).Render(m.textInput.Value()),
			StyleDim.Render("Please wait while evidence is being collected..."),
		))
		return box

	case result:
		if m.err != nil {
			return StyleCard.BorderForeground(ColorClaudeError).Width(62).Render(fmt.Sprintf(
				"%s\n%s\n\nError: %v\n\n%s",
				lipgloss.NewStyle().Foreground(ColorClaudeError).Bold(true).Render("✖ COLLECTION FAILED"),
				StyleDim.Render(strings.Repeat("─", 45)),
				m.err,
				StyleDim.Render("(enter: continue)"),
			))
		}
		return StyleCard.BorderForeground(ColorClaudeSuccess).Width(62).Render(fmt.Sprintf(
			"%s\n%s\n\n%s\n\n%s",
			lipgloss.NewStyle().Foreground(ColorClaudeSuccess).Bold(true).Render("✔ COLLECTION SUCCESS"),
			StyleDim.Render(strings.Repeat("─", 45)),
			m.message,
			StyleDim.Render("(enter: continue)"),
		))
	}
	return ""
}
