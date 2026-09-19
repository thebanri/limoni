package main

import (
	"sync"
	"unsafe"

	"github.com/thebanri/limoni/widgets"
)

// chunkSize is how much line text one chunk holds. A line longer than this
// gets a chunk of its own.
const chunkSize = 1 << 20

// store holds every line read so far, append-only.
//
// Text lives in fixed-capacity chunks that are never reallocated or
// modified once written, so a line can be handed out as a string that views
// the chunk's bytes directly (unsafe.String). That is what lets the viewer
// draw a million-line log without allocating: nothing is copied per line or
// per frame. The price is that a chunk stays alive as long as any of its
// lines does, which for a log viewer is always.
type store struct {
	mu     sync.RWMutex
	chunks [][]byte
	lines  []lineRef
	counts [widgets.LevelFatal + 1]int
}

type lineRef struct {
	chunk uint32
	off   uint32
	n     uint32
	level widgets.LogLevel
}

// add appends one line, without its trailing newline, and returns its level.
func (s *store) add(line []byte) widgets.LogLevel {
	level := detectLevel(line)
	s.mu.Lock()
	defer s.mu.Unlock()
	// An indented line with no level of its own — a stack frame, a wrapped
	// message — belongs to the line above, so it filters with it: showing
	// only errors keeps the whole trace.
	if level == widgets.LevelUnknown && len(line) > 0 && (line[0] == ' ' || line[0] == '\t') && len(s.lines) > 0 {
		level = s.lines[len(s.lines)-1].level
	}
	last := len(s.chunks) - 1
	if last < 0 || len(s.chunks[last])+len(line) > cap(s.chunks[last]) {
		size := chunkSize
		if len(line) > size {
			size = len(line)
		}
		s.chunks = append(s.chunks, make([]byte, 0, size))
		last++
	}
	c := s.chunks[last]
	off := len(c)
	s.chunks[last] = append(c, line...)
	s.lines = append(s.lines, lineRef{chunk: uint32(last), off: uint32(off), n: uint32(len(line)), level: level})
	s.counts[level]++
	return level
}

// Len is the number of lines.
func (s *store) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.lines)
}

// Line returns line i without copying it.
func (s *store) Line(i int) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lineLocked(i)
}

func (s *store) lineLocked(i int) string {
	ref := s.lines[i]
	if ref.n == 0 {
		return ""
	}
	c := s.chunks[ref.chunk]
	return unsafe.String(&c[ref.off], int(ref.n))
}

// Level returns line i's level.
func (s *store) Level(i int) widgets.LogLevel {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lines[i].level
}

// Counts returns how many lines there are of each level.
func (s *store) Counts() [widgets.LevelFatal + 1]int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.counts
}

// bytes is the text held, for the status line.
func (s *store) bytes() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := 0
	for _, c := range s.chunks {
		n += len(c)
	}
	return n
}
