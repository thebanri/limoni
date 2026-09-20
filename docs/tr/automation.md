# Semantik Otomasyon

Bugün bir terminal uygulamasını otomatikleştiren her araç — [termwright](https://github.com/fcoury/termwright), [mcp-tui-test](https://github.com/GeorgePearse/mcp-tui-test) — süreci bir sözde terminale sarıp çizilen karakter ızgarasını parse eder. Başka çareleri yoktur: alttaki uygulamanın sunacak bir semantiği yoktur. Dolayısıyla test, bir metnin bir koordinatta olduğunu doğrular ve düzen bir sütun kaydığı anda kırılır.

Limoni zaten ekran okuyucular için her karede bir semantik ağaç kuruyor. `WithAutomation` aynı ağacı bir Unix soketinde sunar; böylece

```
"Submit" metni 42,7'de mi?  →  42,7'ye tıkla
```

yerine

```go
client.Click(automation.Selector{Role: "button", Label: "Submit"})
```

yazarsınız. Seçici yeniden düzenlemeye, boyutlandırmaya ve stil değişimine dayanır, çünkü hiçbir şeyin nerede çizildiğinden söz etmez.

```go
// Uygulama açık bir politikayla ister. -tags limoni_debug ile derleyin.
limoni.Run(draw, limoni.WithAutomation("/run/user/1000/myapp.sock", limoni.AutomationPolicy{
	AllowInput:   true, // varsayılan kapalı: olmadan istemci yalnızca gözlemleyebilir
	ExposeScreen: true, // varsayılan kapalı: ızgara ekrandaki her karakteri içerir
}))
```

```go
// Test ya da ajan sürer.
client, _ := automation.Dial("/run/user/1000/myapp.sock")
defer client.Close()

list, _ := client.WaitFor(automation.Selector{Role: "list"}, 3*time.Second)
// list.Value == "beta", list.Position == 2, list.SetSize == 3

client.Key("down")
client.Type("hello")
client.Click(automation.Selector{Role: "button", Label: "Submit"})

screen, _ := client.Screen() // ağacın ifade edemediği doğrulamalar için ham ızgara
```

Protokol satır ayrımlı JSON, yani hata ayıklarken `socat` kullanılabilir bir istemcidir. Belirsiz bir seçici **hatadır**, yazı tura değil — iki eşleşen düğmeden sessizce ilkini seçen bir test, ikincisi eklendiği anda yanlış sebeple geçer; hangisini kastettiğinizi `Nth` ile söyleyin.

## Bir yapay zekâ ajanıyla sürmek (MCP)

`cmd/limoni-mcp` aynı ağacı [Model Context Protocol](https://modelcontextprotocol.io) konuşan her ajanın önüne koyar — Claude Code, Claude Desktop, Cursor ve diğerleri. Standart kütüphane dışında bağımlılığı olmayan bir köprüdür: bir yanda stdio üzerinden MCP, öbür yanda uygulamanın soketi.

```bash
go install github.com/thebanri/limoni/cmd/limoni-mcp@latest

# Bunun için yazılmış örnekte deneyin:
# Bir terminalde — uygulama $XDG_RUNTIME_DIR/limoni-checklist.sock üzerinde dinler:
go run -tags limoni_debug ./examples/agent_checklist
# Bir diğerinde:
claude mcp add limoni -- limoni-mcp -socket "$XDG_RUNTIME_DIR/limoni-checklist.sock"
```

Ajan sekiz araç alır: `tree`, `find`, `click`, `press_key`, `type_text`, `wait_for`, `screen` ve `status`. Pratikte iki ayrıntı önemli:

- **Girdi araçları, uygulama yeniden çizdikten sonraki ağacı döndürür.** Girdi asenkrondur — soket bir tuşu, yol açtığı kareden önce onaylar — bu yüzden `click` ağacın değişip durulmasını bekler ve onu döndürür. Ajan hiçbir zaman kendi tıklamasından *önceki* ekran üzerinden akıl yürütmez. Hiçbir şey değişmediyse sonuç bunu söyler, anlayabildiğinde de nedenini: alan gizlidir ya da politika girdi değerlerini saklıyordur.
- **Hatalar açıklamadır.** Belirsiz bir seçici, yanlış yazılmış bir argüman ya da politika reddi, modelin üzerine iş yapabileceği bir metin olarak döner — `label="Remove" matches 2 nodes; set nth to choose one` — protokol hatası olarak değil.

Köprü aşağıdaki sınırların hiçbirini değiştirmez: yalnızca uygulamanın politikasının izin verdiğini yapabilir ve port açmaz. Girdi araçları yıkıcı (destructive) olarak işaretlidir; yan etkiden önce soran bir istemci soracaktır.

Bir denemede Claude Code 2.1.270, yalnızca hedef söylenerek, bir görev ekledi, üçünü işaretledi, bir deploy token'ı yazdı ve 17 araç çağrısında deploy etti. Denemenin kaydındaki hiçbir araç sonucunda token geçmiyor. Bu bir denemedir, benchmark değil.

## Playwright gibi test etmek (`uitest`)

Aynı ağaç, Playwright'ın web sayfaları için yazdığı testlere benzeyen testler yazmayı sağlar. `uitest` widget'ları rol, etiket ve ID ile bulur, onlar üzerinde işlem yapar ve **bekleyen** kontrollerle doğrular: bir işlem, konumlandırıcısı tam olarak bir widget'la eşleşene kadar bekler; bir doğrulama tutana kadar yeniden dener. Böylece test hiçbir zaman `sleep` kullanmaz. Bir kontrol başarısız olduğunda mesaj neyin beklendiğini, neyin görüldüğünü ve son karenin tüm semantik ağacını gösterir.

```go
func TestReleaseFlow(t *testing.T) {
	app := newChecklist()
	page := uitest.Run(t, 80, 24, app.draw) // süreç içinde: terminal yok, build tag yok

	page.GetByRole("input", "New task").Type("Tag v1.0")
	page.GetByRole("button", "Add task").Click()

	rows := page.GetByRole("list-item", "").Within(page.GetByRole("list", "Tasks"))
	page.Expect(rows).ToHaveCount(3)
	rows.Nth(-1).Click()
	page.Press("space")

	page.Expect(page.GetByID("status")).ToContainLabel("Added")
	page.Expect(page.GetByRole("dialog", "")).Not().ToBeVisible()
}
```

```
uitest: expected id="status" to have label "Deployed with 3 tasks complete.": got label "Blocked: the deploy token is empty." after 5s
last frame:
  input#new-task "New task" bounds=2,2 36x1
  button#add "Add task" bounds=40,2 14x1
  …
```

Her aksiyon loglanır; bir hata, ona götüren adımlarla birlikte gelir. `uitest.WithSlowMo` ise testi izleyen biri için yavaşlatır. `uitest.Connect` ile çalışan bir uygulamaya yöneltilen aynı test onu ekranda canlı sürer — [`examples/agent_checklist`](../../examples/agent_checklist) içindeki `TestLiveDemo` tam olarak bunu yapar.

Tek API, üç hedef: immediate-mode çizim fonksiyonu için `uitest.Run`, gerçek mesaj döngüsünden (komutlar dahil) geçen declarative model için `uitest.Program`, otomasyon soketi üzerinden çalışan bir binary için `uitest.Connect`. [`examples/agent_checklist`](../../examples/agent_checklist) bununla test ediliyor ve bir ajanın `limoni-mcp` üzerinden sürdüğü uygulamanın ta kendisi.

Listeler görünür satırlarını `list-item` alt düğümleri olarak sunar; bir satıra tuş basışı sayarak değil, metniyle ulaşılır.

Yapılandırılmış widget'lar görünen öğelerini alt düğüm olarak sunar, böylece bir öğeye tuşa kaç kez basıldığını saymadan metniyle ulaşılır: listenin satırları `list-item`, tablonun satırları `row` (ilk hücreyle etiketlenir, her sütun için bir `cell` alt düğümü vardır), sekme çubuğu `tab`'lardan oluşan bir `tab-list` (`State: &widgets.TabsState{}` verilmeli), ağacın görünen öğeleri ise açık/kapalı durumuyla `tree-item`.

Tıklama durumu tersine çevirir; iki kez çalışan bir adım kendini geri alır. `Check`, `Uncheck` ve `Select` yalnızca widget o durumda değilse tıklar ve o duruma gelmesini bekler:

```go
page.GetByID("agree").Check()               // zaten işaretliyse hiçbir şey yapmaz
page.GetByRole("row", "beta").Select()
page.GetByRole("tab", "Logs").Select()
```

MCP üzerinden aynısı `click` aracına `ensure: "checked" | "unchecked" | "selected"` vermektir.

> [!WARNING]
> **Bu, çalışan bir sürece kontrol kanalı açar.** Her biri kendi başına kapalı başarısız olan katmanlar hâlinde kuruldu.
>
> - **Release derlemelerinde yok.** Geçit yalnızca `-tags limoni_debug` ile derlenen binary'lerde bulunur. Etiket olmadan `WithAutomation`, `Run`'ın `ErrAutomationNotCompiled` döndürmesine yol açar ve soket sunucusu binary'de hiç yer almaz — CI bir release binary derleyip sembol tablosunda otomasyon kodu arıyor. Hiçbir yapılandırma hatası, orada olmayan kodu açamaz.
> - **Varsayılan kapalı.** Sıfır değerli `AutomationPolicy` yalnızca yapıyı gösterir: roller, etiketler, konumlar, sınırlar. Girdi değerleri, ekran görüntüsü ve girdi sentezi her biri kendi alanının açılmasını ister.
> - **Sırlar politika ne derse desin çıkmaz.** `TextInput{Secret: true}` her karakter yerine maske glifi çizer, yani sır hücre tamponuna hiç girmez; düğümü değer taşımaz ve hassas olarak işaretlenir. Geçit hassas değerleri yine de temizler, böylece bunu unutan bir widget sızdırmaz. Seçiciler redakte edilmiş ağaçta çözülür, yani bir istemci `value="…"` eşleşip eşleşmediğine bakarak parola tahmin edemez.
> - **Yalnızca sizin kullanıcınız bağlanabilir.** Linux, macOS ve FreeBSD'de sunucu, soketin 0600 izinlerine ek olarak bağlanan sürecin sahibini çekirdeğe sorar ve başka her kullanıcıyı reddeder. Çekirdeğin bunu söyleyemediği yerlerde — Windows dahil — dosya izinleri tek koruma kalacağı için `AllowUnverifiedPeers` açılmadıkça her bağlantı reddedilir.
> - **Yalnızca Unix soketi, TCP seçeneği yok.** Bilinçli ve yapılandırılamaz: bir port, uygulama kontrolünü makineye erişebilen her şeye açardı.
>
> **Kalan riskler, açıkça:** *aynı kullanıcı olarak* çalışan başka bir süreç yine bağlanabilir — işletim sistemi yerel bir sokette kullanıcıdan güçlü bir kimlik sunmuyor. Ve siz işaretlemedikçe geçit, çizdiğiniz bir paragrafın sır olduğunu bilemez; `ExposeScreen` ekranda ne varsa gönderir. Bir `limoni_debug` binary'sine hata ayıklama konsolu gibi davranın.
