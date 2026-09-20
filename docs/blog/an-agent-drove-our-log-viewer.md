# An AI agent drove our log viewer. Here is where it got stuck.

*Limoni apps have a semantic tree: every widget says what it is, what it's called and what state it's in.
Screen readers use it, our tests use it, and through a small MCP bridge an AI agent can use it too. We
pointed a headless Claude Code agent at zest, our log viewer, gave it a task, and read the transcript.*

---

## The setup

[zest](../../cmd/zest) was running on a generated log of 200,000 lines, still growing by about 80 lines a
second. It was built with `-tags limoni_debug`, which compiles in an automation socket; release builds
don't contain one. The agent ran headless with only the eight tools of
[`limoni-mcp`](../../cmd/limoni-mcp), the bridge from MCP to that socket:

```bash
claude -p "A terminal log viewer called zest is running. Use it to find the first line in the log \
that mentions a connection error, and tell me which service logged it and on which line number." \
  --mcp-config mcp.json --strict-mcp-config --allowedTools "mcp__limoni__*" --tools ""
```

`--tools ""` removes the shell and the file tools. The agent couldn't `grep` the log. It could only use
zest the way a person at the keyboard would, except that it reads the screen as a tree:

```
list#log "Log" position=0/202493 state=focused,busy bounds=0,1 120x28
  list-item "{\"time\":\"2026-09-19T10:04:58.840Z\",\"level\":\"info\",\"service\":\"api\",..." position=202466/202493
  ...
```

We knew the answer beforehand: the demo generator is seeded, and the first connection error is on line 16,
from the `api` service.

## What it did

It answered correctly: **line 16, the `api` service**. It took 35 tool calls and 33 seconds. The first
eight calls did the job:

1. `status`, `tree`, `screen`: look around. The log view is focused and following.
2. Press `/`, type `connection`, press Enter: filter the log.
3. Press `home`: go to the first match. The gutter says line 16.

Then it did something we hadn't asked for: before trusting the match, it cleared the filter to read lines
1 to 15, in case a differently worded connection error came first. That's the right instinct, and it's
where it got stuck.

## Where it got stuck

**Ctrl+U typed a "u".** To clear the filter the agent pressed Ctrl+U, the shortcut shells use to delete the
line. Limoni's text input treated every key with Ctrl held as a plain character and inserted it. That's a
bug for people too. Nobody had reported it, perhaps because nobody had tried.

**Twenty Backspaces.** It then deleted the filter one character at a time. In zest, Esc clears the filter,
and the footer says so. But the footer was drawn as plain text, which the semantic tree doesn't include,
and it was hidden while the filter was being edited. The agent had no way to know.

Neither problem was specific to AI. An agent is a user who reads the documentation you actually shipped,
which here meant the tree, and nothing else.

## What we changed

- **Text inputs understand the readline keys**: Ctrl+A and Ctrl+E move to the start and end, Ctrl+U and
  Ctrl+K delete to the start or end, Ctrl+W deletes a word. Any other Ctrl or Alt key is ignored instead of
  typed.
- **zest's status and key hints are widgets**, so they're in the tree (`generic#keys value=" / filter 1-6
  level ... q quit"`), and while you edit the filter they say `Enter apply · Esc clear · Ctrl+U delete`.

Both changes also help anyone using a screen reader.

## What made it work at all

- **Addressing by role and label, not coordinates.** The agent never computed where a line was on screen.
  It clicked `list-item` nodes by their text, and a layout change doesn't break that.
- **Tools that wait.** Input is asynchronous: a key is acknowledged before the frame it causes. So every
  input tool waits for the tree to change and settle, then returns the new tree. An agent never reasons
  about the screen as it was *before* its own keypress. When nothing changes, the tool says so and says why
  if it can: "the tree did not change within 500ms".
- **Idempotent clicks.** `click` with `ensure: "checked"` or `"selected"` only clicks if needed, so an agent
  that repeats a step doesn't undo it.
- **Nothing leaves unless the application allows it.** The socket is closed by default. Input, screen text
  and field values are separate opt-ins, a secret field's value is never sent whatever the policy says, and
  only the same user can connect. In an earlier run an agent typed a deploy token into a secret field of
  [a checklist app](../../examples/agent_checklist); the token appears nowhere in that run's transcript.

## Try it

```bash
go install github.com/thebanri/limoni/cmd/limoni-mcp@latest
go install -tags limoni_debug github.com/thebanri/limoni/cmd/zest@latest   # a build with the socket
zest -demo 200000 -socket "$XDG_RUNTIME_DIR/zest.sock"
claude mcp add limoni -- limoni-mcp -socket "$XDG_RUNTIME_DIR/zest.sock"
```

Then ask your agent something about the log. The same tree drives
[`uitest`](../automation.md), which lets you write Playwright-style tests for a terminal app. zest's own
end-to-end tests are written that way.
