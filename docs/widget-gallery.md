# Widget Galerisi ve API Referansı

Bu dosya `go run ./internal/tools/widgetdocs` ile kaynak koddan üretilir; her bileşen
`Widget` arayüzünü (`Draw(cell.Context, *buffer.Buffer)`) uygular.
Widget'ı `frame.RenderWidget(widget, area)` ile çizin; tabloda `*` ile başlayan
alıcı tipleri işaretçi olarak verilmelidir (örn. `&widgets.Paragraph{...}`).

| Widget | Alıcı tipi | Alan sayısı |
| --- | --- | --- |
| [Block](#block) | `Block` | 15 |
| [Canvas](#canvas) | `*Canvas` | 0 |
| [Checkbox](#checkbox) | `Checkbox` | 5 |
| [CommandPalette](#commandpalette) | `CommandPalette` | 8 |
| [Dialog](#dialog) | `Dialog` | 12 |
| [Image](#image) | `*Image` | 9 |
| [List](#list) | `List` | 9 |
| [Markdown](#markdown) | `*Markdown` | 5 |
| [Paragraph](#paragraph) | `*Paragraph` | 3 |
| [Popup](#popup) | `Popup` | 10 |
| [ProgressBar](#progressbar) | `ProgressBar` | 8 |
| [RadioButton](#radiobutton) | `RadioButton` | 6 |
| [Select](#select) | `Select` | 10 |
| [Slider](#slider) | `Slider` | 9 |
| [Sparkline](#sparkline) | `Sparkline` | 3 |
| [Table](#table) | `Table` | 15 |
| [Text](#text) | `Text` | 4 |
| [TextArea](#textarea) | `TextArea` | 4 |
| [TextInput](#textinput) | `TextInput` | 6 |
| [Transducer](#transducer) | `Transducer` | 3 |
| [VirtualDataView](#virtualdataview) | `VirtualDataView` | 14 |

## Block

Block, terminal ekranında kenarlık çizebilen, arka plan dolgusu yapabilen ve üstüne başlık (Title) yerleştirebilen en temel kapsayıcı (container) widget'tır.

`RenderWidget` çağrısında kullanılacak tip: `Block`

| Alan | Tip | Açıklama |
| --- | --- | --- |
| `Title` | `string` | Title, bloğun üst kenarında gösterilecek olan başlık metnidir. |
| `TitleAlignment` | `Alignment` | TitleAlignment, başlık metninin kenarlık üzerindeki hizasını belirler (Left, Center, Right). |
| `TitleStyle` | `cell.Style` | TitleStyle, başlık metninin rengini ve stil özelliklerini belirler. |
| `Borders` | `uint8` | Borders, hangi kenarların çizileceğini belirleyen maske alanıdır (örn. |
| `BorderSymbols` | `BorderSymbols` | BorderSymbols, kenarlık çiziminde kullanılacak olan glif sembolleridir (örn. |
| `BorderStyle` | `cell.Style` | BorderStyle, kenarlık çizgilerinin rengini ve stilini belirler. |
| `Margin` | `Insets` | Margin, bloğun dışındaki CSS benzeri boşluktur. |
| `Padding` | `Insets` | Padding, içerik ile kenarlık arasındaki CSS benzeri iç boşluktur. |
| `PaddingLeft` | `uint16` |  |
| `PaddingRight` | `uint16` |  |
| `PaddingTop` | `uint16` |  |
| `PaddingBottom` | `uint16` |  |
| `Style` | `cell.Style` | Style, bloğun arka plan dolgu rengini ve varsayılan genel stilini belirler. |
| `Child` | `Widget` | Child, bloğun içerisine çizilecek olan alt görsel bileşendir. |
| `Opaque` | `bool` | Opaque, true ise bloğun arkasına yerel resimlerin sızmasını engellemek için solid renkli resim katmanı ekler. |

## Canvas

Canvas, hücre başına 2x4 sanal piksel çözünürlüğünde (Braille karakterleri kullanarak) terminal üzerinde yüksek çözünürlüklü vektör çizimleri yapmayı sağlayan görsel bileşendir.

`RenderWidget` çağrısında kullanılacak tip: `*Canvas`

_Alanı yok._

## Checkbox

Checkbox, işaretlenebilir interaktif bir onay kutusudur.

`RenderWidget` çağrısında kullanılacak tip: `Checkbox`

| Alan | Tip | Açıklama |
| --- | --- | --- |
| `ID` | `string` |  |
| `Checked` | `*bool` |  |
| `Label` | `string` |  |
| `Style` | `cell.Style` |  |
| `FocusedStyle` | `cell.Style` |  |

## CommandPalette

CommandPalette, Komut Paleti overlay widget'ıdır.

`RenderWidget` çağrısında kullanılacak tip: `CommandPalette`

| Alan | Tip | Açıklama |
| --- | --- | --- |
| `ID` | `string` |  |
| `State` | `*CommandPaletteState` |  |
| `Position` | `*CommandPalettePosition` |  |
| `Style` | `cell.Style` |  |
| `InputStyle` | `cell.Style` |  |
| `ItemStyle` | `cell.Style` |  |
| `SelStyle` | `cell.Style` |  |
| `DetailStyle` | `cell.Style` |  |

## Dialog

Dialog is a premium, modern glassmorphism dialog widget with glowing gradient borders and blended shadows.

`RenderWidget` çağrısında kullanılacak tip: `Dialog`

| Alan | Tip | Açıklama |
| --- | --- | --- |
| `ID` | `string` |  |
| `Title` | `string` |  |
| `Message` | `string` |  |
| `SubMessage` | `string` |  |
| `Buttons` | `[]DialogButton` |  |
| `Style` | `cell.Style` |  |
| `HeaderStyle` | `cell.Style` |  |
| `BorderStyle` | `cell.Style` |  |
| `ButtonStyle` | `cell.Style` |  |
| `ButtonFocusedStyle` | `cell.Style` |  |
| `BorderSymbols` | `BorderSymbols` |  |
| `Shadow` | `bool` |  |

## Image

Image, terminalde yerel görsel protokolleri (Kitty, Sixel, iTerm2) kullanarak PNG/JPG gibi gerçek resimleri çizebilen TUI bileşenidir.

`RenderWidget` çağrısında kullanılacak tip: `*Image`

| Alan | Tip | Açıklama |
| --- | --- | --- |
| `Img` | `image.Image` | Img, gösterilecek olan ham resim nesnesidir. |
| `ZIndex` | `int` | ZIndex, resmin dikey katman yerleşim sırasıdır. |
| `ForceHalfBlock` | `bool` | ForceHalfBlock, aktif edilirse donanımsal protokoller yerine hücre tabanlı half-block yöntemini zorlar. |
| `CircleMask` | `bool` | CircleMask, resmi daire şeklinde kırpar (avatar). |
| `OpaqueBackground` | `bool` | OpaqueBackground composites transparency over Background before native rendering. |
| `Background` | `cell.Color` |  |
| `Transparent` | `bool` | Transparent, resmin şeffaf piksellerinin korunup korunmayacağını belirtir. |
| `Opacity` | `float64` | Opacity, resmin opaklık değeridir (0.0 ile 1.0 arasında). |
| `OpacitySet` | `bool` | OpacitySet, Opacity alanının bilinçli olarak ayarlandığını belirtir. |

## List

List, terminal ekranında liste şeklinde dikey öğeler çizen interaktif widget'tır.

`RenderWidget` çağrısında kullanılacak tip: `List`

| Alan | Tip | Açıklama |
| --- | --- | --- |
| `Items` | `[]string` | Items, listede gösterilecek olan metin dizilimleridir. |
| `Provider` | `ListProvider` | Provider, sanal liste (virtual scrolling) için veri sağlayıcıdır. |
| `Scrollbar` | `bool` | Scrollbar, aktif edilirse listenin sağ kenarında bir dikey kaydırma çubuğu çizer. |
| `ScrollbarTrackStyle` | `cell.Style` | ScrollbarTrackStyle, kaydırma çubuğu rayının (track) stilidir. |
| `ScrollbarThumbStyle` | `cell.Style` | ScrollbarThumbStyle, kaydırma çubuğu kaydırıcısının (thumb) stilidir. |
| `Style` | `cell.Style` | Style, listenin genel rengini ve yazı stilini belirtir. |
| `SelectedStyle` | `cell.Style` | SelectedStyle, seçili olan öğenin vurgulanacağı stildir. |
| `HighlightSymbol` | `string` | HighlightSymbol, seçili olan öğenin soluna yerleştirilecek semboldür (örn: "> "). |
| `State` | `*ListState` | State, listenin seçili indeksi ve kaydırma durumunu tutan işaretçidir (pointer). |

## Markdown

`RenderWidget` çağrısında kullanılacak tip: `*Markdown`

| Alan | Tip | Açıklama |
| --- | --- | --- |
| `ID` | `string` |  |
| `Content` | `string` | Content, parse edilip çizilecek olan ham markdown metnidir. |
| `Style` | `cell.Style` | Style, varsayılan metin stilini tanımlar. |
| `FocusedStyle` | `cell.Style` |  |
| `ScrollOffset` | `*int` |  |

## Paragraph

Paragraph, çok satırlı metinleri gösteren görsel bileşendir.

`RenderWidget` çağrısında kullanılacak tip: `*Paragraph`

| Alan | Tip | Açıklama |
| --- | --- | --- |
| `Text` | `string` | Text, gösterilecek olan metin içeriğidir. |
| `Style` | `cell.Style` | Style, metnin yazı rengi, arka planı ve modifikatör stillerini belirler. |
| `Wrap` | `bool` | Wrap, metnin sınır genişliğine göre otomatik olarak alt satıra kaydırılıp kaydırılmayacağını belirler. |

## Popup

Popup, açılır menü (dropdown) widget'ıdır.

`RenderWidget` çağrısında kullanılacak tip: `Popup`

| Alan | Tip | Açıklama |
| --- | --- | --- |
| `ID` | `string` | ID, popup'ın benzersiz tanımlayıcısıdır. |
| `Label` | `string` | Label, buton üzerindeki başlangıç metnidir. |
| `Items` | `[]PopupItem` | Items, menüdeki öğelerin listesidir. |
| `State` | `*PopupState` | State, popup'ın açık/kapalı ve seçim durumunu yönetir. |
| `Style` | `cell.Style` | Style, buton ve menü arka plan stilini belirler. |
| `ItemStyle` | `cell.Style` | ItemStyle, menü öğelerinin normal stilini belirler. |
| `SelectedStyle` | `cell.Style` | SelectedStyle, menü öğesinin fare sobre kaldığında/klavye ile seçili olduğundaki stilidir. |
| `DisabledStyle` | `cell.Style` | DisabledStyle, devre dışı bırakılmış menü öğelerinin stilidir. |
| `BorderStyle` | `cell.Style` | BorderStyle, menü kenarlığının stilini belirler. |
| `BorderSymbols` | `BorderSymbols` | BorderSymbols, menü kenarlık sembollerini belirler. |

## ProgressBar

ProgressBar renders a bounded horizontal progress indicator.

`RenderWidget` çağrısında kullanılacak tip: `ProgressBar`

| Alan | Tip | Açıklama |
| --- | --- | --- |
| `ID` | `string` |  |
| `Value` | `float64` |  |
| `Min` | `float64` |  |
| `Max` | `float64` |  |
| `Style` | `cell.Style` |  |
| `FilledStyle` | `cell.Style` |  |
| `EmptyStyle` | `cell.Style` |  |
| `ShowPercent` | `bool` |  |

## RadioButton

RadioButton, çoklu seçenek gruplarında tekil seçim yapmayı sağlayan radyo butonudur.

`RenderWidget` çağrısında kullanılacak tip: `RadioButton`

| Alan | Tip | Açıklama |
| --- | --- | --- |
| `ID` | `string` |  |
| `Selected` | `*string` |  |
| `Value` | `string` |  |
| `Label` | `string` |  |
| `Style` | `cell.Style` |  |
| `FocusedStyle` | `cell.Style` |  |

## Select

Select is a keyboard- and mouse-interactive dropdown field.

`RenderWidget` çağrısında kullanılacak tip: `Select`

| Alan | Tip | Açıklama |
| --- | --- | --- |
| `ID` | `string` |  |
| `Options` | `[]string` |  |
| `State` | `*SelectState` |  |
| `Style` | `cell.Style` |  |
| `FocusedStyle` | `cell.Style` |  |
| `OptionStyle` | `cell.Style` |  |
| `SelectedStyle` | `cell.Style` |  |
| `HoverStyle` | `cell.Style` |  |
| `BorderStyle` | `cell.Style` |  |
| `OnChange` | `func(...)` |  |

## Slider

Slider is a horizontal mouse- and keyboard-controlled numeric slider.

`RenderWidget` çağrısında kullanılacak tip: `Slider`

| Alan | Tip | Açıklama |
| --- | --- | --- |
| `ID` | `string` |  |
| `State` | `*SliderState` |  |
| `Min` | `int` |  |
| `Max` | `int` |  |
| `Style` | `cell.Style` |  |
| `TrackStyle` | `cell.Style` |  |
| `FilledStyle` | `cell.Style` |  |
| `ThumbStyle` | `cell.Style` |  |
| `FocusedStyle` | `cell.Style` |  |

## Sparkline

`RenderWidget` çağrısında kullanılacak tip: `Sparkline`

| Alan | Tip | Açıklama |
| --- | --- | --- |
| `Data` | `[]float64` | Data, çizilecek veri geçmişini temsil eden sayılar dizisidir. |
| `Style` | `cell.Style` | Style, varsayılan hücre stilini tanımlar. |
| `Color` | `cell.Color` | Color, barların rengini belirler. |

## Table

Table, interaktif, esnek sütunlu, dikey kaydırılabilir ve hücre birleştirme destekli tablo bileşenidir.

`RenderWidget` çağrısında kullanılacak tip: `Table`

| Alan | Tip | Açıklama |
| --- | --- | --- |
| `ID` | `string` |  |
| `Header` | `*TableRow` |  |
| `Rows` | `[]TableRow` |  |
| `DataSource` | `TableDataSource` |  |
| `Constraints` | `[]TableConstraint` |  |
| `State` | `*TableState` |  |
| `GridStyle` | `cell.Style` |  |
| `SelectedStyle` | `cell.Style` |  |
| `DrawGrid` | `bool` |  |
| `SortEnabled` | `bool` |  |
| `MultiSelect` | `bool` |  |
| `FilterQuery` | `string` |  |
| `CellStyle` | `func(...)` |  |
| `StickyColumns` | `int` |  |
| `Scrollbar` | `bool` |  |

## Text

Text renders multiple rich-text lines with optional cell-aware wrapping.

`RenderWidget` çağrısında kullanılacak tip: `Text`

| Alan | Tip | Açıklama |
| --- | --- | --- |
| `Lines` | `[]Line` |  |
| `Style` | `cell.Style` |  |
| `Wrap` | `bool` |  |
| `Alignment` | `TextAlignment` |  |

## TextArea

TextArea is a multiline focusable text editor.

`RenderWidget` çağrısında kullanılacak tip: `TextArea`

| Alan | Tip | Açıklama |
| --- | --- | --- |
| `ID` | `string` |  |
| `State` | `*TextAreaState` |  |
| `Style` | `cell.Style` |  |
| `FocusedStyle` | `cell.Style` |  |

## TextInput

TextInput, tek satırlı bir metin girişi kutusudur.

`RenderWidget` çağrısında kullanılacak tip: `TextInput`

| Alan | Tip | Açıklama |
| --- | --- | --- |
| `ID` | `string` |  |
| `State` | `*TextInputState` |  |
| `Placeholder` | `string` |  |
| `Style` | `cell.Style` |  |
| `PlaceholderStyle` | `cell.Style` |  |
| `FocusedStyle` | `cell.Style` |  |

## Transducer

`RenderWidget` çağrısında kullanılacak tip: `Transducer`

| Alan | Tip | Açıklama |
| --- | --- | --- |
| `Child` | `Widget` |  |
| `Type` | `TransducerType` |  |
| `Progress` | `float64` |  |

## VirtualDataView

VirtualDataView renders the visible portion of a VirtualDataState cache.

`RenderWidget` çağrısında kullanılacak tip: `VirtualDataView`

| Alan | Tip | Açıklama |
| --- | --- | --- |
| `ID` | `string` |  |
| `State` | `*VirtualDataState` |  |
| `Source` | `VirtualDataSource` |  |
| `First` | `int` |  |
| `Prefetch` | `int` |  |
| `Style` | `cell.Style` |  |
| `SelectedStyle` | `cell.Style` |  |
| `EmptyText` | `string` |  |
| `LoadingText` | `string` |  |
| `ErrorText` | `string` |  |
| `Offset` | `*int` |  |
| `HorizontalOffset` | `int` | HorizontalOffset scrolls non-sticky cell text by terminal columns. |
| `StickyColumns` | `int` | StickyColumns keeps the first N Row.Cells visible while the remaining cells are horizontally scrolled. |
| `OnSelect` | `func(...)` | OnSelect is called with the virtual row index after a row is clicked. |

