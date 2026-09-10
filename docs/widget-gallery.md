# Widget Galerisi ve API Referansı

Bu dosya `widgets` paketindeki kaynak koddan üretilmiştir; her bileşen `Widget`
arayüzünü (`Draw(cell.Context, *buffer.Buffer)`) uygular.

| Widget | Alıcı tipi | Alan sayısı |
| --- | --- | --- |
| [Ascii3D](#ascii3d) | `Ascii3D` | 32 |
| [BarChart](#barchart) | `BarChart` | 12 |
| [Block](#block) | `Block` | 15 |
| [Canvas](#canvas) | `*Canvas` | 0 |
| [Checkbox](#checkbox) | `Checkbox` | 5 |
| [ColorPicker](#colorpicker) | `ColorPicker` | 5 |
| [CommandPalette](#commandpalette) | `CommandPalette` | 8 |
| [DevTools](#devtools) | `DevTools` | 2 |
| [Dialog](#dialog) | `Dialog` | 14 |
| [Image](#image) | `*Image` | 11 |
| [Label](#label) | `Label` | 2 |
| [LineChart](#linechart) | `LineChart` | 10 |
| [List](#list) | `List` | 11 |
| [Markdown](#markdown) | `*Markdown` | 5 |
| [Paragraph](#paragraph) | `*Paragraph` | 5 |
| [PieChart](#piechart) | `PieChart` | 6 |
| [Popup](#popup) | `Popup` | 10 |
| [ProgressBar](#progressbar) | `ProgressBar` | 9 |
| [RadioButton](#radiobutton) | `RadioButton` | 6 |
| [Select](#select) | `Select` | 12 |
| [Slider](#slider) | `Slider` | 12 |
| [Sparkline](#sparkline) | `Sparkline` | 5 |
| [Table](#table) | `Table` | 16 |
| [Text](#text) | `Text` | 6 |
| [TextArea](#textarea) | `TextArea` | 4 |
| [TextInput](#textinput) | `TextInput` | 6 |
| [ToastManager](#toastmanager) | `*ToastManager` | 3 |
| [Transducer](#transducer) | `Transducer` | 3 |
| [TreeView](#treeview) | `TreeView` | 9 |
| [Viewer3D](#viewer3d) | `*Viewer3D` | 15 |
| [VirtualDataView](#virtualdataview) | `VirtualDataView` | 15 |

## Ascii3D

Ascii3D (or AsciiObject) is a high-performance 3D vector-to-ASCII terminal renderer.

`RenderWidget` çağrısında kullanılacak tip: `Ascii3D`

| Alan | Tip | Açıklama |
| --- | --- | --- |
| `Model` | `graphics.Model3D` | 3D Model geometry or file path |
| `Src` | `string` |  |
| `Mode` | `Ascii3DMode` | Rendering Mode: - ModeASCII: Typography character ramps (default) - ModeBlock: 2x vertical sub-cell Half-Block (▄) - ModeDithered: Retro Bayer 4x4 ordered dithering - ModeBraille: 8x sub-pixel Unicode Braille dot matrix |
| `Scale` | `float64` | Transform & Animation |
| `XOffset` | `float64` |  |
| `YOffset` | `float64` |  |
| `FloatIntensity` | `float64` |  |
| `FloatSpeed` | `float64` |  |
| `RotationIntensity` | `float64` |  |
| `AutoRotate` | `bool` |  |
| `AutoRotateSpeed` | `float64` |  |
| `Time` | `float64` |  |
| `RotX` | `float64` | Manual Rotation Offsets (degrees) |
| `RotY` | `float64` |  |
| `RotZ` | `float64` |  |
| `FOV` | `float64` | Camera & Projection |
| `CameraDistance` | `float64` |  |
| `CellAspect` | `float64` |  |
| `Contrast` | `float64` | Shading & Optics |
| `EdgeContrast` | `float64` |  |
| `Exposure` | `float64` |  |
| `EnvironmentIntensity` | `float64` |  |
| `Roughness` | `float64` |  |
| `LightDirection` | `graphics.Vector3D` |  |
| `Ascii` | `bool` | Rendering Mode & Palette |
| `SubCell` | `bool` |  |
| `Braille` | `bool` |  |
| `Colored` | `bool` |  |
| `Invert` | `bool` |  |
| `Color` | `cell.Color` |  |
| `Highlight` | `cell.Color` |  |
| `Ramp` | `string` |  |

## BarChart

BarChart renders vertical and horizontal bar charts with customizable symbols and labels.

`RenderWidget` çağrısında kullanılacak tip: `BarChart`

| Alan | Tip | Açıklama |
| --- | --- | --- |
| `ID` | `string` |  |
| `Data` | `[]BarData` |  |
| `Direction` | `BarChartDirection` |  |
| `BarWidth` | `int` |  |
| `BarGap` | `int` |  |
| `Max` | `float64` |  |
| `Min` | `float64` |  |
| `ShowValues` | `bool` |  |
| `ValueFormatter` | `func(...)` |  |
| `Style` | `cell.Style` |  |
| `LabelStyle` | `cell.Style` |  |
| `DefaultColor` | `cell.Color` |  |

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

## ColorPicker

ColorPicker is a rich, KDE / desktop-style 2D HSV graphical color picker.

`RenderWidget` çağrısında kullanılacak tip: `ColorPicker`

| Alan | Tip | Açıklama |
| --- | --- | --- |
| `ID` | `string` |  |
| `State` | `*ColorPickerState` |  |
| `Palette` | `[]cell.Color` |  |
| `ShowPreview` | `bool` |  |
| `Style` | `cell.Style` |  |

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

## DevTools

DevTools renders the in-terminal developer inspection dashboard.

`RenderWidget` çağrısında kullanılacak tip: `DevTools`

| Alan | Tip | Açıklama |
| --- | --- | --- |
| `State` | `*DevToolsState` |  |
| `Style` | `cell.Style` |  |

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
| `FocusedButton` | `int` |  |
| `OnButtonHover` | `func(...)` |  |

## Image

Image, terminalde yerel görsel protokolleri (Kitty, Sixel, iTerm2) kullanarak PNG/JPG gibi gerçek resimleri çizebilen TUI bileşenidir.

`RenderWidget` çağrısında kullanılacak tip: `*Image`

| Alan | Tip | Açıklama |
| --- | --- | --- |
| `ID` | `string` | ID, widget odak kimliğidir. |
| `Img` | `image.Image` | Img, gösterilecek olan ham resim nesnesidir. |
| `ZIndex` | `int` | ZIndex, resmin dikey katman yerleşim sırasıdır. |
| `ForceHalfBlock` | `bool` | ForceHalfBlock, aktif edilirse donanımsal protokoller yerine hücre tabanlı half-block yöntemini zorlar. |
| `CircleMask` | `bool` | CircleMask, resmi daire şeklinde kırpar (avatar). |
| `OpaqueBackground` | `bool` | OpaqueBackground composites transparency over Background before native rendering. |
| `Background` | `cell.Color` |  |
| `Transparent` | `bool` | Transparent, resmin şeffaf piksellerinin korunup korunmayacağını belirtir. |
| `Opacity` | `float64` | Opacity, resmin opaklık değeridir (0.0 ile 1.0 arasında). |
| `OpacitySet` | `bool` | OpacitySet, Opacity alanının bilinçli olarak ayarlandığını belirtir. |
| `FocusedStyle` | `cell.Style` | FocusedStyle, odaklandığında uygulanacak kenar/vurgu stilidir. |

## Label

Label is a lightweight stateless widget for rendering single- or multi-line text.

`RenderWidget` çağrısında kullanılacak tip: `Label`

| Alan | Tip | Açıklama |
| --- | --- | --- |
| `Text` | `string` |  |
| `Style` | `cell.Style` |  |

## LineChart

LineChart renders smooth Braille-based multi-series line graphs with labeled axes.

`RenderWidget` çağrısında kullanılacak tip: `LineChart`

| Alan | Tip | Açıklama |
| --- | --- | --- |
| `ID` | `string` |  |
| `Datasets` | `[]LineDataset` |  |
| `MinY` | `float64` |  |
| `MaxY` | `float64` |  |
| `XLabels` | `[]string` |  |
| `ShowAxes` | `bool` |  |
| `ShowGrid` | `bool` |  |
| `ShowLegend` | `bool` |  |
| `Style` | `cell.Style` |  |
| `AxisStyle` | `cell.Style` |  |

## List

List, terminal ekranında liste şeklinde dikey öğeler çizen interaktif widget'tır.

`RenderWidget` çağrısında kullanılacak tip: `List`

| Alan | Tip | Açıklama |
| --- | --- | --- |
| `ID` | `string` | ID, listenin odaklanma ve kimlik belirleme kimliğidir. |
| `Items` | `[]string` | Items, listede gösterilecek olan metin dizilimleridir. |
| `Provider` | `ListProvider` | Provider, sanal liste (virtual scrolling) için veri sağlayıcıdır. |
| `Scrollbar` | `bool` | Scrollbar, aktif edilirse listenin sağ kenarında bir dikey kaydırma çubuğu çizer. |
| `ScrollbarTrackStyle` | `cell.Style` | ScrollbarTrackStyle, kaydırma çubuğu rayının (track) stilidir. |
| `ScrollbarThumbStyle` | `cell.Style` | ScrollbarThumbStyle, kaydırma çubuğu kaydırıcısının (thumb) stilidir. |
| `Style` | `cell.Style` | Style, listenin genel rengini ve yazı stilini belirtir. |
| `FocusedStyle` | `cell.Style` | FocusedStyle, liste odağa sahip olduğunda uygulanacak stildir. |
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
| `ID` | `string` | ID, widget odak kimliğidir. |
| `Text` | `string` | Text, gösterilecek olan metin içeriğidir. |
| `Style` | `cell.Style` | Style, metnin yazı rengi, arka planı ve modifikatör stillerini belirler. |
| `FocusedStyle` | `cell.Style` | FocusedStyle, paragraf odaklandığında uygulanacak stildir. |
| `Wrap` | `bool` | Wrap, metnin sınır genişliğine göre otomatik olarak alt satıra kaydırılıp kaydırılmayacağını belirler. |

## PieChart

PieChart renders pie and donut charts using Braille subpixels and color legends.

`RenderWidget` çağrısında kullanılacak tip: `PieChart`

| Alan | Tip | Açıklama |
| --- | --- | --- |
| `ID` | `string` |  |
| `Data` | `[]PieSlice` |  |
| `DonutHoleRatio` | `float64` |  |
| `ShowLegend` | `bool` |  |
| `ShowPercentages` | `bool` |  |
| `Style` | `cell.Style` |  |

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
| `FocusedStyle` | `cell.Style` |  |
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
| `DisableScroll` | `bool` |  |
| `DisableFocus` | `bool` |  |
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
| `DisableScroll` | `bool` |  |
| `DisableFocus` | `bool` |  |
| `OnChange` | `func(...)` |  |

## Sparkline

`RenderWidget` çağrısında kullanılacak tip: `Sparkline`

| Alan | Tip | Açıklama |
| --- | --- | --- |
| `ID` | `string` | ID, widget odak kimliğidir. |
| `Data` | `[]float64` | Data, çizilecek veri geçmişini temsil eden sayılar dizisidir. |
| `Style` | `cell.Style` | Style, varsayılan hücre stilini tanımlar. |
| `FocusedStyle` | `cell.Style` | FocusedStyle, odaklandığında uygulanacak stildir. |
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
| `FocusedStyle` | `cell.Style` |  |
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
| `ID` | `string` |  |
| `Lines` | `[]Line` |  |
| `Style` | `cell.Style` |  |
| `FocusedStyle` | `cell.Style` |  |
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

## ToastManager

ToastManager manages a stack of auto-dismissing toast notifications.

`RenderWidget` çağrısında kullanılacak tip: `*ToastManager`

| Alan | Tip | Açıklama |
| --- | --- | --- |
| `Toasts` | `[]*ToastItem` |  |
| `Position` | `ToastPosition` |  |
| `MaxVisible` | `int` |  |

## Transducer

`RenderWidget` çağrısında kullanılacak tip: `Transducer`

| Alan | Tip | Açıklama |
| --- | --- | --- |
| `Child` | `Widget` |  |
| `Type` | `TransducerType` |  |
| `Progress` | `float64` |  |

## TreeView

TreeView renders a hierarchical collapsible tree with guide lines and selection.

`RenderWidget` çağrısında kullanılacak tip: `TreeView`

| Alan | Tip | Açıklama |
| --- | --- | --- |
| `ID` | `string` |  |
| `Roots` | `[]TreeNode` |  |
| `State` | `*TreeViewState` |  |
| `Style` | `cell.Style` |  |
| `FocusedStyle` | `cell.Style` |  |
| `SelectedStyle` | `cell.Style` |  |
| `GuideStyle` | `cell.Style` |  |
| `ShowGuides` | `bool` |  |
| `IndentWidth` | `int` |  |

## Viewer3D

Viewer3D is a high-level widget that renders 3D models with rotation, lighting, shading (Wireframe, Solid, Lambertian, Gouraud), and texture mapping.

`RenderWidget` çağrısında kullanılacak tip: `*Viewer3D`

| Alan | Tip | Açıklama |
| --- | --- | --- |
| `ID` | `string` | ID is the widget focus identifier. |
| `Model` | `graphics.Model3D` | Model is the 3D geometry to render. |
| `ImagePath` | `string` | ImagePath is the optional file path to a PNG/JPEG texture. |
| `Image` | `image.Image` | Image is an optional in-memory texture image to map onto the 3D model. |
| `RotX` | `float64` | RotX, RotY, RotZ are the Euler rotation angles in degrees. |
| `RotY` | `float64` |  |
| `RotZ` | `float64` |  |
| `Distance` | `float64` | Distance is the camera distance from the object (default: 3.5). |
| `Scale` | `float64` | Scale is the zoom/scale multiplier (default: 1.0). |
| `Shading` | `string` | Shading mode: "Dokulu" (Texture mapped), "Wireframe", "Dolu Renkli" (Flat), "Gölgeli" (Lambertian), "Gouraud" (Smooth interpolated). |
| `Wireframe` | `bool` | Wireframe overlays edges on top of shaded faces. |
| `WireframeStyle` | `cell.Style` | WireframeStyle is the cell style for wireframe lines. |
| `FocusedStyle` | `cell.Style` | FocusedStyle is applied to wireframe/highlight when the viewer is focused. |
| `Light` | `graphics.Light` | Light is the directional light source for Lambertian and Gouraud shading. |
| `FaceColors` | `[]cell.Color` | FaceColors is an optional palette for coloring distinct faces in solid mode. |

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
| `FocusedStyle` | `cell.Style` |  |
| `EmptyText` | `string` |  |
| `LoadingText` | `string` |  |
| `ErrorText` | `string` |  |
| `Offset` | `*int` |  |
| `HorizontalOffset` | `int` | HorizontalOffset scrolls non-sticky cell text by terminal columns. |
| `StickyColumns` | `int` | StickyColumns keeps the first N Row.Cells visible while the remaining cells are horizontally scrolled. |
| `OnSelect` | `func(...)` | OnSelect is called with the virtual row index after a row is clicked. |

