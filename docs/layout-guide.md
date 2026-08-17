# 📐 Esnek Yerleşim Motoru Kılavuzu (Layout Guide)

Limoni, CSS Flexbox ve Ratatui kısıtlama modellerinden ilham alan, sıfır-tahsisatlı güçlü bir esnek kutu yerleşim motoruna (`layout.FlexLayout`) sahiptir.

---

## 1. Temel Kavramlar

Yerleşim motoru, verilen bir ana `cell.Rect` alanını, belirtilen **Yön (Direction)** ve **Kısıtlamalar (Constraints)** doğrultusunda alt parçalara (`[]cell.Rect`) böler.

```go
chunks := layout.FlexLayout{
    Direction: layout.Vertical,
    Constraints: []layout.Constraint{
        layout.Fixed(3), // Sabit 3 satır
        layout.Fill(),   // Kalan tüm boşluğu doldur
        layout.Fixed(3), // Sabit 3 satır
    },
}.Split(area)
```

---

## 2. Kısıtlama Türleri (Constraints)

| Kısıtlama Fonksiyonu | Açıklama | Kullanım Örneği |
| :--- | :--- | :--- |
| **`layout.Fixed(N)`** | Tam olarak $N$ hücre büyüklüğünde sabit alan tahsis eder. | `layout.Fixed(3)` (Başlık çubuğu için) |
| **`layout.Percentage(P)`** | Kullanılabilir toplam alanın $\%P$ kadarını ayırır (0-100). | `layout.Percentage(30)` (Sol kenar çubuğu) |
| **`layout.Ratio(R)`** | Kalan serbest alanı belirtilen ağırlık oranlarına göre dağıtır. | `layout.Ratio(2)`, `layout.Ratio(1)` ($2/3$ ve $1/3$) |
| **`layout.Fill()`** | Kalan tüm boşluğu kaplar (`Ratio(1)` ile eşdeğerdir). | `layout.Fill()` (Ana içerik görünümü) |
| **`layout.Min(N)`** | En az $N$ hücre büyüklüğünde olmasını garanti eder. | `layout.Min(10)` |
| **`layout.Max(N)`** | En fazla $N$ hücre büyüklüğünde olmasını sınırlar. | `layout.Max(40)` |
| **`layout.FitContent()`** | İçindeki widget'ın `SizeHint` boyutuna göre yer ayırır. | `layout.FitContent()` |

---

## 3. Çok Sütunlu ve İçiçe Yerleşim Örneği

Aşağıdaki örnekte sol tarafta bir menü (%25), sağ tarafta ise üst-alt olarak bölünmüş iki panel oluşturulmaktadır:

```go
// 1. Ana Ekranı Yatay Olarak Böl (Sol %25, Sağ %75)
mainColumns := layout.FlexLayout{
    Direction: layout.Horizontal,
    Constraints: []layout.Constraint{
        layout.Percentage(25),
        layout.Percentage(75),
    },
}.Split(area)

leftSidebarArea := mainColumns[0]
rightContentArea := mainColumns[1]

// 2. Sağ Tarafı Dikey Olarak İkiye Böl (Üst Tablo, Alt Loglar)
rightRows := layout.FlexLayout{
    Direction: layout.Vertical,
    Constraints: []layout.Constraint{
        layout.Ratio(2), // 2/3 oran
        layout.Ratio(1), // 1/3 oran
    },
}.Split(rightContentArea)

tableArea := rightRows[0]
logsArea := rightRows[1]
```

---

## 4. Boşluk (Gap) ve Kenarlık (Padding/Margin)

Elemanlar arasında görsel boşluk bırakmak için `Gap` parametresi kullanılabilir:

```go
cards := layout.FlexLayout{
    Direction: layout.Horizontal,
    Gap: 2, // Elemanlar arasında 2 hücre boşluk bırak
    Constraints: []layout.Constraint{
        layout.Ratio(1),
        layout.Ratio(1),
        layout.Ratio(1),
    },
}.Split(area)
```
