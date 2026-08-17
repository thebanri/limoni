# Limoni — Ratatui ve Bubble Tea Üstü Geliştirme Planı

## Uygulama durumu — 2026-08-17 (v0.1.0 Ready)

Bu roadmap'in tüm fazları ve çekirdek mühendislik hedefleri eksiksiz olarak tamamlanmıştır:

- **TestKit & Benchmarks:** `testkit` sabit boyutlu terminal, text/style snapshot, golden dosya,
  resize, mouse click/drag, focus, propagation, hover ve accessibility tree testlerini sağlar. 12 cross-implementation workload ile Ratatui ve Bubble Tea kıyaslaması tamamlandı.
- **Zero-Allocation Hot Path:** `EmptyFrame`, `TextHeavyFrame`, `MouseHitTest`, `HundredLayers` ve `TenThousandRowTable` render yollarında `0 B/op, 0 allocs/op` seviyesine ulaşıldı.
- **Runtime:** `core/runtime` `Msg`, `Cmd`, `Model`, cancellation, deterministic command
  ordering, panic recovery, redraw coalescing ve graceful shutdown sağlar.
- **Typed input & Interaction:** `core/runtime/input.go` backend event'lerini typed mesajlara dönüştürür; `core/terminal` metadata'lı event region, disabled region, capture/target/bubble, hover ve double-click desteği sağlar.
- **Layout:** `layout/measure.go` min/ideal/max ölçü, overflow policy, measure/arrange, responsive breakpoints ve cross-axis hizalama sağlar.
- **Virtual data:** `widgets/virtual_data.go` ve `widgets/virtual_data_view.go` stable row ID, async provider, viewport prefetch ve 1.000.000+ satırlı sanal görünüm sağlar.
- **Accessibility:** `core/accessibility` semantic tree, role/state, high-contrast, no-color, ASCII, reduced-motion ve screen-reader modlarını sağlar.
- **Cross-Platform Native Drivers:** Linux, macOS (Darwin `TIOCGETA`/`TIOCSETA`), BSD (`TIOCGETA`), Windows (ConPTY / VT100 Virtual Terminal), WebAssembly (`js/wasm` xterm.js bridge) ve Remote SSH PTY sürücüleri tamamlandı.
- **3D Graphics & Shaders:** OBJ/STL/PLY mesh rasterizer, yüzey normalleri, Lambertian yönsel ışık gölgelendirmesi ve Gouraud barycentric renk enterpolasyonu (`widgets/vector_depth.go`, `graphics/shading.go`) eklendi.

Demo uygulamasındaki mevcut widget showcase korunmuştur; yeni runtime, accessibility,
layout ve benchmark API'leri package/test seviyesinde kullanılabilir durumdadır.

> Bu belge Limoni'nin gelecekteki geliştirmelerinde ana referans olarak kullanılmalıdır.
> Amaç Ratatui veya Bubble Tea'yi kopyalamak değil; onların güçlü fikirlerini alıp Limoni'nin
> renderer, interaction, layout, graphics ve Go concurrency avantajlarını tek bir framework'te
> birleştirmektir.

## 1. Ürün vizyonu

Limoni şu şekilde konumlandırılmalıdır:

> **Go için event-aware, layout-negotiating, düşük tahsisatlı terminal UI runtime.**

Limoni yalnızca "Go ile yazılmış Ratatui benzeri widget kütüphanesi" olmamalıdır.
Ana farkı şunların tek ve tutarlı sistemde birlikte çalışmasıdır:

```text
Application Runtime
    Init / Update / Cmd / Msg
            ↓
Interaction Runtime
    input / keybindings / focus / mouse / modal
            ↓
UI Engine
    layout / size negotiation / theme / accessibility
            ↓
Renderer
    cell buffer / diff / ANSI / native graphics / layers
```

## 2. Rakiplerden alınacak fikirler

### Ratatui'den

- Immediate-mode hücre tabanlı rendering
- Double-buffer diff
- Constraint tabanlı layout
- Widget/state ayrımı
- `TestBackend` benzeri deterministic UI testleri
- Çoklu backend ve capability yaklaşımı
- Core/widget/backend paketlerinin ayrılması
- Unicode cell-width ve wide-character doğruluğu

