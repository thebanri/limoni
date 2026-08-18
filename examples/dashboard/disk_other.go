//go:build !unix && !darwin && !linux && !windows

package main

func getDiskSpace() (usedGB, totalGB, percent float64) {
	return 250.0, 500.0, 50.0
}
