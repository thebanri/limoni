//go:build windows

package driver

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ErrReleaseUnsupported is returned by Release where the terminal cannot be
// handed to another program: a portable or remote backend, and an input that
// is not a console, whose reader cannot be stopped.
var ErrReleaseUnsupported = errors.New("limoni: the terminal cannot be handed to another program on this backend")

var (
	kernel32              = windows.NewLazySystemDLL("kernel32.dll")
	procPeekConsoleInputW = kernel32.NewProc("PeekConsoleInputW")
	procReadConsoleInputW = kernel32.NewProc("ReadConsoleInputW")
)

// inputRecord is INPUT_RECORD with its union read as KEY_EVENT_RECORD, the
// only kind looked into.
type inputRecord struct {
	eventType       uint16
	_               uint16
	keyDown         int32
	repeatCount     uint16
	virtualKeyCode  uint16
	virtualScanCode uint16
	unicodeChar     uint16
	controlKeyState uint32
}

// consoleReader reads the console while the application owns it, and stops
// reading while Release has handed it to another program, as ttyReader does
// on Unix. A reader blocked in ReadConsole would take the first keys typed
// into the editor, so it waits on the console handle and a wake-up event,
// and reads only when the console holds text and nobody else owns it.
//
// The console handle is signalled by any input record, not only by text: a
// key released, Shift on its own, focus and size changes. ReadConsole skips
// those and blocks until text comes, so before reading, the queue is looked
// into, and when it holds no text its records are taken out instead.
type consoleReader struct {
	in      *os.File
	h       windows.Handle
	wake    windows.Handle // auto-reset event
	mu      sync.Mutex     // held from the paused check to the end of the read
	resumed chan struct{}  // non-nil while paused; closed on resume
	records []inputRecord
}

// newConsoleReader returns a reader for in, or an error when in is not a
// console (a pipe, a file).
func newConsoleReader(in *os.File) (*consoleReader, error) {
	h := windows.Handle(in.Fd())
	var mode uint32
	if err := windows.GetConsoleMode(h, &mode); err != nil {
		return nil, err
	}
	wake, err := windows.CreateEvent(nil, 0, 0, nil)
	if err != nil {
		return nil, err
	}
	return &consoleReader{in: in, h: h, wake: wake, records: make([]inputRecord, 64)}, nil
}

func (r *consoleReader) wakeUp() { _ = windows.SetEvent(r.wake) }

func (r *consoleReader) pause() {
	r.mu.Lock()
	if r.resumed == nil {
		r.resumed = make(chan struct{})
	}
	r.mu.Unlock()
}

func (r *consoleReader) resume() {
	r.mu.Lock()
	if r.resumed != nil {
		close(r.resumed)
		r.resumed = nil
	}
	r.mu.Unlock()
	r.wakeUp()
}

// textPending reports whether a key press that produces text is queued. When
// none is, the records that are there are taken out of the queue, or they
// would keep the handle signalled with nothing for ReadConsole to return.
func (r *consoleReader) textPending() (bool, error) {
	var n uint32
	if err := windows.GetNumberOfConsoleInputEvents(r.h, &n); err != nil {
		return false, err
	}
	if n == 0 {
		return false, nil
	}
	if int(n) > len(r.records) {
		r.records = make([]inputRecord, n)
	}
	var got uint32
	ok, _, err := procPeekConsoleInputW.Call(uintptr(r.h), uintptr(unsafe.Pointer(&r.records[0])), uintptr(n), uintptr(unsafe.Pointer(&got)))
	if ok == 0 {
		return false, err
	}
	for _, rec := range r.records[:got] {
		if rec.eventType == windows.KEY_EVENT && rec.keyDown != 0 && rec.unicodeChar != 0 {
			return true, nil
		}
	}
	ok, _, err = procReadConsoleInputW.Call(uintptr(r.h), uintptr(unsafe.Pointer(&r.records[0])), uintptr(got), uintptr(unsafe.Pointer(&got)))
	if ok == 0 {
		return false, err
	}
	return false, nil
}

// chunks reads the console on its own goroutine, as readChunks does, until
// it fails or done is closed.
func (r *consoleReader) chunks(size int, done <-chan struct{}) <-chan []byte {
	out := make(chan []byte, 32)
	go func() {
		<-done
		r.wakeUp()
	}()
	go func() {
		defer close(out)
		defer windows.CloseHandle(r.wake)
		buf := make([]byte, size)
		handles := []windows.Handle{r.h, r.wake}
		for {
			ev, err := windows.WaitForMultipleObjects(handles, false, windows.INFINITE)
			if err != nil {
				return
			}
			select {
			case <-done:
				return
			default:
			}
			if ev != windows.WAIT_OBJECT_0 {
				continue // woken: paused, resumed or closing
			}

			r.mu.Lock()
			if wait := r.resumed; wait != nil {
				r.mu.Unlock()
				// The console belongs to another program: leave its input
				// alone until it is handed back.
				select {
				case <-wait:
					continue
				case <-done:
					return
				}
			}
			text, err := r.textPending()
			if err != nil {
				r.mu.Unlock()
				return
			}
			if !text {
				r.mu.Unlock()
				continue
			}
			n, err := r.in.Read(buf)
			r.mu.Unlock()
			if n > 0 {
				chunk := append([]byte(nil), buf[:n]...)
				select {
				case out <- chunk:
				case <-done:
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()
	return out
}

// Release hands the console to fn — an editor, a pager, a shell — and takes
// it back when fn returns. While fn runs the console is in the modes it had
// before Setup, on the normal screen, with nothing here reading its input.
// Afterwards the raw modes and the setup sequence are applied again and the
// capability probe is repeated; the caller should repaint (Terminal.Release
// does).
//
// Ctrl+C typed while fn runs reaches every process on the console, this one
// included; it is ignored here until fn returns, so quitting the program fn
// started does not end the application too.
//
// fn's error is returned; if the console cannot be set up again, that error
// is returned instead.
func (b *Backend) Release(fn func() error) error {
	if b.portableIO != nil || b.in == nil || b.out == nil {
		return ErrReleaseUnsupported
	}
	if b.looping.Load() && b.reader == nil {
		// A reader that cannot be stopped would take the program's input.
		return ErrReleaseUnsupported
	}
	if b.reader != nil {
		b.reader.pause()
		defer b.reader.resume()
	}
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	defer signal.Stop(interrupt)

	restore := fullScreenRestoreCmds()
	if height := b.Inline(); height > 0 {
		restore = inlineRestoreCmds(height)
	}
	if _, err := b.out.WriteString(restore); err != nil {
		return err
	}
	if b.state != nil {
		if err := RestoreConsole(b.state); err != nil {
			return err
		}
		b.state = nil
	}

	fnErr := fn()

	state, err := MakeRaw(b.in.Fd(), b.out.Fd())
	if err != nil {
		return fmt.Errorf("limoni: resume: %w", err)
	}
	b.state = state
	setup := setupSequence(b.Inline(), b.mouse.enabled())
	b.mouse.active.Store(true)
	setup = b.replies.withProbe(setup)
	if _, err := b.out.WriteString(setup); err != nil {
		return err
	}
	return fnErr
}
