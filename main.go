package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	interval := flag.Duration("interval", 2*time.Second, "refresh interval")
	flag.DurationVar(interval, "i", 2*time.Second, "refresh interval (shorthand)")

	once := flag.Bool("once", false, "print once and exit")

	procs := flag.Int("procs", 10, "number of top processes to show")
	flag.IntVar(procs, "n", 10, "number of top processes (shorthand)")

	sortBy := flag.String("sort", "cpu", "sort processes by: cpu or mem")

	alertCPU := flag.Float64("alert-cpu", 85, "CPU alert threshold (%)")
	alertRAM := flag.Float64("alert-ram", 90, "RAM alert threshold (%)")

	flag.Parse()

	if *sortBy != "cpu" && *sortBy != "mem" {
		fmt.Fprintf(os.Stderr, "invalid --sort value: %s (use cpu or mem)\n", *sortBy)
		os.Exit(1)
	}

	cfg := config{
		interval: *interval,
		once:     *once,
		procs:    *procs,
		sortBy:   *sortBy,
		alertCPU: *alertCPU,
		alertRAM: *alertRAM,
	}

	m := newModel(cfg)

	if cfg.once {
		// Collect stats once, print, and exit.
		stats := collectStats()
		procs := collectProcesses(cfg.procs, cfg.sortBy)
		docker := collectDocker()
		m.stats = stats
		m.procs = procs
		m.docker = docker
		m.updateAlerts()
		m.pushHistory()
		if err := logStats(m.stats, m.docker); err != nil {
			fmt.Fprintf(os.Stderr, "warning: %v\n", err)
		}
		fmt.Println(m.View())
		return
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
