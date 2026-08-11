package widgets

import (
	"context"
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
}

func (v VirtualDataView) Draw(ctx cell.Context, buf *buffer.Buffer) {
	if v.State == nil || v.Source == nil || ctx.Area.Width == 0 || ctx.Area.Height == 0 {
		return
	}
	visible := int(ctx.Area.Height)
	if err := v.State.Refresh(context.Background(), v.Source, v.First, visible, v.Prefetch); err != nil {
		buf.SetString(ctx.Area.X, ctx.Area.Y, "Error: "+err.Error(), ctx.Style.Merge(v.Style))
		return
	}
	style := ctx.Style.Merge(v.Style)
	for row := 0; row < visible; row++ {
		index := v.First + row
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
}

func (v VirtualDataView) SizeHint(max cell.Rect) (uint16, uint16) { return max.Width, max.Height }
