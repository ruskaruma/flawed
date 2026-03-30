package main

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var taglines = []string{
	"all systems nominal.",
	"watching your machine so you don't have to.",
	"no segfaults today. probably.",
	"uptime is a lifestyle.",
	"may your load averages be low.",
	"have you tried turning it off and on again?",
	"sudo make me a sandwich.",
	"it works on my machine.",
	"99 little bugs in the code...",
	"there's no place like 127.0.0.1.",
}

var sessionTagline = taglines[rand.Intn(len(taglines))]

// Tasks the ladybug cycles through while "working"
var bugTasks = []string{
	"checking cpu temps",
	"scanning memory",
	"inspecting disk i/o",
	"monitoring network",
	"counting processes",
	"sniffing packets",
	"defragmenting vibes",
	"calibrating uptime",
	"polishing metrics",
	"optimizing loops",
}

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func greeterHostname() string {
	h, err := os.Hostname()
	if err != nil {
		return "localhost"
	}
	return h
}

func greeting(hour int) string {
	switch {
	case hour >= 5 && hour < 12:
		return "Good morning"
	case hour >= 12 && hour < 17:
		return "Good afternoon"
	case hour >= 17 && hour < 21:
		return "Good evening"
	default:
		return "Good night"
	}
}

// px is a single half-block pixel cell.
type px struct {
	ch  string
	fg  [3]int
	bg  [3]int
	vis bool
}

// Color palette for the ladybug
var (
	pxNone = [3]int{}
	pxBK   = [3]int{20, 20, 20}    // black — head/body
	pxAN   = [3]int{50, 50, 55}    // antennae/legs — slightly visible
	pxRD   = [3]int{220, 50, 50}   // red shell
	pxDK   = [3]int{170, 30, 30}   // dark red edge
	pxSP   = [3]int{15, 15, 15}    // spots
	pxLN   = [3]int{25, 25, 25}    // center line
	pxWH   = [3]int{230, 230, 240} // white eyes
	pxHL   = [3]int{240, 80, 70}   // shell highlight
)

func pxT() px              { return px{" ", pxNone, pxNone, false} }
func pxU(fg [3]int) px     { return px{"▀", fg, pxNone, true} }
func pxD(fg [3]int) px     { return px{"▄", fg, pxNone, true} }
func pxF(fg, bg [3]int) px { return px{"▀", fg, bg, true} }

// buildLadybugFrame builds the pixel grid for one animation frame.
func buildLadybugFrame(wiggle int) [][]px {
	t, u, d, f := pxT, pxU, pxD, pxF
	BK, AN, RD, DK, SP, LN, WH, HL := pxBK, pxAN, pxRD, pxDK, pxSP, pxLN, pxWH, pxHL

	var antennaRow []px
	if wiggle == 0 {
		antennaRow = []px{t(), t(), t(), t(), t(), u(AN), t(), d(AN), t(), t(), t(), t(), d(AN), t(), u(AN), t(), t(), t(), t(), t()}
	} else {
		antennaRow = []px{t(), t(), t(), t(), t(), t(), u(AN), d(AN), t(), t(), t(), t(), d(AN), u(AN), t(), t(), t(), t(), t(), t()}
	}

	var shellRow3, shellRow5 []px
	if wiggle == 0 {
		shellRow3 = []px{t(), t(), t(), d(AN), d(DK), f(DK, RD), f(HL, RD), f(RD, SP), f(RD, RD), f(LN, RD), f(LN, RD), f(RD, RD), f(RD, SP), f(RD, RD), f(DK, RD), d(DK), d(AN), t(), t(), t()}
		shellRow5 = []px{t(), u(AN), d(AN), f(DK, DK), f(RD, RD), f(RD, RD), f(RD, RD), f(RD, RD), f(SP, RD), f(LN, RD), f(LN, RD), f(SP, RD), f(RD, RD), f(RD, RD), f(RD, RD), f(RD, RD), f(DK, DK), d(AN), u(AN), t()}
	} else {
		shellRow3 = []px{t(), t(), t(), t(), d(DK), f(DK, RD), f(HL, RD), f(RD, SP), f(RD, RD), f(LN, RD), f(LN, RD), f(RD, RD), f(RD, SP), f(RD, RD), f(DK, RD), d(DK), t(), t(), t(), t()}
		shellRow5 = []px{t(), t(), u(AN), f(DK, DK), f(RD, RD), f(RD, RD), f(RD, RD), f(RD, RD), f(SP, RD), f(LN, RD), f(LN, RD), f(SP, RD), f(RD, RD), f(RD, RD), f(RD, RD), f(RD, RD), f(DK, DK), u(AN), t(), t()}
	}

	return [][]px{
		antennaRow,
		{t(), t(), t(), t(), t(), t(), t(), d(BK), f(BK, BK), f(BK, BK), f(BK, BK), f(BK, BK), d(BK), t(), t(), t(), t(), t(), t(), t()},
		{t(), t(), t(), t(), t(), t(), u(BK), f(BK, BK), f(WH, BK), f(BK, BK), f(BK, BK), f(WH, BK), f(BK, BK), u(BK), t(), t(), t(), t(), t(), t()},
		shellRow3,
		{t(), t(), u(AN), d(DK), f(DK, RD), f(HL, RD), f(RD, SP), f(RD, RD), f(RD, RD), f(LN, RD), f(LN, RD), f(RD, RD), f(RD, RD), f(RD, SP), f(RD, RD), f(DK, RD), d(DK), u(AN), t(), t()},
		shellRow5,
		{t(), t(), t(), u(DK), f(RD, DK), f(SP, RD), f(RD, RD), f(RD, RD), f(RD, RD), f(LN, RD), f(LN, RD), f(RD, RD), f(RD, RD), f(RD, RD), f(SP, RD), f(RD, DK), u(DK), t(), t(), t()},
		{t(), t(), t(), t(), t(), u(DK), f(RD, DK), f(RD, RD), f(SP, RD), f(RD, RD), f(RD, RD), f(SP, RD), f(RD, RD), f(RD, DK), u(DK), t(), t(), t(), t(), t()},
		{t(), t(), t(), t(), t(), t(), t(), u(DK), f(RD, DK), f(RD, DK), f(RD, DK), f(RD, DK), u(DK), t(), t(), t(), t(), t(), t(), t()},
		{t(), t(), t(), t(), t(), t(), t(), t(), t(), u(BK), u(BK), t(), t(), t(), t(), t(), t(), t(), t(), t()},
	}
}

