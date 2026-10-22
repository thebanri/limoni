# Mark & Sweep

A first-person roguelite in a terminal made of terminals, built on [Limoni](../../README.md).

You are the garbage collector. The frameworks have come for the heap: **Bubble Tea**, **Ratatui**,
**tview**, **Ink** and **ncurses**, each a creature made from its own idea. Sweep them, take a card every
level, build something broken, and outlast the pressure. After ten minutes, nothing does for long.

```bash
go install github.com/thebanri/limoni/apps/marksweep@latest
marksweep            # -fps 60 for a smoother picture, -mute for silence
marksweep -window    # in kitty: a window of its own, full screen, with no mouse pointer
```

### Mouse lock

A terminal cannot lock the mouse pointer, so on Wayland the game does what every game does: during a
run it opens a window of its own — full screen, every pixel transparent, so the terminal shows through —
and the compositor locks the pointer to it (`zwp_pointer_constraints_v1`) and hides it. The mouse's
movement comes as relative motion, unaccelerated, and the keys as the compositor's own presses and
releases, so a key let go is never left held. No permission, no setup, no extra program: the Wayland
client is the game's own (`wayland_linux.go`). The cards are chosen with the mouse moving left and right
and a click, or 1, 2, 3. The pause and switching to another window give the mouse and the keys back.

Elsewhere (X11, macOS, Windows), or if the compositor cannot, the terminal's mouse reports turn the view, and at the window's
edge the view keeps turning. In kitty, `-window` at least hides the pointer: it opens the game in a kitty
window of its own with kitty's `mouse_hide_wait` set so the pointer goes once the mouse rests.

## Playing

| | |
| :--- | :--- |
| mouse | look, as in any first-person game: moving it turns the view, and the crosshair stays in the middle. In the browser the pointer is locked. A terminal cannot lock it, so with the pointer at the window's edge the view keeps turning that way; move back in to stop |
| left button | sweep (hold to keep firing). `F` does the same from the keyboard |
| `R` or right button | collect. The heap (magazine) never refills on its own |
| `W` `A` `S` `D` | move. `←` `→` or `Q` `E` turn |
| `Space` | dash: a short burst no hit lands through. Two charges (◆◆ over the health), each back in 1.6 s |
| `1` `2` `3` or a click | take a card |
| `Esc` or `P` | pause. In the browser, Esc also lets the pointer go |
| `M` | sound on and off |

Kitty, Ghostty, WezTerm and foot report key releases, so movement keys stay down exactly as long as you
hold them. Elsewhere a press holds for a moment and the key's repeats keep it going.

## The heap

Enemies come in through the spawn windows on the walls:

| | from | |
| :--- | :--- | :--- |
| spinner | Bubble Tea | a pink orb its dots go round, in swarms of five, fast and fragile |
| boba | Bubble Tea | a jelly cup on stubby legs, a straw on its head and pearls in its belly |
| `</>` | Ink | its brackets are its jaws and its slash a glowing spine; hops, and splits into `<` and `>` |
| crab | Ratatui | a crab with a rat's ears, whiskers and tail; keeps its distance and fires bars |
| modal | tview | a mimic: its title bar lifts on a row of teeth, its `[Y]` and `[N]` buttons glare; slow and tough, and splits into the buttons |
| progress | Bubbles | a serpent, its filled part bright and the rest grey; weaves in at speed |
| elder | ncurses | an old phosphor monitor walking on two legs, a face on its screen; slow, armoured |

Every tenth level a boss comes: **Boba Prime**, who throws rings of pearls and calls spinners, then
**Ferris, Crab King**, crowned, then again, tougher each time. Every ranged enemy shows its attack before
it: it glows and a rising tone plays, and a boss's attack is named under its health bar. Ferris takes
turns between a **claw salvo**, five bars in a fan (get behind cover), and every third time a **claw slam**,
a ring of fire spreading over the floor that hurts as it passes unless you dash through it. What hits you
leaves an arrow round the crosshair pointing back at it and a line over the health ("-24 FERRIS, CRAB
KING claw salvo"); the death screen says what killed you. From level 12 some enemies are elites (gold, three
times the health), and some of those are insulated (cyan health bar): explosions and arcs do nothing to
them.

## Cards

71 of them. A run starts weak on purpose — every level then adds 2.5% damage of its own, rare and
legendary cards grow commoner as the levels go by, and a card of an element you already carry comes more
often, so builds come together and the end of a run is much stronger than its start.

The elements live on the enemies as stacks, with a mark over each one carrying them (`^` `*` `o` `!`):

