// ==========================================================================
// Limoni Documentation Engine (Vanilla JS)
// ==========================================================================

const DOCS_DATA = {
    quickstart: {
        title: "⚡ Hızlı Başlangıç (Quick Start)",
        content: `
            <h1>🚀 Hızlı Başlangıç</h1>
            <p>Limoni, Go dili için <strong>sıfır-tahsisatlı (Zero-Allocation)</strong>, <strong>60+ FPS</strong> yüksek performanslı, 3D grafik ve yerel resim destekli modern bir Terminal Kullanıcı Arayüzü (TUI) kütüphanesidir.</p>

            <div class="cards-grid">
                <div class="feature-card">
                    <div class="card-icon">⚡</div>
                    <div class="card-title">0 B/op Sıfır Tahsisat</div>
                    <p class="card-desc">Render sıcak yolunda GC yükü yoktur, 12 ns boş kare çizimiyle mikro gecikme oluşmaz.</p>
                </div>
                <div class="feature-card">
                    <div class="card-icon">📦</div>
                    <div class="card-title">Tek Paket & Akıcı API</div>
                    <p class="card-desc">Karmaşık alt paketler yerine tek <code>limoni</code> importu ve zincirlenebilir yapıcılar.</p>
                </div>
                <div class="feature-card">
                    <div class="card-icon">🧊</div>
                    <div class="card-title">3D & Resim Motoru</div>
                    <p class="card-desc">Yazılımsal 3D rasterizer (STL/OBJ/PLY) ve yerel Kitty/Sixel/iTerm2 resim desteği.</p>
                </div>
                <div class="feature-card">
                    <div class="card-icon">📐</div>
                    <div class="card-title">Flexbox & CSS Grid</div>
                    <p class="card-desc">CSS standartlarında esnek yerleşimler, hızlı bölücüler ve otomatik boyut pazarlığı.</p>
                </div>
            </div>

            <h2>📦 Kurulum</h2>
            <p>Go 1.22+ yüklü projenizde Limoni'yi bağımlılık olarak ekleyin:</p>
            <div class="code-box">
                <div class="code-header"><span>BASH</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <pre><code>go get github.com/thebanri/limoni</code></pre>
            </div>

            <h2>⚡ 1 Dakikada İnteraktif TUI Uygulaması (limoni.Run)</h2>
            <p>Aşağıdaki kodu <code>main.go</code> dosyasına kaydedip <code>go run main.go</code> ile hemen çalıştırabilirsiniz. Ham mod (raw mode), alternatif ekran, fare desteği ve olay döngüsü otomatik olarak kurulur:</p>

            <div class="code-box">
                <div class="code-header"><span>MAIN.GO</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <pre><code>package main

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
                return false // Çıkış yap
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

        // 2. Ekranı 3 Satıra Böl (Başlık, Gövde, Alt Bilgi)
        rows := limoni.SplitVertical(f.Area(), limoni.Fixed(3), limoni.Fill(), limoni.Fixed(3))

        // 3. Başlık
        header := limoni.NewBlock().
            WithTitle(" 🍋 LIMONI SAYAC UYGULAMASI ").
            WithTitleAlign(limoni.AlignCenter).
            WithBorderStyle(limoni.Fg(limoni.RGB(255, 215, 0)))
        f.RenderWidget(header, rows[0])

        // 4. Gövde Kartı
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

        // 5. Kısayollar
        footer := limoni.NewBlock().
            WithTitle(" [+] Artır  [-] Azalt  [R] Sıfırla  [Q/Esc] Çıkış ").
            WithBorderStyle(limoni.Fg(limoni.RGB(100, 110, 130)))
        f.RenderWidget(footer, rows[2])

        return true // Çalışmaya devam et
    })
}</code></pre>
            </div>

            <h2>🖥️ Tek Satırda Statik Panel (limoni.Start)</h2>
            <div class="code-box">
                <div class="code-header"><span>GOLANG</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <pre><code>limoni.Start(func(f *limoni.Frame) {
    cols := limoni.SplitHorizontal(f.Area(), limoni.Percentage(30), limoni.Percentage(70))
    f.RenderWidget(limoni.NewBlock().Rounded().WithTitle("Menü"), cols[0])
    f.RenderWidget(limoni.NewBlock().Rounded().WithTitle("İçerik"), cols[1])
})</code></pre>
            </div>
        `
    },

    architecture: {
        title: "🏛️ Mimari ve Sıfır-Tahsisat (Architecture)",
        content: `
            <h1>🏛️ Mimari ve Sıfır-Tahsisat Felsefesi</h1>
            <p>Limoni, bellek dostu mimarisi, CPU L1/L2 önbellek verimliliği ve Unicode doğruluğu sayesinde mikro gecikmeleri ve görsel bozulmaları ortadan kaldırır.</p>

            <h2>1. 1D Düz Bellek Izgarası ([]cell.Cell)</h2>
            <p>Geleneksel matrisler (<code>[][]Cell</code>) her satır için ayrı heap tahsisatı ve pointer indirection oluşturarak CPU önbellek ıskalamalarına (cache miss) sebep olur. Limoni, tüm terminal ekranını tek bir ardışık 1D bellek diliminde tutar:</p>
            <div class="code-box">
                <div class="code-header"><span>GOLANG</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <pre><code>// Önbellek dostu hücre erişimi: y * Width + x
cell := buf.Content[y*buf.Area.Width + x]</code></pre>
            </div>

            <h2>2. Çift Tamponlu ANSI Diff Motoru</h2>
            <p>Front Buffer ve Back Buffer arasındaki farklar taranarak yalnızca değişen hücreler minimum ANSI kaçış dizisiyle terminale yazılır. <code>\\x1b[?2026h</code> senkronize güncelleme protokolü ile ekran titremesi (flicker) tamamen engellenir.</p>

            <h2>3. Unicode Doğu Asya Genişliği ve Emoji Güvenliği</h2>
            <p>Doğu Asya Genişlik standardı (East Asian Width W/F) uyarınca 2 genişlikli emojiler (🔴, 🚀, ☕) ve 1 genişlikli semboller (✓, ⚠) kesin doğrulukla işlenir. Geniş karakter devam hücreleri (continuation cells) modal pencereler sürüklendiğinde otomatik olarak algılanır ve kenarlık parçalanması (border tearing) tamamen önlenir.</p>
        `
    },

    benchmarks: {
        title: "📊 Performans ve Benchmark Raporu",
        content: `
            <h1>📊 Performans ve Benchmark Raporu</h1>
            <p>Limoni'nin render sıcak yolundaki sıfır tahsisat başarımı Go microbenchmark testleriyle belgelenmiştir:</p>

            <div class="cards-grid">
                <div class="feature-card">
                    <div class="card-title">Boş Çerçeve (Empty Frame)</div>
                    <p class="card-desc"><strong>12.06 ns/op</strong> — 0 B/op (0 allocs/op)</p>
                </div>
                <div class="feature-card">
                    <div class="card-title">Fare İsabet Testi (Hit Test)</div>
                    <p class="card-desc"><strong>63.50 ns/op</strong> — 0 B/op (0 allocs/op)</p>
                </div>
                <div class="feature-card">
                    <div class="card-title">Metin Ağırlıklı Çerçeve</div>
                    <p class="card-desc"><strong>4.8 µs/op</strong> — 0 B/op (0 allocs/op)</p>
                </div>
                <div class="feature-card">
                    <div class="card-title">1.000.000 Satırlı Sanal Tablo</div>
                    <p class="card-desc"><strong>38.4 µs/op</strong> — 0 B/op (0 allocs/op)</p>
                </div>
            </div>
        `
    },

    "core-cell": {
        title: "🟩 core/cell — Hücre & Stil API",
        content: `
            <h1>🟩 core/cell Paketi</h1>
            <p>Terminal ekranındaki en küçük birim olan karakter hücresi, 24-bit TrueColor RGB renkleri, stil modifikatörleri ve geometrik sınırlayıcıları (Rect) tanımlar.</p>

            <h2>Renk Tanımlama</h2>
            <div class="code-box">
                <div class="code-header"><span>GOLANG</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <pre><code>// 24-bit TrueColor RGB
colRGB := limoni.RGB(255, 180, 0)

// Hex Renk Desteği
colHex := limoni.Hex("#00FFAA")

// 8-bit Standart ANSI (0-255)
colANSI := limoni.ANSI(196)</code></pre>
            </div>

            <h2>Akıcı Stil Zincirleme</h2>
            <div class="code-box">
                <div class="code-header"><span>GOLANG</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <pre><code>style := limoni.NewStyle().
    WithFg(limoni.Hex("#00E5FF")).
    WithBg(limoni.RGB(20, 25, 35)).
    Bold().
    Underline()</code></pre>
            </div>
        `
    },

    "core-buffer": {
        title: "🟨 core/buffer — 1D Tampon & Diff API",
        content: `
            <h1>🟨 core/buffer Paketi</h1>
            <p>Terminal ızgarasını 1D ardışık dizide saklar, güvenli hücre yazımı (SetCellDirect) ve diferansiyel ekran güncellemesini yürütür.</p>

            <h2>Kullanım</h2>
            <div class="code-box">
                <div class="code-header"><span>GOLANG</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <pre><code>buf := buffer.NewBuffer(area)

// Güvenli hücre yazımı (Orphan continuation hücrelerini otomatik temizler)
buf.SetCellDirect(x, y, cell.Cell{Content: '█', Style: style})

// Çift genişlikli karakter ve devam hücresi korumalı metin yazımı
buf.SetString(x, y, "🚀 Limoni TUI", style)</code></pre>
            </div>
        `
    },

    "core-terminal": {
        title: "🟦 core/terminal — Terminal, Katman & Modal",
        content: `
            <h1>🟦 core/terminal Paketi</h1>
            <p>Çift tampon yönetimini, 60+ FPS çizim döngüsünü, Z-Index katmanlarını ve klavye odağını içine hapseden modal pencereleri yönetir.</p>

            <h2>Modal ve Odak Kapsamı</h2>
            <div class="code-box">
                <div class="code-header"><span>GOLANG</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <pre><code>// Modalı kaydet: Altındaki widget'ların olay almasını engeller
f.RegisterModal("exit_dialog", modalArea, onDismiss)

// Klavye odağını (Tab / Shift+Tab) yalnızca modal içine hapsedin
f.BeginFocusScope("exit_dialog")
f.RenderWidget(dialog, modalArea)</code></pre>
            </div>
        `
    },

    "core-runtime": {
        title: "🔄 core/runtime — The Elm Architecture (TEA)",
        content: `
            <h1>🔄 core/runtime Paketi</h1>
            <p>Limoni'nin kurumsal düzeydeki durum yönetim motorudur. Model, Init, Update ve View döngüsüyle deterministik TUI uygulamaları kurmanızı sağlar.</p>

            <h2>Eşzamanlılık ve Güvenlik Garantileri</h2>
            <ul>
                <li><strong>Deterministik Komut Sıralaması</strong>: Asenkron çalışan komutların sonuçları modele gönderiliş sırasıyla teslim edilir.</li>
                <li><strong>İptal Önceliği</strong>: Program durdurulduğunda veya bağlam iptal edildiğinde gecikmiş mesajlar anında elenir, veri yarışları engellenir.</li>
                <li><strong>Panik Yakalama</strong>: Kullanıcı komutlarında veya modellerinde oluşan panikler yakalanarak uygulamanın çökmesi engellenir.</li>
            </ul>
        `
    },

    layout: {
        title: "📐 layout — Flexbox & CSS Grid Yerleşim Motoru",
        content: `
            <h1>📐 layout Paketi</h1>
            <p>Ekran alanını yatay veya dikey olarak CSS Flexbox ve CSS Grid standartlarında böler.</p>

            <h2>Hızlı Bölücüler</h2>
            <div class="code-box">
                <div class="code-header"><span>GOLANG</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <pre><code>// Dikey Bölme (Başlık 3, Gövde Fill, Alt Bilgi 1)
rows := limoni.SplitVertical(f.Area(), limoni.Fixed(3), limoni.Fill(), limoni.Fixed(1))

// Yatay Bölme (Sol %30, Sağ %70)
cols := limoni.SplitHorizontal(rows[1], limoni.Percentage(30), limoni.Percentage(70))</code></pre>
            </div>

            <h2>Kısıtlama Türleri</h2>
            <ul>
                <li><code>Fixed(N)</code>: Sabit N hücre.</li>
                <li><code>Percentage(P)</code>: Toplam alanın %P'si (0-100).</li>
                <li><code>Ratio(R)</code>: Kalan serbest alanı ağırlıklı oranlarla paylaştırır (Ratio(2) ve Ratio(1)).</li>
                <li><code>Fill()</code>: Kalan tüm boşluğu doldurur.</li>
                <li><code>Min(N)</code> / <code>Max(N)</code>: Alt ve üst sınır garantileri.</li>
                <li><code>FitContent()</code>: İçindeki widget'ın SizeHint boyutuna göre dinamik alan ayırır.</li>
            </ul>
        `
    },

    "widgets-display": {
        title: "📦 widgets — Görsel & Tablo Bileşenleri",
        content: `
            <h1>📦 Görsel & Bilgi Widget'ları</h1>
            <p>Limoni; Block, Paragraph, Table, VirtualDataView (1M+ satır), Markdown, RichText, ProgressBar ve List gibi akıcı yapıcılarla donatılmış bileşenler sunar.</p>

            <h2>Akıcı Block & Tablo</h2>
            <div class="code-box">
                <div class="code-header"><span>GOLANG</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <pre><code>table := limoni.NewTable().
    WithHeaders("PID", "PROSES", "CPU %").
    WithRow("1024", "nginx", "4.2%").
    WithRow("2048", "postgres", "12.8%").
    WithConstraints(limoni.Fixed(8), limoni.Fill(), limoni.Fixed(10)).
    WithGrid(true)

block := limoni.NewBlock().
    WithTitle(" Sistem Süreçleri ").
    Rounded().
    WithChild(table)

f.RenderWidget(block, area)</code></pre>
            </div>
        `
    },

    "widgets-inputs": {
        title: "✍️ Formlar & Girdi Kutuları",
        content: `
            <h1>✍️ Formlar & Girdi Kutuları</h1>
            <p><code>TextInput</code>, <code>Checkbox</code>, <code>RadioButton</code>, <code>Slider</code> ve <code>ColorPicker</code> kontrolleri ile modern etkileşimli formlar oluşturun.</p>

            <div class="code-box">
                <div class="code-header"><span>GOLANG</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <pre><code>input := limoni.NewTextInput("token_field").
    WithPlaceholder("API Token giriniz...").
    WithFocusedStyle(limoni.Fg(limoni.Hex("#00E5FF")).Bold())

f.RenderWidget(input, area)</code></pre>
            </div>
        `
    },

    "widgets-modals": {
        title: "🪟 Modallar & Dialoglar",
        content: `
            <h1>🪟 Modallar & Dialoglar</h1>
            <p>Işıltılı degrade kenarlıklar, gölgelendirme, taşınabilir başlık çubuğu ve tekil buton odak kontrolü sunan cam efektli (glassmorphism) pencereler.</p>

            <div class="code-box">
                <div class="code-header"><span>GOLANG</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <pre><code>dialog := widgets.Dialog{
    ID:      "exit_dialog",
    Title:   " ⚠️ DİKKAT ",
    Message: "Değişiklikler kaydedilsin mi?",
    Shadow:  true,
    Buttons: []widgets.DialogButton{
        {Text: "İptal", Handler: cancelFunc},
        {Text: "Kaydet", Handler: saveFunc},
    },
}
f.BeginFocusScope("exit_dialog")
f.RenderWidget(dialog, modalArea)</code></pre>
            </div>
        `
    },

    "widgets-custom": {
        title: "🛠️ Özel Widget Geliştirme",
        content: `
            <h1>🛠️ Özel Widget Geliştirme</h1>
            <p>Kendi özel widget'ınızı oluşturmak için sadece <code>widgets.Widget</code> arayüzünü (Draw ve SizeHint) uygulamanız yeterlidir:</p>

            <div class="code-box">
                <div class="code-header"><span>GOLANG</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <pre><code>type MyStatusBadge struct {
    Online bool
}

func (b MyStatusBadge) Draw(ctx cell.Context, buf *buffer.Buffer) {
    text := "🔴 OFFLINE"
    col := cell.NewColorRGB(255, 80, 80)
    if b.Online {
        text = "🟢 ONLINE"
        col = cell.NewColorRGB(80, 220, 140)
    }
    buf.SetString(ctx.Area.X, ctx.Area.Y, text, cell.Style{Fg: col, Modifier: cell.ModifierBold})
}

func (b MyStatusBadge) SizeHint(maxArea cell.Rect) (uint16, uint16) {
    return 10, 1
}</code></pre>
            </div>
        `
    },

    "graphics-3d": {
        title: "🧊 3D Mesh & Shader Motoru",
        content: `
            <h1>🧊 3D Mesh & Shader Motoru</h1>
            <p>Limoni; terminalde 3D modelleri (.obj, .stl, .ply) yükleyebilir, fareyle 360 derece döndürebilir ve gerçek zamanlı gölgelendirebilir.</p>

            <h2>Desteklenen Gölgelendiriciler</h2>
            <ul>
                <li><strong>Wireframe</strong>: Modelin yapısal kenar çizgileri.</li>
                <li><strong>Lambertian Diffuse</strong>: Işık kaynağı ve yüzey normalleriyle gerçekçi ışıklandırma.</li>
                <li><strong>Gouraud Shading</strong>: Köşeler arası pürüzsüz enterpolasyonlu renk geçişi.</li>
                <li><strong>Doku Kaplama</strong>: PNG dokularının UV haritasıyla giydirilmesi.</li>
            </ul>
        `
    },

    "graphics-canvas": {
        title: "🎨 2D Braille Canvas & Resim Protokolleri",
        content: `
            <h1>🎨 2D Braille Canvas & Resim Protokolleri</h1>
            <p>Braille Unicode ızgarası (2x4) ile her hücrede 8 alt-piksel çözünürlüklü vektör çizimi sunar.</p>

            <h2>Yerel Terminal Resim Protokolleri</h2>
            <ul>
                <li><strong>Kitty Graphics Protocol</strong>: Piksel hassasiyetinde donanım hızlandırmalı resim iletimi.</li>
                <li><strong>Sixel</strong>: 6 piksellik şeritlerle xterm/wezterm/foot uyumlu resim çizimi.</li>
                <li><strong>iTerm2 Protocol</strong>: macOS iTerm2 yerel resim formatı.</li>
                <li><strong>Half-Block Fallback</strong>: Resim protokolü olmayan terminallerde UTF-8 yarım blok (▀ ▄) ile TrueColor görsel önizleme.</li>
            </ul>
        `
    },

    animation: {
        title: "🎬 Animasyon & Fizik Motoru",
        content: `
            <h1>🎬 Animasyon & Fizik Motoru</h1>
            <p><code>animation.Float</code>, <code>animation.Color</code> ve Easing eğrileri (Quad, Cubic, Bounce, Elastic) ile 60 FPS hızında pürüzsüz animasyonlar sağlar.</p>
            <div class="code-box">
                <div class="code-header"><span>GOLANG</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <pre><code>import "github.com/thebanri/limoni"

// Yumuşak renk geçişi
colorAnim := animation.NewColor(limoni.RGB(30, 144, 255))
colorAnim.SetTarget(limoni.RGB(255, 69, 0), 300*time.Millisecond, animation.EaseInOutQuad)

// Sayısal enterpolasyon
valAnim := animation.NewFloat(0.0)
valAnim.SetTarget(100.0, 500*time.Millisecond, animation.EaseOutCubic)</code></pre>
            </div>
        `
    },

    platforms: {
        title: "🌐 Çapraz Platform Sürücüleri",
        content: `
            <h1>🌐 Çapraz Platform Desteği</h1>
            <p>Limoni, harici hiçbir C kütüphanesine (cgo) ihtiyaç duymadan saf Go ile derlenir:</p>
            <ul>
                <li><strong>Linux & macOS</strong>: termios tabanlı ham mod, VT100 / XTerm kontrol dizileri.</li>
                <li><strong>Windows</strong>: Yerel Win32 Console API ve Virtual Terminal Processing.</li>
                <li><strong>WebAssembly (WASM)</strong>: Tarayıcı içinde xterm.js ile sıfır değişiklikle çalıştırma.</li>
                <li><strong>SSH Sunucuları</strong>: Uzak terminal istemcilerine ağ üzerinden doğrudan TUI yayını.</li>
            </ul>
        `
    },

    widgets: {
        title: "📦 Zengin Widget Kataloğu (30 Bileşen)",
        content: `
            <h1>📦 Zengin Widget Ekosistemi</h1>
            <p>Limoni, terminal arayüzünüzü hızlıca inşa etmeniz için 30 adet hazır, yüksek performanslı bileşen ve akıcı (fluent) yapıcılar sunar.</p>

            <div class="cards-grid">
                <div class="feature-card">
                    <div class="card-icon">📊</div>
                    <h3>Görsel & Tablolar</h3>
                    <p><code>Block</code>, <code>Paragraph</code>, <code>Table</code>, <code>VirtualDataView</code> (1M+ satır), <code>ProgressBar</code>, <code>Sparkline</code>, <code>LineChart</code>, <code>PieChart</code>, <code>Markdown</code></p>
                </div>
                <div class="feature-card">
                    <div class="card-icon">✍️</div>
                    <h3>Formlar & Girdi</h3>
                    <p><code>TextInput</code>, <code>TextArea</code>, <code>Checkbox</code>, <code>RadioButton</code>, <code>Select</code>, <code>Slider</code></p>
                </div>
                <div class="feature-card">
                    <div class="card-icon">🪟</div>
                    <h3>Modallar & Gezinti</h3>
                    <p><code>Dialog</code> (Glassmorphism), <code>Popup</code>, <code>CommandPalette</code>, <code>ToastManager</code>, <code>TreeView</code>, <code>List</code></p>
                </div>
                <div class="feature-card">
                    <div class="card-icon">🧊</div>
                    <h3>Grafik & 3D</h3>
                    <p><code>Viewer3D</code> (Wavefront OBJ/STL/PLY), <code>Canvas</code> (2x4 Braille), <code>Image</code> (Kitty, Sixel, iTerm2)</p>
                </div>
            </div>

            <h2>⚡ Akıcı (Fluent) Yapıcı Örneği</h2>
            <div class="code-box">
                <div class="code-header"><span>GOLANG</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <pre><code>import "github.com/thebanri/limoni"

// Birleşik kök paket üzerinden zincirleme widget oluşturma:
card := limoni.NewBlock().
    WithTitle(" 🚀 Sistem Durumu ").
    Rounded().
    WithBorderStyle(limoni.Fg(limoni.ColorCyan)).
    WithChild(
        limoni.NewParagraph("Tüm servisler %99.98 uptime ile çalışıyor.").
            WithStyle(limoni.Fg(limoni.ColorGreen).Bold()),
    )

f.RenderWidget(card, area)</code></pre>
            </div>
        `
    },

    graphics: {
        title: "🎨 2D & 3D Grafik, Canvas ve Resim Protokolleri",
        content: `
            <h1>🎨 2D & 3D Grafik Motoru</h1>
            <p>Limoni, klasik metin tabanlı TUI sınırlarını aşarak terminal içinde Braille vektör çizimleri, 3D gölgelendirme motoru ve piksel resim sürücüleri sunar.</p>

            <h2>🧊 3D Model Render Motoru (Viewer3D)</h2>
            <p>Wavefront OBJ, STL veya PLY dosyalarını terminalde döndürün, Lambert/Gouraud shader ile gölgelendirin veya PNG dokusu giydirin:</p>
            <div class="code-box">
                <div class="code-header"><span>GOLANG</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <pre><code>mesh, _ := graphics.LoadOBJ("assets/teapot.obj")

viewer := widgets.NewViewer3D(mesh).
    WithRotation(angleX, angleY, 0).
    WithShading("Gölgeli"). // "Dokulu", "Wireframe", "Dolu Renkli", "Gölgeli", "Gouraud"
    WithWireframe(true)

f.RenderWidget(viewer, area)</code></pre>
            </div>

            <h2>🎨 2D Braille Canvas</h2>
            <p>Hücre başına 2x4 piksel çözünürlük ile çizgiler, çemberler ve derinlik tamponlu (Z-Buffer) üçgenler çizin:</p>
            <div class="code-box">
                <div class="code-header"><span>GOLANG</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <pre><code>canvas := widgets.NewCanvas(width, height)
canvas.DrawLine(0, 0, 80, 40, limoni.Fg(limoni.RGB(0, 255, 200)))
canvas.DrawCircle(40, 20, 15, limoni.Fg(limoni.RGB(255, 215, 0)))
f.RenderWidget(canvas, area)</code></pre>
            </div>

            <h2>🖼️ Donanım Resim Protokolleri</h2>
            <ul>
                <li><strong>Kitty Graphics Protocol</strong>: RGB piksel akışı.</li>
                <li><strong>Sixel Graphics</strong>: DEC VT terminalleri ve xterm uyumluluğu.</li>
                <li><strong>iTerm2 Protocol</strong>: macOS için base64 resim aktarımı.</li>
                <li><strong>Half-Block Fallback</strong>: Eski terminallerde 1x2 ANSI yarım blok geriye uyumluluk.</li>
            </ul>
        `
    },

    examples: {
        title: "🚀 Örnek Uygulamalar & Canlı Demolar",
        content: `
            <h1>🚀 Canlı Örnekler ve Demolar</h1>
            <p>Limoni deposunda yer alan hazır çalışan uygulamaları tek komutla test edebilirsiniz:</p>

            <div class="cards-grid">
                <div class="feature-card">
                    <div class="card-icon">🌟</div>
                    <h3>Showcase Uygulaması</h3>
                    <p>7 sekmeli vitrin: Markdown okuyucu, 3D döndürücü, canlı grafikler, 1M satırlı tablo ve formlar.</p>
                    <div class="code-box" style="margin-top:10px;">
                        <pre><code>go run ./examples/showcase</code></pre>
                    </div>
                </div>

                <div class="feature-card">
                    <div class="card-icon">⚡</div>
                    <h3>Basit Başlangıç</h3>
                    <p>15 satırda tek import ile çalışan minimal sayaç uygulaması.</p>
                    <div class="code-box" style="margin-top:10px;">
                        <pre><code>go run ./examples/simple</code></pre>
                    </div>
                </div>

                <div class="feature-card">
                    <div class="card-icon">🎮</div>
                    <h3>İnteraktif Demo</h3>
                    <p>Animasyonlar, CPU grafikleri ve cam efektli dialog pencereleri.</p>
                    <div class="code-box" style="margin-top:10px;">
                        <pre><code>go run ./examples/demo</code></pre>
                    </div>
                </div>

                <div class="feature-card">
                    <div class="card-icon">🌐</div>
                    <h3>SSH TUI Sunucusu</h3>
                    <p>Ağ üzerinden bağlanan kullanıcılara canlı TUI sunan SSH servisi.</p>
                    <div class="code-box" style="margin-top:10px;">
                        <pre><code>go run ./examples/ssh_server</code></pre>
                    </div>
                </div>
            </div>
        `
    },

    bubbletea: {
        title: "🍵 Bubble Tea → Limoni Geçiş Kılavuzu",
        content: `
            <h1>🍵 Bubble Tea'den Limoni'ye Geçiş</h1>
            <p>Mevcut Bubble Tea ve Lipgloss projelerinizi Limoni'nin ultra hızlı çekirdeğine zahmetsizce taşıyabilirsiniz.</p>

            <h2>1. Adım: Sıfır Kod Değişikliğiyle Adapter Kullanımı</h2>
            <p>Sadece import satırını değiştirerek mevcut modellerinizi doğrudan çalıştırın:</p>
            <div class="code-box">
                <div class="code-header"><span>GOLANG</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <pre><code>// Önce: import tea "github.com/charmbracelet/bubbletea"
// Şimdi:
import tea "github.com/thebanri/limoni/compat/bubbletea"

p := tea.NewProgram(model)
err := p.RunTerminal(context.Background())</code></pre>
            </div>

            <h2>2. Adım: Native Limoni API'sine Geçiş</h2>
            <p>Daha yüksek kare hızları ve sıfır bellek tahsisatı için <code>limoni.Model</code> arayüzüne geçin:</p>
            <div class="code-box">
                <div class="code-header"><span>GOLANG</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <pre><code>package main

import (
    "context"
    "github.com/thebanri/limoni"
)

type model struct{ count int }

func (m *model) Init() []limoni.Cmd { return nil }

func (m *model) Update(msg limoni.Msg) limoni.UpdateResult {
    switch msg := msg.(type) {
    case limoni.KeyPressMsg:
        if msg.Key.Type == limoni.KeyEsc {
            return limoni.Quit()
        }
        m.count++
        return limoni.Redraw()
    }
    return limoni.Noop()
}

func (m *model) View(f *limoni.Frame) {
    f.RenderWidget(
        limoni.NewBlock().
            Title(" Limoni Native ").
            Rounded().
            Style(limoni.Fg(limoni.ColorCyan)),
        f.Area(),
    )
}

func main() {
    _ = limoni.RunProgram(context.Background(), &model{})
}</code></pre>
            </div>
        `
    },

    a11y: {
        title: "♿ Erişilebilirlik (A11y) & Anlamsal Ağaç",
        content: `
            <h1>♿ Erişilebilirlik ve Standartlar</h1>
            <p>Limoni, terminal arayüzlerinin tüm kullanıcılar tarafından erişilebilir olmasını temel bir tasarım ilkesi kabul eder.</p>

            <h2>1. Anlamsal Ağaç (Semantic Node Tree)</h2>
            <p>Tüm bileşenler çizim anında ekran okuyucular için yapılandırılmış semantik düğümler üretir (Rol, Durum, Değer).</p>

            <h2>2. Yüksek Kontrast (WCAG AAA)</h2>
            <p>Yüksek kontrast modu etkinleştirildiğinde soluk pastel renkler otomatik olarak maksimum kontrastlı zıt renklere yükseltilir.</p>

            <h2>3. NO_COLOR Standardı</h2>
            <p>Ortamda <code>NO_COLOR=1</code> tanımlı olduğunda tüm ANSI renk kodları kapatılarak arayüz sembolik olarak çizilir.</p>

            <h2>4. Azaltılmış Hareket (Reduced Motion)</h2>
            <p>Vestibüler hassasiyeti olan kullanıcılar için animasyon geçişleri otomatik olarak devre dışı bırakılır.</p>
        `
    }
};

