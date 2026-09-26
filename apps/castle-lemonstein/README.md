# Castle Lemonstein

Storm a moonlit castle in this retro first-person terminal game.

A stone fortress, burgundy banners and a gold two-line title welcome you.
Reclaim ten stolen lemons, raise the portcullis, and defeat Ratatui in the keep.

```bash
go install github.com/thebanri/limoni/apps/castle-lemonstein@latest

castle-lemonstein            # play
castle-lemonstein -fps 60    # smoother, where the terminal keeps up
castle-lemonstein -boss      # start at the lair's gate with ten lemons
castle-lemonstein -mute      # no sound
```

The rats have hidden the city's lemons in the castle. Find ten and the gate to
the lair lifts. Ratatui, king of the rats, is waiting there with a health bar
of its own. It charges when its eyes burn and throws cheese you can dodge or
shoot out of the air. At three quarters, half and a quarter of its health, it
calls for more rats, and each third it loses makes it faster, quicker to
charge and freer with the cheese: one piece, then three, then five. Each
lemon gives back 8 HP.

The squirter holds eight squirts. It never runs out of juice, but when it is
empty it has to be refilled with `R`, which takes a second and a half with
no squirting; the gun drops out of sight while it happens. The squirt only
hits what the crosshair is on: about the width of a rat's body.

The first time, the game asks for a name. The title screen has a menu:
`W`/`S` choose ENTER CASTLE, KNIGHT, SOUND or QUIT; `←`/`→` set the volume,
and `ENTER` starts, changes the name,
switches the sound on and off, or quits.

In the game, `W`/`S` walk, `A`/`D` strafe, `←`/`→` turn, and `space` squirts.
Hold it down to keep squirting. `R` reloads, `M` hides and shows the map, and
`Esc` quits. After a win or a loss, `R` plays again.

## Score and leaderboard

A run is scored on aim and time:

| | points |
| :-- | --: |
| aim | 5000 × hits ÷ squirts |
| time, for a win | 10 for every second under ten minutes |
| each rat | 50 |
| each lemon | 100 |

So a quick, careful win beats a slow one that sprayed the walls, and a loss
still scores what it got done. The end screen takes the score apart and
shows the best five; the title shows them too.

The name and the ten best runs are kept between games: in
`~/.config/castle-lemonstein/scores.json` (the user's config directory on other
systems), or, in the browser, in `localStorage`. Existing names and scores
from the former `lemonhunt` save location are read automatically until the
first save under the new name.

With a scoreboard ([`apps/scoreboard`](../scoreboard), on Vercel's free plan with a free Neon database),
every finished run is also sent there, and the screens show the world's
best ten — `WORLD BEST` — and where the run placed in it. The server works
the score out from what the run did, and the game asks it off the frame, so
a slow or missing server never stalls one: without an answer the screens
fall back to this player's own board and say so. `-board URL` (or
`CASTLE_LEMONSTEIN_BOARD`) points the game at a server, `-board off` keeps the
scores at home; in the browser, the page names it, and `?board=URL`
overrides it.

The window must be at least 60×20 terminal cells. The picture grows with
the terminal up to 512 columns × 200 picture rows, plus two rows for the HUD:
a 512×202 terminal shows the maximum 512×400 half-block pixels. Larger
terminals leave unused picture space black. For any terminal size, the
picture is `min(columns, 512) × (2 × min(rows - 2, 200))` pixels. Use
`stty size` to see the current terminal size (rows first, then columns).
Larger pictures take more work to render and send to the terminal; the
playable frame rate depends on the machine and terminal. A whole run takes
a few minutes.

It also runs in the browser, at <https://thebanri.github.io/limoni/?app=castle-lemonstein>:
old `?app=lemonhunt` links automatically switch to this address and open the
same game, preserving other URL options.
It uses
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
  blue-grey masonry, weathered stone, burgundy lemon standards, and the
  throne room's dark walls. Rainwater streaks the stone; puddles on the floor catch the
  light and ripple.
- **Light** comes from lamps that flicker and sometimes nearly go out, from
  a lantern that moves with the player, and from the squirter's flash. Fog
  swallows whatever is far away.
- **Rats and Ratatui** are pixel art drawn in code at start-up, not loaded
  from files. The rats walk upright, a torn waistcoat on the common ones and
  an apron on the fat, and stand most of the way to the player's eye, so one
  close enough to bite is still in sight above the gun. They have a
  four-frame walk cycle and a lunge with the claws up. Ratatui breathes, and
  its eyes burn when it is about to charge.
- **The map** is a round window at the top right, turning with the player so
  that ahead is always up: walls by their stone, the view ahead lit faintly,
  lemons, rats and Ratatui as dots. It is open from the start.
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

Castle Lemonstein asks for key releases with `limoni.WithKeyReleases()`. In
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

Sound is mixed in stereo. On macOS it plays directly through the system's
AudioToolbox using [Oto](https://github.com/ebitengine/oto); no extra player
installation is needed, on either Intel or Apple Silicon. If native audio
cannot start, the game also tries the external players below; check the
macOS sound output or install SoX (`brew install sox`) as a fallback.

Windows also uses Oto to play directly through the system audio output,
without installing an external player. If sound cannot start, check the
selected output device and the game's volume in the Windows volume mixer.

On other native platforms, raw PCM is streamed to whichever player the
system has: `pw-play` (PipeWire), `pacat` (PulseAudio), `aplay` (ALSA) or
`play` (sox). Sounds fade with distance and are panned to the side they come
from. The mixer keeps only about 60 ms ahead of the clock, or a pipe's worth
of audio would queue up and every effect would arrive late. Without a
working audio output, the game is silent and shows `audio unavailable`.
This refers to sound playback, not the player's name or leaderboard account.

In a browser the player is Web Audio (`sound_js.go`). The page makes an
`AudioContext` on the click that starts the game, since browsers start sound
only from a user gesture. Each clip is copied into it once, and a sound is a
buffer source through a gain and a stereo panner.

## Zero allocations

A frame allocates nothing, and the tests hold it to that through Limoni's
diff as well. `TestFramesAllocateNothing` plays scripted input through six
scenes (title, castle with and without key releases, lair, win, loss). Each
frame is drawn and then encoded as the terminal would receive it, and the
test measures the whole thing with `testing.AllocsPerRun`.

What that takes:

- The game's state and every scratch buffer live in one value made at
  start-up: the rays, the framebuffer, the sprite order, the particles, the
  flow field and its queue. It draws a picture of up to 512×200 cells,
  with two additional terminal rows for the HUD.
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

In the browser, bright colours are rounded harder: to sixteen in the middle
tones and 32 in the bright, keeping eight in the dark. xterm.js's WebGL
renderer draws every glyph once for each pair of colours into a texture
atlas, and when the atlas is full it merges pages while a frame waits. Over
one recorded run, from the start to Ratatui's death, that took the pairs
from 33,000 to 12,400, the merges from eleven (7–52 ms each) to one, and the
slowest frames in a hundred from 25 ms to 13 (headless Chromium). The
picture looks the same.

## Why a module of its own

Like `apps/globe`, it has its own `go.mod`, so none of it reaches anyone who
imports Limoni. It depends on a published Limoni and uses only its public
API.
