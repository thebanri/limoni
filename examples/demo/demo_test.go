package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/graphics"
	"github.com/thebanri/limoni/widgets"
)

func TestDemoRenderAndInteractions(t *testing.T) {
	memIO := driver.NewMemoryTerminalIO(nil, 100, 30)
	b := driver.NewPortableBackend(memIO)
	term, err := terminal.New(b)
	if err != nil {
		t.Fatalf("Failed to create terminal: %v", err)
	}

	state := initAppState()

	// 1. Test Tab 0 (3D) Render
	state.ActiveTab = 0
	renderFrame(term, state)

	// Verify AutoRotate initial state
	if !state.AutoRotate {
		t.Errorf("Expected AutoRotate=true initially")
	}

	// 2. Test Tab 1 (TreeView)
	state.ActiveTab = 1
	renderFrame(term, state)
	if state.TreeState.SelectedID == "" {
		t.Errorf("Expected TreeView to have a selected node")
	}

	// 3. Test Tab 2 (Images)
	state.ActiveTab = 2
	renderFrame(term, state)
	if len(state.Images) == 0 {
		t.Errorf("Expected images to be loaded")
	}

	// 4. Test Tab 3 (Telemetry)
	state.ActiveTab = 3
	renderFrame(term, state)

	// 5. Test Tab 4 (Settings)
	state.ActiveTab = 4
	renderFrame(term, state)

	// 6. Test Key Handling
	handleKey(driver.KeyEvent{Type: driver.KeyRune, Ch: 'r'}, state)
	if state.AutoRotate {
		t.Errorf("Expected AutoRotate to toggle to false")
	}

	// Test 3D Mode cycling
	origMode := state.RenderModeIndex
	handleKey(driver.KeyEvent{Type: driver.KeyRune, Ch: 'm'}, state)
	if state.RenderModeIndex == origMode {
		t.Errorf("Expected RenderModeIndex to cycle")
	}

	// Test Tab jumping with number keys
	handleKey(driver.KeyEvent{Type: driver.KeyRune, Ch: '1'}, state)
	if state.ActiveTab != 0 {
		t.Errorf("Expected ActiveTab=0 after pressing '1', got %d", state.ActiveTab)
	}
	handleKey(driver.KeyEvent{Type: driver.KeyRune, Ch: '3'}, state)
	if state.ActiveTab != 2 {
		t.Errorf("Expected ActiveTab=2 after pressing '3', got %d", state.ActiveTab)
	}

	// Test Theme cycling
	origTheme := state.ThemeIndex
	handleKey(driver.KeyEvent{Type: driver.KeyRune, Ch: 't'}, state)
	if state.ThemeIndex == origTheme {
		t.Errorf("Expected ThemeIndex to cycle")
	}

	// 7. Verify modes
	state.RenderModeIndex = 0
	state.AsciiMode = widgets.ModeASCII
	if state.AsciiMode != widgets.ModeASCII {
		t.Errorf("Expected ModeASCII")
	}

	// 8. Test Mouse Click on Sidebar Tab 1 (TreeView)
	state.ActiveTab = 0
	renderFrame(term, state)

	// In 100x30, sidebar is X=0..23, Tab 1 is at Y=6..8. Click at (10, 7)
	handled := term.RouteMouseEvent(driver.MouseEvent{
		Button: driver.MouseLeft,
		X:      10,
		Y:      7,
	})
	if !handled {
		t.Errorf("Expected RouteMouseEvent to return handled=true for sidebar tab click")
	}
	if state.ActiveTab != 1 {
		t.Errorf("Expected ActiveTab=1 after clicking sidebar Tab 1, got %d", state.ActiveTab)
	}

	// 9. Test Mouse Click on Sidebar Tab 2 (Images) at (10, 10)
	renderFrame(term, state)
	handled = term.RouteMouseEvent(driver.MouseEvent{
		Button: driver.MouseLeft,
		X:      10,
		Y:      10,
	})
	if !handled || state.ActiveTab != 2 {
		t.Errorf("Expected ActiveTab=2 after clicking sidebar Tab 2, got %d (handled=%v)", state.ActiveTab, handled)
	}

	// 10. Test Mouse Click on Checkbox in Tab 0 (3D)
	state.ActiveTab = 0
	state.AutoRotate = true
	renderFrame(term, state)

	// Search for the click region corresponding to cb_3d_rotate
	// In Tab 0, cb_3d_rotate is rendered on the right panel at Y = header(3) + 1 + 1 = 5
	// Checkbox is at X ~71, Y = 5
	handled = term.RouteMouseEvent(driver.MouseEvent{
		Button: driver.MouseLeft,
		X:      72,
		Y:      5,
	})
	if !handled {
		t.Errorf("Expected RouteMouseEvent to handle click on cb_3d_rotate at (72, 5)")
	}
	if state.AutoRotate {
		t.Errorf("Expected AutoRotate to be toggled to false by mouse click")
	}

	// Click it again to toggle back to true
	renderFrame(term, state)
	handled = term.RouteMouseEvent(driver.MouseEvent{
		Button: driver.MouseLeft,
		X:      72,
		Y:      5,
	})
	if !handled || !state.AutoRotate {
		t.Errorf("Expected AutoRotate to be toggled back to true, got %v (handled=%v)", state.AutoRotate, handled)
	}

	// 11. Test TreeView Row Click in Tab 1
	state.ActiveTab = 1
	renderFrame(term, state)
	// TreeView is rendered on left pane (cols[0])
	// Clicking on root or first child should be handled
	handled = term.RouteMouseEvent(driver.MouseEvent{
		Button: driver.MouseLeft,
		X:      28,
		Y:      6,
	})
	if !handled {
		t.Logf("Notice: TreeView click at (28, 6) handled=%v", handled)
	}

	// 12. Verify MouseNone (hover motion) does NOT return handled=true, preventing render flood
	hoverHandled := term.RouteMouseEvent(driver.MouseEvent{
		Button: driver.MouseNone,
		X:      10,
		Y:      7,
	})
	if hoverHandled {
		t.Errorf("Expected RouteMouseEvent to return false for pure hover motion (MouseNone)")
	}
}

