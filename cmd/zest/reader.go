package main

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os"
	"time"
)

// pollInterval is how often a followed file is checked for new lines.
const pollInterval = 200 * time.Millisecond

// readLines reads r line by line into s, calling notify after each burst of
// lines. With follow set and r a file, it keeps polling at EOF like tail -f,
// and starts over if the file is truncated (log rotation by copytruncate).
func readLines(ctx context.Context, r io.Reader, s *store, notify func(), follow bool) error {
	file, _ := r.(*os.File)
	br := bufio.NewReaderSize(r, 256<<10)
	var long []byte // a line longer than the reader's buffer, being assembled
	var offset int64
	lastNotify := time.Now()
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		chunk, err := br.ReadSlice('\n')
		offset += int64(len(chunk))
		switch {
		case err == nil:
			line := chunk[:len(chunk)-1]
			if long != nil {
				line = append(long, line...)
				long = nil
			}
			s.add(trimCR(line))
		case errors.Is(err, bufio.ErrBufferFull):
			long = append(long, chunk...)
			continue
		case errors.Is(err, io.EOF):
			if len(chunk) > 0 {
				// A partial last line: keep it until its newline arrives,
				// or add it now if nothing more will.
				long = append(long, chunk...)
			}
			notify()
			if !follow || file == nil {
				if len(long) > 0 {
					s.add(trimCR(long))
					notify()
				}
				return nil
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(pollInterval):
			}
			if fi, err := file.Stat(); err == nil && fi.Size() < offset {
				// Truncated: read it again from the start.
				if _, err := file.Seek(0, io.SeekStart); err == nil {
					offset = 0
					long = nil
					br.Reset(file)
				}
			}
			continue
		default:
			notify()
			return err
		}
		if time.Since(lastNotify) > 30*time.Millisecond {
			notify()
			lastNotify = time.Now()
		}
	}
}

func trimCR(b []byte) []byte {
	if n := len(b); n > 0 && b[n-1] == '\r' {
		return b[:n-1]
	}
	return b
}
