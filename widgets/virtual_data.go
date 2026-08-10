package widgets

import (
	"context"
	"sync"
)

// RowID is a stable identity independent of the row's current index.
type RowID string

// Row is a virtualized data row.
type Row struct {
	ID    RowID
	Cells []TableCell
	Text  string
}

// VirtualDataSource supplies rows asynchronously.
type VirtualDataSource interface {
	RowCount(context.Context) (int, error)
	RowAt(context.Context, int) (Row, error)
	RowID(int) RowID
}

type VirtualStatus uint8

const (
	VirtualIdle VirtualStatus = iota
	VirtualLoading
	VirtualReady
	VirtualError
	VirtualEmpty
)

// VirtualDataState is a concurrency-safe viewport cache.
type VirtualDataState struct {
	mu       sync.RWMutex
	rows     map[int]Row
	selected RowID
	status   VirtualStatus
	err      error
	count    int
}

func NewVirtualDataState() *VirtualDataState {
	return &VirtualDataState{rows: make(map[int]Row), status: VirtualIdle}
}

func (s *VirtualDataState) Status() (VirtualStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status, s.err
}
func (s *VirtualDataState) Count() int { s.mu.RLock(); defer s.mu.RUnlock(); return s.count }
func (s *VirtualDataState) Row(index int) (Row, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	row, ok := s.rows[index]
	return row, ok
}
func (s *VirtualDataState) Selected() RowID { s.mu.RLock(); defer s.mu.RUnlock(); return s.selected }
func (s *VirtualDataState) Select(id RowID) { s.mu.Lock(); s.selected = id; s.mu.Unlock() }

// Refresh loads the count and a viewport plus prefetch rows synchronously for
// deterministic callers; the provider itself may perform async I/O.
func (s *VirtualDataState) Refresh(ctx context.Context, source VirtualDataSource, first, visible, prefetch int) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		s.mu.Lock()
		s.status = VirtualError
		s.err = err
		s.mu.Unlock()
		return err
	}
	s.mu.Lock()
	s.status = VirtualLoading
	s.err = nil
	s.mu.Unlock()
	count, err := source.RowCount(ctx)
	if err != nil {
		s.mu.Lock()
		s.status = VirtualError
		s.err = err
		s.mu.Unlock()
		return err
	}
	if count < 0 {
		count = 0
	}
	last := first + visible + prefetch
	if first < 0 {
		first = 0
	}
	if last > count {
		last = count
	}
	loaded := make(map[int]Row)
	for i := first; i < last; i++ {
		row, rowErr := source.RowAt(ctx, i)
		if rowErr != nil {
			s.mu.Lock()
			s.status = VirtualError
			s.err = rowErr
			s.mu.Unlock()
			return rowErr
		}
		loaded[i] = row
	}
	s.mu.Lock()
	s.rows = loaded
	s.count = count
	s.status = VirtualReady
	if count == 0 {
		s.status = VirtualEmpty
	}
	s.mu.Unlock()
	return nil
}
