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
	FocusedStyle  cell.Style
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
	if v.ID != "" && ctx.RegisterFocus != nil {
		ctx.RegisterFocus(v.ID)
	}
	if v.ID != "" && ctx.RegisterClick != nil {
		ctx.RegisterClick(ctx.Area, func() {
			if ctx.SetFocus != nil {
				ctx.SetFocus(v.ID)
			}
		})
	}
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
	perRowClick := ctx.RegisterMouse == nil && ctx.RegisterClick != nil

	// Register single mouse handler for viewport scrolling and click routing
	if ctx.RegisterMouse != nil {
		ctx.RegisterMouse(ctx.Area, func(ev backend.MouseEvent) {
			if ev.Button == backend.MouseScrollUp && v.Offset != nil && *v.Offset > 0 {
				(*v.Offset)--
				return
			}
			if ev.Button == backend.MouseScrollDown && v.Offset != nil {
				max := v.State.Count() - int(ctx.Area.Height)
				if max < 0 {
					max = 0
				}
				if *v.Offset < max {
					(*v.Offset)++
				}
				return
			}
			if ev.Button == backend.MouseLeft {
				relY := int(ev.Y - ctx.Area.Y)
				if relY >= 0 && relY < visible {
					targetIdx := first + relY
					if item, ok := v.State.Row(targetIdx); ok {
						v.State.Select(item.ID)
						if v.OnSelect != nil {
							v.OnSelect(targetIdx, item)
						}
					}
				}
			}
		})
	}

	visualRow := 0
	for index := first; visualRow < visible; index++ {
		item, ok := v.State.Row(index)
		if !ok {
			break
		}
		line := v.State.GetCachedRowText(item, v.HorizontalOffset, v.StickyColumns)
		rowStyle := style
		if item.ID == v.State.Selected() {
			rowStyle = rowStyle.Merge(v.SelectedStyle)
			if ctx.IsFocused(v.ID) {
				rowStyle = rowStyle.Merge(v.FocusedStyle)
			}
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
		if perRowClick {
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

	separator := " | "
	var sb strings.Builder
	for i := 0; i < sticky; i++ {
		if i > 0 {
			sb.WriteString(separator)
		}
		sb.WriteString(row.Cells[i].Text)
	}
	prefix := sb.String()

	sb.Reset()
	for i := sticky; i < len(row.Cells); i++ {
		if i > sticky {
			sb.WriteString(separator)
		}
		sb.WriteString(row.Cells[i].Text)
	}
	rest := sb.String()

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