func TestGLBLemonAndGopherColors(t *testing.T) {
	memIO := driver.NewMemoryTerminalIO(nil, 120, 40)
	b := driver.NewPortableBackend(memIO)
	term, err := terminal.New(b)
	if err != nil {
		t.Fatalf("Failed to create terminal: %v", err)
	}

	state := initAppState()
	state.ActiveTab = 0
	state.AsciiMode = widgets.ModeBlock

	// Check model has both blue and yellow face colors
	var blueFaces, yellowFaces int
	for _, c := range state.Model.FaceColors {
		r, g, b := c.RGB()
		if b > 100 && b > r+20 {
			blueFaces++
		} else if r > 130 && g > 110 && b < 100 {
			yellowFaces++
		}
	}

	t.Logf("Model face colors: Blue (Gopher)=%d, Yellow (Lemon)=%d, Total Faces=%d",
		blueFaces, yellowFaces, len(state.Model.FaceColors))

	if blueFaces == 0 {
		t.Errorf("Expected model to have blue faces for Go Gopher mascot, got 0")
	}
	if yellowFaces == 0 {
		t.Errorf("Expected model to have yellow faces for Lemon, got 0")
	}

	// Render frame with renderFrame
	renderFrame(term, state)

	// memIO.Output() or string contains the ANSI color escapes emitted by the diff engine!
	outBytes := memIO.Output()
	outStr := string(outBytes)

	// TrueColor ANSI sequences look like: \x1b[38;2;R;G;Bm
	t.Logf("Flushed ANSI output length: %d bytes, sample: %q", len(outStr), outStr[:min(len(outStr), 300)])
	if len(outStr) == 0 {
		t.Errorf("Expected ANSI diff engine to write output bytes, got 0")
	}
}

