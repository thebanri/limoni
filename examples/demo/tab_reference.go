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

func drawReference(t *terminal.Terminal, f *terminal.Frame, state *AppState, theme widgets.Theme, mainColor, accentColor cell.Color, area cell.Rect) {
	rows := layout.NewFlexLayout(layout.Vertical, 1, layout.Fixed(6), layout.Fixed(6), layout.Fixed(6), layout.Fill()).Split(area)
	f.RegisterAccessibility(referenceAccessibilityTree(state, area))
	drawReferenceRuntime(f, state, theme, rows[0])
	drawReferenceInteraction(t, f, state, theme, rows[1])
	drawReferenceLayout(f, state, theme, rows[2])
	drawReferenceData(f, state, theme, accentColor, rows[3])
	_ = mainColor
}

func referenceAccessibilityTree(state *AppState, area cell.Rect) accessibility.AccessibilityNode {
	dataY := area.Y + 21
	dataHeight := uint16(0)
	if area.Height > 21 {
		dataHeight = area.Height - 21
	} else if area.Height > 0 {
		dataY = area.Y + area.Height - 1
		dataHeight = 1
	}
	return accessibility.AccessibilityNode{
		ID: "reference", Role: accessibility.RoleDialog, Label: "Limoni referans ve geliştirici araçları", Bounds: area,
		Children: []accessibility.AccessibilityNode{
			{ID: "runtime", Role: accessibility.RoleGeneric, Label: "Runtime CMD-MSG", Value: fmt.Sprintf("%d mesaj", state.ReferenceRuntimeMessages), Bounds: cell.NewRect(area.X, area.Y, area.Width, 6)},
			{ID: "interaction", Role: accessibility.RoleGeneric, Label: "Interaction Inspector", Value: state.ReferenceInteractionLastRoute, Bounds: cell.NewRect(area.X, area.Y+7, area.Width, 6)},
			{ID: "virtual-data", Role: accessibility.RoleList, Label: "Virtual data", Value: fmt.Sprintf("seçili satır %d", state.ReferenceSelectedRow), State: accessibility.StateBusy, Bounds: cell.NewRect(area.X, dataY, area.Width, dataHeight)},
		},
	}
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
	if state.ReferenceInteractionLast == "" {
		state.ReferenceInteractionLast = "henüz event yok"
	}
	if state.ReferenceInteractionHover == "" {
		state.ReferenceInteractionHover = "yok (panel dışı)"
	}
	history := strings.Join(state.ReferenceInteractionHistory, " | ")
	if history == "" {
		history = "henüz event yok"
	}
	text := fmt.Sprintf("Global event monitor | total: %d | focus: %s\nPointer: (%d,%d) | hover: %s | route: %s\nSon: %s\nHistory: %s", state.ReferenceInteractionEvents, t.FocusManager().Focused(), state.ReferenceInteractionPointerX, state.ReferenceInteractionPointerY, state.ReferenceInteractionHover, state.ReferenceInteractionLastRoute, state.ReferenceInteractionLast, history)
	f.RenderWidget(widgets.Block{Title: " INTERACTION INSPECTOR ", Borders: widgets.BorderAll, BorderSymbols: widgets.SymbolsRounded, BorderStyle: theme.Border, PaddingLeft: 1, Child: referenceLabel{text: text, style: theme.RoleStyle("text")}}, area)
	f.RegisterEventRegion(terminal.EventRegion{Area: area, ID: "reference.interaction", Phase: terminal.TargetPhase, Handler: func(ctx *terminal.EventContext) {
		state.ReferenceInteractionHover = ctx.RegionID
		if ctx.Mouse.Button == backend.MouseLeft && !ctx.Mouse.Drag {
			ctx.PreventDefault()
		}
	}, OnEnter: func(ctx *terminal.EventContext) {
		state.ReferenceInteractionHover = ctx.RegionID
	}, OnLeave: func(ctx *terminal.EventContext) {
		state.ReferenceInteractionHover = "yok (panel dışı)"
	}})
}

func drawReferenceLayout(f *terminal.Frame, state *AppState, theme widgets.Theme, area cell.Rect) {
	idealHeight := uint16(2 + (state.ReferenceLayoutPass % 4))
	measure := layout.Measure{MinWidth: 10, MinHeight: 1, IdealWidth: area.Width, IdealHeight: idealHeight, MaxWidth: area.Width, MaxHeight: area.Height, GrowPriority: 1 + state.ReferenceLayoutPass, Overflow: layout.OverflowClip}.Normalize(area)
	allocated := layout.Arrange(area, []layout.Measure{measure}, layout.Vertical, 0)[0]
	state.ReferenceLayoutAllocated = allocated
	if state.ReferenceLayoutLastAction == "" {
		state.ReferenceLayoutLastAction = "henüz ölçüm yapılmadı"
	}
	text := fmt.Sprintf("Bu panel gerçek measure/arrange sonucunu gösterir.\nTıkla: ideal yükseklik ve grow priority değişsin\nPass: %d | Last action: %s\nMin: %dx%d | Ideal: %dx%d | Max: %dx%d\nGrow priority: %d | Overflow: clip\nAllocated: X=%d Y=%d W=%d H=%d", state.ReferenceLayoutPass, state.ReferenceLayoutLastAction, measure.MinWidth, measure.MinHeight, measure.IdealWidth, measure.IdealHeight, measure.MaxWidth, measure.MaxHeight, measure.GrowPriority, allocated.X, allocated.Y, allocated.Width, allocated.Height)
	f.RenderWidget(widgets.Block{Title: " LAYOUT INSPECTOR ", Borders: widgets.BorderAll, BorderSymbols: widgets.SymbolsRounded, BorderStyle: theme.Border, PaddingLeft: 1, Child: referenceLabel{text: text, style: theme.RoleStyle("text")}}, area)
	f.RegisterEventRegion(terminal.EventRegion{Area: area, ID: "reference.layout", Phase: terminal.TargetPhase, Handler: func(ctx *terminal.EventContext) {
		if ctx.Mouse.Button == backend.MouseLeft && !ctx.Mouse.Drag {
			state.ReferenceLayoutPass = (state.ReferenceLayoutPass + 1) % 10
			state.ReferenceLayoutLastAction = fmt.Sprintf("click target=%s at (%d,%d)", ctx.RegionID, ctx.Mouse.X, ctx.Mouse.Y)
			ctx.PreventDefault()
		}
	}})
}

