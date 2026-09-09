# Widget Gallery and API Reference

This document is generated from source code in the `widgets` package; each component implements the `Widget` interface (`Draw(cell.Context, *buffer.Buffer)`).

| Widget | Receiver Type | Field Count |
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

Type to pass to `RenderWidget`: `Ascii3D`

| Field | Type | Description |
| --- | --- | --- |
| `Model` | `graphics.Model3D` | 3D Model geometry or file path |
| `Src` | `string` |  |
| `Mode` | `Ascii3DMode` | Rendering Mode: - ModeASCII: Typography character ramps (default) - ModeBlock: 2x vertical sub-cell Half-Block (▀/▄) - ModeDithered: Retro Bayer 4x4 ordered dithering - ModeBraille: 8x sub-pixel Unicode Braille dot matrix |
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

Type to pass to `RenderWidget`: `BarChart`

| Field | Type | Description |
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

Block is the foundational container widget capable of rendering borders, filling backgrounds, and hosting titles.

Type to pass to `RenderWidget`: `Block`

| Field | Type | Description |
| --- | --- | --- |
| `Title` | `string` | Title is the title string displayed on the top border of the block. |
| `TitleAlignment` | `Alignment` | TitleAlignment sets the horizontal alignment of the title on the border (Left, Center, Right). |
| `TitleStyle` | `cell.Style` | TitleStyle defines the color and style modifiers for the title text. |
| `Borders` | `uint8` | Borders bitmask determining which edges are rendered (e.g. BorderAll). |
| `BorderSymbols` | `BorderSymbols` | BorderSymbols defines the glyph set used for border rendering. |
| `BorderStyle` | `cell.Style` | BorderStyle defines the color and style of the border lines. |
| `Margin` | `Insets` | Margin is the CSS-like outer spacing around the block. |
| `Padding` | `Insets` | Padding is the CSS-like inner spacing between content and borders. |
| `PaddingLeft` | `uint16` |  |
| `PaddingRight` | `uint16` |  |
| `PaddingTop` | `uint16` |  |
| `PaddingBottom` | `uint16` |  |
| `Style` | `cell.Style` | Style defines the background fill color and default styling for the block. |
| `Child` | `Widget` | Child is the subordinate visual component rendered within the block. |
| `Opaque` | `bool` | Opaque, if true, adds a solid color layer to prevent background terminal graphics from leaking through. |

## Canvas

Canvas is a visual component enabling high-resolution vector rendering on the terminal at 2x4 virtual sub-pixels per cell using Braille glyphs.

Type to pass to `RenderWidget`: `*Canvas`

_No fields._

## Checkbox

Checkbox is an interactive check box widget.

Type to pass to `RenderWidget`: `Checkbox`

| Field | Type | Description |
| --- | --- | --- |
| `ID` | `string` |  |
| `Checked` | `*bool` |  |
| `Label` | `string` |  |
| `Style` | `cell.Style` |  |
| `FocusedStyle` | `cell.Style` |  |

## ColorPicker

ColorPicker is a rich, KDE / desktop-style 2D HSV graphical color picker.

Type to pass to `RenderWidget`: `ColorPicker`

| Field | Type | Description |
| --- | --- | --- |
| `ID` | `string` |  |
| `State` | `*ColorPickerState` |  |
| `Palette` | `[]cell.Color` |  |
| `ShowPreview` | `bool` |  |
| `Style` | `cell.Style` |  |

## CommandPalette

CommandPalette is an interactive command palette overlay widget.

Type to pass to `RenderWidget`: `CommandPalette`

| Field | Type | Description |
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

Type to pass to `RenderWidget`: `DevTools`

| Field | Type | Description |
| --- | --- | --- |
| `State` | `*DevToolsState` |  |
| `Style` | `cell.Style` |  |

## Dialog

Dialog is a premium, modern glassmorphism dialog widget with glowing gradient borders and blended shadows.

Type to pass to `RenderWidget`: `Dialog`

| Field | Type | Description |
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

Image is a TUI component capable of rendering true images (PNG/JPG) using native terminal graphics protocols (Kitty, Sixel, iTerm2).

Type to pass to `RenderWidget`: `*Image`