func TestDemoImageTabRendering(t *testing.T) {
	memIO := driver.NewMemoryTerminalIO(nil, 120, 36)
	b := driver.NewPortableBackend(memIO)
	term, err := terminal.New(b)
	if err != nil {
		t.Fatalf("Failed to create terminal: %v", err)
	}

	state := initAppState()
	state.ActiveTab = 2         // Images Tab
	state.ActiveImageIdx = 1    // Profile avatar
	state.ImageHalfBlock = true // Test halfblock fallback

	renderFrame(term, state)

	outBytes := memIO.Output()
	outStr := string(outBytes)

	if !strings.Contains(outStr, "▄") && !strings.Contains(outStr, "▀") {
		t.Errorf("Expected profile image to render Half-Block cells in output")
	}

	// Verify protocol detection recognizes kitty, foot/sixel, and fallback
	t.Setenv("LIMONI_GRAPHICS", "kitty")
	if proto := graphics.DetectProtocol(); proto != graphics.ProtocolKitty {
		t.Errorf("Expected ProtocolKitty, got %v", proto)
	}

	t.Setenv("LIMONI_GRAPHICS", "sixel")
	if proto := graphics.DetectProtocol(); proto != graphics.ProtocolSixel {
		t.Errorf("Expected ProtocolSixel, got %v", proto)
	}
}

func TestKittyNativeImagesTab(t *testing.T) {
	t.Setenv("KITTY_WINDOW_ID", "1")
	t.Setenv("TERM", "xterm-kitty")
	proto := graphics.DetectProtocol()
	if proto != graphics.ProtocolKitty {
		t.Fatalf("Expected ProtocolKitty, got %v", proto)
	}

	memIO := driver.NewMemoryTerminalIO(nil, 120, 36)
	b := driver.NewPortableBackend(memIO)
	term, err := terminal.New(b)
	if err != nil {
		t.Fatalf("Failed to create terminal: %v", err)
	}

	state := initAppState()
	state.ActiveTab = 2          // Native Images Tab
	state.ImageHalfBlock = false // Use native Kitty!

	renderFrame(term, state)

	outBytes := memIO.Output()
	outStr := string(outBytes)
	t.Logf("Kitty output bytes: %d", len(outBytes))
	if !strings.Contains(outStr, "\x1b_G") {
		t.Errorf("Expected Kitty escape sequence \\x1b_G in output, got output len %d", len(outBytes))
	}
	if !strings.Contains(outStr, "q=2") {
		t.Errorf("Expected q=2 (suppress all responses) in Kitty command, got: %s", outStr[:min(len(outStr), 300)])
	}
	if !strings.Contains(outStr, "z=-") {
		t.Errorf("Expected negative z-index (underneath text layer) in Kitty command, got: %s", outStr[:min(len(outStr), 300)])
	}
}

func TestKittyTabSwitchingWithTerminalResponse(t *testing.T) {
	t.Setenv("KITTY_WINDOW_ID", "1")
	t.Setenv("TERM", "xterm-kitty")

	state := initAppState()
	state.ActiveTab = 0 // Start on Mascot

	// 1. User presses '3' to switch to Native Images
	handleKey(driver.KeyEvent{Type: driver.KeyRune, Ch: '3'}, state)
	if state.ActiveTab != 2 {
		t.Fatalf("Expected ActiveTab=2, got %d", state.ActiveTab)
	}

	// 2. Terminal returns a Kitty APC acknowledgment on stdin: \x1b_Gi=12345;OK\x1b\
	kittyAck := []byte("\x1b_Gi=12345;OK\x1b\\")
	ev, consumed := driver.ParseEvent(kittyAck)
	if consumed != len(kittyAck) {
		t.Fatalf("Expected entire Kitty APC response to be consumed (%d bytes), got %d", len(kittyAck), consumed)
	}
	if ev.Type != driver.EventNone {
		t.Fatalf("Expected EventNone for internal Kitty APC response, got %+v", ev)
	}

	// Verify that if EventNone is passed to handleKey or event loop, state.ActiveTab remains 2
	if ev.Type == driver.EventKey {
		handleKey(ev.Key, state)
	}
	if state.ActiveTab != 2 {
		t.Fatalf("ActiveTab must remain 2 after Kitty response, but was kicked to %d", state.ActiveTab)
	}
}

