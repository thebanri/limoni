package globe

import (
	"strings"
	"time"

	"github.com/thebanri/limoni"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/widgets"
)

var (
	dimStyle   = limoni.Fg(limoni.RGB(140, 150, 170))
	titleStyle = limoni.Fg(limoni.RGB(120, 190, 255))
	hitStyle   = limoni.Fg(limoni.RGB(240, 240, 240))
	selStyle   = limoni.Fg(limoni.RGB(20, 24, 32)).WithBg(limoni.RGB(120, 190, 255))
	pinStyle   = limoni.Fg(limoni.RGB(255, 150, 150))
)

// Draw lays the application out: the globe with whatever room is left, the
// search and the pins beside it, and a line of state underneath.
func (v *Viewer) Draw(f *limoni.Frame, area limoni.Rect) {
	rows := limoni.SplitVertical(area, limoni.Fill(), limoni.Fixed(1))
	body, footer := rows[0], rows[1]

	globePane := body
	if body.Width >= 72 {
		cols := limoni.SplitHorizontal(body, limoni.Fill(), limoni.Fixed(sidebarWidth))
		globePane, _ = cols[0], cols[1]
		v.drawSidebar(f, cols[1])
	} else if v.searching {
		// Too narrow for both: the search takes the screen while it is open.
		v.drawSidebar(f, body)
		v.drawFooter(f, footer)
		return
	}

	v.globeArea = globePane
	f.RenderWidget(v.globe, globePane)
	if v.clickGlobe != nil {
		f.RegisterClickHandler(globePane, v.clickGlobe)
	}
	v.drawFooter(f, footer)
	if v.help {
		v.drawHelp(f, area)
	}
}

func (v *Viewer) drawSidebar(f *limoni.Frame, area limoni.Rect) {
	if area.Width < 12 || area.Height < 6 {
		return
	}
	pinHeight := uint16(len(v.pinRows)) + 2
	if pinHeight > area.Height/3 {
		pinHeight = area.Height / 3
	}
	if pinHeight < 3 {
		pinHeight = 3
	}
	parts := limoni.SplitVertical(area, limoni.Fixed(3), limoni.Fill(), limoni.Fixed(pinHeight))

	// The search box. It keeps the keyboard while searching, and says so.
	f.RenderWidget(widgets.Block{
		Title: " Find a place ", Borders: widgets.BorderAll,
		BorderSymbols: widgets.SymbolsRounded, BorderStyle: dimStyle, TitleStyle: titleStyle,
	}, parts[0])
	inner := inset(parts[0])
	f.RenderWidget(&widgets.TextInput{
		ID:          searchID,
		State:       v.query,
		Placeholder: "country or city",
		Focused:     v.searching,
		Style:       hitStyle,
	}, inner)

	v.resultsArea = parts[1]
	f.RenderWidget(&widgets.List{
		ID:              "results",
		Label:           "Results",
		Items:           v.rows,
		State:           v.results,
		Style:           hitStyle,
		SelectedStyle:   selStyle,
		HighlightSymbol: "› ",
		Scrollbar:       true,
	}, parts[1])
	// Registered after the list, so this handler sees the click first: the
	// viewer decides what a click on a row means, rather than reading back a
	// selection that may not have changed.
	if v.clickResult != nil {
		f.RegisterClickHandler(parts[1], v.clickResult)
	}

	f.RenderWidget(widgets.Block{
		Title: " Pins ", Borders: widgets.BorderAll,
		BorderSymbols: widgets.SymbolsRounded, BorderStyle: dimStyle, TitleStyle: titleStyle,
	}, parts[2])
	if parts[2].Height > 2 {
		v.pinsArea = inset(parts[2])
		f.RenderWidget(&widgets.List{
			ID:            "pins",
			Label:         "Pins",
			Items:         v.pinRows,
			State:         v.pins,
			Style:         pinStyle,
			SelectedStyle: selStyle,
		}, v.pinsArea)
		if v.clickPin != nil {
			f.RegisterClickHandler(v.pinsArea, v.clickPin)
		}
	}
}

