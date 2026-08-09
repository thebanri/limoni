package main

import (
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/layout"
	"github.com/thebanri/limoni/widgets"
)

func drawSettings(t *terminal.Terminal, f *terminal.Frame, state *AppState, demoTheme widgets.Theme, mainColor, accentColor cell.Color, bodyArea cell.Rect) {
	innerArea := cell.Rect{
		X:      bodyArea.X + 1,
		Y:      bodyArea.Y + 1,
		Width:  bodyArea.Width - 2,
		Height: bodyArea.Height - 2,
	}
	formLay := layout.NewFlexLayout(
		layout.Vertical,
		1,               // Elemanlar arasında 1 satır boşluk bırak
		layout.Fixed(1), // Kılavuz / Açıklama satırı
		layout.Fixed(3), // Kullanıcı adı kutusu (Bordered Block height 3)
		layout.Fixed(1), // Checkbox (Mouse modu)
		layout.Fixed(6), // Tema grubu kutusu (Bordered Block height 6)
		layout.Fixed(3), // Bildirim modu açılır kutusu (Bordered Block height 3)
	)
	formChunks := formLay.Split(innerArea)

	// 1. Dış Çerçeveyi Çiz
	settingsBlock := widgets.Block{
		Title:          " LİMONİ SİSTEM AYARLARI ",
		TitleAlignment: widgets.AlignLeft,
		Borders:        widgets.BorderAll,
		BorderSymbols:  widgets.SymbolsRounded,
		BorderStyle:    cell.Style{Fg: mainColor},
	}
	f.RenderWidget(settingsBlock, bodyArea)

	// 2. İç form elemanlarını çiz
	f.RenderWidget(label{text: "Tab / Shift+Tab veya Ok tuşları ile odaklanıp Space ile seçin.", style: cell.Style{Fg: cell.NewColorRGB(160, 160, 160), Modifier: cell.ModifierItalic}}, formChunks[0])

	// Kullanıcı adı kutusu
	inputBorderCol := cell.NewColorRGB(100, 100, 100)
	if t.FocusManager().Focused() == "username_input" {
		inputBorderCol = accentColor
	}

	usernameInput := widgets.TextInput{
		ID:           "username_input",
		State:        state.UsernameInputState,
		Placeholder:  "Kullanıcı adınızı girin...",
		Style:        cell.Style{Fg: cell.NewColorRGB(255, 255, 255)},
		FocusedStyle: cell.Style{Fg: cell.NewColorRGB(255, 255, 255)},
	}
	usernameBlock := widgets.Block{
		Title:          " KULLANICI ADI ",
		TitleAlignment: widgets.AlignLeft,
		Borders:        widgets.BorderAll,
		BorderSymbols:  widgets.SymbolsRounded,
		BorderStyle:    cell.Style{Fg: inputBorderCol},
		Child:          usernameInput,
	}
	f.RenderWidget(usernameBlock, formChunks[1])

	if t.FocusManager().Focused() == "username_input" {
		widgets.DrawFocusRing(f.Buffer, formChunks[1], cell.Style{Fg: accentColor})
	}

	// Mouse Modu Checkbox
	mouseModeCb := widgets.Checkbox{
		ID:           "mouse_mode_cb",
		Checked:      &state.MouseModeChecked,
		Label:        "Mouse Modunu Aktif Et (SGR)",
		FocusedStyle: cell.Style{Fg: accentColor, Modifier: cell.ModifierBold},
	}
	f.RenderWidget(mouseModeCb, formChunks[2])

	// Tema Seçim Paneli (Bordered Block + Radio buttons)
	themeBorderCol := cell.NewColorRGB(100, 100, 100)
	focused := t.FocusManager().Focused()
	if focused == "theme_dark_rb" || focused == "theme_light_rb" || focused == "theme_colored_rb" || focused == "theme_contrast_rb" {
		themeBorderCol = accentColor
	}

	themeInnerArea := cell.Rect{
		X:      formChunks[3].X + 2,
		Y:      formChunks[3].Y + 1,
		Width:  formChunks[3].Width - 4,
		Height: formChunks[3].Height - 2,
	}
	themeLay := layout.NewFlexLayout(
		layout.Vertical,
		0,
		layout.Fixed(1),
		layout.Fixed(1),
		layout.Fixed(1),
		layout.Fixed(1),
	)
	themeChunks := themeLay.Split(themeInnerArea)

	darkRb := widgets.RadioButton{
		ID:           "theme_dark_rb",
		Selected:     &state.ThemeSelected,
		Value:        "Koyu",
		Label:        "Koyu Tema (Cyan/Yeşil)",
		FocusedStyle: cell.Style{Fg: accentColor, Modifier: cell.ModifierBold},
	}
	lightRb := widgets.RadioButton{
		ID:           "theme_light_rb",
		Selected:     &state.ThemeSelected,
		Value:        "Açık",
		Label:        "Açık Tema (Mavi/Gri)",
		FocusedStyle: cell.Style{Fg: accentColor, Modifier: cell.ModifierBold},
	}
	coloredRb := widgets.RadioButton{
		ID:           "theme_colored_rb",
		Selected:     &state.ThemeSelected,
		Value:        "Renkli",
		Label:        "Renkli Tema (Turuncu/Mor)",
		FocusedStyle: cell.Style{Fg: accentColor, Modifier: cell.ModifierBold},
	}
	contrastRb := widgets.RadioButton{
		ID: "theme_contrast_rb", Selected: &state.ThemeSelected, Value: "Yüksek Kontrast",
		Label: "Yüksek Kontrast Tema", FocusedStyle: cell.Style{Fg: accentColor, Modifier: cell.ModifierBold},
	}

	themeBlock := widgets.Block{
		Title:          " ARAYÜZ RENK TEMASI ",
		TitleAlignment: widgets.AlignLeft,
		Borders:        widgets.BorderAll,
		BorderSymbols:  widgets.SymbolsRounded,
		BorderStyle:    cell.Style{Fg: themeBorderCol},
	}
	f.RenderWidget(themeBlock, formChunks[3])

	if t.FocusManager().Focused() == "theme_dark_rb" || t.FocusManager().Focused() == "theme_light_rb" || t.FocusManager().Focused() == "theme_colored_rb" || t.FocusManager().Focused() == "theme_contrast_rb" {
		widgets.DrawFocusRing(f.Buffer, formChunks[3], cell.Style{Fg: accentColor})
	}

	f.RenderWidget(darkRb, themeChunks[0])
	f.RenderWidget(lightRb, themeChunks[1])
	f.RenderWidget(coloredRb, themeChunks[2])
	f.RenderWidget(contrastRb, themeChunks[3])

	// 3. Bildirim Modu Açılır Menü (Popup) Çizimi
	notifBorderCol := cell.NewColorRGB(100, 100, 100)
	if t.FocusManager().Focused() == "notif_popup" {
		notifBorderCol = accentColor
	}

	popupArea := cell.Rect{
		X:      formChunks[4].X + 2,
		Y:      formChunks[4].Y + 1,
		Width:  formChunks[4].Width - 4,
		Height: 1,
	}

	notificationModePopup := widgets.Popup{
		ID:    "notif_popup",
		Label: state.NotificationMode,
		Items: []widgets.PopupItem{
			{Text: "Sessiz Mod", Handler: func() { state.NotificationMode = "Sessiz Mod" }},
			{Text: "Normal Mod", Handler: func() { state.NotificationMode = "Normal Mod" }},
			{Text: "Tümünü Bildir", Handler: func() { state.NotificationMode = "Tümünü Bildir" }},
			{Text: "Devre Dışı", Disabled: true, Handler: func() {}},
		},
		State:         state.NotifPopupState,
		Style:         cell.Style{Fg: cell.NewColorRGB(200, 200, 200), Bg: cell.NewColorRGB(50, 50, 50)},
		ItemStyle:     cell.Style{Fg: cell.NewColorRGB(220, 220, 220)},
		SelectedStyle: cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Bg: accentColor, Modifier: cell.ModifierBold},
		DisabledStyle: cell.Style{Fg: cell.NewColorRGB(100, 100, 100)},
		BorderStyle:   cell.Style{Fg: notifBorderCol},
	}

	notifBlock := widgets.Block{
		Title:          " BİLDİRİM MODU (AÇILIR MENÜ) ",
		TitleAlignment: widgets.AlignLeft,
		Borders:        widgets.BorderAll,
		BorderSymbols:  widgets.SymbolsRounded,
		BorderStyle:    cell.Style{Fg: notifBorderCol},
	}
	f.RenderWidget(notifBlock, formChunks[4])

	if t.FocusManager().Focused() == "notif_popup" {
		widgets.DrawFocusRing(f.Buffer, formChunks[4], cell.Style{Fg: accentColor})
	}

	f.RenderWidget(notificationModePopup, popupArea)
}
