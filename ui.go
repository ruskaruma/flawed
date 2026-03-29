package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Consistent palette
	borderColor = lipgloss.Color("#3B4F6B")
	titleColor  = lipgloss.Color("#56B6C2") // cyan
	valueColor  = lipgloss.Color("#FFFFFF")
	labelColor  = lipgloss.Color("#6C7086") // dim gray
	alertColor  = lipgloss.Color("#FF5555")

	// Panel border style
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Padding(1, 2)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#000000")).
			Background(titleColor).
			Padding(0, 1)

	alertTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(alertColor).
			Padding(0, 1)

	sectionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(titleColor)

	alertStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(alertColor).
			Background(lipgloss.Color("#3B1010"))

	dimStyle = lipgloss.NewStyle().
			Foreground(labelColor)

	greenStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#50FA7B"))

	yellowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F1FA8C"))

	redStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5555"))

	headerStyle = lipgloss.NewStyle().
			Foreground(titleColor).
			Bold(true)

	labelStyle = lipgloss.NewStyle().
			Foreground(labelColor)

	valueStyle = lipgloss.NewStyle().
			Foreground(valueColor)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#BBBBBB")).
			Background(lipgloss.Color("#1E2030")).
			Padding(0, 1)

	statusKeyStyle = lipgloss.NewStyle().
			Foreground(titleColor).
			Background(lipgloss.Color("#1E2030")).
			Bold(true)

	rowEvenStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#1A1B2E"))

	rowOddStyle = lipgloss.NewStyle()

	// Cached style for the process selector marker
	procMarkerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(titleColor)
)

func formatBytes(b uint64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)
	switch {
	case b >= gb:
		return fmt.Sprintf("%.1fG", float64(b)/float64(gb))
	case b >= mb:
		return fmt.Sprintf("%.1fM", float64(b)/float64(mb))
	case b >= kb:
		return fmt.Sprintf("%.1fK", float64(b)/float64(kb))
	default:
		return fmt.Sprintf("%dB", b)
	}
}

func formatBytesPerSec(b uint64) string {
	return formatBytes(b) + "/s"
}

func pctColor(pct float64) lipgloss.Style {
	switch {
	case pct > 80:
		return redStyle
	case pct >= 50:
		return yellowStyle
	default:
		return greenStyle
	}
}

// bar renders a gradient progress bar: █▓▒░ for filled, dim dots for empty.
func bar(pct float64, width int) string {
	filled := int(pct / 100 * float64(width))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	empty := width - filled

	style := pctColor(pct)

	// Gradient: first 70% solid, next 20% medium, last 10% light
	solid := filled * 7 / 10
	med := filled * 2 / 10
	light := filled - solid - med

	return style.Render(strings.Repeat("█", solid)) +
		style.Render(strings.Repeat("▓", med)) +
		style.Render(strings.Repeat("▒", light)) +
		dimStyle.Render(strings.Repeat("░", empty))
}

func rightPad(s string, n int) string {
	if len(s) >= n {
		return s[:n]
	}
	return s + strings.Repeat(" ", n-len(s))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 3 {
		return s[:n]
	}
	return s[:n-3] + "..."
}

// panel wraps content in a bordered box with a title.
func panel(title, content string, width int) string {
	s := panelStyle.Width(width - 2) // account for border
	header := sectionStyle.Render("▎" + title)
	return s.Render(header + "\n" + content)
}
