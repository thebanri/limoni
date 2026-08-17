package widgets

import (
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/thebanri/limoni/core/backend"
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
	Toasts        []*ToastItem
	Position      ToastPosition
	MaxVisible    int
	nextID        int
	renderedRects []toastRect
}

type toastRect struct {
	id   string
	rect cell.Rect
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
	if tm == nil {
		return
	}
	tm.renderedRects = tm.renderedRects[:0]
	if len(tm.Toasts) == 0 || ctx.Area.Width < 24 || ctx.Area.Height < 6 {
		return
	}

	now := time.Now()
	maxVis := tm.MaxVisible
	if maxVis <= 0 {
		maxVis = 4
	}

	active := tm.Toasts
	if len(active) > maxVis {
		active = active[len(active)-maxVis:]
	}

	toastWidth := uint16(38)
	if toastWidth > ctx.Area.Width-8 {
		toastWidth = ctx.Area.Width - 8
	}

	toastHeight := uint16(4)
	gap := uint16(1)

	for i, t := range active {
		var startX, startY uint16

		switch tm.Position {
		case ToastTopRight:
			startX = ctx.Area.X + ctx.Area.Width - toastWidth - 4
			startY = ctx.Area.Y + 2 + uint16(i)*(toastHeight+gap)
		case ToastTopLeft:
			startX = ctx.Area.X + 3
			startY = ctx.Area.Y + 2 + uint16(i)*(toastHeight+gap)
		case ToastBottomRight:
			startX = ctx.Area.X + ctx.Area.Width - toastWidth - 4
			totalH := uint16(len(active)) * (toastHeight + gap)
			startY = ctx.Area.Y + ctx.Area.Height - totalH - 2 + uint16(i)*(toastHeight+gap)
		case ToastBottomLeft:
			startX = ctx.Area.X + 3
			totalH := uint16(len(active)) * (toastHeight + gap)
			startY = ctx.Area.Y + ctx.Area.Height - totalH - 2 + uint16(i)*(toastHeight+gap)
		}

		if startY+toastHeight > ctx.Area.Y+ctx.Area.Height-1 {
			continue
		}

		borderColor := cell.NewColorRGB(52, 152, 219)
		badgeBg := cell.NewColorRGB(20, 45, 70)
		levelText := " ℹ INFO "
		switch t.Level {
		case ToastSuccess:
			borderColor = cell.NewColorRGB(46, 204, 113)
			badgeBg = cell.NewColorRGB(15, 50, 30)
			levelText = " ✓ SUCCESS "
		case ToastWarning:
			borderColor = cell.NewColorRGB(241, 196, 15)
			badgeBg = cell.NewColorRGB(60, 50, 15)
			levelText = " ⚠ WARNING "
		case ToastError:
			borderColor = cell.NewColorRGB(231, 76, 60)
			badgeBg = cell.NewColorRGB(65, 20, 20)
			levelText = " ✕ ERROR "
		}

		toastArea := cell.NewRect(startX, startY, toastWidth, toastHeight)
		bgStyle := cell.Style{
			Bg: cell.NewColorRGB(18, 22, 32),
			Fg: cell.NewColorRGB(240, 245, 255),
		}
		borderStyle := cell.Style{
			Fg: borderColor,
			Bg: cell.NewColorRGB(18, 22, 32),
		}

		// Draw Drop Shadow (ensure within buffer bounds)
		if startX+toastWidth+2 <= ctx.Area.X+ctx.Area.Width {
			DrawShadow(buf, toastArea, 2, 1)
		}

		// Draw Toast Block Box
		block := Block{
			Borders:       BorderAll,
			BorderSymbols: SymbolsRounded,
			BorderStyle:   borderStyle,
			Style:         bgStyle,
		}
		block.Draw(cell.NewContext(toastArea, bgStyle), buf)

		// Header Badge on Top Border
		buf.SetString(startX+2, startY, levelText, cell.Style{Fg: borderColor, Bg: badgeBg, Modifier: cell.ModifierBold})

		// Close button "[✕]" on top-right
		closeStyle := cell.Style{Fg: cell.NewColorRGB(160, 170, 185), Bg: bgStyle.Bg}
		buf.SetString(startX+toastWidth-3, startY, "✕", closeStyle)

		// Title Line
		titleStyle := cell.Style{Fg: cell.NewColorRGB(255, 255, 255), Bg: bgStyle.Bg, Modifier: cell.ModifierBold}
		titleText := t.Title
		maxTextW := int(toastWidth) - 4
		if utf8.RuneCountInString(titleText) > maxTextW {
			runes := []rune(titleText)
			titleText = string(runes[:maxTextW-1]) + "…"
		}
		buf.SetString(startX+2, startY+1, titleText, titleStyle)

		// Message Line
		msgStyle := cell.Style{Fg: cell.NewColorRGB(160, 175, 195), Bg: bgStyle.Bg}
		msgText := t.Message
		if utf8.RuneCountInString(msgText) > maxTextW {
			runes := []rune(msgText)
			msgText = string(runes[:maxTextW-1]) + "…"
		}
		buf.SetString(startX+2, startY+2, msgText, msgStyle)

		// Progress / Remaining Time Indicator on Bottom Border
		if t.Duration > 0 {
			elapsed := now.Sub(t.CreatedAt)
			remainRatio := 1.0 - (float64(elapsed) / float64(t.Duration))
			if remainRatio < 0 {
				remainRatio = 0
			}
			barWidth := int(float64(toastWidth-4) * remainRatio)
			if barWidth > int(toastWidth-4) {
				barWidth = int(toastWidth - 4)
			}
			for bx := 0; bx < barWidth; bx++ {
				buf.SetCell(startX+2+uint16(bx), startY+toastHeight-1, cell.Cell{
					Content: '━',
					Style:   cell.Style{Fg: borderColor, Bg: bgStyle.Bg},
				})
			}
		}

		tm.renderedRects = append(tm.renderedRects, toastRect{
			id:   t.ID,
			rect: toastArea,
		})

		if ctx.RegisterClick != nil {
			targetID := t.ID
			ctx.RegisterClick(toastArea, func() {
				tm.Dismiss(targetID)
			})
		}
	}
}

// HandleMouse processes mouse clicks on toast notification cards and close buttons.
func (tm *ToastManager) HandleMouse(m backend.MouseEvent) bool {
	if tm == nil || len(tm.Toasts) == 0 {
		return false
	}
	if m.Button == backend.MouseLeft {
		for _, r := range tm.renderedRects {
			if m.X >= r.rect.X && m.X < r.rect.X+r.rect.Width &&
				m.Y >= r.rect.Y && m.Y < r.rect.Y+r.rect.Height {
				tm.Dismiss(r.id)
				return true
			}
		}
	}
	return false
}

// SizeHint returns the preferred dimensions for ToastManager.
func (tm *ToastManager) SizeHint(maxArea cell.Rect) (uint16, uint16) {
	return maxArea.Width, maxArea.Height
}
