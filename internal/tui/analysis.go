package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/Gokul-Eswar/Spectre/internal/analysis"
	"github.com/Gokul-Eswar/Spectre/internal/core"
)

type AnalysisStepMsg int
type AnalysisFinishedMsg struct {
	Result *core.Analysis
}
type AnalysisErrorMsg string
type TickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(600*time.Millisecond, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func StartAnalysis(caseID string, modelName string) tea.Cmd {
	// Optimisation: Run analysis in parallel with the visual effects
	return tea.Batch(
		tickCmd(),
		PerformActualAnalysis(caseID, modelName),
	)
}

func PerformActualAnalysis(caseID string, modelName string) tea.Cmd {
	return func() tea.Msg {
		res, err := analysis.AnalyzeCase(caseID, modelName)
		if err != nil {
			return AnalysisErrorMsg(err.Error())
		}
		return AnalysisFinishedMsg{Result: res}
	}
}

func FormatAnalysis(res *core.Analysis) string {
	if res == nil {
		return ""
	}

	var s strings.Builder

	// Title Banner in Claude Code style
	title := StyleTitle.Render("✻ CLAUDE INVESTIGATION SYNTHESIS")
	s.WriteString(title + "\n")
	s.WriteString(StyleDim.Render(strings.Repeat("─", 55)) + "\n\n")

	// Findings
	headerFinding := lipgloss.NewStyle().Bold(true).Foreground(ColorClaudeTerracotta).Render("✦ PRELIMINARY FINDINGS")
	s.WriteString(headerFinding + "\n")
	for _, f := range res.Findings {
		bullet := lipgloss.NewStyle().Foreground(ColorClaudeTerracotta).Render("•")
		text := lipgloss.NewStyle().Foreground(ColorClaudeText).Render(f)
		s.WriteString(fmt.Sprintf("  %s %s\n", bullet, text))
	}

	// Risks
	s.WriteString("\n")
	headerRisk := lipgloss.NewStyle().Bold(true).Foreground(ColorClaudeWarning).Render("▲ IDENTIFIED RISKS")
	s.WriteString(headerRisk + "\n")
	for _, r := range res.Risks {
		warn := lipgloss.NewStyle().Foreground(ColorClaudeWarning).Render("⚠")
		text := lipgloss.NewStyle().Foreground(ColorClaudeText).Render(r)
		s.WriteString(fmt.Sprintf("  %s %s\n", warn, text))
	}

	// Next Steps
	s.WriteString("\n")
	headerSteps := lipgloss.NewStyle().Bold(true).Foreground(ColorClaudeInfo).Render("→ RECOMMENDED NEXT STEPS")
	s.WriteString(headerSteps + "\n")
	for _, n := range res.NextSteps {
		arrow := lipgloss.NewStyle().Foreground(ColorClaudeInfo).Render("→")
		text := lipgloss.NewStyle().Foreground(ColorClaudeText).Render(n)
		s.WriteString(fmt.Sprintf("  %s %s\n", arrow, text))
	}

	if len(res.MissingData) > 0 {
		s.WriteString("\n")
		headerMissing := lipgloss.NewStyle().Bold(true).Foreground(ColorClaudeSecondary).Render("? MISSING DATA")
		s.WriteString(headerMissing + "\n")
		for _, m := range res.MissingData {
			question := StyleDim.Render("?")
			text := StyleMuted.Render(m)
			s.WriteString(fmt.Sprintf("  %s %s\n", question, text))
		}
	}

	if len(res.SuggestedCollectors) > 0 {
		s.WriteString("\n")
		headerColl := lipgloss.NewStyle().Bold(true).Foreground(ColorClaudeSuccess).Render("+ SUGGESTED COLLECTORS")
		s.WriteString(headerColl + "\n")
		for _, sc := range res.SuggestedCollectors {
			plus := lipgloss.NewStyle().Foreground(ColorClaudeSuccess).Render("+")
			text := lipgloss.NewStyle().Foreground(ColorClaudeText).Render(sc)
			s.WriteString(fmt.Sprintf("  %s %s\n", plus, text))
		}
	}

	// Meter & Confidence Level
	filled := int(res.Confidence * 10)
	if filled > 10 {
		filled = 10
	} else if filled < 0 {
		filled = 0
	}
	empty := 10 - filled
	bar := lipgloss.NewStyle().Foreground(ColorClaudeTerracotta).Render(strings.Repeat("█", filled)) +
		StyleDim.Render(strings.Repeat("░", empty))

	s.WriteString(fmt.Sprintf("\n%s: %.2f  [%s] (%d%%)",
		lipgloss.NewStyle().Bold(true).Foreground(ColorClaudeText).Render("Confidence Level"),
		res.Confidence,
		bar,
		int(res.Confidence*100),
	))

	return StyleCard.Render(s.String())
}