| | |
| :--- | :--- |
| **fire** | burns hurt every half second (armour halves it). Firewall, Overheat, Wildfire (spreads as they die), Incinerate (ten burns erupt), Napalm (explosions burn), Afterburner (dashes leave fire), Phoenix Restart (a death restarts you, once in 90 s) |
| **ice** | chill slows, five chills freeze: no step, no bite, no attack. Shatter, Permafrost, Ice Shards, Cold Snap (a dash ends in a frost nova), Absolute Zero, Frostbite, Cryo Plating |
| **lightning** | Goroutines' arcs, Static (the shocked take more), Thunderstrike (every 7th hit a bolt), Tesla Coil, Short Circuit, Spark Gap (a dash zaps), Conductor, Superconductor (arcs carry burn and chill) |
| **poison** | goes through armour. Corrosion (eats armour), Plague, Toxic Fumes, Neurotoxin (slower, softer bites), Dangling Pointer (a share of health), Mark Gas (collecting poisons what is near) |
| **dash** | Double Buffer (+1 charge), goto (further), Context Switch (a dash refills the heap), defer (shots after a dash hit harder), Bulldozer (dashes hit what they pass), Hot Reload |
| **and the rest** | REP, Scroll Region, Diff Engine, fork(), panic(), Recursion, Cascading Diff, Hot Path, Tight Loop, Bigger Heap, Zero-Alloc, Concurrent GC, Escape Analysis, Stack Overflow, kill -9, Mode 2026, Graceful Shutdown, recover(), Mutex, Watchdog, Vampire Thread, GOGC=200, Weak Refs, Reference Counting, Retry, Batch Job, Headroom, Last Resort, Generational GC, Big O, Daemons (orbit you), Cron Job (fires by itself), Polyglot (damage for every element), Thermal Shock (fire on the frozen is a blast) |

Some need another first: Wildfire needs Thermal Throttle, Recursion needs fork(), Thermal Shock needs fire
and ice both. A card can come back. Past its last rank it **overclocks**: +12% to what it does, without
end. The broken builds are the ones that multiply each other. fork() children that bounce off walls,
explode and throw shrapnel fill the arena with more shots than it can hold. The pool simply stays full.

## The pressure

For ten minutes the heap grows with time. Then the pressure starts: enemy health doubles about every
two minutes, they move faster, every one of them gains armour that grows by the minute, and more and more
are insulated. Armour takes a flat amount off every hit, so a cloud of tiny shrapnel stops working just
when it looked unbeatable. A bot with perfect aim (`sim_test.go`, `SIM=1`, ten seeds a build) dies at a median of
7 to 11 minutes, whatever it builds — it never dashes on purpose, so the dash cards do nothing for it.

Records (the longest run, the highest level, the most swept, the most shots alive at once, the biggest
hit) stay where the game is played: `~/.config/marksweep/records.json`, or the browser's localStorage.
There is no leaderboard.

## How it is drawn

The room is a raycaster in half-block pixels, two to a cell, one ray a column. A wall's texture is a
terminal window drawn in a 3×5 pixel font; it is mip-mapped, and the smaller levels draw each character as
a bar of its colour, the way an editor's minimap shows code, because letters averaged down are grey noise.
So text reads up close and far walls stay clean lines. The floor is polished and shows the walls standing
on it; the floor's seams and the ceiling's trays thin out before they would shimmer.

The enemies are pixel creatures over that (`sprites.go`), painted in code at start-up, each with a dark
outline so it reads against any wall, and scaled by picking the nearest pixel. When one dies it comes
apart into its framework's characters. Every card has a picture of its own too (`cardart.go`), on a
glow of its rarity's colour: silver for common, blue for rare, gold for legendary, whose frame has a
glint running round it.

The gun has a small 3D engine of its own (`viewmodel.go`): a mesh of boxes, hulls and prisms built once,
posed every frame, lit, rasterised with a depth buffer into half-block pixels and outlined. Its barrel
points at a point on the ray through the crosshair, so on the screen its line meets the crosshair. It sways behind the aim, kicks on a shot, and on a reload rolls over while the heap (the
canister of glowing bytes under it, as full as the magazine) drops out and a fresh one comes up. The
muzzle flash and the tracer are light added to what is behind them.

The HUD floats on panels that darken the picture under them, its numbers in the walls' pixel font.

A frame allocates nothing: the pools, the gun's pixels and the renderer's scratch are sized once. The
tests hold it to that on every screen, and with the most broken build's pools full (`BenchmarkFullPools`:
about 1.2 ms a frame, 872 shots, 200 enemies).

```bash
go test ./...                                  # rules, allocations, every window size
SIM=1 go test -run TestSimulatedRuns -v .      # the bot's runs, three builds, three seeds
DUMP=/tmp/frames go test -run TestDumpFrame .  # every screen as an HTML page of coloured cells
```
