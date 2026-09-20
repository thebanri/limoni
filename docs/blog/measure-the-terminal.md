# Ask the terminal, then measure it anyway

*How Limoni decides what your terminal can do, and why the answer to "how wide is 👨‍👩‍👧?" is 2, 6, or
"it depends on who you ask".*

---

A terminal UI library has to guess a lot about the terminal it draws into. Does it support 24-bit colour?
Can it repeat a character with `REP` instead of being sent it forty times? Will it present a frame
atomically, or tear it halfway? How many columns does it give a family emoji?

Most libraries guess from environment variables such as `TERM`, `COLORTERM` and `TERM_PROGRAM`. Limoni did
too. Those variables fall short in the places people actually work. Inside tmux, `TERM` is `screen` or
`tmux-256color` whatever runs outside it. Over SSH it's whatever the client sent. And plenty of terminals
don't set `TERM_PROGRAM` at all.

## Asking

When Limoni sets up the terminal it now sends a handful of queries along with the setup sequence:

| Query | Answer |
| :--- | :--- |
| `CSI > 0 q` (XTVERSION) | the terminal's name and version: `kitty(0.48.2)`, `tmux 3.5a` |
| `CSI ? 2026 $ p` (DECRQM) | whether synchronized output is supported |
| `CSI ? 2027 $ p` (DECRQM) | whether grapheme-cluster width mode is supported |
| `CSI ? u` | whether the Kitty keyboard protocol is available |
| `CSI c` (DA1) | device attributes. Sent last, because every terminal answers it and answers in order |

The last one is the useful trick. Once the reply to DA1 has arrived, every query sent before it has been
answered or never will be. So there's no timeout to tune, only a sentinel.

The replies come back on stdin, mixed in with the user's keystrokes. They need care: before this work,
Limoni's input parser read a Kitty keyboard reply (`CSI ? 0 u`) as a key press, and cut DA1 replies longer
than 32 bytes, so their tails arrived as typing. Replies are now parsed as their own event type, taken out
of the stream before the application sees input, and folded into a report the renderer checks with one
atomic load per frame.

## Measuring

Asking tells you what a terminal claims. We ran `limoni doctor`, which prints the report, in three terminals
on the same Linux machine:

| | kitty 0.48.2 | Alacritty | Konsole 26.08.1 |
| :--- | :--- | :--- | :--- |
| XTVERSION | ✔ | no answer | ✔ |
| mode 2026 (sync) | supported | supported | no answer |
| mode 2027 (clusters) | not supported | not supported | no answer |

By those answers no terminal here handles grapheme clusters, the characters built from several code points,
such as flags, skin-toned emoji and family emoji. The handshake also measures, though. It writes a ZWJ family
emoji at the start of a line, asks where the cursor ended up (`CSI 6 n`), then erases the cell and restores
the cursor before anything is drawn:

| | kitty | Alacritty | Konsole |
| :--- | :--- | :--- | :--- |
| 👨‍👩‍👧 drawn as | **2 columns** | **6 columns** | **2 columns** |
| `REP` repeats a glyph | yes | yes | yes |

kitty and Konsole draw the family as one two-column glyph without implementing mode 2027. Alacritty draws
all three people and advances six columns. A library that trusted mode 2027 would handle all three terminals
the same way, and be wrong for two of them.

This matters for every frame. Limoni stores a whole grapheme cluster in a cell. On a terminal that advances
by code point, the rest of the line would shift after each emoji, so the diff re-sends the cursor position
after every cluster. That costs a few bytes per emoji, and kitty and Konsole don't need it. With the
measurement, Limoni skips the re-anchoring where it has seen the terminal cope, and keeps it on Alacritty.

`REP` works the same way: send a space, then "repeat once", then read the cursor. Column 3 means `REP`
works; column 2 means it was ignored. A table of terminal names says which terminals *should* support
`REP`. The measurement says whether this one does, including a terminal the table has never heard of.

## When the answer is late

Over SSH the replies can arrive after the first frame has been drawn. That frame was encoded for the
guessed terminal: with `REP` it may not support, or with cursor positions that assume other widths. Updating
the profile and patching the next frame's diff wouldn't fix the cells already on screen. When the profile
changes after something has been drawn, Limoni repaints the whole screen. A test feeds late replies into a
running terminal and checks that the repaint happens, and that the repaint doesn't use `REP`.

One more detail: a reply that arrives after the application has exited ends up in your shell as
`^[[?62;22c`. So `Close` waits for the DA1 reply, for at most 150 ms, before restoring the terminal.

## Try it

```bash
go run github.com/thebanri/limoni/cmd/limoni@latest doctor
```

It prints what your terminal said, what was measured, and which features Limoni will use, both as guessed
from the environment and after the handshake. Please include it in any rendering bug report.
`LIMONI_PROBE=0` turns the handshake off.

The code is in [`core/driver/probe.go`](../../core/driver/probe.go), and the details are in
[drivers and platforms](../drivers-and-platforms.md#4-terminal-capability-handshake).
