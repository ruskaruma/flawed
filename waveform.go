package main

import (
	"fmt"
	"strings"
)

// ansiColor wraps a character in a 24-bit ANSI foreground color escape.
func ansiColor(r, g, b int, ch string) string {
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm%s\x1b[0m", r, g, b, ch)
}

// cpuWaveform renders a vertical bar chart of the CPU history.
// height is the number of terminal rows; each row has 2 half-row slots (using ▄/█/space).
// width is the number of columns (== number of time samples shown).
func cpuWaveform(values []float64, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	halfRows := height * 2 // total half-row resolution

	// Take the last `width` samples, padding with 0 on the left if needed.
	samples := make([]float64, width)
	for i := 0; i < width; i++ {
		srcIdx := len(values) - width + i
		if srcIdx >= 0 && srcIdx < len(values) {
			samples[i] = values[srcIdx]
		}
	}

	// colorFor returns RGB for a 0-100 value.
	colorFor := func(v float64) (int, int, int) {
		switch {
		case v > 80:
			return 255, 85, 85 // red
		case v >= 50:
			return 241, 250, 140 // yellow
		default:
			return 80, 250, 123 // green
		}
	}

	var sb strings.Builder
	sb.Grow(height * width * 25)

	for row := 0; row < height; row++ {
		if row > 0 {
			sb.WriteString("\n")
		}
		// row 0 = top, row height-1 = bottom.
		// lower half-row index for this terminal row (0 = very bottom of grid):
		lowerHalf := (height - 1 - row) * 2 // index of lower half-block in this row
		upperHalf := lowerHalf + 1          // index of upper half-block

		for _, v := range samples {
			filled := int(v / 100 * float64(halfRows))
			if filled > halfRows {
				filled = halfRows
			}
			r, g, b := colorFor(v)

			lowerFilled := filled > lowerHalf
			upperFilled := filled > upperHalf

			switch {
			case upperFilled:
				sb.WriteString(ansiColor(r, g, b, "█"))
			case lowerFilled:
				sb.WriteString(ansiColor(r, g, b, "▄"))
			default:
				sb.WriteByte(' ')
			}
		}
	}
	return sb.String()
}

// netHeatmap renders a scrolling colored heatmap for network upload and download.
// Returns a 4-row strip: 2 rows for upload (green gradient), 2 rows for download (cyan gradient).
func netHeatmap(upValues, downValues []float64, width int) string {
	if width <= 0 {
		return ""
	}

	// Fetch last `width` samples, padding with 0.
	getSamples := func(values []float64) []float64 {
		out := make([]float64, width)
		for i := 0; i < width; i++ {
			srcIdx := len(values) - width + i
			if srcIdx >= 0 && srcIdx < len(values) {
				out[i] = values[srcIdx]
			}
		}
		return out
	}
	up := getSamples(upValues)
	down := getSamples(downValues)

	// Color intensity: maps 0–100 to a color gradient.
	greenFor := func(v float64) (int, int, int) {
		t := v / 100
		if t > 1 {
			t = 1
		}
		return int(20 + 230*t), int(80 + 170*t), int(20 + 40*t)
	}
	cyanFor := func(v float64) (int, int, int) {
		t := v / 100
		if t > 1 {
			t = 1
		}
		return int(20 + 20*t), int(80 + 170*t), int(100 + 155*t)
	}

	renderStrip := func(samples []float64, colorFn func(float64) (int, int, int)) string {
		var sb strings.Builder
		// Row 1 (top)
		for _, v := range samples {
			r, g, b := colorFn(v)
			sb.WriteString(ansiColor(r, g, b, "█"))
		}
		sb.WriteString("\n")
		// Row 2 (bottom) — slightly dimmer
		for _, v := range samples {
			r, g, b := colorFn(v * 0.7)
			sb.WriteString(ansiColor(r, g, b, "█"))
		}
		return sb.String()
	}

	return renderStrip(up, greenFor) + "\n" + renderStrip(down, cyanFor)
}
