# Scoreboard

The shared leaderboards of Lemon Hunt and Lemon Drop: each takes finished
runs and hands back the best ten. It runs as two Vercel functions over a free Neon database, or
as one long-running server, from the same handler. It is a module of its
own, so none of it, and none of its Postgres driver, reaches anyone who
imports Limoni.

| | |
| :-- | :-- |
| `GET /scores` | the best ten, as `{"board": [...]}` |
| `POST /scores` | a run, as `{"name", "won", "secs", "shots", "hits", "kills", "lemons"}`; answers `{"rank", "board"}` |
| `GET /drop/scores` | Lemon Drop's best ten, as `{"board": [...]}` |
| `POST /drop/scores` | a run of Lemon Drop, as `{"name", "secs", "pieces", "drops", "clears"}`, where `clears` is each clear as `[base, chain]`; answers `{"rank", "board"}` |
| `GET /healthz` | `ok` |

| | |
| :-- | :-- |
| `board/` | the rules, the HTTP handler, and the stores: `PG` (Postgres) and `Mem` (memory, and a file) |
| `fn/` | the board as Vercel runs it: the handler over `DATABASE_URL`, made on the first request |
| `api/scores`, `api/drop/scores`, `api/healthz` | the Vercel functions, one line each into `fn/`; `vercel.json` sends `/scores` and `/healthz` to them. Nothing else goes in `api/`: Vercel would take any other `.go` file there, a test included, for a function |
| `main.go` | the server itself: what Vercel's Go preset runs, and what runs on Render, Railway or a machine of one's own |

The server works the score out itself, with the game's rules (aim, time, rats
and lemons; see [the game's README](../lemonhunt/README.md#score-and-leaderboard)),
and turns away runs the game could not have produced: more hits than squirts,
a win without ten lemons or in under 20 seconds, more rats than the level and
Ratatui hold. Names are cut to 12 characters of what the game can show. One
address may send 6 runs a minute, counted in the database, so it holds
across however many copies of the function are running. Lemon Drop's runs are scored the same way, from what the run did: the
server adds up each clear's base times its level (one more every four
clears) times its chain, and the points for dropping pieces
([`board/drop.go`](board/drop.go)); it turns away a clear narrower than the
board, a chain that skips, and more sand cleared than the pieces brought.
Both games share the limit on runs an address may send. Anyone can still
send a run they did not play: the page is public and the game runs in the
player's browser, so there is nothing to prove a run was played. This keeps
the board plausible, not honest.

It keeps the best hundred runs, in the first of these that is set:

| Variable | |
| :-- | :-- |
| `DATABASE_URL` | a Postgres database: tables `runs`, `drop_runs` and `posts`, made on first use. Vercel's Neon integration sets it |
| `DATA_DIR` | (long-running server only) a directory for `scores.json` and `drop.json`, on a disk that outlives the process; `RAILWAY_VOLUME_MOUNT_PATH` counts too |
| neither | (long-running server only) memory: gone at the next restart |

and also reads:

| Variable | |
| :-- | :-- |
| `ALLOWED_ORIGINS` | pages a browser may call it from, comma-separated, or `*`; `https://thebanri.github.io` by default |
| `PORT` | (long-running server) where to listen; Render and Railway set it |

## Running it

```bash
cd apps/scoreboard
DATA_DIR=/tmp/board go run .
curl localhost:8080/scores
```

Point the game at it with `lemonhunt -board http://localhost:8080`, or the
browser playground with `?app=lemonhunt&board=http://localhost:8080` (and
`ALLOWED_ORIGINS=*` on the server, for a page served from elsewhere).

## Deploying for free: Vercel and Neon

Both have free plans that do not expire: Vercel's Hobby plan (for
non-commercial projects, which this is) and Neon's Free plan.

1. **Vercel.** Sign up with GitHub, then **Add New → Project**, and import
   this repository.
   - **Root Directory**: `apps/scoreboard`. Vercel's Go preset finds
     `go.mod` and `main.go` there and runs the server; leave the preset as
     it detects it. (A project set to **Other** builds the functions in
     `api/` instead, which serve the same handler.)
   - Deploy. The first deploy has no database yet, so `/scores` answers 503;
     that is expected.
2. **Neon.** In the Vercel project, **Storage → Create Database → Neon**
   (or connect an existing Neon project from the Marketplace). This sets
   `DATABASE_URL` on the project. Then **Deployments → ⋯ → Redeploy**, so
   the functions start with it.
3. `https://<project>.vercel.app/healthz` should say `ok`, and
   `https://<project>.vercel.app/scores` `{"board":[]}`.
4. Put that address in `apps/lemonhunt/main.go` (`defaultBoard`) and in
   `examples/wasm/index.html` (`scoreboard:` under `lemonhunt`), and push.
   The playground is rebuilt from `main` and uses it from then on.

In **Settings → Git**, an **Ignored Build Step** of
`git diff --quiet HEAD^ HEAD -- .` keeps changes elsewhere in the repository
from redeploying it.

Two things this was not able to check before it was written, having no
Vercel account to try it on: the Go version Vercel builds with (the module
asks for Go 1.25; if the build says otherwise, that is the place to look),
and the exact names of Vercel's menus. The handler and the Postgres store
are tested against a real Postgres in CI, and the function itself is called
the way Vercel calls it (`fn/fn_test.go`).

If `/healthz` answers 404:

- Open `https://<project>.vercel.app/` itself. It should show one line, "Lemon
  Hunt's shared leaderboard". A 404 there too means Vercel is not building
  `apps/scoreboard`: check **Settings → Build and Deployment → Root
  Directory**.
- Check which commit the deployment was built from (the deployment's page
  names it). **Redeploy** builds the same commit again; a project made before
  `apps/scoreboard` reached `main` needs a deployment of a newer commit,
  which the next push to `main` makes, or **Create Deployment** with `main`.
- A build log warning that "internal rewrites in backend framework projects
  now route requests using the rewritten destination path" means the Go
  preset is running `main.go`, and `vercel.json` hands it `/api/healthz` for
  `/healthz`. The server answers on both since this was found: the first
  deployment of it answered 404.

Neon's free database sleeps when idle and takes a moment to wake; the first
request after a quiet spell waits for it, and the playground knocks on
`/healthz` as the page opens to get ahead of that.

## Deploying elsewhere: Render or Railway

The long-running server runs the same handler from the `Dockerfile`.

- **Render**: a free web service, Root Directory `apps/scoreboard`,
  `DATABASE_URL` set to a Neon database (Render's free disk does not outlive
  a restart, and its free Postgres is deleted after 30 days), and a health
  check on `/healthz`. It sleeps after 15 minutes with no requests and takes
  up to a minute to wake.
- **Railway**: Root Directory `apps/scoreboard` (`railway.toml` is read from
  there), a volume mounted at `/data` and no database needed, and a
  generated domain. Its free plan is $1 of usage a month after a 30-day
  trial, which this server's few megabytes of memory fit inside.
