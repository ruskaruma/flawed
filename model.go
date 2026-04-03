package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Messages
type tickMsg time.Time
type animTickMsg time.Time
type statsMsg sysStats
type procsMsg []procInfo
type dockerMsg []containerInfo
type killResultMsg struct{ err error }

// Model
type model struct {
	cfg    config
	stats  sysStats
	procs  []procInfo
	docker []containerInfo

	// Sparkline histories (last 60 samples)
	cpuHist     *history
	memHist     *history
	swapHist    *history
	diskHist    *history
	netUpHist   *history
	netDownHist *history

	// Alerts
	cpuAlert bool
	ramAlert bool

	// UI state
	killMsg    string
	killMsgTTL int   // ticks remaining to show kill message
	animFrame  int   // animation frame counter for greeter
	logErr     error // last error from logStats

	// Process cursor + filter
	procCursor  int
	filterMode  bool
	filterInput string

	// Help overlay
	showHelp bool

	// Events
	events       []sysEvent
	prevCPUAlert bool
	prevRAMAlert bool

	// Pending kill name (captured at kill time for event log)
	pendingKillName string

	width  int
	height int
}

func newModel(cfg config) model {
	maxSamples := 60
	return model{
		cfg:         cfg,
		cpuHist:     newHistory(maxSamples),
		memHist:     newHistory(maxSamples),
		swapHist:    newHistory(maxSamples),
		diskHist:    newHistory(maxSamples),
		netUpHist:   newHistory(maxSamples),
		netDownHist: newHistory(maxSamples),
		width:       120,
		height:      40,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(m.cfg.interval),
		animTickCmd(),
		fetchStatsCmd,
		fetchProcsCmd(m.cfg.procs, m.cfg.sortBy),
		fetchDockerCmd,
	)
}

// filteredProcs returns the process list filtered by filterInput (case-insensitive name match).
func (m model) filteredProcs() []procInfo {
	if m.filterInput == "" {
		return m.procs
	}
	filter := strings.ToLower(m.filterInput)
	var out []procInfo
	for _, p := range m.procs {
		if strings.Contains(strings.ToLower(p.name), filter) {
			out = append(out, p)
		}
	}
	return out
}

