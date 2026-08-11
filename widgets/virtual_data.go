package widgets

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
)

// ErrVirtualStale reports that a refresh result was superseded by a newer
// refresh before it could be committed to the viewport cache.
var ErrVirtualStale = errors.New("virtual data: stale refresh result")
var ErrVirtualBusy = errors.New("virtual data: refresh already in progress")

// RowID is a stable identity independent of the row's current index.
type RowID string

// Row is a virtualized data row.
type Row struct {
	ID    RowID
	Cells []TableCell
	Text  string
	// Height is the number of terminal rows occupied by this item. Zero uses
	// the default one-cell row height.
	Height uint16
}

// VirtualDataSource supplies rows asynchronously.
type VirtualDataSource interface {
	RowCount(context.Context) (int, error)
	RowAt(context.Context, int) (Row, error)
	RowID(int) RowID
}

// VirtualQuery describes optional provider-side filtering and sorting. It
// keeps large datasets from being loaded into the client cache.
type VirtualQuery struct {
	Filter         string
	SortKey        string
	SortDescending bool
}

// VirtualQueryable is an optional extension to VirtualDataSource. Providers
// can implement it to execute filtering/sorting remotely or incrementally.
type VirtualQueryable interface {
	VirtualDataSource
	ApplyQuery(context.Context, VirtualQuery) error
}

type VirtualQueuePolicy uint8

const (
	VirtualLatestOnly VirtualQueuePolicy = iota
	VirtualDropOldest
	VirtualDropLatest
	VirtualSequential
)

type VirtualQueueStats struct {
	Active, QueueLength                          int
	Started, Completed, Canceled, Stale, Dropped uint64
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
	mu          sync.RWMutex
	rows        map[int]Row
	selected    RowID
	status      VirtualStatus
	err         error
	count       int
	filter      string
	sortKey     string
	sortDesc    bool
	generation  uint64
	cancel      context.CancelFunc
	queuePolicy VirtualQueuePolicy
	activeDone  chan struct{}
	stats       VirtualQueueStats
}

func NewVirtualDataState() *VirtualDataState {
	return &VirtualDataState{rows: make(map[int]Row), status: VirtualIdle, queuePolicy: VirtualLatestOnly}
}

func (s *VirtualDataState) SetQueuePolicy(policy VirtualQueuePolicy) {
	s.mu.Lock()
	s.queuePolicy = policy
	s.mu.Unlock()
}
func (s *VirtualDataState) QueuePolicy() VirtualQueuePolicy {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.queuePolicy
}

func (s *VirtualDataState) QueueStats() VirtualQueueStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	stats := s.stats
	if s.activeDone != nil {
		stats.Active = 1
	}
	return stats
}
func (s *VirtualDataState) SelectedIndex() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for index, row := range s.rows {
		if row.ID == s.selected {
			return index
		}
	}
	return -1
}
func (s *VirtualDataState) RemapSelection() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.selected == "" {
		return false
	}
	for _, row := range s.rows {
		if row.ID == s.selected {
			return true
		}
	}
	s.selected = ""
	return false
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

// FilterQuery returns the current incremental viewport filter.
func (s *VirtualDataState) FilterQuery() string { s.mu.RLock(); defer s.mu.RUnlock(); return s.filter }

// SetFilter updates the filter without discarding stable selection identity.
// Refresh should be called afterwards to load the filtered provider viewport.
func (s *VirtualDataState) SetFilter(query string) { s.mu.Lock(); s.filter = query; s.mu.Unlock() }

func (s *VirtualDataState) SetSort(key string, descending bool) {
	s.mu.Lock()
	s.sortKey, s.sortDesc = key, descending
	s.mu.Unlock()
}

// FilterCached applies an incremental fuzzy-like case-insensitive filter to
// already loaded rows and recycles the cache map without changing RowIDs.
func (s *VirtualDataState) FilterCached(query string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.filter = query
	if query == "" {
		return
	}
	filtered := make(map[int]Row, len(s.rows))
	for index, row := range s.rows {
		if strings.Contains(strings.ToLower(row.Text), strings.ToLower(query)) {
			filtered[index] = row
		}
	}
	s.rows = filtered
}

