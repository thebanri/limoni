---
name: limoni-agent-surface
description: How Limoni exposes applications to AI agents and tests — the semantic tree, the automation socket, cmd/limoni-mcp, and the uitest package. Load before touching core/accessibility, automation/, cmd/limoni-mcp, uitest/, a widget's AccessibilityNode, or anything about agents driving or testing a TUI.
---

# The agent surface: semantic tree → socket → MCP → tests

One tree serves four consumers: screen readers, the automation socket, AI agents
through MCP, and `uitest`. Adding a semantic node helps all four; breaking the
tree breaks all four.

```
widget (accessibility.Provider)
  → Frame.RegisterAccessibility          f.Accessibility, valid for THIS frame only
  → Frame.AccessibilityTree()            deep copy — the only safe way to keep it
  → gateway publish (limoni_debug only)  automation.Snapshot{Tree, Screen, Focused, Injector}
  → automation server                    redacts, resolves selectors, injects input
  → cmd/limoni-mcp                       MCP tools over stdio for Claude Code etc.
  → uitest                               locators + waiting assertions (same selectors)
```

## Invariants that must not regress

- **Secrets never leave.** `TextInput{Secret: true}` draws a mask glyph, so the
  text never reaches the cell buffer; its node is `StateSensitive` with no value;
  `accessibility.Redact` clears sensitive values again whatever the policy says.
- **Selectors resolve against the redacted tree.** Resolving against the raw one
  turns a withheld value into an oracle: `find value="hunter2"` would answer.
- **Policy is closed by default.** The zero `AutomationPolicy` exposes structure
  only. `AllowInput`, `ExposeScreen`, `ExposeInputValues` are separate opt-ins.
- **Peer verification.** Linux/macOS/FreeBSD ask the kernel who owns the
  connecting process; elsewhere connections are refused unless
  `AllowUnverifiedPeers`. Tests must set it off those three platforms.
- **Nothing in release binaries.** The gateway exists only under
  `-tags limoni_debug`. CI greps `go list -deps` and the symbol table of built
  `examples/simple` and `examples/agent_checklist` for `limoni/(automation|session)`.
- **Zero allocations on the draw path**, including node construction —
  `widgets.TestAccessibilityNodeConstructionDoesNotAllocate` enforces it.

## Traps already paid for

- **`f.Accessibility` aliases reused buffers.** `List` writes its row nodes into
  a buffer owned by `ListState` to stay allocation-free, so the slice is only
  valid until the next draw. `Frame.AccessibilityTree` deep-copies; anything
  keeping a tree (gateway, session recording, testkit, tests) must go through it.
- **`FocusManager.Clear` runs before the draw function.** An immediate-mode app
  handles its event at the top of the callback, where the focusable list is
  empty — Tab did nothing in every `limoni.Run` app. Navigation now falls back to
  the previous frame's registrations (`lastFocusable`, `lastBounds`).
- **`Server.Close` must not wait for clients.** It closes open connections;
  otherwise an app hangs on exit while an agent or test is attached.
- **Input is asynchronous.** The socket acknowledges a key before the frame it
  causes. A tool that returns immediately shows the agent the screen *before*
  its own click. `app.act` waits for the tree to change, then to hold still for
  50ms, bounded by `-settle`.
- **"Nothing changed" needs a reason.** Typing into a secret field or with
  `ExposeInputValues` off looks identical to a lost keystroke; the tool says
  which, or the agent types the same text again.
- **Input that ends the app is success.** Esc or a Quit button closes the socket
  right after the input lands; report it as done, not as a failed call.
- **Retry reads, never input.** A dropped connection after a write may mean the
  click already landed. Read-only calls retry once (the app may have restarted).
- **Reject unknown tool arguments.** A misspelled selector field would otherwise
  vanish and silently widen the selector.

## MCP specifics (cmd/limoni-mcp, stdlib only, no SDK)

- Line-delimited JSON-RPC 2.0 on stdio: `initialize`, `ping`, `tools/list`,
  `tools/call`, `notifications/cancelled`. Versions answered: 2025-11-25,
  2025-06-18, 2025-03-26, 2024-11-05; unknown → newest.
- Batches (`[`) were removed from MCP in 2025-06-18 — refuse with -32600.
- A cancelled request gets **no** response. Notifications get no response.
- Tool failures are `isError` text results the model can act on, not JSON-RPC
  errors. Unknown tool name is -32602.
- `defer wg.Wait()` must be registered **before** `defer cancel()`, or closing
  stdin waits out a minute-long `wait_for`.
- Input tools carry `destructiveHint: true`; reads `readOnlyHint: true`.

## Testing recipe

1. **Unit** — run a real `automation.Listen` with a fake app that applies input
   on a later frame (one event per frame, in order), and drive the bridge over
   `io.Pipe`. See `cmd/limoni-mcp/main_test.go`.
2. **Mutation** — break the thing the test claims to protect and watch it fail:
   cancellation skip, settling, reconnect, deep copy, `Not()`, focus wait.
3. **PTY** — build with `-tags limoni_debug` and run under a real pty; scripted
   JSON-RPC against the built binary. Unit tests missed the Tab bug; this found it.
4. **A real agent** — headless run, tools restricted:

```bash
claude -p "<goal>" --mcp-config mcp.json --strict-mcp-config \
  --allowedTools "mcp__limoni__*" --output-format stream-json --verbose
# add --tools "" to prove the agent used no shell or file access
```

Sockets: `sockaddr_un` caps the path at ~103 bytes. The scratchpad path is too
long — use `$XDG_RUNTIME_DIR` (also where a real app's socket belongs).

## uitest, in one screen

```go
page := uitest.Run(t, 80, 24, app.draw)            // immediate mode, in process
page := uitest.Program(t, 80, 24, &model{})        // declarative, real message loop
page := uitest.Connect(t, socket)                  // running binary, over the socket

page.GetByRole("button", "Add task").Click()
page.GetByID("name").Type("x")                     // clicks to focus first, waits for focus
rows := page.GetByRole("list-item", "").Within(page.GetByRole("list", "Tasks"))
page.Expect(rows.Nth(-1)).ToContainLabel("[x]")
page.Expect(page.GetByID("status")).Not().ToBeVisible()
```

Actions wait for exactly one match; assertions retry; failures print the last
frame's tree. `WithTimeout`, `WithSlowMo` (demo pacing). Every action is logged
with `t.Logf`, without the typed text — it may be a password.

`TestLiveDemo` in `examples/agent_checklist` drives a visible app over the
socket for recording:

```bash
go run -tags limoni_debug ./examples/agent_checklist          # terminal 1
LIMONI_DEMO_SOCKET=$XDG_RUNTIME_DIR/limoni-checklist.sock \
  go test -v -run TestLiveDemo ./examples/agent_checklist     # terminal 2
```

## Known gaps

- Table rows (`row` + `cell`), TreeView items and Tabs (`tab-list`/`tab`, only
  with `Tabs.State`) are now children. Table and Tabs build their nodes *during
  Draw* — only Draw knows which filtered/sorted row is on which screen row — so
  a node built without a Draw has no children. Table's `cellNodes` is sized
  before the row loop because rows hold sub-slices of it.
- Custom widgets embedding `widgets.Accessible` are not focusable, so Tab skips
  them (the example's buttons are click-only).
- Declarative mode does not route clicks to frame click handlers, so
  `Locator.Type`'s click-to-focus cannot work there; focus with keys instead.
- `click` toggles. `Locator.Check/Uncheck/Select` and MCP `click` with
  `ensure` are the idempotent forms; use them in any step that may be repeated.
