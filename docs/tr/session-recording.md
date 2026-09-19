# Oturum Kaydı ve Yeniden Oynatma

*"Birkaç tuşa bastım, çöktü"* diyen bir hata raporuyla bir şey yapmak zordur. Oturumun kaydıyla değil. `session` paketi declarative bir uygulamayı kaydeder — modelin aldığı her mesajı, `Update`'in gördüğü sırayla, artı her karenin semantik ağacını — ve modele karşı yeniden oynatıp her kareyi doğrular.

```go
rec, _ := session.Create("hata.limoni", model, genislik, yukseklik, session.Policy{})
defer rec.Close()
limoni.RunProgram(ctx, model, limoni.WithProgramObserver(rec))
```

```go
// Sonra, bir testte: kayıt bir regresyon testine dönüşür.
report, err := session.Replay("hata.limoni", func() limoni.Model { return yeniModel() }, session.ReplayOptions{})
if err != nil { t.Fatal(err) }                       // kayda güvenilemedi
if !report.Verified() { t.Fatal(report.Divergence) } // "replay diverged at step 3 ..."
```

Oynatma **karesi farklılaşan ilk adımı** ve kaydedilmiş bir **çökmenin yeniden üretilip üretilmediğini** raporlar — yani aynı dosya önce hatanın hâlâ orada olduğunu, sonra düzeldiğini söyler.

Terminal yerine `Update` noktasında kaydeder; oynatılabilir olmasını sağlayan şey bu: canlı çalışmada zamanlayıcılar ve komutlar yarışır, `Update`'in çağrıldığı yerde kaydetmek bu yarışların nasıl sonuçlandığını dondurur. Bir komutun etkisi kayda ürettiği mesaj olarak girer, yani oynatma bir isteği yeniden göndermez ya da saati okumaz.

> [!WARNING]
> **Sınırlar, açıkça.**
>
> - **Yalnızca declarative mod.** Immediate mode `Run`'da uygulamayla yan etkileri arasında bir sınır yoktur, kaydedilecek bir nokta da yoktur.
> - **`Update` ve `View` deterministik olmalı.** İçlerinde `time.Now` ya da `math/rand` çağıran bir model oynatmada farklılaşır — oynatma bunu tespit edip adımı gösterir ama düzeltemez. Saati `limoni.NowCmd` ile mesaj olarak alın. [`tools/limonivet`](../../tools/limonivet) bu çağrıları raporlar; CI onu bu depoda çalıştırıyor.
> - **Yalnızca kayıtlı mesajlar.** Bir uygulama mesaj tipi `session.Register` ile kaydedilmedikçe yalnızca adıyla kaydedilir; oynatma ona ulaştığında atlamak yerine yüksek sesle başarısız olur.
>
> **Gizlilik varsayılan olarak kapalı.** Yazılan metin ve yapıştırmalar `RecordText` açılmadıkça `x` olarak yazılır; girdi alanı değerleri `ExposeInputValues` açılmadıkça kayıtlı ağaçlardan düşürülür; hassas alanlar her zaman düşürülür. `RecordText` açıkken bile, gizli bir alan odakta olabileceği her durumda karakterler redakte edilir — *herhangi bir* mesajdan sonraki, bir sonraki kare odağın nereye gittiğini gösterene kadarki pencere dahil; çünkü bir komut sonucu odağı parola alanına Tab kadar kolay taşıyabilir. Dosyalar `0600`, asla üzerine yazılmaz, sağlama toplamlıdır ve değiştirilmişse oynatmada reddedilir.
>
> **Kalan risk:** kaydettiğiniz bir mesaj tipi eksiksiz kaydedilir — sır taşıyan biri için `session.RegisterRedacted` kullanın. Değerler kaydedilmese bile etiketler kaydedilir; *etiketinde* sır gösteren bir widget onu dosyaya koyar. Ve kayıt diskte bir dosyadır: kullanıcılarınızın gördüğünü içerebilecek bir log gibi davranın.
