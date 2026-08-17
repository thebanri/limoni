package widgets

import (
	"unicode/utf8"

	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// TreeNode represents a single item in a hierarchical tree.
type TreeNode struct {
	ID       string
	Label    string
	Icon     string
	Expanded bool
	Children []TreeNode
	Data     any
	Style    cell.Style
}

// TreeViewState maintains the selected node, scroll offset, and expansion states.
type TreeViewState struct {
	SelectedID  string
	Offset      int
	expandedMap map[string]bool
	flatCache   []flatTreeNode
	cacheValid  bool
}

type flatTreeNode struct {
	node      *TreeNode
	depth     int
	isLast    bool
	parentEnd []bool
}

// NewTreeViewState creates a new initialized TreeViewState.
func NewTreeViewState() *TreeViewState {
	return &TreeViewState{
		expandedMap: make(map[string]bool),
	}
}

// IsExpanded returns true if the node ID is currently expanded.
func (s *TreeViewState) IsExpanded(id string, defaultExpanded bool) bool {
	if s == nil || s.expandedMap == nil {
		return defaultExpanded
	}
	if val, ok := s.expandedMap[id]; ok {
		return val
	}
	return defaultExpanded
}

// Toggle inverts the expansion state of the given node ID.
func (s *TreeViewState) Toggle(id string, defaultExpanded bool) {
	if s == nil {
		return
	}
	if s.expandedMap == nil {
		s.expandedMap = make(map[string]bool)
	}
	current := s.IsExpanded(id, defaultExpanded)
	s.expandedMap[id] = !current
	s.cacheValid = false
}

// Expand marks the given node ID as expanded.
func (s *TreeViewState) Expand(id string) {
	if s == nil {
		return
	}
	if s.expandedMap == nil {
		s.expandedMap = make(map[string]bool)
	}
	s.expandedMap[id] = true
	s.cacheValid = false
}

// Collapse marks the given node ID as collapsed.
func (s *TreeViewState) Collapse(id string) {
	if s == nil {
		return
	}
	if s.expandedMap == nil {
		s.expandedMap = make(map[string]bool)
	}
	s.expandedMap[id] = false
	s.cacheValid = false
}

// Select sets the currently highlighted node ID.
func (s *TreeViewState) Select(id string) {
	if s == nil {
		return
	}
	s.SelectedID = id
}

// HandleKey processes keyboard navigation for the tree.
func (s *TreeViewState) HandleKey(ev backend.KeyEvent, roots []TreeNode) bool {
	if s == nil || len(roots) == 0 {
		return false
	}
	flat := s.Flatten(roots)
	if len(flat) == 0 {
		return false
	}

	curIdx := -1
	for i, item := range flat {
		if item.node.ID == s.SelectedID {
			curIdx = i
			break
		}
	}

	switch ev.Type {
	case backend.KeyArrowDown:
		if curIdx < len(flat)-1 {
			s.SelectedID = flat[curIdx+1].node.ID
			return true
		}
	case backend.KeyArrowUp:
		if curIdx > 0 {
			s.SelectedID = flat[curIdx-1].node.ID
			return true
		} else if curIdx == -1 && len(flat) > 0 {
			s.SelectedID = flat[0].node.ID
			return true
		}
	case backend.KeyPageDown:
		next := curIdx + 10
		if next >= len(flat) {
			next = len(flat) - 1
		}
		if next >= 0 && next < len(flat) {
			s.SelectedID = flat[next].node.ID
			return true
		}
	case backend.KeyPageUp:
		prev := curIdx - 10
		if prev < 0 {
			prev = 0
		}
		if prev < len(flat) {
			s.SelectedID = flat[prev].node.ID
			return true
		}
	case backend.KeyArrowRight:
		if curIdx >= 0 && len(flat[curIdx].node.Children) > 0 {
			if !s.IsExpanded(flat[curIdx].node.ID, flat[curIdx].node.Expanded) {
				s.Expand(flat[curIdx].node.ID)
				return true
			} else if curIdx < len(flat)-1 {
				s.SelectedID = flat[curIdx+1].node.ID
				return true
			}
		}
	case backend.KeyArrowLeft:
		if curIdx >= 0 {
			if len(flat[curIdx].node.Children) > 0 && s.IsExpanded(flat[curIdx].node.ID, flat[curIdx].node.Expanded) {
				s.Collapse(flat[curIdx].node.ID)
				return true
			} else if flat[curIdx].depth > 0 {
				// Move to parent
				targetDepth := flat[curIdx].depth - 1
				for i := curIdx - 1; i >= 0; i-- {
					if flat[i].depth == targetDepth {
						s.SelectedID = flat[i].node.ID
						return true
					}
				}
			}
		}
	case backend.KeyEnter, backend.KeySpace:
		if curIdx >= 0 && len(flat[curIdx].node.Children) > 0 {
			s.Toggle(flat[curIdx].node.ID, flat[curIdx].node.Expanded)
			return true
		}
	case backend.KeyHome:
		if len(flat) > 0 {
			s.SelectedID = flat[0].node.ID
			return true
		}
	case backend.KeyEnd:
		if len(flat) > 0 {
			s.SelectedID = flat[len(flat)-1].node.ID
			return true
		}
	case backend.KeyRune:
		switch ev.Ch {
		case 'j', 'J':
			if curIdx < len(flat)-1 {
				s.SelectedID = flat[curIdx+1].node.ID
				return true
			}
		case 'k', 'K':
			if curIdx > 0 {
				s.SelectedID = flat[curIdx-1].node.ID
				return true
			}
		case 'l', 'L':
			if curIdx >= 0 && len(flat[curIdx].node.Children) > 0 {
				if !s.IsExpanded(flat[curIdx].node.ID, flat[curIdx].node.Expanded) {
					s.Expand(flat[curIdx].node.ID)
					return true
				} else if curIdx < len(flat)-1 {
					s.SelectedID = flat[curIdx+1].node.ID
					return true
				}
			}
		case 'h', 'H':
			if curIdx >= 0 {
				if len(flat[curIdx].node.Children) > 0 && s.IsExpanded(flat[curIdx].node.ID, flat[curIdx].node.Expanded) {
					s.Collapse(flat[curIdx].node.ID)
					return true
				} else if flat[curIdx].depth > 0 {
					targetDepth := flat[curIdx].depth - 1
					for i := curIdx - 1; i >= 0; i-- {
						if flat[i].depth == targetDepth {
							s.SelectedID = flat[i].node.ID
							return true
						}
					}
				}
			}
		}
	}
	return false
}

// Flatten produces a visible linear list of nodes based on current expansion state.
func (s *TreeViewState) Flatten(roots []TreeNode) []flatTreeNode {
	var flat []flatTreeNode
	var traverse func(nodes []TreeNode, depth int, parentEnd []bool)
	traverse = func(nodes []TreeNode, depth int, parentEnd []bool) {
		for i := range nodes {
			node := &nodes[i]
			isLast := i == len(nodes)-1
			curParentEnd := append(parentEnd, isLast)

			flat = append(flat, flatTreeNode{
				node:      node,
				depth:     depth,
				isLast:    isLast,
				parentEnd: curParentEnd,
			})

			expanded := node.Expanded
			if s != nil {
				expanded = s.IsExpanded(node.ID, node.Expanded)
			}

			if expanded && len(node.Children) > 0 {
				traverse(node.Children, depth+1, curParentEnd)
			}
		}
	}

	traverse(roots, 0, nil)
	return flat
}

// FindNode searches for a node with the matching ID in the tree hierarchy.
func FindNode(roots []TreeNode, id string) *TreeNode {
	for i := range roots {
		if roots[i].ID == id {
			return &roots[i]
		}
		if len(roots[i].Children) > 0 {
			if found := FindNode(roots[i].Children, id); found != nil {
				return found
			}
		}
	}
	return nil
}

// TreeView renders a hierarchical collapsible tree with guide lines and selection.
type TreeView struct {
	ID            string
	Roots         []TreeNode
	State         *TreeViewState
	Style         cell.Style
	FocusedStyle  cell.Style
	SelectedStyle cell.Style
	GuideStyle    cell.Style
	ShowGuides    bool
	IndentWidth   int
}

// Draw renders the tree widget to the terminal buffer.
func (t TreeView) Draw(ctx cell.Context, buf *buffer.Buffer) {
	area := ctx.Area
	if area.Width == 0 || area.Height == 0 || len(t.Roots) == 0 {
		return
	}

	if t.ID != "" && ctx.RegisterFocus != nil {
		ctx.RegisterFocus(t.ID)
	}

	baseStyle := ctx.Style.Merge(t.Style)
	if ctx.IsFocused(t.ID) {
		baseStyle = baseStyle.Merge(t.FocusedStyle)
	}
	selStyle := baseStyle.Merge(t.SelectedStyle)
	if t.SelectedStyle == (cell.Style{}) && ctx.ThemeStyle != nil {
		selStyle = baseStyle.Merge(ctx.ThemeStyle("selection"))
	}
	guideStyle := baseStyle.Merge(t.GuideStyle)
	if t.GuideStyle == (cell.Style{}) {
		guideStyle = baseStyle.Merge(cell.Style{Fg: cell.NewColorRGB(90, 100, 115)})
	}

	state := t.State
	if state == nil {
		state = &TreeViewState{expandedMap: make(map[string]bool)}
	}

	flat := state.Flatten(t.Roots)
	if len(flat) == 0 {
		return
	}

	// Calculate selection index
	selIdx := -1
	if state.SelectedID != "" {
		for i, item := range flat {
			if item.node.ID == state.SelectedID {
				selIdx = i
				break
			}
		}
	}

	// Auto-scroll to ensure selection is in view
	visibleHeight := int(area.Height)
	if selIdx >= 0 {
		if selIdx < state.Offset {
			state.Offset = selIdx
		} else if selIdx >= state.Offset+visibleHeight {
			state.Offset = selIdx - visibleHeight + 1
		}
	}
	if state.Offset < 0 {
		state.Offset = 0
	}
	maxOffset := len(flat) - visibleHeight
	if maxOffset < 0 {
		maxOffset = 0
	}
	if state.Offset > maxOffset {
		state.Offset = maxOffset
	}

	// Mouse scroll registration
	if ctx.RegisterMouse != nil && t.State != nil {
		st := t.State
		ctx.RegisterMouse(ctx.Area, func(ev backend.MouseEvent) {
			if ev.Button == backend.MouseScrollUp {
				st.Offset--
				if st.Offset < 0 {
					st.Offset = 0
				}
			} else if ev.Button == backend.MouseScrollDown {
				st.Offset++
				if st.Offset > maxOffset {
					st.Offset = maxOffset
				}
			}
		})
	}

	indent := t.IndentWidth
	if indent < 2 {
		indent = 2
	}

	for i := 0; i < visibleHeight; i++ {
		rowIdx := state.Offset + i
		if rowIdx >= len(flat) {
			break
		}

		item := flat[rowIdx]
		currY := area.Y + uint16(i)
		isSel := item.node.ID == state.SelectedID
		rowStyle := baseStyle.Merge(item.node.Style)
		if isSel {
			rowStyle = selStyle
		}

		// Clear row background
		for x := area.X; x < area.X+area.Width; x++ {
			if c := buf.Get(x, currY); c != nil {
				c.Content = ' '
				c.Style = rowStyle
			}
		}

		cursorX := area.X

		// Draw tree guide lines
		if t.ShowGuides && item.depth > 0 {
			for d := 0; d < item.depth; d++ {
				guideChar := ' '
				if d < len(item.parentEnd)-1 && !item.parentEnd[d] {
					guideChar = '│'
				}
				buf.SetCell(cursorX, currY, cell.Cell{Content: guideChar, Style: guideStyle})
				cursorX += uint16(indent)
			}
		} else {
			cursorX += uint16(item.depth * indent)
		}

		// Branch symbol or expander icon
		if len(item.node.Children) > 0 {
			expChar := "▶ "
			if state.IsExpanded(item.node.ID, item.node.Expanded) {
				expChar = "▼ "
			}
			buf.SetString(cursorX, currY, expChar, rowStyle)
			cursorX += uint16(utf8.RuneCountInString(expChar))
		} else if t.ShowGuides && item.depth > 0 {
			branchChar := "├─ "
			if item.isLast {
				branchChar = "└─ "
			}
			buf.SetString(cursorX, currY, branchChar, guideStyle)
			cursorX += uint16(utf8.RuneCountInString(branchChar))
		} else {
			buf.SetString(cursorX, currY, "  ", rowStyle)
			cursorX += 2
		}

		// Draw Node Icon
		if item.node.Icon != "" {
			buf.SetString(cursorX, currY, item.node.Icon+" ", rowStyle)
			cursorX += uint16(utf8.RuneCountInString(item.node.Icon) + 1)
		}

		// Draw Node Label
		buf.SetString(cursorX, currY, item.node.Label, rowStyle)

		// Register click region for interactive toggles and selection
		if ctx.RegisterClick != nil && t.State != nil {
			targetID := item.node.ID
			st := t.State
			hasChildren := len(item.node.Children) > 0
			defExp := item.node.Expanded
			id := t.ID
			setFocus := ctx.SetFocus

			itemRect := cell.Rect{
				X:      area.X,
				Y:      currY,
				Width:  area.Width,
				Height: 1,
			}
			ctx.RegisterClick(itemRect, func() {
				st.Select(targetID)
				if hasChildren {
					st.Toggle(targetID, defExp)
				}
				if id != "" && setFocus != nil {
					setFocus(id)
				}
			})
		}
	}
}

// SizeHint returns the preferred dimensions for TreeView.
func (t TreeView) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	state := t.State
	if state == nil {
		state = &TreeViewState{expandedMap: make(map[string]bool)}
	}
	flat := state.Flatten(t.Roots)
	h := uint16(len(flat))
	if h > maxArea.Height {
		h = maxArea.Height
	}
	return maxArea.Width, h
}

// AccessibilityNode returns semantic accessibility tree information.
func (t TreeView) AccessibilityNode(bounds cell.Rect, focused bool) accessibility.AccessibilityNode {
	st := accessibility.NodeState(0)
	if focused {
		st |= accessibility.StateFocused
	}
	selID := ""
	if t.State != nil {
		selID = t.State.SelectedID
	}
	return accessibility.AccessibilityNode{
		ID:     t.ID,
		Role:   accessibility.RoleTree,
		Label:  "Tree View",
		Value:  selID,
		State:  st,
		Bounds: bounds,
	}
}