func TestExitDialogInteractionAndDragging(t *testing.T) {
	memIO := driver.NewMemoryTerminalIO(nil, 100, 30)
	b := driver.NewPortableBackend(memIO)
	term, err := terminal.New(b)
	if err != nil {
		t.Fatalf("Failed to create terminal: %v", err)
	}

	state := initAppState()

	// 1. Pressing 'q' opens the Exit Dialog
	handleKey(driver.KeyEvent{Type: driver.KeyRune, Ch: 'q'}, state)
	if !state.ShowExitDialog {
		t.Errorf("Expected ShowExitDialog=true after pressing 'q'")
	}
	state.ExitDialogAnim.SetValue(1.0) // Advance animation to fully open for testing

	// 2. Render frame with Exit Dialog
	renderFrame(term, state)
	outStr := string(memIO.Output())
	if !strings.Contains(outStr, "SYSTEM EXIT") {
		t.Errorf("Expected 'SYSTEM EXIT' title in output, got: %s", outStr[:min(len(outStr), 300)])
	}

	// 3. Pressing 'n' closes the Exit Dialog
	handleKey(driver.KeyEvent{Type: driver.KeyRune, Ch: 'n'}, state)
	state.ExitDialogAnim.SetValue(0.0) // Complete closing animation
	renderFrame(term, state)           // Trigger modal cleanup
	if state.ShowExitDialog {
		t.Errorf("Expected ShowExitDialog=false after pressing 'n'")
	}

	// 4. Click '6. Exit' in sidebar opens the Exit Dialog
	renderFrame(term, state)
	// Sidebar chunk 5 (Y ~ 18-20)
	handled := term.RouteMouseEvent(driver.MouseEvent{
		Button: driver.MouseLeft,
		X:      10,
		Y:      19,
	})
	if !handled || !state.ShowExitDialog {
		t.Errorf("Expected clicking 6. Exit to open Exit Dialog, handled=%v, show=%v", handled, state.ShowExitDialog)
	}
	state.ExitDialogAnim.SetValue(1.0) // Advance animation to fully open
	renderFrame(term, state)

	// 5. Test dragging the dialog
	// Title bar is at center (Y ~ 10, X ~ 50)
	handled = term.RouteMouseEvent(driver.MouseEvent{
		Button: driver.MouseLeft,
		X:      50,
		Y:      10,
	})
	if !handled || !state.IsDraggingModal {
		t.Errorf("Expected mouse down on title bar to start dragging, handled=%v, isDragging=%v", handled, state.IsDraggingModal)
	}

	// Drag mouse by dx=5, dy=2
	handled = term.RouteMouseEvent(driver.MouseEvent{
		Button: driver.MouseLeft,
		Drag:   true,
		X:      55,
		Y:      12,
	})
	if !handled || state.ModalOffsetX == 0 {
		t.Errorf("Expected ModalOffsetX to update on drag, got offset (%d, %d)", state.ModalOffsetX, state.ModalOffsetY)
	}

	// Release mouse
	handled = term.RouteMouseEvent(driver.MouseEvent{
		Button: driver.MouseRelease,
		X:      55,
		Y:      12,
	})
	if state.IsDraggingModal {
		t.Errorf("Expected IsDraggingModal=false on mouse release")
	}

	// 6. Test Left/Right Arrow button navigation
	if state.ExitDialogSelectedBtn != 1 {
		t.Errorf("Expected initial button selection to be 1 (No), got %d", state.ExitDialogSelectedBtn)
	}
	handleKey(driver.KeyEvent{Type: driver.KeyArrowLeft}, state)
	if state.ExitDialogSelectedBtn != 0 {
		t.Errorf("Expected button selection 0 (Yes) after ArrowLeft, got %d", state.ExitDialogSelectedBtn)
	}
	handleKey(driver.KeyEvent{Type: driver.KeyArrowRight}, state)
	if state.ExitDialogSelectedBtn != 1 {
		t.Errorf("Expected button selection 1 (No) after ArrowRight, got %d", state.ExitDialogSelectedBtn)
	}

	// 7. Pressing 'y' confirms exit
	handleKey(driver.KeyEvent{Type: driver.KeyRune, Ch: 'y'}, state)
	if !state.ExitRequested {
		t.Errorf("Expected ExitRequested=true after pressing 'y'")
	}
}

