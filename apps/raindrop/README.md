# raindrop

Rain on a puddle, falling where an entropy pool says.

```bash
go install github.com/thebanri/limoni/apps/raindrop@latest

raindrop              # watch it; click to drop one yourself, q to quit
raindrop -theme dusk  # ocean, lemon, dusk, moss or ink; t cycles them
raindrop -fps 30      # fewer frames, for a slow terminal or link
```

Every frame, key and mouse event is stirred into a SHA-256 pool together with
the nanosecond it arrived at. The low bits of those timestamps are scheduling
jitter nobody can predict. The pool decides whether a drop falls and where, and
its current value is shown along the bottom.

That is the idea behind Cloudflare's wall of lava lamps, shown rather than used.
Two things it is not:

- **A source of keys.** The pool is printed on screen. Keys come from
  `crypto/rand`, which draws on the operating system's own entropy.
- **Randomness from the water.** The waves are computed from the pool, so
  feeding them back in would add nothing. The pool is only as unpredictable as
  what was stirred into it: a simulated puddle is deterministic, and only a
  real one filmed by a camera would add entropy.

The water is a height field stepped with the damped wave equation and drawn in
half blocks, two pixels to a cell. Once the pond has the window's size, a frame
allocates nothing, the diff included (`go test -bench .` reports the bytes a
frame sends).