func (m *model) clampCursor() {
	fp := m.filteredProcs()
	if len(fp) == 0 {
		m.procCursor = 0
		return
	}
	if m.procCursor >= len(fp) {
		m.procCursor = len(fp) - 1
	}
	if m.procCursor < 0 {
		m.procCursor = 0
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Filter mode captures all printable keys.
		if m.filterMode {
			switch msg.String() {
			case "esc", "enter":
				m.filterMode = false
			case "backspace":
				if len(m.filterInput) > 0 {
					runes := []rune(m.filterInput)
					m.filterInput = string(runes[:len(runes)-1])
					m.procCursor = 0
				}
			default:
				if len(msg.Runes) > 0 {
					m.filterInput += string(msg.Runes)
					m.procCursor = 0
				}
			}
			m.clampCursor()
			return m, nil
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "?":
			m.showHelp = !m.showHelp
		case "esc":
			if m.showHelp {
				m.showHelp = false
			} else if m.filterInput != "" {
				m.filterInput = ""
				m.procCursor = 0
			}
		case "/":
			m.filterMode = true
			m.filterInput = ""
			m.procCursor = 0
		case "j", "down":
			m.procCursor++
			m.clampCursor()
		case "k", "up":
			m.procCursor--
			m.clampCursor()
		case "x":
			fp := m.filteredProcs()
			if m.procCursor < len(fp) {
				p := fp[m.procCursor]
				m.pendingKillName = p.name
				return m, killCmd(p.pid)
			}
		case "s":
			if m.cfg.sortBy == "cpu" {
				m.cfg.sortBy = "mem"
			} else {
				m.cfg.sortBy = "cpu"
			}
			return m, fetchProcsCmd(m.cfg.procs, m.cfg.sortBy)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case animTickMsg:
		m.animFrame++
		return m, animTickCmd()

	case tickMsg:
		// killMsg TTL counts down only on data ticks, not anim ticks
		if m.killMsgTTL > 0 {
			m.killMsgTTL--
			if m.killMsgTTL == 0 {
				m.killMsg = ""
			}
		}
		return m, tea.Batch(
			tickCmd(m.cfg.interval),
			fetchStatsCmd,
			fetchProcsCmd(m.cfg.procs, m.cfg.sortBy),
			fetchDockerCmd,
		)

	case statsMsg:
		m.stats = sysStats(msg)
		m.updateAlerts()
		m.pushHistory()
		// Rising-edge event detection.
		if m.cpuAlert && !m.prevCPUAlert {
			m.addEvent("crit", fmt.Sprintf("CPU alert: %.0f%%", m.stats.cpuOverall))
		}
		if m.ramAlert && !m.prevRAMAlert {
			m.addEvent("warn", fmt.Sprintf("RAM alert: %.0f%%", m.stats.memPercent))
		}
		if m.stats.netUp > 10*1024*1024 || m.stats.netDown > 10*1024*1024 {
			m.addEvent("warn", fmt.Sprintf("net spike ↑%s ↓%s",
				formatBytes(m.stats.netUp), formatBytes(m.stats.netDown)))
		}
		m.prevCPUAlert = m.cpuAlert
		m.prevRAMAlert = m.ramAlert
		if err := logStats(m.stats, m.docker, m.cfg.verbose); err != nil {
			m.logErr = err
		}

	case procsMsg:
		m.procs = []procInfo(msg)
		m.clampCursor()

	case dockerMsg:
		m.docker = []containerInfo(msg)

	case killResultMsg:
		if msg.err != nil {
			m.killMsg = fmt.Sprintf("Kill failed: %v", msg.err)
			m.addEvent("warn", fmt.Sprintf("kill failed: %s", m.pendingKillName))
		} else {
			m.killMsg = fmt.Sprintf("Killed: %s", m.pendingKillName)
			m.addEvent("info", fmt.Sprintf("killed: %s", m.pendingKillName))
		}
		m.pendingKillName = ""
		m.killMsgTTL = 3
	}

	return m, nil
}

func (m *model) updateAlerts() {
	m.cpuAlert = m.stats.cpuOverall > m.cfg.alertCPU
	m.ramAlert = m.stats.memPercent > m.cfg.alertRAM
}

func (m *model) pushHistory() {
	m.cpuHist.push(m.stats.cpuOverall)
	m.memHist.push(m.stats.memPercent)
	m.swapHist.push(m.stats.swapPercent)
	m.diskHist.push(m.stats.diskPercent)
	upPct := float64(m.stats.netUp) / (100 * 1024 * 1024) * 100
	downPct := float64(m.stats.netDown) / (100 * 1024 * 1024) * 100
	m.netUpHist.push(upPct)
	m.netDownHist.push(downPct)
}

func (m model) viewHelp() string {
	helpContent := `
  KEYBINDINGS

  Navigation
  j / ↓        move cursor down in process list
  k / ↑        move cursor up in process list

  Actions
  x            kill selected process
  s            toggle sort order (cpu ↔ mem)
  /            filter processes by name
  Escape        exit filter mode / close help
  ?            toggle this help overlay
  q / Ctrl+C   quit
`
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(titleColor).
		Padding(1, 3).
		Render(helpContent)

	titleBar := lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).
		Render(titleStyle.Render(" FLAWED  System Monitor "))
	body := lipgloss.Place(m.width, m.height-2, lipgloss.Center, lipgloss.Center, box)
	return titleBar + "\n" + body
}

