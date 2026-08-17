package widgets

import (
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
)

// ToastLevel represents the severity and visual styling of a toast.
type ToastLevel uint8

const (
	ToastInfo ToastLevel = iota
	ToastSuccess
	ToastWarning
	ToastError
)

// ToastPosition determines which corner notifications stack in.
type ToastPosition uint8

const (
	ToastTopRight ToastPosition = iota
	ToastTopLeft
	ToastBottomRight
	ToastBottomLeft
)

// ToastItem is a single notification message.
type ToastItem struct {
	ID        string
	Title     string
	Message   string
	Level     ToastLevel
	CreatedAt time.Time
	Duration  time.Duration
	Dismissed bool
}

// ToastManager manages a stack of auto-dismissing toast notifications.
type ToastManager struct {
	Toasts     []*ToastItem
	Position   ToastPosition
	MaxVisible int
	nextID     int
}

// NewToastManager creates an initialized notification manager.
func NewToastManager(position ToastPosition) *ToastManager {
	return &ToastManager{
		Position:   position,
		MaxVisible: 4,
	}
}

// Show adds a new toast notification with custom duration.
func (tm *ToastManager) Show(title, message string, level ToastLevel, duration time.Duration) *ToastItem {
	if tm == nil {
		return nil
	}
	if duration <= 0 {
		duration = 4 * time.Second
	}
	tm.nextID++
	item := &ToastItem{
		ID:        fmt.Sprintf("toast_%d", tm.nextID),
		Title:     title,
		Message:   message,
		Level:     level,
		CreatedAt: time.Now(),
		Duration:  duration,
	}
	tm.Toasts = append(tm.Toasts, item)
	return item
}

// Info displays an informational notification.
func (tm *ToastManager) Info(title, message string) *ToastItem {
	return tm.Show(title, message, ToastInfo, 4*time.Second)
}

// Success displays a success notification.
func (tm *ToastManager) Success(title, message string) *ToastItem {
	return tm.Show(title, message, ToastSuccess, 4*time.Second)
}

// Warning displays a warning notification.
func (tm *ToastManager) Warning(title, message string) *ToastItem {
	return tm.Show(title, message, ToastWarning, 5*time.Second)
}

// Error displays an error notification.
func (tm *ToastManager) Error(title, message string) *ToastItem {
	return tm.Show(title, message, ToastError, 6*time.Second)
}

// Dismiss marks a toast as dismissed by ID.
func (tm *ToastManager) Dismiss(id string) {
	if tm == nil {
		return
	}
	for _, t := range tm.Toasts {
		if t.ID == id {
			t.Dismissed = true
			break
		}
	}
}

// Update cleans up expired and dismissed toasts.
func (tm *ToastManager) Update(now time.Time) {
	if tm == nil || len(tm.Toasts) == 0 {
		return
	}
	var active []*ToastItem
	for _, t := range tm.Toasts {
		if !t.Dismissed && now.Sub(t.CreatedAt) < t.Duration {
			active = append(active, t)
		}
	}
	tm.Toasts = active
}

// Draw renders the active stack of toasts into the buffer.
func (tm *ToastManager) Draw(ctx cell.Context, buf *buffer.Buffer) {
	if tm == nil || len(tm.Toasts) == 0 || ctx.Area.Width < 20 || ctx.Area.Height < 4 {
		return
	}

	maxVis := tm.MaxVisible
	if maxVis <= 0 {
		maxVis = 4
	}

	active := tm.Toasts
	if len(active) > maxVis {
		active = active[len(active)-maxVis:]
	}

	toastWidth := uint16(34)
	if toastWidth > ctx.Area.Width-4 {
		toastWidth = ctx.Area.Width - 4
	}

	toastHeight := uint16(3)

	for i, t := range active {
		var startX, startY uint16

		switch tm.Position {
		case ToastTopRight:
			startX = ctx.Area.X + ctx.Area.Width - toastWidth - 2
			startY = ctx.Area.Y + 1 + uint16(i)*(toastHeight+1)
		case ToastTopLeft:
			startX = ctx.Area.X + 2
			startY = ctx.Area.Y + 1 + uint16(i)*(toastHeight+1)
		case ToastBottomRight:
			startX = ctx.Area.X + ctx.Area.Width - toastWidth - 2
			totalH := uint16(len(active)) * (toastHeight + 1)
			startY = ctx.Area.Y + ctx.Area.Height - totalH + uint16(i)*(toastHeight+1)
		case ToastBottomLeft:
			startX = ctx.Area.X + 2
			totalH := uint16(len(active)) * (toastHeight + 1)
			startY = ctx.Area.Y + ctx.Area.Height - totalH + uint16(i)*(toastHeight+1)
		}

		if startY+toastHeight > ctx.Area.Y+ctx.Area.Height {
			continue
		}

		borderColor := cell.NewColorRGB(52, 152, 219)
		icon := "ℹ "
		switch t.Level {
		case ToastSuccess:
			borderColor = cell.NewColorRGB(46, 204, 113)
			icon = "✓ "
		case ToastWarning:
			borderColor = cell.NewColorRGB(241, 196, 15)
			icon = "⚠ "
		case ToastError:
			borderColor = cell.NewColorRGB(231, 76, 60)
			icon = "✕ "
		}

		toastArea := cell.NewRect(startX, startY, toastWidth, toastHeight)
		bgStyle := cell.Style{
			Bg: cell.NewColorRGB(24, 28, 38),
			Fg: cell.NewColorRGB(240, 245, 255),
		}
		borderStyle := cell.Style{
			Fg: borderColor,
			Bg: cell.NewColorRGB(24, 28, 38),
		}

		// Draw Drop Shadow
		DrawShadow(buf, toastArea, 2, 1)

		// Draw Toast Block Box
		block := Block{
			Borders:       BorderAll,
			BorderSymbols: SymbolsRounded,
			BorderStyle:   borderStyle,
			Style:         bgStyle,
		}
		block.Draw(cell.NewContext(toastArea, bgStyle), buf)

		// Draw Icon + Title
		titleStyle := cell.Style{Fg: borderColor, Bg: bgStyle.Bg, Modifier: cell.ModifierBold}
		buf.SetString(startX+1, startY+1, icon+t.Title, titleStyle)

		// Draw Message (if space allows)
		if t.Message != "" && toastHeight >= 3 {
			msgX := startX + 1 + uint16(utf8.RuneCountInString(icon+t.Title)) + 1
			if msgX < startX+toastWidth-2 {
				msgStyle := cell.Style{Fg: cell.NewColorRGB(180, 190, 205), Bg: bgStyle.Bg}
				buf.SetString(msgX, startY+1, "- "+t.Message, msgStyle)
			}
		}

		// Close button "[x]" on top-right
		closeStyle := cell.Style{Fg: cell.NewColorRGB(120, 130, 145), Bg: bgStyle.Bg}
		buf.SetString(startX+toastWidth-4, startY, "✕", closeStyle)

		if ctx.RegisterClick != nil {
			targetID := t.ID
			ctx.RegisterClick(toastArea, func() {
				tm.Dismiss(targetID)
			})
		}
	}
}
