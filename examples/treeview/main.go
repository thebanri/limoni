package main

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/layout"
	"github.com/thebanri/limoni/widgets"
)

type FileItem struct {
	Name    string
	Path    string
	IsDir   bool
	Size    int64
	ModTime time.Time
	Mode    os.FileMode
	Icon    string
}

type PinnedBookmark struct {
	Name string
	Path string
	Icon string
	Key  rune
}

type SuperfileState struct {
	CurrentDir    string
	Items         []FileItem
	SelectedIndex int
	ScrollOffset  int
	ShowHidden    bool
	Bookmarks     []PinnedBookmark
	ActivePanel   int // 0: Sidebar, 1: FileList, 2: Preview
	PreviewLines  []string
	PreviewTotal  int
	PreviewErr    string
	IsImage       bool
	ImageObj      image.Image
	ImageFormat   string
	ImageWidth    int
	ImageHeight   int
}

func getFileIcon(name string, isDir bool) string {
	if isDir {
		if strings.HasPrefix(name, ".") {
			return "📁"
		}
		return "📂"
	}
	lower := strings.ToLower(name)
	ext := filepath.Ext(lower)

	switch ext {
	case ".go":
		return "🐹"
	case ".rs":
		return "🦀"
	case ".py":
		return "🐍"
	case ".js", ".ts", ".jsx", ".tsx":
		return "📜"
	case ".md", ".txt", ".rst":
		return "📄"
	case ".json", ".yaml", ".yml", ".toml", ".ini", ".conf":
		return "⚙️"
	case ".sh", ".bash", ".zsh":
		return "⚡"
	case ".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp":
		return "🖼️"
	case ".zip", ".tar", ".gz", ".bz2", ".xz", ".7z":
		return "📦"
	case ".mod", ".sum":
		return "📦"
	case ".html", ".css":
		return "🌐"
	default:
		if strings.HasPrefix(name, ".") {
			return "🔒"
		}
		return "📄"
	}
}

func formatSize(bytes int64) string {
	const unit = 1024.0
	if bytes < int64(unit) {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / int64(unit); n >= int64(unit); n /= int64(unit) {
		div *= int64(unit)
		exp++
	}
	units := []string{"KB", "MB", "GB", "TB"}
	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), units[exp])
}

func (s *SuperfileState) LoadDirectory(dir string) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		return err
	}

	s.CurrentDir = abs
	var items []FileItem

	for _, e := range entries {
		name := e.Name()
		if !s.ShowHidden && strings.HasPrefix(name, ".") {
			continue
		}
		info, err := e.Info()
		var size int64
		var mod time.Time
		var mode os.FileMode
		if err == nil {
			size = info.Size()
			mod = info.ModTime()
			mode = info.Mode()
		}

		fullPath := filepath.Join(abs, name)
		isDir := e.IsDir()
		items = append(items, FileItem{
			Name:    name,
			Path:    fullPath,
			IsDir:   isDir,
			Size:    size,
			ModTime: mod,
			Mode:    mode,
			Icon:    getFileIcon(name, isDir),
		})
	}

	// Sort directories first, then alphabetical
	sort.Slice(items, func(i, j int) bool {
		if items[i].IsDir != items[j].IsDir {
			return items[i].IsDir
		}
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})

	s.Items = items
	if s.SelectedIndex >= len(items) {
		s.SelectedIndex = len(items) - 1
	}
	if s.SelectedIndex < 0 && len(items) > 0 {
		s.SelectedIndex = 0
	}
	s.UpdatePreview()
	return nil
}

func isImageFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".bmp", ".webp":
		return true
	}
	return false
}

