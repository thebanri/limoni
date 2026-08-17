# ♿ Erişilebilirlik (A11y) ve Temalar (Accessibility & Theming)

Limoni, erişilebilirliği sonradan eklenen bir yama değil, çekirdek motorun temel bir bileşeni olarak ele alır (`core/accessibility`).

---

## 1. Ekran Okuyucu Semantik Ağacı (Semantic A11y Tree)

Tüm widget'lar çizim anında terminal hücresinin ötesinde semantik bir düğüm (Node) üretir:
- Rol (`RoleButton`, `RoleTable`, `RoleInput`, `RoleDialog`)
- Değer, başlık ve durum (Örn: "Seçili", "Devre Dışı", "%60 Tamamlandı")
- Ekran okuyucu yazılımlar terminal metinlerini rastgele okumak yerine bu yapılandırılmış ağacı tüketebilir.

---

## 2. Standart A11y Modları

1. **Yüksek Kontrast Modu (High Contrast)**:
   - Düşük kontrastlı gri veya pastel tonları otomatik olarak WCAG AAA uyumlu zıt renklere (Siyah-Beyaz-Sarı) yükseltir.
2. **`NO_COLOR` Standardı**:
   - Ortam değişkeni `NO_COLOR=1` olduğunda veya açıkça aktif edildiğinde tüm ANSI renk kaçış kodları devre dışı bırakılır; arayüz sembolik olarak çizilir.
3. **Azaltılmış Hareket Modu (Reduced Motion)**:
   - Vestibüler bozukluğu olan kullanıcılar için animasyon süreleri 0'a çekilir; arayüz anında nihai haline geçer.
