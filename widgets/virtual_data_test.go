package widgets

import (
	"context"
	"errors"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"testing"
	"time"
)

type virtualSource struct{ fail bool }

func (v virtualSource) RowCount(context.Context) (int, error) {
	if v.fail {
		return 0, errors.New("count")
	}
	return 100, nil
}
func (v virtualSource) RowAt(_ context.Context, i int) (Row, error) {
	if v.fail {
		return Row{}, errors.New("row")
	}
	return Row{ID: RowID("row" + string(rune(i))), Text: "row"}, nil
}
func (v virtualSource) RowID(i int) RowID { return RowID("row" + string(rune(i))) }

func TestVirtualDataRefreshAndPrefetch(t *testing.T) {
	s := NewVirtualDataState()
	if err := s.Refresh(context.Background(), virtualSource{}, 10, 5, 2); err != nil {
		t.Fatal(err)
	}
	status, _ := s.Status()
	if status != VirtualReady || s.Count() != 100 {
		t.Fatalf("status/count = %v/%d", status, s.Count())
	}
	if _, ok := s.Row(16); !ok {
		t.Fatal("expected prefetched row")
	}
}

func TestVirtualDataError(t *testing.T) {
	s := NewVirtualDataState()
	if s.Refresh(context.Background(), virtualSource{fail: true}, 0, 1, 0) == nil {
		t.Fatal("expected error")
	}
	if status, _ := s.Status(); status != VirtualError {
		t.Fatalf("status = %v", status)
	}
}

