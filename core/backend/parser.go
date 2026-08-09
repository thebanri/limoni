package backend

import (
	"unicode/utf8"
)

// ParseEvent gelen byte akışından tek bir TUI olayını ayrıştırır.
// Çıktı olarak ayrıştırılan olay yapısını ve bu olay için tüketilen (consumed) byte sayısını döner.
// Eğer tamponda eksik bir ANSI dizisi varsa tüketilen byte sayısı 0 döner; bu durumda daha fazla veri beklenmelidir.
func ParseEvent(buf []byte) (Event, int) {
	if len(buf) == 0 {
		return Event{}, 0
	}

	// 1. Bir ANSI kaçış dizisi değilse, standart UTF-8 karakter vuruşudur
	if buf[0] != '\x1b' {
		r, size := utf8.DecodeRune(buf)
		if r == utf8.RuneError {
			// Yarım kalan UTF-8 karakteri ise veri beklemeye devam et
			if !utf8.FullRune(buf) {
				return Event{}, 0
			}
			// Hatalı UTF-8 verisi, 1 byte tüketip geç
			return Event{}, 1
		}

		ev := Event{Type: EventKey}
		switch r {
		case '\n', '\r':
			ev.Key.Type = KeyEnter
		case 127, '\b':
			ev.Key.Type = KeyBackspace
		case '\t':
			ev.Key.Type = KeyTab
		case ' ':
			ev.Key.Type = KeySpace
		default:
			// Ctrl karakterlerinin algılanması: ASCII 1-26 aralığı Ctrl-A ile Ctrl-Z'ye denk gelir.
			if r >= 1 && r <= 26 {
				ev.Key.Type = KeyRune
				ev.Key.Ch = rune('a' + r - 1)
				ev.Key.Ctrl = true
			} else {
				ev.Key.Type = KeyRune
				ev.Key.Ch = r
			}
		}
		return ev, size
	}

	// 2. Escape (\x1b) karakteri ile başlayan dizi kontrolü
	if len(buf) == 1 {
		// Tamponda sadece ESC tuşu var (Sonrasının gelip gelmediği Event Loop zaman aşımı ile kontrol edilir)
		return Event{Type: EventKey, Key: KeyEvent{Type: KeyEsc}}, 1
	}

	switch buf[1] {
	case '[': // CSI (Control Sequence Introducer) dizisi
		return parseCSI(buf)
	case 'O': // SS3 Alternatif Fonksiyon tuşları dizisi (örn: \x1b[OP -> \x1bOP)
		if len(buf) < 3 {
			return Event{}, 0
		}
		ev := Event{Type: EventKey}
		switch buf[2] {
		case 'P':
			ev.Key.Type = KeyF1
		case 'Q':
			ev.Key.Type = KeyF2
		case 'R':
			ev.Key.Type = KeyF3
		case 'S':
			ev.Key.Type = KeyF4
		default:
			return Event{}, 3
		}
		return ev, 3
	default:
		// ESC + Karakter kombinasyonu (Alt + Tuş)
		r, size := utf8.DecodeRune(buf[1:])
		if r == utf8.RuneError {
			if !utf8.FullRune(buf[1:]) {
				return Event{}, 0
			}
			return Event{}, 2
		}
		ev := Event{Type: EventKey, Key: KeyEvent{Type: KeyRune, Ch: r, Alt: true}}
		return ev, 1 + size
	}
}

