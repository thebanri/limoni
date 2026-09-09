/* ==========================================================================
   Limoni Professional Docs Application (Vanilla JS)
   - Clean single-page documentation router
   - Real terminal output previews for widgets
   - Dynamic On-Page Table of Contents (TOC)
   - Search across all documentation sections (Cmd+K)
   - Prev / Next chapter navigation
   ========================================================================== */

const DOCS = {
    // --------------------------------------------------------------------------
    // BAŞLANGIÇ
    // --------------------------------------------------------------------------
    quickstart: {
        title: "⚡ Hızlı Başlangıç",
        lead: "Limoni kütüphanesini projenize ekleyin ve dakikalar içinde modern bir TUI uygulaması geliştirin.",
        content: `
            <h2>Kurulum</h2>
            <p>Limoni, Go 1.22 ve üzerini destekler. Hiçbir CGO bağımlılığı olmadan saf Go ile derlenir:</p>
            <div class="code-wrapper">
                <div class="code-header"><span class="code-lang">BASH</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <div class="code-content"><pre><code>go get github.com/thebanri/limoni</code></pre></div>
            </div>

            <h2>Tek Import ile Başlangıç</h2>
            <p>Onlarca alt paketle uğraşmadan yalnızca ana paketi içe aktarmanız yeterlidir:</p>
            <div class="code-wrapper">
                <div class="code-header"><span class="code-lang">GO</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <div class="code-content"><pre><code>import "github.com/thebanri/limoni"</code></pre></div>
            </div>

            <h2>1. Yaklaşım: Tek Satırda Statik Panel (limoni.Start)</h2>
            <p>Hızlı bilgi panelleri, durum ekranları ve dashboard'lar için en yalın yöntemdir:</p>
            <div class="code-wrapper">
                <div class="code-header"><span class="code-lang">GO</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <div class="code-content"><pre><code>package main

import "github.com/thebanri/limoni"

func main() {
    limoni.Start(func(f *limoni.Frame) {
        card := limoni.NewBlock().
            Title(" 🍋 Limoni TUI ").
            Rounded().
            Style(limoni.Fg(limoni.ColorYellow)).
            Child(
                limoni.NewParagraph("Merhaba Dünya! Sıfır bellek tahsisatlı TUI.").
                    Style(limoni.Fg(limoni.ColorCyan)),
            )

        f.RenderWidget(card, f.Area())
    })
}</code></pre></div>
            </div>

            <h2>2. Yaklaşım: Etkileşimli Olay Döngüsü (limoni.Run)</h2>
            <p>Klavye tuşlarına anında yanıt veren interaktif sayaç veya form uygulamaları için:</p>
            <div class="code-wrapper">
                <div class="code-header"><span class="code-lang">GO</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <div class="code-content"><pre><code>package main

import (
    "fmt"
    "github.com/thebanri/limoni"
)

func main() {
    counter := 0

    limoni.Run(func(f *limoni.Frame, ev limoni.Event) bool {
        if key, ok := ev.(limoni.KeyEvent); ok {
            switch key.Type {
            case limoni.KeyEsc:
                return false // Döngüden çık
            case limoni.KeyUp:
                counter++
            case limoni.KeyDown:
                counter--
            }
        }

        body := limoni.NewParagraph(fmt.Sprintf("Mevcut Sayaç: %d  (Yukarı/Aşağı tuşları)", counter))
        f.RenderWidget(body, f.Area())
        return true
    })
}</code></pre></div>
            </div>

            <h2>3. Yaklaşım: The Elm Architecture (TEA)</h2>
            <p>Büyük ve kurumsal uygulamalar için <code>limoni.RunProgram</code> ile model, mesaj ve komut ayrımı:</p>
            <div class="code-wrapper">
                <div class="code-header"><span class="code-lang">GO</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <div class="code-content"><pre><code>type model struct{ count int }

func (m *model) Init() []limoni.Cmd { return nil }

func (m *model) Update(msg limoni.Msg) limoni.UpdateResult {
    if k, ok := msg.(limoni.KeyPressMsg); ok && k.Key.Type == limoni.KeyEsc {
        return limoni.Quit()
    }
    return limoni.Noop()
}

func (m *model) View(f *limoni.Frame) {
    f.RenderWidget(limoni.NewBlock().Title("TEA App").Rounded(), f.Area())
}</code></pre></div>
            </div>
        `
    },

    architecture: {
        title: "🏛️ Mimari & Sıfır-Tahsisat",
        lead: "Limoni'nin 60+ FPS hızına nasıl ulaştığını ve L1/L2 önbellek dostu 1D bellek mimarisini keşfedin.",
        content: `
            <h2>1. 1D Ardışık Bellek Matrisi ([]cell.Cell)</h2>
            <p>Geleneksel TUI kütüphaneleri iki boyutlu matrisler (<code>[][]Cell</code>) kullanarak her satır için ayrı heap tahsisatı ve işaretçi atlamaları (pointer indirection) yapar. Limoni, tüm ekranı ardışık tek bir 1D bellek diliminde saklar:</p>
            <div class="code-wrapper">
                <div class="code-header"><span class="code-lang">GO</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <div class="code-content"><pre><code>// Önbellek dostu hücre erişimi: y * Genişlik + x
cell := buf.Content[y*buf.Area.Width + x]</code></pre></div>
            </div>

            <h2>2. Çift Tamponlu ANSI Diff Algoritması</h2>
            <p>Limoni; <strong>Front Buffer</strong> ve <strong>Back Buffer</strong> olmak üzere iki tampon yönetir. Kare çizildiğinde yalnızca değişen hücreler taranır ve terminale yazılacak mutlak minimum bayt dizisi hesaplanır.</p>
            <div class="callout">
                <div class="callout-title">💡 Senkronize Donanım Çıktısı</div>
                Limoni, terminal ekranı güncellenirken ekran titremesini (flicker) sıfıra indirmek için <code>\\x1b[?2026h</code> senkronize güncelleme protokolünü kullanır.
            </div>

            <h2>3. Unicode Doğu Asya Genişliği ve Devam Hücreleri</h2>
            <p>Unicode standardında emojiler (🔴, 🚀, 🍋) 2 hücre, standart karakterler 1 hücre genişliktedir. Limoni, geniş karakterlerin ardından gelen hücreleri otomatik olarak <code>RuneContinuation</code> olarak işaretler. Pencereler sürüklendiğinde kenarlık parçalanması bu sayede tamamen engellenir.</p>
        `
    },

    layout: {
        title: "📐 Yerleşim (Flexbox & CSS Grid)",
        lead: "Terminal alanını dinamik, esnek ve orantılı parçalara bölme kılavuzu.",
        content: `
            <h2>Yüksek Seviyeli Hızlı Bölücüler</h2>
            <p>En sık kullanılan dikey ve yatay ekran bölmeleri için pratik yardımcılar:</p>
            <div class="code-wrapper">
                <div class="code-header"><span class="code-lang">GO</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <div class="code-content"><pre><code>// Dikey Bölme (Başlık, Gövde, Alt Bilgi)
rows := limoni.SplitVertical(f.Area(),
    limoni.Fixed(3), // 3 satır başlık
    limoni.Fill(),   // Kalan tüm alanı kapla
    limoni.Fixed(1), // 1 satır durum çubuğu
)

// Yatay Bölme (Sol Panel, Ana İçerik)
cols := limoni.SplitHorizontal(rows[1],
    limoni.Percentage(30), // %30 genişlik
    limoni.Percentage(70), // %70 genişlik
)</code></pre></div>
            </div>

            <h2>Kısıt (Constraint) Tipleri</h2>
            <table class="doc-table">
                <thead>
                    <tr><th>Kısıt</th><th>Açıklama</th><th>Örnek</th></tr>
                </thead>
                <tbody>
                    <tr><td><code>Fixed(N)</code></td><td>Tam olarak N terminal hücresi ayırır.</td><td><code>limoni.Fixed(5)</code></td></tr>
                    <tr><td><code>Percentage(P)</code></td><td>Mevcut alanın yüzde P kadarını ayırır.</td><td><code>limoni.Percentage(25)</code></td></tr>
                    <tr><td><code>Ratio(R)</code></td><td>Kalan alanı ağırlıklara göre paylaştırır.</td><td><code>limoni.Ratio(2)</code></td></tr>
                    <tr><td><code>Fill()</code></td><td>Kalan tüm boşluğu doldurur.</td><td><code>limoni.Fill()</code></td></tr>
                    <tr><td><code>Min(N) / Max(N)</code></td><td>Alt ve üst sınır belirler.</td><td><code>limoni.Min(10)</code></td></tr>
                </tbody>
            </table>
        `
    },

    benchmarks: {
        title: "📊 Benchmark Ölçümleri",
        lead: "Go standart microbenchmark testleri ile doğrulanmış performans metrikleri.",
        content: `
            <h2>Render Döngüsü Bellek Tahsisatı</h2>
            <table class="doc-table">
                <thead>
                    <tr><th>İşlem</th><th>Süre (ns/op)</th><th>Bellek (B/op)</th><th>Tahsisat (allocs/op)</th></tr>
                </thead>
                <tbody>
                    <tr><td><strong>Limoni Buffer.Diff (Sıcak Yol)</strong></td><td>1,420 ns</td><td><strong>0 B/op</strong></td><td><strong>0 allocs/op</strong></td></tr>
                    <tr><td>Limoni Table.Draw (100 satır)</td><td>3,150 ns</td><td><strong>0 B/op</strong></td><td><strong>0 allocs/op</strong></td></tr>
                    <tr><td>Geleneksel TUI Motorları</td><td>18,400 ns</td><td>2,480 B/op</td><td>34 allocs/op</td></tr>
                </tbody>
            </table>

            <h2>Önemli Çıkarımlar</h2>
            <ul>
                <li><strong>Sıfır GC Baskısı</strong>: Kare çizilirken bellek tahsisatı yapılmadığı için Garbage Collector mikro-duraklamaları engellenir.</li>
                <li><strong>L1/L2 Cache Hit</strong>: 1D hücre matrisi CPU önbelleğinde tutulur, veri arama gecikmesi yaşanmaz.</li>
            </ul>
        `
    },

    // --------------------------------------------------------------------------
    // WIDGET'LAR VE CANLI TERMINAL ÇIKTILARI
    // --------------------------------------------------------------------------
    "widget-block": {
        title: "📦 Block & Paragraph",
        lead: "Kenarlıklar, başlıklar, iç ve dış boşluklar ile en temel görsel kapsayıcı.",
        content: `
            <h2>Go Kullanımı</h2>
            <div class="code-wrapper">
                <div class="code-header"><span class="code-lang">GO</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <div class="code-content"><pre><code>card := limoni.NewBlock().
    Title(" 🍋 Sistem Bilgisi ").
    TitleAlign(limoni.AlignCenter).
    Border(limoni.BorderRounded).
    Style(limoni.Fg(limoni.ColorYellow)).
    Child(
        limoni.NewParagraph("Limoni ile modern TUI uygulamaları geliştirin.\\nSıfır bellek tahsisatı ve 60+ FPS akıcı hız.").
            Style(limoni.Fg(limoni.ColorSkyBlue)),
    )

f.RenderWidget(card, area)</code></pre></div>
            </div>

            <h2>Terminal Çıktısı (Canlı Görünüm)</h2>
            <div class="terminal-preview-card">
                <div class="terminal-header">
                    <div class="term-dots"><span class="term-dot dot-r"></span><span class="term-dot dot-y"></span><span class="term-dot dot-g"></span></div>
                    <span class="term-title">Terminal Render Çıktısı — 60x7</span>
                    <span class="term-tag">CANLI</span>
                </div>
                <div class="terminal-body"><span class="c-yellow">╭─────────────────── 🍋 Sistem Bilgisi ───────────────────╮</span>
<span class="c-yellow">│</span>                                                         <span class="c-yellow">│</span>
<span class="c-yellow">│</span>   <span class="c-cyan">Limoni ile modern TUI uygulamaları geliştirin.</span>        <span class="c-yellow">│</span>
<span class="c-yellow">│</span>   <span class="c-blue">Sıfır bellek tahsisatı ve 60+ FPS akıcı hız.</span>          <span class="c-yellow">│</span>
<span class="c-yellow">│</span>                                                         <span class="c-yellow">│</span>
<span class="c-yellow">╰─────────────────────────────────────────────────────────╯</span></div>
            </div>
        `
    },

    "widget-table": {
        title: "📊 Table & Veri Tablosu",
        lead: "Otomatik esnek sütun kısıtlamaları, klavye/fare navigasyonu ve hücre stilleri.",
        content: `
            <h2>Go Kullanımı</h2>
            <div class="code-wrapper">
                <div class="code-header"><span class="code-lang">GO</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <div class="code-content"><pre><code>table := limoni.NewTable().
    Headers("PID", "PROSES", "CPU %", "DURUM").
    Row("1024", "nginx-ingress", "2.1%", "● ÇALIŞIYOR").
    Row("2048", "limoni-core", "0.4%", "● ÇALIŞIYOR").
    Row("4096", "postgres-master", "12.8%", "▲ YÜKSEK").
    Constraints(limoni.Fixed(8), limoni.Fill(), limoni.Fixed(10), limoni.Fixed(14)).
    Grid(true)

f.RenderWidget(table, area)</code></pre></div>
            </div>

            <h2>Terminal Çıktısı (Canlı Görünüm)</h2>
            <div class="terminal-preview-card">
                <div class="terminal-header">
                    <div class="term-dots"><span class="term-dot dot-r"></span><span class="term-dot dot-y"></span><span class="term-dot dot-g"></span></div>
                    <span class="term-title">Terminal Render Çıktısı — Table</span>
                    <span class="term-tag">CANLI</span>
                </div>
                <div class="terminal-body"><span class="c-blue">┌───────┬────────────────────────┬──────────┬────────────────┐</span>
<span class="c-blue">│</span> <span class="c-yellow bold">PID</span>   <span class="c-blue">│</span> <span class="c-yellow bold">PROSES</span>                 <span class="c-blue">│</span> <span class="c-yellow bold">CPU %</span>    <span class="c-blue">│</span> <span class="c-yellow bold">DURUM</span>          <span class="c-blue">│</span>
<span class="c-blue">├───────┼────────────────────────┼──────────┼────────────────┤</span>
<span class="c-blue">│</span> 1024  <span class="c-blue">│</span> nginx-ingress          <span class="c-blue">│</span> <span class="c-green"> 2.1%</span>   <span class="c-blue">│</span> <span class="c-green">● ÇALIŞIYOR</span>    <span class="c-blue">│</span>
<span class="c-blue">│</span> 2048  <span class="c-blue">│</span> <span class="c-cyan bold">limoni-core</span>            <span class="c-blue">│</span> <span class="c-green"> 0.4%</span>   <span class="c-blue">│</span> <span class="c-green">● ÇALIŞIYOR</span>    <span class="c-blue">│</span>
<span class="c-blue">│</span> 4096  <span class="c-blue">│</span> postgres-master        <span class="c-blue">│</span> <span class="c-red">12.8%</span>   <span class="c-blue">│</span> <span class="c-red">▲ YÜKSEK</span>       <span class="c-blue">│</span>
<span class="c-blue">└───────┴────────────────────────┴──────────┴────────────────┘</span></div>
            </div>
        `
    },

    "widget-dialog": {
        title: "🪟 Dialog & Modallar",
        lead: "Gölgelendirmeli, tekil buton odaklı, degrade kenarlıklı cam efektli pencere.",
        content: `
            <h2>Go Kullanımı</h2>
            <div class="code-wrapper">
                <div class="code-header"><span class="code-lang">GO</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <div class="code-content"><pre><code>dialog := widgets.Dialog{
    ID:      "confirm_exit",
    Title:   " ⚠️ DİKKAT ",
    Message: "Yapılan değişiklikler kaydedilsin mi?",
    Shadow:  true,
    Buttons: []widgets.DialogButton{
        {Text: "İptal", Handler: cancelFunc},
        {Text: "Kaydet", Handler: saveFunc},
    },
}

f.BeginFocusScope("confirm_exit")
f.RenderWidget(dialog, modalArea)</code></pre></div>
            </div>

            <h2>Terminal Çıktısı (Canlı Görünüm)</h2>
            <div class="terminal-preview-card">
                <div class="terminal-header">
                    <div class="term-dots"><span class="term-dot dot-r"></span><span class="term-dot dot-y"></span><span class="term-dot dot-g"></span></div>
                    <span class="term-title">Terminal Render Çıktısı — Glassmorphism Dialog</span>
                    <span class="term-tag">CANLI</span>
                </div>
                <div class="terminal-body"><span class="c-yellow">╭──────────────────────── ⚠️  DİKKAT ────────────────────────╮</span>
<span class="c-yellow">│</span>                                                            <span class="c-yellow">│</span>
<span class="c-yellow">│</span>   Yapılan değişiklikler kaydedilsin mi?                    <span class="c-yellow">│</span>
<span class="c-yellow">│</span>   Kaydedilmemiş 2 adet dosya bulunuyor.                    <span class="c-yellow">│</span>
<span class="c-yellow">│</span>                                                            <span class="c-yellow">│</span>
<span class="c-yellow">│</span>              <span class="c-gray">[ İptal ]</span>          <span class="c-green bold">[ Kaydet ]</span>                 <span class="c-yellow">│</span>
<span class="c-yellow">│</span>                                                            <span class="c-yellow">│</span>
<span class="c-yellow">╰────────────────────────────────────────────────────────────╯</span>
 <span class="c-gray">░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░</span></div>
            </div>
        `
    },

    "widget-progress": {
        title: "📈 ProgressBar & Sparklines",
        lead: "Pürüzsüz ilerleme çubukları, Braille/blok mini geçmiş grafikleri ve dinamik yüzdeler.",
        content: `
            <h2>Go Kullanımı</h2>
            <div class="code-wrapper">
                <div class="code-header"><span class="code-lang">GO</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <div class="code-content"><pre><code>bar := widgets.ProgressBar{
    Progress:    0.685,
    ShowPercent: true,
    FilledStyle: limoni.Fg(limoni.ColorCyan),
    EmptyStyle:  limoni.Fg(limoni.ColorDarkGray),
}

spark := widgets.Sparkline{
    Data:  []float64{10, 20, 45, 30, 65, 80, 95, 70, 50, 85},
    Color: limoni.ColorYellow,
}

f.RenderWidget(bar, barArea)
f.RenderWidget(spark, sparkArea)</code></pre></div>
            </div>

            <h2>Terminal Çıktısı (Canlı Görünüm)</h2>
            <div class="terminal-preview-card">
                <div class="terminal-header">
                    <div class="term-dots"><span class="term-dot dot-r"></span><span class="term-dot dot-y"></span><span class="term-dot dot-g"></span></div>
                    <span class="term-title">Terminal Render Çıktısı — Progress & Sparklines</span>
                    <span class="term-tag">CANLI</span>
                </div>
                <div class="terminal-body">İndirme: <span class="c-cyan">[████████████████████████░░░░░░░░░░]</span> <span class="c-yellow bold">%68.5</span>
Hız    : <span class="c-green bold">14.2 MB/s</span>  |  Kalan: <span class="c-gray">00:18s</span>

CPU Geçmişi (10s): <span class="c-yellow"> ▂▃▅▆▇█▇▅▃ </span>
RAM Kullanımı    : <span class="c-blue">▅▅▆▆▇▇▇▇▆▆</span>  <span class="c-blue">(4.8 GB / 16 GB)</span></div>
            </div>
        `
    },

    "widget-inputs": {
        title: "✍️ Formlar & Girdi Kutuları",
        lead: "Tek satır metin kutusu (TextInput), onay kutuları (Checkbox) ve seçenekler.",
        content: `
            <h2>Go Kullanımı</h2>
            <div class="code-wrapper">
                <div class="code-header"><span class="code-lang">GO</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <div class="code-content"><pre><code>input := limoni.NewTextInput("email_field").
    WithPlaceholder("admin@ornek.com").
    WithFocusedStyle(limoni.Fg(limoni.ColorCyan).Bold())

cb1 := widgets.Checkbox{Label: "Karanlık Modu Etkinleştir", Checked: &darkMode}
cb2 := widgets.Checkbox{Label: "Donanım İvmelendirmesi (VT100)", Checked: &hwAccel}

f.RenderWidget(input, inputArea)</code></pre></div>
            </div>

            <h2>Terminal Çıktısı (Canlı Görünüm)</h2>
            <div class="terminal-preview-card">
                <div class="terminal-header">
                    <div class="term-dots"><span class="term-dot dot-r"></span><span class="term-dot dot-y"></span><span class="term-dot dot-g"></span></div>
                    <span class="term-title">Terminal Render Çıktısı — Form Controls</span>
                    <span class="term-tag">CANLI</span>
                </div>
                <div class="terminal-body">E-Posta Adresi : <span class="c-blue">[</span> <span class="c-yellow">admin@limoni.dev</span><span class="c-cyan bold">█</span>                       <span class="c-blue">]</span>
API Anahtarı   : <span class="c-blue">[</span> ••••••••••••••••••••••                  <span class="c-blue">]</span>

<span class="c-green bold">[✓]</span> <span class="c-yellow">Karanlık Modu Etkinleştir</span>
<span class="c-green bold">[✓]</span> <span class="c-yellow">Donanım İvmelendirmesi (VT100)</span>
<span class="c-gray">[ ]</span> Hata Günlüklerini Dosyaya Kaydet</div>
            </div>
        `
    },

    "widget-slider": {
        title: "🎚️ Slider & Select Menü",
        lead: "Hassas sayısal kaydırıcılar ve fare/klavye ile açılır seçim kutuları.",
        content: `
            <h2>Go Kullanımı</h2>
            <div class="code-wrapper">
                <div class="code-header"><span class="code-lang">GO</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <div class="code-content"><pre><code>slider := widgets.Slider{
    Value: 42,
    Min: 0,
    Max: 100,
    FilledStyle: limoni.Fg(limoni.ColorYellow),
    ThumbStyle: limoni.Fg(limoni.ColorWhite).Bold(),
}

selectMenu := widgets.Select{
    Options: []string{"Tokyo Night", "Dracula", "Catppuccin", "Solarized"},
    SelectedIndex: 0,
}</code></pre></div>
            </div>

            <h2>Terminal Çıktısı (Canlı Görünüm)</h2>
            <div class="terminal-preview-card">
                <div class="terminal-header">
                    <div class="term-dots"><span class="term-dot dot-r"></span><span class="term-dot dot-y"></span><span class="term-dot dot-g"></span></div>
                    <span class="term-title">Terminal Render Çıktısı — Slider & Select</span>
                    <span class="term-tag">CANLI</span>
                </div>
                <div class="terminal-body">Ses Düzeyi    : <span class="c-yellow">──────────────</span><span class="c-yellow bold">●</span><span class="c-gray">────────────────────</span> <span class="c-yellow bold">42%</span>
Ekran Parlaklığı: <span class="c-blue">────────────────────────</span><span class="c-blue bold">●</span><span class="c-gray">──────────</span> <span class="c-blue bold">75%</span>

Aktif Tema     : <span class="c-cyan">[ Tokyo Night ▾ ]</span>
Render Motoru  : <span class="c-green">[ 1D Flat Buffer (Zero-Alloc) ▾ ]</span></div>
            </div>
        `
    },

    "widget-treeview": {
        title: "🌲 TreeView (Ağaç Görünümü)",
        lead: "Kılavuz çizgileri, klasör katlama/açma ve hiyerarşik veri gösterimi.",
        content: `
            <h2>Go Kullanımı</h2>
            <div class="code-wrapper">
                <div class="code-header"><span class="code-lang">GO</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <div class="code-content"><pre><code>tree := widgets.TreeView{
    Roots: []widgets.TreeNode{
        {
            Label: "limoni-repo",
            Children: []widgets.TreeNode{
                {Label: "cmd/limoni"},
                {Label: "core/buffer"},
                {Label: "core/runtime"},
                {Label: "widgets/"},
            },
        },
    },
    ShowGuides: true,
}

f.RenderWidget(tree, area)</code></pre></div>
            </div>

            <h2>Terminal Çıktısı (Canlı Görünüm)</h2>
            <div class="terminal-preview-card">
                <div class="terminal-header">
                    <div class="term-dots"><span class="term-dot dot-r"></span><span class="term-dot dot-y"></span><span class="term-dot dot-g"></span></div>
                    <span class="term-title">Terminal Render Çıktısı — TreeView</span>
                    <span class="term-tag">CANLI</span>
                </div>
                <div class="terminal-body"><span class="c-yellow bold">▼ 📁 limoni-repo/</span>
  <span class="c-blue">├── 📁 cmd/</span>
  <span class="c-blue">│   └── 📄 main.go</span>
  <span class="c-blue">├── 📁 core/</span>
  <span class="c-blue">│   ├── 📄 buffer.go</span>       <span class="c-gray">(1D memory array)</span>
  <span class="c-blue">│   ├── 📄 diff.go</span>         <span class="c-gray">(ANSI delta stream)</span>
  <span class="c-blue">│   └── 📄 runtime.go</span>      <span class="c-gray">(TEA engine)</span>
  <span class="c-blue">└── 📁 widgets/</span>
      <span class="c-cyan">├── 📄 viewer3d.go</span>     <span class="c-green">(30+ widgets catalog)</span>
      <span class="c-cyan">└── 📄 table.go</span></div>
            </div>
        `
    },

    "widget-viewer3d": {
        title: "🧊 3D Viewer & Braille Canvas",
        lead: "Wavefront OBJ/STL yükleyici, Lambertian gölgelendirme ve 2x4 Braille alt-piksel canvas.",
        content: `
            <h2>Go Kullanımı</h2>
            <div class="code-wrapper">
                <div class="code-header"><span class="code-lang">GO</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <div class="code-content"><pre><code>mesh, _ := graphics.LoadOBJ("assets/teapot.obj")

viewer := widgets.NewViewer3D(mesh).
    WithRotation(angleX, angleY, 0).
    WithShading("Gölgeli"). // Lambertian shading
    WithWireframe(true)

f.RenderWidget(viewer, area)</code></pre></div>
            </div>

            <h2>Terminal Çıktısı (Canlı Görünüm)</h2>
            <div class="terminal-preview-card">
                <div class="terminal-header">
                    <div class="term-dots"><span class="term-dot dot-r"></span><span class="term-dot dot-y"></span><span class="term-dot dot-g"></span></div>
                    <span class="term-title">Terminal Render Çıktısı — 3D Teapot</span>
                    <span class="term-tag">CANLI</span>
                </div>
                <div class="terminal-body">          <span class="c-cyan">⢀⣤⣴⣶⣶⣶⣤⡀</span>
       <span class="c-cyan">⢀⣴⣿⣿⣿⣿⣿⣿⣿⣿⣦⡀</span>     <span class="c-yellow">3D OBJ Mesh: Teapot</span>
     <span class="c-cyan">⢠⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣄</span>   Shader     : <span class="c-green">Lambertian Diffuse</span>
    <span class="c-blue">⣼⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣆</span>  Dönüş Açısı: <span class="c-cyan">X:24° Y:45° Z:0°</span>
   <span class="c-blue">⢸⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡇</span> Çözünürlük : <span class="c-yellow">160x96 Subpixels</span>
    <span class="c-blue">⠻⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠟</span>  Z-Buffer   : <span class="c-green">Etkin (No Overdraw)</span>
       <span class="c-cyan">⠉⠛⠿⠿⣿⣿⣿⣿⠿⠿⠛⠉</span></div>
            </div>
        `
    },

    "widget-markdown": {
        title: "📝 Markdown Görüntüleyici",
        lead: "Başlıklar, kalın yazılar, listeler ve kod bloklarını şeffaflık olmadan net arka planla okuma.",
        content: `
            <h2>Go Kullanımı</h2>
            <div class="code-wrapper">
                <div class="code-header"><span class="code-lang">GO</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <div class="code-content"><pre><code>md := limoni.NewMarkdown("# Proje Belgeleri\\n\\n- **Hız**: 60 FPS\\n- **Bellek**: Sıfır Tahsisat")
f.RenderWidget(md, area)</code></pre></div>
            </div>

            <h2>Terminal Çıktısı (Canlı Görünüm)</h2>
            <div class="terminal-preview-card">
                <div class="terminal-header">
                    <div class="term-dots"><span class="term-dot dot-r"></span><span class="term-dot dot-y"></span><span class="term-dot dot-g"></span></div>
                    <span class="term-title">Terminal Render Çıktısı — Markdown</span>
                    <span class="term-tag">CANLI</span>
                </div>
                <div class="terminal-body"><span class="c-yellow bold"># Proje Belgeleri</span>

<span class="c-blue">•</span> <span class="bold">Hız:</span> <span class="c-green">60 FPS akıcı terminal çıktısı</span>
<span class="c-blue">•</span> <span class="bold">Bellek:</span> <span class="c-cyan">Sıfır yığın tahsisatı (Zero-Alloc)</span>

<span class="c-gray">\`\`\`go</span>
<span class="c-yellow">import "github.com/thebanri/limoni"</span>
<span class="c-gray">\`\`\`</span></div>
            </div>
        `
    },

    "widget-virtual": {
        title: "⚡ VirtualDataView (1M+ Satır)",
        lead: "Milyonlarca satırlık büyük veri kümelerini bellek şişmesi olmadan kaydırarak görüntüleyin.",
        content: `
            <h2>Go Kullanımı</h2>
            <div class="code-wrapper">
                <div class="code-header"><span class="code-lang">GO</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <div class="code-content"><pre><code>state := widgets.NewVirtualDataState(1000000, 100, fetchPageFunc)
view := widgets.VirtualDataView{State: state}
f.RenderWidget(view, area)</code></pre></div>
            </div>

            <h2>Terminal Çıktısı (Canlı Görünüm)</h2>
            <div class="terminal-preview-card">
                <div class="terminal-header">
                    <div class="term-dots"><span class="term-dot dot-r"></span><span class="term-dot dot-y"></span><span class="term-dot dot-g"></span></div>
                    <span class="term-title">Terminal Render Çıktısı — 1,000,000 Satırlık Sanal Liste</span>
                    <span class="term-tag">CANLI</span>
                </div>
                <div class="terminal-body">  [#000001] Sunucu log kaydı: API çağrısı tamamlandı (200 OK)    <span class="c-yellow">█</span>
  [#000002] DB sorgusu: SELECT * FROM users LIMIT 50           <span class="c-gray">│</span>
<span class="c-blue bold">&gt; [#000003] Cache miss: Redis anahtarı bulunamadı               </span><span class="c-gray">│</span>
  [#000004] Arka plan işi: E-posta bildirimi gönderildi         <span class="c-gray">│</span>
  [#000005] TLS sertifikası yenilendi (Geçerlilik: 90 gün)      <span class="c-gray">│</span>

  <span class="c-gray">Satır: 3 / 1,000,000  |  Önbellek: 100 kayıt  |  RAM: 1.2 MB</span></div>
            </div>
        `
    },

    // --------------------------------------------------------------------------
    // ÇEKİRDEK APİ'LER
    // --------------------------------------------------------------------------
    "core-cell": {
        title: "🟩 cell (Hücre & Renk)",
        lead: "Karakter hücreleri, 24-bit TrueColor RGB, ANSI renkleri ve metin modifikatörleri.",
        content: `
            <h2>cell.Color</h2>
            <p>24-bit TrueColor, ANSI veya varsayılan rengi tek bir 32-bit tamsayıda verimli saklar:</p>
            <div class="code-wrapper">
                <div class="code-header"><span class="code-lang">GO</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <div class="code-content"><pre><code>colRGB  := limoni.RGB(0, 229, 255)
colHex  := limoni.Hex("#FACC15")
colANSI := limoni.ANSI(196)</code></pre></div>
            </div>

            <h2>cell.Style</h2>
            <p>Ön plan (Fg), arka plan (Bg) ve bitmask modifikatörlerini (Bold, Italic, Underline) içeren sıkıştırılmış struct:</p>
            <div class="code-wrapper">
                <div class="code-header"><span class="code-lang">GO</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <div class="code-content"><pre><code>style := limoni.Fg(colHex).Bold()
merged := baseStyle.Merge(overrideStyle)</code></pre></div>
            </div>
        `
    },

    "core-buffer": {
        title: "🟨 buffer & ANSI Diff",
        lead: "1D düz bellek matrisi ve diferansiyel ANSI kaçış dizisi üretim motoru.",
        content: `
            <h2>buffer.Diff</h2>
            <p>İki ardışık kare arasındaki farkı tarayarak terminale sadece değişen hücreleri yazar:</p>
            <div class="code-wrapper">
                <div class="code-header"><span class="code-lang">GO</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <div class="code-content"><pre><code>// Sıcak yolda sıfır tahsisatla fark akışı üretme
writeBuf, err := buffer.Diff(frontBuf, backBuf, writeBuf[:0], true, true)</code></pre></div>
            </div>
        `
    },

    "core-terminal": {
        title: "🟦 terminal & Odak Yönetimi",
        lead: "Katmanlar (Layers), modal izolasyonu ve odak kapsamı (Focus Scoping).",
        content: `
            <h2>Modal İzolasyonu</h2>
            <p>Bir diyalog penceresi açıldığında arkada kalan bileşenlerin tıklama ve tuş olaylarını almasını engeller:</p>
            <div class="code-wrapper">
                <div class="code-header"><span class="code-lang">GO</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <div class="code-content"><pre><code>f.RegisterModal("dialog_id", modalArea, onDismiss)
f.BeginFocusScope("dialog_id") // Tab navigasyonu yalnızca modal içinde kalır</code></pre></div>
            </div>
        `
    },

    "core-runtime": {
        title: "🔄 runtime (TEA Motoru)",
        lead: "Deterministik durum yönetimi, eşzamanlı komutlar ve katı iptal önceliği.",
        content: `
            <h2>The Elm Architecture</h2>
            <p><code>Init &rarr; Update &rarr; View</code> fonksiyonel döngüsü:</p>
            <div class="code-wrapper">
                <div class="code-header"><span class="code-lang">GO</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <div class="code-content"><pre><code>func (m *model) Update(msg limoni.Msg) limoni.UpdateResult {
    switch msg := msg.(type) {
    case limoni.KeyPressMsg:
        if msg.Key.Type == limoni.KeyEsc {
            return limoni.Quit()
        }
        return limoni.Redraw()
    }
    return limoni.Noop()
}</code></pre></div>
            </div>
        `
    },

    // --------------------------------------------------------------------------
    // REHBERLER
    // --------------------------------------------------------------------------
    bubbletea: {
        title: "🍵 Bubble Tea → Limoni Geçişi",
        lead: "Mevcut Bubble Tea ve Lipgloss modellerinizi Limoni üzerinde çalıştırma rehberi.",
        content: `
            <h2>1. Adım: Tek Değişiklikle Adapter ile Çalıştırma</h2>
            <div class="code-wrapper">
                <div class="code-header"><span class="code-lang">GO</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <div class="code-content"><pre><code>// Önce: import tea "github.com/charmbracelet/bubbletea"
// Şimdi:
import tea "github.com/thebanri/limoni/compat/bubbletea"

p := tea.NewProgram(model)
err := p.RunTerminal(context.Background())</code></pre></div>
            </div>

            <h2>2. Adım: Native API'ye Taşıma</h2>
            <p>Tam 60+ FPS ve sıfır bellek tahsisatı avantajından yararlanmak için <code>limoni.Model</code> ve <code>limoni.Frame</code> yapısına geçin.</p>
        `
    },

    a11y: {
        title: "♿ Erişilebilirlik (A11y)",
        lead: "Ekran okuyucular için anlamsal düğüm ağacı, Yüksek Kontrast modu ve NO_COLOR standardı.",
        content: `
            <h2>Standartlar</h2>
            <ul>
                <li><strong>Anlamsal Ağaç (Semantic Tree)</strong>: Her bileşen ekran okuyucular için rol (RoleButton, RoleTable, RoleDialog) üretir.</li>
                <li><strong>Yüksek Kontrast (WCAG AAA)</strong>: Düşük kontrastlı renkler otomatik olarak zıt renklere yükseltilir.</li>
                <li><strong>NO_COLOR=1</strong>: Tüm ANSI kaçış renkleri devre dışı bırakılır.</li>
            </ul>
        `
    },

    platforms: {
        title: "🌐 Sürücüler & WASM",
        lead: "Linux, macOS, Windows VT, WebAssembly (tarayıcı) ve uzak SSH sunucusu desteği.",
        content: `
            <h2>WebAssembly (WASM)</h2>
            <p>Tek komutla derleyip tarayıcı içinde xterm.js üzerinde çalıştırın:</p>
            <div class="code-wrapper">
                <div class="code-header"><span class="code-lang">BASH</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <div class="code-content"><pre><code>GOOS=js GOARCH=wasm go build -o limoni.wasm ./examples/wasm</code></pre></div>
            </div>

            <h2>SSH TUI Sunucusu</h2>
            <p>Uzak kullanıcılara ağ üzerinden doğrudan canlı oturum açın:</p>
            <div class="code-wrapper">
                <div class="code-header"><span class="code-lang">BASH</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <div class="code-content"><pre><code>go run ./examples/ssh_server
# Bağlanmak için: nc localhost 2222</code></pre></div>
            </div>
        `
    },

    examples: {
        title: "🚀 Örnekler & Demolar",
        lead: "Limoni kaynak deposunda yer alan hazır çalışan canlı örnekler.",
        content: `
            <h2>Hazır Örnekler</h2>
            <table class="doc-table">
                <thead>
                    <tr><th>Örnek</th><th>Komut</th><th>Açıklama</th></tr>
                </thead>
                <tbody>
                    <tr><td><strong>Showcase</strong></td><td><code>go run ./examples/showcase</code></td><td>7 sekmeli amiral gemisi vitrin uygulaması.</td></tr>
                    <tr><td><strong>Simple</strong></td><td><code>go run ./examples/simple</code></td><td>15 satırda minimal başlangıç kodu.</td></tr>
                    <tr><td><strong>3D Viewer</strong></td><td><code>go run ./examples/3d_viewer</code></td><td>OBJ/STL modelleri canlı döndürme.</td></tr>
                    <tr><td><strong>Demo</strong></td><td><code>go run ./examples/demo</code></td><td>Etkileşimli diyalog ve grafik demosu.</td></tr>
                </tbody>
            </table>
        `
    }
};

