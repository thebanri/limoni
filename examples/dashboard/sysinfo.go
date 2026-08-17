package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// SystemMetrics contains live hardware and OS telemetry.
type SystemMetrics struct {
	CPUPercent     float64
	RAMUsedMB      float64
	RAMTotalMB     float64
	RAMPercent     float64
	DiskUsedGB     float64
	DiskTotalGB    float64
	DiskPercent    float64
	NetRxRateMB    float64
	NetTxRateMB    float64
	TotalRxMB      float64
	TotalTxMB      float64
	UptimeStr      string
	Hostname       string
	PlatformInfo   string
	CoreCount      int
}

// SysCollector collects real system statistics on Linux with fallback on other OSes.
type SysCollector struct {
	lastCPUTotal uint64
	lastCPUIdle  uint64
	lastNetRx    uint64
	lastNetTx    uint64
	lastNetTime  time.Time
	pageSize     uint64
	prevProcCPUs map[int]procCPUTime
	lastProcTime time.Time
}

type procCPUTime struct {
	utime uint64
	stime uint64
}

// NewSysCollector creates a new telemetry collector.
func NewSysCollector() *SysCollector {
	sc := &SysCollector{
		pageSize:     uint64(os.Getpagesize()),
		prevProcCPUs: make(map[int]procCPUTime),
		lastNetTime:  time.Now(),
		lastProcTime: time.Now(),
	}
	sc.sampleCPU()
	sc.sampleNet()
	return sc
}

func (sc *SysCollector) sampleCPU() float64 {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0
	}

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "cpu ") {
			fields := strings.Fields(line)
			if len(fields) < 5 {
				return 0
			}

			var total, idle uint64
			for i := 1; i < len(fields); i++ {
				val, _ := strconv.ParseUint(fields[i], 10, 64)
				total += val
				if i == 4 || i == 5 { // idle and iowait
					idle += val
				}
			}

			if sc.lastCPUTotal == 0 {
				sc.lastCPUTotal = total
				sc.lastCPUIdle = idle
				return 0
			}

			deltaTotal := total - sc.lastCPUTotal
			deltaIdle := idle - sc.lastCPUIdle
			sc.lastCPUTotal = total
			sc.lastCPUIdle = idle

			if deltaTotal == 0 {
				return 0
			}

			usage := 100.0 * (1.0 - float64(deltaIdle)/float64(deltaTotal))
			if usage < 0 {
				usage = 0
			}
			if usage > 100 {
				usage = 100
			}
			return usage
		}
	}
	return 0
}

func (sc *SysCollector) sampleRAM() (usedMB, totalMB, percent float64) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		var mem runtime.MemStats
		runtime.ReadMemStats(&mem)
		used := float64(mem.Alloc) / (1024 * 1024)
		total := float64(mem.Sys) / (1024 * 1024)
		if total == 0 {
			total = 1024
		}
		return used, total, (used / total) * 100
	}

	var memTotalKB, memAvailableKB, memFreeKB, buffersKB, cachedKB uint64
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		key := strings.TrimSuffix(fields[0], ":")
		val, _ := strconv.ParseUint(fields[1], 10, 64)

		switch key {
		case "MemTotal":
			memTotalKB = val
		case "MemAvailable":
			memAvailableKB = val
		case "MemFree":
			memFreeKB = val
		case "Buffers":
			buffersKB = val
		case "Cached":
			cachedKB = val
		}
	}

	if memTotalKB == 0 {
		return 0, 0, 0
	}

	var usedKB uint64
	if memAvailableKB > 0 {
		if memTotalKB > memAvailableKB {
			usedKB = memTotalKB - memAvailableKB
		}
	} else {
		usedKB = memTotalKB - (memFreeKB + buffersKB + cachedKB)
	}

	totalMB = float64(memTotalKB) / 1024.0
	usedMB = float64(usedKB) / 1024.0
	percent = (usedMB / totalMB) * 100.0
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	return usedMB, totalMB, percent
}

func (sc *SysCollector) sampleDisk() (usedGB, totalGB, percent float64) {
	var stat syscall.Statfs_t
	err := syscall.Statfs("/", &stat)
	if err != nil {
		return 0, 0, 0
	}

	totalBytes := stat.Blocks * uint64(stat.Bsize)
	freeBytes := stat.Bavail * uint64(stat.Bsize)
	if totalBytes == 0 {
		return 0, 0, 0
	}
	usedBytes := totalBytes - freeBytes

	totalGB = float64(totalBytes) / (1024 * 1024 * 1024)
	usedGB = float64(usedBytes) / (1024 * 1024 * 1024)
	percent = (usedGB / totalGB) * 100.0
	return usedGB, totalGB, percent
}

func (sc *SysCollector) sampleNet() (rxRateMB, txRateMB, totalRxMB, totalTxMB float64) {
	data, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return 0, 0, 0, 0
	}

	var curRx, curTx uint64
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, ":") {
			continue
		}
		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			continue
		}
		iface := strings.TrimSpace(parts[0])
		if iface == "lo" {
			continue // ignore loopback
		}
		fields := strings.Fields(parts[1])
		if len(fields) >= 9 {
			rx, _ := strconv.ParseUint(fields[0], 10, 64)
			tx, _ := strconv.ParseUint(fields[8], 10, 64)
			curRx += rx
			curTx += tx
		}
	}

	now := time.Now()
	elapsed := now.Sub(sc.lastNetTime).Seconds()
	if elapsed <= 0 {
		elapsed = 1.0
	}

	if sc.lastNetRx > 0 && curRx >= sc.lastNetRx {
		rxRateMB = float64(curRx-sc.lastNetRx) / elapsed / (1024 * 1024)
	}
	if sc.lastNetTx > 0 && curTx >= sc.lastNetTx {
		txRateMB = float64(curTx-sc.lastNetTx) / elapsed / (1024 * 1024)
	}

	sc.lastNetRx = curRx
	sc.lastNetTx = curTx
	sc.lastNetTime = now

	totalRxMB = float64(curRx) / (1024 * 1024)
	totalTxMB = float64(curTx) / (1024 * 1024)

	return rxRateMB, txRateMB, totalRxMB, totalTxMB
}

