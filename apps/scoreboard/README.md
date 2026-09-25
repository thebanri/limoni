# Scoreboard

Lemon Hunt's shared leaderboard: a small HTTP server that takes each finished
run and hands back the best ten. It is a module of its own, so none of it,
and none of its Postgres driver, reaches anyone who imports Limoni.

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

It keeps the best hundred runs, in the first of these that is set:

| Variable | |
| :-- | :-- |
| `DATABASE_URL` | a Postgres database: one table, `runs`, made on start |
| `DATA_DIR` | a directory for `scores.json`, on a disk that outlives the process; `RAILWAY_VOLUME_MOUNT_PATH` counts too |
| neither | memory only: gone at the next restart |

and also reads:

| Variable | |
| :-- | :-- |
| `PORT` | where to listen; Render and Railway set it |
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

## Deploying for free: Render and Neon

Render's free web services have no disk that outlives a restart, and its
free Postgres is deleted after 30 days, so the runs go to a free
[Neon](https://neon.com) database instead, which does not expire.

1. **Neon.** Sign up, create a project, and copy its connection string
   (`postgresql://…neon.tech/neondb?sslmode=require`) from the dashboard's
   **Connect** button.
2. **Render.** Sign up with GitHub, then **New → Web Service**, and pick this
   repository.
   - **Root Directory**: `apps/scoreboard`. Render finds the `Dockerfile`
     there and builds it.
   - **Instance Type**: Free.
   - **Environment Variables**: `DATABASE_URL` = the Neon string.
   - Under **Advanced**, **Health Check Path**: `/healthz`, and **Build
     Filters** → included paths: `apps/scoreboard/**`, so that changes
     elsewhere in the repository do not redeploy it.
3. Once it is live, `https://<name>.onrender.com/healthz` should say `ok`.
4. Put that address in `apps/lemonhunt/main.go` (`defaultBoard`) and in
   `examples/wasm/index.html` (`scoreboard:` under `lemonhunt`), and push.
   The playground is rebuilt from `main` and uses it from then on.

What free costs:

- A free Render service sleeps after 15 minutes with no requests, and the
  first request after that waits for it to wake, up to a minute. The game
  asks off the frame and says it is connecting meanwhile, and the playground
  knocks on `/healthz` as soon as the page opens, while the module is still
  loading, so the wait is mostly over by the time a run ends.
- Neon's free database also sleeps when idle and wakes in a moment; the
  server waits up to 30 seconds for it.
- Render gives a workspace 750 free instance hours a month: one service,
  awake all month, is 720.

## Deploying on Railway instead

Railway runs the same `Dockerfile` (with `railway.toml` for its settings) and
can give the service a volume, so no database is needed: set the service's
**Root Directory** to `apps/scoreboard`, add a volume mounted at `/data`
(Railway tells the server where it is), and **Generate Domain**. Railway's
free plan is $1 of usage a month, which this server's few megabytes of
memory fit inside, after a 30-day trial.

Hosts rename their menus from time to time; if a step is not where this
says, their docs use the same words: root directory, environment variable,
health check, volume, domain.
