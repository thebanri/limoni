# Widget Gallery and API Reference

This file is generated from the source of the `widgets` package; every widget implements
the `Widget` interface (`Draw(cell.Context, *buffer.Buffer)`).

| Widget | Receiver type | Fields |
| --- | --- | --- |
| [Ascii3D](#ascii3d) | `Ascii3D` | 32 |
| [Autocomplete](#autocomplete) | `Autocomplete` | 10 |
| [BarChart](#barchart) | `BarChart` | 12 |
| [BigText](#bigtext) | `BigText` | 5 |
| [Block](#block) | `Block` | 16 |
| [Button](#button) | `Button` | 6 |
| [Calendar](#calendar) | `Calendar` | 15 |
| [Canvas](#canvas) | `*Canvas` | 1 |
| [Checkbox](#checkbox) | `Checkbox` | 5 |
| [CodeView](#codeview) | `CodeView` | 7 |
| [ColorPicker](#colorpicker) | `ColorPicker` | 5 |
| [CommandPalette](#commandpalette) | `CommandPalette` | 10 |
| [DevTools](#devtools) | `DevTools` | 2 |
| [Dialog](#dialog) | `Dialog` | 15 |
| [DiffView](#diffview) | `DiffView` | 8 |
| [FilePicker](#filepicker) | `FilePicker` | 10 |
| [Gauge](#gauge) | `Gauge` | 7 |
| [HelpOverlay](#helpoverlay) | `*HelpOverlay` | 10 |
| [Image](#image) | `*Image` | 11 |
| [Label](#label) | `Label` | 2 |
| [LineChart](#linechart) | `LineChart` | 11 |
| [LineGauge](#linegauge) | `LineGauge` | 9 |
| [List](#list) | `List` | 13 |
| [LogView](#logview) | `*LogView` | 11 |
| [Markdown](#markdown) | `*Markdown` | 8 |
| [Paragraph](#paragraph) | `*Paragraph` | 5 |
| [PieChart](#piechart) | `PieChart` | 7 |
| [Popup](#popup) | `Popup` | 10 |
| [ProgressBar](#progressbar) | `ProgressBar` | 9 |
| [RadioButton](#radiobutton) | `RadioButton` | 6 |
| [Scrollbar](#scrollbar) | `Scrollbar` | 10 |
| [Select](#select) | `Select` | 12 |
| [Slider](#slider) | `Slider` | 12 |
| [Sparkline](#sparkline) | `Sparkline` | 5 |
| [Spinner](#spinner) | `Spinner` | 8 |
| [SplitPane](#splitpane) | `SplitPane` | 10 |
| [StatusBar](#statusbar) | `StatusBar` | 7 |
| [Table](#table) | `Table` | 16 |
| [Tabs](#tabs) | `Tabs` | 11 |
| [Text](#text) | `Text` | 6 |
| [TextArea](#textarea) | `TextArea` | 6 |
| [TextInput](#textinput) | `TextInput` | 12 |
| [ToastManager](#toastmanager) | `*ToastManager` | 3 |
| [Transducer](#transducer) | `Transducer` | 3 |
| [TreeView](#treeview) | `TreeView` | 9 |
| [Viewer3D](#viewer3d) | `*Viewer3D` | 18 |
| [Viewport](#viewport) | `Viewport` | 11 |
| [VirtualDataView](#virtualdataview) | `VirtualDataView` | 15 |

## Ascii3D

Ascii3D (or AsciiObject) is a high-performance 3D vector-to-ASCII terminal renderer.

Type to pass to `RenderWidget`: `Ascii3D`

| Field | Type | Description |
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

## Autocomplete

Autocomplete is a text input with a list of suggestions under it that narrows as the user types, matched fuzzily the way the command palette matches: "gco" finds "git checkout".

Type to pass to `RenderWidget`: `Autocomplete`

| Field | Type | Description |
| --- | --- | --- |
| `ID` | `string` |  |
| `Suggestions` | `[]string` |  |
| `State` | `*AutocompleteState` |  |
| `Placeholder` | `string` |  |
| `MaxVisible` | `int` | MaxVisible caps the rows of suggestions; 0 is 6. |
| `Style` | `cell.Style` |  |
| `PlaceholderStyle` | `cell.Style` |  |
| `FocusedStyle` | `cell.Style` |  |
| `SuggestionStyle` | `cell.Style` |  |
| `SelectedStyle` | `cell.Style` |  |

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

## BigText

BigText draws text in large letters from an 8×8 bitmap font, for titles, clocks and counters:  	██████   ████ 	  ██    ██  ██ 	  ██    ██  ██ ...

Type to pass to `RenderWidget`: `BigText`

| Field | Type | Description |
| --- | --- | --- |
| `ID` | `string` |  |
| `Text` | `string` |  |
| `Size` | `BigTextSize` |  |
| `Alignment` | `Alignment` |  |
| `Style` | `cell.Style` |  |

## Block

Block is the most basic container widget: it can draw a border, fill a background and carry a Title on top.

Type to pass to `RenderWidget`: `Block`

| Field | Type | Description |
| --- | --- | --- |
| `Title` | `string` | Title is the text shown on the block's top border. |
| `TitleAlignment` | `Alignment` | TitleAlignment sets where the title sits on the border (Left, Center, Right). |
| `TitleStyle` | `cell.Style` | TitleStyle sets the title's colour and style. |
| `Borders` | `uint8` | Borders is the mask of which sides to draw (e.g. |
| `BorderSymbols` | `BorderSymbols` | BorderSymbols are the glyphs the border is drawn with (e.g. |
| `MergeBorders` | `bool` | MergeBorders joins this block's border with light box-drawing characters already in the buffer, so two adjacent blocks share one edge and meet in a junction (├ ┤ ┬ ┴ ┼) instead of one line overwriting the other. |
| `BorderStyle` | `cell.Style` | BorderStyle sets the colour and style of the border lines. |
| `Margin` | `Insets` | Margin is CSS-like space outside the block. |
| `Padding` | `Insets` | Padding is CSS-like space between the content and the border. |
| `PaddingLeft` | `uint16` |  |
| `PaddingRight` | `uint16` |  |
| `PaddingTop` | `uint16` |  |
| `PaddingBottom` | `uint16` |  |
| `Style` | `cell.Style` | Style sets the block's background fill and default style. |
| `Child` | `Widget` | Child is the component drawn inside the block. |
| `Opaque` | `bool` | Opaque, when true, adds a solid-colour image layer behind the block so native images do not show through it. |

## Button

Button is a labelled push button: "[ Save ]".

Type to pass to `RenderWidget`: `Button`

| Field | Type | Description |
| --- | --- | --- |
| `ID` | `string` |  |
| `Label` | `string` |  |
| `OnPress` | `func(...)` |  |
| `Disabled` | `bool` | Disabled buttons draw dimmed, ignore clicks and say so in the tree. |
| `Style` | `cell.Style` |  |
| `FocusedStyle` | `cell.Style` | FocusedStyle defaults to reverse video and bold. |

## Calendar

Calendar shows one month as a grid of days, seven columns of three cells:  	 September 2026 	Mo Tu We Th Fr Sa Su 	    1  2  3  4  5  6 	 7  8  9 10 11 12 13 	...

Type to pass to `RenderWidget`: `Calendar`

| Field | Type | Description |
| --- | --- | --- |
| `ID` | `string` |  |
| `State` | `*CalendarState` | State holds the selected day; the calendar shows its month. |
| `Month` | `time.Time` |  |
| `Today` | `time.Time` | Today, if set, is drawn in TodayStyle. |
| `FirstWeekday` | `time.Weekday` | FirstWeekday is the column the week starts in; the zero value is Sunday. |
| `MonthNames` | `*[]string` | MonthNames and WeekdayNames replace the English ones, Sunday first. |
| `WeekdayNames` | `*[]string` |  |
| `HideHeader` | `bool` | HideHeader drops the month and year line; HideWeekdays the day names. |
| `HideWeekdays` | `bool` |  |
| `DayStyle` | `func(...)` | DayStyle, if set, is asked about every day shown; returning ok merges the style over the day, for marking days with events. |
| `Style` | `cell.Style` |  |
| `HeaderStyle` | `cell.Style` |  |
| `WeekdayStyle` | `cell.Style` |  |
| `TodayStyle` | `cell.Style` |  |
| `SelectedStyle` | `cell.Style` |  |

## Canvas

Canvas draws high-resolution vector graphics in the terminal, at 2x4 virtual pixels per cell (using Braille characters).

Type to pass to `RenderWidget`: `*Canvas`

| Field | Type | Description |
| --- | --- | --- |
| `Marker` | `Marker` | Marker picks the characters the canvas is drawn with; the zero value is Braille. |

## Checkbox

Checkbox is an interactive box that can be ticked.

Type to pass to `RenderWidget`: `Checkbox`

| Field | Type | Description |
| --- | --- | --- |
| `ID` | `string` |  |
| `Checked` | `*bool` |  |
| `Label` | `string` |  |
| `Style` | `cell.Style` |  |
| `FocusedStyle` | `cell.Style` | FocusedStyle defaults to reverse bold, or the theme focus style when both styles are unset. |

## CodeView

CodeView shows source code with line numbers and syntax highlighting.

Type to pass to `RenderWidget`: `CodeView`

| Field | Type | Description |
| --- | --- | --- |
| `ID` | `string` |  |
| `State` | `*CodeViewState` |  |
| `Theme` | `*CodeTheme` | Theme styles each token kind; nil is DefaultCodeTheme. |
| `HideLineNumbers` | `bool` | HideLineNumbers drops the gutter. |
| `Style` | `cell.Style` |  |
| `GutterStyle` | `cell.Style` |  |
| `CursorStyle` | `cell.Style` |  |

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

CommandPalette is the command palette overlay widget.

Type to pass to `RenderWidget`: `CommandPalette`

| Field | Type | Description |
| --- | --- | --- |
| `ID` | `string` |  |
| `State` | `*CommandPaletteState` |  |
| `Position` | `*CommandPalettePosition` |  |
| `Style` | `cell.Style` |  |
| `InputStyle` | `cell.Style` |  |
| `ItemStyle` | `cell.Style` |  |
| `SelectedStyle` | `cell.Style` |  |
| `DetailStyle` | `cell.Style` |  |
| `Title` | `string` | Title is drawn in the top border. |
| `Placeholder` | `string` | Placeholder is shown while the query is empty. |

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
| `State` | `*DialogState` | State keeps the buttons' IDs, labels and mouse handlers between frames. |

## DiffView

DiffView shows a DiffViewState: a gutter of line numbers, a + or - for each line added or removed, those lines on a green or red background, and the code highlighted as in CodeView.

Type to pass to `RenderWidget`: `DiffView`

| Field | Type | Description |
| --- | --- | --- |
| `ID` | `string` |  |
| `State` | `*DiffViewState` |  |
| `Theme` | `*CodeTheme` | Theme styles each token kind; nil is DefaultCodeTheme. |
| `HideLineNumbers` | `bool` | HideLineNumbers drops the gutter. |
| `Style` | `cell.Style` |  |
| `GutterStyle` | `cell.Style` |  |
| `AddedStyle` | `cell.Style` | AddedStyle and RemovedStyle are merged over added and removed lines; zero is a dark green and a dark red background. |
| `RemovedStyle` | `cell.Style` | AddedStyle and RemovedStyle are merged over added and removed lines; zero is a dark green and a dark red background. |

## FilePicker

FilePicker lists a directory for choosing a file or a folder: the path on the first row, then ".." and the entries, directories first and marked with a trailing slash, file sizes on the right.

Type to pass to `RenderWidget`: `FilePicker`

| Field | Type | Description |
| --- | --- | --- |
| `ID` | `string` |  |
| `State` | `*FilePickerState` |  |
| `HideHeader` | `bool` | HideHeader drops the path row; HideSizes the size column. |
| `HideSizes` | `bool` |  |
| `Style` | `cell.Style` |  |
| `HeaderStyle` | `cell.Style` |  |
| `DirStyle` | `cell.Style` |  |
| `SizeStyle` | `cell.Style` |  |
| `SelectedStyle` | `cell.Style` |  |
| `ErrorStyle` | `cell.Style` |  |

## Gauge

Gauge fills its whole area in proportion to Ratio, with a label centred on it.

Type to pass to `RenderWidget`: `Gauge`

| Field | Type | Description |
| --- | --- | --- |
| `ID` | `string` |  |
| `Ratio` | `float64` | Ratio is the filled fraction, clamped to [0, 1]. |
| `Label` | `string` | Label is drawn in the middle. |
| `HideLabel` | `bool` | HideLabel draws no label at all. |
| `Style` | `cell.Style` | Style is the unfilled part and the label. |
| `GaugeStyle` | `cell.Style` | GaugeStyle is the filled part; its Fg is the fill colour. |
| `WholeCells` | `bool` | WholeCells turns off the eighth-block edge, for fonts that draw the partial blocks badly. |

## HelpOverlay

HelpOverlay is a keyboard help card built from a KeybindingManager: the "?" screen of a terminal application.

Type to pass to `RenderWidget`: `*HelpOverlay`

| Field | Type | Description |
| --- | --- | --- |
| `ID` | `string` | ID is the card's focus and identity ID. |
| `Label` | `string` | Label names the card in the semantic tree. |
| `Manager` | `*KeybindingManager` | Manager supplies the bindings. |
| `Scopes` | `[]string` | Scopes limits the card to bindings in these scopes. |
| `Title` | `string` | Title is drawn above the groups when it is not empty. |
| `Columns` | `int` | Columns caps the number of columns. |
| `State` | `*HelpState` | State points to the card's scroll offset and layout buffers. |
| `Style` | `cell.Style` | Style is the card's overall style. |
| `CategoryStyle` | `cell.Style` |  |
| `KeyStyle` | `cell.Style` |  |

## Image

Image draws real pictures such as PNG/JPG in the terminal, using the native image protocols (Kitty, Sixel, iTerm2).

Type to pass to `RenderWidget`: `*Image`

| Field | Type | Description |
| --- | --- | --- |
| `ID` | `string` | ID is the widget's focus ID. |
| `Img` | `image.Image` | Img is the raw image to show. |
| `ZIndex` | `int` | ZIndex is the image's layer order. |
| `ForceHalfBlock` | `bool` | ForceHalfBlock, when set, forces the cell-based half-block method instead of the hardware protocols. |
| `CircleMask` | `bool` | CircleMask crops the image to a circle (for avatars). |
| `OpaqueBackground` | `bool` | OpaqueBackground composites transparency over Background before native rendering. |
| `Background` | `cell.Color` |  |
| `Transparent` | `bool` | Transparent reports whether the image's transparent pixels are kept. |
| `Opacity` | `float64` | Opacity is the image's opacity (between 0.0 and 1.0). |
| `OpacitySet` | `bool` | OpacitySet reports that Opacity was set on purpose, so that 0.0 can be told apart from the default (unset) value. |
| `FocusedStyle` | `cell.Style` | FocusedStyle is the border/highlight style applied when focused. |

## Label

Label is a lightweight stateless widget for rendering single- or multi-line text.

Type to pass to `RenderWidget`: `Label`

| Field | Type | Description |
| --- | --- | --- |
| `Text` | `string` |  |
| `Style` | `cell.Style` |  |

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
| `Marker` | `Marker` | Marker picks the characters the lines are drawn with (Braille by default). |

## LineGauge

LineGauge is a one-row gauge: an optional label, then a line drawn heavy up to Ratio and light after it.

Type to pass to `RenderWidget`: `LineGauge`

| Field | Type | Description |
| --- | --- | --- |
| `ID` | `string` |  |
| `Ratio` | `float64` | Ratio is the filled fraction, clamped to [0, 1]. |
| `Label` | `string` | Label is drawn before the line, followed by a space. |
| `HideLabel` | `bool` | HideLabel draws the line alone. |
| `Style` | `cell.Style` | Style is the label; FilledStyle and UnfilledStyle the two parts of the line. |
| `FilledStyle` | `cell.Style` |  |
| `UnfilledStyle` | `cell.Style` |  |
| `FilledRune` | `rune` | FilledRune and UnfilledRune replace ━ and ─. |
| `UnfilledRune` | `rune` |  |

## List

List is an interactive widget that draws items as a vertical list.

Type to pass to `RenderWidget`: `List`

| Field | Type | Description |
| --- | --- | --- |
| `ID` | `string` | ID is the list's focus and identity ID. |
| `Label` | `string` | Label names the list in the semantic tree — what a screen reader announces and what an automation selector matches. |
| `Items` | `[]string` | Items are the strings shown in the list. |
| `Provider` | `ListProvider` | Provider supplies the data for a virtual (virtually scrolled) list. |
| `Scrollbar` | `bool` | Scrollbar, when set, draws a vertical scrollbar on the right edge of the list. |
| `ScrollbarTrackStyle` | `cell.Style` | ScrollbarTrackStyle is the style of the scrollbar track. |
| `ScrollbarThumbStyle` | `cell.Style` | ScrollbarThumbStyle is the style of the scrollbar thumb. |
| `Style` | `cell.Style` | Style is the list's overall colour and text style. |
| `FocusedStyle` | `cell.Style` | FocusedStyle is the style applied when the list has the focus. |
| `SelectedStyle` | `cell.Style` | SelectedStyle is the style the selected item is highlighted with. |
| `HighlightSymbol` | `string` | HighlightSymbol is the symbol placed left of the selected item (e.g. |
| `HighlightSpacing` | `bool` | HighlightSpacing keeps the symbol's width free on every row, so the text of the selected row lines up with the rest instead of moving right when it is selected. |
| `State` | `*ListState` | State points to the list's selected index and scroll state. |

## LogView

LogView shows a long, growing list of log lines: scrolled virtually, so only the lines on screen are touched; coloured by level when the source knows levels; with an optional line-number gutter and highlighted search text.

Type to pass to `RenderWidget`: `*LogView`

| Field | Type | Description |
| --- | --- | --- |
| `ID` | `string` |  |
| `Label` | `string` |  |
| `Source` | `LogSource` |  |
| `State` | `*LogViewState` |  |
| `LineNumbers` | `bool` | LineNumbers draws a gutter with 1-based line numbers. |
| `Highlight` | `string` | Highlight marks every case-insensitive occurrence of this text. |
| `Style` | `cell.Style` |  |
| `SelectedStyle` | `cell.Style` |  |
| `HighlightStyle` | `cell.Style` |  |
| `GutterStyle` | `cell.Style` |  |
| `LevelStyles` | `[]cell.Style` | LevelStyles overrides the colour of each level; zero entries keep the default. |

## Markdown

Type to pass to `RenderWidget`: `*Markdown`

| Field | Type | Description |
| --- | --- | --- |
| `ID` | `string` |  |
| `Content` | `string` | Content is the raw markdown text to parse and draw. |
| `Style` | `cell.Style` | Style is the default text style. |
| `FocusedStyle` | `cell.Style` |  |
| `ScrollOffset` | `*int` |  |
| `Theme` | `*MarkdownTheme` | Theme colours headings, links, code, quotes and the rest; nil is DefaultMarkdownTheme. |
| `Images` | `func(...)` | Images returns the picture an image refers to — the src of ![alt](src) — or nil while there is none to show. |
| `MaxImageRows` | `int` | MaxImageRows caps the height of a picture in rows; zero is 20. |

## Paragraph

Paragraph shows multi-line text.

Type to pass to `RenderWidget`: `*Paragraph`

| Field | Type | Description |
| --- | --- | --- |
| `ID` | `string` | ID is the widget's focus ID. |
| `Text` | `string` | Text is the text to show. |
| `Style` | `cell.Style` | Style sets the text's colour, background and modifiers. |
| `FocusedStyle` | `cell.Style` | FocusedStyle is the style applied when the paragraph is focused. |
| `Wrap` | `bool` | Wrap sets whether the text wraps to the next line at the bounds' width. |

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
| `Marker` | `Marker` | Marker picks the characters the slices are drawn with (Braille by default); MarkerSextant or MarkerQuadrant fill them solid. |

## Popup

Popup is a dropdown menu widget.

Type to pass to `RenderWidget`: `Popup`

| Field | Type | Description |
| --- | --- | --- |
| `ID` | `string` | ID uniquely identifies the popup. |
| `Label` | `string` | Label is the text on the button. |
| `Items` | `[]PopupItem` | Items is the list of menu items. |
| `State` | `*PopupState` | State holds whether the popup is open and what is selected. |
| `Style` | `cell.Style` | Style sets the button and menu background style. |
| `ItemStyle` | `cell.Style` | ItemStyle sets the normal style of the menu items. |
| `SelectedStyle` | `cell.Style` | SelectedStyle is the style of a menu item under the mouse or selected with the keyboard. |
| `DisabledStyle` | `cell.Style` | DisabledStyle is the style of disabled menu items. |
| `BorderStyle` | `cell.Style` | BorderStyle sets the style of the menu border. |
| `BorderSymbols` | `BorderSymbols` | BorderSymbols sets the menu border symbols. |

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

RadioButton is a radio button for choosing one option from a group.

Type to pass to `RenderWidget`: `RadioButton`

| Field | Type | Description |
| --- | --- | --- |
| `ID` | `string` |  |
| `Selected` | `*string` |  |
| `Value` | `string` |  |
| `Label` | `string` |  |
| `Style` | `cell.Style` |  |
| `FocusedStyle` | `cell.Style` | FocusedStyle defaults to reverse bold, or the theme focus style when both styles are unset. |

## Scrollbar

Scrollbar renders a track and a proportional thumb for a scrollable region.

Type to pass to `RenderWidget`: `Scrollbar`

| Field | Type | Description |
| --- | --- | --- |
| `Orientation` | `ScrollbarOrientation` | Orientation selects the axis. |
| `ContentLength` | `int` | ContentLength is the total number of scrollable units. |
| `ViewportLength` | `int` | ViewportLength is the number of units visible at once. |
| `Offset` | `int` | Offset is the index of the first visible unit. |
| `TrackStyle` | `cell.Style` | TrackStyle and ThumbStyle override the theme defaults when set. |
| `ThumbStyle` | `cell.Style` |  |
| `TrackSymbol` | `rune` | TrackSymbol and ThumbSymbol override the default glyphs when non-zero. |
| `ThumbSymbol` | `rune` |  |
| `AlwaysVisible` | `bool` | AlwaysVisible draws the track even when the content fits the viewport. |
| `OnScroll` | `func(...)` | OnScroll, when set, makes the bar interactive: clicking the track jumps to that position and dragging the thumb scrolls continuously. |

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
| `ID` | `string` | ID is the widget's focus ID. |
| `Data` | `[]float64` | Data is the series of numbers to draw. |
| `Style` | `cell.Style` | Style is the default cell style. |
| `FocusedStyle` | `cell.Style` | FocusedStyle is the style applied when focused. |
| `Color` | `cell.Color` | Color sets the colour of the bars. |

## Spinner

Spinner renders an animated activity indicator with an optional label.

Type to pass to `RenderWidget`: `Spinner`

| Field | Type | Description |
| --- | --- | --- |
| `Set` | `SpinnerSet` | Set selects the animation. |
| `Since` | `time.Time` | Since is the time the activity started. |
| `Frame` | `int` | Frame is an explicit frame index, used when Since is zero. |
| `Label` | `string` | Label is optional text drawn after the spinner. |
| `Style` | `cell.Style` | Style applies to the spinner glyph, LabelStyle to the label. |
| `LabelStyle` | `cell.Style` |  |
| `Gap` | `uint16` | Gap is the number of spaces between the glyph and the label. |
| `GapZero` | `bool` | GapZero places the label immediately after the glyph. |

## SplitPane

SplitPane shows two widgets side by side or stacked, with a divider the mouse can drag between them.

Type to pass to `RenderWidget`: `SplitPane`

| Field | Type | Description |
| --- | --- | --- |
| `ID` | `string` |  |
| `Direction` | `SplitDirection` |  |
| `First` | `Widget` |  |
| `Second` | `Widget` |  |
| `State` | `*SplitState` |  |
| `MinFirst` | `uint16` | MinFirst and MinSecond keep each pane at least this many cells wide (or tall) while the divider moves, as far as the space allows. |
| `MinSecond` | `uint16` | MinFirst and MinSecond keep each pane at least this many cells wide (or tall) while the divider moves, as far as the space allows. |
| `DividerStyle` | `cell.Style` |  |
| `FocusedDividerStyle` | `cell.Style` |  |
| `DividerRune` | `rune` | DividerRune replaces │ (side by side) or ─ (stacked). |

## StatusBar

StatusBar is a one-row bar with items on the left, centre and right, the usual last line of a full-screen application.

Type to pass to `RenderWidget`: `StatusBar`

| Field | Type | Description |
| --- | --- | --- |
| `ID` | `string` | ID names the bar in the semantic tree. |
| `Left` | `[]StatusItem` |  |
| `Center` | `[]StatusItem` |  |
| `Right` | `[]StatusItem` |  |
| `Style` | `cell.Style` | Style fills the whole row; KeyStyle is merged over it for item keys. |
| `KeyStyle` | `cell.Style` |  |
| `Separator` | `string` | Separator goes between items in a group. |

## Table

Table is an interactive table with flexible columns, vertical scrolling and cell spanning.

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

## Tabs

Tabs renders a single-line horizontal tab bar and reports clicks.

Type to pass to `RenderWidget`: `Tabs`

| Field | Type | Description |
| --- | --- | --- |
| `ID` | `string` | ID registers the tab bar with the focus manager when set. |
| `Titles` | `[]string` | Titles are the tab labels, rendered left to right. |
| `Selected` | `int` | Selected is the index of the active tab. |
| `Style` | `cell.Style` | Style applies to inactive tabs and the bar background. |
| `SelectedStyle` | `cell.Style` | SelectedStyle applies to the active tab. |
| `DividerStyle` | `cell.Style` | DividerStyle applies to the separator between tabs. |
| `Divider` | `*string` | Divider separates adjacent titles. |
| `Padding` | `uint16` | Padding is the number of spaces added on each side of a title. |
| `PaddingZero` | `bool` | PaddingZero renders titles with no surrounding spaces. |
| `OnSelect` | `func(...)` | OnSelect is invoked with the tab index when a tab is clicked. |
| `State` | `*TabsState` | State, when set, holds a buffer the tab bar reuses to expose each visible tab in the semantic tree, so tests and agents can find a tab by its title and click it. |

## Text

Text renders multiple rich-text lines with optional cell-aware wrapping.

Type to pass to `RenderWidget`: `Text`

| Field | Type | Description |
| --- | --- | --- |
| `ID` | `string` | ID focuses the text and names it in the semantic tree, where its value is the text of its spans, line by line; building that costs one allocation per frame. |
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
| `Placeholder` | `string` | Placeholder is shown, dimmed, while the text is empty. |
| `Focused` | `bool` | Focused draws the area as focused whatever the focus manager says, for applications that track focus themselves. |

## TextInput

TextInput is a single-line text input box.

Type to pass to `RenderWidget`: `TextInput`

| Field | Type | Description |
| --- | --- | --- |
| `ID` | `string` |  |
| `State` | `*TextInputState` |  |
| `Placeholder` | `string` |  |
| `Style` | `cell.Style` |  |
| `PlaceholderStyle` | `cell.Style` |  |
| `FocusedStyle` | `cell.Style` |  |
| `SelectionStart` | `int` |  |
| `SelectionEnd` | `int` |  |
| `SelectionStyle` | `cell.Style` |  |
| `Focused` | `bool` |  |
| `Secret` | `bool` | Secret masks the input for passwords and tokens. |
| `MaskRune` | `rune` | MaskRune is the glyph drawn in place of each character when Secret is set. |

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
| `State` | `*Viewer3DState` | State, when set, lets the mouse and keyboard orbit the model; see Viewer3DState. |
| `Scale` | `float64` | Scale is the zoom/scale multiplier (default: 1.0). |
| `Shading` | `string` | Shading is one of ShadingTexture, ShadingWireframe, ShadingFlat, ShadingLambert or ShadingGouraud. |
| `Wireframe` | `bool` | Wireframe overlays edges on top of shaded faces. |
| `WireframeStyle` | `cell.Style` | WireframeStyle is the cell style for wireframe lines. |
| `FocusedStyle` | `cell.Style` | FocusedStyle is applied to wireframe/highlight when the viewer is focused. |
| `Light` | `graphics.Light` | Light is the directional light source for Lambertian and Gouraud shading. |
| `FaceColors` | `[]cell.Color` | FaceColors is an optional palette for coloring distinct faces in solid mode. |
| `Marker` | `Marker` | Marker picks the characters the model is drawn with (Braille by default). |
| `Pixels` | `bool` | Pixels renders the model as a picture, sent with the terminal's image protocol (kitty, iTerm2, Sixel), instead of as Braille dots: full resolution and colour per pixel. |

## Viewport

Viewport renders a child widget that may be larger than the available area and shows a scrollable window onto it.

Type to pass to `RenderWidget`: `Viewport`

| Field | Type | Description |
| --- | --- | --- |
| `ID` | `string` | ID registers the viewport with the focus manager when set. |
| `Child` | `Widget` | Child is the widget rendered inside the viewport. |
| `State` | `*ViewportState` | State carries the scroll offsets. |
| `ContentHeight` | `int` | ContentHeight overrides the measured child height. |
| `ContentWidth` | `int` | ContentWidth overrides the measured child width. |
| `Style` | `cell.Style` | Style is merged into the inherited context style for the whole area. |
| `Scrollbar` | `bool` | Scrollbar draws a vertical scrollbar along the right edge, reserving one column from the child's width. |
| `ScrollbarTrackStyle` | `cell.Style` | ScrollbarTrackStyle and ScrollbarThumbStyle override theme defaults. |
| `ScrollbarThumbStyle` | `cell.Style` |  |
| `MouseWheelStep` | `int` | MouseWheelStep is the number of rows scrolled per wheel notch. |
| `DisableMouse` | `bool` | DisableMouse turns off wheel scrolling and scrollbar interaction. |

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

