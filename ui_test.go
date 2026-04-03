package main

import (
	"strings"
	"testing"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input uint64
		want  string
	}{
		{0, "0B"},
		{512, "512B"},
		{1023, "1023B"},
		{1024, "1.0K"},
		{1536, "1.5K"},
		{1024 * 1024, "1.0M"},
		{1024 * 1024 * 1024, "1.0G"},
		{uint64(2.5 * 1024 * 1024 * 1024), "2.5G"},
	}
	for _, tt := range tests {
		got := formatBytes(tt.input)
		if got != tt.want {
			t.Errorf("formatBytes(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFormatBytesPerSec(t *testing.T) {
	got := formatBytesPerSec(1024)
	if got != "1.0K/s" {
		t.Errorf("formatBytesPerSec(1024) = %q, want %q", got, "1.0K/s")
	}
}

func TestRightPad(t *testing.T) {
	tests := []struct {
		s    string
		n    int
		want string
	}{
		{"foo", 5, "foo  "},
		{"hello", 5, "hello"},
		{"toolong", 4, "tool"},
		{"", 3, "   "},
	}
	for _, tt := range tests {
		got := rightPad(tt.s, tt.n)
		if got != tt.want {
			t.Errorf("rightPad(%q, %d) = %q, want %q", tt.s, tt.n, got, tt.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		s    string
		n    int
		want string
	}{
		{"short", 10, "short"},
		{"exactly10x", 10, "exactly10x"},
		{"this is too long", 10, "this is..."},
		{"ab", 2, "ab"},
		{"abc", 2, "ab"},
	}
	for _, tt := range tests {
		got := truncate(tt.s, tt.n)
		if got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.s, tt.n, got, tt.want)
		}
	}
}

func TestBar(t *testing.T) {
	// 0% bar should contain only empty characters.
	result0 := bar(0, 10)
	if !strings.Contains(result0, "░") {
		t.Error("bar(0, 10): expected empty-fill characters")
	}

	// 100% bar should not be empty.
	result100 := bar(100, 10)
	if result100 == "" {
		t.Error("bar(100, 10) returned empty string")
	}

	// Should not panic on boundary values.
	bar(-1, 10)
	bar(101, 10)
	bar(50, 0)
}
