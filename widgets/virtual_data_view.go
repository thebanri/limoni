package widgets

import (
	"context"
	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"strings"
)

// VirtualDataView renders the visible portion of a VirtualDataState cache.
type VirtualDataView struct {
	ID            string
	State         *VirtualDataState
	Source        VirtualDataSource
	First         int
	Prefetch      int
	Style         cell.Style
	SelectedStyle cell.Style
	EmptyText     string
	LoadingText   string
	ErrorText     string
	Offset        *int
	// HorizontalOffset scrolls non-sticky cell text by terminal columns.
	HorizontalOffset int
	// StickyColumns keeps the first N Row.Cells visible while the remaining
	// cells are horizontally scrolled.
	StickyColumns int
	// OnSelect is called with the virtual row index after a row is clicked.
	// It lets applications keep selection metadata alongside the stable RowID.
	OnSelect func(index int, row Row)
}

func (v VirtualDataView) Draw(ctx cell.Context, buf *buffer.Buffer) {
	if v.State == nil || v.Source == nil || ctx.Area.Width == 0 || ctx.Area.Height == 0 {
		return
	}
	visible := int(ctx.Area.Height)
	first := v.First
	if v.Offset != nil {
		first += *v.Offset
		if first < 0 {
			first = 0
		}
	}
	if status, _ := v.State.Status(); status == VirtualLoading {
		buf.SetString(ctx.Area.X, ctx.Area.Y, fallback(v.LoadingText, "Loading..."), ctx.Style.Merge(v.Style))
		return
	}
	if err := v.State.Refresh(context.Background(), v.Source, first, visible, v.Prefetch); err != nil {
		buf.SetString(ctx.Area.X, ctx.Area.Y, fallback(v.ErrorText, "Error: ")+err.Error(), ctx.Style.Merge(v.Style))
		return
	}
	if v.State.Count() == 0 {
		buf.SetString(ctx.Area.X, ctx.Area.Y, fallback(v.EmptyText, "No data"), ctx.Style.Merge(v.Style))
		return
	}
	style := ctx.Style.Merge(v.Style)
	// Register the viewport handler before row click regions. The router walks
	// regions from top to bottom, so row clicks must win over the wheel area
	// while wheel events still reach this handler.
	if ctx.RegisterMouse != nil && v.Offset != nil {
		ctx.RegisterMouse(ctx.Area, func(ev backend.MouseEvent) {
			max := v.State.Count() - int(ctx.Area.Height)
			if max < 0 {
				max = 0
			}
			if ev.Button == backend.MouseScrollUp && *v.Offset > 0 {
				(*v.Offset)--
			}
			if ev.Button == backend.MouseScrollDown && *v.Offset < max {
				(*v.Offset)++
			}
		})
	}
	visualRow := 0
	for index := first; visualRow < visible; index++ {
		item, ok := v.State.Row(index)
		if !ok {
			break
		}
		line := virtualRowText(item, v.HorizontalOffset, v.StickyColumns)
		rowStyle := style
		if item.ID == v.State.Selected() {
			rowStyle = rowStyle.Merge(v.SelectedStyle)
		}
		height := item.Height
		if height == 0 {
			height = 1
		}
		if visualRow+int(height) > visible {
			height = uint16(visible - visualRow)
		}
		for lineRow := uint16(0); lineRow < height; lineRow++ {
			buf.SetString(ctx.Area.X, ctx.Area.Y+uint16(visualRow)+lineRow, line, rowStyle)
		}
		if ctx.RegisterClick != nil {
			id := item.ID
			rowIndex := index
			ctx.RegisterClick(cell.NewRect(ctx.Area.X, ctx.Area.Y+uint16(visualRow), ctx.Area.Width, height), func() {
				v.State.Select(id)
				if v.OnSelect != nil {
					v.OnSelect(rowIndex, item)
				}
			})
		}
		visualRow += int(height)
	}
}

func virtualRowText(row Row, offset, sticky int) string {
	if len(row.Cells) == 0 {
		if row.Text != "" {
			return row.Text
		}
		return string(row.ID)
	}
	if sticky < 0 {
		sticky = 0
	}
	if sticky > len(row.Cells) {
		sticky = len(row.Cells)
	}
	parts := make([]string, len(row.Cells))
	for i, cell := range row.Cells {
		parts[i] = cell.Text
	}
	separator := " | "
	prefix := strings.Join(parts[:sticky], separator)
	rest := strings.Join(parts[sticky:], separator)
	if offset > 0 {
		runes := []rune(rest)
		if offset >= len(runes) {
			rest = ""
		} else {
			rest = string(runes[offset:])
		}
	}
	if prefix == "" {
		return rest
	}
	if rest == "" {
		return prefix
	}
	return prefix + separator + rest
}

func fallback(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

func (v VirtualDataView) SizeHint(max cell.Rect) (uint16, uint16) { return max.Width, max.Height }
