package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/Gokul-Eswar/Spectre/internal/storage"
)

func RenderTimeline(caseID string) string {
	if caseID == "" {
		return "No case selected."
	}

	events, err := storage.GetCaseTimeline(caseID)
	if err != nil {
		return StyleCard.BorderForeground(ColorClaudeError).Render("Error fetching timeline: " + err.Error())
	}

	if len(events) == 0 {
		return StyleCard.Render("Timeline is empty. Collect some evidence first.")
	}

	var s strings.Builder
	s.WriteString(StyleTitle.Render(fmt.Sprintf("✦ TIMELINE — %s", caseID)) + "\n")
	s.WriteString(StyleDim.Render(strings.Repeat("─", 50)) + "\n\n")

	for i, e := range events {
		timeStr := lipgloss.NewStyle().Foreground(ColorClaudeTerracotta).Bold(true).Render(e.Timestamp.Format("15:04:05"))
		typeBadge := lipgloss.NewStyle().Foreground(ColorClaudePeach).Background(ColorClaudeSurfaceHigh).Padding(0, 1).Render(e.Type)
		desc := lipgloss.NewStyle().Foreground(ColorClaudeText).Render(e.Description)

		dot := lipgloss.NewStyle().Foreground(ColorClaudeTerracotta).Render("●")
		s.WriteString(fmt.Sprintf(" %s  %s  %s  %s\n", dot, timeStr, typeBadge, desc))

		if i < len(events)-1 {
			s.WriteString(StyleDim.Render(" │\n"))
		}
	}

	return StyleCard.Render(s.String())
}
