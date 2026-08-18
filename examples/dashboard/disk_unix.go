//go:build unix || darwin || linux

package main

import "syscall"

func getDiskSpace() (usedGB, totalGB, percent float64) {
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