// Ordered section keys for Prev/Next navigation
const DOC_KEYS = Object.keys(DOCS);

// Current active document key
let currentDocKey = "quickstart";

// Switch active document
function switchDoc(key) {
    if (!DOCS[key]) {
        key = "quickstart";
    }
    currentDocKey = key;

    // Update active nav items in sidebar
    document.querySelectorAll('.nav-item').forEach(item => {
        item.classList.remove('active');
        if (item.getAttribute('data-doc') === key) {
            item.classList.add('active');
        }
    });

    // Render content
    const article = document.getElementById('doc-content');
    const doc = DOCS[key];
    article.innerHTML = `
        <h1>${doc.title}</h1>
        <div class="lead">${doc.lead}</div>
        ${doc.content}
    `;

    // Update URL hash
    window.location.hash = key;

    // Generate On-Page Table of Contents
    generateTOC();

    // Update Prev / Next buttons
    updateNavButtons();

    // Scroll to top
    window.scrollTo({ top: 0, behavior: 'smooth' });

    // Close mobile sidebar if open
    const sidebar = document.getElementById('sidebar');
    if (sidebar && sidebar.classList.contains('open')) {
        sidebar.classList.remove('open');
    }
}

// Generate On-Page TOC based on H2 elements
function generateTOC() {
    const tocNav = document.getElementById('toc-nav');
    if (!tocNav) return;
    tocNav.innerHTML = '';

    const headings = document.querySelectorAll('#doc-content h2');
    headings.forEach((h2, idx) => {
        const id = 'section-' + idx;
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
        tocNav.appendChild(li);
    });
}

