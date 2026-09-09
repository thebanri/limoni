package main

import (
	"fmt"

	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/layout"
	"github.com/thebanri/limoni/widgets"
)

func drawTabTreeView(f *terminal.Frame, area cell.Rect, state *AppState, theme Theme) {
	// Split: Left TreeView (46%), Right Inspector & Preview (54%)
	cols := layout.HBoxWithGap(area, 1,
		layout.Percentage(46),
		layout.Percentage(54),
	)

	// ==========================================
	// 1. LEFT: TREEVIEW FILE EXPLORER
	// ==========================================
	treeBlock := widgets.Block{
		Title:          " PROJECT EXPLORER (widgets.TreeView) ",
		TitleAlignment: widgets.AlignLeft,
		Borders:        widgets.BorderAll,
		BorderSymbols:  widgets.SymbolsRounded,
		BorderStyle:    cell.Style{Fg: theme.Primary},
		TitleStyle:     cell.Style{Fg: theme.Accent, Modifier: cell.ModifierBold},
	}
	f.RenderWidget(treeBlock, cols[0])

	innerLeft := layout.Padded(cols[0], 1, 1, 1, 1)

	// Split inner left: Options bar at top (1 line), TreeView body (Fill)
	leftSplit := layout.VBoxWithGap(innerLeft, 1,
		layout.Fixed(1), // Checkbox: Show Guides
		layout.Fill(),   // TreeView
	)

	cbGuides := widgets.Checkbox{
		ID:           "cb_tree_guides",
		Checked:      &state.TreeShowGuides,
		Label:        "Show Tree Guides & Branch Connectors",
		Style:        cell.Style{Fg: theme.Text},
		FocusedStyle: cell.Style{Fg: theme.Accent, Modifier: cell.ModifierBold},
	}
	f.RenderWidget(cbGuides, leftSplit[0])

	treeWidget := widgets.TreeView{
		ID:            "project_tree",
		Roots:         state.TreeRoots,
		State:         state.TreeState,
		ShowGuides:    state.TreeShowGuides,
		IndentWidth:   2,
		Style:         cell.Style{Fg: theme.Text},
		SelectedStyle: cell.Style{Fg: theme.Accent, Bg: theme.BgCard, Modifier: cell.ModifierBold},
		FocusedStyle:  cell.Style{Fg: theme.Primary},
		GuideStyle:    cell.Style{Fg: theme.Muted},
	}
	f.RenderWidget(treeWidget, leftSplit[1])

	// ==========================================
	// 2. RIGHT: FILE INSPECTOR & CODE PREVIEW
	// ==========================================
	inspectBlock := widgets.Block{
		Title:          " FILE INSPECTOR & COMPONENT OVERVIEW ",
		TitleAlignment: widgets.AlignLeft,
		Borders:        widgets.BorderAll,
		BorderSymbols:  widgets.SymbolsRounded,
		BorderStyle:    cell.Style{Fg: theme.Secondary},
		TitleStyle:     cell.Style{Fg: theme.Secondary, Modifier: cell.ModifierBold},
	}
	f.RenderWidget(inspectBlock, cols[1])

	innerRight := layout.Padded(cols[1], 1, 1, 1, 1)
	drawTreeInspector(f, innerRight, state, theme)
}

func drawTreeInspector(f *terminal.Frame, area cell.Rect, state *AppState, theme Theme) {
	selID := state.TreeState.SelectedID
	if selID == "" {
		selID = "lemon_glb"
	}

	y := area.Y
	f.Buffer.SetString(area.X, y, fmt.Sprintf("Selected Node: %s", selID), cell.Style{Fg: theme.Accent, Modifier: cell.ModifierBold})
	y += 2

	fileData := getFileInfo(selID)

	metaBlock := widgets.Block{
		Title:          fmt.Sprintf(" [%s] %s ", fileData.TypeTag, fileData.Name),
		TitleAlignment: widgets.AlignLeft,
		Borders:        widgets.BorderAll,
		BorderSymbols:  widgets.SymbolsRounded,
		BorderStyle:    cell.Style{Fg: theme.Primary},
		TitleStyle:     cell.Style{Fg: theme.Text, Modifier: cell.ModifierBold},
	}

	// Split area: Metadata box (Fixed 5 rows), Code snippet preview (Fill)
	chunks := layout.VBoxWithGap(cell.Rect{X: area.X, Y: y, Width: area.Width, Height: area.Height - 2}, 1,
		layout.Fixed(5),
		layout.Fill(),
	)

	f.RenderWidget(metaBlock, chunks[0])
	innerMeta := layout.Padded(chunks[0], 1, 1, 1, 1)
	f.Buffer.SetString(innerMeta.X+1, innerMeta.Y, fmt.Sprintf("• Path: %s", fileData.Path), cell.Style{Fg: theme.Text})
	f.Buffer.SetString(innerMeta.X+1, innerMeta.Y+1, fmt.Sprintf("• Type: %s │ Size: %s", fileData.Type, fileData.Size), cell.Style{Fg: theme.Muted})
	f.Buffer.SetString(innerMeta.X+1, innerMeta.Y+2, fmt.Sprintf("• Info: %s", fileData.Description), cell.Style{Fg: theme.Accent})

	codeBlock := widgets.Block{
		Title:          " LIVE SOURCE CODE / METADATA PREVIEW ",
		TitleAlignment: widgets.AlignLeft,
		Borders:        widgets.BorderAll,
		BorderSymbols:  widgets.SymbolsRounded,
		BorderStyle:    cell.Style{Fg: theme.Muted},
		TitleStyle:     cell.Style{Fg: theme.Text, Modifier: cell.ModifierBold},
	}
	f.RenderWidget(codeBlock, chunks[1])
	innerCode := layout.Padded(chunks[1], 1, 1, 1, 1)

	for i, line := range fileData.CodeSnippet {
		if uint16(i) >= innerCode.Height {
			break
		}
		style := cell.Style{Fg: theme.Text}
		if len(line) > 0 && (line[0] == '/' || line[0] == '#') {
			style.Fg = theme.Muted
		} else if len(line) > 4 && line[0:4] == "type" || (len(line) > 4 && line[0:4] == "func") {
			style.Fg = theme.Accent
			style.Modifier = cell.ModifierBold
		}
		f.Buffer.SetString(innerCode.X+1, innerCode.Y+uint16(i), line, style)
	}
}

