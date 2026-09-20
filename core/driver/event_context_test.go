package driver

import "testing"

// EventContext defaults to allowing propagation and default behavior so that
// handlers only pay for what they intercept.
func TestEventContextDefaultsAllowPropagationAndDefault(t *testing.T) {
	var ctx EventContext
	if ctx.IsPropagationStopped() {
		t.Error("a zero-value EventContext should not report propagation stopped")
	}
	if ctx.IsDefaultPrevented() {
		t.Error("a zero-value EventContext should not report default prevented")
	}
}

// StopPropagation and PreventDefault are independent flags: setting one must
// not affect the other.
func TestEventContextPropagationAndDefaultFlagsAreIndependent(t *testing.T) {
	// Calling StopPropagation stops further handlers from receiving the event.
	var stopCtx EventContext
	stopCtx.StopPropagation()
	if !stopCtx.IsPropagationStopped() {
		t.Error("StopPropagation did not set propagation stopped flag")
	}
	if stopCtx.IsDefaultPrevented() {
		t.Error("StopPropagation should not prevent default behavior")
	}

	// Calling PreventDefault prevents the default terminal action without stopping propagation.
	var prevCtx EventContext
	prevCtx.PreventDefault()
	if !prevCtx.IsDefaultPrevented() {
		t.Error("PreventDefault did not set default prevented flag")
	}
	if prevCtx.IsPropagationStopped() {
		t.Error("PreventDefault should not stop event propagation")
	}

	// Calling both sets both flags independently.
	var bothCtx EventContext
	bothCtx.StopPropagation()
	bothCtx.PreventDefault()
	if !bothCtx.IsPropagationStopped() || !bothCtx.IsDefaultPrevented() {
		t.Errorf("both flags should be set: stopped=%v, defaultPrevented=%v",
			bothCtx.IsPropagationStopped(), bothCtx.IsDefaultPrevented())
	}
}
