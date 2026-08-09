// forms demonstrates the Select and Slider widgets.
package main

import (
	"fmt"
	"os"

	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/widgets"
)

type text struct {
	value string
	style cell.Style
}

func (t text) Draw(ctx cell.Context, buf *buffer.Buffer) {
	buf.SetString(ctx.Area.X, ctx.Area.Y, t.value, ctx.Style.Merge(t.style))
}
func (t text) SizeHint(maxArea cell.Rect) (uint16, uint16) { return maxArea.Width, 1 }

func main() {
	b := backend.NewBackend(os.Stdin, os.Stdout)
	if err := b.Setup(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	defer b.Close()
	t, err := terminal.New(b)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	b.StartEventLoop()

	selectState := widgets.NewSelectState()
	sliderState := widgets.NewSliderState(50)
	textAreaState := widgets.NewTextAreaState()
	textAreaState.SetValue("Form validation için not yazın...")
	textAreaRule := widgets.Validator{Required: true, MinLength: 5, Message: "Not en az 5 karakter olmalı."}
	draw := func() {
		t.Draw(func(f *terminal.Frame) {
			f.SetTheme(widgets.DarkTheme())
			area := f.Buffer.Area
			f.RenderWidget(widgets.Block{
				Title: " FORM WIDGETS ", Borders: widgets.BorderAll,
				Margin:        widgets.Insets{Top: 1, Right: 1, Bottom: 0, Left: 1},
				Padding:       widgets.UniformInsets(1),
				BorderSymbols: widgets.SymbolsRounded,
				BorderStyle:   cell.Style{Fg: cell.NewColorRGB(100, 200, 255)},
				Child:         text{value: "Select ve Slider örneği — Tab ile odak değiştir, Esc/q ile çık", style: cell.Style{Fg: cell.NewColorRGB(220, 220, 220)}},
			}, cell.NewRect(1, 1, area.Width-2, 5))
			f.RenderWidget(text{value: "Environment", style: cell.Style{Fg: cell.NewColorRGB(180, 180, 190)}}, cell.NewRect(3, 6, 18, 1))
			f.RenderWidget(widgets.Select{ID: "environment", Options: []string{"Development", "Staging", "Production"}, State: selectState, Style: cell.Style{Bg: cell.NewColorRGB(35, 35, 50)}, FocusedStyle: cell.Style{Fg: cell.NewColorRGB(100, 220, 255)}}, cell.NewRect(22, 6, 30, 4))
			f.RenderWidget(text{value: fmt.Sprintf("Load: %d%%", sliderState.Value), style: cell.Style{Fg: cell.NewColorRGB(180, 180, 190)}}, cell.NewRect(3, 12, 18, 1))
			f.RenderWidget(widgets.Slider{ID: "load", State: sliderState, Min: 0, Max: 100, TrackStyle: cell.Style{Fg: cell.NewColorRGB(70, 70, 90)}, FilledStyle: cell.Style{Fg: cell.NewColorRGB(80, 200, 140)}, ThumbStyle: cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Modifier: cell.ModifierBold}}, cell.NewRect(22, 12, 40, 1))
			error := textAreaRule.Validate(textAreaState.Value())
			f.RenderWidget(widgets.Block{Title: " NOTLAR / VALIDATION ", Borders: widgets.BorderAll, BorderSymbols: widgets.SymbolsRounded, BorderStyle: cell.Style{Fg: cell.NewColorRGB(255, 180, 70)}, Child: widgets.TextArea{ID: "notes", State: textAreaState, Style: cell.Style{Fg: cell.NewColorRGB(220, 220, 220), Bg: cell.NewColorRGB(25, 28, 36)}, FocusedStyle: cell.Style{Fg: cell.NewColorRGB(100, 200, 255)}}}, cell.NewRect(3, 15, 60, 5))
			if error != "" {
				f.RenderWidget(text{value: error, style: cell.Style{Fg: cell.NewColorRGB(255, 90, 90)}}, cell.NewRect(3, 21, 60, 1))
			}
		})
	}
	draw()
	for ev := range b.Events() {
		switch ev.Type {
		case backend.EventKey:
			if ev.Key.Type == backend.KeyEsc || (ev.Key.Type == backend.KeyRune && ev.Key.Ch == 'q') {
				return
			}
			if ev.Key.Type == backend.KeyTab {
				t.FocusManager().Next()
			}
			switch t.FocusManager().Focused() {
			case "environment":
				selectState.HandleKey(ev.Key, 3)
			case "load":
				sliderState.HandleKey(ev.Key, 0, 100)
			case "notes":
				textAreaState.HandleKey(ev.Key)
			}
			draw()
		case backend.EventMouse:
			t.RouteMouseEvent(ev.Mouse)
			draw()
		}
	}
}
