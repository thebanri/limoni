package widgets

import (
	"strconv"
	"unicode"
	"unicode/utf8"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/graphics"
)

// CommandItem, Komut Paleti'nde gösterilecek bir komutu temsil eder.
type CommandItem struct {
	// Label, komutun gösterileceği ana metindir.
	Label string
	// Detail, kısayol tuşu veya ek açıklama metnidir (ör: "Ctrl+P").
	Detail string
	// Category, komutun ait olduğu kategoridir (ör: "Navigasyon").
	Category string
	// Handler, komut seçildiğinde çalıştırılacak callback fonksiyonudur.
	Handler func()
}

// CommandPaletteState, Komut Paleti'nin veri durumunu yönetir.
type CommandPaletteState struct {
	// IsOpen, paletin açık olup olmadığını belirtir.
	IsOpen bool
	// Query, arama kutusunun metin durumudur.
	Query TextInputState
	// Selected, mevcut seçili sonuç indeksidir.
	Selected int
	// AllItems, tüm kayıtlı komutlardır.
	AllItems []CommandItem
	// Filtered, bulanık arama sonucu filtrelenmiş komutlardır.
	Filtered []CommandItem
	// MaxVisible, aynı anda gösterilecek maksimum sonuç sayısıdır.
	MaxVisible int
	// ScrollOffset, uzun listelerde kaydırma ofseti.
	ScrollOffset int
}

// NewCommandPaletteState, yeni bir CommandPaletteState oluşturur.
func NewCommandPaletteState() *CommandPaletteState {
	return &CommandPaletteState{
		Query:      *NewTextInputState(),
		Selected:   0,
		MaxVisible: 10,
	}
}

// Open, paleti açar ve arama kutusunu temizler.
func (cps *CommandPaletteState) Open() {
	cps.IsOpen = true
	cps.Query.Text = cps.Query.Text[:0]
	cps.Query.Cursor = 0
	cps.Selected = 0
	cps.ScrollOffset = 0
	cps.Filtered = FuzzyFilter("", cps.AllItems)
}

// Close, paleti kapatır.
func (cps *CommandPaletteState) Close() {
	cps.IsOpen = false
}

// Toggle, paleti açar veya kapatır.
func (cps *CommandPaletteState) Toggle() {
	if cps.IsOpen {
		cps.Close()
	} else {
		cps.Open()
	}
}

// HandleKey, Command Palette açıkken gelen tuş olayını işler.
// true döner ise olay tüketilmiştir, dış event loop'a yayılmamalıdır.
func (cps *CommandPaletteState) HandleKey(ev driver.KeyEvent) bool {
	if !cps.IsOpen {
		return false
	}

	// Ctrl+P, açık paleti de aynı kısayolla kapatır.
	if ev.Type == driver.KeyRune && ev.Ch == 'p' && ev.Ctrl {
		cps.Close()
		return true
	}

	switch ev.Type {
	case driver.KeyEsc:
		cps.Close()
		return true

	case driver.KeyEnter:
		if len(cps.Filtered) > 0 && cps.Selected >= 0 && cps.Selected < len(cps.Filtered) {
			handler := cps.Filtered[cps.Selected].Handler
			cps.Close()
			if handler != nil {
				handler()
			}
		} else {
			// Sonuç yoksa veya seçim geçersizse yine de paleti kapat
			cps.Close()
		}
		return true

	case driver.KeyArrowUp:
		if cps.Selected > 0 {
			cps.Selected--
			// Scroll up if needed
			if cps.Selected < cps.ScrollOffset {
				cps.ScrollOffset = cps.Selected
			}
		}
		return true

	case driver.KeyArrowDown:
		if cps.Selected < len(cps.Filtered)-1 {
			cps.Selected++
			// Scroll down if needed
			if cps.Selected >= cps.ScrollOffset+cps.MaxVisible {
				cps.ScrollOffset = cps.Selected - cps.MaxVisible + 1
			}
		}
		return true

	default:
		// Metin girişine yönlendir
		changed := cps.Query.HandleKey(ev)
		if changed {
			query := cps.Query.Value()
			cps.Filtered = FuzzyFilter(query, cps.AllItems)
			cps.Selected = 0
			cps.ScrollOffset = 0
		}
		return true
	}
}

// CommandPalettePosition controls an overlay's distance from terminal edges.
// Use NewCommandPalettePosition so unspecified edges are initialized to -1.
type CommandPalettePosition struct {
	Top    int
	Right  int
	Bottom int
	Left   int
}

func NewCommandPalettePosition() *CommandPalettePosition {
	return &CommandPalettePosition{Top: -1, Right: -1, Bottom: -1, Left: -1}
}

