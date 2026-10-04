package widgets

import (
	"image"
	"testing"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

func prepareBenchmarkEnv() (*buffer.Buffer, cell.Context) {
	area := cell.NewRect(0, 0, 80, 25)
	buf := buffer.NewBuffer(area)
	style := cell.Style{}
	style.Reset()
	ctx := cell.NewContext(area, style)
	// Mock focus context fields to avoid early returns
	ctx.FocusedID = "widget_id"
	return buf, ctx
}

func BenchmarkBlockDraw(b *testing.B) {
	buf, ctx := prepareBenchmarkEnv()
	w := Block{
		Title:          "Benchmark Block",
		TitleAlignment: AlignLeft,
		Borders:        BorderAll,
		BorderSymbols:  SymbolsRounded,
		PaddingTop:     1,
		PaddingBottom:  1,
		PaddingLeft:    2,
		PaddingRight:   2,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Draw(ctx, buf)
	}
}

func BenchmarkParagraphDraw(b *testing.B) {
	buf, ctx := prepareBenchmarkEnv()
	w := Paragraph{
		Text: "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Draw(ctx, buf)
	}
}

func BenchmarkTableDraw(b *testing.B) {
	buf, ctx := prepareBenchmarkEnv()
	w := Table{
		Header: &TableRow{
			Cells: []TableCell{{Text: "Col 1"}, {Text: "Col 2"}},
		},
		Rows: []TableRow{
			{Cells: []TableCell{{Text: "Val 1"}, {Text: "Val 2"}}},
			{Cells: []TableCell{{Text: "Val 3"}, {Text: "Val 4"}}},
		},
		Constraints: []TableConstraint{
			{Type: ConstraintPercentage, Value: 50},
			{Type: ConstraintFill},
		},
		DrawGrid: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Draw(ctx, buf)
	}
}

func BenchmarkListDraw(b *testing.B) {
	buf, ctx := prepareBenchmarkEnv()
	w := List{
		Items: []string{"Item 1", "Item 2", "Item 3", "Item 4", "Item 5"},
		State: &ListState{Selected: 2},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Draw(ctx, buf)
	}
}

func BenchmarkTextInputDraw(b *testing.B) {
	buf, ctx := prepareBenchmarkEnv()
	state := NewTextInputState()
	state.SetValue("Input text")
	w := TextInput{
		ID:          "widget_id",
		State:       state,
		Placeholder: "Enter text...",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Draw(ctx, buf)
	}
}

func BenchmarkCheckboxDraw(b *testing.B) {
	buf, ctx := prepareBenchmarkEnv()
	checked := true
	w := Checkbox{
		ID:      "widget_id",
		Checked: &checked,
		Label:   "Checkbox Label",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Draw(ctx, buf)
	}
}

func BenchmarkRadioDraw(b *testing.B) {
	buf, ctx := prepareBenchmarkEnv()
	selected := "value1"
	w := RadioButton{
		ID:       "widget_id",
		Selected: &selected,
		Value:    "value1",
		Label:    "Radio Label",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Draw(ctx, buf)
	}
}

func BenchmarkSliderDraw(b *testing.B) {
	buf, ctx := prepareBenchmarkEnv()
	state := NewSliderState(50)
	w := Slider{
		ID:    "widget_id",
		State: state,
		Min:   0,
		Max:   100,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Draw(ctx, buf)
	}
}

func BenchmarkProgressBarDraw(b *testing.B) {
	buf, ctx := prepareBenchmarkEnv()
	w := ProgressBar{
		Value: 65,
		Min:   0,
		Max:   100,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Draw(ctx, buf)
	}
}

func BenchmarkSparklineDraw(b *testing.B) {
	buf, ctx := prepareBenchmarkEnv()
	w := Sparkline{
		Data: []float64{10, 20, 15, 30, 45, 12, 18, 25, 35},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Draw(ctx, buf)
	}
}

func BenchmarkRichTextDraw(b *testing.B) {
	buf, ctx := prepareBenchmarkEnv()
	w := Paragraph{
		Text: "Hello **world** *italic* `code` text",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Draw(ctx, buf)
	}
}

func BenchmarkCanvasDraw(b *testing.B) {
	buf, ctx := prepareBenchmarkEnv()
	w := NewCanvas(160, 100)
	w.DrawLine(0, 0, 160, 100, cell.Style{})
	w.DrawCircle(80, 50, 25, cell.Style{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Draw(ctx, buf)
	}
}

func BenchmarkImageDrawHalfBlock(b *testing.B) {
	buf, ctx := prepareBenchmarkEnv()
	img := image.NewRGBA(rect(0, 0, 40, 20))
	w := Image{
		Img:            img,
		ForceHalfBlock: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Draw(ctx, buf)
	}
}

func rect(x, y, w, h int) image.Rectangle {
	return image.Rect(x, y, x+w, y+h)
}

func BenchmarkLogViewDraw(b *testing.B) {
	buf, ctx := prepareBenchmarkEnv()
	src := &testLog{}
	for i := 0; i < 100000; i++ {
		lv := LevelInfo
		if i%7 == 0 {
			lv = LevelWarn
		}
		src.add(`{"level":"info","msg":"request served","status":200,"path":"/api/v1/items"}`, lv)
	}
	w := &LogView{ID: "log", Source: src, State: NewLogViewState(), LineNumbers: true, Highlight: "served"}
	w.Draw(ctx, buf)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Draw(ctx, buf)
	}
}

func BenchmarkButtonDraw(b *testing.B) {
	buf, ctx := prepareBenchmarkEnv()
	pressed := 0
	onPress := func() { pressed++ }
	w := Button{ID: "save", Label: "Save", OnPress: onPress}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Draw(ctx, buf)
	}
}

