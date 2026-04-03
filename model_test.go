package main

import "testing"

func TestFormatUptime(t *testing.T) {
	tests := []struct {
		secs uint64
		want string
	}{
		{0, "0m"},
		{59, "0m"},
		{60, "1m"},
		{3599, "59m"},
		{3600, "1h 0m"},
		{3660, "1h 1m"},
		{86400, "1d 0h 0m"},
		{90061, "1d 1h 1m"},
	}
	for _, tt := range tests {
		got := formatUptime(tt.secs)
		if got != tt.want {
			t.Errorf("formatUptime(%d) = %q, want %q", tt.secs, got, tt.want)
		}
	}
}

func TestTrimSpark(t *testing.T) {
	tests := []struct {
		s    string
		n    int
		want string
	}{
		{"▁▂▃▄▅", 3, "▃▄▅"},
		{"▁▂▃", 5, "▁▂▃"},
		{"", 5, ""},
		{"▁▂▃▄▅", 5, "▁▂▃▄▅"},
	}
	for _, tt := range tests {
		got := trimSpark(tt.s, tt.n)
		if got != tt.want {
			t.Errorf("trimSpark(%q, %d) = %q, want %q", tt.s, tt.n, got, tt.want)
		}
	}
}
