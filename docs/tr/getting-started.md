# 🚀 Limoni Hızlı Başlangıç Rehberi

Bu rehber, Limoni TUI motorunu sıfırdan kurarak ilk terminal uygulamanızı nasıl oluşturacağınızı açıklar.

---

## 📦 Kurulum

Go 1.22 veya daha güncel bir sürüm gereklidir:

```bash
go get github.com/thebanri/limoni
```

---

## 🎨 İki Farklı Mimari Yaklaşımı

Limoni iki farklı programlama yaklaşımını destekler:

### 1. Doğrudan Anlık Mod (Immediate Mode - `terminal.Draw`)
Küçük ve orta ölçekli CLI araçları, paneller ve canlı grafikler için en hızlı ve doğrudan yöntemdir:

```go
package main

import (
	"os"
	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/widgets"
)

func main() {
	b := backend.NewBackend(os.Stdin, os.Stdout)
	b.Setup()
	defer b.Close()

	t, err := terminal.New(b)
	if err != nil {
		panic(err)
	}

	b.StartEventLoop()

	draw := func() {
		t.Draw(func(f *terminal.Frame) {
			f.RenderWidget(widgets.Block{
				Title:         " İLK LİMONİ UYGULAMASI ",
				Borders:       widgets.BorderAll,
				BorderSymbols: widgets.SymbolsRounded,
				BorderStyle:   cell.Style{Fg: cell.NewColorRGB(0, 210, 255)},
			}, f.Buffer.Area)
		})
	}

	draw()

	for ev := range b.Events() {
		switch ev.Type {
		case backend.EventKey:
			if ev.Key.Type == backend.KeyEsc || (ev.Key.Type == backend.KeyRune && ev.Key.Ch == 'q') {
				return
			}
		case backend.EventResize:
			draw()
		}
	}
}
```

---

### 2. TEA Mimarisi (`The Elm Architecture` / Runtime Modu)
Durum (state) geçişlerinin karmaşıklaştığı büyük ölçekli kurumsal uygulamalar için `Init`, `Update`, `View` döngüsü:

```go
package main

import (
	"context"
	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/runtime"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/widgets"
)

type AppModel struct {
	counter int
}

func (m AppModel) Init(ctx context.Context) (runtime.Model, runtime.Cmd) {
	return m, nil
}

func (m AppModel) Update(msg runtime.Msg) (runtime.Model, runtime.Cmd) {
	switch msg := msg.(type) {
	case runtime.KeyMsg:
		if msg.Key.Type == backend.KeyArrowUp {
			m.counter++
		}
	}
	return m, nil
}

func (m AppModel) View(f *terminal.Frame) {
	f.RenderWidget(widgets.Block{
		Title: " TEA SAYAÇ ",
	}, f.Buffer.Area)
}
```