func (m model) View() string {
	if m.showHelp {
		return m.viewHelp()
	}

	w := m.width
	if w < 40 {
		w = 40
	}

	// Two-column layout: left ~60%, right ~40%
	leftW := w*3/5 - 1
	rightW := w - leftW - 1
	barW := 20
	sparkW := 30
	if leftW < 60 {
		barW = 15
		sparkW = 20
	}

	// === TITLE BAR ===
	title := titleStyle.Render(" FLAWED  System Monitor ")
	if m.cpuAlert || m.ramAlert {
		// Pulse effect: alternate between bright red and dim red
		if m.animFrame%6 < 3 {
			title = alertTitleStyle.Render(" ● ALERT  FLAWED  System Monitor ")
		} else {
			title = lipgloss.NewStyle().Bold(true).
				Foreground(lipgloss.Color("#FF8888")).
				Background(lipgloss.Color("#4A1515")).
				Padding(0, 1).
				Render(" ○ ALERT  FLAWED  System Monitor ")
		}
	}
	titleBar := lipgloss.NewStyle().Width(w).Align(lipgloss.Center).Render(title)

	// === LEFT COLUMN PANELS ===

	// CPU Panel — alert border pulses when threshold exceeded
	cpuContent := m.viewCPU(barW, sparkW)
	cpuPanel := panelAlert("CPU", cpuContent, leftW, m.cpuAlert, m.animFrame)

	// Memory Panel — alert border pulses when threshold exceeded
	memContent := m.viewMemory(barW, sparkW)
	memPanel := panelAlert("Memory", memContent, leftW, m.ramAlert, m.animFrame)

	// Filesystems panel (full width left column)
	fsContent := m.viewFilesystems(leftW - 8)
	fsPanel := panel("Filesystems", fsContent, leftW)

	// Disk I/O + Network side by side
	halfW := (leftW - 1) / 2
	diskIOContent := m.viewDiskIO()
	netContent := m.viewNetwork(sparkW)
	diskIOPanel := panel("Disk I/O", diskIOContent, halfW)
	netPanel := panel("Network", netContent, halfW)
	diskNetRow := lipgloss.JoinHorizontal(lipgloss.Top, diskIOPanel, " ", netPanel)

	// Greeter panel — lives in the left column to fill the space below system stats
	// Give greeter more rain rows at larger terminal heights so the left
	// column fills space and the column-equalizer has less work to do.
	rainRows := 14 + (m.height-40)/3
	if rainRows < 14 {
		rainRows = 14
	}
	greeterContent := viewGreeter(leftW-6, m.animFrame, rainRows)
	greeterPanel := panel("", greeterContent, leftW)

	// System Info panel
	sysInfoContent := m.viewSysInfo()
	sysInfoPanel := panel("System", sysInfoContent, leftW)

	leftCol := lipgloss.JoinVertical(lipgloss.Left,
		sysInfoPanel,
		cpuPanel,
		memPanel,
		fsPanel,
		diskNetRow,
		greeterPanel,
	)

	// === RIGHT COLUMN PANELS ===

	// Processes Panel
	procsTitle := fmt.Sprintf("Processes [%s]", m.cfg.sortBy)
	if m.filterInput != "" {
		procsTitle = fmt.Sprintf("Processes [%s] /%s", m.cfg.sortBy, m.filterInput)
	}
	procsContent := m.viewProcesses(rightW - 4)
	procsPanel := panel(procsTitle, procsContent, rightW)

	// Events Panel
	eventsContent := m.viewEvents(rightW - 4)
	eventsPanel := panel("Events", eventsContent, rightW)

	// Temperature Panel (top 5)
	tempContent := m.viewTemps()
	tempPanel := panel("Temperature", tempContent, rightW)

	// Battery + Docker side by side
	batHalfW := (rightW - 1) / 3
	dockHalfW := rightW - batHalfW - 1
	batContent := m.viewBattery()
	batPanel := panel("Battery", batContent, batHalfW)
	dockerContent := m.viewDocker(dockHalfW - 4)
	dockerPanel := panel("Docker", dockerContent, dockHalfW)
	batDockRow := lipgloss.JoinHorizontal(lipgloss.Top, batPanel, " ", dockerPanel)

	sceneContent := viewScene(rightW-4, m.animFrame)
	scenePanel := panel("", sceneContent, rightW)

	rightCol := lipgloss.JoinVertical(lipgloss.Left,
		procsPanel,
		eventsPanel,
		tempPanel,
		batDockRow,
		scenePanel,
	)

	// === JOIN COLUMNS ===
	// Equalize column heights so neither leaves a blank strip beside the other.
	leftLines := strings.Count(leftCol, "\n") + 1
	rightLines := strings.Count(rightCol, "\n") + 1
	if rightLines > leftLines {
		leftCol = lipgloss.NewStyle().Height(rightLines).Render(leftCol)
	} else if leftLines > rightLines {
		rightCol = lipgloss.NewStyle().Height(leftLines).Render(rightCol)
	}
	body := lipgloss.JoinHorizontal(lipgloss.Top, leftCol, " ", rightCol)

	// === STATUS BAR ===
	hostname := m.stats.hostname
	if hostname == "" {
		hostname = "unknown"
	}
	uptimeStr := formatUptime(m.stats.uptime)
	intervalStr := m.cfg.interval.String()

	usersStr := ""
	if len(m.stats.users) > 0 {
		usersStr = fmt.Sprintf(" │ %d user(s)", len(m.stats.users))
	}
	cpuMiniSpark := trimSpark(sparkline(m.cpuHist.values()), 10)
	statusLeft := statusBarStyle.Render(fmt.Sprintf(" %s │ up %s │ refresh %s%s │ %s",
		hostname, uptimeStr, intervalStr, usersStr, cpuMiniSpark))
	statusRight := statusBarStyle.Render(
		statusKeyStyle.Render("q") + statusBarStyle.Render(" quit  ") +
			statusKeyStyle.Render("x") + statusBarStyle.Render(" kill  ") +
			statusKeyStyle.Render("j/k") + statusBarStyle.Render(" nav  ") +
			statusKeyStyle.Render("/") + statusBarStyle.Render(" filter  ") +
			statusKeyStyle.Render("?") + statusBarStyle.Render(" help "))

	gap := w - lipgloss.Width(statusLeft) - lipgloss.Width(statusRight)
	if gap < 0 {
		gap = 0
	}
	statusBar := statusBarStyle.Width(w).Render(
		statusLeft + strings.Repeat(" ", gap) + statusRight)

	killLine := ""
	if m.killMsg != "" {
		killLine = "\n" + yellowStyle.Render("  "+m.killMsg)
	}

	return titleBar + "\n" + body + "\n" + statusBar + killLine
}

