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
