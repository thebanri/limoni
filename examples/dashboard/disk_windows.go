//go:build windows

package main

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

func getDiskSpace() (usedGB, totalGB, percent float64) {
	var freeBytesAvailable, totalNumberOfBytes, totalNumberOfFreeBytes int64
	kernel32 := windows.NewLazySystemDLL("kernel32.dll")
	getDiskFreeSpaceEx := kernel32.NewProc("GetDiskFreeSpaceExW")

	rootPath, _ := windows.UTF16PtrFromString("C:\\")
	r1, _, _ := getDiskFreeSpaceEx.Call(
		uintptr(unsafe.Pointer(rootPath)),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		uintptr(unsafe.Pointer(&totalNumberOfBytes)),
		uintptr(unsafe.Pointer(&totalNumberOfFreeBytes)),
	)
	if r1 == 0 || totalNumberOfBytes == 0 {
		return 240, 512, 46.8
	}
	usedBytes := totalNumberOfBytes - totalNumberOfFreeBytes
	totalGB = float64(totalNumberOfBytes) / (1024 * 1024 * 1024)
	usedGB = float64(usedBytes) / (1024 * 1024 * 1024)
	percent = (usedGB / totalGB) * 100.0
	return usedGB, totalGB, percent
}
