//go:build unix

package driver

import (
	"errors"
	"fmt"
	"os"
	"sync"

	"golang.org/x/sys/unix"
)

// ErrReleaseUnsupported is returned by Release where the terminal cannot be
// handed to another program: a portable or remote backend, which has no
// terminal of its own to give, and platforms without one.
var ErrReleaseUnsupported = errors.New("limoni: the terminal cannot be handed to another program on this backend")

// ttyReader reads the terminal while the application owns it, and stops
// reading while Release has handed it to another program. A reader blocked in
// read(2) would take the first keys typed into the editor it was handed to,
// so it waits in select(2) on the terminal and a wake-up pipe instead, and
// reads only once the terminal has bytes and nobody else owns them.
//
// select rather than poll: macOS's poll does not work on terminals.
type ttyReader struct {
	fd      int // the terminal, taken once: File.Fd races with Close
	wakeR   *os.File
	wakeW   *os.File
	wakeFD  int
	mu      sync.Mutex    // held from the paused check to the end of the read
	resumed chan struct{} // non-nil while paused; closed on resume
}

func newTTYReader(in *os.File) (*ttyReader, error) {
	r, w, err := os.Pipe() // close-on-exec, so a released program does not inherit it
	if err != nil {
		return nil, err
	}
	return &ttyReader{fd: int(in.Fd()), wakeR: r, wakeW: w, wakeFD: int(r.Fd())}, nil
}

// wake interrupts a select the reader is waiting in.
func (r *ttyReader) wake() { _, _ = r.wakeW.Write([]byte{0}) }

// pause stops the reader before its next read. Once it returns, no byte
// typed afterwards is read until resume: the reader holds mu from its check
// to the end of a read that select said would not block.
func (r *ttyReader) pause() {
	r.mu.Lock()
	if r.resumed == nil {
		r.resumed = make(chan struct{})
	}
	r.mu.Unlock()
}

func (r *ttyReader) resume() {
	r.mu.Lock()
	if r.resumed != nil {
		close(r.resumed)
		r.resumed = nil
	}
	r.mu.Unlock()
	r.wake()
}

// chunks reads the terminal on its own goroutine, as readChunks does, until
// it fails or done is closed.
func (r *ttyReader) chunks(size int, done <-chan struct{}) <-chan []byte {
	out := make(chan []byte, 32)
	go func() {
		<-done
		r.wake()
	}()
	go func() {
		defer close(out)
		defer r.wakeR.Close()
		defer r.wakeW.Close()
		buf := make([]byte, size)
		drain := make([]byte, 64)
		fd, wfd := r.fd, r.wakeFD
		var set unix.FdSet
		for {
			set.Zero()
			set.Set(fd)
			set.Set(wfd)
			_, err := unix.Select(max(fd, wfd)+1, &set, nil, nil, nil)
			if err == unix.EINTR {
				continue
			}
			if err != nil {
				return
			}
			select {
			case <-done:
				return
			default:
			}
			if set.IsSet(wfd) {
				_, _ = r.wakeR.Read(drain)
			}
			if !set.IsSet(fd) {
				continue
			}

			r.mu.Lock()
			if wait := r.resumed; wait != nil {
				r.mu.Unlock()
				// The terminal belongs to another program: leave its bytes
				// alone until it is handed back.
				select {
				case <-wait:
					continue
				case <-done:
					return
				}
			}
			n, err := unix.Read(fd, buf)
			r.mu.Unlock()
			if n > 0 {
				chunk := append([]byte(nil), buf[:n]...)
				select {
				case out <- chunk:
				case <-done:
					return
				}
			}
			if err == unix.EINTR || err == unix.EAGAIN {
				continue
			}
			if err != nil || n == 0 {
				return
			}
		}
	}()
	return out
}

// Release hands the terminal to fn — an editor, a pager, a shell — and takes
// it back when fn returns. While fn runs the terminal is the way the shell
// left it: the normal screen, line editing and echo, no mouse reporting, and
// nothing here reading its input. Afterwards raw mode and the setup sequence
// are applied again and the capability probe is repeated; the caller should
// repaint (Terminal.Release does).
//
// An interrupt (Ctrl+C) typed while fn runs goes to the whole foreground
// process group, this process included; it is ignored here until fn returns,
// so quitting the program fn started does not quit the application too.
//
// fn's error is returned; if the terminal cannot be set up again, that error
// is returned instead.
func (b *Backend) Release(fn func() error) error {
	if b.portableIO != nil || b.in == nil || b.out == nil {
		return ErrReleaseUnsupported
	}
	if b.reader != nil {
		b.reader.pause()
		defer b.reader.resume()
	}
	b.released.Store(true)
	defer b.released.Store(false)
	return b.handOver(fn)
}

// handOver restores the terminal for the shell or another program, runs fn,
// and sets the terminal up again. The order matters: leave the alternate
// screen and raw mode *before* fn, or it gets a terminal with no echo and the
// application's screen still on it.
func (b *Backend) handOver(fn func() error) error {
	restore := fullScreenRestoreCmds()
	if height := b.Inline(); height > 0 {
		restore = inlineRestoreCmds(height)
	}
	if _, err := b.out.WriteString(restore); err != nil {
		return err
	}
	if b.state != nil {
		if err := Restore(int(b.in.Fd()), b.state); err != nil {
			return err
		}
		b.state = nil
	}

	fnErr := fn()

	state, err := MakeRaw(int(b.in.Fd()))
	if err != nil {
		return fmt.Errorf("limoni: resume: %w", err)
	}
	b.state = state
	setup := fullScreenSetupCmds()
	if height := b.Inline(); height > 0 {
		setup = inlineSetupCmds(height)
	}
	// The terminal may be a different one, or the same one reconfigured, so
	// ask it again what it supports.
	setup = b.replies.withProbe(setup)
	if _, err := b.out.WriteString(setup); err != nil {
		return err
	}
	return fnErr
}