// renderPixelRows renders pixel art using raw ANSI escapes for performance.
// No lipgloss.NewStyle() allocation per pixel — just direct escape codes.
func renderPixelRows(rows [][]px) string {
	var sb strings.Builder
	sb.Grow(len(rows) * 20 * 30) // pre-allocate
	for i, row := range rows {
		if i > 0 {
			sb.WriteString("\n")
		}
		for _, p := range row {
			if !p.vis {
				sb.WriteString(" ")
				continue
			}
			fmt.Fprintf(&sb, "\x1b[38;2;%d;%d;%dm", p.fg[0], p.fg[1], p.fg[2])
			if p.bg != pxNone {
				fmt.Fprintf(&sb, "\x1b[48;2;%d;%d;%dm", p.bg[0], p.bg[1], p.bg[2])
			}
			sb.WriteString(p.ch)
			sb.WriteString("\x1b[0m")
		}
	}
	return sb.String()
}

// Pre-rendered frames at startup — only 2 frames, no need to rebuild each tick.
var (
	ladybugFrame0 = renderPixelRows(buildLadybugFrame(0))
	ladybugFrame1 = renderPixelRows(buildLadybugFrame(1))
)

func viewGreeter(width int, frame int) string {
	now := time.Now()
	hour := now.Hour()

	// Pick pre-rendered frame — wiggle every frame
	art := ladybugFrame0
	if frame%2 == 1 {
		art = ladybugFrame1
	}

	// Walking offset — walk advances every 3rd frame (slower than wiggle)
	bugWidth := 20
	maxOffset := width - bugWidth
	if maxOffset < 0 {
		maxOffset = 0
	}
	cycle := maxOffset * 2
	if cycle == 0 {
		cycle = 1
	}
	walkFrame := (frame / 3) % cycle
	var offset int
	if walkFrame < maxOffset {
		offset = walkFrame
	} else {
		offset = cycle - walkFrame
	}

	// Apply walking offset
	artLines := strings.Split(art, "\n")
	var centered strings.Builder
	pad := strings.Repeat(" ", offset)
	for i, line := range artLines {
		if i > 0 {
			centered.WriteString("\n")
		}
		centered.WriteString(pad)
		centered.WriteString(line)
	}

	// Animated task line with spinner
	spinner := spinnerFrames[frame%len(spinnerFrames)]
	task := bugTasks[(frame/10)%len(bugTasks)]
	dots := strings.Repeat(".", (frame%3)+1) + strings.Repeat(" ", 3-(frame%3)-1)
	taskLine := lipgloss.NewStyle().Foreground(titleColor).Render(spinner) + " " +
		dimStyle.Render(task+dots)

	// Static content
	greet := lipgloss.NewStyle().Foreground(titleColor).Bold(true).
		Render(greeting(hour) + ", " + greeterHostname())
	clock := valueStyle.Render(now.Format("03:04:05 PM"))
	date := dimStyle.Render(now.Format("Monday, January 2"))
	tag := lipgloss.NewStyle().Foreground(lipgloss.Color("#6C7086")).Italic(true).
		Render(fmt.Sprintf("\"%s\"", sessionTagline))

	return "\n" + centered.String() + "\n" +
		"  " + taskLine + "\n\n" +
		"  " + greet + "\n\n" +
		"  " + clock + "\n" +
		"  " + date + "\n\n" +
		"  " + tag + "\n"
}