// The widgets below register handlers or format text; drawn through a frame
// they are held at zero by TestInteractiveWidgetsDrawWithoutAllocating in
// benchmarks/. Here their drawing alone is held there by CI (#12).

func BenchmarkTreeViewDraw(b *testing.B) {
	buf, ctx := prepareBenchmarkEnv()
	tree := &TreeView{ID: "widget_id", State: NewTreeViewState(), ShowGuides: true, Roots: []TreeNode{
		{ID: "src", Label: "src", Icon: "📁", Expanded: true, Children: []TreeNode{{ID: "a", Label: "main.go"}, {ID: "b", Label: "util.go"}}},
		{ID: "doc", Label: "docs"},
	}}
	tree.Draw(ctx, buf)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tree.Draw(ctx, buf)
	}
}

func BenchmarkToastDraw(b *testing.B) {
	buf, ctx := prepareBenchmarkEnv()
	tm := NewToastManager(ToastTopRight)
	tm.Success("Saved", "3 files written")
	tm.Error("Failed", "network unreachable")
	tm.Draw(ctx, buf)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tm.Draw(ctx, buf)
	}
}

func BenchmarkDialogDraw(b *testing.B) {
	buf, ctx := prepareBenchmarkEnv()
	d := &Dialog{ID: "widget_id", Title: "Quit?", Message: "You have unsaved changes in three files.",
		SubMessage: "They will be lost.", Buttons: []DialogButton{{Text: "Save"}, {Text: "Discard"}}, State: &DialogState{}}
	d.Draw(ctx, buf)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.Draw(ctx, buf)
	}
}

func BenchmarkSelectOpenDraw(b *testing.B) {
	buf, ctx := prepareBenchmarkEnv()
	state := NewSelectState()
	state.Open = true
	s := &Select{ID: "widget_id", Options: []string{"small", "medium", "large"}, State: state}
	s.Draw(ctx, buf)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Draw(ctx, buf)
	}
}

func BenchmarkColorPickerDraw(b *testing.B) {
	buf, ctx := prepareBenchmarkEnv()
	cp := &ColorPicker{ID: "widget_id", State: NewColorPickerState(200, 120, 40)}
	cp.Draw(ctx, buf)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cp.Draw(ctx, buf)
	}
}

func BenchmarkCommandPaletteOpenDraw(b *testing.B) {
	buf, ctx := prepareBenchmarkEnv()
	state := NewCommandPaletteState()
	state.AllItems = []CommandItem{{Label: "Open file", Detail: "Ctrl+O"}, {Label: "Save", Detail: "Ctrl+S"}}
	state.Open()
	cp := &CommandPalette{ID: "widget_id", State: state}
	cp.Draw(ctx, buf)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cp.Draw(ctx, buf)
	}
}

func BenchmarkTextAreaDraw(b *testing.B) {
	buf, ctx := prepareBenchmarkEnv()
	state := NewTextAreaState()
	state.SetValue("a message that wraps over more than one row of the area,\nand a second line")
	ta := TextArea{ID: "widget_id", State: state, Focused: true}
	ta.Draw(ctx, buf)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ta.Draw(ctx, buf)
	}
}

func BenchmarkMarkdownBlocksDraw(b *testing.B) {
	buf, ctx := prepareBenchmarkEnv()
	md := NewMarkdown("### Steps\n1. one\n   - nested\n> a quote\n```go\nfunc main() {}\n```\n| a | b |\n|---|--:|\n| x | 1 |\n")
	md.Draw(ctx, buf)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		md.Draw(ctx, buf)
	}
}