// inset returns the inside of a bordered block.
func inset(r limoni.Rect) limoni.Rect {
	if r.Width < 3 || r.Height < 3 {
		return r
	}
	return limoni.NewRect(r.X+1, r.Y+1, r.Width-2, r.Height-2)
}

func (v *Viewer) drawFooter(f *limoni.Frame, area limoni.Rect) {
	if area.Height == 0 {
		return
	}
	lat, lon := v.globe.Centre()

	var b strings.Builder
	b.Grow(96)
	b.WriteByte(' ')
	b.WriteString(formatCoord(lat, lon))
	b.WriteString(" · zoom ")
	b.WriteString(formatZoom(v.globe.Zoom))
	switch {
	case v.fly != nil:
		b.WriteString(" · flying to ")
		b.WriteString(v.fly.name)
	case v.spinning:
		b.WriteString(" · turning")
	default:
		b.WriteString(" · held")
	}
	if v.status != "" {
		b.WriteString(" · ")
		b.WriteString(v.status)
	}
	v.footer = b.String()

	f.RenderWidget(&widgets.Paragraph{ID: "status", Text: v.footer, Style: dimStyle}, area)

	hints := " / find  ←→↑↓ turn  +− zoom  m pin  c clear  a ascii  ? keys  q quit"
	if w := uint16(len(hints)); area.Width > w+uint16(len(v.footer))+2 {
		f.RenderWidget(&widgets.Paragraph{
			ID: "keys", Text: hints, Style: dimStyle,
		}, limoni.NewRect(area.X+area.Width-w, area.Y, w, 1))
	}
}

func (v *Viewer) drawHelp(f *limoni.Frame, area limoni.Rect) {
	lines := []string{
		"",
		"  /          search for a country or a city",
		"  ⏎          fly there and drop a pin",
		"  ←→ ↑↓      turn the globe",
		"  + −        zoom in and out",
		"  space      hold the rotation, or let it go",
		"  click      pin the point under the pointer",
		"  m          pin the centre of the view",
		"  c          clear the pins",
		"  a          ASCII shading instead of colour",
		"  g          parallels and meridians",
		"  r          back to the whole globe",
		"  ? q        this help, and quit",
		"",
		"  The land is Natural Earth 1:110m, in the public domain.",
		"",
	}
	w := uint16(62)
	if w > area.Width-4 {
		w = area.Width - 4
	}
	h := uint16(len(lines)) + 2
	if h > area.Height-2 {
		h = area.Height - 2
	}
	box := limoni.NewRect(area.X+(area.Width-w)/2, area.Y+(area.Height-h)/2, w, h)
	f.RenderWidget(widgets.Block{
		Title: " Keys ", Borders: widgets.BorderAll, BorderSymbols: widgets.SymbolsRounded,
		BorderStyle: titleStyle, TitleStyle: titleStyle,
		Style: limoni.Bg(limoni.RGB(16, 20, 28)),
	}, box)
	in := inset(box)
	for i, line := range lines {
		if uint16(i) >= in.Height {
			break
		}
		f.Buffer.SetStringWithin(in.X, in.Y+uint16(i), line, dimStyle, in.Width)
	}
}

// clickAt pins whatever is under the pointer. It is stored once in the
// viewer rather than built per frame, because a closure made during Draw is
// a heap allocation on every frame.
func (v *Viewer) clickAt(ev driver.MouseEvent) {
	area := v.globeArea
	if ev.X < area.X || ev.Y < area.Y || ev.X >= area.X+area.Width || ev.Y >= area.Y+area.Height {
		return
	}
	lat, lon, hit := v.globe.At(area, ev.X-area.X, ev.Y-area.Y, false)
	if !hit {
		return
	}
	now := time.Now()
	name := formatCoord(lat, lon)
	if p, ok := v.nearest(lat, lon); ok {
		name = p.Display()
		lat, lon = p.Lat, p.Lon
	}
	v.Pin(name, lat, lon, now)
	v.status = "pinned " + name
	v.wantFocus = globeID
	v.Wake()
}
