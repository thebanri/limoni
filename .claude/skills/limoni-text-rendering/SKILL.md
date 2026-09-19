---
name: limoni-text-rendering
description: How Limoni measures and stores text — UAX #29 grapheme clusters, cell width, the cluster table, mode 2027 and the diff's cursor resync. Load before touching core/grapheme, core/cell widths, Buffer.SetString, the ANSI diff's glyph emission, or anything about emoji, flags, combining marks or East Asian width.
---

# Text: clusters, widths, and the cost of getting them right

A cell holds one `rune`, but a character can be several code points. Multi-code-
point clusters are interned and the cell stores a handle above the Unicode range
(`cell.RuneClusterBase`), so `Cell` stays 16 bytes and plain text never touches
the table.

## Layout of the work

| Path | What it owns |
| :--- | :--- |
| `core/grapheme/gen.go` | `//go:build ignore` generator; reads the UCD, writes `tables.go` |
| `core/grapheme/tables.go` | generated ranges + property bits (GCB, ExtPict, InCB, emoji, width) |
| `core/grapheme/grapheme.go` | `Next`, `RuneWidth`, `StringWidth`, `Count`, GB3–GB999 |
| `core/grapheme/mode.go` | clusters on/off (`LIMONI_GRAPHEME=0`), mode 2027 sequences |
| `core/cell/clusters.go` | interning, `NextCluster`, `ClusterContent`, `AppendContent` |
| `core/buffer/*.go` | `SetStringWithin` writes clusters; the diff emits them |

Regenerating for a new Unicode version: download the files `gen.go` lists
(`GraphemeBreakProperty.txt`, `emoji-data.txt`, `DerivedCoreProperties.txt`,
`EastAsianWidth.txt`, `extracted/DerivedGeneralCategory.txt`), run
`go run gen.go -version <v> -dir <ucd>`, and replace
`testdata/GraphemeBreakTest.txt` too.

## Rules worth remembering

- **Width of a cluster is the widest code point in it**, with VS16 forcing two
  columns and VS15 one. Zero width comes from general category Mn/Me/Cf — *not*
  from GCB Extend, which wrongly zeroes U+FF9E. `EastAsianWidth.txt` has
  `@missing` defaults; ignoring them mismeasures whole unassigned blocks.
- **Correctness is checked against the standard**: all 766 cases of
  `GraphemeBreakTest.txt` pass. Breaking GB11, GB9c or GB12/13 on purpose fails
  3, 16 and 6 cases — use that when changing the state machine.
- **Terminals disagree.** Setup asks for mode 2027 (`CSI ? 2027 h`), which
  Ghostty, WezTerm, foot and Contour implement. Others advance per code point,
  so the diff re-anchors the cursor after every cluster: CHA in stream and
  inline mode, a forgotten cursor position in sparse mode. Without it, one
  family emoji shifts the rest of the row. `core/buffer/cluster_test.go`
  interprets the output the way such a terminal would.
- **REP never repeats a cluster**: a terminal may repeat only its last code point.
- **The table is capped** at `1 << 20` clusters; past it a cluster degrades to
  its first code point rather than growing the process without limit.
- **`LIMONI_GRAPHEME=0`** (or `cell.SetGraphemeClusters(false)`) restores one
  code point per cell and stops the mode 2027 request.

## Performance traps (this cost a day)

Segmentation is on the hot path; integrating it first cost 20–90% across the
draw benchmarks. What recovered it:

- **`cell.AppendContent` must stay inlinable.** Budget is 80; inlining
  `utf8.AppendRune` into it pushed the cost to 143 and the diff lost a quarter
  of its speed. Keep the ASCII byte fast path first and everything else in
  `appendContentSlow`. Check with `go build -gcflags=-m`.
- **ASCII fast paths** in `Buffer.SetStringWithin`, `cell.StringWidth` and
  `grapheme.Next`: printable ASCII followed by another ASCII byte is a
  one-column cluster, no segmentation needed.
- **ASCII after non-ASCII always breaks** (no ASCII code point is Extend, ZWJ,
  SpacingMark or Extended_Pictographic), so `Next` returns right after the first
  rune unless it is Prepend. This is the common shape of UI text.
- **Direct tables beat binary search**: `bmpProps[0x10000]` and
  `pictProps` for U+1F000–U+1FFFF, filled from `propTable` in `init`, cover
  nearly everything a UI draws. 136 KB static, emoji width 10.7ns → 4.5ns.

Final measured cost against the previous commit, same machine, `-count=3`
medians: TextHeavyFrame +5% (three symbols per line), Diff_FullChanges +2%,
HundredLayers and Diff_PartialChanges unchanged, zero allocations throughout.

## Benchmarking method (use this, not README numbers)

Absolute figures drift with the machine's state. Compare commits on one machine
in one sitting:

```bash
git worktree add --detach /tmp/base HEAD      # or the commit before the change
(cd /tmp/base && go test ./benchmarks -run '^$' -bench X -count=3)
go test ./benchmarks -run '^$' -bench X -count=3
git worktree remove /tmp/base
```

Then follow the repo's benchmark honesty rules: publish the delta and say the
table's absolute numbers were not re-measured, rather than quietly editing rows.

## Still open

Widgets that cut or place text by `[]rune` — `TextInput`, `TextArea`, `Table`,
`Toast`, `Dialog`, `Fuzzy` — can split a cluster at the truncation point.
Drawing and measuring are cluster-aware; positioning inside those widgets is not.

## The handshake decides whether re-anchoring is needed

`driver.ProbeQueries` asks DECRQM 2027 *and measures*: it writes a ZWJ family
emoji and reads the cursor back. `CapabilityProfile.ClusterWidths` (→
`DiffOptions.ClusterWidths`) is true when mode 2027 is on or the family measured
2 columns; the diff then skips `appendClusterResync`. Keep the
`cell.IsCluster(...) && !opts.ClusterWidths` order: the other order cost 5% on
`BenchmarkDiff_FullChanges`, because the option load ran for every cell.
