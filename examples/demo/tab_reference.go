package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/layout"
	"github.com/thebanri/limoni/testkit"
	"github.com/thebanri/limoni/widgets"
)

// drawReference is the demo's compact reference application for the roadmap
// foundations. It intentionally uses the public runtime/TestKit/layout/data
// APIs instead of duplicating their internals.
func drawReference(t *terminal.Terminal, f *terminal.Frame, state *AppState, theme widgets.Theme, mainColor, accentColor cell.Color, area cell.Rect) {
	rows := layout.NewFlexLayout(layout.Vertical, 1, layout.Fixed(6), layout.Fixed(6), layout.Fixed(6), layout.Fill()).Split(area)
	drawReferenceRuntime(f, theme, rows[0])
	drawReferenceInteraction(t, f, theme, rows[1])
	drawReferenceLayout(f, theme, rows[2])
	drawReferenceData(f, theme, accentColor, rows[3])

	// Register a semantic root so TestKit and accessibility inspectors can see
	// the same metadata used by the reference screen.
	f.RegisterAccessibility(accessibility.AccessibilityNode{
		ID: "reference", Role: accessibility.RoleDialog, Label: "Limoni referans ve geliştirici araçları", Bounds: area,
	})
	_ = mainColor
	_ = state
}

func drawReferenceRuntime(f *terminal.Frame, theme widgets.Theme, area cell.Rect) {
	message := "Bu panel uygulama runtime durumunu gösterir.\nMsg queue: ready\nCmd scheduler: active\nCancellation: context.Context\nRedraw: coalesced"
	f.RenderWidget(widgets.Block{Title: " RUNTIME / CMD-MSG ", Borders: widgets.BorderAll, BorderSymbols: widgets.SymbolsRounded, BorderStyle: theme.Border, PaddingLeft: 1, Child: referenceLabel{text: message, style: theme.RoleStyle("text")}}, area)
}

func drawReferenceInteraction(t *terminal.Terminal, f *terminal.Frame, theme widgets.Theme, area cell.Rect) {
	f.RegisterEventRegion(terminal.EventRegion{Area: area, ID: "reference-interaction", Phase: terminal.TargetPhase, Handler: func(ctx *terminal.EventContext) { _ = ctx }})
	text := fmt.Sprintf("Bu panel fare/odak olaylarını gösterir.\nFocused: %s\nHovered region: %s\nMouse capture: supported\nPropagation: capture → target → bubble", t.FocusManager().Focused(), f.HoveredRegionID())
	f.RenderWidget(widgets.Block{Title: " INTERACTION INSPECTOR ", Borders: widgets.BorderAll, BorderSymbols: widgets.SymbolsRounded, BorderStyle: theme.Border, PaddingLeft: 1, Child: referenceLabel{text: text, style: theme.RoleStyle("text")}}, area)
}

func drawReferenceLayout(f *terminal.Frame, theme widgets.Theme, area cell.Rect) {
	measure := layout.Measure{MinWidth: 10, IdealWidth: area.Width, IdealHeight: area.Height, MaxWidth: area.Width, MaxHeight: area.Height, GrowPriority: 1, Overflow: layout.OverflowClip}.Normalize(area)
	text := fmt.Sprintf("Bu panel measure/arrange sonucunu gösterir.\nMeasured: %dx%d\nAllocated: X=%d Y=%d W=%d H=%d\nOverflow: clip\nGrow priority: %d", measure.IdealWidth, measure.IdealHeight, area.X, area.Y, area.Width, area.Height, measure.GrowPriority)
	f.RenderWidget(widgets.Block{Title: " LAYOUT INSPECTOR ", Borders: widgets.BorderAll, BorderSymbols: widgets.SymbolsRounded, BorderStyle: theme.Border, PaddingLeft: 1, Child: referenceLabel{text: text, style: theme.RoleStyle("text")}}, area)
}

func drawReferenceData(f *terminal.Frame, theme widgets.Theme, accent cell.Color, area cell.Rect) {
	provider := referenceProvider{}
	data := widgets.NewVirtualDataState()
	_ = data.Refresh(nil, provider, 0, int(area.Height)-2, 2)
	data.Select(provider.RowID(42))
	status, _ := data.Status()
	mode := accessibility.Mode{HighContrast: theme.Colors.Primary == cell.NewColorRGB(255, 255, 0), ReducedMotion: false}
	testTerm := testkit.NewTerminal(20, 2)
	testTerm.Draw(func(frame *terminal.Frame) { frame.Buffer.SetString(0, 0, "benchmark-ready", cell.Style{Fg: accent}) })
	snapshot := strings.ReplaceAll(testTerm.Snapshot(), "\n", " / ")
	text := fmt.Sprintf("Bu panel virtual data, accessibility ve TestKit örneğidir.\nVirtual rows: %d (%v)\nStable ID: %s\nAccessibility: high-contrast=%t\nTestKit snapshot: %s\nBenchmark durumu: hazır\nUpdated: %s", data.Count(), status, data.Selected(), mode.HighContrast, snapshot, time.Now().Format("15:04:05"))
	f.RenderWidget(widgets.Block{Title: " VIRTUAL DATA / ACCESSIBILITY / BENCHMARK ", Borders: widgets.BorderAll, BorderSymbols: widgets.SymbolsRounded, BorderStyle: cell.Style{Fg: accent}, PaddingLeft: 1, Child: referenceLabel{text: text, style: theme.RoleStyle("text")}}, area)
}

type referenceLabel struct {
	text  string
	style cell.Style
}

func (l referenceLabel) Draw(ctx cell.Context, buf *buffer.Buffer) {
	style := ctx.Style.Merge(l.style)
	for row, line := range strings.Split(l.text, "\n") {
		if row >= int(ctx.Area.Height) {
			break
		}
		buf.SetString(ctx.Area.X, ctx.Area.Y+uint16(row), line, style)
	}
}
func (l referenceLabel) SizeHint(max cell.Rect) (uint16, uint16) { return max.Width, max.Height }

type referenceProvider struct{}

func (referenceProvider) RowCount(context.Context) (int, error) { return 1000000, nil }
func (referenceProvider) RowAt(_ context.Context, index int) (widgets.Row, error) {
	return widgets.Row{ID: widgets.RowID(fmt.Sprintf("row-%d", index)), Text: "lazy row"}, nil
}
func (referenceProvider) RowID(index int) widgets.RowID {
	return widgets.RowID(fmt.Sprintf("row-%d", index))
}
