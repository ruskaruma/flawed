package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/shirou/gopsutil/v3/process"
)

type procInfo struct {
	pid  int32
	name string
	cpu  float64
	mem  float32
}

func collectProcesses(n int, sortBy string) []procInfo {
	procs, err := process.Processes()
	if err != nil {
		return nil
	}

	var infos []procInfo
	for _, p := range procs {
		name, _ := p.Name()
		cpuPct, _ := p.CPUPercent()
		memPct, _ := p.MemoryPercent()
		if name == "" {
			continue
		}
		infos = append(infos, procInfo{
			pid:  p.Pid,
			name: name,
			cpu:  cpuPct,
			mem:  memPct,
		})
	}

	switch sortBy {
	case "mem":
		sort.Slice(infos, func(i, j int) bool { return infos[i].mem > infos[j].mem })
	default:
		sort.Slice(infos, func(i, j int) bool { return infos[i].cpu > infos[j].cpu })
	}

	if len(infos) > n {
		infos = infos[:n]
	}
	return infos
}

func killProcess(pid int32) error {
	p, err := os.FindProcess(int(pid))
	if err != nil {
		return fmt.Errorf("find process %d: %w", pid, err)
	}
	return p.Kill()
}
