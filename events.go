package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

type sysEvent struct {
	ts    time.Time
	msg   string
	level string // "info", "warn", "crit"
}

func (m *model) addEvent(level, msg string) {
	m.events = append(m.events, sysEvent{ts: time.Now(), msg: msg, level: level})
	if len(m.events) > 20 {
		m.events = m.events[len(m.events)-20:]
	}
}

func (m model) viewEvents(colW int) string {
	if len(m.events) == 0 {
		return "  " + dimStyle.Render("no events yet")
	}
	var b strings.Builder
	// Show last 5, newest first.
	n := 5
	if len(m.events) < n {
		n = len(m.events)
	}
	for i := len(m.events) - 1; i >= len(m.events)-n; i-- {
		e := m.events[i]
		ts := dimStyle.Render(e.ts.Format("15:04:05"))
		var msgStyle lipgloss.Style
		switch e.level {
		case "crit":
			msgStyle = redStyle
		case "warn":
			msgStyle = yellowStyle
		default:
			msgStyle = dimStyle
		}
		maxMsgW := colW - 12
		if maxMsgW < 1 {
			maxMsgW = 1
		}
		b.WriteString(fmt.Sprintf("  %s  %s\n", ts, msgStyle.Render(truncate(e.msg, maxMsgW))))
	}
	return strings.TrimRight(b.String(), "\n")
}
