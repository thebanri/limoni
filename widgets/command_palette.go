package widgets

import (
	"strings"
	"unicode/utf8"

	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
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
func (cps *CommandPaletteState) HandleKey(ev backend.KeyEvent) bool {
	if !cps.IsOpen {
		return false
	}

	// Ctrl+P, açık paleti de aynı kısayolla kapatır.
	if ev.Type == backend.KeyRune && ev.Ch == 'p' && ev.Ctrl {
		cps.Close()
		return true
	}

	switch ev.Type {
	case backend.KeyEsc:
		cps.Close()
		return true

	case backend.KeyEnter:
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

	case backend.KeyArrowUp:
		if cps.Selected > 0 {
			cps.Selected--
			// Scroll up if needed
			if cps.Selected < cps.ScrollOffset {
				cps.ScrollOffset = cps.Selected
			}
		}
		return true

	case backend.KeyArrowDown:
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
	State       *CommandPaletteState
	Position    *CommandPalettePosition
	Style       cell.Style // Arka plan stili
	InputStyle  cell.Style // Arama kutusu stili
	ItemStyle   cell.Style // Normal öğe stili
	SelStyle    cell.Style // Seçili öğe stili
	DetailStyle cell.Style // Kısayol/detay stili
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
	// Başlık
	title := " ⌘ Komut Paleti "
	titleRunes := []rune(title)
	titleStart := startX + 2
	titleStyle := cell.Style{Fg: cell.NewColorRGB(100, 200, 255), Bg: bgStyle.Bg, Modifier: cell.ModifierBold}
	titleX := titleStart
	for _, r := range titleRunes {
		if titleX >= startX+paletW-1 {
			break
		}
		titleX += drawRune(buf, titleX, startY, r, titleStyle)
	}

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

	// Arama metni
	queryText := cp.State.Query.Value()
	if queryText == "" {
		// Placeholder
		placeholder := "Komut ara..."
		phRunes := []rune(placeholder)
		phStyle := cell.Style{Fg: cell.NewColorRGB(120, 120, 140), Bg: inputStyle.Bg}
		for i, r := range phRunes {
			x := startX + 1 + searchIconWidth + i
			if x >= startX+paletW-1 {
				break
			}
			drawRune(buf, x, inputY, r, phStyle)
		}
	} else {
		qRunes := []rune(queryText)
		for i, r := range qRunes {
			x := startX + 1 + searchIconWidth + i
			if x >= startX+paletW-1 {
				break
			}
			drawRune(buf, x, inputY, r, inputStyle)
		}
	}

	// İmleç
	cursorX := startX + 1 + searchIconWidth + cp.State.Query.Cursor
	if cursorX < startX+paletW-1 {
		if c := buf.Get(uint16(cursorX), uint16(inputY)); c != nil {
			c.Style = cell.Style{Fg: inputStyle.Bg, Bg: inputStyle.Fg} // Ters renkler
			if c.Content == ' ' || c.Content == 0 {
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

		// Etiket
		labelRunes := []rune(item.Label)
		queryLower := strings.ToLower(cp.State.Query.Value())
		labelLower := strings.ToLower(item.Label)
		matchPositions := computeMatchPositions(queryLower, labelLower)

		labelStart := startX + 3
		for ci, r := range labelRunes {
			x := labelStart + ci
			if x >= startX+paletW-1 {
				break
			}
			charStyle := rowStyle
			// Eşleşen harfleri vurgula
			if _, ok := matchPositions[ci]; ok {
				charStyle.Fg = cell.NewColorRGB(100, 200, 255)
				charStyle.Modifier = cell.ModifierBold
			}
			if c := buf.Get(uint16(x), uint16(y)); c != nil {
				c.Content = r
				c.Style = charStyle
			}
		}

		// Detay (sağa yasla)
		if item.Detail != "" {
			detailRunes := []rune(item.Detail)
			detailLen := utf8.RuneCountInString(item.Detail)
			detailStart := startX + paletW - 2 - detailLen
			if detailStart > labelStart+len(labelRunes)+1 {
				dStyle := detailStyle
				if isSelected {
					dStyle.Bg = selStyle.Bg
				}
				for di, r := range detailRunes {
					x := detailStart + di
					if x >= startX+paletW-1 || x < startX+1 {
						continue
					}
					if c := buf.Get(uint16(x), uint16(y)); c != nil {
						c.Content = r
						c.Style = dStyle
					}
				}
			}
		}
	}

	// Sonuç sayısı göstergesi (alt satır veya kenarlıkta)
	totalStr := []rune(strings.Replace(strings.Replace("  X/Y  ", "X", itoa(len(cp.State.Filtered)), 1), "Y", itoa(len(cp.State.AllItems)), 1))
	countStart := startX + paletW - 2 - len(totalStr)
	if countStart > startX+1 {
		countStyle := cell.Style{Fg: cell.NewColorRGB(80, 80, 100), Bg: bgStyle.Bg}
		for i, r := range totalStr {
			if c := buf.Get(uint16(countStart+i), uint16(bottomY)); c != nil {
				c.Content = r
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

// computeMatchPositions, fuzzy arama sonucu eşleşen karakter konumlarını döndürür.
func computeMatchPositions(queryLower, targetLower string) map[int]bool {
	positions := make(map[int]bool)
	qRunes := []rune(queryLower)
	tRunes := []rune(targetLower)

	qi := 0
	for ti := 0; ti < len(tRunes) && qi < len(qRunes); ti++ {
		if tRunes[ti] == qRunes[qi] {
			positions[ti] = true
			qi++
		}
	}
	return positions
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
