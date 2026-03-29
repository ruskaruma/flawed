package main

import (
	"os"
	"testing"
)

func TestGreeting(t *testing.T) {
	tests := []struct {
		name     string
		hour     int
		expected string
	}{
		{"early morning", 5, "Good morning"},
		{"mid morning", 9, "Good morning"},
		{"just before noon", 11, "Good morning"},
		{"noon", 12, "Good afternoon"},
		{"afternoon", 14, "Good afternoon"},
		{"just before evening", 16, "Good afternoon"},
		{"evening", 17, "Good evening"},
		{"late evening", 20, "Good evening"},
		{"night", 21, "Good night"},
		{"midnight", 0, "Good night"},
		{"late night", 3, "Good night"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := greeting(tt.hour)
			if got != tt.expected {
				t.Errorf("greeting(%d) = %q, want %q", tt.hour, got, tt.expected)
			}
		})
	}
}

func TestGreeterHostname(t *testing.T) {
	got := greeterHostname()
	if got == "" {
		t.Error("greeterHostname() returned empty string")
	}

	// If we can get the real hostname, verify it matches
	if h, err := os.Hostname(); err == nil {
		if got != h {
			t.Errorf("greeterHostname() = %q, want %q", got, h)
		}
	} else {
		// os.Hostname failed — function should fall back to "localhost"
		if got != "localhost" {
			t.Errorf("greeterHostname() = %q, want %q when hostname unavailable", got, "localhost")
		}
	}
}
