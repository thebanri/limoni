package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type processSample struct {
	ticks uint64
	at    time.Time
}

// readLiveProcesses reads Linux /proc process data and computes CPU deltas.
func readLiveProcesses(previous map[string]processSample, now time.Time) ([]ProcessInfo, map[string]processSample) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, previous
	}
	if previous == nil {
		previous = make(map[string]processSample)
	}
	current := make(map[string]processSample, len(entries))
	processes := make([]ProcessInfo, 0, len(entries))
	for _, entry := range entries {
		pid := entry.Name()
		if !entry.IsDir() {
			continue
		}
		if _, err := strconv.Atoi(pid); err != nil {
			continue
		}
		statData, err := os.ReadFile(filepath.Join("/proc", pid, "stat"))
		if err != nil {
			continue
		}
		closeParen := strings.LastIndexByte(string(statData), ')')
		if closeParen < 0 {
			continue
		}
		name := strings.TrimPrefix(string(statData[:closeParen+1]), pid+" ")
		name = strings.Trim(name, "()")
		fields := strings.Fields(string(statData[closeParen+2:]))
		if len(fields) <= 19 {
			continue
		}
		utime, err1 := strconv.ParseUint(fields[11], 10, 64)
		stime, err2 := strconv.ParseUint(fields[12], 10, 64)
		if err1 != nil || err2 != nil {
			continue
		}
		ticks := utime + stime
		current[pid] = processSample{ticks: ticks, at: now}
		cpu := 0.0
		if old, ok := previous[pid]; ok && now.After(old.at) && ticks >= old.ticks {
			// Linux USER_HZ is conventionally 100; this is the same unit used by /proc/stat.
			cpu = float64(ticks-old.ticks) / now.Sub(old.at).Seconds()
		}
		memory := readProcessMemoryMB(pid)
		status := processState(fields[0])
		processes = append(processes, ProcessInfo{PID: pid, Name: name, CPU: fmt.Sprintf("%.1f%%", cpu), Memory: fmt.Sprintf("%.1f MB", memory), Status: status})
	}
	sort.Slice(processes, func(i, j int) bool {
		a, _ := strconv.Atoi(processes[i].PID)
		b, _ := strconv.Atoi(processes[j].PID)
		return a < b
	})
	return processes, current
}

func readProcessMemoryMB(pid string) float64 {
	data, err := os.ReadFile(filepath.Join("/proc", pid, "statm"))
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(data))
	if len(fields) < 2 {
		return 0
	}
	rss, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return 0
	}
	return float64(rss*uint64(os.Getpagesize())) / (1024 * 1024)
}

func processState(state string) string {
	switch state {
	case "R":
		return "Çalışıyor"
	case "S", "D", "I":
		return "Beklemede"
	case "Z":
		return "Zombi"
	case "T", "t":
		return "Durduruldu"
	default:
		return state
	}
}
