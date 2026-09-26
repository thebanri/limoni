# Lemon Drop

Falling blocks that turn to sand.

```bash
go install github.com/thebanri/limoni/apps/lemondrop@latest

lemondrop            # play
lemondrop -fps 30    # fewer frames, for a slow terminal or link
lemondrop -mute      # no sound
lemondrop -board off # keep the scores on this machine
```

Or [play it in the browser](https://thebanri.github.io/limoni/?app=lemondrop):
the same code, compiled to WebAssembly and drawn by xterm.js, with sound
and the same leaderboard.

Pieces fall as in any falling-block game, in one of four colours: lemon,
lime, grapefruit and blueberry. When one lands it crumbles into sand of its
colour, grains of it in their own shades, and the sand runs down the pile:
it starts slowly and gathers speed as it falls, and slides down the sides
of a heap a little slower still, so a landed block visibly crumbles. A run of one colour that reaches from the
left wall to the right wall is the line: it flashes and is gone. Grains that
touch only at a corner still join. Whatever rested on it then falls, and if
that makes another run before the next piece lands, the second clear is a
chain and scores double, the third triple, and so on.

A clear scores ten points for each block's worth of grains in it, times the
level and the chain. Every four clears is a level, and pieces fall faster.
Dropping a piece with `space` or `↓` scores a little too.

| key | |
| :-- | :-- |
| `←` `→` or `A` `D` | move |
| `↑`, `W` or `X` | turn clockwise |
| `Z` | turn the other way |
| `↓` or `S` | soft drop |
| `space` | drop |
| `P` or `Esc`, or click `[ \|\| PAUSE ]` | pause, and go on (`R` restarts from the pause) |
| `M` | sound off and on |
| `N` | on the title: change your name |
| `Q` | quit, from the title, the pause or the end |

`Esc` never quits: in a run it pauses, so a run is not lost to a slip, and in
the browser's fullscreen it reaches the game too (hold it down to leave
fullscreen, in Chromium browsers). If the window shrinks under a run, as it
does leaving fullscreen, the board is redrawn with smaller blocks and the run
goes on.

In kitty, Ghostty, WezTerm and foot, which report key releases, held arrows
repeat at the game's own pace and `↓` drops fast until you let go. Elsewhere
the terminal's own key repeat does the repeating, and each `↓` drops the
piece a block.

## Leaderboard

The first time, the game asks for a name. Every run that ends goes on two
boards: the game's own best ten, kept with the name in
`limoni/lemondrop.json` under your configuration directory (or in the
browser's storage), and the world's, on the same server as Castle Lemonstein's
([apps/scoreboard](../scoreboard), at `/drop/scores`). The title and the end
show the world's board when it answers, the game's own when it does not, and
where the run just ended stands on it.

The game does not send its score. It sends what the run did: how long it
lasted, how many pieces it placed, the points from dropping them, and each
clear as its size and its place in a chain. The server works the score out
from that with the same rule, and turns away a run the game could not have
produced (a clear narrower than the board, a chain that skips, more sand
cleared than the pieces brought). Anyone could still send a run they did not
play; this keeps the board plausible, not honest.

`-board URL` points the game at another server, or `LEMONDROP_BOARD`;
`-board off` keeps the scores on this machine. The requests run off the
frame, and a slow or missing server never holds up a frame.

## How it is drawn

The board is ten blocks by eighteen, and a block is a square of grains, from
2×2 in an 80×24 window up to 6×6 in a large one. The size is chosen when a
run starts, from the window. A grain is one pixel, and a cell holds two
pixels as a half block (`▀`, upper pixel in the foreground, lower in the
background), so grains come out square.

Behind the board sits a faint half lemon, a slice seen from the cut face,
turning once every minute and a half. It is drawn once for each board size.
Turning it moves only the membranes between the segments, so only the flesh
is repainted, about 20 times a second, from each pixel's distance and angle
worked out once; each membrane edge is smoothed in halves, which keeps the
turning lemon to about 150 bytes a frame on the wire.

## Sound

Every sound is synthesised when the game starts, as in Castle Lemonstein, whose
mixer this is: moving and turning, a block landing and giving way, a drop,
the hiss of sand running (louder the more of it runs), a clear, a chain, a
new level, and the end. It is streamed to `pw-play`, `pacat`, `aplay` or
`sox`, whichever is installed, and in a browser to Web Audio. Without any of
them the game is silent and says so when it quits.

## Performance

A step and a frame allocate nothing. The board, the sand, the flood fill's
stack and the background all live in fixed arrays in one value made at
start-up, and the tests check it through Limoni's diff as well
(`TestFramesAllocateNothing`).

Each grain has a speed and how far it has got towards its next cell, one
byte each, which move with it. The sand only visits the rows that can hold a
loose grain: rows with one still moving or waiting after the last step, rows
a landed piece or a clear woke, and the row above one a grain has just left,
which is visited in the same pass. A settled pile costs nothing. A clear is looked
for only from the left wall, because a run that does not touch it cannot
reach across, and each grain is visited at most once.

On one core of a 2.1 GHz Xeon, at 100×40 with a board of loose sand:

| | time | allocations | sent to the terminal |
| :-- | --: | --: | --: |
| a frame: step, render and diff | ~65 µs | 0 | ~1.7 KB |
| the worst sand pass: every row of the largest board awake | ~40 µs | 0 | |

About half of a frame is Limoni's diff, which grows with the cells that
changed: sand that falls smoothly changes more of them, for longer.

```bash
go test -run '^$' -bench . -benchmem
```