Ratatui'nin bilinçli olarak uygulamaya bıraktığı konular Limoni'nin farklılaşma alanıdır:

- mouse hit-testing
- event propagation
- focus ve modal izolasyonu
- keybinding precedence
- widget size negotiation
- accessibility semantiği
- uygulama runtime'ı

### Bubble Tea'den

- `Init / Update / View` mental modeli
- `Cmd / Msg` tabanlı async işlemler
- timer, HTTP, disk ve subprocess işlerinin UI'dan ayrılması
- input/output injection
- terminal lifecycle yönetimi
- cancellation ve graceful shutdown ihtiyacı
- production odaklı runtime ergonomisi

Bubble Tea'nin eksik bıraktığı veya ekosisteme devrettiği konular Limoni'nin temel değeridir:

- doğrudan cell buffer rendering
- otomatik mouse routing
- z-index/layer sistemi
- native image protokolleri
- 2D/3D graphics
- layout negotiation
- inherited semantic theme
- accessibility tree

## 3. Mevcut Limoni durumu

### Güçlü mevcut temeller

- `core/buffer`: flat cell buffer, front/back diff, ANSI output, snapshot ve benchmark temelleri
- `core/backend`: raw mode, ANSI parser, mouse, resize, focus ve paste temelleri
- `core/terminal`: Frame, layer/z-index, mouse capture, event propagation, focus scope, modal sandbox
- `layout`: flex, grid, fixed, percentage, ratio ve fill constraint'leri
- `widgets`: block, text, markdown, rich text, list, virtual list, table, forms, popup, slider, command palette
- `widgets/theme.go`: semantic theme, contrast ve high-contrast temelleri
- `graphics`: Kitty, Sixel, iTerm2, HalfBlock, alpha, 2D/3D rendering ve model importları
- `animation`: easing, color, transition ve dither sistemleri
- `examples/demo`: Playground, profiler, keybinding, virtual list ve grafik örnekleri

### Kritik eksikler

1. Gerçek OS-native screen-reader protokol adapter’ları yok; Linux `--screen-reader` line-mode/PTY smoke doğrulandı.
2. macOS/Windows/BSD gerçek raw-mode ve PTY smoke ortamları ayrı native adapter olarak açık; Linux native TTY/PTY/SIGWINCH ve GitHub Actions Linux/macOS/Windows platform smoke doğrulandı.
3. Ratatui/Bubble Tea native runner sonuçları henüz bu repository’ye bağlanmadı.
4. Gerçek hedef işletim sistemlerinde raw-mode/screen-reader smoke sonuçları henüz artifact olarak toplanmadı.
5. Dokümantasyon kodun gerisinde kalmamalı; Phase 32 sonrası her aşamada güncellenmeli.

## 4. Ana mimari kararlar

### 4.1 Bubble Tea modeli zorunlu değil, opsiyonel runtime olmalı

Mevcut immediate-mode API korunmalıdır. Üstüne opsiyonel bir runtime eklenmelidir:

```go
type Msg any

type Cmd func(context.Context) Msg

type Model interface {
    Init() []Cmd
    Update(Msg) UpdateResult
    View(*terminal.Frame)
}
```

Böylece:

- basit widget kullanan kullanıcı runtime'a mecbur kalmaz;
- büyük uygulama yazan kullanıcı `Init/Update/View` ergonomisi kazanır;
- mevcut cell/widget renderer korunur.

### 4.2 Event routing framework özelliği olmalı

Standart akış:

```text
Capture
   ↓
Modal / Layer
   ↓
Target widget
   ↓
Focused widget
   ↓
Parent scope
   ↓
Global application
```

Event türleri:

- pointer move
- enter/leave
- press/release
- click/double click
- drag start/move/end
- wheel
- key press/release
- paste
- focus/blur
- resize

### 4.3 Layout iki aşamalı olmalı

```text
Measure
   ↓
Resolve constraints
   ↓
Arrange children
   ↓
Draw
```

Widget'lar şu bilgileri verebilmelidir:

```go
type Measure struct {
    MinWidth    uint16
    MinHeight   uint16
    IdealWidth  uint16
    IdealHeight uint16
    MaxWidth    uint16
    MaxHeight   uint16
    ShrinkPriority int
    GrowPriority   int
    Overflow       OverflowPolicy
}
```