// Routing and Section Switcher
function switchSection(sectionId) {
    if (!DOCS_DATA[sectionId]) {
        sectionId = 'quickstart';
    }

    // Update active nav links
    document.querySelectorAll('.sidebar-link').forEach(link => {
        link.classList.remove('active');
        if (link.getAttribute('href') === `#${sectionId}`) {
            link.classList.add('active');
        }
    });

    document.querySelectorAll('.nav-link').forEach(link => {
        link.classList.remove('active');
        if (link.getAttribute('href') === `#${sectionId}`) {
            link.classList.add('active');
        }
    });

    // Render content
    const container = document.getElementById('content-container');
    container.innerHTML = DOCS_DATA[sectionId].content;

    // Generate Table of Contents
    generateTOC();

    // Scroll to top
    window.scrollTo({ top: 0, behavior: 'smooth' });
}

// Generate On-Page TOC
function generateTOC() {
    const tocList = document.getElementById('toc-list');
    tocList.innerHTML = '';

    const headings = document.querySelectorAll('#content-container h2');
    headings.forEach((h2, idx) => {
        const id = 'heading-' + idx;
        h2.id = id;

        const li = document.createElement('li');
        const a = document.createElement('a');
        a.href = '#' + id;
        a.className = 'toc-link';
        a.textContent = h2.textContent;
        a.onclick = (e) => {
            e.preventDefault();
            h2.scrollIntoView({ behavior: 'smooth' });
        };
        li.appendChild(a);
        tocList.appendChild(li);
    });
}

