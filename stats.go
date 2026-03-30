package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

type sysStats struct {
	// System info
	hostname    string
	uptime      uint64 // seconds
	os          string // e.g. "linux"
	platform    string // e.g. "ubuntu"
	platformVer string // e.g. "22.04"
	kernelVer   string
	kernelArch  string
	virtSystem  string // e.g. "docker", "kvm", ""
	virtRole    string // e.g. "guest", "host", ""
	cpuModel    string // e.g. "Intel Core i7-10750H"
	cpuFreqMHz  float64
	cpuCores    int // physical
	cpuThreads  int // logical

	// Load average
	load1  float64
	load5  float64
	load15 float64

	// CPU
	cpuOverall float64
	cpuPerCore []float64

	// Memory
	memTotal    uint64
	memUsed     uint64
	memPercent  float64
	swapTotal   uint64
	swapUsed    uint64
	swapPercent float64

	// Disk — root partition (kept for sparkline history)
	diskTotal   uint64
	diskUsed    uint64
	diskPercent float64

	// Filesystems — all mounts
	filesystems []fsInfo

	// Disk I/O
	diskReadSpeed  uint64 // bytes/sec
	diskWriteSpeed uint64 // bytes/sec

	// Network
	netUp    uint64 // bytes/sec
	netDown  uint64 // bytes/sec
	netConns int    // open connections count

	// Users
	users []string // logged-in usernames (deduplicated)

	// Hardware
	battery *batteryInfo
	temps   []tempInfo
}

type batteryInfo struct {
	percent   float64
	pluggedIn bool
}

type tempInfo struct {
	label string
	temp  float64
}

type fsInfo struct {
	mount   string
	fstype  string
	total   uint64
	used    uint64
	percent float64
}

var prevNetSent, prevNetRecv uint64
var prevNetTime time.Time

var prevDiskRead, prevDiskWrite uint64
var prevDiskTime time.Time

func collectStats() sysStats {
	var s sysStats

	// Host info
	info, err := host.Info()
	if err == nil {
		s.hostname = info.Hostname
		s.uptime = info.Uptime
		s.os = info.OS
		s.platform = info.Platform
		s.platformVer = info.PlatformVersion
		s.kernelVer = info.KernelVersion
		s.kernelArch = info.KernelArch
		s.virtSystem = info.VirtualizationSystem
		s.virtRole = info.VirtualizationRole
	}

	// CPU info (model, freq)
	cpuInfos, err := cpu.Info()
	if err == nil && len(cpuInfos) > 0 {
		s.cpuModel = cpuInfos[0].ModelName
		s.cpuFreqMHz = cpuInfos[0].Mhz
	}
	physical, err := cpu.Counts(false)
	if err == nil {
		s.cpuCores = physical
	}
	logical, err := cpu.Counts(true)
	if err == nil {
		s.cpuThreads = logical
	}

	// Load average
	avg, err := load.Avg()
	if err == nil {
		s.load1 = avg.Load1
		s.load5 = avg.Load5
		s.load15 = avg.Load15
	}

	// CPU overall
	pcts, err := cpu.Percent(0, false)
	if err == nil && len(pcts) > 0 {
		s.cpuOverall = pcts[0]
	}

	// CPU per core
	corePcts, err := cpu.Percent(0, true)
	if err == nil {
		s.cpuPerCore = corePcts
	}

	// Memory
	vm, err := mem.VirtualMemory()
	if err == nil {
		s.memTotal = vm.Total
		s.memUsed = vm.Used
		s.memPercent = vm.UsedPercent
	}

	// Swap
	sw, err := mem.SwapMemory()
	if err == nil {
		s.swapTotal = sw.Total
		s.swapUsed = sw.Used
		s.swapPercent = sw.UsedPercent
	}

	// Disk (root partition — for sparkline history)
	root := "/"
	if runtime.GOOS == "windows" {
		root = "C:\\"
	}
	du, err := disk.Usage(root)
	if err == nil {
		s.diskTotal = du.Total
		s.diskUsed = du.Used
		s.diskPercent = du.UsedPercent
	}

	// Filesystems — all real partitions
	s.filesystems = collectFilesystems()

	// Disk I/O
	ioCounters, err := disk.IOCounters()
	if err == nil {
		var totalRead, totalWrite uint64
		for _, d := range ioCounters {
			totalRead += d.ReadBytes
			totalWrite += d.WriteBytes
		}
		now := time.Now()
		if !prevDiskTime.IsZero() {
			dt := now.Sub(prevDiskTime).Seconds()
			if dt > 0 {
				s.diskReadSpeed = uint64(float64(totalRead-prevDiskRead) / dt)
				s.diskWriteSpeed = uint64(float64(totalWrite-prevDiskWrite) / dt)
			}
		}
		prevDiskRead = totalRead
		prevDiskWrite = totalWrite
		prevDiskTime = now
	}

	// Network speed
	counters, err := net.IOCounters(false)
	if err == nil && len(counters) > 0 {
		now := time.Now()
		sent := counters[0].BytesSent
		recv := counters[0].BytesRecv
		if !prevNetTime.IsZero() {
			dt := now.Sub(prevNetTime).Seconds()
			if dt > 0 {
				s.netUp = uint64(float64(sent-prevNetSent) / dt)
				s.netDown = uint64(float64(recv-prevNetRecv) / dt)
			}
		}
		prevNetSent = sent
		prevNetRecv = recv
		prevNetTime = now
	}

	// Network connections count
	conns, err := net.Connections("all")
	if err == nil {
		s.netConns = len(conns)
	}

	// Logged-in users
	users, err := host.Users()
	if err == nil {
		seen := make(map[string]bool)
		for _, u := range users {
			if !seen[u.User] {
				seen[u.User] = true
				s.users = append(s.users, u.User)
			}
		}
	}

	// Battery
	s.battery = collectBattery()

	// Temperatures
	s.temps = collectTemps()

	return s
}

