package layout

import "github.com/thebanri/limoni/core/cell"

// LayoutDiagnostic describes the result of a measure/arrange pass for one
// child. It is intentionally data-only so inspectors and tests can consume it
// without depending on terminal rendering.
type LayoutDiagnostic struct {
	Index          int
	Measured       Measure
	Allocated      cell.Rect
	Overflowed     bool
	Shrunk         bool
	Grown          bool
	BaselineOffset uint16
	Policy         OverflowPolicy
}

// Diagnose pairs measurements with allocated rectangles and reports whether a
// child was clipped on either axis. Missing rectangles are represented by a
// zero rectangle, making mismatched inputs deterministic.
func Diagnose(measurements []Measure, allocated []cell.Rect) []LayoutDiagnostic {
	result := make([]LayoutDiagnostic, len(measurements))
	for i, measured := range measurements {
		var rect cell.Rect
		if i < len(allocated) {
			rect = allocated[i]
		}
		shrunk := rect.Width < measured.IdealWidth || rect.Height < measured.IdealHeight
		grown := rect.Width > measured.IdealWidth || rect.Height > measured.IdealHeight
		result[i] = LayoutDiagnostic{
			Index:          i,
			Measured:       measured,
			Allocated:      rect,
			Overflowed:     shrunk,
			Shrunk:         shrunk,
			Grown:          grown,
			BaselineOffset: measured.Baseline,
			Policy:         measured.Overflow,
		}
	}
	return result
}