// View helpers — each returns the inner content for a panel.

func (m model) viewCPU(barW, sparkW int) string {
	var b strings.Builder

	// Waveform history chart.
	wave := cpuWaveform(m.cpuHist.values(), sparkW, 6)
	if wave != "" {
		b.WriteString(wave + "\n")
	}

	spark := trimSpark(sparkline(m.cpuHist.values()), sparkW)

	// Load average
	loadColor := func(l float64, cores int) lipgloss.Style {
		ratio := l / float64(cores)
		if ratio > 1.0 {
			return redStyle
		} else if ratio > 0.7 {
			return yellowStyle
		}
		return greenStyle
	}
	cores := m.stats.cpuCores
	if cores == 0 {
		cores = 1
	}
	loadLine := fmt.Sprintf("  Load  %s  %s  %s",
		loadColor(m.stats.load1, cores).Render(fmt.Sprintf("%.2f", m.stats.load1)),
		loadColor(m.stats.load5, cores).Render(fmt.Sprintf("%.2f", m.stats.load5)),
		loadColor(m.stats.load15, cores).Render(fmt.Sprintf("%.2f", m.stats.load15)))
	loadLine += dimStyle.Render(fmt.Sprintf("  (%d cores)", cores))
	b.WriteString(loadLine + "\n")

	overall := fmt.Sprintf("Overall  %s %s  %s",
		bar(m.stats.cpuOverall, barW),
		pctColor(m.stats.cpuOverall).Render(fmt.Sprintf("%5.1f%%", m.stats.cpuOverall)),
		dimStyle.Render(spark))
	if m.cpuAlert {
		overall = alertStyle.Render(overall)
	}
	b.WriteString(overall)

	// Per-core: clean 2-column list, physical cores only (every other core on hyperthreaded systems)
	perCore := m.stats.cpuPerCore
	physicalCores := perCore
	if len(perCore) > 8 {
		// Likely hyperthreaded — show only even-indexed (physical) cores
		physicalCores = make([]float64, 0, len(perCore)/2)
		for i := 0; i < len(perCore); i += 2 {
			physicalCores = append(physicalCores, perCore[i])
		}
	}
	maxCores := 8
	if len(physicalCores) > maxCores {
		physicalCores = physicalCores[:maxCores]
	}

	// Two columns
	half := (len(physicalCores) + 1) / 2
	for row := 0; row < half; row++ {
		b.WriteString("\n")
		for col := 0; col < 2; col++ {
			idx := row + col*half
			if idx >= len(physicalCores) {
				break
			}
			pct := physicalCores[idx]
			miniSpark := trimSpark(sparkline([]float64{pct}), 1)
			b.WriteString(fmt.Sprintf("  %s  %s %s %s",
				dimStyle.Render(fmt.Sprintf("Core %-2d", idx)),
				dimStyle.Render(miniSpark),
				bar(pct, 8),
				pctColor(pct).Render(fmt.Sprintf("%5.1f%%", pct))))
		}
	}
	if len(perCore) > len(physicalCores)*2 {
		b.WriteString(fmt.Sprintf("\n  %s", dimStyle.Render(fmt.Sprintf("+%d logical cores hidden", len(perCore)-len(physicalCores)))))
	}
	return b.String()
}

