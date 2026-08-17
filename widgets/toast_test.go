package widgets

import (
	"testing"
	"time"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

func TestToastManager_AddAndDismiss(t *testing.T) {
	tm := NewToastManager(ToastTopRight)
	t1 := tm.Info("Build Started", "Compiling package...")
	if t1 == nil || len(tm.Toasts) != 1 {
		t.Fatalf("expected 1 toast, got %d", len(tm.Toasts))
	}

	t2 := tm.Success("Build Succeeded", "All binaries compiled.")
	if len(tm.Toasts) != 2 {
		t.Fatalf("expected 2 toasts, got %d", len(tm.Toasts))
	}

	// Dismiss t1
	tm.Dismiss(t1.ID)
	tm.Update(time.Now())
	if len(tm.Toasts) != 1 || tm.Toasts[0].ID != t2.ID {
		t.Errorf("expected only t2 to remain, got %d", len(tm.Toasts))
	}

	// Auto-expire t2
	future := time.Now().Add(10 * time.Second)
	tm.Update(future)
	if len(tm.Toasts) != 0 {
		t.Errorf("expected all toasts to be expired, got %d", len(tm.Toasts))
	}
}

func TestToastManager_Draw(t *testing.T) {
	tm := NewToastManager(ToastTopRight)
	tm.Success("Success", "Operation finished successfully.")

	area := cell.NewRect(0, 0, 80, 24)
	buf := buffer.NewBuffer(area)
	ctx := cell.NewContext(area, cell.Style{})

	tm.Draw(ctx, buf)

	// Check that top-right corner region has cells drawn
	c := buf.Get(area.Width-10, 1)
	if c == nil {
		t.Fatal("expected non-nil cell for drawn toast")
	}
}