// CommandPalette, Komut Paleti overlay widget'ıdır.
type CommandPalette struct {
	ID          string
	State       *CommandPaletteState
	Position    *CommandPalettePosition
	Style       cell.Style // Arka plan stili
	InputStyle  cell.Style // Arama kutusu stili
	ItemStyle   cell.Style // Normal öğe stili
	SelStyle    cell.Style // Seçili öğe stili
	DetailStyle cell.Style // Kısayol/detay stili

	// Title is drawn in the top border. Empty means " ⌘ Commands ".
	Title string
	// Placeholder is shown while the query is empty. Empty means
	// "Search commands...".
	Placeholder string
}

func (cp CommandPalette) panelArea(area cell.Rect) cell.Rect {
	if cp.State == nil || !cp.State.IsOpen || area.Width < 20 || area.Height < 6 {
		return cell.Rect{}
	}
	paletteWidth := int(area.Width) * 60 / 100
	if paletteWidth < 30 {
		paletteWidth = 30
	}
	if paletteWidth > int(area.Width)-4 {
		paletteWidth = int(area.Width) - 4
	}
	visibleCount := len(cp.State.Filtered)
	if visibleCount > cp.State.MaxVisible {
		visibleCount = cp.State.MaxVisible
	}
	paletteHeight := 4 + visibleCount
	if paletteHeight > int(area.Height)-4 {
		paletteHeight = int(area.Height) - 4
	}

	// Position yoksa eski davranış: yatayda ortalı, üstten 2 satır.
	x := int(area.X) + (int(area.Width)-paletteWidth)/2
	y := int(area.Y) + 2
	if cp.Position != nil {
		p := cp.Position
		if p.Left > 0 {
			x = int(area.X) + p.Left
		}
		if p.Right > 0 {
			x = int(area.X) + int(area.Width) - paletteWidth - p.Right
		}
		if p.Top > 0 {
			y = int(area.Y) + p.Top
		}
		if p.Bottom > 0 {
			y = int(area.Y) + int(area.Height) - paletteHeight - p.Bottom
		}
	}
	if x < int(area.X) {
		x = int(area.X)
	}
	if y < int(area.Y) {
		y = int(area.Y)
	}
	maxX := int(area.X) + int(area.Width) - paletteWidth
	maxY := int(area.Y) + int(area.Height) - paletteHeight
	if x > maxX {
		x = maxX
	}
	if y > maxY {
		y = maxY
	}
	return cell.NewRect(uint16(x), uint16(y), uint16(paletteWidth), uint16(paletteHeight))
}

// DebugArea, komut paletinin gerçek ekrandaki sınırını döndürür.
// Çizim sırasında palette tam terminal alanını alır; panel ise bu alanın
// içinde ortalandığı için Layout Inspector'a gerçek panel sınırını bildirir.
func (cp CommandPalette) DebugArea(area cell.Rect) cell.Rect {
	return cp.panelArea(area)
}

