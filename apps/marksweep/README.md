# Mark & Sweep

A first-person roguelite in a terminal made of terminals, built on [Limoni](../../README.md).

You are the garbage collector. The frameworks have come for the heap: **Bubble Tea**, **Ratatui**,
**tview**, **Ink** and **ncurses**, each drawn the way its own apps look. Sweep them, take a card every
level, build something broken, and outlast the pressure. After ten minutes, nothing does for long.

```bash
go install github.com/thebanri/limoni/apps/marksweep@latest
marksweep            # -fps 60 for a smoother picture, -mute for silence
```

Or [play it in the browser](https://thebanri.github.io/limoni/?app=marksweep).

## Playing

| | |
| :--- | :--- |
| mouse | look, as in any first-person game: moving it turns the view, and the crosshair stays in the middle. In the browser the pointer is locked. A terminal cannot lock it, so with the pointer at the window's edge the view keeps turning that way; move back in to stop |
| left button | sweep (hold to keep firing). `F` does the same from the keyboard |
| `R` or right button | collect. The heap (magazine) never refills on its own |
| `W` `A` `S` `D` | move. `←` `→` or `Q` `E` turn |
| `Space` | dash: a short burst, and a moment no hit lands |
| `1` `2` `3` or a click | take a card |
| `Esc` or `P` | pause. In the browser, Esc also lets the pointer go |
| `M` | sound on and off |

Kitty, Ghostty, WezTerm and foot report key releases, so movement keys stay down exactly as long as you
hold them. Elsewhere a press holds for a moment and the key's repeats keep it going.

## The heap

Enemies come in through the spawn windows on the walls:

| | from | |
| :--- | :--- | :--- |
| spinner | Bubble Tea | `⠋⠙⠹` in swarms of five, fast and fragile |
| boba | Bubble Tea | a pink rounded cup with tapioca pearls |
| `</>` | Ink | hops, and splits into `<` and `>` |
| crab | Ratatui | Ferris keeps its distance and fires bars |
| modal | tview | slow and tough, and splits into its `[Y]` and `[N]` buttons |
| progress | Bubbles | weaves in at speed |
| elder | ncurses | plain ASCII, slow, armoured |

Every tenth level a boss comes: **Boba Prime**, who throws rings of pearls and calls spinners, then
**Ferris, Crab King**, then again, tougher each time. From level 12 some enemies are elites (gold, three
times the health), and some of those are insulated (cyan health bar): explosions and arcs do nothing to
them.

## Cards

| | rarity | |
| :--- | :--- | :--- |
| REP | common | +1 shot per trigger pull |
| Scroll Region | common | shots bounce off walls |
| Diff Engine | common | shots pass through enemies |
| fork() | rare | a hit splits the shot |
| panic() | rare | hits explode |
| Goroutines | rare | hits arc to nearby enemies |
| Hot Path, Tight Loop, Bigger Heap, Concurrent GC | common | damage, fire rate, magazine, collecting speed |
| Zero-Alloc | rare | a chance that a shot costs nothing |
| Escape Analysis | rare | critical hits |
| Graceful Shutdown, recover() | common, rare | health; health on every kill |
| Hot Reload, GOGC=200 | common | speed and dash; experience |
| Cascading Diff | legendary | explosions throw shrapnel (needs panic()) |
| Recursion | legendary | children split again (needs fork()) |
| Stack Overflow | legendary | every hit in a row adds damage |
| Mode 2026 | legendary | a kill can slow the world for two seconds |
| kill -9 | legendary | enemies under a share of their health die |

A card can come back. Past its last rank it **overclocks**: +12% to what it does, without end. The
broken builds are the ones that multiply each other. fork() children that bounce off walls, explode
and throw shrapnel fill the arena with more shots than it can hold. The pool simply stays full.

## The pressure

For ten minutes the heap grows with time. Then the pressure starts: enemy health doubles about every
two minutes, they move faster, every one of them gains armour that grows by the minute, and more and more
are insulated. Armour takes a flat amount off every hit, so a cloud of tiny shrapnel stops working just
when it looked unbeatable. A bot with perfect aim (`sim_test.go`) dies between 10 and 19 minutes,
whatever it builds.

Records (the longest run, the highest level, the most swept, the most shots alive at once, the biggest
hit) stay where the game is played: `~/.config/marksweep/records.json`, or the browser's localStorage.
There is no leaderboard.

## How it is drawn

The room is a raycaster in half-block pixels, two to a cell, one ray a column. A wall's texture is a
terminal window drawn in a 3×5 pixel font; it is mip-mapped, and the smaller levels draw each character as
a bar of its colour, the way an editor's minimap shows code, because letters averaged down are grey noise.
So text reads up close and far walls stay clean lines. The floor is polished and shows the walls standing
on it; the floor's seams and the ceiling's trays thin out before they would shimmer.

The enemies are drawn in characters over that, art scaled by picking the nearest character: lines and
blocks stretch, letters stay one character, so a near Bubble Tea box grows and its face stays readable.
Boxed enemies are filled with a panel colour so they read against the walls.

The gun has a small 3D engine of its own (`viewmodel.go`): a mesh of boxes, hulls and prisms built once,
posed every frame, lit, rasterised with a depth buffer into half-block pixels and outlined. Its barrel
points at a point on the ray through the crosshair, so on the screen its line meets the crosshair. It sways behind the aim, kicks on a shot, and on a reload rolls over while the heap (the
canister of glowing bytes under it, as full as the magazine) drops out and a fresh one comes up. The
muzzle flash and the tracer are light added to what is behind them.

The HUD floats on panels that darken the picture under them, its numbers in the walls' pixel font.

A frame allocates nothing: the pools, the gun's pixels and the renderer's scratch are sized once. The
tests hold it to that on every screen, and with the most broken build's pools full (`BenchmarkFullPools`:
about 0.9 ms a frame, 878 shots, 200 enemies).

```bash
go test ./...                                  # rules, allocations, every window size
SIM=1 go test -run TestSimulatedRuns -v .      # the bot's runs, three builds, three seeds
DUMP=/tmp/frames go test -run TestDumpFrame .  # every screen as an HTML page of coloured cells
```