// SortCached sorts loaded rows deterministically by text while preserving the
// selected RowID. It is intentionally cache-local for virtual data sources.
func (s *VirtualDataState) SortCached(less func(a, b Row) bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	indices := make([]int, 0, len(s.rows))
	rows := make([]Row, 0, len(s.rows))
	for index, row := range s.rows {
		indices = append(indices, index)
		rows = append(rows, row)
	}
	sort.SliceStable(rows, func(i, j int) bool { return less(rows[i], rows[j]) })
	s.rows = make(map[int]Row, len(rows))
	for i, row := range rows {
		s.rows[indices[i]] = row
	}
}

// Typeahead returns the first loaded row whose text starts with query.
func (s *VirtualDataState) Typeahead(query string) (RowID, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	query = strings.ToLower(query)
	indices := make([]int, 0, len(s.rows))
	for index := range s.rows {
		indices = append(indices, index)
	}
	sort.Ints(indices)
	for _, index := range indices {
		row := s.rows[index]
		if strings.HasPrefix(strings.ToLower(row.Text), query) {
			return row.ID, true
		}
	}
	return "", false
}

// Refresh loads the count and a viewport plus prefetch rows synchronously for
// deterministic callers; the provider itself may perform async I/O.
func (s *VirtualDataState) Refresh(ctx context.Context, source VirtualDataSource, first, visible, prefetch int) error {
	if ctx == nil {
		ctx = context.Background()
	}
	s.mu.Lock()
	for s.activeDone != nil {
		policy := s.queuePolicy
		if policy == VirtualDropLatest {
			s.mu.Unlock()
			return ErrVirtualBusy
		}
		if policy == VirtualSequential {
			done := s.activeDone
			s.mu.Unlock()
			<-done
			s.mu.Lock()
			continue
		}
		if s.cancel != nil {
			s.cancel()
		}
		break
	}
	if s.cancel != nil {
		s.cancel()
	}
	requestCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.generation++
	generation := s.generation
	s.stats.Started++
	s.activeDone = make(chan struct{})
	activeDone := s.activeDone
	s.status = VirtualLoading
	s.err = nil
	query := VirtualQuery{Filter: s.filter, SortKey: s.sortKey, SortDescending: s.sortDesc}
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		if s.generation == generation {
			s.cancel = nil
			if s.activeDone == activeDone {
				close(activeDone)
				s.activeDone = nil
			}
		}
		s.mu.Unlock()
	}()
	if err := requestCtx.Err(); err != nil {
		s.mu.Lock()
		if s.generation == generation {
			s.status, s.err = VirtualError, err
		}
		s.mu.Unlock()
		return err
	}
	if queryable, ok := source.(VirtualQueryable); ok && (query.Filter != "" || query.SortKey != "") {
		if err := queryable.ApplyQuery(requestCtx, query); err != nil {
			s.mu.Lock()
			if s.generation != generation {
				s.mu.Unlock()
				return ErrVirtualStale
			}
			s.status, s.err = VirtualError, err
			s.mu.Unlock()
			return err
		}
	}
	count, err := source.RowCount(requestCtx)
	if err != nil {
		s.mu.Lock()
		if s.generation != generation {
			s.stats.Stale++
			s.mu.Unlock()
			return ErrVirtualStale
		}
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
		row, rowErr := source.RowAt(requestCtx, i)
		if rowErr != nil {
			s.mu.Lock()
			if s.generation != generation {
				s.mu.Unlock()
				return ErrVirtualStale
			}
			s.status = VirtualError
			s.err = rowErr
			s.mu.Unlock()
			return rowErr
		}
		loaded[i] = row
	}
	s.mu.Lock()
	if s.generation != generation {
		s.stats.Stale++
		s.mu.Unlock()
		return ErrVirtualStale
	}
	s.rows = loaded
	s.count = count
	s.status = VirtualReady
	if count == 0 {
		s.status = VirtualEmpty
	}
	s.mu.Unlock()
	return nil
}

// RefreshLatest starts a refresh asynchronously. Starting another refresh on
// the same state cancels this one and makes its result stale.
func (s *VirtualDataState) RefreshLatest(ctx context.Context, source VirtualDataSource, first, visible, prefetch int) <-chan error {
	done := make(chan error, 1)
	go func() {
		done <- s.Refresh(ctx, source, first, visible, prefetch)
		close(done)
	}()
	return done
}