func (m model) viewMemory(barW, sparkW int) string {
	var b strings.Builder
	memSpark := trimSpark(sparkline(m.memHist.values()), sparkW)
	swapSpark := trimSpark(sparkline(m.swapHist.values()), sparkW)

	ramLine := fmt.Sprintf("RAM  %s %5.1f%% %s/%s %s",
		bar(m.stats.memPercent, barW), m.stats.memPercent,
		formatBytes(m.stats.memUsed), formatBytes(m.stats.memTotal),
		dimStyle.Render(memSpark))
	if m.ramAlert {
		ramLine = alertStyle.Render(ramLine)
	}
	b.WriteString(ramLine)
	b.WriteString(fmt.Sprintf("\nSwap %s %5.1f%% %s/%s %s",
		bar(m.stats.swapPercent, barW), m.stats.swapPercent,
		formatBytes(m.stats.swapUsed), formatBytes(m.stats.swapTotal),
		dimStyle.Render(swapSpark)))
	return b.String()
}

func (m model) viewSysInfo() string {
	var b strings.Builder

	// CPU model
	if m.stats.cpuModel != "" {
		model := truncate(m.stats.cpuModel, 45)
		b.WriteString(fmt.Sprintf("  %s  %s", dimStyle.Render("CPU"), valueStyle.Render(model)))
		if m.stats.cpuFreqMHz > 0 {
			b.WriteString(dimStyle.Render(fmt.Sprintf(" @ %.0f MHz", m.stats.cpuFreqMHz)))
		}
		b.WriteString(dimStyle.Render(fmt.Sprintf("  %dc/%dt", m.stats.cpuCores, m.stats.cpuThreads)))
	}

	// OS / Kernel
	if m.stats.platform != "" {
		b.WriteString(fmt.Sprintf("\n  %s  %s %s",
			dimStyle.Render("OS "),
			valueStyle.Render(m.stats.platform),
			dimStyle.Render(m.stats.platformVer)))
	}
	if m.stats.kernelVer != "" {
		b.WriteString(fmt.Sprintf("  %s %s",
			dimStyle.Render("kernel"),
			dimStyle.Render(truncate(m.stats.kernelVer, 25))))
	}
	if m.stats.kernelArch != "" {
		b.WriteString(dimStyle.Render(" " + m.stats.kernelArch))
	}

	// Virtualization
	if m.stats.virtSystem != "" {
		b.WriteString(fmt.Sprintf("\n  %s  %s",
			dimStyle.Render("VM "),
			yellowStyle.Render(m.stats.virtSystem+" ("+m.stats.virtRole+")")))
	}

	return b.String()
}

