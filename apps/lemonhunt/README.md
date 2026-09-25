# Lemon Hunt

A short first-person game in the terminal.

```bash
go install github.com/thebanri/limoni/apps/lemonhunt@latest

lemonhunt            # play
lemonhunt -fps 60    # smoother, where the terminal keeps up
lemonhunt -boss      # start at the lair's gate with ten lemons
lemonhunt -mute      # no sound
```

The rats have hidden the city's lemons in the sewer. Find ten and the gate to
the lair lifts. Ratatui, king of the rats, is waiting there with a health bar
of its own. It charges when its eyes burn and throws cheese you can dodge or
shoot out of the air. At two thirds and one third of its health, it calls
for more rats. Each lemon gives back 10 HP.

The title screen has a menu: `W`/`S` choose, `←`/`→` set the volume, and
`ENTER` starts, switches the sound on and off, or quits.

In the game, `W`/`S` walk, `A`/`D` strafe, `←`/`→` turn, and `space` squirts.
Hold it down to keep squirting. `M` shows the map and `Esc` quits. After a
win or a loss, `R` plays again.

The window must be at least 60×20. A whole run takes a few minutes.

It also runs in the browser, at <https://thebanri.github.io/limoni/?app=lemonhunt>:
the same code compiled with `GOOS=js GOARCH=wasm`, drawn by xterm.js, and
built from `main` by the Pages workflow. xterm.js reports no key releases,
so the page sends them itself from the browser's `keyup`, in the kitty
protocol's spelling, and keys are held exactly there too (see below).

## How it draws

It is a raycaster drawn in half blocks. Every terminal column casts one ray,
and every cell holds two pixels: `▀`, with the upper pixel as the foreground
colour and the lower as the background. A cell is about twice as tall as it
is wide, so the pixels come out square.

- **Walls, floor and ceiling** use 32×32 textures generated at start-up:
  brick, stone with moss, rusting pipes, and the lair's glowing terminal
  panels. Slime runs down the stone walls; puddles on the floor catch the
  light and ripple.
- **Light** comes from lamps that flicker and sometimes nearly go out, from
  a lantern that moves with the player, and from the squirter's flash. Fog
  swallows whatever is far away.
- **Rats and Ratatui** are pixel art drawn in code at start-up, not loaded
  from files. Rats have a four-frame walk cycle and a lunge, and face the way
  they are going. Ratatui breathes, and its eyes burn when it is about to
  charge.
- **The lemons are solid.** Each is half a lemon, cut across, turning in the
  air above its shadow. Every pixel it might cover casts a ray at a half
  ellipsoid and shades what it hits. The dome is dimpled rind that turns
  green towards the stalk. The cut face has nine segments divided by
  membranes, the white pith and a few seeds.
- **Particles**: juice, splashes, drips from the ceiling, and sparkles when
  you find a lemon. Also screen shake, head bob, a sway to the squirter, and
  tints when you are bitten or find a lemon.

The rats find you on a flow field: a breadth-first search from your tile,
rebuilt whenever you step onto a new one. Each rat walks downhill on it,
which takes them round corners and gets Ratatui past its pillars.

## Keys that stay down

Most terminals report key presses and auto-repeats, but not releases, so a
game cannot know a key is still held. It can only guess from auto-repeat,
which starts after a pause (600 ms on many desktops), and walking stutters.

Lemon Hunt asks for key releases with `limoni.WithKeyReleases()`. In
terminals with the kitty keyboard protocol (kitty, Ghostty, WezTerm, foot),
a key is then held exactly as long as it is held. Elsewhere it falls back to
guessing: a press counts as held long enough to bridge the auto-repeat
pause. The first release proves the terminal sends them, and the game
switches over by itself.

## Sound

Every sound is synthesised at start-up, from sine and square waves and
filtered noise: the squirt, splashes, rats squeaking and dying, bites,
lemons, the gate's chain, Ratatui's roar, footsteps, drips. There are no
sample files.

Sound is mixed in stereo and streamed as raw PCM to whichever player the
system has: `pw-play` (PipeWire), `pacat` (PulseAudio), `aplay` (ALSA) or
`play` (sox). Sounds fade with distance and are panned to the side they come
from. The mixer keeps only about 60 ms ahead of the clock, or a pipe's worth
of audio would queue up and every effect would arrive late. Without a
player, the game is silent and says so on exit.

In a browser the player is Web Audio (`sound_js.go`). The page makes an
`AudioContext` on the click that starts the game, since browsers start sound
only from a user gesture. Each clip is copied into it once, and a sound is a
buffer source through a gain and a stereo panner.

## Zero allocations

A frame allocates nothing, and the tests hold it to that through Limoni's
diff as well. `TestFramesAllocateNothing` plays scripted input through six
scenes (title, sewer with and without key releases, lair, win, loss). Each
frame is drawn and then encoded as the terminal would receive it, and the
test measures the whole thing with `testing.AllocsPerRun`.

What that takes:

- The game's state and every scratch buffer live in one value made at
  start-up: the rays, the framebuffer, the sprite order, the particles, the
  flow field and its queue. It draws a window of up to 512×200 cells.
- Nothing on screen is bold or italic. The diff builds a style through its
  cache only when a modifier has to be switched off, and with a picture of
  free colours that cache would fill with styles never seen again.
  `TestNoCellCarriesAModifier` guards it.
- Asking for a sound hands a small value to the mixer's channel and never
  waits. If the queue is full, the sound is dropped, not the frame.
- Numbers on the HUD are written digit by digit rather than with `strconv`,
  and sprites are sorted with an insertion sort rather than `sort.Slice`.

```bash
go test -run '^$' -bench . -benchmem
```

On an AMD Ryzen 5 5600, at 160×48 cells (160×92 pixels):

| | time | allocs | bytes sent |
| :-- | --: | --: | --: |
| `BenchmarkFrame` (step + draw) | 0.39 ms | 0 | — |
| `BenchmarkFrameWithDiff` (+ ANSI diff) | 0.60 ms | 0 | ≈ 78 KB a frame |

A moving picture in truecolor is expensive to send: about 2.3 MB/s at 30
frames a second. That is nothing for a local terminal and a lot over a slow
SSH link. Colours are rounded to multiples of eight, which saves a quarter
of the bytes with no visible difference; rounding to sixteen saves half, but
bands in the dark.

## Why a module of its own

Like `apps/globe`, it has its own `go.mod`, so none of it reaches anyone who
imports Limoni. It depends on a published Limoni and uses only its public
API.
