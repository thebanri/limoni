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
