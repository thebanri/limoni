package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/backend"
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
	drawReferenceRuntime(f, state, theme, rows[0])
	drawReferenceInteraction(t, f, state, theme, rows[1])
	drawReferenceLayout(f, state, theme, rows[2])
	drawReferenceData(f, state, theme, accentColor, rows[3])

	// Register a semantic root so TestKit and accessibility inspectors can see
	// the same metadata used by the reference screen.
	f.RegisterAccessibility(accessibility.AccessibilityNode{
		ID: "reference", Role: accessibility.RoleDialog, Label: "Limoni referans ve geliştirici araçları", Bounds: area,
	})
	_ = mainColor
	_ = state
}

func drawReferenceRuntime(f *terminal.Frame, state *AppState, theme widgets.Theme, area cell.Rect) {
	message := fmt.Sprintf("Bu panel uygulama runtime durumunu gösterir.\nBu panele tıkla: örnek Msg gönder\nMesaj sayısı: %d\nCmd scheduler: active\nCancellation: context.Context\nRedraw: coalesced", state.ReferenceRuntimeMessages)
	f.RenderWidget(widgets.Block{Title: " RUNTIME / CMD-MSG ", Borders: widgets.BorderAll, BorderSymbols: widgets.SymbolsRounded, BorderStyle: theme.Border, PaddingLeft: 1, Child: referenceLabel{text: message, style: theme.RoleStyle("text")}}, area)
	f.RegisterClickHandler(area, func(ev backend.MouseEvent) {
		if ev.Button == backend.MouseLeft && !ev.Drag {
			state.ReferenceRuntimeMessages++
		}
	})
}

func drawReferenceInteraction(t *terminal.Terminal, f *terminal.Frame, state *AppState, theme widgets.Theme, area cell.Rect) {
	text := fmt.Sprintf("Bu panel fare/odak olaylarını gösterir.\nBu panele tıkla: event kaydet\nFocused: %s\nHovered region: %s\nSon event: %s\nPropagation: capture → target → bubble", t.FocusManager().Focused(), f.HoveredRegionID(), state.ReferenceInteractionLast)
	f.RenderWidget(widgets.Block{Title: " INTERACTION INSPECTOR ", Borders: widgets.BorderAll, BorderSymbols: widgets.SymbolsRounded, BorderStyle: theme.Border, PaddingLeft: 1, Child: referenceLabel{text: text, style: theme.RoleStyle("text")}}, area)
	f.RegisterClickHandler(area, func(ev backend.MouseEvent) {
		if ev.Button == backend.MouseLeft && !ev.Drag {
			state.ReferenceInteractionLast = fmt.Sprintf("click (%d,%d)", ev.X, ev.Y)
		}
	})
}

func drawReferenceLayout(f *terminal.Frame, state *AppState, theme widgets.Theme, area cell.Rect) {
	measure := layout.Measure{MinWidth: 10, IdealWidth: area.Width, IdealHeight: area.Height, MaxWidth: area.Width, MaxHeight: area.Height, GrowPriority: 1 + state.ReferenceLayoutPass, Overflow: layout.OverflowClip}.Normalize(area)
	text := fmt.Sprintf("Bu panel measure/arrange sonucunu gösterir.\nBu panele tıkla: ölçüm turunu değiştir\nMeasure pass: %d\nMeasured: %dx%d\nAllocated: X=%d Y=%d W=%d H=%d\nOverflow: clip", state.ReferenceLayoutPass, measure.IdealWidth, measure.IdealHeight, area.X, area.Y, area.Width, area.Height)
	f.RenderWidget(widgets.Block{Title: " LAYOUT INSPECTOR ", Borders: widgets.BorderAll, BorderSymbols: widgets.SymbolsRounded, BorderStyle: theme.Border, PaddingLeft: 1, Child: referenceLabel{text: text, style: theme.RoleStyle("text")}}, area)
	f.RegisterClickHandler(area, func(ev backend.MouseEvent) {
		if ev.Button == backend.MouseLeft && !ev.Drag {
			state.ReferenceLayoutPass = (state.ReferenceLayoutPass + 1) % 10
		}
	})
}

func drawReferenceData(f *terminal.Frame, state *AppState, theme widgets.Theme, accent cell.Color, area cell.Rect) {
	provider := referenceProvider{}
	data := widgets.NewVirtualDataState()
	_ = data.Refresh(nil, provider, 0, int(area.Height)-2, 2)
	data.Select(provider.RowID(state.ReferenceSelectedRow))
	status, _ := data.Status()
	mode := accessibility.Mode{HighContrast: theme.Colors.Primary == cell.NewColorRGB(255, 255, 0), ReducedMotion: false}
	testTerm := testkit.NewTerminal(20, 2)
	testTerm.Draw(func(frame *terminal.Frame) { frame.Buffer.SetString(0, 0, "benchmark-ready", cell.Style{Fg: accent}) })
	snapshot := strings.ReplaceAll(testTerm.Snapshot(), "\n", " / ")
	text := fmt.Sprintf("Bu panel virtual data, accessibility ve TestKit örneğidir.\nTıkla: satır seç / benchmark sayacını artır\nVirtual rows: %d (%v) | Stable ID: %s\nAccessibility: high-contrast=%t | Snapshot: %s\nBenchmark runs: %d | Updated: %s", data.Count(), status, data.Selected(), mode.HighContrast, snapshot, state.ReferenceBenchmarkRuns, time.Now().Format("15:04:05"))
	inner := cell.NewRect(area.X+1, area.Y+1, area.Width-2, area.Height-2)
	f.RenderWidget(widgets.Block{Title: " VIRTUAL DATA / ACCESSIBILITY / BENCHMARK ", Borders: widgets.BorderAll, BorderSymbols: widgets.SymbolsRounded, BorderStyle: cell.Style{Fg: accent}, PaddingLeft: 1, Child: referenceLabel{text: text, style: theme.RoleStyle("text")}}, area)
	if inner.Width > 2 && inner.Height > 1 {
		f.RenderWidget(widgets.VirtualDataView{State: data, Source: provider, First: state.ReferenceSelectedRow, Prefetch: 2, Style: theme.RoleStyle("muted"), SelectedStyle: cell.Style{Fg: accent, Modifier: cell.ModifierBold}}, cell.NewRect(inner.X, inner.Y+5, inner.Width, inner.Height-5))
	}
	f.RegisterClickHandler(area, func(ev backend.MouseEvent) {
		if ev.Button == backend.MouseLeft && !ev.Drag {
			state.ReferenceSelectedRow = (state.ReferenceSelectedRow + 1) % 1000000
			state.ReferenceBenchmarkRuns++
		}
	})
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