func (s *SuperfileState) UpdatePreview() {
	s.PreviewLines = nil
	s.PreviewErr = ""
	s.PreviewTotal = 0
	s.IsImage = false
	s.ImageObj = nil
	s.ImageFormat = ""
	s.ImageWidth = 0
	s.ImageHeight = 0

	if len(s.Items) == 0 || s.SelectedIndex < 0 || s.SelectedIndex >= len(s.Items) {
		return
	}

	sel := s.Items[s.SelectedIndex]
	if sel.IsDir {
		subEntries, err := os.ReadDir(sel.Path)
		if err != nil {
			s.PreviewErr = fmt.Sprintf("Access Denied: %v", err)
			return
		}
		dirs := 0
		files := 0
		var subList []string
		for _, se := range subEntries {
			if se.IsDir() {
				dirs++
				if len(subList) < 15 {
					subList = append(subList, fmt.Sprintf(" 📁 %s/", se.Name()))
				}
			} else {
				files++
				if len(subList) < 15 {
					subList = append(subList, fmt.Sprintf(" %s %s", getFileIcon(se.Name(), false), se.Name()))
				}
			}
		}
		s.PreviewTotal = len(subEntries)
		lines := []string{
			fmt.Sprintf("Directory: %s", sel.Name),
			fmt.Sprintf("Contents : %d items (%d folders, %d files)", len(subEntries), dirs, files),
			"────────────────────────────────────────",
		}
		lines = append(lines, subList...)
		if len(subEntries) > 15 {
			lines = append(lines, fmt.Sprintf(" … and %d more items", len(subEntries)-15))
		}
		s.PreviewLines = lines
		return
	}

	// 1. IMAGE FILE PREVIEW
	if isImageFile(sel.Path) {
		f, err := os.Open(sel.Path)
		if err == nil {
			defer f.Close()
			img, fmtName, err := image.Decode(f)
			if err == nil {
				s.IsImage = true
				s.ImageObj = img
				s.ImageFormat = strings.ToUpper(fmtName)
				b := img.Bounds()
				s.ImageWidth = b.Dx()
				s.ImageHeight = b.Dy()
				s.PreviewLines = []string{
					fmt.Sprintf("Format    : %s Image", s.ImageFormat),
					fmt.Sprintf("Dimensions: %d × %d px", s.ImageWidth, s.ImageHeight),
					fmt.Sprintf("File Size : %s", formatSize(sel.Size)),
				}
				return
			}
		}
	}

	// 2. TEXT & CODE FILE PREVIEW
	f, err := os.Open(sel.Path)
	if err != nil {
		s.PreviewErr = fmt.Sprintf("Cannot open file: %v", err)
		return
	}
	defer f.Close()

	buf := make([]byte, 16384)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		s.PreviewErr = fmt.Sprintf("Read error: %v", err)
		return
	}

	// Check if file is binary (contains null bytes)
	isBinary := false
	for i := 0; i < n; i++ {
		if buf[i] == 0 {
			isBinary = true
			break
		}
	}

	if isBinary {
		s.PreviewLines = []string{
			fmt.Sprintf("Binary File: %s", sel.Name),
			fmt.Sprintf("File Size  : %s", formatSize(sel.Size)),
			"────────────────────────────────────────",
			" [Binary Data / Non-Text Executable]",
		}
		return
	}

	content := string(buf[:n])
	rawLines := strings.Split(content, "\n")
	if len(rawLines) > 60 {
		s.PreviewTotal = len(rawLines)
		rawLines = rawLines[:60]
	} else {
		s.PreviewTotal = len(rawLines)
	}

	s.PreviewLines = rawLines
}