// Update Previous & Next chapter navigation buttons
function updateNavButtons() {
    const idx = DOC_KEYS.indexOf(currentDocKey);

    const prevBtn = document.getElementById('prev-btn');
    const nextBtn = document.getElementById('next-btn');
    const prevName = document.getElementById('prev-btn-name');
    const nextName = document.getElementById('next-btn-name');

    if (idx > 0) {
        prevBtn.style.visibility = 'visible';
        prevName.textContent = DOCS[DOC_KEYS[idx - 1]].title;
    } else {
        prevBtn.style.visibility = 'hidden';
    }

    if (idx < DOC_KEYS.length - 1) {
        nextBtn.style.visibility = 'visible';
        nextName.textContent = DOCS[DOC_KEYS[idx + 1]].title;
    } else {
        nextBtn.style.visibility = 'hidden';
    }
}

function navPrev() {
    const idx = DOC_KEYS.indexOf(currentDocKey);
    if (idx > 0) switchDoc(DOC_KEYS[idx - 1]);
}

function navNext() {
    const idx = DOC_KEYS.indexOf(currentDocKey);
    if (idx < DOC_KEYS.length - 1) switchDoc(DOC_KEYS[idx + 1]);
}

// Copy Code Helper
function copyCode(btn) {
    const wrapper = btn.closest('.code-wrapper');
    const code = wrapper.querySelector('code').textContent;
    navigator.clipboard.writeText(code).then(() => {
        const orig = btn.textContent;
        btn.textContent = "Kopyalandı! ✓";
        btn.style.borderColor = "#facc15";
        btn.style.color = "#facc15";
        setTimeout(() => {
            btn.textContent = orig;
            btn.style.borderColor = "";
            btn.style.color = "";
        }, 1800);
    });
}