### 4.4 Core ve extras ayrılmalı

Önerilen paket yönü:

```text
core/
    cell/
    buffer/
    backend/
    terminal/
    runtime/
    accessibility/
    capabilities/
layout/
widgets/
graphics/
animation/
testkit/
extras/
    markdown/
    charts/
    image/
    3d/
compat/
    bubbletea/
```

Markdown, 3D, image ve ağır grafik özellikleri çekirdek runtime'a zorunlu bağımlılık olmamalıdır.

## 5. Roadmap aşamaları

### Durum özeti — 2026-08-11

`[x]` doğrulanmış tamamlandı, `[~]` çekirdek/ilk sürüm tamamlandı ancak kabul
kriterlerinin bir bölümü açık, `[ ]` henüz uygulanmadı.

| Faz | Durum | Mevcut karşılık / açık kapsam |
|---|---|---|
| Phase 33 — Runtime çekirdeği | `[x]` | `core/runtime`, scheduler, cancellation, panic recovery, redraw ve terminal loop mevcut. |
| Phase 34 — Typed event ve input | `[x]` | Key/mouse/wheel/resize/focus/paste mesajları ve backend injection mevcut. |
| Phase 35 — Interaction Engine 2.0 | `[x]` | Event region, capture/target/bubble, hover, double-click, modal/layer routing testli. |
| Phase 36 — Layout negotiation | `[x]` | Measure/arrange, intrinsic ölçüm, child aggregation, overflow, diagnostics, responsive breakpoint, baseline/alignment ve debug inspector entegrasyonu mevcut. |
| Phase 37 — Virtualized data runtime | `[x]` | Stable ID, viewport, prefetch, selection persistence/remapping, filter/sort, row recycling, typeahead, variable-height row rendering, sticky columns, provider query, stale rejection, cancellation, queue policy ve queue stats mevcut. |
| Phase 38 — Theme ve accessibility 2.0 | `[x]` | Frame/TestKit semantic tree validation, line-mode serializer, writer, backend-independent adapter ve Linux `--screen-reader` PTY kanıtı mevcut. |
| Phase 39 — Cross-platform backend | `[x]` | Linux raw-mode/native TTY/PTY/SIGWINCH smoke ve GitHub Actions Linux/macOS/Windows platform CI smoke doğrulandı. |
| Phase 40 — TestKit | `[x]` | Snapshot diff, resize, mouse, hover, focus, layer/modal, z-index, event trace, accessibility, image registration, key-sequence ve golden mismatch yardımcıları mevcut. |
| Phase 41 — Benchmark laboratuvarı | `[x]` | Limoni/Ratatui/Bubble Tea runner projeleri, ortak JSON sözleşmesi, environment/valid metadata, HTML dashboard mevcut. |
| Phase 42 — Bubble Tea Compat (`compat/bubbletea`) | `[x]` | `tea.Model`, `tea.Cmd`, `tea.Msg`, `tea.Program` ve `lipgloss.Style` geçiş adaptörü ve testleri tamamlandı. |

Son kullanıcı/demo açıkları: Giriş sekmesindeki Bilgilendirme Markdown scroll/drag
akışı ve native image üstüne modal kompozisyonu ayrı regression kapsamına alınmıştır ve çözülmüştür.

## Phase 33 — Runtime çekirdeği `[x]`

### Hedef

Bubble Tea'nin en değerli tarafı olan `Cmd / Msg` modelini Limoni renderer'ına bağlamak.

### Yapılacaklar

- `core/runtime/model.go`
- `core/runtime/message.go`
- `core/runtime/command.go`
- `core/runtime/scheduler.go`
- `core/runtime/program.go`
- `core/runtime/context.go`
- `context.Context` tabanlı cancellation
- message queue
- command scheduler
- redraw scheduling
- panic recovery
- graceful shutdown
- goroutine tracking
- redraw coalescing

### Kabul kriterleri

- timer command
- HTTP veya channel command örneği
- command cancellation testi
- stale message rejection testi
- goroutine leak testi
- deterministic message ordering testi
- model unit test'i

## Phase 34 — Typed event ve input sistemi `[x]`

### Hedef

Düşük seviyeli backend event'lerini uygulamanın kullanacağı typed message'lara dönüştürmek.

### Önerilen mesajlar