func main() {
	b := backend.NewBackend(os.Stdin, os.Stdout)
	if err := b.Setup(); err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing backend: %v\n", err)
		os.Exit(1)
	}
	defer b.Close()

	t, err := terminal.New(b)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing terminal: %v\n", err)
		os.Exit(1)
	}

	b.StartEventLoop()

	cwd, _ := os.Getwd()
	homeDir, _ := os.UserHomeDir()

	state := &SuperfileState{
		CurrentDir:    cwd,
		SelectedIndex: 0,
		ShowHidden:    false,
		ActivePanel:   1, // File List
		Bookmarks: []PinnedBookmark{
			{Name: "Project Root", Path: cwd, Icon: "📂", Key: '1'},
			{Name: "core/", Path: filepath.Join(cwd, "core"), Icon: "📁", Key: '2'},
			{Name: "widgets/", Path: filepath.Join(cwd, "widgets"), Icon: "📁", Key: '3'},
			{Name: "layout/", Path: filepath.Join(cwd, "layout"), Icon: "📁", Key: '4'},
			{Name: "examples/", Path: filepath.Join(cwd, "examples"), Icon: "📁", Key: '5'},
			{Name: "docs/", Path: filepath.Join(cwd, "docs"), Icon: "📁", Key: '6'},
			{Name: "Home (~)", Path: homeDir, Icon: "🏠", Key: '7'},
		},
	}
	state.LoadDirectory(cwd)

	var sideRect, listRect cell.Rect

	draw := func() {
		t.Draw(func(f *terminal.Frame) {
			area := f.Buffer.Area
			accent := cell.NewColorRGB(0, 220, 255)
			bgCard := cell.NewColorRGB(18, 22, 32)
			borderCard := cell.NewColorRGB(60, 70, 90)

			chunks := layout.VBox(area, layout.Fixed(3), layout.Fill(), layout.Fixed(1))

			// 1. TOP HEADER & BREADCRUMBS
			itemCount := len(state.Items)
			hiddenStatus := "Off"
			if state.ShowHidden {
				hiddenStatus = "ON"
			}
			f.RenderWidget(widgets.Block{
				Title:         " 📂 SUPERFILE TERMINAL EXPLORER ",
				Borders:       widgets.BorderAll,
				BorderSymbols: widgets.SymbolsRounded,
				BorderStyle:   cell.Style{Fg: accent, Modifier: cell.ModifierBold},
				Style:         cell.Style{Bg: bgCard},
				Child: &widgets.Paragraph{
					Text: fmt.Sprintf(" Path: %s  │  Items: %d  │  Hidden: %s", state.CurrentDir, itemCount, hiddenStatus),
					Style: cell.Style{
						Fg:       cell.NewColorRGB(220, 230, 245),
						Modifier: cell.ModifierBold,
					},
				},
			}, chunks[0])

			// 2. MAIN 3-PANEL BODY: Sidebar (20%) + File List (45%) + Code Preview (35%)
			colW0 := chunks[1].Width * 20 / 100
			colW1 := chunks[1].Width * 45 / 100
			colW2 := chunks[1].Width - colW0 - colW1
			if colW0 < 16 {
				colW0 = 16
			}
			if colW2 < 20 {
				colW2 = 20
			}
			if colW0+colW1+colW2 > chunks[1].Width {
				colW1 = chunks[1].Width - colW0 - colW2
			}

			cols := []cell.Rect{
				cell.NewRect(chunks[1].X, chunks[1].Y, colW0, chunks[1].Height),
				cell.NewRect(chunks[1].X+colW0, chunks[1].Y, colW1, chunks[1].Height),
				cell.NewRect(chunks[1].X+colW0+colW1, chunks[1].Y, colW2, chunks[1].Height),
			}
			sideRect = cols[0]
			listRect = cols[1]

			// PANEL 1: SIDEBAR & PINNED BOOKMARKS
			f.RenderWidget(widgets.Block{
				Title:         " 📌 PINNED ",
				Borders:       widgets.BorderAll,
				BorderSymbols: widgets.SymbolsRounded,
				BorderStyle:   cell.Style{Fg: borderCard},
				Style:         cell.Style{Bg: bgCard},
			}, cols[0])

			// Fill Sidebar background
			for y := cols[0].Y + 1; y < cols[0].Y+cols[0].Height-1; y++ {
				for x := cols[0].X + 1; x < cols[0].X+cols[0].Width-1; x++ {
					if c := f.Buffer.Get(x, y); c != nil {
						c.Content = ' '
						c.Style = cell.Style{Bg: bgCard}
					}
				}
			}

			sideInnerY := cols[0].Y + 1
			for _, bm := range state.Bookmarks {
				if sideInnerY >= cols[0].Y+cols[0].Height-1 {
					break
				}
				bmStyle := cell.Style{Fg: cell.NewColorRGB(170, 180, 200), Bg: bgCard}
				if state.CurrentDir == bm.Path {
					bmStyle = cell.Style{Fg: cell.NewColorRGB(0, 255, 180), Bg: bgCard, Modifier: cell.ModifierBold}
				}
				maxBmW := int(cols[0].Width) - 4
				bmText := fmt.Sprintf("[%c] %s %s", bm.Key, bm.Icon, bm.Name)
				if len([]rune(bmText)) > maxBmW && maxBmW > 0 {
					bmText = string([]rune(bmText)[:maxBmW])
				}
				f.Buffer.SetString(cols[0].X+2, sideInnerY, bmText, bmStyle)
				sideInnerY++
			}

			// PANEL 2: MAIN DIRECTORY FILE LIST
			fileListBorder := borderCard
			if state.ActivePanel == 1 {
				fileListBorder = cell.NewColorRGB(0, 220, 255)
			}
			f.RenderWidget(widgets.Block{
				Title:         fmt.Sprintf(" 📁 CONTENTS (%d) ", len(state.Items)),
				Borders:       widgets.BorderAll,
				BorderSymbols: widgets.SymbolsRounded,
				BorderStyle:   cell.Style{Fg: fileListBorder},
				Style:         cell.Style{Bg: bgCard},
			}, cols[1])

			listH := int(cols[1].Height) - 2
			if listH < 1 {
				listH = 1
			}

			// Scroll adjustment
			if state.SelectedIndex < state.ScrollOffset {
				state.ScrollOffset = state.SelectedIndex
			} else if state.SelectedIndex >= state.ScrollOffset+listH {
				state.ScrollOffset = state.SelectedIndex - listH + 1
			}
			if state.ScrollOffset < 0 {
				state.ScrollOffset = 0
			}

			for row := 0; row < listH; row++ {
				idx := state.ScrollOffset + row
				rowY := cols[1].Y + 1 + uint16(row)

				if idx >= len(state.Items) {
					for x := cols[1].X + 1; x < cols[1].X+cols[1].Width-1; x++ {
						if c := f.Buffer.Get(x, rowY); c != nil {
							c.Content = ' '
							c.Style = cell.Style{Bg: bgCard}
						}
					}
					continue
				}

				item := state.Items[idx]
				isSel := idx == state.SelectedIndex

				rowBg := bgCard
				nameStyle := cell.Style{Fg: cell.NewColorRGB(220, 225, 240), Bg: rowBg}
				sizeStyle := cell.Style{Fg: cell.NewColorRGB(120, 135, 155), Bg: rowBg}
				prefix := "  "

				if isSel {
					rowBg = cell.NewColorRGB(30, 45, 65)
					nameStyle = cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Bg: rowBg, Modifier: cell.ModifierBold}
					sizeStyle = cell.Style{Fg: cell.NewColorRGB(0, 255, 200), Bg: rowBg, Modifier: cell.ModifierBold}
					prefix = "❯ "
				}

				if item.IsDir {
					nameStyle.Fg = cell.NewColorRGB(0, 200, 255)
					if isSel {
						nameStyle.Fg = cell.NewColorRGB(100, 240, 255)
					}
				}

				// Fill Row background
				for x := cols[1].X + 1; x < cols[1].X+cols[1].Width-1; x++ {
					f.Buffer.SetCell(x, rowY, cell.Cell{Content: ' ', Style: cell.Style{Bg: rowBg}})
				}

				// Draw Item Icon & Name
				iconPrefix := fmt.Sprintf("%s%s %s", prefix, item.Icon, item.Name)
				if item.IsDir {
					iconPrefix += "/"
				}
				maxNameW := int(cols[1].Width) - 14
				if maxNameW < 8 {
					maxNameW = 8
				}
				if len([]rune(iconPrefix)) > maxNameW {
					runes := []rune(iconPrefix)
					iconPrefix = string(runes[:maxNameW-1]) + "…"
				}
				f.Buffer.SetString(cols[1].X+2, rowY, iconPrefix, nameStyle)

				// Draw Size on Right
				sizeStr := formatSize(item.Size)
				if item.IsDir {
					sizeStr = "[DIR]"
				}
				sizeX := cols[1].X + cols[1].Width - uint16(len(sizeStr)) - 3
				f.Buffer.SetString(sizeX, rowY, sizeStr, sizeStyle)
			}

			// PANEL 3: CODE & FILE PREVIEW / METADATA
			var selItem *FileItem
			if len(state.Items) > 0 && state.SelectedIndex >= 0 && state.SelectedIndex < len(state.Items) {
				selItem = &state.Items[state.SelectedIndex]
			}

			previewTitle := " PREVIEW "
			if selItem != nil {
				cleanName := selItem.Name
				maxTitleW := int(cols[2].Width) - 16
				if len([]rune(cleanName)) > maxTitleW && maxTitleW > 0 {
					cleanName = string([]rune(cleanName)[:maxTitleW]) + "…"
				}
				previewTitle = fmt.Sprintf(" PREVIEW: %s ", cleanName)
			}
			f.RenderWidget(widgets.Block{
				Title:         previewTitle,
				Borders:       widgets.BorderAll,
				BorderSymbols: widgets.SymbolsRounded,
				BorderStyle:   cell.Style{Fg: borderCard},
				Style:         cell.Style{Bg: bgCard},
			}, cols[2])

			// Fill Preview Panel background
			for y := cols[2].Y + 1; y < cols[2].Y+cols[2].Height-1; y++ {
				for x := cols[2].X + 1; x < cols[2].X+cols[2].Width-1; x++ {
					if c := f.Buffer.Get(x, y); c != nil {
						c.Content = ' '
						c.Style = cell.Style{Bg: bgCard}
					}
				}
			}

			pInnerY := cols[2].Y + 1
			if selItem != nil {
				// Metadata header
				metaStyle := cell.Style{Fg: cell.NewColorRGB(0, 255, 180), Bg: bgCard, Modifier: cell.ModifierBold}
				f.Buffer.SetString(cols[2].X+2, pInnerY, fmt.Sprintf("Type: %s │ Size: %s", selItem.Mode.String(), formatSize(selItem.Size)), metaStyle)
				f.Buffer.SetString(cols[2].X+2, pInnerY+1, fmt.Sprintf("Modified: %s", selItem.ModTime.Format("2006-01-02 15:04:05")), cell.Style{Fg: cell.NewColorRGB(140, 150, 165), Bg: bgCard})
				pInnerY += 3

				// Divider
				divW := int(cols[2].Width) - 2
				if divW > 0 {
					f.Buffer.SetString(cols[2].X+1, pInnerY-1, strings.Repeat("─", divW), cell.Style{Fg: borderCard, Bg: bgCard})
				}

				// 1. IMAGE THUMBNAIL PREVIEW
				if state.IsImage && state.ImageObj != nil {
					// Draw image info lines
					for i, line := range state.PreviewLines {
						f.Buffer.SetString(cols[2].X+2, pInnerY+uint16(i), line, cell.Style{Fg: cell.NewColorRGB(0, 220, 255), Bg: bgCard, Modifier: cell.ModifierBold})
					}

					img := state.ImageObj
					b := img.Bounds()
					imgW := b.Dx()
					imgH := b.Dy()

					availW := int(cols[2].Width) - 4
					availH := int(cols[2].Height) - int(pInnerY-cols[2].Y) - 5
					if availW > 2 && availH > 2 && imgW > 0 && imgH > 0 {
						scaleX := float64(availW) / float64(imgW)
						scaleY := float64(availH*2) / float64(imgH)
						scale := math.Min(scaleX, scaleY)
						if scale > 1.0 {
							scale = 1.0
						}

						renderW := int(math.Round(float64(imgW) * scale))
						renderH := int(math.Round(float64(imgH) * scale))
						if renderW < 1 {
							renderW = 1
						}
						if renderH < 2 {
							renderH = 2
						}
						if renderH%2 != 0 {
							renderH++
						}

						cellRows := renderH / 2
						startDrawY := pInnerY + uint16(len(state.PreviewLines)) + 1

						for cy := 0; cy < cellRows; cy++ {
							currCellY := startDrawY + uint16(cy)
							if currCellY >= cols[2].Y+cols[2].Height-1 {
								break
							}

							topSrcY := int(float64(cy*2) / scale)
							botSrcY := int(float64(cy*2+1) / scale)

							for cx := 0; cx < renderW; cx++ {
								currCellX := cols[2].X + 2 + uint16(cx)
								if currCellX >= cols[2].X+cols[2].Width-1 {
									break
								}

								srcX := int(float64(cx) / scale)
								if srcX >= imgW {
									srcX = imgW - 1
								}

								var topR, topG, topB, topA uint32
								if topSrcY < imgH {
									topR, topG, topB, topA = img.At(b.Min.X+srcX, b.Min.Y+topSrcY).RGBA()
								}
								var botR, botG, botB, botA uint32
								if botSrcY < imgH {
									botR, botG, botB, botA = img.At(b.Min.X+srcX, b.Min.Y+botSrcY).RGBA()
								}

								topCol := cell.NewColorRGB(uint8(topR>>8), uint8(topG>>8), uint8(topB>>8))
								botCol := cell.NewColorRGB(uint8(botR>>8), uint8(botG>>8), uint8(botB>>8))

								cellSymbol := '▀'
								cellStyle := cell.Style{Fg: topCol, Bg: botCol}

								if topA < 1000 && botA < 1000 {
									cellSymbol = ' '
									cellStyle = cell.Style{Bg: bgCard}
								} else if topA < 1000 {
									cellSymbol = '▄'
									cellStyle = cell.Style{Fg: botCol, Bg: bgCard}
								} else if botA < 1000 {
									cellSymbol = '▀'
									cellStyle = cell.Style{Fg: topCol, Bg: bgCard}
								}

								f.Buffer.SetCell(currCellX, currCellY, cell.Cell{
									Content: cellSymbol,
									Style:   cellStyle,
								})
							}
						}
					}
				} else {
					// 2. CODE & DIRECTORY LINES PREVIEW
					maxLines := int(cols[2].Height) - 5
				for i, rawLine := range state.PreviewLines {
					if i >= maxLines {
						break
					}
					currLineY := pInnerY + uint16(i)
					lineNumStyle := cell.Style{Fg: cell.NewColorRGB(90, 100, 120), Bg: bgCard}
					codeStyle := cell.Style{Fg: cell.NewColorRGB(215, 225, 240), Bg: bgCard}

					// Expand tabs and remove \r
					cleanLine := strings.ReplaceAll(rawLine, "\t", "    ")
					cleanLine = strings.ReplaceAll(cleanLine, "\r", "")

					if selItem.IsDir {
						maxCodeW := int(cols[2].Width) - 4
						if len([]rune(cleanLine)) > maxCodeW && maxCodeW > 0 {
							cleanLine = string([]rune(cleanLine)[:maxCodeW])
						}
						f.Buffer.SetString(cols[2].X+2, currLineY, cleanLine, codeStyle)
					} else {
						lineNumStr := fmt.Sprintf("%2d │ ", i+1)
						f.Buffer.SetString(cols[2].X+2, currLineY, lineNumStr, lineNumStyle)

						maxCodeW := int(cols[2].Width) - 8
						if len([]rune(cleanLine)) > maxCodeW && maxCodeW > 0 {
							cleanLine = string([]rune(cleanLine)[:maxCodeW])
						}

						// Basic syntax highlight tinting
						if strings.Contains(cleanLine, "func ") || strings.Contains(cleanLine, "package ") || strings.Contains(cleanLine, "import ") || strings.Contains(cleanLine, "type ") {
							codeStyle.Fg = cell.NewColorRGB(0, 220, 255)
						} else if strings.Contains(cleanLine, "//") || strings.Contains(cleanLine, "/*") {
							codeStyle.Fg = cell.NewColorRGB(100, 110, 130)
						} else if strings.Contains(cleanLine, "\"") {
							codeStyle.Fg = cell.NewColorRGB(46, 204, 113)
						}

						f.Buffer.SetString(cols[2].X+7, currLineY, cleanLine, codeStyle)
					}
				}
			}
		}

			// 3. BOTTOM FOOTER SHORTCUTS
			footerText := " [j/k/▲/▼] Move  [Enter/l] Open  [Backspace/h] Parent  [1-7] Pinned  [.] Hidden  [q] Quit"
			f.RenderWidget(widgets.Block{
				Borders: widgets.BorderNone,
				Child: &widgets.Paragraph{
					Text: footerText,
					Style: cell.Style{
						Fg: cell.NewColorRGB(140, 150, 165),
					},
				},
			}, chunks[2])
		})
	}

	draw()

	renderTicker := time.NewTicker(40 * time.Millisecond)
	defer renderTicker.Stop()

	for {
		select {
		case <-renderTicker.C:
			draw()
		case ev, ok := <-b.Events():
			if !ok {
				return
			}
			switch ev.Type {
			case backend.EventKey:
				if ev.Key.Type == backend.KeyRune {
					switch ev.Key.Ch {
					case 'q', 'Q':
						return
					case 'j', 'J':
						if state.SelectedIndex < len(state.Items)-1 {
							state.SelectedIndex++
							state.UpdatePreview()
						}
					case 'k', 'K':
						if state.SelectedIndex > 0 {
							state.SelectedIndex--
							state.UpdatePreview()
						}
					case 'l', 'L':
						if len(state.Items) > 0 && state.SelectedIndex >= 0 && state.SelectedIndex < len(state.Items) {
							sel := state.Items[state.SelectedIndex]
							if sel.IsDir {
								state.LoadDirectory(sel.Path)
							}
						}
					case 'h', 'H':
						parent := filepath.Dir(state.CurrentDir)
						if parent != state.CurrentDir {
							state.LoadDirectory(parent)
						}
					case '.':
						state.ShowHidden = !state.ShowHidden
						state.LoadDirectory(state.CurrentDir)
					case '1', '2', '3', '4', '5', '6', '7':
						idx := int(ev.Key.Ch - '1')
						if idx >= 0 && idx < len(state.Bookmarks) {
							state.LoadDirectory(state.Bookmarks[idx].Path)
						}
					}
					draw()
				}
				switch ev.Key.Type {
				case backend.KeyEsc:
					return
				case backend.KeyArrowDown:
					if state.SelectedIndex < len(state.Items)-1 {
						state.SelectedIndex++
						state.UpdatePreview()
						draw()
					}
				case backend.KeyArrowUp:
					if state.SelectedIndex > 0 {
						state.SelectedIndex--
						state.UpdatePreview()
						draw()
					}
				case backend.KeyEnter:
					if len(state.Items) > 0 && state.SelectedIndex >= 0 && state.SelectedIndex < len(state.Items) {
						sel := state.Items[state.SelectedIndex]
						if sel.IsDir {
							state.LoadDirectory(sel.Path)
							draw()
						}
					}
				case backend.KeyBackspace:
					parent := filepath.Dir(state.CurrentDir)
					if parent != state.CurrentDir {
						state.LoadDirectory(parent)
						draw()
					}
				case backend.KeyPageDown:
					state.SelectedIndex += 10
					if state.SelectedIndex >= len(state.Items) {
						state.SelectedIndex = len(state.Items) - 1
					}
					state.UpdatePreview()
					draw()
				case backend.KeyPageUp:
					state.SelectedIndex -= 10
					if state.SelectedIndex < 0 {
						state.SelectedIndex = 0
					}
					state.UpdatePreview()
					draw()
				case backend.KeyHome:
					state.SelectedIndex = 0
					state.UpdatePreview()
					draw()
				case backend.KeyEnd:
					if len(state.Items) > 0 {
						state.SelectedIndex = len(state.Items) - 1
						state.UpdatePreview()
						draw()
					}
				}

			case backend.EventMouse:
				m := ev.Mouse
				if m.Button == backend.MouseLeft {
					// Check Sidebar Bookmark Click
					if m.X >= sideRect.X && m.X < sideRect.X+sideRect.Width &&
						m.Y >= sideRect.Y+1 && m.Y < sideRect.Y+sideRect.Height {
						bmIdx := int(m.Y) - int(sideRect.Y) - 1
						if bmIdx >= 0 && bmIdx < len(state.Bookmarks) {
							state.LoadDirectory(state.Bookmarks[bmIdx].Path)
							draw()
						}
					}
					// Check File List Item Click
					if m.X >= listRect.X && m.X < listRect.X+listRect.Width &&
						m.Y >= listRect.Y+1 && m.Y < listRect.Y+listRect.Height-1 {
						row := int(m.Y) - int(listRect.Y) - 1
						idx := state.ScrollOffset + row
						if idx >= 0 && idx < len(state.Items) {
							if state.SelectedIndex == idx && state.Items[idx].IsDir {
								state.LoadDirectory(state.Items[idx].Path)
							} else {
								state.SelectedIndex = idx
								state.UpdatePreview()
							}
							draw()
						}
					}
				} else if m.Button == backend.MouseScrollUp {
					if state.SelectedIndex > 0 {
						state.SelectedIndex--
						state.UpdatePreview()
						draw()
					}
				} else if m.Button == backend.MouseScrollDown {
					if state.SelectedIndex < len(state.Items)-1 {
						state.SelectedIndex++
						state.UpdatePreview()
						draw()
					}
				}
				draw()

			case backend.EventResize:
				draw()
			}
		}
	}
}
