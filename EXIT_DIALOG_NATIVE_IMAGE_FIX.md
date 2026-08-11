# Çıkış Dialogu ve Native Profil Resmi Sorunları

Bu belge, çıkış dialogu profil resminin üzerine geldiğinde yaşanan sorunların kök nedenlerini ve kalıcı çözümünü kaydeder.

## Belirtiler

- Dialog profil resminin üzerine gelince uygulama yavaşlıyordu.
- Profil resmi itilmiş, yeniden konumlandırılmış veya bozulmuş görünüyordu.
- Dialogun arka planı resmi kapatmıyor, profil resmi dialogun içinden görünüyordu.
- Dialog gölgesi profil resminin arkasında kalıyordu.
- Açılış/kapanış animasyonu kaldırılınca görüntü düzeliyor, ancak animasyon kayboluyordu.
- Animasyon geri getirildiğinde backdrop/gölge dialogdan önce oluşabiliyordu.

## Kök neden

Profil resmi normal terminal hücresi olarak değil, Kitty/Sixel/iTerm2 native image katmanı olarak çiziliyor:

```text
Image widget -> Frame.ImageRegions -> native image escape sequence
```

Native image terminal hücre buffer'ından bağımsızdır. Bu nedenle `Dialog.Draw` içindeki `Style.Bg` profil resmini kapatamaz.

İlk performans sorunu, modal alanının animasyon sırasında `ScaleRect` ile her karede değişmesinden kaynaklanıyordu. Native image alanı değişince resim crop/yeniden encode/yeniden yerleştirme işlemine giriyor ve hareket ediyor gibi görünüyordu.

## Uygulanan çözüm

### 1. Modal olay alanı sabit tutuldu

Dosya: `/home/thebanri/Projects/Limoni/examples/demo/main.go`

Çıkış modalı sabit `46x9` `dialogArea` üzerinde kayıtlıdır:

```go
f.RegisterModal("exit_dialog", dialogArea, onClickOutside)
```

Animasyon modal olay alanını değiştirmez. Mouse olayları ve native profil resmi kararlı kalır.

### 2. Görsel dialog animasyonu korundu

Animasyon modal kayıt alanından ayrıdır:

```go
progress := state.ExitDialogAnim.Value()
animatedArea := terminal.ScaleRect(dialogArea, progress)
f.RenderWidget(exitDialog, animatedArea)
```

Dialog açılırken büyür, kapanırken küçülür; profil resminin alanı değişmez.

### 3. Native profil resminin üstüne opak backdrop eklendi

Dialogdan hemen önce animasyonlu alana opak bir blok çizilir:

```go
shadowBackdrop := cell.NewRect(
    animatedArea.X,
    animatedArea.Y,
    animatedArea.Width+2,
    animatedArea.Height+1,
)
f.RenderWidget(widgets.Block{
    Style:  cell.Style{Bg: cell.NewColorRGB(18, 20, 24)},
    Opaque: true,
}, shadowBackdrop)
```

Bu kaplama profil resmini değiştirmez; yalnızca dialogun kapladığı alanı örter. `animatedArea` kullanıldığı için backdrop dialogdan önce görünmez ve animasyonla senkron hareket eder.

### 4. Z-index sırası düzeltildi

Dosya: `/home/thebanri/Projects/Limoni/examples/demo/tab_playground.go`

Profil resmi `ZIndex: -3` seviyesindedir. Frame içindeki `-99` opaque marker modal bağlamında `-2` seviyesine eşlenir:

```text
Profil resmi           -3
Dialog opaque backdrop -2
Dialog shadow          ASCII buffer
Dialog border/text     ASCII buffer, en üstte
```

### 5. Native image crop işlemi kaldırıldı

Dosya: `/home/thebanri/Projects/Limoni/core/terminal/terminal.go`

`clippedImageRegions` artık modal alanına göre resmi parçalara ayırmaz veya `CropImage` çağırmaz; `Frame.ImageRegions` doğrudan korunur.

Profil resmine kesinlikle şunlar uygulanmamalıdır:

- modal alanına göre crop,
- modal hareketine göre resize,
- dialog animasyonuna göre image area değiştirme,
- her mouse hareketinde native image yeniden encode etme.

## Gölge neden `+2`, `+1` alanıyla kaplanıyor?

`widgets.Dialog.Draw` şu çağrıyı yapar:

```go
DrawShadow(buf, ctx.Area, 2, 1)
```

Gölge dialogun sağında 2 hücre ve altında 1 hücre taşar. Bu nedenle backdrop ölçüsü `animatedArea.Width+2` ve `animatedArea.Height+1` olmalıdır.

## Değiştirilmemesi gerekenler

1. `RegisterModal` içine `ScaleRect` sonucu verilmemeli; `dialogArea` kullanılmalı.
2. Native resimler modal alanına göre crop edilmemeli.
3. Profil resminin `Area` değeri dialog animasyonuna bağlanmamalı.
4. Profil resmi backdrop ile aynı z-index seviyesine alınmamalı.
5. Dialog arka planı yalnızca `cell.Style.Bg` ile çözülmeye çalışılmamalı.
6. Backdrop animasyon için `animatedArea` ile çizilmeli.
7. Gölge için `Width+2` ve `Height+1` payı kaldırılmamalı.

## Hızlı kontrol listesi

- [ ] `RegisterModal("exit_dialog", dialogArea, ...)` kullanılıyor mu?
- [ ] `RenderWidget(exitDialog, animatedArea)` kullanılıyor mu?
- [ ] Backdrop `animatedArea` üzerinden mi oluşturuluyor?
- [ ] Backdrop ölçüsü `Width+2`, `Height+1` mi?
- [ ] Profil resmi `ZIndex: -3` mü?
- [ ] `terminal.go` içinde modal kaynaklı `CropImage` var mı?
- [ ] Native resmin `Area` değeri dialog animasyonundan etkileniyor mu?
- [ ] `go test ./...` başarılı mı?

## Doğrulama komutları

```bash
gofmt -w /home/thebanri/Projects/Limoni/examples/demo/main.go
gofmt -w /home/thebanri/Projects/Limoni/examples/demo/tab_playground.go
gofmt -w /home/thebanri/Projects/Limoni/core/terminal/terminal.go
git diff --check
go test ./...
```

Temel prensip:

> Dialog animasyonunu yalnızca dialogun görsel hücre alanında uygula; native profil resminin alanını, içeriğini ve transformunu animasyona bağlama.