// Draw, Komut Paleti'ni ekranın ortasına overlay olarak çizer.
func (cp CommandPalette) Draw(ctx cell.Context, buf *buffer.Buffer) {
	if cp.State == nil || !cp.State.IsOpen {
		return
	}

	area := ctx.Area
	if area.Width < 20 || area.Height < 6 {
		return
	}
	if cp.ID != "" && ctx.RegisterFocus != nil {
		ctx.RegisterFocus(cp.ID)
	}
	if cp.ID != "" && ctx.RegisterClick != nil {
		ctx.RegisterClick(ctx.Area, func() {
			if ctx.SetFocus != nil {
				ctx.SetFocus(cp.ID)
			}
		})
	}

	panel := cp.panelArea(area)
	paletW := int(panel.Width)
	paletH := int(panel.Height)
	visibleCount := paletH - 4
	if visibleCount < 0 {
		visibleCount = 0
	}
	startX := int(panel.X)
	startY := int(panel.Y)

	// Gölge efekti (sağ ve alt kenarda)
	shadowStyle := cell.Style{Bg: cell.NewColorRGB(15, 15, 15), Fg: cell.NewColorRGB(15, 15, 15)}
	for dy := 1; dy <= paletH; dy++ {
		for dx := 0; dx < 2; dx++ {
			x := startX + paletW + dx
			y := startY + dy
			if c := buf.Get(uint16(x), uint16(y)); c != nil {
				c.Content = ' '
				c.Style = shadowStyle
			}
		}
	}
	for dx := 2; dx < paletW+2; dx++ {
		y := startY + paletH
		x := startX + dx
		if c := buf.Get(uint16(x), uint16(y)); c != nil {
			c.Content = ' '
			c.Style = shadowStyle
		}
	}

	// Arka plan
	bgStyle := cp.Style
	if bgStyle.Bg == 0 {
		bgStyle.Bg = cell.NewColorRGB(30, 30, 40)
		bgStyle.Fg = cell.NewColorRGB(220, 220, 230)
	}

	// Arka plandaki yerel grafiklerin (Kitty/Sixel/iTerm2) sızmasını engellemek için
	// palet ve gölge alanına z = -2 seviyesinde solid arka plan resmi kaydet:
	if ctx.RegisterImage != nil {
		proto := graphics.DetectProtocol()
		if proto != graphics.ProtocolHalfBlock {
			backdropW := uint16(paletW + 2)
			backdropH := uint16(paletH + 1)
			if uint16(startX)+backdropW > area.X+area.Width {
				backdropW = area.X + area.Width - uint16(startX)
			}
			if uint16(startY)+backdropH > area.Y+area.Height {
				backdropH = area.Y + area.Height - uint16(startY)
			}
			shadowBackdrop := cell.NewRect(
				uint16(startX),
				uint16(startY),
				backdropW,
				backdropH,
			)
			solidImg := getSolidImage(bgStyle.Bg)
			ctx.RegisterImage(shadowBackdrop, solidImg, -2, false)
		}
	}

	for dy := 0; dy < paletH; dy++ {
		for dx := 0; dx < paletW; dx++ {
			if c := buf.Get(uint16(startX+dx), uint16(startY+dy)); c != nil {
				c.Content = ' '
				c.Style = bgStyle
			}
		}
	}

	// Üst/Alt kenarlık (yuvarlak)
	borderStyle := cell.Style{Fg: cell.NewColorRGB(100, 200, 255), Bg: bgStyle.Bg}
	// Üst çizgi
	if c := buf.Get(uint16(startX), uint16(startY)); c != nil {
		c.Content = '╭'
		c.Style = borderStyle
	}
	if c := buf.Get(uint16(startX+paletW-1), uint16(startY)); c != nil {
		c.Content = '╮'
		c.Style = borderStyle
	}
	for dx := 1; dx < paletW-1; dx++ {
		if c := buf.Get(uint16(startX+dx), uint16(startY)); c != nil {
			c.Content = '─'
			c.Style = borderStyle
		}
	}
	// Title, in the top border.
	title := cp.Title
	if title == "" {
		title = " ⌘ Commands "
	}
	titleStyle := cell.Style{Fg: cell.NewColorRGB(100, 200, 255), Bg: bgStyle.Bg, Modifier: cell.ModifierBold}
	setEllipsized(buf, uint16(startX+2), uint16(startY), title, titleStyle, paletW-3, "…")

	// Alt çizgi
	bottomY := startY + paletH - 1
	if c := buf.Get(uint16(startX), uint16(bottomY)); c != nil {
		c.Content = '╰'
		c.Style = borderStyle
	}
	if c := buf.Get(uint16(startX+paletW-1), uint16(bottomY)); c != nil {
		c.Content = '╯'
		c.Style = borderStyle
	}
	for dx := 1; dx < paletW-1; dx++ {
		if c := buf.Get(uint16(startX+dx), uint16(bottomY)); c != nil {
			c.Content = '─'
			c.Style = borderStyle
		}
	}

	// Sol/Sağ kenarlar
	for dy := 1; dy < paletH-1; dy++ {
		if c := buf.Get(uint16(startX), uint16(startY+dy)); c != nil {
			c.Content = '│'
			c.Style = borderStyle
		}
		if c := buf.Get(uint16(startX+paletW-1), uint16(startY+dy)); c != nil {
			c.Content = '│'
			c.Style = borderStyle
		}
	}

	// Arama kutusu satırı (y = startY + 1)
	inputY := startY + 1
	inputStyle := cp.InputStyle
	if inputStyle.Fg == 0 {
		inputStyle.Fg = cell.NewColorRGB(255, 255, 255)
		inputStyle.Bg = cell.NewColorRGB(50, 50, 65)
	}

	// İkon: geniş karakterin devam hücresini de işaretle; aksi halde eski frame
	// içeriği ikinci hücrede kalıp paletin kenarında kayma/artefakt oluşturabilir.
	searchIconWidth := drawRune(buf, startX+1, inputY, '🔍', inputStyle)

	// Arama kutusu arka planı
	for dx := 1 + searchIconWidth; dx < paletW-1; dx++ {
		if c := buf.Get(uint16(startX+dx), uint16(inputY)); c != nil {
			c.Content = ' '
			c.Style = inputStyle
		}
	}

	// Query text, or the placeholder. Measured in columns and cut at cluster
	// boundaries, like TextInput.
	textX := startX + 1 + searchIconWidth
	textW := paletW - 2 - searchIconWidth
	if cp.State.Query.Value() == "" {
		placeholder := cp.Placeholder
		if placeholder == "" {
			placeholder = "Search commands..."
		}
		phStyle := cell.Style{Fg: cell.NewColorRGB(120, 120, 140), Bg: inputStyle.Bg}
		setEllipsized(buf, uint16(textX), uint16(inputY), placeholder, phStyle, textW, "…")
	} else {
		buf.SetStringWithin(uint16(textX), uint16(inputY), cp.State.Query.Value(), inputStyle, uint16(max(textW, 0)))
	}

	// Cursor.
	cursorX := textX + cp.State.Query.cursorColumn()
	if cursorX < startX+paletW-1 {
		if c := buf.Get(uint16(cursorX), uint16(inputY)); c != nil {
			c.Style = cell.Style{Fg: inputStyle.Bg, Bg: inputStyle.Fg} // inverted
			if c.Content == 0 {
				c.Content = ' '
			}
		}
	}

	// Ayırıcı çizgi (y = startY + 2)
	sepY := startY + 2
	sepStyle := cell.Style{Fg: cell.NewColorRGB(60, 60, 80), Bg: bgStyle.Bg}
	if c := buf.Get(uint16(startX), uint16(sepY)); c != nil {
		c.Content = '├'
		c.Style = sepStyle
	}
	if c := buf.Get(uint16(startX+paletW-1), uint16(sepY)); c != nil {
		c.Content = '┤'
		c.Style = sepStyle
	}
	for dx := 1; dx < paletW-1; dx++ {
		if c := buf.Get(uint16(startX+dx), uint16(sepY)); c != nil {
			c.Content = '─'
			c.Style = sepStyle
		}
	}

	// Sonuç listesi (y = startY + 3 ... )
	itemStyle := cp.ItemStyle
	if itemStyle.Fg == 0 {
		itemStyle.Fg = cell.NewColorRGB(200, 200, 210)
		itemStyle.Bg = bgStyle.Bg
	}
	selStyle := cp.SelStyle
	if selStyle.Fg == 0 {
		selStyle.Fg = cell.NewColorRGB(255, 255, 255)
		selStyle.Bg = cell.NewColorRGB(50, 100, 200)
	}
	detailStyle := cp.DetailStyle
	if detailStyle.Fg == 0 {
		detailStyle.Fg = cell.NewColorRGB(120, 120, 150)
		detailStyle.Bg = bgStyle.Bg
	}

	for i := 0; i < visibleCount; i++ {
		idx := i + cp.State.ScrollOffset
		if idx >= len(cp.State.Filtered) {
			break
		}

		item := cp.State.Filtered[idx]
		y := startY + 3 + i

		isSelected := idx == cp.State.Selected
		rowStyle := itemStyle
		if isSelected {
			rowStyle = selStyle
		}

		// Arka plan
		for dx := 1; dx < paletW-1; dx++ {
			if c := buf.Get(uint16(startX+dx), uint16(y)); c != nil {
				c.Content = ' '
				c.Style = rowStyle
			}
		}

		// Seçim işaretçisi
		if isSelected {
			if c := buf.Get(uint16(startX+1), uint16(y)); c != nil {
				c.Content = '▸'
				c.Style = cell.Style{Fg: cell.NewColorRGB(100, 200, 255), Bg: rowStyle.Bg}
			}
		}

		// Label, with the characters the query matched highlighted.
		labelStart := startX + 3
		labelLimit := startX + paletW - 1
		matchStyle := rowStyle
		matchStyle.Fg = cell.NewColorRGB(100, 200, 255)
		matchStyle.Modifier = cell.ModifierBold
		labelEnd := drawHighlighted(buf, labelStart, y, labelLimit, item.Label, cp.State.Query.Value(), rowStyle, matchStyle)

		// Detail, right-aligned, if it fits beside the label.
		if item.Detail != "" {
			detailStart := startX + paletW - 2 - cell.StringWidth(item.Detail)
			if detailStart > labelEnd+1 {
				dStyle := detailStyle
				if isSelected {
					dStyle.Bg = selStyle.Bg
				}
				buf.SetStringWithin(uint16(detailStart), uint16(y), item.Detail, dStyle, uint16(labelLimit-detailStart))
			}
		}

		// Fare tıklama ve üzerine gelme (hover) olaylarını kaydet
		rowArea := cell.NewRect(uint16(startX+1), uint16(y), uint16(paletW-2), 1)
		itemIdx := idx
		itemHandler := item.Handler
		if ctx.RegisterClick != nil {
			ctx.RegisterClick(rowArea, func() {
				if cp.State != nil {
					cp.State.Selected = itemIdx
					cp.State.Close()
				}
				if itemHandler != nil {
					itemHandler()
				}
			})
		}
		if ctx.RegisterMouse != nil {
			// A copy, so that capturing it does not move visibleCount to the
			// heap on frames that register no handlers.
			visibleCount := visibleCount
			ctx.RegisterMouse(rowArea, func(ev driver.MouseEvent) {
				if cp.State == nil {
					return
				}
				switch ev.Button {
				case driver.MouseLeft:
					if cp.State != nil {
						cp.State.Selected = itemIdx
						cp.State.Close()
					}
					if itemHandler != nil {
						itemHandler()
					}
				case driver.MouseNone:
					cp.State.Selected = itemIdx
				case driver.MouseScrollUp:
					if cp.State.Selected > 0 {
						cp.State.Selected--
						if cp.State.Selected < cp.State.ScrollOffset {
							cp.State.ScrollOffset = cp.State.Selected
						}
					}
				case driver.MouseScrollDown:
					if cp.State.Selected < len(cp.State.Filtered)-1 {
						cp.State.Selected++
						if cp.State.Selected >= cp.State.ScrollOffset+visibleCount {
							cp.State.ScrollOffset = cp.State.Selected - visibleCount + 1
						}
					}
				}
			})
		}
	}

	// Result count, "  filtered/total  ", in the bottom border. Built in a
	// stack buffer: formatting it as a string allocated on every frame.
	var countBuf [48]byte
	count := append(countBuf[:0], "  "...)
	count = strconv.AppendInt(count, int64(len(cp.State.Filtered)), 10)
	count = append(count, '/')
	count = strconv.AppendInt(count, int64(len(cp.State.AllItems)), 10)
	count = append(count, "  "...)
	countStart := startX + paletW - 2 - len(count)
	if countStart > startX+1 {
		countStyle := cell.Style{Fg: cell.NewColorRGB(80, 80, 100), Bg: bgStyle.Bg}
		for i, ch := range count {
			if c := buf.Get(uint16(countStart+i), uint16(bottomY)); c != nil {
				c.Content = rune(ch)
				c.Style = countStyle
			}
		}
	}
}