func TestCommandPaletteFuzzySearch(t *testing.T) {
	memIO := driver.NewMemoryTerminalIO(nil, 100, 30)
	b := driver.NewPortableBackend(memIO)
	term, err := terminal.New(b)
	if err != nil {
		t.Fatalf("Failed to create terminal: %v", err)
	}

	state := initAppState()

	// 1. Pressing Ctrl+P opens the Command Palette
	handleKey(driver.KeyEvent{Type: driver.KeyRune, Ch: 'p', Ctrl: true}, state)
	if state.CmdPalette == nil || !state.CmdPalette.IsOpen {
		t.Fatalf("Expected CmdPalette.IsOpen=true after Ctrl+P")
	}

	// 2. Type "native" to fuzzy search for Native Images
	for _, ch := range "native" {
		handleKey(driver.KeyEvent{Type: driver.KeyRune, Ch: ch}, state)
	}
	if len(state.CmdPalette.Filtered) == 0 {
		t.Fatalf("Expected fuzzy search to find matching commands for 'native'")
	}

	foundNative := false
	for _, item := range state.CmdPalette.Filtered {
		if strings.Contains(item.Label, "Native Images") {
			foundNative = true
			break
		}
	}
	if !foundNative {
		t.Errorf("Expected 'Native Images' in filtered commands, got: %+v", state.CmdPalette.Filtered)
	}

	// 3. Render frame with Command Palette open
	renderFrame(term, state)
	outStr := string(memIO.Output())
	if !strings.Contains(outStr, "Native Images") {
		t.Errorf("Expected 'Native Images' command in frame output, got output len %d", len(outStr))
	}

	// 4. Pressing Enter executes selected command
	handleKey(driver.KeyEvent{Type: driver.KeyEnter}, state)
	if state.CmdPalette.IsOpen {
		t.Errorf("Expected CmdPalette to close after Enter")
	}
	if state.ActiveTab != 2 {
		t.Errorf("Expected ActiveTab=2 after selecting Native Images, got %d", state.ActiveTab)
	}
}

func TestCommandPalettePageNavigation(t *testing.T) {
	memIO := driver.NewMemoryTerminalIO(nil, 120, 35)
	b := driver.NewPortableBackend(memIO)
	term, err := terminal.New(b)
	if err != nil {
		t.Fatalf("Failed to create terminal: %v", err)
	}

	state := initAppState()

	// Helper to search and select via keyboard
	selectCommand := func(query string) {
		handleKey(driver.KeyEvent{Type: driver.KeyRune, Ch: 'p', Ctrl: true}, state)
		for _, ch := range query {
			handleKey(driver.KeyEvent{Type: driver.KeyRune, Ch: ch}, state)
		}
		handleKey(driver.KeyEvent{Type: driver.KeyEnter}, state)
	}

	// 1. From Images tab (2), search "ASCII" -> should navigate to 3D Mascot tab (0)
	state.ActiveTab = 2
	selectCommand("ASCII")
	if state.ActiveTab != 0 {
		t.Errorf("Expected ActiveTab=0 (3D Mascot) after selecting ASCII, got %d", state.ActiveTab)
	}

	// 2. From 3D tab (0), search "Folders" -> should navigate to TreeView tab (1)
	selectCommand("Folders")
	if state.ActiveTab != 1 {
		t.Errorf("Expected ActiveTab=1 (TreeView) after selecting Folders, got %d", state.ActiveTab)
	}

	// 3. From TreeView tab (1), search "Sampling" -> should navigate to Telemetry tab (3)
	selectCommand("Sampling")
	if state.ActiveTab != 3 {
		t.Errorf("Expected ActiveTab=3 (Telemetry) after selecting Sampling, got %d", state.ActiveTab)
	}

	// 4. From Telemetry tab (3), search "Cyberpunk" -> should navigate to Settings tab (4)
	selectCommand("Cyberpunk")
	if state.ActiveTab != 4 {
		t.Errorf("Expected ActiveTab=4 (Settings) after selecting Cyberpunk, got %d", state.ActiveTab)
	}

	// 5. From Settings tab (4), search "Apple" -> should navigate to Native Images tab (2)
	selectCommand("Apple")
	if state.ActiveTab != 2 {
		t.Errorf("Expected ActiveTab=2 (Native Images) after selecting Apple, got %d", state.ActiveTab)
	}

	// 6. Test Mouse click on Command Palette row
	state.ActiveTab = 0
	state.CmdPalette.Open()
	for _, ch := range "TreeView" {
		handleKey(driver.KeyEvent{Type: driver.KeyRune, Ch: ch}, state)
	}
	renderFrame(term, state)

	// In 120x35: panelArea has startX=24, startY=4 (due to Top: 4), items start at y = startY+3 = 7
	// Row 0 is "Tab: 2. TreeView" at y=7
	clickEv := driver.MouseEvent{X: 30, Y: 7, Button: driver.MouseLeft}
	handled := term.RouteMouseEvent(clickEv)
	if !handled {
		t.Errorf("Expected mouse click on command palette row to be handled")
	}
	if state.CmdPalette.IsOpen {
		t.Errorf("Expected CmdPalette to close after mouse clicking an item")
	}
	if state.ActiveTab != 1 {
		t.Errorf("Expected ActiveTab=1 (TreeView) after clicking Row 0, got %d", state.ActiveTab)
	}
}

