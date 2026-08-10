package terminal

import (
	"testing"

	"github.com/thebanri/limoni/core/cell"
)

func TestFocusManagerScopeNavigation(t *testing.T) {
	manager := NewFocusManager()
	manager.Register("outside")
	manager.BeginScope("modal")
	manager.Register("first")
	manager.Register("second")
	manager.SetFocused("first")
	manager.Next()
	if manager.Focused() != "second" {
		t.Fatalf("scope next = %q; want second", manager.Focused())
	}
	manager.Next()
	if manager.Focused() != "first" {
		t.Fatalf("scope wrap = %q; want first", manager.Focused())
	}
	manager.EndScope()
	manager.SetFocused("outside")
	manager.Next()
	if manager.Focused() != "first" {
		t.Fatalf("global navigation = %q; want first", manager.Focused())
	}
}

func TestFocusManagerNextExcluding(t *testing.T) {
	manager := NewFocusManager()
	manager.Register("tab_Giriş")
	manager.Register("input")
	manager.Register("tab_Ayarlar")
	manager.Register("slider")
	manager.SetFocused("input")
	manager.NextExcluding("tab_")
	if manager.Focused() != "slider" {
		t.Fatalf("next non-tab focus = %q; want slider", manager.Focused())
	}
}

func TestFocusManagerIsFocused(t *testing.T) {
	manager := NewFocusManager()
	manager.Register("input")
	if !manager.IsFocused("input") {
		t.Fatal("registered input should be focused")
	}
	if manager.IsFocused("other") {
		t.Fatal("other widget should not be focused")
	}
	manager.SetFocused("other")
	if manager.IsFocused("input") || !manager.IsFocused("other") {
		t.Fatal("focus state did not update")
	}
}

func TestFocusManagerSpatialNavigation(t *testing.T) {
	manager := NewFocusManager()
	
	// Create a 2D layout:
	// A [0,0,10,3]    B [15,0,10,3]
	// C [0,5,10,3]    D [15,5,10,3]
	
	rectA := cell.NewRect(0, 0, 10, 3)
	rectB := cell.NewRect(15, 0, 10, 3)
	rectC := cell.NewRect(0, 5, 10, 3)
	rectD := cell.NewRect(15, 5, 10, 3)
	
	manager.Register("A")
	manager.RegisterBounds("A", rectA)
	
	manager.Register("B")
	manager.RegisterBounds("B", rectB)
	
	manager.Register("C")
	manager.RegisterBounds("C", rectC)
	
	manager.Register("D")
	manager.RegisterBounds("D", rectD)
	
	manager.SetFocused("A")
	
	// Move Right: A -> B
	if !manager.MoveFocus2D(DirRight) || manager.Focused() != "B" {
		t.Fatalf("MoveRight from A: got %q, want B", manager.Focused())
	}
	
	// Move Down: B -> D
	if !manager.MoveFocus2D(DirDown) || manager.Focused() != "D" {
		t.Fatalf("MoveDown from B: got %q, want D", manager.Focused())
	}
	
	// Move Left: D -> C
	if !manager.MoveFocus2D(DirLeft) || manager.Focused() != "C" {
		t.Fatalf("MoveLeft from D: got %q, want C", manager.Focused())
	}
	
	// Move Up: C -> A
	if !manager.MoveFocus2D(DirUp) || manager.Focused() != "A" {
		t.Fatalf("MoveUp from C: got %q, want A", manager.Focused())
	}
	
	// Try invalid direction: A has nothing directly left
	original := manager.Focused()
	if manager.MoveFocus2D(DirLeft) && manager.Focused() != original {
		t.Fatalf("MoveLeft from A should not have changed focus, got %q", manager.Focused())
	}
}

func TestFocusManagerSpatialScopeTrapping(t *testing.T) {
	manager := NewFocusManager()
	
	// Create two scopes:
	// Global: A [0,0,10,3]
	// Scope "sub": B [15,0,10,3], C [30,0,10,3]
	rectA := cell.NewRect(0, 0, 10, 3)
	rectB := cell.NewRect(15, 0, 10, 3)
	rectC := cell.NewRect(30, 0, 10, 3)
	
	manager.Register("A")
	manager.RegisterBounds("A", rectA)
	
	manager.BeginScope("sub")
	manager.Register("B")
	manager.RegisterBounds("B", rectB)
	manager.Register("C")
	manager.RegisterBounds("C", rectC)
	
	manager.SetFocused("B")
	
	// Move Left from B: should NOT jump to A because it's outside active scope
	if manager.MoveFocus2D(DirLeft) && manager.Focused() == "A" {
		t.Fatalf("Directional focus escaped scope: got %q, want B", manager.Focused())
	}
	
	// Move Right from B: should jump to C
	if !manager.MoveFocus2D(DirRight) || manager.Focused() != "C" {
		t.Fatalf("MoveRight from B inside scope: got %q, want C", manager.Focused())
	}
}

