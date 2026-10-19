package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/Gokul-Eswar/Spectre/internal/storage"
)

func RenderASCIIGraph(caseID string) string {
	if caseID == "" {
		return "No case selected."
	}

	entities, err := storage.ListEntitiesByCase(caseID)
	if err != nil {
		return StyleCard.BorderForeground(ColorClaudeError).Render("Error fetching entities: " + err.Error())
	}

	relationships, err := storage.ListRelationshipsByCase(caseID)
	if err != nil {
		return StyleCard.BorderForeground(ColorClaudeError).Render("Error fetching relationships: " + err.Error())
	}

	if len(entities) == 0 {
		return StyleCard.Render("No entities found for this case. Run collectors to discover intelligence.")
	}

	var s strings.Builder
	s.WriteString(StyleTitle.Render(fmt.Sprintf("✦ INTELLIGENCE GRAPH — %s", caseID)) + "\n")
	s.WriteString(StyleDim.Render(strings.Repeat("─", 50)) + "\n\n")

	// Map to look up entity values by ID
	entityMap := make(map[string]string)
	entityTypeMap := make(map[string]string)
	for _, e := range entities {
		entityMap[e.ID] = e.Value
		entityTypeMap[e.ID] = e.Type
	}

	// Simple representation: list relationships as connected nodes
	if len(relationships) == 0 {
		s.WriteString(StyleMuted.Render("No relationships identified yet. Run more collectors.\n\n"))
		s.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ColorClaudeText).Render("Entities:") + "\n")
		for _, e := range entities {
			badge := lipgloss.NewStyle().Foreground(ColorClaudeTerracotta).Background(ColorClaudeSurfaceHigh).Padding(0, 1).Render(e.Type)
			val := lipgloss.NewStyle().Foreground(ColorClaudeText).Render(e.Value)
			s.WriteString(fmt.Sprintf("  • %s %s\n", badge, val))
		}
	} else {
		for _, r := range relationships {
			from := entityMap[r.FromEntityID]
			fromType := entityTypeMap[r.FromEntityID]
			if from == "" {
				from = r.FromEntityID
				fromType = "ENTITY"
			}
			to := entityMap[r.ToEntityID]
			toType := entityTypeMap[r.ToEntityID]
			if to == "" {
				to = r.ToEntityID
				toType = "ENTITY"
			}

			fromNode := fmt.Sprintf("[%s: %s]", fromType, from)
			toNode := fmt.Sprintf("[%s: %s]", toType, to)
			relStr := lipgloss.NewStyle().Foreground(ColorClaudeTerracotta).Render(fmt.Sprintf("──(%s)──>", r.Type))

			s.WriteString(fmt.Sprintf(" %s %s %s\n",
				lipgloss.NewStyle().Foreground(ColorClaudeText).Bold(true).Render(fromNode),
				relStr,
				lipgloss.NewStyle().Foreground(ColorClaudePeach).Bold(true).Render(toNode),
			))
		}
	}

	return StyleCard.Render(s.String())
}