// parseCSI \x1b[ ile başlayan kontrol dizilerini çözümler.
func parseCSI(buf []byte) (Event, int) {
	if len(buf) < 3 {
		return Event{}, 0
	}

	// CSI dizisinin son komut karakterini (A-Z, a-z veya ~) bul
	endIdx := -1
	for i := 2; i < len(buf); i++ {
		c := buf[i]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || c == '~' {
			endIdx = i
			break
		}
	}

	// Komut bitiş karakteri bulunamadıysa dizinin tamamlanmasını bekle
	if endIdx == -1 {
		// Sonsuz döngü ve bellek şişmesini engellemek için geçersiz uzunlukta dizi kontrolü
		if len(buf) > 32 {
			return Event{}, 1
		}
		return Event{}, 0
	}

	cmd := buf[endIdx]
	paramsStr := string(buf[2:endIdx])
	consumed := endIdx + 1

	// Noktalı virgül (;) ile ayrılmış parametreleri sayı dizisine dönüştür
	var params []int
	var currentVal int
	hasVal := false
	for i := 0; i < len(paramsStr); i++ {
		c := paramsStr[i]
		if c >= '0' && c <= '9' {
			currentVal = currentVal*10 + int(c-'0')
			hasVal = true
		} else if c == ';' {
			if hasVal {
				params = append(params, currentVal)
				currentVal = 0
				hasVal = false
			} else {
				params = append(params, 0)
			}
		}
	}
	if hasVal {
		params = append(params, currentVal)
	}

	// Komut karakterine göre olayı oluştur
	switch cmd {
	case 'A': // Yukarı Ok
		return makeKeyEvent(KeyArrowUp, params), consumed
	case 'B': // Aşağı Ok
		return makeKeyEvent(KeyArrowDown, params), consumed
	case 'C': // Sağ Ok
		return makeKeyEvent(KeyArrowRight, params), consumed
	case 'D': // Sol Ok
		return makeKeyEvent(KeyArrowLeft, params), consumed
	case 'Z': // Shift+Tab (backtab)
		return Event{Type: EventKey, Key: KeyEvent{Type: KeyTab, Shift: true}}, consumed
	case 'H': // Home
		return makeKeyEvent(KeyHome, params), consumed
	case 'F': // End
		return makeKeyEvent(KeyEnd, params), consumed
	case 'I': // Focus Gained
		return Event{Type: EventFocus, Focus: FocusEvent{Gained: true}}, consumed
	case 'O': // Focus Lost
		return Event{Type: EventFocus, Focus: FocusEvent{Gained: false}}, consumed
	case 'M', 'm':
		// SGR Fare Protokolü kontrolü (\x1b[<btn;x;yM veya \x1b[<btn;x;ym)
		if len(paramsStr) > 0 && paramsStr[0] == '<' {
			return parseSGRMouse(paramsStr, cmd, consumed)
		}
	case '~':
		// Keypad ve fonksiyon tuşları (\x1b[sayı~)
		if len(params) == 0 {
			return Event{}, consumed
		}
		num := params[0]
		var modParams []int
		if len(params) > 1 {
			modParams = params[1:]
		}

		var kt KeyType
		switch num {
		case 1, 7:
			kt = KeyHome
		case 2:
			kt = KeyInsert
		case 3:
			kt = KeyDelete
		case 4, 8:
			kt = KeyEnd
		case 5:
			kt = KeyPageUp
		case 6:
			kt = KeyPageDown
		case 11:
			kt = KeyF1
		case 12:
			kt = KeyF2
		case 13:
			kt = KeyF3
		case 14:
			kt = KeyF4
		case 15:
			kt = KeyF5
		case 17:
			kt = KeyF6
		case 18:
			kt = KeyF7
		case 19:
			kt = KeyF8
		case 20:
			kt = KeyF9
		case 21:
			kt = KeyF10
		case 23:
			kt = KeyF11
		case 24:
			kt = KeyF12
		default:
			return Event{}, consumed
		}
		return makeKeyEvent(kt, modParams), consumed
	}

	return Event{}, consumed
}

// makeKeyEvent tuş modifikatörlerini çözümler ve KeyEvent olayını döner.
func makeKeyEvent(kt KeyType, params []int) Event {
	ev := Event{Type: EventKey}
	ev.Key.Type = kt
	if len(params) > 1 {
		ev.Key.Shift, ev.Key.Alt, ev.Key.Ctrl = decodeModifiers(params[1])
	} else if len(params) == 1 {
		ev.Key.Shift, ev.Key.Alt, ev.Key.Ctrl = decodeModifiers(params[0])
	}
	return ev
}

// decodeModifiers standart VT100/Xterm modifikatör kodlarını çözümler.
func decodeModifiers(code int) (shift, alt, ctrl bool) {
	if code <= 1 {
		return
	}
	shift = (code-1)&1 != 0
	alt = (code-1)&2 != 0
	ctrl = (code-1)&4 != 0
	return
}

// parseSGRMouse SGR fare formatını (\x1b[<btn;x;yM/m) çözümleyip MouseEvent üretir.
func parseSGRMouse(paramsStr string, cmd byte, consumed int) (Event, int) {
	s := paramsStr[1:] // Başındaki '<' karakterini atla

	var params []int
	var currentVal int
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= '0' && c <= '9' {
			currentVal = currentVal*10 + int(c-'0')
		} else if c == ';' {
			params = append(params, currentVal)
			currentVal = 0
		}
	}
	params = append(params, currentVal)

	if len(params) < 3 {
		return Event{}, consumed
	}

	btnCode := params[0]
	mouseX := params[1]
	mouseY := params[2]

	ev := Event{Type: EventMouse}

	// 1-tabanlı terminal koordinatlarını 0-tabanlı koordinata dönüştür
	if mouseX > 0 {
		ev.Mouse.X = uint16(mouseX - 1)
	}
	if mouseY > 0 {
		ev.Mouse.Y = uint16(mouseY - 1)
	}

	// Modifikatör bitlerini kontrol et (Shift: 4, Alt: 8, Ctrl: 16)
	ev.Mouse.Shift = (btnCode & 4) != 0
	ev.Mouse.Alt = (btnCode & 8) != 0
	ev.Mouse.Ctrl = (btnCode & 16) != 0

	// Modifikatör bitlerini temizleyerek butonu ve sürükleme bilgisini ayır
	btnRaw := btnCode & ^(4 | 8 | 16)
	ev.Mouse.Drag = (btnRaw & 32) != 0
	btnBase := btnRaw & ^32

	if cmd == 'm' {
		// Tuş bırakma olayı
		ev.Mouse.Button = MouseRelease
	} else {
		switch btnBase {
		case 0:
			ev.Mouse.Button = MouseLeft
		case 1:
			ev.Mouse.Button = MouseMiddle
		case 2:
			ev.Mouse.Button = MouseRight
		case 64:
			ev.Mouse.Button = MouseScrollUp
		case 65:
			ev.Mouse.Button = MouseScrollDown
		default:
			ev.Mouse.Button = MouseNone
		}
	}

	return ev, consumed
}
