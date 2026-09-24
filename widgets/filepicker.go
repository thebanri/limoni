package widgets

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
)

// FileEntry is one line of a FilePicker.
type FileEntry struct {
	Name  string
	IsDir bool
	Size  int64
}

// FilePickerState is the directory being browsed and the cursor in it.
// Reading the directory happens in Load and HandleKey, never in Draw.
type FilePickerState struct {
	Dir      string
	Entries  []FileEntry
	Selected int
	Offset   int

	// ShowHidden lists dot files; Ctrl+H toggles it.
	ShowHidden bool
	// Extensions, if set, lists only files with one of these extensions
	// (".go", ".md"); directories are always listed.
	Extensions []string
	// DirsOnly lists directories alone, for picking a folder.
	DirsOnly bool

	// Chosen is the path of the file picked with Enter, or of the
	// directory when DirsOnly is set and Enter is pressed on ".".
	Chosen string
	// Err is the error from the last Load, if the directory could not be read.
	Err error

	rows    int
	top     uint16
	area    cell.Rect
	onMouse func(driver.MouseEvent)
}

// NewFilePickerState opens dir; an empty dir is the working directory.
func NewFilePickerState(dir string) *FilePickerState {
	s := &FilePickerState{}
	s.Load(dir)
	return s
}

// Load lists dir: ".." first unless dir is a root, then directories, then
// files, each group by name. On error the old listing is kept and Err set.
func (s *FilePickerState) Load(dir string) error {
	if dir == "" {
		dir = "."
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		s.Err = err
		return err
	}
	list, err := os.ReadDir(abs)
	if err != nil {
		s.Err = err
		return err
	}
	s.Dir, s.Err = abs, nil
	s.Entries = s.Entries[:0]
	if filepath.Dir(abs) != abs {
		s.Entries = append(s.Entries, FileEntry{Name: "..", IsDir: true})
	}
	for _, e := range list {
		name := e.Name()
		if !s.ShowHidden && strings.HasPrefix(name, ".") {
			continue
		}
		isDir := e.IsDir()
		if e.Type()&os.ModeSymlink != 0 {
			if info, err := os.Stat(filepath.Join(abs, name)); err == nil {
				isDir = info.IsDir()
			}
		}
		if !isDir && (s.DirsOnly || !s.wants(name)) {
			continue
		}
		var size int64
		if !isDir {
			if info, err := e.Info(); err == nil {
				size = info.Size()
			}
		}
		s.Entries = append(s.Entries, FileEntry{Name: name, IsDir: isDir, Size: size})
	}
	start := 0
	if len(s.Entries) > 0 && s.Entries[0].Name == ".." {
		start = 1
	}
	slices.SortFunc(s.Entries[start:], func(a, b FileEntry) int {
		if a.IsDir != b.IsDir {
			if a.IsDir {
				return -1
			}
			return 1
		}
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
	s.Selected, s.Offset = 0, 0
	return nil
}

func (s *FilePickerState) wants(name string) bool {
	if len(s.Extensions) == 0 {
		return true
	}
	ext := filepath.Ext(name)
	for _, want := range s.Extensions {
		if strings.EqualFold(ext, want) {
			return true
		}
	}
	return false
}

// SelectedEntry is the entry under the cursor, if there is one.
func (s *FilePickerState) SelectedEntry() (FileEntry, bool) {
	if s.Selected < 0 || s.Selected >= len(s.Entries) {
		return FileEntry{}, false
	}
	return s.Entries[s.Selected], true
}

// open enters the selected directory, or chooses the selected file.
func (s *FilePickerState) open() bool {
	e, ok := s.SelectedEntry()
	if !ok {
		return false
	}
	if !e.IsDir {
		s.Chosen = filepath.Join(s.Dir, e.Name)
		return true
	}
	if e.Name == ".." {
		return s.up()
	}
	return s.Load(filepath.Join(s.Dir, e.Name)) == nil
}

// up goes to the parent directory with the cursor on the one just left.
func (s *FilePickerState) up() bool {
	parent := filepath.Dir(s.Dir)
	if parent == s.Dir {
		return false
	}
	left := filepath.Base(s.Dir)
	if s.Load(parent) != nil {
		return false
	}
	for i, e := range s.Entries {
		if e.Name == left {
			s.Selected = i
		}
	}
	return true
}

// HandleKey moves the cursor (↑ ↓ Page Up/Down Home End), enters a
// directory (Enter, →), goes up (Backspace, ←) and chooses a file (Enter),
// which sets Chosen. Ctrl+H shows or hides dot files.
func (s *FilePickerState) HandleKey(ev driver.KeyEvent) bool {
	if s == nil {
		return false
	}
	page := max(s.rows-1, 1)
	switch ev.Type {
	case driver.KeyArrowUp:
		s.move(-1)
	case driver.KeyArrowDown:
		s.move(1)
	case driver.KeyPageUp:
		s.move(-page)
	case driver.KeyPageDown:
		s.move(page)
	case driver.KeyHome:
		s.move(-len(s.Entries))
	case driver.KeyEnd:
		s.move(len(s.Entries))
	case driver.KeyEnter:
		if ev.Alt || ev.Ctrl || ev.Shift {
			return false
		}
		return s.open()
	case driver.KeyArrowRight:
		if e, ok := s.SelectedEntry(); ok && e.IsDir && e.Name != ".." {
			return s.open()
		}
		return false
	case driver.KeyBackspace, driver.KeyArrowLeft:
		return s.up()
	case driver.KeyRune:
		if ev.Ctrl && (ev.Ch == 'h' || ev.Ch == 'H') {
			s.ShowHidden = !s.ShowHidden
			sel, _ := s.SelectedEntry()
			s.Load(s.Dir)
			for i, e := range s.Entries {
				if e.Name == sel.Name {
					s.Selected = i
				}
			}
			return true
		}
		return false
	default:
		return false
	}
	return true
}

func (s *FilePickerState) move(delta int) {
	s.Selected = max(0, min(s.Selected+delta, len(s.Entries)-1))
}

func (s *FilePickerState) handler() func(driver.MouseEvent) {
	if s.onMouse == nil {
		s.onMouse = func(ev driver.MouseEvent) {
			switch ev.Button {
			case driver.MouseScrollUp:
				s.move(-1)
			case driver.MouseScrollDown:
				s.move(1)
			case driver.MouseLeft:
				if ev.Drag || ev.Y < s.top {
					return
				}
				i := s.Offset + int(ev.Y-s.top)
				if i >= len(s.Entries) {
					return
				}
				// A click selects; a click on the selected entry opens it.
				if i == s.Selected {
					s.open()
					return
				}
				s.Selected = i
			}
		}
	}
	return s.onMouse
}

// FilePicker lists a directory for choosing a file or a folder: the path on
// the first row, then ".." and the entries, directories first and marked
// with a trailing slash, file sizes on the right.
type FilePicker struct {
	ID    string
	State *FilePickerState

	// HideHeader drops the path row; HideSizes the size column.
	HideHeader bool
	HideSizes  bool

	Style         cell.Style
	HeaderStyle   cell.Style
	DirStyle      cell.Style
	SizeStyle     cell.Style
	SelectedStyle cell.Style
	ErrorStyle    cell.Style
}

// Draw renders the listing. It does not allocate.
func (fp FilePicker) Draw(ctx cell.Context, buf *buffer.Buffer) {
	area := ctx.Area
	s := fp.State
	if area.Width == 0 || area.Height == 0 || s == nil {
		return
	}
	if fp.ID != "" && ctx.RegisterFocus != nil {
		ctx.RegisterFocus(fp.ID)
	}
	base := ctx.Style.Merge(fp.Style)
	for y := area.Y; y < area.Y+area.Height; y++ {
		for x := area.X; x < area.X+area.Width; x++ {
			buf.SetCell(x, y, cell.Cell{Content: ' ', Style: base})
		}
	}

	y := area.Y
	if !fp.HideHeader {
		style := base.Merge(fp.HeaderStyle)
		if s.Err != nil {
			style = base.Merge(fp.ErrorStyle)
		}
		setLeftClipped(buf, area.X, y, s.Dir, style, int(area.Width))
		y++
	}
	rows := int(area.Y+area.Height) - int(y)
	s.rows, s.top, s.area = rows, y, area
	if rows <= 0 {
		return
	}
	if s.Selected < s.Offset {
		s.Offset = s.Selected
	}
	if s.Selected >= s.Offset+rows {
		s.Offset = s.Selected - rows + 1
	}
	s.Offset = max(0, min(s.Offset, len(s.Entries)-rows))

	selected := fp.SelectedStyle
	if selected == (cell.Style{}) {
		selected = cell.Style{}.Reverse()
	}
	for i := 0; i < rows && s.Offset+i < len(s.Entries); i++ {
		e := s.Entries[s.Offset+i]
		row := y + uint16(i)
		style := base
		if e.IsDir {
			style = style.Merge(fp.DirStyle)
		}
		if s.Offset+i == s.Selected {
			style = style.Merge(selected)
			for x := area.X; x < area.X+area.Width; x++ {
				buf.SetCell(x, row, cell.Cell{Content: ' ', Style: style})
			}
		}

		var sizeBuf [8]byte
		size := sizeBuf[:0]
		if !e.IsDir && !fp.HideSizes {
			size = appendHumanSize(size, e.Size)
		}
		nameWidth := int(area.Width)
		if len(size) > 0 {
			nameWidth -= len(size) + 1
		}
		if e.IsDir {
			nameWidth-- // room for the slash
		}
		x := area.X + setClipped(buf, area.X, row, e.Name, style, nameWidth)
		if e.IsDir && x < area.X+area.Width {
			buf.SetCell(x, row, cell.Cell{Content: '/', Style: style})
		}
		if len(size) > 0 && int(area.Width) > len(size) {
			sx := area.X + area.Width - uint16(len(size))
			sizeStyle := style.Merge(fp.SizeStyle)
			for j, ch := range size {
				buf.SetCell(sx+uint16(j), row, cell.Cell{Content: rune(ch), Style: sizeStyle})
			}
		}
	}

	if ctx.RegisterMouse != nil {
		ctx.RegisterMouse(cell.NewRect(area.X, y, area.Width, uint16(rows)), s.handler())
	}
}

// setLeftClipped draws s, cutting it from the left with "…" when it is too
// wide: the end of a path is the part worth keeping.
func setLeftClipped(buf *buffer.Buffer, x, y uint16, s string, style cell.Style, maxW int) {
	w := cell.StringWidth(s)
	if w <= maxW {
		buf.SetStringWithin(x, y, s, style, uint16(maxW))
		return
	}
	if maxW < 2 {
		return
	}
	rest, cut := skipColumns(s, w-(maxW-1))
	for cut < w-(maxW-1) && rest != "" { // a wide cluster straddled the cut
		var cw int
		_, cw, rest = cell.NextCluster(rest)
		cut += cw
	}
	buf.SetCell(x, y, cell.Cell{Content: '…', Style: style})
	buf.SetStringWithin(x+1, y, rest, style, uint16(maxW-1))
}

// appendHumanSize appends n bytes as at most five characters: "999B",
// "1.5K", "12M", "3.2G".
func appendHumanSize(dst []byte, n int64) []byte {
	if n < 1000 {
		return append(strconv.AppendInt(dst, n, 10), 'B')
	}
	f := float64(n)
	for _, unit := range []byte("KMGTPE") {
		f /= 1024
		if f < 1000 || unit == 'E' {
			if f < 10 {
				dst = strconv.AppendFloat(dst, f, 'f', 1, 64)
			} else {
				dst = strconv.AppendInt(dst, int64(f+0.5), 10)
			}
			return append(dst, unit)
		}
	}
	return dst
}

// SizeHint takes all the space offered.
func (fp FilePicker) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	return maxArea.Width, maxArea.Height
}

// AccessibilityNode is a list whose value is the selected entry.
func (fp FilePicker) AccessibilityNode(bounds cell.Rect, focused bool) accessibility.AccessibilityNode {
	var state accessibility.NodeState
	if focused {
		state |= accessibility.StateFocused
	}
	node := accessibility.AccessibilityNode{ID: fp.ID, Role: accessibility.RoleList, Label: "Files", State: state, Bounds: bounds}
	if s := fp.State; s != nil {
		node.Label = s.Dir
		if e, ok := s.SelectedEntry(); ok {
			node.Value = e.Name
			node.Position, node.SetSize = s.Selected+1, len(s.Entries)
		}
	}
	return node
}
