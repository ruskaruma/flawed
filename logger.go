package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func logStats(s sysStats, containers []containerInfo) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("logStats: get home dir: %w", err)
	}

	dir := filepath.Join(home, ".flawed")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("logStats: create log dir: %w", err)
	}

	path := filepath.Join(dir, "log.csv")
	isNew := false
	if _, err := os.Stat(path); os.IsNotExist(err) {
		isNew = true
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("logStats: open log file: %w", err)
	}
	defer f.Close()

	if isNew {
		fmt.Fprintln(f, "timestamp,cpu_pct,mem_pct,swap_pct,disk_pct,net_up_bytes,net_down_bytes,containers")
	}

	fmt.Fprintf(f, "%s,%.1f,%.1f,%.1f,%.1f,%d,%d,%d\n",
		time.Now().Format(time.RFC3339),
		s.cpuOverall,
		s.memPercent,
		s.swapPercent,
		s.diskPercent,
		s.netUp,
		s.netDown,
		len(containers),
	)
	return nil
}
