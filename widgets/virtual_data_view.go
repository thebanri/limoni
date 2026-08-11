package widgets

import (
	"context"
	"github.com/thebanri/limoni/core/backend"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
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
	for row := 0; row < visible; row++ {
		index := first + row
		item, ok := v.State.Row(index)
		if !ok {
			continue
		}
		line := item.Text
		if line == "" {
			line = string(item.ID)
		}
		rowStyle := style
		if item.ID == v.State.Selected() {
			rowStyle = rowStyle.Merge(v.SelectedStyle)
		}
		buf.SetString(ctx.Area.X, ctx.Area.Y+uint16(row), line, rowStyle)
		if ctx.RegisterClick != nil {
			id := item.ID
			ctx.RegisterClick(cell.NewRect(ctx.Area.X, ctx.Area.Y+uint16(row), ctx.Area.Width, 1), func() { v.State.Select(id) })
		}
	}
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
}

func fallback(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

func (v VirtualDataView) SizeHint(max cell.Rect) (uint16, uint16) { return max.Width, max.Height }
