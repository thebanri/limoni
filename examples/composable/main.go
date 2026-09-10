package main

import (
	"fmt"

	"github.com/thebanri/limoni"
	"github.com/thebanri/limoni/widgets"
)

func main() {
	listState := limoni.NewListState()
	listState.Select(0) // Varsayılan olarak ilk öğe seçili

	inputState := limoni.NewTextInputState()
	inputState.SetValue("Limoni Lego UI")

	menuItems := []string{
		"🧱 Unified Component",
		"🎨 Pad, Border, Align",
		"📐 VStack & HStack (Flex)",
		"🥞 ZStack (Depth-Axis)",
		"🎯 Overlay (Compositor)",
		"⚡ Transform (Live Text)",
		"🔲 Border Presets Explained",
		"❓ Conditionals (When/Match)",
		"🎭 Style Cascading",
		"🖱️ Interactive & Events",
		"⚡ Zero-Alloc Performance",
	}

	list := limoni.NewList(menuItems...).
		WithState(listState).
		WithHighlightSymbol("👉 ").
		WithSelectedStyle(limoni.Fg(limoni.Hex("#000000")).WithBg(limoni.Hex("#00FFAA")).Bold())

	// Performans tablosu (10. öğe için)
	table := limoni.NewTable().
		WithHeaders("PRIMITIVE", "STATUS", "LATENCY", "ALLOCS").
		WithRow("VStack / HStack Draw", "PASS", "660 ns", "0 B/op (0 allocs)").
		WithRow("ZStack Layering", "PASS", "431 ns", "0 B/op (0 allocs)").
		WithRow("Overlay Compositor", "PASS", "368 ns", "0 B/op (0 allocs)").
		WithRow("Transform (Uppercase)", "PASS", "1061 ns", "0 B/op (0 allocs)").
		WithRow("Transform (Mask)", "PASS", "623 ns", "0 B/op (0 allocs)").
		WithRow("Spacer Draw", "PASS", "131 ns", "0 B/op (0 allocs)").
		WithRow("Divider Draw", "PASS", "1666 ns", "0 B/op (0 allocs)").
		WithRow("Margin Draw", "PASS", "215 ns", "0 B/op (0 allocs)").
		WithRow("BorderEdges Draw", "PASS", "2300 ns", "0 B/op (0 allocs)").
		WithRow("Constrain Draw", "PASS", "566 ns", "0 B/op (0 allocs)").
		WithRow("Flexbox Justify/Align", "PASS", "771 ns", "0 B/op (0 allocs)").
		WithRow("Style Cascading", "PASS", "435 ns", "0 B/op (0 allocs)").
		WithSelectedStyle(limoni.Fg(limoni.ColorWhite).WithBg(limoni.Hex("#224466"))).
		WithConstraints(
			widgets.TableConstraint{Type: widgets.ConstraintPercentage, Value: 35},
			widgets.TableConstraint{Type: widgets.ConstraintPercentage, Value: 15},
			widgets.TableConstraint{Type: widgets.ConstraintPercentage, Value: 20},
			widgets.TableConstraint{Type: widgets.ConstraintPercentage, Value: 30},
		)

	input := limoni.NewTextInput("demo_input").
		WithState(inputState).
		WithStyle(limoni.Fg(limoni.Hex("#FFFFFF"))).
		WithFocusedStyle(limoni.Fg(limoni.Hex("#FFFF00")).Bold())

	showModal := false
	clickCount := 0
	modes := []string{"monitoring", "debug", "production"}
	modeIdx := 0

	err := limoni.Run(func(f *limoni.Frame, ev *limoni.Event) bool {
		// Klavye olayları
		if ev != nil && ev.Type == limoni.EventKey {
			switch ev.Key.Type {
			case limoni.KeyEsc:
				if showModal {
					showModal = false
				} else {
					return false
				}
			case limoni.KeyRune:
				switch ev.Key.Ch {
				case '?', 'h', 'H':
					showModal = !showModal
				case 'm', 'M':
					modeIdx = (modeIdx + 1) % len(modes)
				case 'c', 'C':
					clickCount++
				default:
					inputState.HandleKey(ev.Key)
				}
			case limoni.KeyUp:
				if listState.Selected > 0 {
					listState.Previous()
				}
			case limoni.KeyDown:
				if listState.Selected < len(menuItems)-1 {
					listState.Next()
				}
			default:
				inputState.HandleKey(ev.Key)
			}
		}

		currentMode := modes[modeIdx]

		// Mode rozeti (Match fonksiyonu ile)
		modeBadge := limoni.Match(currentMode, map[string]limoni.Component{
			"monitoring": limoni.WithForeground(limoni.Hex("#00FFAA"), limoni.Label("🟢 MONITORING")),
			"debug":      limoni.WithForeground(limoni.Hex("#FFCC00"), limoni.Label("🟡 DEBUG")),
			"production": limoni.WithForeground(limoni.Hex("#FF5555"), limoni.Label("🔴 PROD")),
		}, limoni.Label("⚪ UNKNOWN"))

		// Tıklanabilir buton
		clickBtn := limoni.OnClick(
			limoni.Border(
				limoni.Pad(
					limoni.Label(fmt.Sprintf("🖱️ Clicks: %d", clickCount), limoni.Bold().WithFg(limoni.Hex("#FF77AA"))),
					0, 1, 0, 1,
				),
				widgets.SymbolsRounded,
				limoni.Fg(limoni.Hex("#FF77AA")),
			),
			func(m limoni.MouseEvent) {
				clickCount++
			},
		)

		// -----------------------------------------------------------------
		// Sol Sütun: Menü Listesi + Overlay Rozeti
		// -----------------------------------------------------------------
		listColumn := limoni.Overlay(
			limoni.Border(
				limoni.Pad(limoni.AsComponent(list), 1, 0, 0, 0),
				widgets.SymbolsRounded,
				limoni.Fg(limoni.Hex("#5588EE")),
			),
			limoni.FixedSize(14, 1,
				limoni.Label(" 🧱 MODÜLLER ", limoni.Bold().WithFg(limoni.Hex("#000000")).WithBg(limoni.Hex("#5588EE"))),
			),
			3, 0,
		)

		// -----------------------------------------------------------------
		// Sağ Sütun: Seçili Menüye Göre Dinamik İçerik
		// -----------------------------------------------------------------
		selectedIndex := listState.Selected
		if selectedIndex < 0 {
			selectedIndex = 0
		}

		var detailContent limoni.Component

		switch selectedIndex {
		case 0: // 🧱 Unified Component
			detailContent = limoni.VStack(
				limoni.Label("🧱 Unified Component Interface", limoni.Bold().WithFg(limoni.Hex("#00FFAA"))),
				limoni.Label("Tüm görsel elemanlar aynı Component arayüzünü uygular:", limoni.Fg(limoni.Hex("#AAAAAA"))),
				limoni.Label("  • Draw(ctx, buf)        -> Çizim tamponuna doğrudan çizim (0 B/op)", limoni.Fg(limoni.Hex("#CCCCCC"))),
				limoni.Label("  • LayoutInfo(maxArea)   -> Flex/Min/Max kısıtlarını bildirir", limoni.Fg(limoni.Hex("#CCCCCC"))),
				limoni.Label("  • SizeHint(maxArea)     -> Boyut ipucunu döner", limoni.Fg(limoni.Hex("#CCCCCC"))),
				limoni.Label(""),
				limoni.Label("Canlı Örnek: İç İçe Sarmalama (Nesting)", limoni.Bold().WithFg(limoni.Hex("#FFCC00"))),
				limoni.Border(
					limoni.Pad(
						limoni.VStack(
							limoni.Label("Dış Kutu: Border(Cyan) -> Pad()", limoni.Fg(limoni.Hex("#00FFFF"))),
							limoni.Border(
								limoni.Pad(
									limoni.Label("İç Kutu: Border(Pink) -> Pad() -> Label", limoni.Fg(limoni.Hex("#FF77AA"))),
									0, 1, 0, 1,
								),
								widgets.SymbolsRounded,
								limoni.Fg(limoni.Hex("#FF77AA")),
							),
						),
						1, 2, 1, 2,
					),
					widgets.SymbolsRounded,
					limoni.Fg(limoni.Hex("#00FFFF")),
				),
			)

		case 1: // 🎨 Pad, Border, Align
			detailContent = limoni.VStack(
				limoni.Label("🎨 Pad, Border ve Align Düzenleyicileri", limoni.Bold().WithFg(limoni.Hex("#00FFAA"))),
				limoni.Label("Bileşenleri sıfır bellek tahsisiyle sarmalayan dekoratörler:", limoni.Fg(limoni.Hex("#AAAAAA"))),
				limoni.Label(""),
				limoni.HStack(
					limoni.Flex(1, limoni.Border(
						limoni.Pad(limoni.Center(limoni.Label("Center()\nOrtalanmış")), 1, 1, 1, 1),
						widgets.SymbolsRounded,
						limoni.Fg(limoni.Hex("#00FFAA")),
					)),
					limoni.Flex(1, limoni.Border(
						limoni.AlignComponent(limoni.Label("AlignTopLeft"), limoni.HAlignLeft, limoni.AlignTop),
						widgets.SymbolsDouble,
						limoni.Fg(limoni.Hex("#FFCC00")),
					)),
					limoni.Flex(1, limoni.Border(
						limoni.AlignComponent(limoni.Label("AlignBottomRight"), limoni.HAlignRight, limoni.AlignBottom),
						widgets.SymbolsThick,
						limoni.Fg(limoni.Hex("#FF77AA")),
					)),
				),
			)

		case 2: // 📐 VStack & HStack (Flex)
			detailContent = limoni.VStack(
				limoni.Label("📐 Flexbox Yığınları (VStack & HStack)", limoni.Bold().WithFg(limoni.Hex("#00FFAA"))),
				limoni.Label("Flex ağırlıkları, JustifyContent ve AlignItems ile tam flexbox düzeni.", limoni.Fg(limoni.Hex("#AAAAAA"))),
				limoni.Label(""),
				limoni.Label("Orantılı Genişlikler (Flex 1 : Flex 2 : Flex 1):", limoni.Fg(limoni.Hex("#FFFF00"))),
				limoni.HStack(
					limoni.Flex(1, limoni.Border(limoni.Center(limoni.Label("Flex(1) - 25%")), widgets.SymbolsRounded, limoni.Fg(limoni.Hex("#00FFAA")))),
					limoni.Flex(2, limoni.Border(limoni.Center(limoni.Label("Flex(2) - 50%")), widgets.SymbolsRounded, limoni.Fg(limoni.Hex("#3399FF")))),
					limoni.Flex(1, limoni.Border(limoni.Center(limoni.Label("Flex(1) - 25%")), widgets.SymbolsRounded, limoni.Fg(limoni.Hex("#00FFAA")))),
				),
				limoni.Label(""),
				limoni.Label("Hizalama: SpaceBetween / Center ile boşluk dağıtımı.", limoni.Fg(limoni.Hex("#888888"))),
			)

		case 3: // 🥞 ZStack (Depth)
			detailContent = limoni.VStack(
				limoni.Label("🥞 ZStack (Derinlik / Ressam Algoritması)", limoni.Bold().WithFg(limoni.Hex("#00FFAA"))),
				limoni.Label("Bileşenleri arka plandan ön plana üst üste çizer. Ara bellek (buffer) tahsis etmez.", limoni.Fg(limoni.Hex("#AAAAAA"))),
				limoni.Label(""),
				limoni.Border(
					limoni.Pad(
						limoni.ZStack(
							limoni.Label(". . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . .\n. Arka plan katmanı (Z-0) . . . . . . . . . . . . . . . . . . . . . . . . . . .\n. . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . .\n. . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . .\n. . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . .", limoni.Fg(limoni.Hex("#444444"))),
							limoni.Center(
								limoni.Border(
									limoni.Pad(limoni.Label("Ön Plan Kartı (Z-1)", limoni.Bold().WithFg(limoni.Hex("#FFFFFF"))), 0, 2, 0, 2),
									widgets.SymbolsDouble,
									limoni.Fg(limoni.Hex("#FFCC00")),
								),
							),
						),
						1, 1, 1, 1,
					),
					widgets.SymbolsRounded,
					limoni.Fg(limoni.Hex("#666666")),
				),
				limoni.Label("💡 İpucu: Tam ekran modalı açmak için [?] veya [h] tuşuna basabilirsiniz.", limoni.Fg(limoni.Hex("#FFD700"))),
			)

		case 4: // 🎯 Overlay
			detailContent = limoni.VStack(
				limoni.Label("🎯 Overlay (Mutlak Koordinat Bindirici)", limoni.Bold().WithFg(limoni.Hex("#00FFAA"))),
				limoni.Label("Bir bileşenin üzerine (X, Y) koordinatıyla başka bir bileşeni bindirir.", limoni.Fg(limoni.Hex("#AAAAAA"))),
				limoni.Label("Sol menünün üstündeki rozet 'Overlay' ile yerleştirilmiştir.", limoni.Fg(limoni.Hex("#AAAAAA"))),
				limoni.Label(""),
				limoni.Overlay(
					limoni.Border(
						limoni.Pad(
							limoni.VStack(
								limoni.Label("Ana Panel (Base)", limoni.Bold().WithFg(limoni.Hex("#FFFFFF"))),
								limoni.Label("Bu panelin sol üst veya sağ üst köşesine"),
								limoni.Label("Overlay ile rozet, bildirim veya ipucu eklenebilir."),
							),
							1, 2, 1, 2,
						),
						widgets.SymbolsRounded,
						limoni.Fg(limoni.Hex("#3399FF")),
					),
					limoni.FixedSize(14, 1,
						limoni.Label(" ★ BİLDİRİM ", limoni.Bold().WithFg(limoni.Hex("#000000")).WithBg(limoni.Hex("#FF5599"))),
					),
					3, 0, // X: 3, Y: 0 (Üst kenarlık hizası)
				),
			)

		case 5: // ⚡ Transform
			inputValue := inputState.Value()
			if inputValue == "" {
				inputValue = "Merhaba Dunya"
			}
			detailContent = limoni.VStack(
				limoni.Label("⚡ Transform (Yerinde Hücre Dönüştürücü)", limoni.Bold().WithFg(limoni.Hex("#00FFAA"))),
				limoni.Label("String kopyalaması yapmadan tampon üzerindeki hücreleri anında dönüştürür.", limoni.Fg(limoni.Hex("#AAAAAA"))),
				limoni.Label("Aşağıdaki kutuya bir şeyler yazın ve anlık değişimi izleyin:", limoni.Fg(limoni.Hex("#FFFF00"))),
				limoni.Label(""),
				limoni.HStack(
					limoni.Flex(1, limoni.Border(
						limoni.VStack(
							limoni.Label("Orijinal Metin:", limoni.Fg(limoni.Hex("#888888"))),
							limoni.Label(inputValue, limoni.Fg(limoni.Hex("#FFFFFF"))),
						),
						widgets.SymbolsRounded,
						limoni.Fg(limoni.Hex("#666666")),
					)),
					limoni.Flex(1, limoni.Border(
						limoni.VStack(
							limoni.Label("Uppercase(Metin):", limoni.Fg(limoni.Hex("#888888"))),
							limoni.Uppercase(limoni.Label(inputValue, limoni.Bold().WithFg(limoni.Hex("#00FFAA")))),
						),
						widgets.SymbolsRounded,
						limoni.Fg(limoni.Hex("#00FFAA")),
					)),
				),
				limoni.HStack(
					limoni.Flex(1, limoni.Border(
						limoni.VStack(
							limoni.Label("Lowercase(Metin):", limoni.Fg(limoni.Hex("#888888"))),
							limoni.Lowercase(limoni.Label(inputValue, limoni.Fg(limoni.Hex("#3399FF")))),
						),
						widgets.SymbolsRounded,
						limoni.Fg(limoni.Hex("#3399FF")),
					)),
					limoni.Flex(1, limoni.Border(
						limoni.VStack(
							limoni.Label("Mask(Metin, '•'):", limoni.Fg(limoni.Hex("#888888"))),
							limoni.Mask(limoni.Label(inputValue, limoni.Fg(limoni.Hex("#FF77AA"))), '•'),
						),
						widgets.SymbolsRounded,
						limoni.Fg(limoni.Hex("#FF77AA")),
					)),
				),
			)

		case 6: // 🔲 Border Presets Explained
			detailContent = limoni.VStack(
				limoni.Label("🔲 Kenarlık Çeşitleri & Half-Block Farkı", limoni.Bold().WithFg(limoni.Hex("#00FFAA"))),
				limoni.Label("Half-Block (Yarım Blok) kenarlıklar Unicode yarım blok karakterlerini (▀ ▄ ▌ ▐ ▛ ▜ ▙ ▟)", limoni.Fg(limoni.Hex("#AAAAAA"))),
				limoni.Label("kullanır. Bazı terminal yazı tiplerinde bu karakterler kalın veya pikselli görünebilir.", limoni.Fg(limoni.Hex("#AAAAAA"))),
				limoni.Label("Temiz hatlar için Rounded veya Double tercih edilebilir:", limoni.Fg(limoni.Hex("#AAAAAA"))),
				limoni.Label(""),
				limoni.HStack(
					limoni.Flex(1, limoni.Border(
						limoni.Center(limoni.Label("Rounded\n╭───╮\n╰───╯")),
						widgets.SymbolsRounded,
						limoni.Fg(limoni.Hex("#00FFAA")),
					)),
					limoni.Flex(1, limoni.Border(
						limoni.Center(limoni.Label("Double\n╔═══╗\n╚═══╝")),
						widgets.SymbolsDouble,
						limoni.Fg(limoni.Hex("#3399FF")),
					)),
					limoni.Flex(1, limoni.Border(
						limoni.Center(limoni.Label("Thick\n┏━━━┓\n┗━━━┛")),
						widgets.SymbolsThick,
						limoni.Fg(limoni.Hex("#FFCC00")),
					)),
				),
				limoni.HStack(
					limoni.Flex(1, limoni.Border(
						limoni.Center(limoni.Label("Single\n┌───┐\n└───┘")),
						widgets.SymbolsSingle,
						limoni.Fg(limoni.Hex("#AAAAAA")),
					)),
					limoni.Flex(1, limoni.Border(
						limoni.Center(limoni.Label("OuterHalfBlock\n▛▀▀▀▜\n▙▄▄▄▟")),
						widgets.SymbolsOuterHalfBlock,
						limoni.Fg(limoni.Hex("#FF77AA")),
					)),
					limoni.Flex(1, limoni.Border(
						limoni.Center(limoni.Label("InnerHalfBlock\n▗▄▄▄▖\n▘▀▀▀▝")),
						widgets.SymbolsInnerHalfBlock,
						limoni.Fg(limoni.Hex("#FFAA88")),
					)),
				),
			)

		case 7: // ❓ Conditionals (When/Match)
			detailContent = limoni.VStack(
				limoni.Label("❓ Koşullu Çizim (When & Match)", limoni.Bold().WithFg(limoni.Hex("#00FFAA"))),
				limoni.Label("Koşul sağlanmadığında DOM'a boş bileşen vererek sıfır maliyet sağlar.", limoni.Fg(limoni.Hex("#AAAAAA"))),
				limoni.Label(""),
				limoni.Label("Klavye [m] tuşuna basarak modu değiştirin:", limoni.Fg(limoni.Hex("#FFFF00"))),
				limoni.Border(
					limoni.Pad(
						limoni.VStack(
							limoni.HStack(
								limoni.Label("Şu anki mod: "),
								modeBadge,
							),
							limoni.When(currentMode == "monitoring",
								limoni.Label("📊 Sistem metrikleri izleniyor... Her şey yolunda.", limoni.Fg(limoni.Hex("#00FFAA"))),
							),
							limoni.When(currentMode == "debug",
								limoni.Label("🐛 Debug logları aktif: Ayrıntılı bellek izleme açık.", limoni.Fg(limoni.Hex("#FFCC00"))),
							),
							limoni.When(currentMode == "production",
								limoni.Label("🚀 Production modu: Maksimum performans kilidi açık.", limoni.Fg(limoni.Hex("#FF5555"))),
							),
						),
						1, 2, 1, 2,
					),
					widgets.SymbolsRounded,
					limoni.Fg(limoni.Hex("#FFFFFF")),
				),
			)

		case 8: // 🎭 Style Cascading
			detailContent = limoni.VStack(
				limoni.Label("🎭 Stil Kalıtımı (Cascading Styles)", limoni.Bold().WithFg(limoni.Hex("#00FFAA"))),
				limoni.Label("Üst bileşenin stilleri alt bileşenlere otomatik aktarılır.", limoni.Fg(limoni.Hex("#AAAAAA"))),
				limoni.Label("Alt bileşen sadece değiştirmek istediği özelliği (Fg, Bg, Bold) ezer.", limoni.Fg(limoni.Hex("#AAAAAA"))),
				limoni.Label(""),
				limoni.WithBackground(limoni.Hex("#1A202C"),
					limoni.Border(
						limoni.Pad(
							limoni.WithForeground(limoni.Hex("#E2E8F0"),
								limoni.VStack(
									limoni.Label("Üst Kutu: Koyu arka plan + Açık gri metin"),
									limoni.WithForeground(limoni.Hex("#48BB78"), limoni.Label("  → Alt Satır: Rengi yeşil yaptık (Arka plan miras alındı)")),
									limoni.WithForeground(limoni.Hex("#F6E05E"), limoni.Label("  → Alt Satır: Rengi sarı yaptık")),
									limoni.Label("  → Alt Satır: Üst rengi kullanmaya devam eder"),
								),
							),
							1, 2, 1, 2,
						),
						widgets.SymbolsRounded,
						limoni.Fg(limoni.Hex("#4A5568")),
					),
				),
			)

		case 9: // 🖱️ Interactive & Events
			detailContent = limoni.VStack(
				limoni.Label("🖱️ Etkileşim & Olay Yönlendirme (Event Routing)", limoni.Bold().WithFg(limoni.Hex("#00FFAA"))),
				limoni.Label("Fare koordinatları bileşenin kendi lokal alanına göre otomatik hesaplanır.", limoni.Fg(limoni.Hex("#AAAAAA"))),
				limoni.Label(""),
				limoni.Label(fmt.Sprintf("Toplam Tıklama: %d (Aşağıdaki butona tıklayın veya [c] tuşuna basın)", clickCount), limoni.Fg(limoni.Hex("#FFFF00"))),
				limoni.Label(""),
				limoni.HStack(
					clickBtn,
					limoni.OnClick(
						limoni.Border(
							limoni.Pad(limoni.Label("🔄 Sayacı Sıfırla", limoni.Bold().WithFg(limoni.Hex("#FF5555"))), 0, 1, 0, 1),
							widgets.SymbolsRounded,
							limoni.Fg(limoni.Hex("#FF5555")),
						),
						func(m limoni.MouseEvent) {
							clickCount = 0
						},
					),
				).WithGap(2),
			)

		case 10: // ⚡ Zero-Alloc Performance
			detailContent = limoni.VStack(
				limoni.Label("⚡ Sıfır Bellek Tahsisi (Zero-Allocation Benchmarks)", limoni.Bold().WithFg(limoni.Hex("#00FFAA"))),
				limoni.Label("Tüm çizim ve yerleşim hesaplamaları 0 B/op (0 bayt) bellek ayırır:", limoni.Fg(limoni.Hex("#AAAAAA"))),
				limoni.Label(""),
				limoni.Flex(1, limoni.Border(
					limoni.AsComponent(table),
					widgets.SymbolsRounded,
					limoni.Fg(limoni.Hex("#00FFAA")),
				)),
			)
		}

		// -------------------------------------------------------------
		// Base Dashboard View (VStack + HStack):
		// -------------------------------------------------------------
		baseDashboard := limoni.VStack(
			// Top Header (Fixed Height 3): Justified title and status badge
			limoni.FixedSize(0, 3, limoni.Border(
				limoni.Pad(
					limoni.HStack(
						limoni.Label("🍋 LIMONI LEGO ARCHITECTURE", limoni.Bold().WithFg(limoni.Hex("#00FFAA"))),
						limoni.Uppercase(limoni.Label("v1.0 • 0 B/op", limoni.Fg(limoni.Hex("#88CCFF")))),
						modeBadge,
						clickBtn,
					).WithJustify(limoni.JustifySpaceBetween).WithAlignItems(limoni.AlignItemsCenter),
					0, 1, 0, 1,
				),
				widgets.SymbolsRounded,
				limoni.Fg(limoni.Hex("#00FFAA")),
			)),

			// Main Body: HStack with 1:2 flex ratio
			limoni.Flex(1, limoni.HStack(
				// Left Column: List wrapped in a rounded border + Overlay badge
				limoni.Flex(1, listColumn),

				// Right Column: Dynamic Detail View
				limoni.Flex(2, limoni.Border(
					limoni.Pad(detailContent, 1, 2, 1, 2),
					widgets.SymbolsRounded,
					limoni.Fg(limoni.Hex("#3399FF")),
				)),
			).WithGap(1)),

			// Bottom Input Bar (Fixed Height 3)
			limoni.FixedSize(0, 3, limoni.HStack(
				limoni.Flex(3, limoni.Border(
					limoni.Pad(limoni.AsComponent(input), 0, 1, 0, 1),
					widgets.SymbolsRounded,
					limoni.Fg(limoni.Hex("#FF5599")),
				)),
				limoni.Flex(1, limoni.Border(
					limoni.Pad(
						limoni.Mask(limoni.Label(inputState.Value(), limoni.Fg(limoni.Hex("#FFAA88"))), '•'),
						0, 1, 0, 1,
					),
					widgets.SymbolsRounded,
					limoni.Fg(limoni.Hex("#FFAA88")),
				)),
			).WithGap(1)),

			// Footer: SpaceBetween distribution with shortcuts
			limoni.FixedSize(0, 3, limoni.Border(
				limoni.Pad(
					limoni.HStack(
						limoni.WithForeground(limoni.Hex("#888888"), limoni.Label("↑/↓: Modül Seç | c: Tıkla | m: Mod Değiştir | ?: Modal | ESC: Çıkış")),
						limoni.WithForeground(limoni.Hex("#00FFAA"), limoni.Label("0 B/op Zero Alloc")),
					).WithJustify(limoni.JustifySpaceBetween).WithAlignItems(limoni.AlignItemsCenter),
					0, 1, 0, 1,
				),
				widgets.SymbolsSingle,
				limoni.Fg(limoni.Hex("#555555")),
			)),
		)

		// -------------------------------------------------------------
		// Modal Overlay via ZStack + When (Painter's Algorithm):
		// -------------------------------------------------------------
		modalLayer := limoni.When(showModal,
			limoni.Center(
				limoni.FixedSize(60, 14,
					limoni.WithBackground(limoni.Hex("#111827"),
						limoni.Border(
							limoni.Pad(
								limoni.VStack(
									limoni.Center(limoni.Label("✨ LEGO BLOK MİMARİSİ ✨", limoni.Bold().WithFg(limoni.Hex("#00FFAA")))),
									limoni.Label("• ZStack: Sıfır ek tamponla derinlik ekseninde katmanlama", limoni.Fg(limoni.Hex("#FFFFFF"))),
									limoni.Label("• Overlay: Mutlak koordinatla bileşen bindirme", limoni.Fg(limoni.Hex("#E5E7EB"))),
									limoni.Label("• Transform: Bellek kopyalamasız yerinde hücre dönüştürme", limoni.Fg(limoni.Hex("#D1D5DB"))),
									limoni.Label("• Flexbox: JustifyContent & AlignItems destekli HStack/VStack", limoni.Fg(limoni.Hex("#9CA3AF"))),
									limoni.Label("• Conditionals: Bildirimsel When & Match akış kontrolü", limoni.Fg(limoni.Hex("#6EE7B7"))),
									limoni.Label("• Stil Kalıtımı: WithStyle, WithForeground ile otomatik miras", limoni.Fg(limoni.Hex("#93C5FD"))),
									limoni.Center(limoni.Label("[ Kapatmak için Esc veya ? tuşuna basın ]", limoni.Fg(limoni.Hex("#F59E0B")).Bold())),
								).WithJustify(limoni.JustifySpaceAround),
								1, 2, 1, 2,
							),
							widgets.SymbolsDouble,
							limoni.Fg(limoni.Hex("#F59E0B")),
						),
					),
				),
			),
		)

		// ZStack: Taban görünümü ve modal katmanını birleştirir
		rootView := limoni.ZStack(
			baseDashboard,
			modalLayer,
		)

		f.RenderComponent(rootView, f.Area())
		return true
	})

	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