| Field | Type | Description |
| --- | --- | --- |
| `ID` | `string` | ID is the widget focus identifier. |
| `Img` | `image.Image` | Img is the raw image object to display. |
| `ZIndex` | `int` | ZIndex is the vertical layer stacking order of the image. |
| `ForceHalfBlock` | `bool` | ForceHalfBlock, if enabled, forces cell-based half-block rendering instead of hardware protocols. |
| `CircleMask` | `bool` | CircleMask clips the image into a circular mask (e.g. for avatars). |
| `OpaqueBackground` | `bool` | OpaqueBackground composites transparency over Background before native rendering. |
| `Background` | `cell.Color` |  |
| `Transparent` | `bool` | Transparent specifies whether transparent pixels of the image should be preserved. |
| `Opacity` | `float64` | Opacity is the opacity value of the image (between 0.0 and 1.0). |
| `OpacitySet` | `bool` | OpacitySet indicates that the Opacity field has been explicitly configured. |
| `FocusedStyle` | `cell.Style` | FocusedStyle is the border/highlight style applied when focused. |

## LineChart

LineChart renders smooth Braille-based multi-series line graphs with labeled axes.

Type to pass to `RenderWidget`: `LineChart`

| Field | Type | Description |
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

List is an interactive widget rendering vertical items in a list format.

Type to pass to `RenderWidget`: `List`

| Field | Type | Description |
| --- | --- | --- |
| `ID` | `string` | ID is the focus and identification ID of the list. |
| `Items` | `[]string` | Items are the string items displayed in the list. |
| `Provider` | `ListProvider` | Provider is the data provider for virtual scrolling. |
| `Scrollbar` | `bool` | Scrollbar, if enabled, renders a vertical scrollbar on the right edge of the list. |
| `ScrollbarTrackStyle` | `cell.Style` | ScrollbarTrackStyle is the style for the scrollbar track. |
| `ScrollbarThumbStyle` | `cell.Style` | ScrollbarThumbStyle is the style for the scrollbar thumb. |
| `Style` | `cell.Style` | Style defines the overall color and typography of the list. |
| `FocusedStyle` | `cell.Style` | FocusedStyle is the style applied when the list is focused. |
| `SelectedStyle` | `cell.Style` | SelectedStyle is the style used to highlight the currently selected item. |
| `HighlightSymbol` | `string` | HighlightSymbol is the symbol placed to the left of the selected item (e.g. "> "). |
| `State` | `*ListState` | State is the pointer maintaining the selected index and scroll offset of the list. |

## Markdown

Type to pass to `RenderWidget`: `*Markdown`

| Field | Type | Description |
| --- | --- | --- |
| `ID` | `string` |  |
| `Content` | `string` | Content is the raw markdown string to parse and render. |
| `Style` | `cell.Style` | Style defines the default text style. |
| `FocusedStyle` | `cell.Style` |  |
| `ScrollOffset` | `*int` |  |

## Paragraph

Paragraph is a visual component for displaying multi-line text.

Type to pass to `RenderWidget`: `*Paragraph`

| Field | Type | Description |
| --- | --- | --- |
| `ID` | `string` | ID is the widget focus identifier. |
| `Text` | `string` | Text is the text content to display. |
| `Style` | `cell.Style` | Style determines text color, background color, and modifier styles. |
| `FocusedStyle` | `cell.Style` | FocusedStyle is the style applied when the paragraph is focused. |
| `Wrap` | `bool` | Wrap determines whether text is automatically wrapped to fit bounded width. |

## PieChart

PieChart renders pie and donut charts using Braille subpixels and color legends.

Type to pass to `RenderWidget`: `PieChart`

| Field | Type | Description |
| --- | --- | --- |
| `ID` | `string` |  |
| `Data` | `[]PieSlice` |  |
| `DonutHoleRatio` | `float64` |  |
| `ShowLegend` | `bool` |  |
| `ShowPercentages` | `bool` |  |
| `Style` | `cell.Style` |  |

## Popup

Popup is an interactive dropdown menu widget.

Type to pass to `RenderWidget`: `Popup`

| Field | Type | Description |
| --- | --- | --- |
| `ID` | `string` | ID is the unique identifier of the popup. |
| `Label` | `string` | Label is the initial text displayed on the button. |
| `Items` | `[]PopupItem` | Items is the list of items in the dropdown menu. |
| `State` | `*PopupState` | State manages the open/closed state and selection of the popup. |
| `Style` | `cell.Style` | Style defines the button and menu background style. |
| `ItemStyle` | `cell.Style` | ItemStyle defines the default style for menu items. |
| `SelectedStyle` | `cell.Style` | SelectedStyle is the style applied when a menu item is hovered or selected via keyboard. |
| `DisabledStyle` | `cell.Style` | DisabledStyle is the style for disabled menu items. |
| `BorderStyle` | `cell.Style` | BorderStyle defines the border style of the menu. |
| `BorderSymbols` | `BorderSymbols` | BorderSymbols defines the border glyph set. |