type fileMeta struct {
	TypeTag     string
	Name        string
	Path        string
	Type        string
	Size        string
	Description string
	CodeSnippet []string
}

func getFileInfo(id string) fileMeta {
	switch id {
	case "lemon_glb":
		return fileMeta{
			TypeTag:     "3D",
			Name:        "limoni.glb",
			Path:        "examples/demo/limoni.glb",
			Type:        "Binary 3D Model (glTF 2.0)",
			Size:        "1.8 MB (26,153 Triangles)",
			Description: "Limoni Mascot: Go Gopher (Cyan Blue) holding Lemon (Citrus Yellow).",
			CodeSnippet: []string{
				"// glTF 2.0 Binary Model Container",
				"Header:",
				"  Magic:       0x46546C67 ('glTF')",
				"  Version:     2",
				"  Faces:       26,153 Triangles",
				"",
				"JSON Chunk 0: Scene, Node, Mesh & Material definitions",
				"  Materials:   [ { pbrMetallicRoughness: { baseColorTexture: 0 } } ]",
				"",
				"BIN Chunk 1: Positions, UVs, and TrueColor Texture Atlas (Blue Gopher & Yellow Lemon)",
			},
		}

	case "term_engine":
		return fileMeta{
			TypeTag:     "GO",
			Name:        "terminal.go",
			Path:        "core/terminal/terminal.go",
			Type:        "Go Source File (Engine Core)",
			Size:        "21.2 KB",
			Description: "Hardware double-buffer diffing engine with zero-alloc inner loops.",
			CodeSnippet: []string{
				"package terminal",
				"",
				"// Draw executes the render closure on the front buffer,",
				"// compares against back buffer, and emits minimal ANSI escape sequences.",
				"func (t *Terminal) Draw(renderFn func(f *Frame)) error {",
				"    f := t.framePool.Get().(*Frame)",
				"    defer t.framePool.Put(f)",
				"",
				"    renderFn(f)",
				"    diffs := t.diffEngine.ComputeDiff(t.backBuffer, f.Buffer)",
				"    return t.backend.FlushDiffs(diffs) // ~18.2 µs per frame",
				"}",
			},
		}

	case "treeview_w":
		return fileMeta{
			TypeTag:     "WIDGET",
			Name:        "treeview.go",
			Path:        "widgets/treeview.go",
			Type:        "Go Source File (Widget)",
			Size:        "12.1 KB",
			Description: "Interactive hierarchical TreeView with mouse and keyboard navigation.",
			CodeSnippet: []string{
				"package widgets",
				"",
				"type TreeView struct {",
				"    ID          string",
				"    Roots       []TreeNode",
				"    State       *TreeViewState",
				"    ShowGuides  bool",
				"    IndentWidth int",
				"}",
				"",
				"// Toggling and node selection seamlessly handled via Mouse clicks & keyboard.",
			},
		}

	case "image_w":
		return fileMeta{
			TypeTag:     "WIDGET",
			Name:        "image.go",
			Path:        "widgets/image.go",
			Type:        "Go Source File (Widget)",
			Size:        "11.8 KB",
			Description: "High-performance direct pixel terminal image renderer.",
			CodeSnippet: []string{
				"package widgets",
				"",
				"// Image renders arbitrary Go image.Image instances using",
				"// dual-pixel TrueColor half-blocks ('▀') and circle avatars.",
				"type Image struct {",
				"    Image      image.Image",
				"    CircleMask bool",
				"    Style      cell.Style",
				"}",
			},
		}

	default:
		return fileMeta{
			TypeTag:     "MOD",
			Name:        id,
			Path:        "Limoni Architecture Component",
			Type:        "High-Performance Go Module",
			Size:        "Varies",
			Description: "Limoni Engine Core component with hardware accelerated rendering.",
			CodeSnippet: []string{
				"// Limoni Engine Architecture",
				"// Built from the ground up for maximum speed and minimum allocations.",
				"• Zero heap allocations on hot render loops",
				"• Native TrueColor ANSI Terminal Graphic Pipelines",
				"• SGR 1006 Extended Mouse Tracking & Event Routing",
			},
		}
	}
}