func (sc *SysCollector) sampleUptime() string {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return "N/A"
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return "N/A"
	}
	secFloat, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return "N/A"
	}
	totalSec := int(secFloat)
	days := totalSec / 86400
	hours := (totalSec % 86400) / 3600
	mins := (totalSec % 3600) / 60
	secs := totalSec % 60

	if days > 0 {
		return fmt.Sprintf("%dd %02dh %02dm", days, hours, mins)
	}
	return fmt.Sprintf("%02dh %02dm %02ds", hours, mins, secs)
}

// CollectMetrics returns the latest snapshot of system health.
func (sc *SysCollector) CollectMetrics() SystemMetrics {
	cpu := sc.sampleCPU()
	usedRAM, totalRAM, ramPct := sc.sampleRAM()
	usedDisk, totalDisk, diskPct := sc.sampleDisk()
	rxRate, txRate, totRx, totTx := sc.sampleNet()
	uptime := sc.sampleUptime()

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "localhost"
	}

	return SystemMetrics{
		CPUPercent:   cpu,
		RAMUsedMB:    usedRAM,
		RAMTotalMB:   totalRAM,
		RAMPercent:   ramPct,
		DiskUsedGB:   usedDisk,
		DiskTotalGB:  totalDisk,
		DiskPercent:  diskPct,
		NetRxRateMB:  rxRate,
		NetTxRateMB:  txRate,
		TotalRxMB:    totRx,
		TotalTxMB:    totTx,
		UptimeStr:    uptime,
		Hostname:     hostname,
		PlatformInfo: fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		CoreCount:    runtime.NumCPU(),
	}
}

// CollectProcesses reads real running processes from /proc.
func (sc *SysCollector) CollectProcesses() []Process {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}

	now := time.Now()
	deltaSec := now.Sub(sc.lastProcTime).Seconds()
	if deltaSec <= 0 {
		deltaSec = 1.0
	}
	newProcCPUs := make(map[int]procCPUTime)

	var processes []Process

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}

		pidPath := filepath.Join("/proc", entry.Name())

		// Read process name
		commData, err := os.ReadFile(filepath.Join(pidPath, "comm"))
		name := ""
		if err == nil {
			name = strings.TrimSpace(string(commData))
		}
		if name == "" {
			cmdData, _ := os.ReadFile(filepath.Join(pidPath, "cmdline"))
			parts := bytes.Split(cmdData, []byte{0})
			if len(parts) > 0 && len(parts[0]) > 0 {
				name = filepath.Base(string(parts[0]))
			}
		}
		if name == "" {
			name = fmt.Sprintf("proc_%d", pid)
		}

		// Read /proc/[pid]/stat for CPU & RSS
		statData, err := os.ReadFile(filepath.Join(pidPath, "stat"))
		if err != nil {
			continue
		}
		statStr := string(statData)
		rParen := strings.LastIndex(statStr, ")")
		if rParen == -1 || rParen+2 >= len(statStr) {
			continue
		}
		rest := strings.Fields(statStr[rParen+2:])
		if len(rest) < 22 {
			continue
		}

		stateChar := rest[0]
		status := "Sleeping"
		switch stateChar {
		case "R":
			status = "Running"
		case "S":
			status = "Sleeping"
		case "D":
			status = "Disk IO"
		case "Z":
			status = "Zombie"
		case "T":
			status = "Stopped"
		case "I":
			status = "Idle"
		}

		utime, _ := strconv.ParseUint(rest[11], 10, 64)
		stime, _ := strconv.ParseUint(rest[12], 10, 64)
		rssPages, _ := strconv.ParseUint(rest[21], 10, 64)

		memMB := float64(rssPages*sc.pageSize) / (1024.0 * 1024.0)

		newProcCPUs[pid] = procCPUTime{utime: utime, stime: stime}

		cpuPercent := 0.0
		if prev, ok := sc.prevProcCPUs[pid]; ok {
			totalTicks := (utime + stime) - (prev.utime + prev.stime)
			// User clock ticks per second (usually 100 on Linux)
			cpuPercent = (float64(totalTicks) / 100.0) / deltaSec * 100.0
			if cpuPercent > 100.0*float64(runtime.NumCPU()) {
				cpuPercent = 100.0 * float64(runtime.NumCPU())
			}
		}

		processes = append(processes, Process{
			PID:    pid,
			Name:   name,
			CPU:    cpuPercent,
			Memory: memMB,
			Status: status,
		})
	}

	sc.prevProcCPUs = newProcCPUs
	sc.lastProcTime = now

	// Sort by CPU usage descending by default
	sort.Slice(processes, func(i, j int) bool {
		if processes[i].CPU != processes[j].CPU {
			return processes[i].CPU > processes[j].CPU
		}
		return processes[i].Memory > processes[j].Memory
	})

	return processes
}