func drawReferenceData(f *terminal.Frame, state *AppState, theme widgets.Theme, accent cell.Color, area cell.Rect) {
	provider := referenceProvider{}
	if state.ReferenceDataState == nil {
		state.ReferenceDataState = widgets.NewVirtualDataState()
	}
	data := state.ReferenceDataState
	viewportHeight := int(area.Height) - 7
	if viewportHeight < 1 {
		viewportHeight = 1
	}
	_ = data.Refresh(nil, provider, state.ReferenceDataOffset, viewportHeight, 2)
	data.Select(provider.RowID(state.ReferenceSelectedRow))
	status, _ := data.Status()
	lastVisible := state.ReferenceDataOffset + viewportHeight - 1
	if lastVisible >= data.Count() {
		lastVisible = data.Count() - 1
	}
	mode := accessibility.Mode{HighContrast: theme.Colors.Primary == cell.NewColorRGB(255, 255, 0), ReducedMotion: false, ScreenReader: true, ASCIIOnly: state.ReferenceAccessibilityASCII}
	lineMode := f.AccessibilityLineMode(mode)
	previewLines := strings.Split(lineMode, "\n")
	if len(previewLines) > 3 {
		previewLines = previewLines[:3]
	}
	linePreview := strings.Join(previewLines, " / ")
	testTerm := testkit.NewTerminal(20, 2)
	testTerm.Draw(func(frame *terminal.Frame) { frame.Buffer.SetString(0, 0, "benchmark-ready", cell.Style{Fg: accent}) })
	snapshot := strings.ReplaceAll(testTerm.Snapshot(), "\n", " / ")
	text := fmt.Sprintf("Virtual data: %d kayıt | görünür: %d-%d | scroll: mouse wheel\nSatıra tıkla: seç (seçili satır: #%d) | Stable ID: %s\nStatus: %v | Accessibility: high-contrast=%t | screen-reader=line mode\nLine mode (%s): %s\nTestKit snapshot: %s | Benchmark runs: %d | Updated: %s", data.Count(), state.ReferenceDataOffset+1, lastVisible+1, state.ReferenceSelectedRow, data.Selected(), status, mode.HighContrast, map[bool]string{true: "ASCII", false: "Unicode"}[mode.ASCIIOnly], linePreview, snapshot, state.ReferenceBenchmarkRuns, time.Now().Format("15:04:05"))
	inner := cell.NewRect(area.X+1, area.Y+1, area.Width-2, area.Height-2)
	f.RenderWidget(widgets.Block{Title: " VIRTUAL DATA / ACCESSIBILITY / BENCHMARK ", Borders: widgets.BorderAll, BorderSymbols: widgets.SymbolsRounded, BorderStyle: cell.Style{Fg: accent}, PaddingLeft: 1, Child: referenceLabel{text: text, style: theme.RoleStyle("text")}}, area)
	if inner.Width > 2 && inner.Height > 1 {
		f.RenderWidget(widgets.VirtualDataView{State: data, Source: provider, First: 0, Prefetch: 2, Offset: &state.ReferenceDataOffset, OnSelect: func(index int, _ widgets.Row) { state.ReferenceSelectedRow = index; state.ReferenceBenchmarkRuns++ }, Style: theme.RoleStyle("muted"), SelectedStyle: cell.Style{Fg: accent, Modifier: cell.ModifierBold}}, cell.NewRect(inner.X, inner.Y+5, inner.Width, inner.Height-5))
	}
}

type referenceLabel struct {
	text  string
	style cell.Style
}

func (l referenceLabel) Draw(ctx cell.Context, buf *buffer.Buffer) {
	style := ctx.Style.Merge(l.style)
	for row, line := range strings.Split(l.text, "\n") {
		if row < int(ctx.Area.Height) {
			buf.SetString(ctx.Area.X, ctx.Area.Y+uint16(row), line, style)
		}
	}
}
func (l referenceLabel) SizeHint(max cell.Rect) (uint16, uint16) { return max.Width, max.Height }

type referenceProvider struct{}

func (referenceProvider) RowCount(context.Context) (int, error) { return 1000000, nil }
func (referenceProvider) RowAt(_ context.Context, index int) (widgets.Row, error) {
	return widgets.Row{ID: widgets.RowID(fmt.Sprintf("row-%d", index)), Text: fmt.Sprintf("#%06d  |  örnek kayıt %d  |  viewport cache", index, index)}, nil
}
func (referenceProvider) RowID(index int) widgets.RowID {
	return widgets.RowID(fmt.Sprintf("row-%d", index))
}
