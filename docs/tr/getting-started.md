# 🚀 Limoni Hızlı Başlangıç Rehberi

Limoni, Go dili için sıfır-tahsisat (Zero-Allocation) felsefesiyle tasarlanmış, 60+ FPS yüksek performanslı, 3D grafik ve zengin widget desteğine sahip modern bir Terminal Kullanıcı Arayüzü (TUI) kütüphanesidir.

Tek paket importu (`import "github.com/thebanri/limoni"`) ve akıcı (fluent) API sayesinde hiçbir karmaşık alt paketle uğraşmadan dakikalar içinde modern TUI uygulamaları geliştirebilirsiniz.

---

## 📦 Kurulum

Go 1.22 veya daha güncel bir sürüm gereklidir:

```bash
go get github.com/thebanri/limoni
```

---

## ⚡ 1. İnteraktif Uygulama Başlatma (`limoni.Run`)

Terminalin ham modunu (raw mode), alternatif ekranını (alt-screen), fare takibini ve olay döngüsünü sıfır kurulum zahmetiyle başlatan en pratik yöntemdir:

```go
package main

import (
	"fmt"
	"github.com/thebanri/limoni"
)

func main() {
	count := 0

	limoni.Run(func(f *limoni.Frame, ev *limoni.Event) bool {
		// 1. Olayları Yönet
		if ev != nil && ev.Type == limoni.EventKey {
			switch ev.Key.Ch {
			case 'q', 'Q':
				return false // Uygulamadan çık
			case '+', '=':
				count++
			case '-', '_':
				count--
			case 'r', 'R':
				count = 0
			}
			if ev.Key.Type == limoni.KeyEsc {
				return false
			}
		}

		// 2. Ekranı 3 Satıra Böl (Başlık, Gövde Kartı, Alt Bilgi)
		rows := limoni.SplitVertical(f.Area(), limoni.Fixed(3), limoni.Fill(), limoni.Fixed(3))

		// 3. Başlık
		header := limoni.NewBlock().
			WithTitle(" 🍋 LIMONI SAYAÇ UYGULAMASI ").
			WithTitleAlign(limoni.AlignCenter).
			WithBorderStyle(limoni.Fg(limoni.RGB(255, 215, 0)))
		f.RenderWidget(header, rows[0])

		// 4. Durum Kartı
		color := limoni.RGB(80, 220, 140)
		if count < 0 {
			color = limoni.RGB(255, 80, 80)
		}
		body := limoni.NewBlock().
			Rounded().
			WithTitle(" DURUM ").
			WithPadding(1, 2, 1, 2).
			WithChild(limoni.NewParagraph(fmt.Sprintf("Mevcut Değer: %d", count)).
				WithStyle(limoni.Fg(color).Bold()))
		f.RenderWidget(body, rows[1])

		// 5. Kısayol Çubuğu
		footer := limoni.NewBlock().
			WithTitle(" [+] Artır  [-] Azalt  [R] Sıfırla  [Q/Esc] Çıkış ").
			WithBorderStyle(limoni.Fg(limoni.RGB(100, 110, 130)))
		f.RenderWidget(footer, rows[2])

		return true // Çalışmaya devam et
	})
}
```

Çalıştırmak için:
```bash
go run main.go
```

---

## 🖥️ 2. Tek Satırda Statik Panel / Dashboard Çizimi (`limoni.Start`)

Sadece bir bilgi ekranı veya anlık durum raporu çizdirmek istiyorsanız:

```go
package main

import "github.com/thebanri/limoni"

func main() {
	limoni.Start(func(f *limoni.Frame) {
		cols := limoni.SplitHorizontal(f.Area(), limoni.Percentage(30), limoni.Percentage(70))

		sidebar := limoni.NewBlock().
			WithTitle(" Menü ").
			Rounded().
			WithChild(limoni.NewList("Genel Bakış", "Metrikler", "Ayarlar").
				WithHighlightSymbol("👉 "))
		f.RenderWidget(sidebar, cols[0])

		content := limoni.NewBlock().
			WithTitle(" Sistem Durumu ").
			Rounded().
			WithChild(limoni.NewParagraph("Limoni TUI kütüphanesine hoş geldiniz!").
				WithStyle(limoni.Fg(limoni.Hex("#00FFAA")).Bold()))
		f.RenderWidget(content, cols[1])
	})
}
```

---

## 🔄 3. TEA Mimarisi (`The Elm Architecture`)

Büyük ölçekli, asenkron komut ve alt modeller içeren projeler için tam TEA motoru:

```go
package main

import (
	"context"
	"fmt"

	"github.com/thebanri/limoni"
)

type Model struct {
	count int
}

func (m Model) Init() []limoni.Cmd { return nil }

func (m Model) Update(msg limoni.Msg) limoni.UpdateResult {
	switch msg := msg.(type) {
	case limoni.KeyMsg:
		switch msg.Key.Ch {
		case 'q', 'Q':
			return limoni.UpdateResult{Quit: true}
		case '+':
			m.count++
			return limoni.UpdateResult{Redraw: true}
		case '-':
			m.count--
			return limoni.UpdateResult{Redraw: true}
		}
	}
	return limoni.UpdateResult{}
}

func (m Model) View(f *limoni.Frame) {
	card := limoni.NewBlock().
		WithTitle(" TEA Mimarisi ").
		Rounded().
		WithChild(limoni.NewParagraph(fmt.Sprintf("Sayaç: %d (Çıkış: Q)", m.count)).
			WithStyle(limoni.Fg(limoni.Hex("#00E5FF")).Bold()))
	f.RenderWidget(card, f.Area())
}

func main() {
	program := limoni.NewProgram(
		limoni.WithModel(Model{count: 0}),
		limoni.WithAltScreen(),
		limoni.WithFPS(60),
	)
	_ = program.Run(context.Background())
}
```

---

## 🎨 Renkler ve Stiller

```go
// 24-bit TrueColor ve Hex
gold := limoni.RGB(255, 215, 0)
neon := limoni.Hex("#00FFAA")

// Akıcı Zincirleme
style := limoni.NewStyle().
	WithFg(neon).
	WithBg(limoni.RGB(20, 24, 32)).
	Bold().
	Underline()

// Hızlı Ön Plan Kısayolu
fgOnly := limoni.Fg(gold).Bold()
```

---

## 📐 Hızlı Yerleşim Bölücüler

```go
// Dikey Bölme (Üst, Orta, Alt)
rows := limoni.SplitVertical(area, limoni.Fixed(3), limoni.Fill(), limoni.Fixed(1))

// Yatay Bölme (Sol %25, Sağ %75)
cols := limoni.SplitHorizontal(area, limoni.Percentage(25), limoni.Percentage(75))
```

---

## 📚 Diğer Kaynaklar

- [Çekirdek Motor API'leri (docs/core-api.md)](../core-api.md)
- [Esnek Yerleşim Sistemi (docs/layout-guide.md)](../layout-guide.md)
- [Zengin Widget Kataloğu (docs/widgets-reference.md)](../widgets-reference.md)
- [Örnek Uygulamalar Galerisi (docs/examples.md)](../examples.md)
