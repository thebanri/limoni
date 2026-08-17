package main

import (
	"fmt"
	"io"
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
	CurrentDir   string
	Items        []FileItem
	SelectedIndex int
	ScrollOffset  int
	ShowHidden   bool
	Bookmarks    []PinnedBookmark
	ActivePanel  int // 0: Sidebar, 1: FileList, 2: Preview
	PreviewLines []string
	PreviewTotal int
	PreviewErr   string
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

func (s *SuperfileState) UpdatePreview() {
	s.PreviewLines = nil
	s.PreviewErr = ""
	s.PreviewTotal = 0

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

	// File Preview: read first 200 lines
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
			cols := layout.HBox(chunks[1], layout.Percentage(22), layout.Percentage(45), layout.Percentage(33))
			sideRect = cols[0]
			listRect = cols[1]

			// PANEL 1: SIDEBAR & PINNED BOOKMARKS
			f.RenderWidget(widgets.Block{
				Title:         " 📌 PINNED LOCATIONS ",
				Borders:       widgets.BorderAll,
				BorderSymbols: widgets.SymbolsRounded,
				BorderStyle:   cell.Style{Fg: borderCard},
				Style:         cell.Style{Bg: bgCard},
			}, cols[0])

			sideInnerY := cols[0].Y + 1
			for _, bm := range state.Bookmarks {
				if sideInnerY >= cols[0].Y+cols[0].Height-1 {
					break
				}
				bmStyle := cell.Style{Fg: cell.NewColorRGB(170, 180, 200)}
				if state.CurrentDir == bm.Path {
					bmStyle = cell.Style{Fg: cell.NewColorRGB(0, 255, 180), Modifier: cell.ModifierBold}
				}
				f.Buffer.SetString(cols[0].X+2, sideInnerY, fmt.Sprintf("[%c] %s %s", bm.Key, bm.Icon, bm.Name), bmStyle)
				sideInnerY++
			}

			// PANEL 2: MAIN DIRECTORY FILE LIST
			fileListBorder := borderCard
			if state.ActivePanel == 1 {
				fileListBorder = cell.NewColorRGB(0, 220, 255)
			}
			f.RenderWidget(widgets.Block{
				Title:         fmt.Sprintf(" 📁 DIRECTORY CONTENTS (%d) ", len(state.Items)),
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
				if idx >= len(state.Items) {
					break
				}
				item := state.Items[idx]
				rowY := cols[1].Y + 1 + uint16(row)
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

			previewTitle := " 👁️ FILE PREVIEW "
			if selItem != nil {
				previewTitle = fmt.Sprintf(" 👁️ PREVIEW: %s ", selItem.Name)
			}
			f.RenderWidget(widgets.Block{
				Title:         previewTitle,
				Borders:       widgets.BorderAll,
				BorderSymbols: widgets.SymbolsRounded,
				BorderStyle:   cell.Style{Fg: borderCard},
				Style:         cell.Style{Bg: bgCard},
			}, cols[2])

			pInnerY := cols[2].Y + 1
			if selItem != nil {
				// Metadata header
				metaStyle := cell.Style{Fg: cell.NewColorRGB(0, 255, 180), Modifier: cell.ModifierBold}
				f.Buffer.SetString(cols[2].X+2, pInnerY, fmt.Sprintf("Type: %s │ Size: %s", selItem.Mode.String(), formatSize(selItem.Size)), metaStyle)
				f.Buffer.SetString(cols[2].X+2, pInnerY+1, fmt.Sprintf("Modified: %s", selItem.ModTime.Format("2006-01-02 15:04:05")), cell.Style{Fg: cell.NewColorRGB(140, 150, 165)})
				pInnerY += 3

				// Divider
				f.Buffer.SetString(cols[2].X+1, pInnerY-1, strings.Repeat("─", int(cols[2].Width)-2), cell.Style{Fg: borderCard})

				// Code or Directory Lines Preview
				maxLines := int(cols[2].Height) - 5
				for i, line := range state.PreviewLines {
					if i >= maxLines {
						break
					}
					currLineY := pInnerY + uint16(i)
					lineNumStyle := cell.Style{Fg: cell.NewColorRGB(90, 100, 120)}
					codeStyle := cell.Style{Fg: cell.NewColorRGB(215, 225, 240)}

					// Format code line with line number
					var lineStr string
					if selItem.IsDir {
						lineStr = line
					} else {
						lineStr = fmt.Sprintf("%2d │ %s", i+1, line)
					}

					maxCodeW := int(cols[2].Width) - 4
					if len([]rune(lineStr)) > maxCodeW && maxCodeW > 0 {
						lineStr = string([]rune(lineStr)[:maxCodeW])
					}

					// Basic syntax highlight tinting
					if strings.Contains(lineStr, "func ") || strings.Contains(lineStr, "package ") || strings.Contains(lineStr, "import ") || strings.Contains(lineStr, "type ") {
						codeStyle.Fg = cell.NewColorRGB(0, 220, 255)
					} else if strings.Contains(lineStr, "//") || strings.Contains(lineStr, "/*") {
						codeStyle.Fg = cell.NewColorRGB(100, 110, 130)
					} else if strings.Contains(lineStr, "\"") {
						codeStyle.Fg = cell.NewColorRGB(46, 204, 113)
					}

					if !selItem.IsDir {
						f.Buffer.SetString(cols[2].X+2, currLineY, fmt.Sprintf("%2d │ ", i+1), lineNumStyle)
						codeOnly := line
						if len([]rune(codeOnly)) > maxCodeW-5 && maxCodeW-5 > 0 {
							codeOnly = string([]rune(codeOnly)[:maxCodeW-5])
						}
						f.Buffer.SetString(cols[2].X+7, currLineY, codeOnly, codeStyle)
					} else {
						f.Buffer.SetString(cols[2].X+2, currLineY, lineStr, codeStyle)
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