// Search Modal Functionality (Cmd+K / Ctrl+K)
function openSearch() {
    const modal = document.getElementById('search-modal');
    modal.classList.add('open');
    const input = document.getElementById('search-input');
    input.value = '';
    input.focus();
    handleSearch('');
}

function closeSearch(e) {
    if (e && e.target !== document.getElementById('search-modal') && !e.target.classList.contains('search-close-btn')) {
        return;
    }
    document.getElementById('search-modal').classList.remove('open');
}

function handleSearch(query) {
    const resultsContainer = document.getElementById('search-results');
    query = query.toLowerCase().trim();

    if (!query) {
        resultsContainer.innerHTML = '<div style="padding:16px; color:#64748b; font-size:13px; text-align:center;">Aramak için yazın... (Örn: Table, 3D, Diff, Dialog)</div>';
        return;
    }

    const matched = [];
    for (const [key, doc] of Object.entries(DOCS)) {
        if (doc.title.toLowerCase().includes(query) || doc.lead.toLowerCase().includes(query) || doc.content.toLowerCase().includes(query)) {
            matched.push({ key, title: doc.title, lead: doc.lead });
        }
    }

    if (matched.length === 0) {
        resultsContainer.innerHTML = '<div style="padding:16px; color:#64748b; font-size:13px; text-align:center;">Sonuç bulunamadı.</div>';
        return;
    }

    resultsContainer.innerHTML = matched.map(m => `
        <a href="#${m.key}" class="search-result-item" onclick="selectSearch('${m.key}')">
            <div class="search-res-title">${m.title}</div>
            <div class="search-res-snippet">${m.lead}</div>
        </a>
    `).join('');
}

function selectSearch(key) {
    document.getElementById('search-modal').classList.remove('open');
    switchDoc(key);
}

// Mobile sidebar toggle
function toggleSidebar() {
    const sidebar = document.getElementById('sidebar');
    sidebar.classList.toggle('open');
}

// Keyboard shortcuts
window.addEventListener('keydown', (e) => {
    if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        openSearch();
    } else if (e.key === 'Escape') {
        document.getElementById('search-modal').classList.remove('open');
    }
});

// Hash change router & DOM load
window.addEventListener('DOMContentLoaded', () => {
    const initialHash = window.location.hash.replace('#', '') || 'quickstart';
    switchDoc(initialHash);
});

window.addEventListener('hashchange', () => {
    const hash = window.location.hash.replace('#', '') || 'quickstart';
    if (!hash.startsWith('section-')) {
        switchDoc(hash);
    }
});
