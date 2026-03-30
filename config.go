package main

import "time"

type config struct {
	interval time.Duration
	once     bool
	procs    int
	sortBy   string
	alertCPU float64
	alertRAM float64
	verbose  bool
}
