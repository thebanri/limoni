package main

import (
	"fmt"

	"github.com/thebanri/limoni"
)

func main() {
	listState := limoni.NewListState()
	inputState := limoni.NewTextInputState()
	inputState.SetValue("Örnek Türkçe Metin 🇹🇷")

	err := limoni.Run(func(f *limoni.Frame, ev *limoni.Event) bool {
		// Tuş kontrolleri
		if ev != nil && ev.Type == limoni.EventKey {
			switch ev.Key.Type {
			case limoni.KeyEsc:
				return false
			case limoni.KeyUp:
				listState.Previous()
			case limoni.KeyDown:
				listState.Next()
			default:
				inputState.HandleKey(ev.Key)
			}
		}

		area := f.Area()

		// 1. Ekranı dikey olarak Başlık (3), Gövde (kalan) ve Alt Bilgi (3) olarak böl
		rows := limoni.SplitVertical(area,
			limoni.Fixed(3),
			limoni.Fill(),
			limoni.Fixed(3),
		)

		// Başlık Paneli
		header := limoni.NewBlock().
			WithTitle("🍋 Limoni TUI - Yeni Basit ve Güçlü API 🚀").
			WithTitleAlign(limoni.AlignCenter).
			WithBorderStyle(limoni.Fg(limoni.Hex("#00FFAA"))).
			WithChild(
				limoni.NewParagraph("Harf ve border kayması olmadan, kusursuz Unicode ve Emoji desteği!").
					WithStyle(limoni.Fg(limoni.Hex("#EEEEEE")).Bold()),
			)
		f.RenderWidget(header, rows[0])

		// Gövdeyi yatay olarak Sol (35%) ve Sağ (65%) sütunlara böl
		cols := limoni.SplitHorizontal(rows[1],
			limoni.Percentage(35),
			limoni.Percentage(65),
		)

		// Sol Sütun: Liste
		leftBlock := limoni.NewBlock().
			WithTitle("📋 Görev Listesi").
			WithTitleAlign(limoni.AlignLeft).
			WithBorderStyle(limoni.Fg(limoni.Hex("#FFCC00"))).
			WithChild(
				limoni.NewList(
					"✅ API Kullanımı Basitleştirildi",
					"✅ Harf ve Border Kaymaları Düzeltildi",
					"✅ Türkçe Karakter Desteği (ç, ğ, ı, ö, ş, ü, İ)",
					"⚡ Emoji & BMP Sembol Desteği (🚀, ⚠️, ✨)",
					"🔍 Sıfır Bellek Tahsisi (Zero-Allocation)",
				).
					WithState(listState).
					WithHighlightSymbol("👉 ").
					WithSelectedStyle(limoni.Fg(limoni.Hex("#000000")).WithBg(limoni.Hex("#00FFAA")).Bold()),
			)
		f.RenderWidget(leftBlock, cols[0])

		// Sağ Sütun: Tablo & Metin Girişi
		rightRows := limoni.SplitVertical(cols[1],
			limoni.Fill(),
			limoni.Fixed(3),
		)

		// Tablo
		tableBlock := limoni.NewBlock().
			WithTitle("📊 Sistem Durumu ve Metrikler").
			WithTitleAlign(limoni.AlignLeft).
			WithBorderStyle(limoni.Fg(limoni.Hex("#3399FF"))).
			WithChild(
				limoni.NewTable().
					WithHeaders("Bileşen", "Durum", "Gecikme", "Bellek").
					WithRow("Diff Motoru", "Aktif ✓", "18.3 µs", "0 B/op").
					WithRow("Unicode Parser", "Hatasız ✓", "5.7 µs", "0 B/op").
					WithRow("Türkçe Hizalama", "Kusursuz ✓", "2.1 µs", "0 B/op").
					WithRow("Border Çizimi", "Hizalı ✓", "3.4 µs", "0 B/op").
					WithSelectedStyle(limoni.Fg(limoni.ColorWhite).WithBg(limoni.Hex("#224466"))),
			)
		f.RenderWidget(tableBlock, rightRows[0])

		// Metin Girişi
		inputBlock := limoni.NewBlock().
			WithTitle("✏️ Metin Kutusu (Yazmayı Deneyin)").
			WithBorderStyle(limoni.Fg(limoni.Hex("#FF5599"))).
			WithChild(
				limoni.NewTextInput("demo_input").
					WithState(inputState).
					WithStyle(limoni.Fg(limoni.Hex("#FFFFFF"))).
					WithFocusedStyle(limoni.Fg(limoni.Hex("#FFFF00")).Bold()),
			)
		f.RenderWidget(inputBlock, rightRows[1])

		// Alt Bilgi Paneli
		footer := limoni.NewBlock().
			WithBorderStyle(limoni.Fg(limoni.Hex("#666666"))).
			WithChild(
				limoni.NewParagraph("Çıkış: ESC | Gezinme: ↑/↓ | Yazma: Klavyeyi kullanın | Tıklama: Fare destekli").
					WithStyle(limoni.Fg(limoni.Hex("#888888"))),
			)
		f.RenderWidget(footer, rows[2])

		return true
	})

	if err != nil {
		fmt.Printf("Hata: %v\n", err)
	}
}
