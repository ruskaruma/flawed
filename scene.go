package main

import (
	"fmt"
	"strings"
)

// viewScene renders an animated ASCII mountain-cabin scene: twinkling stars,
// mountain silhouette, smoke rising from a lit hut, and a ground row.
// Returns 13 newline-joined rows.
func viewScene(width, frame int) string {
	if width < 20 {
		return ""
	}

	const cw = 40 // canonical canvas width

	// ANSI 24-bit foreground color wrapper.
	fg := func(r, g, b int, s string) string {
		return fmt.Sprintf("\x1b[38;2;%d;%d;%dm%s\x1b[0m", r, g, b, s)
	}
	clamp := func(v, lo, hi int) int {
		if v < lo {
			return lo
		}
		if v > hi {
			return hi
		}
		return v
	}

	sidePad := (width - cw) / 2
	if sidePad < 0 {
		sidePad = 0
	}
	prefix := strings.Repeat(" ", sidePad)

	// ── Stars ──────────────────────────────────────────────────────────────
	type starType struct {
		ch      string
		r, g, b int
	}
	starPalette := []starType{
		{"*", 255, 255, 175},
		{".", 70, 70, 110},
		{"·", 140, 140, 195},
		{" ", 0, 0, 0},
		{" ", 0, 0, 0},
		{" ", 0, 0, 0},
		{" ", 0, 0, 0},
		{"+", 210, 210, 255},
		{" ", 0, 0, 0},
		{" ", 0, 0, 0},
	}

	// starLine builds one star row. If moonCol >= 0, a crescent "( )" is painted there.
	starLine := func(seed, moonCol int) string {
		var sb strings.Builder
		for c := 0; c < cw; c++ {
			if moonCol >= 0 && c >= moonCol && c < moonCol+3 {
				switch c - moonCol {
				case 0:
					sb.WriteString(fg(210, 220, 140, "("))
				case 1:
					sb.WriteByte(' ')
				case 2:
					sb.WriteString(fg(210, 220, 140, ")"))
				}
				continue
			}
			idx := (seed*13 + c*7 + frame/7) % len(starPalette)
			st := starPalette[idx]
			if st.ch == " " {
				sb.WriteByte(' ')
			} else {
				sb.WriteString(fg(st.r, st.g, st.b, st.ch))
			}
		}
		return prefix + sb.String()
	}

	// ── Smoke ──────────────────────────────────────────────────────────────
	const chimneyX = 14 // center column of chimney

	// level 0 = densest (just above chimney), level 2 = faintest (drifting high).
	smokeLine := func(level int) string {
		drift := ((frame/4 + level*5) % 9) - 4
		center := chimneyX + drift
		halfW := 1 + level // narrow at bottom, wider up top
		opacity := clamp(185-level*55, 60, 185)
		smokeChars := []string{"~", "~", "-", "~", " ", "~", "~", "-"}

		var sb strings.Builder
		for c := 0; c < cw; c++ {
			dist := c - center
			if dist >= -halfW && dist <= halfW {
				idx := (frame/3 + c + level*2) % len(smokeChars)
				ch := smokeChars[idx]
				if ch == " " {
					sb.WriteByte(' ')
				} else {
					sb.WriteString(fg(opacity, opacity, opacity+12, ch))
				}
			} else {
				sb.WriteByte(' ')
			}
		}
		return prefix + sb.String()
	}

	// ── Scene row builder ───────────────────────────────────────────────────
	type seg struct {
		x       int
		s       string
		r, g, b int
	}
	sceneRow := func(segs ...seg) string {
		cells := make([]string, cw)
		for i := range cells {
			cells[i] = " "
		}
		for _, sg := range segs {
			for i, ch := range []rune(sg.s) {
				pos := sg.x + i
				if pos >= 0 && pos < cw {
					cells[pos] = fg(sg.r, sg.g, sg.b, string(ch))
				}
			}
		}
		var sb strings.Builder
		for _, c := range cells {
			sb.WriteString(c)
		}
		return prefix + sb.String()
	}

	// ── Animated values ─────────────────────────────────────────────────────
	// Firelight flicker in the window.
	flicker := (frame * 11 % 17) - 8
	winR := clamp(222+flicker, 190, 255)
	winG := clamp(165+flicker/2, 128, 208)

	// ── Row assembly ───────────────────────────────────────────────────────
	//
	// Layout (cw=40):
	//   Left mountain center col 6  (peak /\ at 5-6, slope /  \ at 4-7)
	//   Right mountain center col 26 (peak /\ at 25-26, slope /  \ at 24-27)
	//   Valley underscores cols 8-23 (16 chars)
	//   Chimney || at cols 13-14
	//   Hut: roof /  \ cols 12-15, base /    \ cols 11-16, walls || cols 10-17
	//   Window [  ] cols 12-15
	//   Moon ( ) at col 35 in row 1

	valley := strings.Repeat("_", 16)      // cols 8-23
	rightGround := strings.Repeat("_", 12) // cols 28-39

	rows := []string{
		starLine(0, -1), // row 0: pure stars
		starLine(7, 35), // row 1: stars + crescent moon at col 35

		// Mountain peaks
		sceneRow(
			seg{5, "/\\", 185, 208, 232},
			seg{25, "/\\", 168, 192, 222},
		),
		// Mountain slopes + valley floor
		sceneRow(
			seg{4, "/  \\", 148, 175, 212},
			seg{8, valley, 90, 112, 148},
			seg{24, "/  \\", 133, 162, 200},
			seg{28, rightGround, 90, 112, 148},
		),

		// Smoke (rising from chimney up through mountain valley)
		smokeLine(2), // faintest, highest
		smokeLine(1),
		smokeLine(0), // densest, just above chimney

		// Chimney shaft
		sceneRow(seg{13, "||", 155, 115, 75}),

		// Hut roof (two rows to give height)
		sceneRow(seg{12, "/  \\", 200, 155, 100}),
		sceneRow(seg{11, "/    \\", 200, 155, 100}),

		// Hut walls + flickering window
		sceneRow(
			seg{10, "| ", 185, 135, 85},
			seg{12, "[  ]", winR, winG, 55},
			seg{16, " |", 185, 135, 85},
		),

		// Hut base
		sceneRow(seg{10, "|______|", 155, 105, 65}),

		// Ground
		sceneRow(seg{0, strings.Repeat("~", cw), 38, 68, 42}),
	}

	return strings.Join(rows, "\n")
}
