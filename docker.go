package main

import (
	"os/exec"
	"strconv"
	"strings"
)

type containerInfo struct {
	name   string
	status string
	cpu    float64
	memMB  float64
}

func collectDocker() []containerInfo {
	// Use docker CLI — works everywhere Docker is installed, no SDK dependency issues.
	out, err := exec.Command("docker", "ps", "--format", "{{.Names}}\t{{.Status}}").Output()
	if err != nil {
		return nil
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return nil
	}

	var infos []containerInfo
	for _, line := range lines {
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) < 2 {
			continue
		}
		infos = append(infos, containerInfo{
			name:   parts[0],
			status: parts[1],
		})
	}

	// Get CPU/mem stats for running containers
	statsOut, err := exec.Command("docker", "stats", "--no-stream", "--format", "{{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}").Output()
	if err != nil {
		return infos
	}

	statsMap := make(map[string][2]string)
	for _, line := range strings.Split(strings.TrimSpace(string(statsOut)), "\n") {
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 3 {
			continue
		}
		statsMap[parts[0]] = [2]string{parts[1], parts[2]}
	}

	for i := range infos {
		if s, ok := statsMap[infos[i].name]; ok {
			infos[i].cpu = parsePct(s[0])
			infos[i].memMB = parseMemMB(s[1])
		}
	}

	return infos
}

func parsePct(s string) float64 {
	s = strings.TrimSuffix(strings.TrimSpace(s), "%")
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func parseMemMB(s string) float64 {
	// Format: "123.4MiB / 16GiB" — we want the first part
	parts := strings.SplitN(s, "/", 2)
	if len(parts) == 0 {
		return 0
	}
	val := strings.TrimSpace(parts[0])
	val = strings.ReplaceAll(val, " ", "")

	multiplier := 1.0
	switch {
	case strings.HasSuffix(val, "GiB"):
		multiplier = 1024
		val = strings.TrimSuffix(val, "GiB")
	case strings.HasSuffix(val, "MiB"):
		val = strings.TrimSuffix(val, "MiB")
	case strings.HasSuffix(val, "KiB"):
		multiplier = 1.0 / 1024
		val = strings.TrimSuffix(val, "KiB")
	case strings.HasSuffix(val, "B"):
		multiplier = 1.0 / 1024 / 1024
		val = strings.TrimSuffix(val, "B")
	}

	v, _ := strconv.ParseFloat(val, 64)
	return v * multiplier
}

