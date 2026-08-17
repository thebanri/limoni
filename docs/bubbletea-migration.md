# Bubble Tea → Limoni Geçiş Kılavuzu

`compat/bubbletea` paketi, mevcut Bubble Tea modellerinizi tek satır değişiklikle
Limoni runtime'ı üzerinde çalıştırmanızı sağlar. Bu kılavuz, kademeli geçiş
(adapter ile çalıştır → native API'ye taşı) yolunu anlatır.

## 1. Adım: Adapter ile çalıştırma

Bubble Tea arayüzü aynen korunur:

```go
type Model interface {
	Init() Cmd
	Update(Msg) (Model, Cmd)
	View() string
}
```

Tek değişiklik import ve program oluşturma:

```go
// önce
// import tea "github.com/charmbracelet/bubbletea"
// p := tea.NewProgram(model)
// _, err := p.Run()

// sonra
import tea "github.com/thebanri/limoni/compat/bubbletea"

p := tea.NewProgram(model)
err := p.RunTerminal(context.Background())
```

- `Program.RunTerminal(ctx)` → backend kurulumunu, olay döngüsünü ve kare çizimini
  Limoni runtime'ına devreder (gerçek TTY gerekir).
- `Program.Run(ctx)` → terminal bağlamadan yalnızca Init/Update döngüsünü çalıştırır;
  headless testler için uygundur.

## 2. Adım: Mesaj eşlemesi

Limoni mesajları Bubble Tea muadillerine otomatik çevrilir:

| Limoni (`core/runtime`) | Bubble Tea uyumlu (`compat/bubbletea`) |
| --- | --- |
| `runtime.KeyPressMsg` | `KeyMsg{Type, Runes, Alt, Ctrl, Shift}` |
| `runtime.ResizeMsg` | `WindowSizeMsg{Width, Height}` |
| diğer tüm mesajlar | değiştirilmeden iletilir |

Otomatik çevrilen tuşlar: `KeyRunes`, `KeyEnter`, `KeyBackspace`, `KeyTab`,
`KeyEsc`, `KeyUp`, `KeyDown`, `KeyLeft`, `KeyRight` ve `Ctrl+C`.
`KeyPgUp`, `KeyPgDown`, `KeyHome`, `KeyEnd`, `KeyDelete`, `KeySpace` ve
`KeyCtrlA`…`KeyCtrlZ` sabitleri tanımlıdır ancak şu an otomatik eşlemeye dahil
değildir; bu tuşları `runtime.KeyPressMsg` üzerinden okuyabilirsiniz.
`KeyMsg.String()` Bubble Tea'deki gibi `"ctrl+c"`, `"enter"`, `"up"` üretir.

`Ctrl+C` her zaman `KeyMsg{Type: KeyCtrlC}` olarak gelir; `Quit()` komutu
`QuitMsg` üretir ve adapter bunu `runtime.UpdateResult{Quit: true}`'a çevirir.

## 3. Adım: Lipgloss stilleri

`compat/bubbletea` içindeki `Style`, Lipgloss zincirleme API'sinin bir alt kümesini
sunar:

```go
style := tea.NewStyle().
	Foreground(cell.NewColorRGB(100, 200, 255)).
	Bold(true).
	Padding(1, 2)

out := style.Render("Merhaba")       // ANSI'li string
cs := style.ToCellStyle()            // native cell.Style'a köprü
```

`ToCellStyle()`, geçiş sırasında Lipgloss stillerini native widget'lara
(`Block.BorderStyle`, `Paragraph.Style` vb.) taşımak için köprüdür.

## 4. Adım: Native API'ye taşıma

Adapter `View() string` çıktısını satır satır tampona basar; bu, stil ve
kısmi güncelleme (diff) avantajlarını sınırlar. Kademeli geçiş için modeli
`runtime.Model`'a çevirin:

```go
func (m *model) Init() []runtime.Cmd { return nil }

func (m *model) Update(msg runtime.Msg) runtime.UpdateResult {
	if ev, ok := msg.(runtime.KeyPressMsg); ok {
		if ev.Key.Type == backend.KeyEsc {
			return runtime.UpdateResult{Quit: true}
		}
		return runtime.UpdateResult{Redraw: true}
	}
	return runtime.UpdateResult{}
}

func (m *model) View(f *terminal.Frame) {
	f.RenderWidget(widgets.Block{Title: " App ", Borders: widgets.BorderAll}, f.Buffer.Area)
}
```

Karşılıklar:

| Bubble Tea | Limoni native |
| --- | --- |
| `Init() Cmd` | `Init() []runtime.Cmd` |
| `Update(Msg) (Model, Cmd)` | `Update(runtime.Msg) runtime.UpdateResult` |
| `View() string` | `View(*terminal.Frame)` (doğrudan hücre tamponu) |
| `tea.Batch(a, b)` | `[]runtime.Cmd{a, b}` |
| `tea.Quit` | `runtime.UpdateResult{Quit: true}` |
| string birleştirme ile layout | `layout` paketi + `widgets` bileşenleri |

Yeni proje iskeleti için:

```bash
go run github.com/thebanri/limoni/cmd/limoni@latest new myapp
```

## Bilinen sınırlar

- `View() string` yolunda stil bilgisi taşınmaz; hücreler varsayılan stille yazılır.
  Renk gerekiyorsa native `View(*terminal.Frame)` yoluna geçin.
- `tea.Batch` içindeki komutlar sırayla çalıştırılır ve dönüş mesajları yutulur;
  paralel komut davranışı için `runtime.Cmd` listesi kullanın.
- Fare olayları adapter tarafından `KeyMsg`'e çevrilmez; `runtime.MousePressMsg`
  mesajları modele olduğu gibi iletilir.
