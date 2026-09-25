# Scoreboard

Lemon Hunt's shared leaderboard: a small HTTP server that takes each finished
run and hands back the best ten. It uses nothing outside Go's standard
library, and is a module of its own, so none of it reaches anyone who
imports Limoni.

| | |
| :-- | :-- |
| `GET /scores` | the best ten, as `{"board": [...]}` |
| `POST /scores` | a run, as `{"name", "won", "secs", "shots", "hits", "kills", "lemons"}`; answers `{"rank", "board"}` |
| `GET /healthz` | `ok` |

The server works the score out itself, with the game's rules (aim, time, rats
and lemons; see [the game's README](../lemonhunt/README.md#score-and-leaderboard)),
and turns away runs the game could not have produced: more hits than squirts,
a win without ten lemons or in under 20 seconds, more rats than the level and
Ratatui hold. Names are cut to 12 characters of what the game can show. One
address may send 6 runs a minute. Anyone can still send a run they did not
play: the page is public and the game runs in the player's browser, so there
is nothing to prove a run was played. This keeps the board plausible, not
honest.

It keeps the best hundred runs in `scores.json`, in `DATA_DIR`.

| Variable | |
| :-- | :-- |
| `PORT` | where to listen; Railway sets it |
| `DATA_DIR` | where `scores.json` goes; falls back to `RAILWAY_VOLUME_MOUNT_PATH`, then to memory only |
| `ALLOWED_ORIGINS` | pages a browser may call it from, comma-separated, or `*`; `https://thebanri.github.io` by default |

## Running it

```bash
cd apps/scoreboard
DATA_DIR=/tmp/board go run .
curl localhost:8080/scores
```

Point the game at it with `lemonhunt -board http://localhost:8080`, or the
browser playground with `?app=lemonhunt&board=http://localhost:8080` (and
`ALLOWED_ORIGINS=*` on the server, for a page served from elsewhere).

## Deploying on Railway

1. On [railway.com](https://railway.com), **New Project → Deploy from GitHub
   repo**, and pick this repository.
2. In the service's **Settings → Source**, set the **Root Directory** to
   `apps/scoreboard`. Railway then builds the `Dockerfile` here and reads
   `railway.toml`, which checks `/healthz` after each deploy. Setting
   **Watch Paths** to `/apps/scoreboard/**` keeps changes elsewhere in the
   repository from redeploying it.
3. **Add a volume** to the service (right-click it, or ⌘K → "volume"), mounted
   at `/data`. Without one, the scores are gone at the next deploy. Railway
   tells the server where the volume is (`RAILWAY_VOLUME_MOUNT_PATH`), so no
   variable is needed.
4. In **Settings → Networking**, **Generate Domain**. Opening
   `https://<that domain>/healthz` should say `ok`.
5. Put that address in `apps/lemonhunt/main.go` (`defaultBoard`) and in
   `examples/wasm/index.html` (`scoreboard:` under `lemonhunt`), and push.
   The playground is rebuilt from `main` and uses it from then on.

Railway's pages and menus change from time to time; if a step is not where
this says, its docs name the same things: root directory, volume, domain.
