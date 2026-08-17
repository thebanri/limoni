//go:build windows

package backend

import (
	"golang.org/x/sys/windows"
)

// WindowsConsoleState terminalin önceki konsol modlarını saklar.
type WindowsConsoleState struct {
	inHandle  windows.Handle
	outHandle windows.Handle
	inMode    uint32
	outMode   uint32
}

// MakeRaw Windows konsolunu VT100 / Sanal Terminal (Raw) moduna geçirir.
func MakeRaw(inFd, outFd uintptr) (*WindowsConsoleState, error) {
	inHandle := windows.Handle(inFd)
	outHandle := windows.Handle(outFd)

	var inMode, outMode uint32
	if err := windows.GetConsoleMode(inHandle, &inMode); err != nil {
		return nil, err
	}
	if err := windows.GetConsoleMode(outHandle, &outMode); err != nil {
		return nil, err
	}

	state := &WindowsConsoleState{
		inHandle:  inHandle,
		outHandle: outHandle,
		inMode:    inMode,
		outMode:   outMode,
	}

	// Ham giriş modu: Line/Echo/Processed kapat, Virtual Terminal Input ve Window/Mouse Input aç
	rawInMode := inMode &^ (windows.ENABLE_ECHO_INPUT | windows.ENABLE_LINE_INPUT | windows.ENABLE_PROCESSED_INPUT)
	rawInMode |= windows.ENABLE_VIRTUAL_TERMINAL_INPUT | windows.ENABLE_EXTENDED_FLAGS

	// Ham çıkış modu: VT100 Sanal Terminal İşleme aç
	rawOutMode := outMode | windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING | windows.ENABLE_PROCESSED_OUTPUT

	if err := windows.SetConsoleMode(inHandle, rawInMode); err != nil {
		return nil, err
	}
	if err := windows.SetConsoleMode(outHandle, rawOutMode); err != nil {
		_ = windows.SetConsoleMode(inHandle, inMode)
		return nil, err
	}

	return state, nil
}

// Restore terminali eski konsol modlarına döndürür.
func RestoreConsole(state *WindowsConsoleState) error {
	if state == nil {
		return nil
	}
	var lastErr error
	if err := windows.SetConsoleMode(state.inHandle, state.inMode); err != nil {
		lastErr = err
	}
	if err := windows.SetConsoleMode(state.outHandle, state.outMode); err != nil {
		lastErr = err
	}
	return lastErr
}
