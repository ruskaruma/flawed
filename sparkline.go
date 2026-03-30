package main

const sparkChars = "▁▂▃▄▅▆▇█"

var sparkRunes = []rune(sparkChars)

// sparkline renders a sparkline string from a slice of float64 values (0-100).
func sparkline(data []float64) string {
	if len(data) == 0 {
		return ""
	}
	out := make([]rune, len(data))
	for i, v := range data {
		if v < 0 {
			v = 0
		}
		if v > 100 {
			v = 100
		}
		idx := int(v / 100 * float64(len(sparkRunes)-1))
		if idx >= len(sparkRunes) {
			idx = len(sparkRunes) - 1
		}
		out[i] = sparkRunes[idx]
	}
	return string(out)
}

// history tracks the last maxLen samples.
type history struct {
	data   []float64
	maxLen int
}

func newHistory(maxLen int) *history {
	return &history{maxLen: maxLen}
}

func (h *history) push(v float64) {
	h.data = append(h.data, v)
	if len(h.data) > h.maxLen {
		h.data = h.data[len(h.data)-h.maxLen:]
	}
}

func (h *history) values() []float64 {
	return h.data
}
