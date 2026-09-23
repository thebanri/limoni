# globe

A world you can turn, search, zoom into and pin, drawn in the terminal.

```bash
go install github.com/thebanri/limoni/apps/globe@latest

globe                        # the whole planet, turning
globe -at Türkiye            # open looking at a place
globe -at Istanbul -zoom 8   # and closer in
globe -ascii                 # shading characters instead of colour
```

`go install` puts `globe` in `$(go env GOPATH)/bin`, which is `~/go/bin` unless
you changed it. A fresh Go install does not add that directory to your `PATH`,
so if the shell says `globe: command not found`, add it:

```bash
export PATH="$PATH:$(go env GOPATH)/bin"   # bash/zsh; put it in ~/.bashrc or ~/.zshrc
fish_add_path (go env GOPATH)/bin          # fish
```

or run it by its full path, `~/go/bin/globe`. There is no need to clone the
repository or `cd` into it.

`/` finds a country or a city, `⏎` flies there and drops a pin, the arrows
turn the globe, `+` and `−` zoom, `space` holds the rotation, a click pins
the point under the pointer, `p` puts the panel away so the world has the
whole screen, `b` turns the borders off, and `?` lists the rest.

## What it is for

It is a real application, and it is also what a Limoni application looks like
**from the outside**: a module of its own that depends on a published Limoni
and uses nothing but its public API. If something here needs an `internal`
package, that is a hole in the library, not a shortcut to take.

It is small on purpose — a globe, a search box and a list — because the
interesting part is that all three are in the semantic tree.

## An agent can drive it

Everything on screen is a node with a role, a label and a value, so a test or
an agent works by name rather than by pixel:

```bash
go run -tags limoni_debug . -socket "$XDG_RUNTIME_DIR/globe.sock"
claude mcp add globe -- limoni-mcp -socket "$XDG_RUNTIME_DIR/globe.sock"
```

Then "find Turkey on the world map" is three calls — type into `search`,
click the row, read the globe's value back:

```
input#search "country or city" value="Türkiye" state=focused
list#results "Results" position=1/1
  list-item "Turkey  Asia" position=1/1 state=selected
image#globe "Globe" value="39.3°N 34.5°E · zoom 2.6×"
  list-item "Turkey" value="39.3°N 34.5°E" bounds=43,19 1x1
```

The name is typed in Turkish and the list answers in English: local names
are searched, English is what the map is written in.

The globe's value says where it is pointed and the marker says where on the
screen it landed, so the agent can confirm that it worked without looking at
a single pixel — and can keep pressing `+` until the value reads the zoom it
was asked for. `globe/agent_test.go` is that same conversation as a test.

## How it draws

The projection is worked backwards. For every pixel inside the disc the
renderer asks which point of the sphere is there, looks it up in a land mask,
and shades it by the angle to a fixed light. Zooming therefore costs nothing:
the sphere is sampled at whatever resolution the pane happens to have.

A terminal cell is about twice as tall as it is wide, so the renderer works in
half-cells — each cell holds two square pixels, drawn as `▀` with the top
pixel as the foreground and the bottom as the background. That is what makes
the sphere round rather than an ellipse, and it doubles the vertical
resolution for free.

The draw path allocates nothing, which `TestDrawDoesNotAllocate` keeps true.

## The data

Natural Earth, which is in the public domain — 1:110m for the map, 1:10m for
the cities:

- a land/sea bitmask, 2048×1024, gzipped to 14 KiB and expanded once into
  256 KiB so that a lookup is one indexed read with no allocation
- the borders between countries, at six resolutions, gzipped to 16 KiB. They
  are stored as a pyramid because a one-cell-wide line sampled at a whole-
  globe zoom breaks into dots; the globe asks the coarsest level whose cells
  still fit inside a pixel, so the line stays a line at every zoom. Coasts
  are not in it — the water's edge is already a change of colour
- 177 countries with the point Natural Earth labels them at, their ISO codes
  and their names in Turkish as well as English
- 7,342 cities and towns from Natural Earth 1:10m — every capital and
  provincial city, searchable by their plain ASCII spelling as well
  ("izmir" finds İzmir)

`globe/geo/gen.go` rebuilds all of it from the source vectors; it is the only thing
in the tree that needs the network.

## Why it is a module of its own

A directory with its own `go.mod` is not part of the module around it, so
`go get github.com/thebanri/limoni` downloads none of this — not the code and
not the 14 KiB of map data. It lives in the Limoni repository because it is
maintained with Limoni, and it ships to nobody who did not ask for it.
