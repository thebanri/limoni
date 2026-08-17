// ==========================================================================
// Limoni Documentation Engine (Vanilla JS)
// ==========================================================================

const DOCS_DATA = {
    quickstart: {
        title: "⚡ Hızlı Başlangıç (Quick Start)",
        content: `
            <h1>🚀 Hızlı Başlangıç</h1>
            <p>Limoni, Go dili için <strong>sıfır-tahsisatlı (Zero-Allocation)</strong>, <strong>60+ FPS</strong> yüksek performanslı ve 3D grafik destekli modern bir Terminal Kullanıcı Arayüzü (TUI) kütüphanesidir.</p>

            <div class="cards-grid">
                <div class="feature-card">
                    <div class="card-icon">⚡</div>
                    <div class="card-title">0 B/op Sıfır Tahsisat</div>
                    <p class="card-desc">Render sıcak yolunda GC yükü yoktur, mikro gecikme oluşmaz.</p>
                </div>
                <div class="feature-card">
                    <div class="card-icon">🧊</div>
                    <div class="card-title">3D Shader & Mesh</div>
                    <p class="card-desc">OBJ/STL/PLY desteği, Lambertian Diffuse ve Gouraud gölgelendirme.</p>
                </div>
                <div class="feature-card">
                    <div class="card-icon">🔄</div>
                    <div class="card-title">The Elm Architecture</div>
                    <p class="card-desc">Öngörülebilir durum yönetimi (Model, Msg, Cmd, Update, View).</p>
                </div>
            </div>

            <h2>📦 Kurulum</h2>
            <p>Go 1.22+ yüklü projenizde Limoni'yi bağımlılık olarak ekleyin:</p>
            <div class="code-box">
                <div class="code-header"><span>BASH</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <pre><code>go get github.com/thebanri/limoni</code></pre>
            </div>

            <h2>⚡ İlk Sayaç (Counter) Uygulamanız</h2>
            <p>Aşağıdaki kodu <code>main.go</code> dosyasına yapıştırıp <code>go run main.go</code> ile anında çalıştırabilirsiniz:</p>

            <div class="code-box">
                <div class="code-header"><span>MAIN.GO</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <pre><code>package main

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

type CounterModel struct {
    Count int
}

func (m CounterModel) Init() runtime.Cmd { return nil }

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
            }
        }
    }
    return m, nil
}

func (m CounterModel) View(f *terminal.Frame) {
    area := f.Area()
    chunks := layout.FlexLayout{
        Direction: layout.Vertical,
        Constraints: []layout.Constraint{
            layout.Fixed(3),
            layout.Fill(),
            layout.Fixed(3),
        },
    }.Split(area)

    // Başlık
    f.RenderWidget(widgets.Block{
        Title:       " 🍋 LIMONI SAYAC ",
        TitleStyle:  cell.Style{Fg: cell.NewColorRGB(255, 215, 0), Modifier: cell.ModifierBold},
        BorderStyle: cell.Style{Fg: cell.NewColorRGB(0, 200, 255)},
    }, chunks[0])

    // Gövde
    f.RenderWidget(&widgets.Paragraph{
        Text:  fmt.Sprintf("Mevcut Değer: %d", m.Count),
        Style: cell.Style{Fg: cell.NewColorRGB(0, 255, 200), Modifier: cell.ModifierBold},
    }, chunks[1])

    // Kısayollar
    f.RenderWidget(widgets.Block{
        Title:       " [+] Artır  [-] Azalt  [Q/Esc] Çıkış ",
        BorderStyle: cell.Style{Fg: cell.NewColorRGB(100, 110, 120)},
    }, chunks[2])
}

func main() {
    b := backend.NewBackend(os.Stdin, os.Stdout)
    b.Setup()
    defer b.Close()

    term, _ := terminal.New(b)
    p := runtime.New(runtime.WithModel(CounterModel{Count: 0}), runtime.WithFPS(60))
    p.RunTerminal(context.Background(), term, b)
}</code></pre>
            </div>
        `
    },

    architecture: {
        title: "🏛️ Mimari ve Sıfır-Tahsisat (Architecture)",
        content: `
            <h1>🏛️ Mimari ve Sıfır-Tahsisat Felsefesi</h1>
            <p>Limoni, bellek dostu mimarisi ve CPU L1/L2 önbellek verimliliği sayesinde mikro gecikmeleri ortadan kaldırır.</p>

            <h2>1. 1D Düz Bellek Izgarası ([]cell.Cell)</h2>
            <p>Geleneksel matrisler <code>[][]Cell</code> her satır için ayrı pointer tutar ve cache miss yaratır. Limoni, tüm ekranı ardışık tek bir 1D dilimde saklar:</p>
            <div class="code-box">
                <div class="code-header"><span>GOLANG</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <pre><code>// İndeks Hesaplama: y * Width + x
cell := buf.Content[y*buf.Area.Width + x]</code></pre>
            </div>

            <h2>2. Çift Tamponlu ANSI Diff Motoru</h2>
            <p>Front Buffer ve Back Buffer arasındaki farklar (diff) taranarak yalnızca değişen karakterler minimum ANSI kaçış koduyla fiziksel terminale yazılır. <code>\x1b[?2026h</code> senkronize yenileme protokolü ile ekran titremesi (flicker) tamamen engellenir.</p>
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
                    <p class="card-desc"><strong>11.5 ns/op</strong> — 0 B/op (0 allocs/op)</p>
                </div>
                <div class="feature-card">
                    <div class="card-title">Metin Çerçevesi (Text Heavy)</div>
                    <p class="card-desc"><strong>4.8 µs/op</strong> — 0 B/op (0 allocs/op)</p>
                </div>
                <div class="feature-card">
                    <div class="card-title">10.000 Satırlı Tablo</div>
                    <p class="card-desc"><strong>41.2 µs/op</strong> — 0 B/op (0 allocs/op)</p>
                </div>
                <div class="feature-card">
                    <div class="card-title">100 Katman Z-Index</div>
                    <p class="card-desc"><strong>40.1 ns/op</strong> — 0 B/op (0 allocs/op)</p>
                </div>
            </div>
        `
    },

    "core-cell": {
        title: "🟩 core/cell — Hücre & Stil API",
        content: `
            <h1>🟩 core/cell Paketi</h1>
            <p>Terminal ekranındaki en küçük birim olan karakter hücresi, 24-bit TrueColor RGB renkleri ve stil modifikatörlerini tanımlar.</p>

            <h2>Renk Tanımlama</h2>
            <div class="code-box">
                <div class="code-header"><span>GOLANG</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <pre><code>// 24-bit TrueColor RGB
colRGB := cell.NewColorRGB(255, 180, 0)

// 8-bit Standart ANSI (0-255)
colANSI := cell.NewColorANSI(196)

// Varsayılan Terminal Rengi
colDef := cell.NewColorDefault()</code></pre>
            </div>

            <h2>Stil ve Modifikatörler</h2>
            <div class="code-box">
                <div class="code-header"><span>GOLANG</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <pre><code>style := cell.Style{
    Fg: cell.NewColorRGB(0, 255, 200),
    Bg: cell.NewColorRGB(20, 25, 40),
    Modifier: cell.ModifierBold | cell.ModifierUnderline,
}</code></pre>
            </div>
        `
    },

    "core-buffer": {
        title: "🟨 core/buffer — 1D Tampon & Diff API",
        content: `
            <h1>🟨 core/buffer Paketi</h1>
            <p>Terminal hücresel ızgarasını 1D ardışık dizide saklar ve ekran diff algoritmalarını yürütür.</p>

            <h2>Kullanım Örneği</h2>
            <div class="code-box">
                <div class="code-header"><span>GOLANG</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <pre><code>buf := buffer.NewBuffer(cell.NewRect(0, 0, 80, 24))
buf.Clear()

// Hücre yaz
buf.SetCell(10, 5, cell.Cell{
    Content: '🍋',
    Style: cell.Style{Fg: cell.NewColorRGB(255, 215, 0)},
})

// Metin dizgisi yaz
buf.SetString(10, 6, "Sıfır-Tahsisat Motoru", cell.Style{Modifier: cell.ModifierBold})</code></pre>
            </div>
        `
    },

    "core-terminal": {
        title: "🟦 core/terminal — Terminal & Frame API",
        content: `
            <h1>🟦 core/terminal Paketi</h1>
            <p>Çift tampon yönetimini, kare çizim döngüsünü ve fare/klavye olaylarının widget'lara dağıtılmasını sağlar.</p>

            <h2>Frame Çizim Döngüsü</h2>
            <div class="code-box">
                <div class="code-header"><span>GOLANG</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <pre><code>term.Draw(func(f *terminal.Frame) {
    area := f.Area()
    f.RenderWidget(myWidget, area)
})</code></pre>
            </div>
        `
    },

    "core-runtime": {
        title: "🔄 core/runtime — The Elm Architecture (TEA)",
        content: `
            <h1>🔄 core/runtime Paketi</h1>
            <p>Limoni'nin durum yönetim motorudur. Model, Init, Update ve View döngüsüyle deterministik TUI uygulamaları kurmanızı sağlar.</p>
        `
    },

    layout: {
        title: "📐 layout — Esnek Flexbox Yerleşim Motoru",
        content: `
            <h1>📐 layout Paketi</h1>
            <p>Ekran alanını yatay (Horizontal) veya dikey (Vertical) olarak sabit, yüzdelik veya oranlı kısıtlamalarla böler.</p>

            <h2>Kısıtlamalar Listesi</h2>
            <ul>
                <li><code>layout.Fixed(N)</code>: Sabit N satır veya sütun ayırır.</li>
                <li><code>layout.Percentage(P)</code>: Toplam alanın %P kadarını ayırır (0-100).</li>
                <li><code>layout.Ratio(R)</code>: Kalan serbest alanı ağırlıklı oranlarla böler.</li>
                <li><code>layout.Fill()</code>: Kalan tüm boşluğu kaplar.</li>
                <li><code>layout.Min(N)</code> / <code>layout.Max(N)</code>: Min/Max sınırları belirler.</li>
            </ul>
        `
    },

    "widgets-display": {
        title: "📦 widgets — Görsel & Tablo Bileşenleri",
        content: `
            <h1>📦 Görsel & Bilgi Widget'ları</h1>
            <p>Limoni; Block, Paragraph, Table (Sanal 1M+ satır), Sparkline, ProgressBar ve List gibi zengin bileşenler sunar.</p>

            <h2>Tablo (Table) Tanımlama</h2>
            <div class="code-box">
                <div class="code-header"><span>GOLANG</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <pre><code>table := &widgets.Table{
    Header: &widgets.TableRow{
        Cells: []widgets.TableCell{
            {Text: "SIRA"},
            {Text: "PROSES"},
            {Text: "CPU%"},
        },
    },
    Rows: myRows,
    Constraints: []widgets.TableConstraint{
        {Type: widgets.ConstraintFixed, Value: 8},
        {Type: widgets.ConstraintFill},
        {Type: widgets.ConstraintFixed, Value: 10},
    },
    DrawGrid: true,
}
f.RenderWidget(table, area)</code></pre>
            </div>
        `
    },

    "widgets-inputs": {
        title: "✍️ Formlar & Girdi Kutuları",
        content: `
            <h1>✍️ Formlar & Girdi Kutuları</h1>
            <p><code>TextInput</code>, <code>TextArea</code>, <code>Checkbox</code>, <code>RadioGroup</code>, <code>Select</code> ve <code>Slider</code> kontrolleri ile interaktif formlar oluşturun.</p>
        `
    },

    "widgets-modals": {
        title: "🪟 Modallar & Katmanlar",
        content: `
            <h1>🪟 Modallar & Katmanlar</h1>
            <p>Limoni Z-Index derinlik motoru sayesinde arka planı karartan (overlay) ve klavye odağını içine hapseden açılır diyalog pencereleri ve popuplar çizebilirsiniz.</p>
        `
    },

    "widgets-custom": {
        title: "🛠️ Özel Widget Geliştirme",
        content: `
            <h1>🛠️ Özel Widget Geliştirme</h1>
            <p>Kendi widget'ınızı oluşturmak için yalnızca <code>widgets.Widget</code> arayüzünü (Draw ve SizeHint metodları) uygulamanız yeterlidir.</p>

            <div class="code-box">
                <div class="code-header"><span>GOLANG</span><button class="copy-btn" onclick="copyCode(this)">Kopyala</button></div>
                <pre><code>type MyGauge struct {
    Value float64
}

func (g *MyGauge) Draw(ctx cell.Context, buf *buffer.Buffer) {
    buf.SetString(ctx.Area.X, ctx.Area.Y, fmt.Sprintf("Değer: %.1f", g.Value), cell.Style{
        Fg: cell.NewColorRGB(255, 215, 0),
        Modifier: cell.ModifierBold,
    })
}

func (g *MyGauge) SizeHint(maxArea cell.Rect) (uint16, uint16) {
    return 20, 1
}</code></pre>
            </div>
        `
    },

    "graphics-3d": {
        title: "🧊 3D Mesh & Shader Motoru",
        content: `
            <h1>🧊 3D Mesh & Shader Motoru</h1>
            <p>Limoni, terminalde 3D modelleri (.obj, .stl, .ply) yükleyebilir, fareyle serbest döndürebilir ve gerçek zamanlı gölgelendirebilir.</p>

            <h2>Desteklenen Gölgelendiriciler (Shaders)</h2>
            <ul>
                <li><strong>Tel Kafes (Wireframe)</strong>: Kenar çizgileriyle model iskeleti.</li>
                <li><strong>Lambertian Diffuse</strong>: Yüzey normalleri ve ışık kaynağı ile gerçekçi aydınlatma.</li>
                <li><strong>Gouraud Shaded</strong>: Köşeler arası pürüzsüz RGB enterpolasyonu.</li>
                <li><strong>Doku Kaplama (Texture Mapping)</strong>: PNG/prosedürel doku resimlerinin UV koordinatlarıyla giydirilmesi.</li>
            </ul>
        `
    },

    "graphics-canvas": {
        title: "🎨 2D Braille Canvas",
        content: `
            <h1>🎨 2D Braille Canvas</h1>
            <p>Braille Unicode ızgarası (2x4) ile her terminal hücresinde 8 alt-piksel çözünürlük sunarak çemberler, çizgiler ve grafikler çizer.</p>
        `
    },

    animation: {
        title: "🎬 Animasyon & Fizik Motoru",
        content: `
            <h1>🎬 Animasyon & Fizik Motoru</h1>
            <p><code>animation.Float</code>, <code>animation.Color</code> ve Easing eğrileri (Quad, Cubic, Bounce, Elastic) ile 60 FPS akıcı arayüz geçişleri sağlar.</p>
        `
    },

    platforms: {
        title: "🌐 WASM & SSH Çapraz Platform Sürücüleri",
        content: `
            <h1>🌐 WASM & SSH Sürücüleri</h1>
            <p>Linux, macOS, Windows VT100, WebAssembly (xterm.js ile tarayıcı) ve Uzak SSH/TCP sunucularında sıfır ek kütüphane bağımlılığıyla çalışır.</p>
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
        resultsContainer.innerHTML = '<div class="search-hint">Aramak için yazmaya başlayın... (Örn: Table, 3D, FlexLayout)</div>';
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
