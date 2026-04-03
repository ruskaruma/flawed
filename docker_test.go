package main

import "testing"

func TestParsePct(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"12.5%", 12.5},
		{"0.00%", 0},
		{"100%", 100},
		{"  42.3%  ", 42.3},
		{"", 0},
	}
	for _, tt := range tests {
		got := parsePct(tt.input)
		if got != tt.want {
			t.Errorf("parsePct(%q) = %f, want %f", tt.input, got, tt.want)
		}
	}
}

func TestParseMemMB(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"123.4MiB / 16GiB", 123.4},
		{"2GiB / 16GiB", 2048},
		{"512KiB / 2GiB", 0.5},
		{"0MiB / 8GiB", 0},
		{"", 0},
	}
	for _, tt := range tests {
		got := parseMemMB(tt.input)
		if got != tt.want {
			t.Errorf("parseMemMB(%q) = %f, want %f", tt.input, got, tt.want)
		}
	}
}
