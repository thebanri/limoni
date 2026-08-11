package layout

// Breakpoint describes the minimum width at which a responsive variant is
// eligible. Breakpoints are evaluated by MinWidth, not by declaration order.
type Breakpoint struct {
	Name     string
	MinWidth uint16
}

// SelectBreakpoint returns the widest eligible breakpoint. The zero value is
// returned when no breakpoint applies, making the result deterministic for
// narrow terminals and empty breakpoint lists.
func SelectBreakpoint(width uint16, breakpoints []Breakpoint) Breakpoint {
	selected := Breakpoint{}
	for _, breakpoint := range breakpoints {
		if breakpoint.MinWidth <= width && breakpoint.MinWidth >= selected.MinWidth {
			selected = breakpoint
		}
	}
	return selected
}

// BreakpointValue associates an arbitrary layout value with a minimum width.
// Values can be constraints, directions, gaps, or application-specific
// responsive configuration without coupling this package to widgets.
type BreakpointValue[T any] struct {
	MinWidth uint16
	Value    T
}

// ResolveBreakpoint returns the value belonging to the widest eligible
// breakpoint, or fallback when no breakpoint applies.
func ResolveBreakpoint[T any](width uint16, values []BreakpointValue[T], fallback T) T {
	selected := fallback
	selectedWidth := uint16(0)
	found := false
	for _, value := range values {
		if value.MinWidth <= width && (!found || value.MinWidth >= selectedWidth) {
			selected = value.Value
			selectedWidth = value.MinWidth
			found = true
		}
	}
	return selected
}
