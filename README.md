# flawed

A terminal system monitor written in Go. Tracks CPU, memory, filesystems, disk I/O, network, processes, Docker containers, temperatures, and battery — all in a live TUI.

![demo](ascii-art%20(1).gif)

## Features

- **CPU** — overall usage with sparkline history, per-core breakdown, and load averages
- **Memory** — RAM and swap with sparkline history
- **Filesystems** — all mounted partitions with usage bars
- **Disk I/O** — real-time read/write throughput
- **Network** — upload/download speed with sparklines and active connection count
- **Processes** — top N by CPU or memory, with live kill support
- **Docker** — running containers with CPU/memory stats (requires Docker)
- **Temperature** — top 5 hottest hardware sensors
- **Battery** — charge level and charging state
- **Alerts** — pulsing title bar when CPU or RAM exceed configurable thresholds
- **Log** — CSV snapshot appended to `~/.flawed/log.csv` on every refresh

## Installation

Build from source:

```bash
git clone https://github.com/ruskaruma/flawed
cd flawed
go build -o flawed .
```

## Usage

```
flawed [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--interval`, `-i` | `2s` | Refresh interval |
| `--procs`, `-n` | `10` | Number of top processes to show |
| `--sort` | `cpu` | Initial sort order: `cpu` or `mem` |
| `--alert-cpu` | `85` | CPU alert threshold (%) |
| `--alert-ram` | `90` | RAM alert threshold (%) |
| `--verbose`, `-v` | `false` | Print stats to stderr on each refresh |
| `--once` | `false` | Print one snapshot and exit (non-interactive) |

## Keybindings

| Key | Action |
|-----|--------|
| `q` / `Ctrl+C` | Quit |
| `k` | Kill the top process in the list |
| `s` | Toggle sort order (CPU / memory) |

## Log output

On each refresh, flawed appends a row to `~/.flawed/log.csv`:

```
timestamp,cpu_pct,mem_pct,swap_pct,disk_pct,net_up_bytes,net_down_bytes,containers
2024-01-15T10:30:00Z,12.3,45.6,0.0,78.9,1024,2048,3
```

## Requirements

- Go 1.24+
- Linux or macOS (Windows: best-effort, battery/temperature may not be available)
- Docker CLI (optional — container panel is hidden when Docker is not running)
