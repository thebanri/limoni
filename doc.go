// Package limoni is a terminal UI engine for Go: tests can click it, AI agents
// can drive it, and its render path makes no heap allocations.
//
// Every frame is drawn into a flat cell buffer and diffed against the previous
// one, so only the cells that changed reach the terminal. Layout, the diff and
// every built-in widget's drawing allocate nothing per frame, which keeps
// dashboards, log viewers and animations free of GC pauses and cheap over SSH.
//
// # Two application models
//
// Immediate mode, for dashboards, 3D viewers, games and animation:
//
//	limoni.Run(func(f *limoni.Frame, ev *limoni.Event) bool {
//		if ev != nil && ev.Type == limoni.EventKey && ev.Key.Type == limoni.KeyEsc {
//			return false // quit
//		}
//		f.RenderComponent(limoni.Border(
//			limoni.Center(limoni.Label("Hello from Limoni", limoni.Bold())),
//			limoni.SymbolsRounded,
//			limoni.Fg(limoni.Hex("#FFCC00")),
//		), f.Area())
//		return true
//	})
//
// The Elm architecture, for forms, wizards and asynchronous work: implement
// [Model] and call [RunProgram], or [NewProgram] when you own the terminal.
// Both models take a context ([RunWithContext]), and several [App] values can
// run in one process, one per SSH session.
//
// # Testing and AI agents
//
// Each frame also builds a semantic tree of roles, labels, values and states,
// the same tree a screen reader uses. The uitest package runs Playwright-style
// tests against it, addressing widgets by role and label instead of screen
// coordinates. The limoni-mcp command serves the tree to MCP clients such as
// Claude Code and Cursor, so an agent can read and operate a running app. The
// automation socket exists only in builds tagged limoni_debug.
//
// # What is included
//
// Tables and lists with virtual paging, text input, trees, tabs, charts,
// markdown, code highlighting, a canvas with Braille and sextant markers, a
// software 3D rasteriser (OBJ, STL, PLY, GLB), images over the kitty, iTerm2
// and Sixel protocols, UAX #29 grapheme clusters, OSC 8 hyperlinks, inline
// mode, and a Bubble Tea compatibility layer in compat/bubbletea. It runs on
// Linux, macOS, the BSDs, Windows and WebAssembly, and depends only on
// golang.org/x/sys and golang.org/x/crypto.
//
// Documentation, benchmarks and a browser playground:
// https://github.com/thebanri/limoni
package limoni