func (m model) viewFilesystems(colW int) string {
	if len(m.stats.filesystems) == 0 {
		spark := trimSpark(sparkline(m.diskHist.values()), 15)
		return fmt.Sprintf("  %s %5.1f%%  %s/%s  %s",
			bar(m.stats.diskPercent, 15), m.stats.diskPercent,
			formatBytes(m.stats.diskUsed), formatBytes(m.stats.diskTotal),
			dimStyle.Render(spark))
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("  %s %s %s %s %s\n",
		headerStyle.Render(rightPad("MOUNT", 14)),
		headerStyle.Render(rightPad("TYPE", 6)),
		headerStyle.Render(rightPad("SIZE", 7)),
		headerStyle.Render(rightPad("USED%", 18)),
		headerStyle.Render("FREE")))

	maxFs := 6
	if len(m.stats.filesystems) < maxFs {
		maxFs = len(m.stats.filesystems)
	}
	for i := 0; i < maxFs; i++ {
		fs := m.stats.filesystems[i]
		free := fs.total - fs.used
		row := fmt.Sprintf("  %s %s %s %s %s  %s",
			valueStyle.Render(rightPad(truncate(fs.mount, 14), 14)),
			dimStyle.Render(rightPad(truncate(fs.fstype, 6), 6)),
			dimStyle.Render(rightPad(formatBytes(fs.total), 7)),
			bar(fs.percent, 10),
			pctColor(fs.percent).Render(fmt.Sprintf("%5.1f%%", fs.percent)),
			dimStyle.Render(formatBytes(free)))
		if i%2 == 0 {
			row = rowEvenStyle.Render(row)
		}
		b.WriteString(row + "\n")
	}
	if len(m.stats.filesystems) > maxFs {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  +%d more", len(m.stats.filesystems)-maxFs)))
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m model) viewDiskIO() string {
	readStr := formatBytesPerSec(m.stats.diskReadSpeed)
	writeStr := formatBytesPerSec(m.stats.diskWriteSpeed)

	return fmt.Sprintf("%s  %s\n%s  %s",
		dimStyle.Render("R"),
		greenStyle.Render(rightPad(readStr, 10)),
		dimStyle.Render("W"),
		yellowStyle.Render(rightPad(writeStr, 10)))
}

func (m model) viewNetwork(sparkW int) string {
	upSpark := trimSpark(sparkline(m.netUpHist.values()), sparkW/2)
	downSpark := trimSpark(sparkline(m.netDownHist.values()), sparkW/2)

	upPct := float64(m.stats.netUp) / (100 * 1024 * 1024) * 100
	downPct := float64(m.stats.netDown) / (100 * 1024 * 1024) * 100
	if upPct > 100 {
		upPct = 100
	}
	if downPct > 100 {
		downPct = 100
	}

	connStr := dimStyle.Render(fmt.Sprintf("  %d conns", m.stats.netConns))

	// Heatmap strip — uses the same width cap as sparklines.
	heatW := sparkW / 2
	if heatW < 4 {
		heatW = 4
	}
	heatmap := netHeatmap(m.netUpHist.values(), m.netDownHist.values(), heatW)

	return fmt.Sprintf("%s\n%s %s  %s\n  %s  %s\n\n%s %s  %s\n  %s  %s\n%s",
		heatmap,
		greenStyle.Render("↑ Up  "),
		valueStyle.Render(rightPad(formatBytesPerSec(m.stats.netUp), 10)),
		dimStyle.Render(upSpark),
		bar(upPct, 18),
		pctColor(upPct).Render(fmt.Sprintf("%5.1f%%", upPct)),
		greenStyle.Render("↓ Down"),
		valueStyle.Render(rightPad(formatBytesPerSec(m.stats.netDown), 10)),
		dimStyle.Render(downSpark),
		bar(downPct, 18),
		pctColor(downPct).Render(fmt.Sprintf("%5.1f%%", downPct)),
		connStr)
}

func (m model) viewBattery() string {
	if m.stats.battery == nil {
		return dimStyle.Render("  no battery detected (desktop)")
	}
	pct := m.stats.battery.percent
	icon := "🔋"
	status := "discharging"
	if m.stats.battery.pluggedIn {
		icon = "⚡"
		status = "charging"
	}
	if pct >= 100 {
		status = "full"
	}
	return fmt.Sprintf("  %s %s %s  %s",
		icon,
		bar(pct, 15),
		pctColor(100-pct).Render(fmt.Sprintf("%.0f%%", pct)),
		dimStyle.Render(status))
}

func (m model) viewTemps() string {
	if len(m.stats.temps) == 0 {
		return dimStyle.Render("not available")
	}
	// Sort by temperature descending, show top 5
	sorted := make([]tempInfo, len(m.stats.temps))
	copy(sorted, m.stats.temps)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].temp > sorted[j].temp })
	n := 5
	if len(sorted) < n {
		n = len(sorted)
	}
	var b strings.Builder
	for i := 0; i < n; i++ {
		t := sorted[i]
		style := greenStyle
		if t.temp > 70 {
			style = redStyle
		} else if t.temp >= 50 {
			style = yellowStyle
		}
		name := truncate(t.label, 20)
		temp := style.Render(fmt.Sprintf("%5.0f°C", t.temp))
		b.WriteString(fmt.Sprintf("  %s%s",
			dimStyle.Render(rightPad(name, 22)),
			temp))
		if i < n-1 {
			b.WriteString("\n")
		}
	}
	if len(sorted) > 5 {
		b.WriteString(dimStyle.Render(fmt.Sprintf("\n  +%d sensors", len(sorted)-5)))
	}
	return b.String()
}

