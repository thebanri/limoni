# Session Recording & Replay

A bug report that says *"I pressed some keys and it crashed"* is hard to act on. A recording of the session is not. The `session` package records a declarative application — every message its model received, in the order `Update` saw them, plus the semantic tree of every frame — and replays it against the model, verifying each frame.

```go
rec, _ := session.Create("bug.limoni", model, width, height, session.Policy{})
defer rec.Close()
limoni.RunProgram(ctx, model, limoni.WithProgramObserver(rec))
```

```go
// Later, in a test: the recording becomes a regression test.
report, err := session.Replay("bug.limoni", func() limoni.Model { return newModel() }, session.ReplayOptions{})
if err != nil { t.Fatal(err) }          // the recording could not be trusted
if !report.Verified() { t.Fatal(report.Divergence) } // "replay diverged at step 3 ..."
```

A replay reports the **first step whose frame differs**, and whether a recorded **crash reproduces** — so the same file tells you the bug is still there, and later that it is fixed.

It records at `Update` rather than at the terminal, which is what makes it replayable: timers and commands race in a live run, and recording where `Update` is called freezes how those races came out. A command's effect enters the recording as the message it produced, so a replay never re-sends a request or reads the clock.

> [!WARNING]
> **Limits, stated plainly.**
>
> - **Declarative mode only.** Immediate-mode `Run` has no boundary between the application and its side effects, so there is nothing to record at.
> - **`Update` and `View` must be deterministic.** A model that calls `time.Now` or `math/rand` inside them diverges on replay — the replay detects it and names the step, but cannot fix it. Receive the clock as a message with `limoni.NowCmd`. [`tools/limonivet`](../tools/limonivet) reports such calls; CI runs it on this repository.
> - **Registered messages only.** An application message type is recorded by name only unless registered with `session.Register`, and a replay that reaches one fails loudly rather than skipping it.
>
> **Privacy is closed by default.** Typed text and pastes are written as `x` unless `RecordText` is set; input-field values are dropped from recorded trees unless `ExposeInputValues` is set; sensitive fields are always dropped. Even with `RecordText`, characters are redacted whenever a secret field might have focus — including the window after *any* message, before the next frame shows where focus went, because a command result can move focus into a password field as easily as Tab can. Files are `0600`, never overwritten, checksummed, and refused on replay if modified.
>
> **What is left:** a message type you register is recorded in full — use `session.RegisterRedacted` for one that carries a secret. Labels are recorded even when values are not, so a widget whose *label* shows a secret puts it in the file. And a recording is a file on disk: treat it like a log that may contain what your users saw.
