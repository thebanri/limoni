package widgets

import (
	"context"
	"errors"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"testing"
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