func (cp CommandPalette) SizeHint(maxArea cell.Rect) (width, height uint16) { return 0, 0 }

// drawRune writes a rune and marks the continuation cell for wide terminal runes.
// It returns the number of terminal columns consumed by the rune.
func drawRune(buf *buffer.Buffer, x, y int, r rune, style cell.Style) int {
	width := cell.RuneWidth(r)
	if width <= 0 {
		return 0
	}
	if c := buf.Get(uint16(x), uint16(y)); c != nil {
		c.Content = r
		c.Style = style
	}
	if width == 2 {
		if c := buf.Get(uint16(x+1), uint16(y)); c != nil {
			c.Content = cell.RuneContinuation
			c.Style = style
		}
	}
	return width
}

// drawHighlighted draws label from x, stopping before limit, and styles the
// clusters that the query matches as a case-insensitive subsequence — the
// fuzzy match the palette filters by. It returns the column after the label.
// It does not allocate; the previous version built a map and lowered both
// strings for every row on every frame.
func drawHighlighted(buf *buffer.Buffer, x, y, limit int, label, query string, style, match cell.Style) int {
	for rest := label; rest != ""; {
		cluster, w, next := cell.NextCluster(rest)
		rest = next
		if w == 0 {
			continue
		}
		if x+w > limit {
			break
		}
		st := style
		if query != "" {
			q, qsize := utf8.DecodeRuneInString(query)
			c, _ := utf8.DecodeRuneInString(cluster)
			if unicode.ToLower(c) == unicode.ToLower(q) {
				st = match
				query = query[qsize:]
			}
		}
		buf.SetStringWithin(uint16(x), uint16(y), cluster, st, uint16(w))
		x += w
	}
	return x
}

// itoa, basit int -> string dönüşümü.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	negative := false
	if n < 0 {
		negative = true
		n = -n
	}
	digits := make([]byte, 0, 10)
	for n > 0 {
		digits = append(digits, byte('0'+n%10))
		n /= 10
	}
	// Reverse
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}
	s := string(digits)
	if negative {
		s = "-" + s
	}
	return s
}
