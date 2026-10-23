# Mark & Sweep

A first-person roguelite in a terminal made of terminals, built on [Limoni](../../README.md).

You are the garbage collector. The frameworks have come for the heap: **Bubble Tea**, **Ratatui**,
**tview**, **Ink** and **ncurses**, each a creature made from its own idea. Sweep them, take a card every
level, build something broken, and get through three maps — ten minutes on each, then its warden. Or
play `/dev/zero`, the endless map, for as long as the build holds.

```bash
go install github.com/thebanri/limoni/apps/marksweep@latest
marksweep            # up to 144 frames a second; -showfps shows them, -mute for silence
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
| SETTINGS | in the title and the pause menus: the sound, and the crosshair: style, colour, size, outline |

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
| cursor | termbox | a block cursor with a gopher's eyes; blinks, and every blink brings it nearer |
| bomb | notcurses | a worried bomb; beside you it lights its fuse and marks its blast on the floor. Shot dead it goes off at once, among its own |
| crossterm | crossterm | a steel beetle with a cursor key's cross for a horn; stops, shows its line on the floor, and charges along it |
| lip gloss | Lip Gloss | a lipstick with a glossy gem; keeps behind the others and shields them (`+`): shielded, they take half |
| caret | prompt_toolkit | a floating `>` with a blinking cursor; keeps far, aims a red laser that holds still just before the shot, and fires one fast, hard one |

Kinds come in by level — spinner and boba from the start, termbox from 2, `</>` 3, bomb 4, crab 5,
crossterm 6, modal 8, Lip Gloss 9, caret 10, progress 11, elder 15 — and a kind not yet met this run
comes four times as often until it has, its arrival announced.

Every tenth level a boss comes, in turn, then round again, tougher:

| | from | |
| :--- | :--- | :--- |
| **Boba Prime** | Bubble Tea | rings of pearls, and spinners |
| **Ferris, Crab King** | Ratatui | a claw salvo, five bars in a fan; every third time a claw slam, a ring of fire over the floor, to dash through |
| **The Modal Queen** | tview | a dialog wall, eleven bars side by side with a gap in it; every second time her buttons and a modal |
| **Ink, the Reconciler** | Ink | props drilled down to you, three shots that turn after you; every second time a re-render: it lands beside you where the floor glows |
| **Textual, the Cascade** | Textual | a CSS cascade, blocks falling on you and round you on marked floor; every third time a storm of carets and cursors |

Every ranged enemy shows its attack before
it: it glows and a rising tone plays, and a boss's attack is named under its health bar. Marked floor
(`hazard.go`) is a ring that fills as the moment nears; standing on it, MARKED: MOVE blinks under the
crosshair, since you cannot see your own feet. What hits you
leaves an arrow round the crosshair pointing back at it and a line over the health ("-24 FERRIS, CRAB
KING claw salvo"); the death screen says what killed you. From level 12 some enemies are elites (gold, three
times the health), and some of those are insulated (cyan health bar): explosions and arcs do nothing to
them.

## Cards

79 of them, in two kinds, as in Megabonk: a build holds **six actives and six passives**. Once a kind's
six slots are full, its cards are only upgrades of what is held, so what you take is a choice. The card
screen shows the slots (ACTIVE 3/6 · PASSIVE 3/6), and every card says which kind it is and whether it
is new.

**Actives** are the build's core: abilities that act by themselves, each on its own clock, for a share of
the gun's damage, so whatever makes the gun stronger makes them stronger too. A rank makes one sooner,
bigger or more of it.

| | |
| :--- | :--- |
| Tesla Coil | arcs to the nearest foes |
| Broadcast Storm | bolts from above on the nearest |
| IRQ Field | a field round you that shocks everything inside |
| Segfault | a fire blast where they are, that sets them burning |
| Burn-in | fire left wherever you walk |
| Absolute Zero | a frost nova that hurts and freezes all it reaches |
| Cold Path | lances of ice that pierce and chill |
| Gas Leak | poison clouds thrown onto them |
| Daemons | daemons circling you, hitting what they touch |
| Cron Job | a round at each of the nearest, by itself |
| Round Trip | boomerangs thrown where you look, that hit going and coming |
| Compaction | a vortex that pulls them together, then bursts |

**Passives** are everything else: what the gun's shots do, the elements, health, the dash.

A run starts weak on purpose. Every level then adds 2.5% damage of its own, rare and legendary cards
grow commoner as the levels go by, and a card of an element you already carry comes more often, so
builds come together and the end of a run is much stronger than its start.

The elements live on the enemies as stacks, with a mark over each one carrying them (`^` `*` `o` `!`):

| | |
| :--- | :--- |
| **fire** | burns hurt every half second (armour halves it). Firewall, Overheat, Wildfire (spreads as they die), Incinerate (ten burns erupt), Napalm (explosions burn), Afterburner (dashes leave fire), Phoenix Restart (a death restarts you, once in 90 s) |
| **ice** | chill slows, five chills freeze: no step, no bite, no attack. Shatter, Permafrost, Ice Shards, Cold Snap (a dash ends in a frost nova), Frostbite, Cryo Plating |
| **lightning** | Goroutines' arcs, Static (the shocked take more), Thunderstrike (every 7th hit a bolt), Short Circuit, Spark Gap (a dash zaps), Conductor, Superconductor (arcs carry burn and chill) |
| **poison** | goes through armour. Corrosion (eats armour), Plague, Toxic Fumes, Neurotoxin (slower, softer bites), Dangling Pointer (a share of health), Mark Gas (collecting poisons what is near) |
| **dash** | Double Buffer (+1 charge), goto (further), Context Switch (a dash refills the heap), defer (shots after a dash hit harder), Bulldozer (dashes hit what they pass), Hot Reload |
| **and the rest** | REP, Scroll Region, Diff Engine, fork(), panic(), Recursion, Cascading Diff, Hot Path, Tight Loop, Bigger Heap, Zero-Alloc, Concurrent GC, Escape Analysis, Stack Overflow, kill -9, Mode 2026, Graceful Shutdown, recover(), Mutex, Watchdog, Vampire Thread, GOGC=200, Weak Refs, Reference Counting, Retry, Batch Job, Headroom, Last Resort, Generational GC, Big O, Polyglot (damage for every element), Thermal Shock (fire on the frozen is a blast) |

Some need another first: Wildfire needs Thermal Throttle, Recursion needs fork(), Thermal Shock needs fire
and ice both. A card can come back. Past its last rank it **overclocks**: +12% to what it does, without
end. The broken builds are the ones that multiply each other. fork() children that bounce off walls,
explode and throw shrapnel fill the arena with more shots than it can hold. The pool simply stays full.

## Three maps, and /dev/zero

**SWEEP** is a run through three maps, each a part of the machine with its own walls and light:
**The Heap** (open, a few windows standing in it), **The Stack** (frames in lanes, each with a gap) and
**Kernel Space** (a grid of pins). The heap grows with time on each. At ten minutes the map's **warden**
comes — the Modal Queen, Ink's Reconciler, Textual's Cascade — with less of the crowd round it. Sweep it
and the next map opens a moment later: the build comes along, health is full again, and the new map
starts as hard as five minutes into the last. Sweep the third and the run is won.

A warden that still stands two minutes on brings the **pressure**: enemy health doubles about every two
minutes, they move faster, every one of them gains armour that grows by the minute, and more and more
are insulated. Armour takes a flat amount off every hit, so a cloud of tiny shrapnel stops working just
when it looked unbeatable.

**ENDLESS** is `/dev/zero`: one open map with spawn windows all round, no warden and no end. It climbs
slower, since no map ends there to give the health back; its pressure comes at fifteen minutes and builds
at a third of the pace, but it does not stop.

Enemies walk round walls (`flow.go`): a count of steps from the player's tile over the whole map, made
again when the player moves to another tile; one with a wall in its way follows it downhill.

A bot with perfect aim (`sim_test.go`, `SIM=1`, ten seeds a build of six actives and six passives) gets
to a median of 26 to 38 minutes on the three maps, and wins about one run in twenty; most runs end at
the last warden. On `/dev/zero` it lasts 18 to 20 minutes. It heeds the warnings — a step aside on a
wind-up, a dash through a slam or a shot about to land — and turns to a boss in sight, and nothing else,
so the dash cards do little for it.

Records stay where the game is played: `~/.config/marksweep/records.json`, or the browser's
localStorage. The title shows them as a card for each map: how often its warden was swept, the fewest
seconds one stood, the most swept there in one run; for `/dev/zero` the longest run, its level and its
count; and under them the longest run of all. There is no leaderboard.

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
points at a point on the ray through the crosshair, so on the screen its line meets the crosshair. It
sways behind the aim and kicks on a shot. A reload is a heap change: the gun tips and comes up, the spent
heap (the canister of glowing bytes under it, as full as the magazine) slides out and falls, a full one
rises into the socket and clicks home with a jolt. The muzzle flash, the tracer and the rounds in flight
(hot streaks) are light added to what is behind them; a shot also lights the room round the gun and
lifts the view, and a kill, a blast or a hit taken shakes it. A kill's experience counts the moment it
dies; the freed bytes only stream home for the picture.

The HUD floats on panels that darken the picture under them, its numbers in the walls' pixel font. The
build is listed down the right edge, unframed: ACTIVE 3/6, then a row for each — its name, its ranks as
pips, and a mark of its element's colour that, on an active, brightens as it readies to act and flashes
as it does. A reload is a ring filling round the crosshair. The crosshair is yours
to choose in SETTINGS (from the title or the pause): six styles — cross, cross and dot, dot, circle,
chevron, and the first one in characters — six colours, three sizes, a dark outline or none, with a
preview beside them; the choice is kept with the records (`reticle.go`). Every style opens with a shot's
kick and turns red over an enemy, and a hit marker spreads as it fades (red on a kill).

The elements and the actives are drawn in pixels too (`fx.go`): clouds, flames, mist and fireballs as
soft blobs that grow and fade; rings and patches painted on the floor itself — a nova spreading, the IRQ
field round you, a vortex closing in, a burning or poisoned patch, Ferris's slam; lightning as jagged,
glowing lines, from the sky for a bolt. All in fixed pools: a frame with every active at its last rank,
150 enemies and every effect out costs about 3.4 ms and allocates nothing (`BenchmarkEveryActive`). The menus are cards over a veiled picture,
headed in the pixel font, their choices buttons.

The game draws up to 144 frames a second (`-fps` to change it; a frame costs about 1.5 ms with the pools
full). What reaches the screen is the terminal's to decide: kitty waits 10 ms between its own frames
unless told otherwise, about 100 a second at most — `repaint_delay 4` and `input_delay 1` in
`kitty.conf` let it keep up with a 144 Hz screen, and `-window` opens it so.

A frame allocates nothing: the pools, the gun's pixels and the renderer's scratch are sized once. The
tests hold it to that on every screen, and with the most broken build's pools full (`BenchmarkFullPools`:
about 1.5 ms a frame, 844 shots, 200 enemies — more than before enemies walked round walls, since
fewer are stuck out of sight).

```bash
go test ./...                                  # rules, allocations, every window size
SIM=1 go test -run TestSimulatedRuns -v .      # the bot: eight builds, ten seeds, maps and /dev/zero (SIMBUILD=broken: one build)
DUMP=/tmp/frames go test -run TestDumpFrame .  # every screen as an HTML page of coloured cells
```