func collectFilesystems() []fsInfo {
	parts, err := disk.Partitions(false) // false = only real devices
	if err != nil {
		return nil
	}

	var fss []fsInfo
	seen := make(map[string]bool)
	for _, p := range parts {
		// Skip pseudo and duplicate filesystems
		if seen[p.Device] {
			continue
		}
		seen[p.Device] = true

		// Skip common virtual filesystems
		switch p.Fstype {
		case "squashfs", "overlay", "tmpfs", "devtmpfs", "devfs", "iso9660":
			continue
		}

		usage, err := disk.Usage(p.Mountpoint)
		if err != nil || usage.Total == 0 {
			continue
		}
		fss = append(fss, fsInfo{
			mount:   p.Mountpoint,
			fstype:  p.Fstype,
			total:   usage.Total,
			used:    usage.Used,
			percent: usage.UsedPercent,
		})
	}
	return fss
}

func collectBattery() *batteryInfo {
	matches, err := filepath.Glob("/sys/class/power_supply/BAT*")
	if err != nil || len(matches) == 0 {
		return nil
	}
	batPath := matches[0]

	capBytes, err := os.ReadFile(filepath.Join(batPath, "capacity"))
	if err != nil {
		return nil
	}
	capStr := strings.TrimSpace(string(capBytes))
	var pct float64
	fmt.Sscanf(capStr, "%f", &pct)

	statusBytes, _ := os.ReadFile(filepath.Join(batPath, "status"))
	status := strings.TrimSpace(string(statusBytes))
	plugged := status == "Charging" || status == "Full"

	return &batteryInfo{percent: pct, pluggedIn: plugged}
}

func collectTemps() []tempInfo {
	sensors, err := host.SensorsTemperatures()
	if err != nil || len(sensors) == 0 {
		return nil
	}
	var temps []tempInfo
	for _, s := range sensors {
		if s.Temperature > 0 {
			temps = append(temps, tempInfo{label: s.SensorKey, temp: s.Temperature})
		}
	}
	return temps
}