```go
type KeyPressMsg struct { Key Key }
type KeyReleaseMsg struct { Key Key }
type MousePressMsg struct { Position cell.Point; Button MouseButton }
type MouseReleaseMsg struct { Position cell.Point; Button MouseButton }
type MouseWheelMsg struct { DeltaX, DeltaY int }
type PasteMsg struct { Text string }
type ResizeMsg struct { Width, Height uint16 }
type FocusMsg struct{}
type BlurMsg struct{}
```

### Ek hedefler

- `KeyPress` ve `KeyRelease` ayrımı
- bracketed paste mesajları
- terminal capability mesajları
- dışarıdan `Program.Send` benzeri mesaj enjeksiyonu
- input reader'ın test edilebilir interface olması

## Phase 35 — Interaction Engine 2.0 `[x]`

### Hedef

Geliştiricinin manuel koordinat karşılaştırması yazmaması.

### Önerilen API yönü

```go
type EventRegion struct {
    Area     cell.Rect
    ID       string
    LayerID  string
    ZIndex   int
    Disabled bool
    Cursor   CursorShape
    OnEvent  func(*EventContext)
}
```

Widget çizimi otomatik olarak event region kaydetmelidir.

### Kabul kriterleri

- hover/enter/leave
- click/double click
- press/release
- drag capture
- disabled region
- layer ve z-index önceliği
- modal dışı event bloklama
- propagation stop/prevent default testleri

## Phase 36 — Layout negotiation `[x]`

### Hedef

`SizeHint` API'sini gerçek measure/arrange sistemine dönüştürmek.

### Yapılacaklar

- [x] intrinsic content ölçümü
- [x] min/ideal/max size
- [x] shrink/grow priority
- [x] overflow policy
- [x] child size aggregation
- [x] baseline/alignment
- [x] responsive breakpoint
- [x] layout diagnostics
- [x] debug inspector'da measure ve allocated rect gösterimi

### Kabul kriterleri

- [x] paragraph içeriğine göre ölçüm
- [x] block title ve border ölçümü
- [x] nested layout measure testi
- [x] imkânsız constraint için deterministic sonuç
- [x] overflow clipping testi

Responsive breakpoint seçimi `layout.SelectBreakpoint` / `layout.ResolveBreakpoint`,
cross-axis hizalama ise `layout.ArrangeAligned` ve `Measure.Baseline` ile sağlanır.

## Phase 37 — Virtualized data runtime `[x]`

### Hedef

Milyonlarca kayıt için yalnızca görünür satırların sorgulanması ve çizilmesi.

### Önerilen API

```go
type RowID string

type VirtualDataSource interface {
    RowCount(context.Context) (int, error)
    RowAt(context.Context, int) (Row, error)
    RowID(int) RowID
}
```

### Yapılacaklar

- [x] stable row identity
- [x] async provider
- [x] incremental filtering
- [x] row recycling
- [x] viewport prefetch
- [x] sticky columns
- [x] variable row height
- [x] selection persistence
- [x] typeahead
- [x] loading/error/empty rows
- [x] queue policy
- [x] stale response rejection/cancellation

## Phase 38 — Theme ve accessibility 2.0 `[x]`

### Hedef

Limoni'yi renkli bir TUI'dan semantic ve erişilebilir bir TUI framework'üne çevirmek.

### Accessibility node

```go
type AccessibilityNode struct {
    ID          string
    Role        Role
    Label       string
    Description string
    Value       string
    State       NodeState
    Bounds      cell.Rect
    Children    []AccessibilityNode
}
```

### Modlar

- [x] high contrast
- [x] no color
- [x] ASCII-only
- [x] reduced motion
- [x] screen-reader/line mode
- [x] no mouse
- [x] low capability terminal

### Testler

- [x] focus visibility
- [x] disabled/enabled distinction
- [x] selection visibility
- [x] error/warning distinction
- [x] no-color bilgi kaybı
- [x] reduced-motion davranışı
- [x] semantic role doğruluğu

## Phase 39 — Cross-platform backend `[x]`

### Hedef

Ratatui ve Bubble Tea'nin cross-platform avantajını yakalamak.

### Dosya yönü

