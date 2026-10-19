package tui

import (
	"github.com/charmbracelet/bubbles/table"
	"github.com/Gokul-Eswar/Spectre/internal/storage"
)

func FetchRelationships(caseID string) ([]table.Row, error) {
	rels, err := storage.ListRelationshipsByCase(caseID)
	if err != nil {
		return nil, err
	}

	// We need a map for entity values to make the table readable
	entities, _ := storage.ListEntitiesByCase(caseID)
	entMap := make(map[string]string)
	for _, e := range entities {
		entMap[e.ID] = e.Value
	}

	var rows []table.Row
	for _, r := range rels {
		from := entMap[r.FromEntityID]
		if from == "" {
			from = r.FromEntityID
		}
		to := entMap[r.ToEntityID]
		if to == "" {
			to = r.ToEntityID
		}

		rows = append(rows, table.Row{
			from,
			r.Type,
			to,
		})
	}
	return rows, nil
}

func NewRelationshipTable() table.Model {
	columns := []table.Column{
		{Title: "SOURCE ENTITY", Width: 26},
		{Title: "RELATIONSHIP", Width: 18},
		{Title: "TARGET ENTITY", Width: 26},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(12),
	)

	t.SetStyles(ClaudeTableStyles())
	return t
}
