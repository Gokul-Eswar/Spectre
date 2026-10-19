package tui

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

var (
	// Claude Code Signature Color Palette
	ColorClaudeTerracotta  = lipgloss.Color("#D97757") // Anthropic / Claude warm terracotta
	ColorClaudePeach       = lipgloss.Color("#E8987E") // Soft warm peach highlight
	ColorClaudeAmber       = lipgloss.Color("#F59E0B") // Warm warning amber
	ColorClaudeDark        = lipgloss.Color("#18181B") // Dark graphite base (Zinc-900)
	ColorClaudeSurface     = lipgloss.Color("#202024") // Card / container surface
	ColorClaudeSurfaceHigh = lipgloss.Color("#27272A") // Elevated surface (Zinc-800)
	ColorClaudeBorder      = lipgloss.Color("#3F3F46") // Subtle border (Zinc-700)
	ColorClaudeBorderSubtle= lipgloss.Color("#2E2E33") // Very subtle separator
	ColorClaudeText        = lipgloss.Color("#F4F4F5") // Bright foreground (Zinc-100)
	ColorClaudeSecondary   = lipgloss.Color("#A1A1AA") // Muted body text (Zinc-400)
	ColorClaudeDim         = lipgloss.Color("#71717A") // Dim metadata (Zinc-500)
	ColorClaudeSuccess     = lipgloss.Color("#10B981") // Soft emerald green
	ColorClaudeWarning     = lipgloss.Color("#F59E0B") // Amber
	ColorClaudeError       = lipgloss.Color("#EF4444") // Soft coral red
	ColorClaudeInfo        = lipgloss.Color("#38BDF8") // Sky blue
	ColorClaudePurple      = lipgloss.Color("#A78BFA") // Soft violet / intelligence

	// Standard theme variable compatibility
	ColorPrimary = ColorClaudeText
	ColorAccent  = ColorClaudeTerracotta
	ColorMuted   = ColorClaudeSecondary
	ColorError   = ColorClaudeError
	ColorSuccess = ColorClaudeSuccess
	ColorBG      = ColorClaudeDark

	// Typography & Element Styles (Claude Code Style)
	StyleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorClaudeTerracotta)

	StyleHeader = lipgloss.NewStyle().
			Foreground(ColorClaudeText).
			Background(ColorClaudeSurface).
			Padding(0, 1)

	StyleNav = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, true, false, false).
			BorderForeground(ColorClaudeBorder).
			Padding(1, 1).
			Background(ColorClaudeDark)

	StyleMain = lipgloss.NewStyle().
			Padding(1, 2).
			Background(ColorClaudeDark)

	StyleStatus = lipgloss.NewStyle().
			Foreground(ColorClaudeSecondary).
			Background(ColorClaudeSurface).
			Border(lipgloss.NormalBorder(), true, false, false, false).
			BorderForeground(ColorClaudeBorder).
			Padding(0, 1)

	StyleSelectedNav = lipgloss.NewStyle().
				Foreground(ColorClaudeTerracotta).
				Bold(true)

	StyleMuted = lipgloss.NewStyle().
			Foreground(ColorClaudeSecondary)

	StyleDim = lipgloss.NewStyle().
			Foreground(ColorClaudeDim)

	// Claude Code Card & Container Styles
	StyleCard = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorClaudeBorder).
			Background(ColorClaudeSurface).
			Padding(1, 2)

	StyleCardFocused = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorClaudeTerracotta).
				Background(ColorClaudeSurface).
				Padding(1, 2)

	StyleInputBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorClaudeBorder).
			Background(ColorClaudeSurface).
			Padding(0, 1)

	StyleInputBoxFocused = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorClaudeTerracotta).
				Background(ColorClaudeSurface).
				Padding(0, 1)

	// Badges & Pills
	StylePillTerracotta = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(ColorClaudeTerracotta).
				Bold(true).
				Padding(0, 1)

	StylePillSubtle = lipgloss.NewStyle().
			Foreground(ColorClaudeSecondary).
			Background(ColorClaudeSurfaceHigh).
			Padding(0, 1)

	StylePillActive = lipgloss.NewStyle().
			Foreground(ColorClaudeSuccess).
			Background(ColorClaudeSurfaceHigh).
			Padding(0, 1)

	StyleBadge = lipgloss.NewStyle().
			Foreground(ColorClaudeTerracotta).
			Bold(true)
)

// Glyphs & Icons matching Claude Code CLI
const (
	IconClaude   = "✻"
	IconChevron  = "❯"
	IconDot      = "●"
	IconCheck    = "✔"
	IconCross    = "✖"
	IconWarning  = "▲"
	IconInfo     = "ℹ"
	IconDiamond  = "◆"
	IconSpark    = "✦"
	IconFolder   = "📂"
	IconBrain    = "🧠"
	IconEvidence = "📋"
	IconGraph    = "☍"
	IconTimeline = "⏱"
	IconReport   = "📑"
	IconChat     = "💬"
	IconSettings = "⚙"
	IconWeb      = "↗"
	IconRunner   = "⚡"
)

// NewClaudeListDelegate returns a customized bubbletea list.ItemDelegate styled in Claude Code aesthetic
func NewClaudeListDelegate() list.DefaultDelegate {
	d := list.NewDefaultDelegate()

	// Normal items
	d.Styles.NormalTitle = lipgloss.NewStyle().
		Foreground(ColorClaudeText).
		Padding(0, 0, 0, 2)

	d.Styles.NormalDesc = lipgloss.NewStyle().
		Foreground(ColorClaudeDim).
		Padding(0, 0, 0, 2)

	// Selected items with Claude terracotta indicator
	d.Styles.SelectedTitle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(ColorClaudeTerracotta).
		Foreground(ColorClaudeTerracotta).
		Bold(true).
		Padding(0, 0, 0, 1)

	d.Styles.SelectedDesc = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(ColorClaudeTerracotta).
		Foreground(ColorClaudeSecondary).
		Padding(0, 0, 0, 1)

	return d
}

// ClaudeTableStyles returns a modern table style matching Claude Code's clean typography
func ClaudeTableStyles() table.Styles {
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(ColorClaudeBorder).
		BorderBottom(true).
		Bold(true).
		Foreground(ColorClaudeSecondary)

	s.Selected = s.Selected.
		Foreground(ColorClaudeText).
		Background(lipgloss.Color("#2E2826")). // Subtle dark warm charcoal
		Bold(true)

	s.Cell = s.Cell.
		Foreground(ColorClaudeText)

	return s
}