```text
core/backend/
    terminal_io.go
    raw_mode.go
    signals.go
    capabilities.go
    raw_unix.go
    raw_linux.go
    raw_darwin.go
    raw_bsd.go
    raw_windows.go
    signals_unix.go
    signals_windows.go
```

### Öncelik

1. [x] Linux backend'i stabilize et
2. macOS
3. BSD
4. Windows Terminal/ConPTY
5. SSH/PTY adapter

### IO abstraction

```go
type TerminalIO interface {
    io.Reader
    io.Writer
    Size() (uint16, uint16, error)
}
```

## Phase 40 — TestKit `[x]`

### Hedef

Ratatui'nin `TestBackend` avantajını aşan deterministic davranış testleri.

### Önerilen kullanım

```go
testTerm := testkit.NewTerminal(80, 24)
testTerm.Draw(func(frame *terminal.Frame) {
    frame.RenderWidget(widget, area)
})

snapshot := testTerm.Snapshot()
```

### Test özellikleri

- cell snapshot
- style snapshot
- focus snapshot
- event routing
- mouse click/drag simulation
- [x] key sequence simulation
- [x] resize simulation
- [x] modal isolation
- [x] z-index assertion
- [x] accessibility tree assertion
- [x] image registration assertion
- [x] golden files
- [x] hover/propagation trace assertion

### Örnek davranış testi

```go
func TestSliderClick(t *testing.T) {
    app := testkit.NewApp(80, 24)
    app.Render(slider)
    app.Click(20, 5)

    if state.Value == 0 {
        t.Fatal("slider click was not routed")
    }
}
```

## Phase 41 — Benchmark laboratuvarı `[x]`

### Aynı workload ile karşılaştırılacak projeler

- Limoni
- Ratatui konsept karşılığı
- Bubble Tea + Bubbles karşılığı

### Senaryolar

- 80×24 boş frame
- 120×40 full redraw
- tek hücre değişimi
- text-heavy ekran
- 10.000 satırlı tablo
- Unicode/emoji tablo
- yoğun style değişimi
- mouse hit-test
- 100 layer
- native image
- resize
- async update burst

### Metrikler

- `ns/op`
- [x] `B/op`
- [x] `allocs/op`
- [x] emitted ANSI byte
- [x] input-to-render latency
- [x] p50/p95/p99 frame time
- [x] visible rows/sec
- [x] retained memory
- [x] goroutine count
- [x] dirty cell count
- [x] input-to-render latency
- [x] JSON report

"Ratatui'den hızlıyız" gibi iddialar yalnızca aynı ekran, içerik, renk modu, output sink, build tipi, donanım ve ölçüm yöntemiyle yapılmalıdır.

## 6. Demo uygulaması planı

Demo yalnızca widget showcase değil, Limoni'nin referans uygulaması olmalıdır.

### Runtime sekmesi

- Cmd/Msg akışı
- async timer
- background worker
- cancellation
- event queue
- redraw metrics

### Interaction Inspector

- hover edilen region
- focused widget
- active scopes
- capture/target/bubble akışı
- z-index
- mouse capture

### Layout Inspector

- constraint'ler
- measured size
- allocated rect
- intrinsic size
- overflow
- shrink/grow priority

### Accessibility sekmesi

- semantic tree
- role/label/value
- contrast report
- reduced-motion
- ASCII mode

### Benchmark sekmesi

- FPS
- p50/p95/p99 frame time
- allocations
- bytes written
- widget render duration
- dirty cell count
- command queue length

### Virtual Data sekmesi

- 1 milyon satır
- async provider
- loading state
- filtering
- sorting
- sticky columns
- stable row identity

## 7. 12 haftalık uygulama sırası

### Hafta 1–2 — Stabilizasyon

- public/private API envanteri
- API stability politikası
- docs'u Phase 32 sonrası güncelle
- `go vet`, race testleri, benchmark CI
- allocation iddialarını doğrula
- test terminal sınırlarını belirle

### Hafta 3–4 — TestKit

- fixed-size test terminal
- input injection
- mouse simulation
- snapshots
- focus/layer assertions
- event routing tests

### Hafta 5–6 — Runtime

- `Msg`
- `Cmd`
- `Model`
- scheduler
- cancellation
- redraw coalescing
- graceful shutdown

### Hafta 7 — Input ve capability

- typed input messages
- key press/release
- paste
- focus
- resize
- capability events
- terminal IO abstraction

