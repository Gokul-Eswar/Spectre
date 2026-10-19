package tui

import (
	"github.com/charmbracelet/bubbles/table"
	"github.com/Gokul-Eswar/Spectre/internal/storage"
)

// FetchEntities fetches all entities for a case and converts them to table rows
func FetchEntities(caseID string) ([]table.Row, error) {
	entities, err := storage.ListEntitiesByCase(caseID)
	if err != nil {
		return nil, err
	}

	var rows []table.Row
	for _, e := range entities {
		rows = append(rows, table.Row{
			e.Type,
			e.Value,
			e.Source,
		})
	}
	return rows, nil
}

func NewEntityTable() table.Model {
	columns := []table.Column{
		{Title: "TYPE", Width: 12},
		{Title: "VALUE", Width: 36},
		{Title: "SOURCE", Width: 18},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(12),
	)

	t.SetStyles(ClaudeTableStyles())
	return t
}