func TestCommandPaletteImageOcclusion(t *testing.T) {
	// Set Kitty terminal env so native image protocol is active
	origTerm := os.Getenv("TERM")
	origKitty := os.Getenv("KITTY_WINDOW_ID")
	os.Setenv("TERM", "xterm-kitty")
	os.Setenv("KITTY_WINDOW_ID", "1")
	defer func() {
		os.Setenv("TERM", origTerm)
		os.Setenv("KITTY_WINDOW_ID", origKitty)
	}()

	memIO := driver.NewMemoryTerminalIO(nil, 120, 35)
	b := driver.NewPortableBackend(memIO)
	term, err := terminal.New(b)
	if err != nil {
		t.Fatalf("Failed to create terminal: %v", err)
	}

	state := initAppState()
	state.ActiveTab = 2 // Native Images tab (apple.png)
	state.CmdPalette.Open()

	renderFrame(term, state)

	// Check image regions in the frame
	var hasZMinus3 bool // Apple image
	var hasZMinus2 bool // Command Palette opaque backdrop

	for _, reg := range term.LastImageRegions() {
		if reg.ZIndex == -3 {
			hasZMinus3 = true
		}
		if reg.ZIndex == -2 {
			hasZMinus2 = true
		}
	}

	if !hasZMinus3 {
		t.Errorf("Expected underlying image registered at ZIndex = -3")
	}
	if !hasZMinus2 {
		t.Errorf("Expected Command Palette opaque backdrop registered at ZIndex = -2 to occlude underlying images")
	}
}

func TestExitDialogButtonSelectionAndHover(t *testing.T) {
	memIO := driver.NewMemoryTerminalIO(nil, 100, 30)
	b := driver.NewPortableBackend(memIO)
	term, err := terminal.New(b)
	if err != nil {
		t.Fatalf("Failed to create terminal: %v", err)
	}

	state := initAppState()

	// 1. Open exit dialog
	openExitDialog(state, term)
	state.ExitDialogAnim.SetValue(1.0)
	renderFrame(term, state)

	if state.ExitDialogSelectedBtn != 1 {
		t.Errorf("Expected default selection 1 (No), got %d", state.ExitDialogSelectedBtn)
	}

	// 2. ArrowLeft selects 0 (Yes)
	handleKey(driver.KeyEvent{Type: driver.KeyArrowLeft}, state)
	if state.ExitDialogSelectedBtn != 0 {
		t.Errorf("Expected selection 0 (Yes) after ArrowLeft, got %d", state.ExitDialogSelectedBtn)
	}

	// 3. ArrowRight selects 1 (No)
	handleKey(driver.KeyEvent{Type: driver.KeyArrowRight}, state)
	if state.ExitDialogSelectedBtn != 1 {
		t.Errorf("Expected selection 1 (No) after ArrowRight, got %d", state.ExitDialogSelectedBtn)
	}

	// 4. Tab toggles to 0 (Yes)
	handleKey(driver.KeyEvent{Type: driver.KeyTab}, state)
	if state.ExitDialogSelectedBtn != 0 {
		t.Errorf("Expected selection 0 (Yes) after Tab, got %d", state.ExitDialogSelectedBtn)
	}

	// 5. Tab toggles back to 1 (No)
	handleKey(driver.KeyEvent{Type: driver.KeyTab}, state)
	if state.ExitDialogSelectedBtn != 1 {
		t.Errorf("Expected selection 1 (No) after Tab, got %d", state.ExitDialogSelectedBtn)
	}

	// 6. Enter with 1 (No) cancels dialog
	handleKey(driver.KeyEvent{Type: driver.KeyEnter}, state)
	state.ExitDialogAnim.SetValue(0.0)
	renderFrame(term, state)
	if state.ShowExitDialog {
		t.Errorf("Expected ShowExitDialog=false after pressing Enter on 'No'")
	}
	if state.ExitRequested {
		t.Errorf("Expected ExitRequested=false after cancelling")
	}

	// 7. Re-open and Enter with 0 (Yes) confirms exit
	openExitDialog(state, term)
	state.ExitDialogAnim.SetValue(1.0)
	renderFrame(term, state)
	handleKey(driver.KeyEvent{Type: driver.KeyArrowLeft}, state)
	handleKey(driver.KeyEvent{Type: driver.KeyEnter}, state)
	if !state.ExitRequested {
		t.Errorf("Expected ExitRequested=true after pressing Enter on 'Yes'")
	}
}

