//go:build !darwin && !windows

package main

func openSystemOutput() *output { return nil }