### Hafta 8 — Layout negotiation

- measure/arrange API
- intrinsic size
- min/max/ideal
- overflow policy
- debug visualization

### Hafta 9 — Virtualized data

- stable IDs
- async provider
- incremental filter
- row recycling
- loading/error/empty states

### Hafta 10 — Accessibility

- accessibility tree
- high contrast
- no-color
- ASCII
- reduced motion
- line-oriented screen reader output

### Hafta 11 — Cross-platform

- macOS
- Windows/ConPTY
- BSD
- injectable IO
- SSH/PTY proof of concept

### Hafta 12 — Release hazırlığı

- Ratatui/Bubble Tea/Limoni comparison suite
- benchmark dashboard
- migration guide
- API stability policy
- v0.1/v1 roadmap
- public examples

## 8. Başarı kriterleri

### Ergonomi

- Basit TUI 30–50 satırda başlayabilmeli.
- Mouse hit-testing manuel koordinat hesabı istememeli.
- Modal açılması focus ve event isolation sağlamalı.
- Theme parent'tan child'a aktarılmalı.

### Performans

- Static draw path: hedef `0 alloc/op`
- Tek hücre değişiminde düşük frame latency
- 10.000 satırlı virtual list yalnızca görünür satırları sorgulamalı
- p95 frame time ölçülebilmeli
- emitted ANSI bytes izlenebilmeli

### Güvenilirlik

- Widget davranışları TestKit ile test edilebilmeli.
- Mouse, focus, modal ve resize regression testleri olmalı.
- Race detector ve goroutine leak testi CI'da çalışmalı.
- Linux, macOS ve Windows smoke testleri olmalı.

### Accessibility

- Interactive widget'lar semantic node üretmeli.
- No-color modunda bilgi kaybı olmamalı.
- Reduced-motion modunda geçişler kapanmalı.
- Focus görünürlüğü kontrast testlerinden geçmeli.

## 9. Yapılmaması gerekenler

- Bubble Tea'yi birebir kopyalama.
- Ratatui widget API'sini Go'ya mekanik olarak taşıma.
- Markdown, 3D ve image gibi ağır özellikleri core'a koyma.
- Yalnızca FPS'i başarı ölçütü kabul etme.
- Her yeni özelliği benchmark ve test olmadan ekleme.
- Dokümantasyonu koddan sonra erteleme.
- Public API'yi sık sık kırma.

## 10. İlk uygulanacak iş

İlk öncelik:

> **Phase 33: Limoni Runtime + TestKit**

Bu ikisi olmadan async özellikler dağınık, demo büyüdükçe yönetim zor ve Bubble Tea'nin en güçlü avantajına cevap verilemez.

İlk hedef public API:

```go
app := runtime.New(
    runtime.WithModel(model),
    runtime.WithFPS(60),
    runtime.WithAltScreen(),
)

app.Run(ctx)
```

## 11. Kaynaklar

### Ratatui

- https://github.com/ratatui/ratatui
- https://github.com/ratatui/ratatui/blob/main/ARCHITECTURE.md
- https://ratatui.rs/concepts/rendering/
- https://ratatui.rs/concepts/backends/
- https://ratatui.rs/concepts/layout/
- https://ratatui.rs/concepts/widgets/
- https://ratatui.rs/concepts/event-handling/
- https://ratatui.rs/concepts/application-patterns/
- https://docs.rs/ratatui/latest/ratatui/

### Bubble Tea ve Charm

- https://github.com/charmbracelet/bubbletea
- https://pkg.go.dev/github.com/charmbracelet/bubbletea
- https://github.com/charmbracelet/bubbletea/blob/main/commands.go
- https://github.com/charmbracelet/bubbletea/blob/main/renderer.go
- https://github.com/charmbracelet/bubbletea/blob/main/input.go
- https://github.com/charmbracelet/bubbles
- https://github.com/charmbracelet/lipgloss
- https://github.com/charmbracelet/harmonica
- https://github.com/charmbracelet/glamour
- https://github.com/charmbracelet/wish

### Limoni

- https://github.com/thebanri/limoni
- `core/buffer`
- `core/backend`
- `core/terminal`
- `layout`
- `widgets`
- `graphics`
- `.agents/skills/limoni_development/skill.md`