## ProgressBar

ProgressBar renders a bounded horizontal progress indicator.

Type to pass to `RenderWidget`: `ProgressBar`

| Field | Type | Description |
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

RadioButton is a widget for selecting a single option from a group of choices.

Type to pass to `RenderWidget`: `RadioButton`

| Field | Type | Description |
| --- | --- | --- |
| `ID` | `string` |  |
| `Selected` | `*string` |  |
| `Value` | `string` |  |
| `Label` | `string` |  |
| `Style` | `cell.Style` |  |
| `FocusedStyle` | `cell.Style` |  |

## Select

Select is a keyboard- and mouse-interactive dropdown field.

Type to pass to `RenderWidget`: `Select`

| Field | Type | Description |
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

Type to pass to `RenderWidget`: `Slider`

| Field | Type | Description |
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

Type to pass to `RenderWidget`: `Sparkline`

| Field | Type | Description |
| --- | --- | --- |
| `ID` | `string` | ID is the widget focus identifier. |
| `Data` | `[]float64` | Data is a slice of numerical values representing historical data trend. |
| `Style` | `cell.Style` | Style defines the default cell style. |
| `FocusedStyle` | `cell.Style` | FocusedStyle is the style applied when focused. |
| `Color` | `cell.Color` | Color sets the color of the sparkline bars. |

## Table

Table is an interactive, flex-column, vertically scrollable table component with cell-spanning support.

Type to pass to `RenderWidget`: `Table`

| Field | Type | Description |
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

Type to pass to `RenderWidget`: `Text`

| Field | Type | Description |
| --- | --- | --- |
| `ID` | `string` |  |
| `Lines` | `[]Line` |  |
| `Style` | `cell.Style` |  |
| `FocusedStyle` | `cell.Style` |  |
| `Wrap` | `bool` |  |
| `Alignment` | `TextAlignment` |  |

## TextArea

TextArea is a multiline focusable text editor.

Type to pass to `RenderWidget`: `TextArea`

| Field | Type | Description |
| --- | --- | --- |
| `ID` | `string` |  |
| `State` | `*TextAreaState` |  |
| `Style` | `cell.Style` |  |
| `FocusedStyle` | `cell.Style` |  |

## TextInput

TextInput is an interactive single-line text input field.

Type to pass to `RenderWidget`: `TextInput`

| Field | Type | Description |
| --- | --- | --- |
| `ID` | `string` |  |
| `State` | `*TextInputState` |  |
| `Placeholder` | `string` |  |
| `Style` | `cell.Style` |  |
| `PlaceholderStyle` | `cell.Style` |  |
| `FocusedStyle` | `cell.Style` |  |

## ToastManager

ToastManager manages a stack of auto-dismissing toast notifications.

Type to pass to `RenderWidget`: `*ToastManager`

| Field | Type | Description |
| --- | --- | --- |
| `Toasts` | `[]*ToastItem` |  |
| `Position` | `ToastPosition` |  |
| `MaxVisible` | `int` |  |

## Transducer

Type to pass to `RenderWidget`: `Transducer`

| Field | Type | Description |
| --- | --- | --- |
| `Child` | `Widget` |  |
| `Type` | `TransducerType` |  |
| `Progress` | `float64` |  |

## TreeView

TreeView renders a hierarchical collapsible tree with guide lines and selection.

Type to pass to `RenderWidget`: `TreeView`

| Field | Type | Description |
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

Type to pass to `RenderWidget`: `*Viewer3D`

| Field | Type | Description |
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
| `Shading` | `string` | Shading mode: "textured" (Texture mapped), "wireframe", "solid" (Flat), "shaded" (Lambertian), "gouraud" (Smooth interpolated). |
| `Wireframe` | `bool` | Wireframe overlays edges on top of shaded faces. |
| `WireframeStyle` | `cell.Style` | WireframeStyle is the cell style for wireframe lines. |
| `FocusedStyle` | `cell.Style` | FocusedStyle is applied to wireframe/highlight when the viewer is focused. |
| `Light` | `graphics.Light` | Light is the directional light source for Lambertian and Gouraud shading. |
| `FaceColors` | `[]cell.Color` | FaceColors is an optional palette for coloring distinct faces in solid mode. |

## VirtualDataView

VirtualDataView renders the visible portion of a VirtualDataState cache.

Type to pass to `RenderWidget`: `VirtualDataView`

| Field | Type | Description |
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

