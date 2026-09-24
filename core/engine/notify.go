package engine

import "context"

// notifyMsg asks the terminal for a desktop notification. The runtime takes
// it before Update: the model never sees it.
type notifyMsg struct{ title, body string }

// NotifyCmd shows a desktop notification, where the terminal can: kitty,
// iTerm2, WezTerm, Ghostty and foot, or any terminal named with LIMONI_NOTIFY.
// Elsewhere it does nothing. See terminal.Terminal.Notify.
func NotifyCmd(title, body string) Cmd {
	return func(context.Context) Msg { return notifyMsg{title, body} }
}

// Notifications are the notifications NotifyCmd asked for, for whoever
// writes to the terminal. RunTerminal reads them; a caller driving Draw
// itself should too, and pass each to terminal.Terminal.Notify.
func (p *Program) Notifications() <-chan Notification { return p.notifications }

// Notification is one request from NotifyCmd.
type Notification struct{ Title, Body string }

// takeNotify hands a notifyMsg to the terminal side, dropping it if nobody
// has read the last few: a notification is not worth blocking Update for.
func (p *Program) takeNotify(message Msg) bool {
	n, ok := message.(notifyMsg)
	if !ok {
		return false
	}
	select {
	case p.notifications <- Notification{Title: n.title, Body: n.body}:
	default:
	}
	return true
}
