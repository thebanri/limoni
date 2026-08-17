# 🚀 Hızlı Başlangıç (Getting Started)

Limoni, Go dili için sıfır-tahsisat (Zero-Allocation) felsefesiyle tasarlanmış, 60+ FPS yüksek performanslı, 3D grafik ve zengin widget desteğine sahip modern bir Terminal Kullanıcı Arayüzü (TUI) kütüphanesidir.

Bu kılavuzda 5 dakika içinde ilk interaktif Limoni uygulamanızı nasıl kurup çalıştıracağınızı öğreneceksiniz.

---

## 📦 Kurulum

Go 1.22+ yüklü projenizde Limoni'yi bağımlılık olarak ekleyin:

```bash
go get github.com/thebanri/limoni
```

---

## ⚡ 5 Dakikada İlk TUI Uygulaması: Sayaç (Counter)

Limoni, **The Elm Architecture (TEA)** tasarım kalıbını birinci sınıf bir mimari olarak benimser. Her uygulama 3 ana bileşenden oluşur:
1. **Model**: Uygulamanın anlık durumunu (State) tutan veri yapısı.
2. **Update**: Gelen klavye, fare veya sistem olaylarına (Message) göre durumu güncelleyen saf fonksiyon.
3. **View**: Durumu ekrana çizen görsel fonksiyon.

### `main.go`

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/runtime"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/layout"
	"github.com/thebanri/limoni/widgets"
)

// 1. Model: Durum Yapısı
type CounterModel struct {
	Count int
}

func (m CounterModel) Init() runtime.Cmd {
	return nil
}

// 2. Update: Olay Yönetimi
func (m CounterModel) Update(msg runtime.Msg) (runtime.Model, runtime.Cmd) {
	switch msg := msg.(type) {
	case runtime.KeyPressMsg:
		switch msg.Key.Type {
		case backend.KeyEsc:
			return m, runtime.Quit
		case backend.KeyRune:
			switch msg.Key.Ch {
			case 'q', 'Q':
				return m, runtime.Quit
			case '+', '=':
				m.Count++
			case '-', '_':
				m.Count--
			case 'r', 'R':
				m.Count = 0
			}
		}
	}
	return m, nil
}

// 3. View: Ekran Çizimi
func (m CounterModel) View(frame *terminal.Frame) {
	area := frame.Area()

	// Esnek Yerleşim: Üst Başlık, Orta Sayaç Kartı, Alt Kısayollar
	chunks := layout.FlexLayout{
		Direction: layout.Vertical,
		Constraints: []layout.Constraint{
			layout.Fixed(3), // Başlık
			layout.Fill(),   // Gövde
			layout.Fixed(3), // Kısayol Çubuğu
		},
	}.Split(area)

	// Başlık
	frame.RenderWidget(widgets.Block{
		Title:          " 🍋 LIMONI SAYAC UYGULAMASI ",
		TitleAlignment: widgets.AlignCenter,
		BorderStyle:    cell.Style{Fg: cell.NewColorRGB(255, 215, 0)},
		TitleStyle:     cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Modifier: cell.ModifierBold},
	}, chunks[0])

	// Orta Kart
	countText := fmt.Sprintf("Mevcut Değer: %d", m.Count)
	countStyle := cell.Style{Fg: cell.NewColorRGB(0, 255, 200), Modifier: cell.ModifierBold}
	if m.Count < 0 {
		countStyle.Fg = cell.NewColorRGB(255, 80, 80)
	}

	bodyBlock := widgets.Block{
		Title:          " DURUM ",
		TitleAlignment: widgets.AlignLeft,
		BorderStyle:    cell.Style{Fg: cell.NewColorRGB(0, 180, 255)},
	}
	frame.RenderWidget(bodyBlock, chunks[1])
	frame.RenderWidget(&widgets.Paragraph{
		Text:  countText,
		Style: countStyle,
	}, bodyBlock.Inner(chunks[1]))

	// Alt Çubuk
	frame.RenderWidget(widgets.Block{
		Title:          " [+] Artır  [-] Azalt  [R] Sıfırla  [Q/Esc] Çıkış ",
		TitleAlignment: widgets.AlignLeft,
		BorderStyle:    cell.Style{Fg: cell.NewColorRGB(100, 110, 120)},
	}, chunks[2])
}

func main() {
	// Terminal arayüzünü hazırla
	b := backend.NewBackend(os.Stdin, os.Stdout)
	if err := b.Setup(); err != nil {
		fmt.Fprintf(os.Stderr, "Terminal başlatılamadı: %v\n", err)
		os.Exit(1)
	}
	defer b.Close()

	term, err := terminal.New(b)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Terminal motoru oluşturulamadı: %v\n", err)
		os.Exit(1)
	}

	// TEA Programını başlat
	p := runtime.New(
		runtime.WithModel(CounterModel{Count: 0}),
		runtime.WithFPS(60),
	)

	if err := p.RunTerminal(context.Background(), term, b); err != nil {
		fmt.Fprintf(os.Stderr, "Uygulama hatası: %v\n", err)
		os.Exit(1)
	}
}
```

---

## 🏃‍♂️ Çalıştırma

Terminalinizde aşağıdaki komutu verin:

```bash
go run main.go
```

`+`, `-`, `R` ve `Q` tuşlarıyla anında tepki veren, 60 FPS hızında pürüzsüz bir TUI deneyimi elde edersiniz!

---

## 🛠️ Sıradaki Adımlar

- [Mimari ve Sıfır-Tahsisat Felsefesi (docs/architecture.md)](./architecture.md)
- [Çekirdek Motor API'leri (docs/core-api.md)](./core-api.md)
- [Esnek Yerleşim Sistemi (docs/layout-guide.md)](./layout-guide.md)
- [Zengin Widget Kataloğu (docs/widgets-reference.md)](./widgets-reference.md)
- [Örnek Uygulamalar Galerisi (docs/examples.md)](./examples.md)
