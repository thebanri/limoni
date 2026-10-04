//go:build windows

package driver

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// releaseChild is the child side of TestReleaseHandsTheConsoleToTheProgram:
// a backend on the console it was given, which hands the console to a
// stand-in editor and then waits for one key.
func releaseChild() {
	b := NewBackend(os.Stdin, os.Stdout)
	if err := b.Setup(); err != nil {
		fmt.Printf("SETUP[%v]", err)
		os.Exit(2)
	}
	b.StartEventLoop()
	fmt.Print("READY")
	err := b.Release(func() error {
		fmt.Print("EDITOR>")
		// An editor reads at its own pace: the line is typed and waits in
		// the console's queue before this one asks for it. A reader that
		// was not stopped takes it in the meantime.
		time.Sleep(500 * time.Millisecond)
		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		fmt.Printf("GOT[%s]", strings.TrimSpace(line))
		return err
	})
	if err != nil {
		fmt.Printf("RELEASE[%v]", err)
		os.Exit(3)
	}
	fmt.Print("BACK")
	for {
		select {
		case ev := <-b.Events():
			if ev.Type == EventKey && ev.Key.Ch == 'x' {
				fmt.Print("EVENT[x]")
				_ = b.Close()
				os.Exit(0)
			}
		case <-time.After(15 * time.Second):
			fmt.Print("TIMEOUT")
			os.Exit(4)
		}
	}
}

func TestMain(m *testing.M) {
	if os.Getenv("LIMONI_RELEASE_CHILD") == "1" {
		releaseChild()
	}
	os.Exit(m.Run())
}

// On a Windows console, what is typed while the console belongs to another
// program is that program's: the event loop must not read it. A reader
// blocked in ReadConsole takes the first keys typed into the editor, which
// is why Release used to be refused on Windows. Afterwards keys reach the
// application again. The child runs in a pseudo console, as it would in
// Windows Terminal; a CI runner's own standard input is a pipe.
func TestReleaseHandsTheConsoleToTheProgram(t *testing.T) {
	t.Setenv("LIMONI_RELEASE_CHILD", "1")
	t.Setenv("LIMONI_PROBE", "0")

	inR, inW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	var pc windows.Handle
	if err := windows.CreatePseudoConsole(windows.Coord{X: 100, Y: 30}, windows.Handle(inR.Fd()), windows.Handle(outW.Fd()), 0, &pc); err != nil {
		t.Skipf("no pseudo console: %v", err)
	}
	closed := false
	closeConsole := func() {
		if !closed {
			closed = true
			windows.ClosePseudoConsole(pc)
		}
	}
	defer closeConsole()

	attrs, err := windows.NewProcThreadAttributeList(1)
	if err != nil {
		t.Fatal(err)
	}
	defer attrs.Delete()
	// The attribute's value is the pseudo console handle itself.
	if err := attrs.Update(windows.PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE, *(*unsafe.Pointer)(unsafe.Pointer(&pc)), unsafe.Sizeof(pc)); err != nil {
		t.Fatal(err)
	}
	var si windows.StartupInfoEx
	si.Cb = uint32(unsafe.Sizeof(si))
	// No standard handles of ours: the child's are the pseudo console's.
	si.Flags = windows.STARTF_USESTDHANDLES
	si.ProcThreadAttributeList = attrs.List()
	cmdline := windows.ComposeCommandLine([]string{os.Args[0], "-test.run=^$"})
	var pi windows.ProcessInformation
	if err := windows.CreateProcess(nil, windows.StringToUTF16Ptr(cmdline), nil, nil, false,
		windows.EXTENDED_STARTUPINFO_PRESENT, nil, nil, &si.StartupInfo, &pi); err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(pi.Process)
	defer windows.CloseHandle(pi.Thread)
	inR.Close()
	outW.Close()

	var mu sync.Mutex
	var out strings.Builder
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := outR.Read(buf)
			mu.Lock()
			out.Write(buf[:n])
			mu.Unlock()
			if err != nil {
				return
			}
		}
	}()
	output := func() string { mu.Lock(); defer mu.Unlock(); return out.String() }
	waitFor := func(marker string) {
		t.Helper()
		deadline := time.Now().Add(15 * time.Second)
		for !strings.Contains(output(), marker) {
			if time.Now().After(deadline) {
				t.Fatalf("no %q from the child; it wrote %q", marker, output())
			}
			time.Sleep(20 * time.Millisecond)
		}
	}

	waitFor("EDITOR>")
	// Typed into the editor: every character must reach it.
	if _, err := inW.Write([]byte("typed into the editor\r")); err != nil {
		t.Fatal(err)
	}
	waitFor("BACK")
	if !strings.Contains(output(), "GOT[typed into the editor]") {
		t.Fatalf("the program did not get its line whole: the event loop took input; output %q", output())
	}
	if _, err := inW.Write([]byte("x")); err != nil {
		t.Fatal(err)
	}
	waitFor("EVENT[x]")
	if ev, _ := windows.WaitForSingleObject(pi.Process, 10000); ev != windows.WAIT_OBJECT_0 {
		t.Fatal("the child did not exit")
	}
	var code uint32
	_ = windows.GetExitCodeProcess(pi.Process, &code)
	if code != 0 {
		t.Fatalf("the child exited with %d; output %q", code, output())
	}
	closeConsole()
	inW.Close()
}
