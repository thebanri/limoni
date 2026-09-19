# Semantic Automation

Every tool that automates a terminal application today — [termwright](https://github.com/fcoury/termwright), [mcp-tui-test](https://github.com/GeorgePearse/mcp-tui-test) — wraps the process in a pseudo-terminal and parses the rendered character grid. They have no choice: the application underneath has no semantics to offer. So a test asserts that some text sits at some coordinate, and breaks the moment the layout shifts by one column.

Limoni already builds a semantic tree every frame, for screen readers. `WithAutomation` serves that same tree on a Unix socket, which turns

```
assert text "Submit" at 42,7  →  click 42,7
```

into

```go
client.Click(automation.Selector{Role: "button", Label: "Submit"})
```

The selector survives relayout, resizing and restyling, because it never mentions where anything is drawn.

```go
// The application opts in, with an explicit policy. Build with -tags limoni_debug.
limoni.Run(draw, limoni.WithAutomation("/run/user/1000/myapp.sock", limoni.AutomationPolicy{
	AllowInput:   true, // off by default: without it a client can only observe
	ExposeScreen: true, // off by default: the grid holds every character on screen
}))
```

```go
// A test, or an agent, drives it.
client, _ := automation.Dial("/run/user/1000/myapp.sock")
defer client.Close()

list, _ := client.WaitFor(automation.Selector{Role: "list"}, 3*time.Second)
// list.Value == "beta", list.Position == 2, list.SetSize == 3

client.Key("down")
client.Type("hello")
client.Click(automation.Selector{Role: "button", Label: "Submit"})

screen, _ := client.Screen() // the raw grid, for assertions the tree cannot make
```

The protocol is newline-delimited JSON, so `socat` is a usable client when debugging. An ambiguous selector is an **error**, not a coin toss — a test that silently takes the first of two matching buttons passes for the wrong reason as soon as the second one appears; pass `Nth` to say which you meant.

## Driving it from an AI agent (MCP)

`cmd/limoni-mcp` puts the same tree in front of any agent that speaks the [Model Context Protocol](https://modelcontextprotocol.io) — Claude Code, Claude Desktop, Cursor and others. It is a bridge with no dependencies beyond the standard library: MCP over stdio on one side, the application's socket on the other.

```bash
go install github.com/thebanri/limoni/cmd/limoni-mcp@latest

# Try it on the example built for this:
# In one terminal — the app listens on $XDG_RUNTIME_DIR/limoni-checklist.sock:
go run -tags limoni_debug ./examples/agent_checklist
# In another:
claude mcp add limoni -- limoni-mcp -socket "$XDG_RUNTIME_DIR/limoni-checklist.sock"
```

The agent gets eight tools: `tree`, `find`, `click`, `press_key`, `type_text`, `wait_for`, `screen` and `status`. Two details matter in practice:

- **Input tools return the tree after the application redraws.** Input is asynchronous — the socket acknowledges a key before the frame it causes — so `click` waits for the tree to change and settle, and returns that. An agent never reasons about the screen as it was *before* its own click. If nothing changed, the result says so, and says why when it can tell: the field is secret, or the policy hides input values.
- **Failures are explanations.** An ambiguous selector, a misspelled argument or a policy refusal comes back as text the model can act on — `label="Remove" matches 2 nodes; set nth to choose one` — rather than as a protocol error.

The bridge changes none of the limits below: it can do only what the application's policy allows, and it opens no port. Input tools are annotated as destructive, so a client that asks before side effects will ask.

In one run, Claude Code 2.1.270, told only the goal, added a task, ticked three, typed a deploy token and deployed in 17 tool calls. The run's transcript contains no tool result with the token in it. That is one run, not a benchmark.

## Testing it like Playwright (`uitest`)

The same tree makes for tests that read like the ones Playwright writes for web pages. `uitest` finds widgets by role, label and ID, acts on them, and asserts with checks that **wait**: an action waits until its locator matches exactly one widget, and an assertion retries until it holds, so a test never sleeps. When a check does fail, the message shows what was expected, what was seen and the whole semantic tree of the last frame.

```go
func TestReleaseFlow(t *testing.T) {
	app := newChecklist()
	page := uitest.Run(t, 80, 24, app.draw) // in process: no terminal, no build tag

	page.GetByRole("input", "New task").Type("Tag v1.0")
	page.GetByRole("button", "Add task").Click()

	rows := page.GetByRole("list-item", "").Within(page.GetByRole("list", "Tasks"))
	page.Expect(rows).ToHaveCount(3)
	rows.Nth(-1).Click()
	page.Press("space")

	page.Expect(page.GetByID("status")).ToContainLabel("Added")
	page.Expect(page.GetByRole("dialog", "")).Not().ToBeVisible()
}
```

```
uitest: expected id="status" to have label "Deployed with 3 tasks complete.": got label "Blocked: the deploy token is empty." after 5s
last frame:
  input#new-task "New task" bounds=2,2 36x1
  button#add "Add task" bounds=40,2 14x1
  …
```

Every action is logged, so a failure arrives with the steps that led to it, and `uitest.WithSlowMo` paces a test for someone watching. Pointed at a running application with `uitest.Connect`, the same test drives it on screen — `TestLiveDemo` in [`examples/agent_checklist`](../examples/agent_checklist) does exactly that.

One API, three targets: `uitest.Run` for an immediate-mode draw function, `uitest.Program` for a declarative model running through its real message loop (commands included), and `uitest.Connect` for a running binary over its automation socket. [`examples/agent_checklist`](../examples/agent_checklist) is tested with it, and is the same application an agent drives through `limoni-mcp`.

Lists expose their visible rows as `list-item` children, so a row is addressed by its text rather than by counting key presses.

> [!WARNING]
> **This opens a control channel into a running process.** It is built in layers so that each one fails closed on its own.
>
> - **Absent from release builds.** The gateway only exists in binaries built with `-tags limoni_debug`. Without the tag, `WithAutomation` makes `Run` return `ErrAutomationNotCompiled`, and the socket server is not in the binary at all — CI builds a release binary and checks its symbol table for any automation code. No configuration mistake can switch on code that is not there.
> - **Closed by default.** The zero `AutomationPolicy` exposes structure only: roles, labels, positions, bounds. Input values, the screen snapshot and input synthesis each need their own field set.
> - **Secrets never leave, whatever the policy says.** `TextInput{Secret: true}` draws a mask glyph per character, so the secret never reaches the cell buffer, and its node carries no value and is marked sensitive. The gateway clears sensitive values again regardless, so a widget that forgets does not leak. Selectors are resolved against the redacted tree, so a client cannot guess a password and learn it from whether `value="…"` matched.
> - **Only your user can connect.** On Linux, macOS and FreeBSD the server asks the kernel who owns the connecting process and refuses any other user, in addition to the socket's 0600 permissions. Where the kernel cannot say — Windows among them — every connection is refused unless `AllowUnverifiedPeers` is set, because file permissions would be the only protection left.
> - **Unix socket only, no TCP option.** Deliberate and not configurable: a port would offer application control to anything that can reach the host.
>
> **What is left, stated plainly:** another process running *as the same user* can still connect — the operating system offers no stronger identity than the user for a local socket. And the gateway cannot know that a paragraph you render is secret unless you mark it; `ExposeScreen` sends whatever is on screen. Treat a `limoni_debug` binary as you would a debug console.
