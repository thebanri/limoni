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

## Structure beyond List

- **Rows, tabs, tree items are children too.** Table rows (`row`, labelled by
  the first cell, with a `cell` child per column), TreeView items (`tree-item`,
  with `StateExpanded`) and tabs (`tab-list`/`tab`, only when `Tabs.State` is
  set). Table and Tabs build their nodes *during Draw* — only Draw knows which
  filtered and sorted row lands on which screen row — into buffers on their
  state, so a node built without a Draw has no children. Table's `cellNodes` is
  sized *before* the row loop: rows keep sub-slices of it, and a later append
  that grows it would leave them pointing at the old array.
- **Nested widgets need `ctx.Describe`.** The frame only registered widgets it
  rendered itself, so a widget drawn as a Block's `Child` or through
  `AsComponent` was invisible — most of a real app. A container that draws a
  child itself calls `ctx.Describe(child, area)` after drawing it; Block and
  `WidgetAdapter` do. New containers must too.
- **Copy the whole Context.** Block once built its child's context field by
  field, so every Context field added later (click actions, wheel scrolling,
  Describe) silently never reached nested widgets. `childCtx := ctx`, then
  change Area/Style.
- **Anything on screen an agent must know belongs in the tree.** Text drawn
  with `SetString` is invisible to it. zest's "Esc clears" hint was plain text
  and hidden while typing; a real agent cleared a filter with twenty
  Backspaces. Draw status and key hints as widgets (`Paragraph{ID: "keys"}`).

## uitest, the parts added later

- `Locator.Check/Uncheck/Select` click only when needed and wait for the state;
  MCP `click` takes `ensure: checked|unchecked|selected`. Use them in any step
  that may run twice — `click` toggles.
- `Expect(...).ToContainValue(s)`: a Paragraph's text is its *value* (its label
  is "Text"), so `ToContainLabel` never matches it.
- `Locator.Node()` waits for a match, not for a change. After an action, assert
  with `Expect` (which retries); reading `Node().Value` straight away raced the
  counter template's redraw.
- `Page.ExpectExit()` waits for the app to quit. `Exited()` is immediate, and a
  declarative program quits through its message loop a moment after the key.
- Declarative mode: type with `page.Type`/`page.Press` into whatever the model
  focuses; `Locator.Type` clicks to focus, which Program mode does not route.

## Testing with a real agent: what one run taught

A headless run against zest (`--tools ""`, only `mcp__limoni__*`, seeded demo
log so the answer was known in advance: line 16, service `api`) answered
correctly in 35 calls, 33 s, $1.08. Its detours were real bugs: Ctrl+U was
typed as a "u" (TextInput inserted every Ctrl/Alt key — now ignored, readline
keys implemented) and the missing hints above. Read transcripts for detours,
not just the final answer. Summarise them from the stream-json with the tool
calls and the first line of each result.

## Harness traps (each cost a debugging session)

- **Drain the PTY continuously.** A harness that reads the app's output only
  between tool calls lets the PTY buffer fill; the app blocks in `write`, stops
  drawing, and every tool reports "the tree did not change" — it looks exactly
  like an app hang. Pump output on a thread. To tell which it is, redirect the
  app's stderr to a file and send SIGQUIT: a goroutine dump showing
  `Terminal.Draw → Backend.Write → syscall.write` is the harness.
- **`pkill -f <pattern>` kills your own shell** when the pattern appears in the
  command line that runs it. Stop processes by pid file.
- **Windows resets AF_UNIX connections closed with unread data.** The server's
  refusal message was lost because the client's request was still unread; it
  now half-closes and drains (bounded) before closing. Only the Windows CI
  runner shows this.
- **Ground truth first.** Compute the expected answer from a seeded source
  before running an agent, or a plausible wrong answer passes.

## Known gaps

- (Fixed) Custom widgets embedding `widgets.Accessible` are focusable now:
  `Accessible.WantsFocus` is true for an ID with an interactive role, and
  `Frame.RenderWidget` registers such a widget itself.
  `TestTabReachesAccessibleWidgets` in `testkit`.
- (Fixed in d2d4355, #47) Declarative mode routes mouse events to the frame's
  click regions (`Program.routeMouse`).