func (m model) viewDocker(colW int) string {
	if len(m.docker) == 0 {
		return dimStyle.Render("  no containers running")
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("  %s %s %s %s\n",
		headerStyle.Render(rightPad("NAME", 16)),
		headerStyle.Render(rightPad("STATUS", 12)),
		headerStyle.Render(rightPad("CPU%", 6)),
		headerStyle.Render(rightPad("MEM", 8))))
	for i, c := range m.docker {
		// Status icon
		icon := dimStyle.Render("●")
		if strings.Contains(strings.ToLower(c.status), "up") {
			icon = greenStyle.Render("●")
		}
		row := fmt.Sprintf("  %s %s %s %s %s",
			icon,
			valueStyle.Render(rightPad(truncate(c.name, 15), 15)),
			dimStyle.Render(rightPad(truncate(c.status, 11), 11)),
			pctColor(c.cpu).Render(fmt.Sprintf("%5.1f%%", c.cpu)),
			dimStyle.Render(fmt.Sprintf("%6.1fM", c.memMB)))
		if i%2 == 0 {
			row = rowEvenStyle.Render(row)
		}
		b.WriteString(row + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m model) viewProcesses(colW int) string {
	var b strings.Builder

	// Filter input bar.
	if m.filterMode {
		cursor := "█"
		if m.animFrame%4 >= 2 {
			cursor = " "
		}
		b.WriteString(lipgloss.NewStyle().Foreground(titleColor).Render("  / ") +
			valueStyle.Render(m.filterInput+cursor) + "\n")
	}

	b.WriteString(fmt.Sprintf("  %s %s %s %s\n",
		headerStyle.Render(rightPad("PID", 7)),
		headerStyle.Render(rightPad("NAME", 20)),
		headerStyle.Render(rightPad("CPU%", 7)),
		headerStyle.Render(rightPad("MEM%", 7))))

	fp := m.filteredProcs()
	for i, p := range fp {
		marker := "  "
		if i == m.procCursor {
			marker = procMarkerStyle.Render("> ")
		}
		row := fmt.Sprintf("%s%s %s %s %s",
			marker,
			dimStyle.Render(rightPad(fmt.Sprintf("%d", p.pid), 7)),
			valueStyle.Render(rightPad(truncate(p.name, 20), 20)),
			pctColor(float64(p.cpu)).Render(fmt.Sprintf("%5.1f%%", p.cpu)),
			pctColor(float64(p.mem)*10).Render(fmt.Sprintf("%5.1f%%", p.mem)))

		if i == m.procCursor {
			row = rowSelectedStyle.Render(row)
		} else if i%2 == 0 {
			row = rowEvenStyle.Render(row)
		} else {
			row = rowOddStyle.Render(row)
		}
		b.WriteString(row + "\n")
	}
	if len(fp) == 0 {
		b.WriteString("  " + dimStyle.Render("no matches") + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// trimSpark trims a sparkline to at most n runes.
func trimSpark(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[len(runes)-n:])
}

// Commands

func tickCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func fetchStatsCmd() tea.Msg {
	return statsMsg(collectStats())
}

func fetchProcsCmd(n int, sortBy string) tea.Cmd {
	return func() tea.Msg {
		return procsMsg(collectProcesses(n, sortBy))
	}
}

func fetchDockerCmd() tea.Msg {
	return dockerMsg(collectDocker())
}

func animTickCmd() tea.Cmd {
	return tea.Tick(150*time.Millisecond, func(t time.Time) tea.Msg {
		return animTickMsg(t)
	})
}

func killCmd(pid int32) tea.Cmd {
	return func() tea.Msg {
		return killResultMsg{err: killProcess(pid)}
	}
}

func formatUptime(secs uint64) string {
	d := secs / 86400
	h := (secs % 86400) / 3600
	m := (secs % 3600) / 60
	if d > 0 {
		return fmt.Sprintf("%dd %dh %dm", d, h, m)
	}
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}
