package backend

// EventType olay türünü temsil eder.
type EventType uint8

const (
	EventKey EventType = iota
	EventMouse
	EventResize
	EventFocus
	EventPaste
)

// KeyType özel klavye tuşlarını temsil eder.
type KeyType uint16

const (
	KeyRune KeyType = iota
	KeyEsc
	KeyEnter
	KeyBackspace
	KeyTab
	KeySpace

	KeyArrowUp
	KeyArrowDown
	KeyArrowLeft
	KeyArrowRight

	KeyHome
	KeyEnd
	KeyPageUp
	KeyPageDown
	KeyInsert
	KeyDelete

	KeyF1
	KeyF2
	KeyF3
	KeyF4
	KeyF5
	KeyF6
	KeyF7
	KeyF8
	KeyF9
	KeyF10
	KeyF11
	KeyF12
)

// KeyEvent klavye tuş basımını ve modifier durumlarını temsil eder.
type KeyEvent struct {
	Type  KeyType // Tuşun türü (Rune veya Özel Tuş)
	Ch    rune    // Eğer Type == KeyRune ise basılan karakter
	Alt   bool    // Alt tuşu basılı mı?
	Ctrl  bool    // Ctrl tuşu basılı mı?
	Shift bool    // Shift tuşu basılı mı?
}

// MouseButton fare butonlarını temsil eder.
type MouseButton uint8

const (
	MouseNone MouseButton = iota
	MouseLeft
	MouseRight
	MouseMiddle
	MouseRelease
	MouseScrollUp
	MouseScrollDown
)

// MouseEvent fare hareketlerini, tıklamalarını ve koordinatlarını temsil eder.
type MouseEvent struct {
	Button MouseButton // Tıklanan buton
	X      uint16      // Terminal koordinat sisteminde X konumu (0-tabanlı)
	Y      uint16      // Terminal koordinat sisteminde Y konumu (0-tabanlı)
	Drag   bool        // Sürükleme hareketi mi?
	Alt    bool        // Alt tuşu basılı mı?
	Ctrl   bool        // Ctrl tuşu basılı mı?
	Shift  bool        // Shift tuşu basılı mı?
}

// ResizeEvent terminal penceresi boyut değişimini temsil eder.
type ResizeEvent struct {
	Width  uint16
	Height uint16
}

// FocusEvent pencere odaklanma durumunu temsil eder (Focus Gained / Focus Lost).
type FocusEvent struct {
	Gained bool
}

type PasteEvent struct{ Text string }

// Event tüm TUI olaylarını tek bir düz yapıda birleştiren kapsayıcıdır.
// Interface'ler yerine bu yapıyı kullanmak bellek tahsisatını (heap allocation) sıfıra indirir.
type Event struct {
	Type   EventType
	Key    KeyEvent
	Mouse  MouseEvent
	Resize ResizeEvent
	Focus  FocusEvent
	Paste  PasteEvent
}

// PlatformCapabilityMatrix reports terminal and OS capability support.
type PlatformCapabilityMatrix struct {
	OS             string
	HasRawMode     bool
	HasMouseSGR    bool
	HasFocusReport bool
	HasAltBuffer   bool
	HasIoctlResize bool
}

// GetPlatformCapabilities returns capabilities based on operating system.
func GetPlatformCapabilities(goos string) PlatformCapabilityMatrix {
	switch goos {
	case "linux", "darwin", "freebsd", "openbsd", "netbsd":
		return PlatformCapabilityMatrix{
			OS:             goos,
			HasRawMode:     true,
			HasMouseSGR:    true,
			HasFocusReport: true,
			HasAltBuffer:   true,
			HasIoctlResize: true,
		}
	case "windows":
		return PlatformCapabilityMatrix{
			OS:             goos,
			HasRawMode:     true,
			HasMouseSGR:    true,
			HasFocusReport: true,
			HasAltBuffer:   true,
			HasIoctlResize: false,
		}
	default:
		return PlatformCapabilityMatrix{
			OS:             goos,
			HasRawMode:     false,
			HasMouseSGR:    false,
			HasFocusReport: false,
			HasAltBuffer:   false,
			HasIoctlResize: false,
		}
	}
}