func TestVirtualDataStableSelectionAndCancellation(t *testing.T) {
	s := NewVirtualDataState()
	s.Select(RowID("row-42"))
	if s.Selected() != RowID("row-42") {
		t.Fatalf("selected ID = %q", s.Selected())
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := s.Refresh(ctx, virtualSource{}, 0, 1, 0); !errors.Is(err, context.Canceled) {
		t.Fatalf("refresh error = %v, want context.Canceled", err)
	}
}

func TestVirtualDataFilterSortRecycleAndTypeahead(t *testing.T) {
	s := NewVirtualDataState()
	if err := s.Refresh(context.Background(), virtualSource{}, 0, 3, 0); err != nil {
		t.Fatal(err)
	}
	s.Select(RowID("row" + string(rune(1))))
	s.FilterCached("row")
	if _, ok := s.Typeahead("row"); !ok {
		t.Fatal("expected typeahead match")
	}
	s.SortCached(func(a, b Row) bool { return a.ID > b.ID })
	if s.Selected() == "" {
		t.Fatal("stable selection was lost during cache transforms")
	}
	if s.SelectedIndex() < 0 || !s.RemapSelection() {
		t.Fatal("selection remapping failed")
	}
}

func TestVirtualDataOptionalProviderQuery(t *testing.T) {
	s := NewVirtualDataState()
	provider := &queryVirtualSource{}
	s.SetFilter("needle")
	s.SetSort("name", true)
	if err := s.Refresh(context.Background(), provider, 0, 1, 0); err != nil {
		t.Fatal(err)
	}
	if provider.query.Filter != "needle" || provider.query.SortKey != "name" || !provider.query.SortDescending {
		t.Fatalf("provider query = %+v", provider.query)
	}
}

func TestVirtualDataRejectsStaleRefresh(t *testing.T) {
	s := NewVirtualDataState()
	provider := &blockingVirtualSource{started: make(chan struct{}), release: make(chan struct{})}
	firstDone := make(chan error, 1)
	go func() { firstDone <- s.Refresh(context.Background(), provider, 0, 1, 0) }()
	<-provider.started
	if err := s.Refresh(context.Background(), virtualSource{}, 0, 1, 0); err != nil {
		t.Fatal(err)
	}
	close(provider.release)
	if err := <-firstDone; !errors.Is(err, ErrVirtualStale) {
		t.Fatalf("stale refresh error = %v", err)
	}
	if status, _ := s.Status(); status != VirtualReady {
		t.Fatalf("status after stale refresh = %v, want ready", status)
	}
}

func TestVirtualDataBackpressureCancelsPreviousRefresh(t *testing.T) {
	s := NewVirtualDataState()
	provider := &cancelAwareVirtualSource{started: make(chan struct{}), canceled: make(chan struct{})}
	first := s.RefreshLatest(context.Background(), provider, 0, 1, 0)
	<-provider.started
	second := s.RefreshLatest(context.Background(), virtualSource{}, 0, 1, 0)
	if err := <-second; err != nil {
		t.Fatal(err)
	}
	if err := <-first; !errors.Is(err, ErrVirtualStale) {
		t.Fatalf("first refresh error = %v, want stale", err)
	}
	select {
	case <-provider.canceled:
	case <-time.After(time.Second):
		t.Fatal("previous provider was not canceled")
	}
}

func TestVirtualDataQueuePolicies(t *testing.T) {
	s := NewVirtualDataState()
	provider := &cancelAwareVirtualSource{started: make(chan struct{}), canceled: make(chan struct{})}
	s.SetQueuePolicy(VirtualDropLatest)
	ctx, cancel := context.WithCancel(context.Background())
	first := s.RefreshLatest(ctx, provider, 0, 1, 0)
	<-provider.started
	if err := s.Refresh(context.Background(), virtualSource{}, 0, 1, 0); !errors.Is(err, ErrVirtualBusy) {
		t.Fatalf("drop-latest error = %v", err)
	}
	cancel()
	_ = <-first

	s.SetQueuePolicy(VirtualSequential)
	provider2 := &blockingVirtualSource{started: make(chan struct{}), release: make(chan struct{})}
	first = s.RefreshLatest(context.Background(), provider2, 0, 1, 0)
	<-provider2.started
	second := s.RefreshLatest(context.Background(), virtualSource{}, 0, 1, 0)
	close(provider2.release)
	if err := <-second; err != nil {
		t.Fatal(err)
	}
	_ = <-first
}

type cancelAwareVirtualSource struct{ started, canceled chan struct{} }

func (c *cancelAwareVirtualSource) RowCount(ctx context.Context) (int, error) {
	close(c.started)
	<-ctx.Done()
	close(c.canceled)
	return 0, ctx.Err()
}
func (*cancelAwareVirtualSource) RowAt(context.Context, int) (Row, error) { return Row{}, nil }
func (*cancelAwareVirtualSource) RowID(int) RowID                         { return "cancel" }

type blockingVirtualSource struct{ started, release chan struct{} }

func (b *blockingVirtualSource) RowCount(context.Context) (int, error) {
	close(b.started)
	<-b.release
	return 1, nil
}
func (*blockingVirtualSource) RowAt(context.Context, int) (Row, error) {
	return Row{ID: "blocked", Text: "blocked"}, nil
}
func (*blockingVirtualSource) RowID(int) RowID { return "blocked" }

type queryVirtualSource struct{ query VirtualQuery }

func (q *queryVirtualSource) ApplyQuery(_ context.Context, query VirtualQuery) error {
	q.query = query
	return nil
}
func (*queryVirtualSource) RowCount(context.Context) (int, error) { return 1, nil }
func (*queryVirtualSource) RowAt(context.Context, int) (Row, error) {
	return Row{ID: "query", Text: "needle"}, nil
}
func (*queryVirtualSource) RowID(int) RowID { return "query" }

func TestVirtualDataViewRendersVisibleRows(t *testing.T) {
	state := NewVirtualDataState()
	if err := state.Refresh(context.Background(), virtualSource{}, 4, 3, 0); err != nil {
		t.Fatal(err)
	}
	view := VirtualDataView{State: state, Source: virtualSource{}, First: 4}
	buf := buffer.NewBuffer(cell.NewRect(0, 0, 20, 3))
	view.Draw(cell.NewContext(cell.NewRect(0, 0, 20, 3), cell.Style{}), buf)
	if buf.Get(0, 0).Content == ' ' {
		t.Fatal("expected visible virtual row")
	}
}

func TestVirtualDataViewSelectsRowAndReportsIndex(t *testing.T) {
	state := NewVirtualDataState()
	selected := -1
	click := func(area cell.Rect, handler func()) {
		if area.Contains(2, 1) {
			handler()
		}
	}
	view := VirtualDataView{
		State: state, Source: virtualSource{}, OnSelect: func(index int, _ Row) { selected = index },
	}
	ctx := cell.NewContext(cell.NewRect(0, 0, 20, 3), cell.Style{})
	ctx.RegisterClick = click
	view.Draw(ctx, buffer.NewBuffer(cell.NewRect(0, 0, 20, 3)))
	if selected != 1 || state.Selected() == "" {
		t.Fatalf("selection = index %d, id %q; want row 1", selected, state.Selected())
	}
}

func TestVirtualDataViewRendersVariableRowHeight(t *testing.T) {
	state := NewVirtualDataState()
	source := variableVirtualSource{}
	if err := state.Refresh(context.Background(), source, 0, 3, 0); err != nil {
		t.Fatal(err)
	}
	view := VirtualDataView{State: state, Source: source, First: 0}
	buf := buffer.NewBuffer(cell.NewRect(0, 0, 20, 3))
	view.Draw(cell.NewContext(cell.NewRect(0, 0, 20, 3), cell.Style{}), buf)
	if buf.Get(0, 0).Content != 'a' || buf.Get(0, 1).Content != 'a' || buf.Get(0, 2).Content != 'b' {
		t.Fatalf("variable rows snapshot = %q/%q/%q", buf.Get(0, 0).Content, buf.Get(0, 1).Content, buf.Get(0, 2).Content)
	}
}

func TestVirtualRowTextKeepsStickyColumnsDuringHorizontalScroll(t *testing.T) {
	row := Row{Cells: []TableCell{{Text: "ID"}, {Text: "name"}, {Text: "status"}}}
	if got := virtualRowText(row, 5, 1); got != "ID | | status" {
		t.Fatalf("sticky row text = %q", got)
	}
	if got := virtualRowText(row, 0, 2); got != "ID | name | status" {
		t.Fatalf("unscrolled row text = %q", got)
	}
}

type variableVirtualSource struct{}

func (variableVirtualSource) RowCount(context.Context) (int, error) { return 3, nil }
func (variableVirtualSource) RowAt(_ context.Context, index int) (Row, error) {
	height := uint16(1)
	if index == 0 {
		height = 2
	}
	return Row{ID: RowID(string(rune('a' + index))), Text: string(rune('a' + index)), Height: height}, nil
}
func (variableVirtualSource) RowID(index int) RowID { return RowID(string(rune('a' + index))) }

func TestVirtualDataAdvancedFeatures(t *testing.T) {
	state := NewVirtualDataState()
	source := virtualSource{}

	// 1. Test Multi-select helpers
	state.ToggleSelect("row-1")
	state.ToggleSelect("row-2")
	if !state.IsSelected("row-1") || !state.IsSelected("row-2") {
		t.Error("expected row-1 and row-2 to be selected")
	}
	state.ToggleSelect("row-1")
	if state.IsSelected("row-1") {
		t.Error("expected row-1 to be deselected")
	}
	state.ClearSelected()
	if len(state.SelectedSet()) != 0 {
		t.Error("expected selected set to be empty")
	}

	// 2. Test Selection Remapping to Closest
	err := state.Refresh(nil, source, 0, 5, 0)
	if err != nil {
		t.Fatal(err)
	}
	// Select row 4 (ID: row\x04)
	id4 := source.RowID(4)
	state.Select(id4)
	if state.SelectedIndex() != 4 {
		t.Errorf("SelectedIndex = %d, want 4", state.SelectedIndex())
	}

	// Now refresh viewport to rows 6 to 10
	err = state.Refresh(nil, source, 6, 4, 0)
	if err != nil {
		t.Fatal(err)
	}
	// Remap selection should map to the closest loaded row (which is index 6, row\x06)
	remapped := state.RemapSelection()
	if !remapped {
		t.Error("expected RemapSelection to return true")
	}
	id6 := source.RowID(6)
	if state.Selected() != id6 {
		t.Errorf("selected RowID = %s, want %s", state.Selected(), id6)
	}

	// 3. Test Queue Stats
	stats := state.QueueStats()
	if stats.Completed != 2 {
		t.Errorf("stats.Completed = %d, want 2", stats.Completed)
	}
	if stats.Started != 2 {
		t.Errorf("stats.Started = %d, want 2", stats.Started)
	}
}
