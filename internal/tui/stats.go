package tui

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/Gokul-Eswar/Spectre/internal/storage"
)

func GetSystemStats() string {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	cases, _ := storage.ListCases()

	var s strings.Builder
	s.WriteString(StyleTitle.Render("✻ SPECTRE SYSTEM TELEMETRY") + "\n")
	s.WriteString(StyleDim.Render(strings.Repeat("─", 45)) + "\n\n")

	s.WriteString(fmt.Sprintf("  %s %-16s : %s (%s)\n",
		lipgloss.NewStyle().Foreground(ColorClaudeTerracotta).Render("•"),
		StyleMuted.Render("Architecture"),
		lipgloss.NewStyle().Foreground(ColorClaudeText).Bold(true).Render(runtime.GOOS),
		runtime.GOARCH,
	))

	s.WriteString(fmt.Sprintf("  %s %-16s : %s\n",
		lipgloss.NewStyle().Foreground(ColorClaudeTerracotta).Render("•"),
		StyleMuted.Render("Allocated Memory"),
		lipgloss.NewStyle().Foreground(ColorClaudeText).Bold(true).Render(fmt.Sprintf("%v MB", m.Alloc/1024/1024)),
	))

	s.WriteString(fmt.Sprintf("  %s %-16s : %s\n",
		lipgloss.NewStyle().Foreground(ColorClaudeTerracotta).Render("•"),
		StyleMuted.Render("Active Cases"),
		lipgloss.NewStyle().Foreground(ColorClaudeSuccess).Bold(true).Render(fmt.Sprintf("%d", len(cases))),
	))

	s.WriteString(fmt.Sprintf("  %s %-16s : %s\n\n",
		lipgloss.NewStyle().Foreground(ColorClaudeTerracotta).Render("•"),
		StyleMuted.Render("Storage Backend"),
		lipgloss.NewStyle().Foreground(ColorClaudeText).Render("Local SQLite (WAL mode)"),
	))

	s.WriteString(StyleDim.Render("(esc: back)"))

	return StyleCard.Render(s.String())
}
