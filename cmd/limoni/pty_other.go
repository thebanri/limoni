//go:build !linux && !darwin

package main

import (
	"os"
	"os/exec"
)

const ptySupported = false

func startInPTY(*exec.Cmd, uint16, uint16) (*os.File, error) { return nil, errNoPTY }
func setPTYSize(*os.File, uint16, uint16) error              { return errNoPTY }
func hangUp(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}
func killSession(cmd *exec.Cmd) { hangUp(cmd) }
