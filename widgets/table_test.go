package widgets

import (
	"testing"
)

func TestTableWidthSolver(t *testing.T) {
	constraints := []TableConstraint{
		{Type: ConstraintFixed, Value: 10},
		{Type: ConstraintPercentage, Value: 50},
		{Type: ConstraintFill, Value: 0},
	}

	widths := SolveWidths(100, constraints)
	if len(widths) != 3 {
		t.Fatalf("SolveWidths length = %d; 3 bekleniyordu", len(widths))
	}

	// 100 genişliğinde:
	// Fixed 10 -> 10
	// Percentage 50 -> 50
	// Fill -> 100 - (10 + 50) = 40
	if widths[0] != 10 {
		t.Errorf("widths[0] = %d; 10 bekleniyordu", widths[0])
	}
	if widths[1] != 50 {
		t.Errorf("widths[1] = %d; 50 bekleniyordu", widths[1])
	}
	if widths[2] != 40 {
		t.Errorf("widths[2] = %d; 40 bekleniyordu", widths[2])
	}
}

func TestTableStateNavigation(t *testing.T) {
	state := NewTableState()
	if state.Selected != -1 {
		t.Errorf("NewTableState.Selected = %d; -1 bekleniyordu", state.Selected)
	}

	state.Next(5)
	if state.Selected != 0 {
		t.Errorf("Selected after Next() = %d; 0 bekleniyordu", state.Selected)
	}

	state.Next(5)
	if state.Selected != 1 {
		t.Errorf("Selected after second Next() = %d; 1 bekleniyordu", state.Selected)
	}

	state.Prev()
	if state.Selected != 0 {
		t.Errorf("Selected after Prev() = %d; 0 bekleniyordu", state.Selected)
	}

	// Sınır koruma (min index 0)
	state.Prev()
	if state.Selected != 0 {
		t.Errorf("Selected after Prev() boundary = %d; 0 bekleniyordu", state.Selected)
	}
}

func TestTableTextClipping(t *testing.T) {
	s1 := clipString("Hello World", 5)
	if s1 != "He..." {
		t.Errorf("clipString = %q; 'He...' bekleniyordu", s1)
	}

	s2 := clipString("Hi", 5)
	if s2 != "Hi" {
		t.Errorf("clipString = %q; 'Hi' bekleniyordu", s2)
	}

	s3 := clipString("Hello", 2)
	if s3 != "He" {
		t.Errorf("clipString = %q; 'He' bekleniyordu", s3)
	}
}