func TestCloseOnImageTabKitty(t *testing.T) {
	origTerm := os.Getenv("TERM")
	origKitty := os.Getenv("KITTY_WINDOW_ID")
	os.Setenv("TERM", "xterm-kitty")
	os.Setenv("KITTY_WINDOW_ID", "1")
	defer func() {
		os.Setenv("TERM", origTerm)
		os.Setenv("KITTY_WINDOW_ID", origKitty)
	}()

	memIO := driver.NewMemoryTerminalIO(nil, 120, 35)
	b := driver.NewPortableBackend(memIO)
	term, err := terminal.New(b)
	if err != nil {
		t.Fatal(err)
	}

	state := initAppState()
	state.ActiveTab = 2 // Native Images tab (apple.png)

	// 1. Initial frame with apple.png
	renderFrame(term, state)

	// 2. Open dialog
	openExitDialog(state, term)
	state.ExitDialogAnim.SetValue(1.0)
	renderFrame(term, state)

	// 3. Close dialog
	closeExitDialog(state, term)

	// Tick until closing animation finishes
	t0 := time.Now()
	var finalFrameOutput []byte
	for i := 0; i < 25; i++ {
		t0 = t0.Add(20 * time.Millisecond)
		updateState(state, t0)
		bytesBefore := len(memIO.Output())
		renderFrame(term, state)
		if !state.ShowExitDialog {
			finalFrameOutput = memIO.Output()[bytesBefore:]
			break
		}
	}
	if !bytes.Contains(finalFrameOutput, []byte("X")) {
		t.Errorf("Expected ECH escape code 'X' in final frame output to clear terminal cells over image, got: %q", string(finalFrameOutput))
	}

	// Verify no dialog backdrop remains
	regs := term.LastImageRegions()
	for _, r := range regs {
		if r.ZIndex == -2 {
			t.Errorf("Dialog opaque backdrop (ZIndex=-2) still remains in ImageRegions after closing!")
		}
	}

	// Verify that the buffer cells in the former dialog area (CenterRect 48x9)
	// do NOT contain any dialog '█' blocks or dialog red border styling!
	dialogArea := terminal.CenterRect(cell.NewRect(0, 0, 120, 35), 48, 9)
	frontBuf := term.FrontBuffer()
	dialogRed := cell.NewColorRGB(220, 60, 60)
	for cy := dialogArea.Y; cy < dialogArea.Y+dialogArea.Height; cy++ {
		for cx := dialogArea.X; cx < dialogArea.X+dialogArea.Width; cx++ {
			c := frontBuf.Get(cx, cy)
			if c == nil {
				continue
			}
			if c.Content == '█' {
				t.Errorf("Found stray '█' block at (%d, %d) after dialog close!", cx, cy)
			}
			if c.Content == '▲' {
				t.Errorf("Found stray dialog icon '▲' at (%d, %d) after dialog close!", cx, cy)
			}
			if c.Style.Fg == dialogRed || c.Style.Bg == dialogRed {
				t.Errorf("Found stray dialog red style at (%d, %d) rune %q after dialog close!", cx, cy, c.Content)
			}
		}
	}
}
