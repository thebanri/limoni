package widgets

import (
	"context"
	"errors"
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