// Copy Code Button
function copyCode(btn) {
    const codeBox = btn.closest('.code-box');
    const code = codeBox.querySelector('code').textContent;
    navigator.clipboard.writeText(code).then(() => {
        btn.textContent = 'Kopyalandı! ✓';
        btn.style.color = '#39d353';
        setTimeout(() => {
            btn.textContent = 'Kopyala';
            btn.style.color = '';
        }, 2000);
    });
}

// Search Modal Functionality (Cmd+K / Ctrl+K)
function openSearchModal() {
    const modal = document.getElementById('search-modal');
    modal.classList.add('open');
    const input = document.getElementById('search-input');
    input.value = '';
    input.focus();
    handleSearch('');
}

function closeSearchModal(event) {
    if (event && event.target !== document.getElementById('search-modal') && !event.target.classList.contains('search-modal-close')) {
        return;
    }
    const modal = document.getElementById('search-modal');
    modal.classList.remove('open');
}

function handleSearch(query) {
    const resultsContainer = document.getElementById('search-results');
    query = query.toLowerCase().trim();

    if (!query) {
        resultsContainer.innerHTML = '<div class="search-hint">Aramak için yazmaya başlayın... (Örn: Table, 3D, Flexbox)</div>';
        return;
    }

    const matched = [];
    for (const [key, doc] of Object.entries(DOCS_DATA)) {
        if (doc.title.toLowerCase().includes(query) || doc.content.toLowerCase().includes(query)) {
            matched.push({ key, title: doc.title });
        }
    }

    if (matched.length === 0) {
        resultsContainer.innerHTML = '<div class="search-hint">Sonuç bulunamadı.</div>';
        return;
    }

    resultsContainer.innerHTML = matched.map(m => `
        <a href="#${m.key}" class="search-result-item" onclick="selectSearchResult('${m.key}')">
            <div class="search-result-title">${m.title}</div>
            <div class="search-result-desc">Limoni Dokümantasyon Rehberi &rarr; ${m.key}</div>
        </a>
    `).join('');
}

function selectSearchResult(key) {
    closeSearchModal();
    switchSection(key);
}

// Keyboard shortcuts (Cmd+K / Ctrl+K and ESC)
window.addEventListener('keydown', (e) => {
    if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        openSearchModal();
    } else if (e.key === 'Escape') {
        closeSearchModal();
    }
});

// Handle initial URL hash
window.addEventListener('DOMContentLoaded', () => {
    const initialHash = window.location.hash.replace('#', '') || 'quickstart';
    switchSection(initialHash);
});

window.addEventListener('hashchange', () => {
    const hash = window.location.hash.replace('#', '') || 'quickstart';
    if (!hash.startsWith('heading-')) {
        switchSection(hash);
    }
});
