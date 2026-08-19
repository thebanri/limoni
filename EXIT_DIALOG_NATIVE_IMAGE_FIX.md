# Çıkış Dialogu, Native Profil Resmi ve Terminal Uyumluluğu (Kitty & Alacritty)

Bu belge, çıkış diyalogu yerel (native) profil resminin üzerine geldiğinde yaşanan tüm sorunların kök nedenlerini, çözümlerini ve kalıcı mimarisini belgeler.

## Belirtiler ve Yaşanan Sorunlar

1. **Şeffaf Diyalog:** Diyalog profil resminin üzerine geldiğinde arka planı resmi kapatmıyor, profil resmi diyalogun içinden doğrudan görünüyordu.
2. **Karakter ve Çizgi Kalıntıları (Ghost Lines):** Önceki sekmelerden (Settings, Graphics, Home) kalan gradyan çubukları ve tablolar ya da sürüklenen diyalogun kenarlıkları profil resminin üzerinde asılı kalıyordu.
3. **Siyah Basamak Silueti (Black Staircase):** Diyalog sürüklenirken resmin üzerine siyah basamak şeklinde dikdörtgen bloklar kesiliyordu.
4. **Alacritty Ekran Yanıp Sönmesi ve Yırtılma:** Diyalog açıldığında, kapandığında veya animasyon oynatıldığında Alacritty'de tüm ekran silinip kararıyor veya titriyordu.

---

## Kök Nedenler

1. **Katmanlama ve Z-Index:**
   - Kitty Graphics protokolünde resimler `z < 0` seviyesinde metin ızgarasının arkasına yerleştirilir. Düz `Dialog.Draw` hücre arka planı (`Style.Bg`), GPU katmanındaki resmi tam örtemez.
2. **İmleç Takibi ve Diff Atlama Hatası (`diff.go`):**
   - Resim hücreleri (`cell.RuneImage`) atlanırken terminal imleci sanal olarak sağa kaydırılıyordu (`cursorX++`), fakat terminal donanımında imleç sol panelde asılı kalıyordu. Resimden sonra gelen karakterler resmin üzerine basılıyordu.
3. **Eski Hücrelerin Silinmemesi:**
   - Diyalog resmin üzerinden çekildiğinde, eski diyalog karakterleri terminal donanımından silinmediği için GPU resminin üzerinde görünmeye devam ediyordu.
4. **Gereksiz Ekran Silme (`\x1b[2J`):**
   - Diyalog açılırken çağrılan `ForceFullRedraw()` Alacritty altında tam ekran silme (`\x1b[2J`) tetikliyor ve ekranı yırtıyordu.

---

## Uygulanan Kesin Mimari Çözüm

### 1. Z-Index Katmanlama Mimarisi
```text
Profil resmi           -3 (En altta, Frame.imageClosure ile)
Dialog opaque backdrop -2 (shadowBackdrop: animatedArea.Width+2, Height+1)
Dialog shadow          ASCII buffer
Dialog border/text     ASCII buffer (z = 0, en üstte net başlık, soru ve butonlar)
```

- Dosya: `examples/demo/main.go`
```go
if animatedArea.Width > 0 && animatedArea.Height > 0 {
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

    exitDialog := widgets.Dialog{ ... }
    f.RenderWidget(exitDialog, animatedArea)
}
```

### 2. Dinamik ECMA-48 ECH Temizleme & İmleç Geçersiz Kılma
- Dosya: `core/buffer/diff.go`
- Resim hücreleri taranırken imleç takibi geçersiz kılınır (`cursorX = 9999, cursorY = 9999`).
- Önceki karede diyalog veya metin bulunan resim hücreleri açığa çıktığında (`needsErase = true`), `\x1b[0m` ile stil sıfırlanıp `\x1b[<uzunluk>X` (ECH - Erase Characters) komutuyla terminal donanımından anında silinir.

### 3. Alacritty (ProtocolHalfBlock) İzolasyonu
- Dosyalar: `widgets/block.go`, `core/terminal/terminal.go`
- `Block.Opaque` yalnızca `proto != graphics.ProtocolHalfBlock` durumunda native resim kaydı yapar.
- Alacritty'de tüm çizimler doğrudan hücre matrisi üzerinden sıfır gecikmeyle 60 FPS yapılır; hiçbir ekran silme veya yırtılma oluşmaz.

### 4. Sekme Geçişinde Temizleme
- Dosyalar: `examples/demo/main.go`, `examples/demo/helpers.go`
- Sekme geçişleri anında (`main.go` - Fare tıklaması, Enter/Space, Shift+Tab ve Command Palette) tek seferlik `\x1b[2J` ve `t.ForceFullRedraw()` tetiklenerek eski sekmeden kalan tüm metinler terminal donanımından silinir.

---

## Doğrulama Kontrol Listesi

- [x] Sekmeler arası geçişlerde resim üzerinde hiçbir eski sekme çizgisi kalmıyor.
- [x] Çıkış diyalogu açıldığında profil resmini %100 örtüyor, şeffaflık oluşmuyor.
- [x] Diyalog sürüklendiğinde arkasında sıfır hayalet çizgi ve sıfır siyah kutu bırakıyor.
- [x] Alacritty altında hiçbir ekran yanıp sönmesi (`\x1b[2J`) ve yırtılma olmuyor.
- [x] `go test ./...` ve `go test -race ./...` %100 başarılı.

Temel prensip:

> Dialog animasyonunu yalnızca dialogun görsel hücre alanında uygula; native profil resminin alanını, içeriğini ve transformunu animasyona bağlama.