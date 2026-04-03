package main

import "testing"

func TestSparklineEmpty(t *testing.T) {
	if sparkline(nil) != "" {
		t.Error("sparkline(nil) should return empty string")
	}
	if sparkline([]float64{}) != "" {
		t.Error("sparkline([]) should return empty string")
	}
}

func TestSparklineLength(t *testing.T) {
	data := []float64{0, 25, 50, 75, 100}
	got := sparkline(data)
	if len([]rune(got)) != len(data) {
		t.Errorf("sparkline length: got %d, want %d", len([]rune(got)), len(data))
	}
}

func TestSparklineClampsOutOfRange(t *testing.T) {
	// Should not panic on out-of-range values.
	sparkline([]float64{-10, 150})
}

func TestSparklineMinMax(t *testing.T) {
	runes := []rune(sparkChars)
	low := []rune(sparkline([]float64{0}))[0]
	high := []rune(sparkline([]float64{100}))[0]
	if low != runes[0] {
		t.Errorf("sparkline(0) = %q, want %q", string(low), string(runes[0]))
	}
	if high != runes[len(runes)-1] {
		t.Errorf("sparkline(100) = %q, want %q", string(high), string(runes[len(runes)-1]))
	}
}

func TestHistoryMaxLen(t *testing.T) {
	h := newHistory(3)
	h.push(1)
	h.push(2)
	h.push(3)
	h.push(4) // evicts 1

	vals := h.values()
	if len(vals) != 3 {
		t.Fatalf("history length: got %d, want 3", len(vals))
	}
	if vals[0] != 2 || vals[1] != 3 || vals[2] != 4 {
		t.Errorf("history values: got %v, want [2 3 4]", vals)
	}
}

func TestHistoryUnderCapacity(t *testing.T) {
	h := newHistory(10)
	h.push(5)
	h.push(10)
	vals := h.values()
	if len(vals) != 2 {
		t.Errorf("history length: got %d, want 2", len(vals))
	}
}
