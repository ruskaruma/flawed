package main

import (
	"fmt"
	"math"
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

// sessionTaglineOffset randomises which tagline we start cycling from.
var sessionTaglineOffset = rand.Intn(len(taglines))

var matrixCharSet = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789@#$%^&*!?")

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

// matrixRain renders a deterministic matrix-style falling character grid.
// All randomness derives from (column, frame, row) arithmetic — no rand state used.
func matrixRain(width, rows, frame int) string {
	if width <= 0 || rows <= 0 {
		return ""
	}

	var sb strings.Builder
	sb.Grow(rows * width * 25)

	for row := 0; row < rows; row++ {
		if row > 0 {
			sb.WriteString("\n")
		}
		for col := 0; col < width; col++ {
			speed := 1 + col%3
			cycleLen := rows * 2
			if cycleLen < 8 {
				cycleLen = 8
			}
			phase := (frame/speed + col*13) % cycleLen
			// Column is "active" for the first half of its cycle.
			if phase >= cycleLen/2 {
				sb.WriteByte(' ')
				continue
			}
			headPos := phase % rows
			dist := headPos - row
			if dist < 0 {
				sb.WriteByte(' ')
				continue
			}
			chIdx := (col*31 + row*17 + frame/max(speed, 1)) % len(matrixCharSet)
			ch := string(matrixCharSet[chIdx])
			switch dist {
			case 0: // bright head
				fmt.Fprintf(&sb, "\x1b[38;2;200;255;210m%s\x1b[0m", ch)
			case 1: // mid trail
				fmt.Fprintf(&sb, "\x1b[38;2;80;200;100m%s\x1b[0m", ch)
			default: // dim tail
				fmt.Fprintf(&sb, "\x1b[38;2;25;75;40m%s\x1b[0m", ch)
			}
		}
	}
	return sb.String()
}

// max returns the larger of two ints.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// animatedTagline types a tagline in, holds it, then erases it, cycling indefinitely.
// All state is derived from frame — no persistent model state needed.
func animatedTagline(frame int) string {
	const cycleLen = 128
	tagIdx := ((frame / cycleLen) + sessionTaglineOffset) % len(taglines)
	phase := frame % cycleLen
	tag := taglines[tagIdx]
	n := len(tag)

	const holdEnd = cycleLen - 12
	var visible int
	switch {
	case phase < n && phase < holdEnd:
		visible = phase
	case phase < holdEnd:
		visible = n
	default:
		erased := (phase - holdEnd) * 3
		visible = n - erased
		if visible < 0 {
			visible = 0
		}
	}
	if visible > n {
		visible = n
	}

	shown := tag[:visible]

	isTyping := phase < n
	isErasing := phase >= holdEnd
	cursor := ""
	if isTyping || isErasing {
		if (frame/3)%2 == 0 {
			cursor = "█"
		} else {
			cursor = " "
		}
	}

	return lipgloss.NewStyle().Foreground(lipgloss.Color("#6C7086")).Italic(true).
		Render(fmt.Sprintf("\"%s%s\"", shown, cursor))
}

// lerpColor linearly interpolates between two RGB colors and returns a hex string.
func lerpColor(a, b [3]int, t float64) string {
	r := int(float64(a[0]) + t*float64(b[0]-a[0]))
	g := int(float64(a[1]) + t*float64(b[1]-a[1]))
	bl := int(float64(a[2]) + t*float64(b[2]-a[2]))
	if r < 0 {
		r = 0
	}
	if r > 255 {
		r = 255
	}
	if g < 0 {
		g = 0
	}
	if g > 255 {
		g = 255
	}
	if bl < 0 {
		bl = 0
	}
	if bl > 255 {
		bl = 255
	}
	return fmt.Sprintf("#%02x%02x%02x", r, g, bl)
}

// pulsingGreeting renders the greeting line with a slow color pulse.
func pulsingGreeting(hour, frame int) string {
	t := (math.Sin(float64(frame)*0.08) + 1) / 2
	base := [3]int{86, 182, 194} // titleColor #56B6C2
	bright := [3]int{180, 240, 255}
	col := lipgloss.Color(lerpColor(base, bright, t))
	return lipgloss.NewStyle().Foreground(col).Bold(true).
		Render(greeting(hour) + ", " + greeterHostname())
}

// colorCyclingClock renders the clock with a slowly rotating RGB hue.
func colorCyclingClock(now time.Time, frame int) string {
	t := float64(frame) * 0.12
	r := int(127 + 120*math.Sin(t))
	g := int(200 + 55*math.Sin(t+2.094))
	b := int(220 + 35*math.Sin(t+4.189))
	if r < 0 {
		r = 0
	}
	if r > 255 {
		r = 255
	}
	if g < 0 {
		g = 0
	}
	if g > 255 {
		g = 255
	}
	if b < 0 {
		b = 0
	}
	if b > 255 {
		b = 255
	}
	col := lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))
	return lipgloss.NewStyle().Foreground(col).Bold(true).Render(now.Format("03:04:05 PM"))
}

func viewGreeter(width, frame, rainRows int) string {
	now := time.Now()
	hour := now.Hour()

	greet := pulsingGreeting(hour, frame)
	clock := colorCyclingClock(now, frame)
	date := dimStyle.Render(now.Format("Monday, January 2"))
	tag := animatedTagline(frame)

	// Matrix rain fills the space above the text content.
	rain := matrixRain(width, rainRows, frame)

	return rain + "\n\n" +
		"  " + greet + "\n\n" +
		"  " + clock + "\n" +
		"  " + date + "\n\n" +
		"  " + tag + "\n"
}
