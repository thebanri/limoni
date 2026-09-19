# Stability and versioning

Limoni is **pre-1.0**. This page says what that means in practice, so you can
decide how tightly to pin it.

## The rules until v1.0.0

- **Patch releases (`v0.x.y`) do not break the API.** Upgrading within a minor
  line should be safe: bug fixes, performance work, new widgets and new options only.
- **Minor releases (`v0.x.0`) may break the API.** Every break is listed under
  **Breaking** in [CHANGELOG.md](../CHANGELOG.md), with what to do instead.
- Before a release is tagged, `gorelease` is run against the previous tag, so
  a break cannot slip into a patch release unnoticed.
- Allocation budgets are part of the contract. A draw path that is `0 B/op`
  today does not start allocating in a patch release.

## How settled each package is

| Package | Status | What to expect |
| :--- | :--- | :--- |
| `limoni` (root), `component`, `layout` | **Settling** | The shape is intended to survive to 1.0. Names may still change in a minor release. |
| `widgets`, `core/cell`, `core/buffer`, `testkit` | **Settling** | Same, but widget *state* types may gain unexported fields. Don't compare them with `==`. |
| `core/engine`, `core/terminal`, `core/driver`, `core/accessibility` | **Internal-ish** | Public so advanced users can drive them. Prefer the root package re-exports. |
| `automation`, `uitest`, `session`, `cmd/limoni-mcp` | **Experimental** | New in v0.3. Selectors and the wire protocol may change as agents and tests use them. |
| `graphics`, `animation`, `core/grapheme` | **Settling** | Functionally stable. The API surface is small. |

## Road to v1.0

v1.0 will be tagged when:

1. ~~Instance isolation, so that two Limoni apps can run in one process, and
   context-aware immediate mode (`RunWithContext`).~~ Done after v0.3.0.
2. ~~Terminal capabilities are probed rather than guessed from environment
   variables.~~ Done after v0.3.0.
3. ~~Widgets that truncate text are grapheme-cluster aware.~~ Done after v0.3.0,
   except Markdown's word wrap.
4. The experimental packages have been used by at least one application outside
   this repository, and their API has had one round of changes from that use.

After v1.0, breaking changes need a new major version (`/v2`), as Go modules require.

## Supported Go versions

The module declares the oldest Go it builds with (currently **1.25**). CI
checks it with `GOTOOLCHAIN=local` on that version. Raising the floor is called
out in the changelog.
