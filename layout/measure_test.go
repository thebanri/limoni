package layout

import (
	"github.com/thebanri/limoni/core/cell"
	"testing"
)

type measuredWidget struct{}

func (measuredWidget) SizeHint(cell.Rect) (uint16, uint16) { return 8, 3 }

func TestMeasureWidgetAndNormalize(t *testing.T) {
	m := MeasureWidget(measuredWidget{}, cell.NewRect(0, 0, 5, 2)).Normalize(cell.NewRect(0, 0, 5, 2))
	if m.IdealWidth != 5 || m.IdealHeight != 2 {
		t.Fatalf("normalized measure = %+v", m)
	}
}

func TestArrangeDeterministicImpossibleConstraints(t *testing.T) {
	result := Arrange(cell.NewRect(0, 0, 5, 1), []Measure{{MinWidth: 4, IdealWidth: 4}, {MinWidth: 4, IdealWidth: 4}}, Horizontal, 1)
	if len(result) != 2 || result[0].Width+result[1].Width > 4 {
		t.Fatalf("arrangement exceeds available area: %+v", result)
	}
}

func TestArrangeAlignedCenterAndEnd(t *testing.T) {
	area := cell.NewRect(0, 0, 20, 10)
	measurements := []Measure{{IdealWidth: 4, IdealHeight: 2}}

	center := ArrangeAligned(area, measurements, Horizontal, 0, AlignCenter)[0]
	if center.Y != 4 || center.Height != 2 {
		t.Fatalf("center alignment = %+v, want y=4 height=2", center)
	}
	end := ArrangeAligned(area, measurements, Horizontal, 0, AlignEnd)[0]
	if end.Y != 8 || end.Height != 2 {
		t.Fatalf("end alignment = %+v, want y=8 height=2", end)
	}
}

func TestArrangeAlignedBaseline(t *testing.T) {
	area := cell.NewRect(0, 0, 30, 10)
	measurements := []Measure{
		{IdealWidth: 5, IdealHeight: 3, Baseline: 2},
		{IdealWidth: 7, IdealHeight: 5, Baseline: 4},
	}
	result := ArrangeAligned(area, measurements, Horizontal, 1, AlignBaseline)
	if result[0].Y != 2 || result[1].Y != 0 {
		t.Fatalf("baseline alignment = %+v, want y values 2 and 0", result)
	}
	if result[0].Height != 3 || result[1].Height != 5 {
		t.Fatalf("baseline heights = %d/%d", result[0].Height, result[1].Height)
	}
}

func TestMeasureAnyAndAggregateMeasures(t *testing.T) {
	legacy := measuredWidget{}
	measured := MeasureAny(legacy, cell.NewRect(0, 0, 20, 10))
	if measured.IdealWidth != 8 || measured.IdealHeight != 3 {
		t.Fatalf("intrinsic measure = %+v", measured)
	}
	aggregate := AggregateMeasures([]Measure{
		{MinWidth: 2, IdealWidth: 4, MaxWidth: 8, MinHeight: 1, IdealHeight: 2, MaxHeight: 4},
		{MinWidth: 3, IdealWidth: 5, MaxWidth: 9, MinHeight: 2, IdealHeight: 3, MaxHeight: 5},
	}, Horizontal, 1)
	if aggregate.MinWidth != 6 || aggregate.IdealWidth != 10 || aggregate.MaxWidth != 18 || aggregate.IdealHeight != 3 {
		t.Fatalf("aggregate measure = %+v", aggregate)
	}
}

func TestDiagnoseReportsAllocatedOverflow(t *testing.T) {
	measurements := []Measure{{IdealWidth: 10, IdealHeight: 4, Baseline: 2, Overflow: OverflowScroll}}
	diagnostics := Diagnose(measurements, []cell.Rect{cell.NewRect(0, 0, 8, 4)})
	if len(diagnostics) != 1 || !diagnostics[0].Overflowed {
		t.Fatalf("diagnostics = %+v, want width overflow", diagnostics)
	}
	if !diagnostics[0].Shrunk {
		t.Error("expected Shrunk to be true")
	}
	if diagnostics[0].Grown {
		t.Error("expected Grown to be false")
	}
	if diagnostics[0].BaselineOffset != 2 {
		t.Errorf("expected BaselineOffset to be 2, got %d", diagnostics[0].BaselineOffset)
	}
	if diagnostics[0].Policy != OverflowScroll {
		t.Errorf("expected Policy to be OverflowScroll, got %v", diagnostics[0].Policy)
	}
}
