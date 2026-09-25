package board

import (
	"context"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PG keeps the board in Postgres — a free Neon database, on Vercel — and
// holds nothing between requests: the board, each run's place and the
// runs an address has sent lately are all asked of the database. That is
// what a Vercel function needs, since each request may meet a fresh copy
// of it, and it serves a long-running server just as well.
type PG struct{ pool *pgxpool.Pool }

// A database that scales to zero, as Neon's free one does, takes a moment
// to wake; this is how long a request waits for it.
const pgTimeout = 20 * time.Second

// OpenPG connects to the database at url and makes the tables if they are
// not there.
func OpenPG(ctx context.Context, url string) (*PG, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, err
	}
	// Neon's pooled address is PgBouncer in transaction mode, where a
	// statement prepared on one connection may be run on another. These
	// queries are sent unprepared, and cost a round trip less that way.
	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	cfg.MaxConns = 4
	ctx, cancel := context.WithTimeout(ctx, pgTimeout)
	defer cancel()
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS runs (
			id    bigserial PRIMARY KEY,
			name  text    NOT NULL,
			score integer NOT NULL,
			secs  integer NOT NULL,
			aim   integer NOT NULL,
			won   boolean NOT NULL,
			at    bigint  NOT NULL
		);
		CREATE TABLE IF NOT EXISTS posts (
			addr text        NOT NULL,
			at   timestamptz NOT NULL
		);
		CREATE INDEX IF NOT EXISTS posts_addr ON posts (addr, at)`)
	if err != nil {
		pool.Close()
		return nil, err
	}
	return &PG{pool}, nil
}

// Close lets go of the database.
func (p *PG) Close() { p.pool.Close() }

// LazyPG is a PG opened on first use. A server that opened its database as
// it started would not start at all while the database was out of reach,
// and on Vercel every request, /healthz too, would fail with it; this one
// starts, answers /healthz, says the board is not available, and tries the
// database again on the next request.
type LazyPG struct {
	url string
	mu  sync.Mutex
	pg  *PG
}

// NewLazyPG returns a store for the database at url, not yet opened.
func NewLazyPG(url string) *LazyPG { return &LazyPG{url: url} }

func (l *LazyPG) get(ctx context.Context) (*PG, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.pg != nil {
		return l.pg, nil
	}
	pg, err := OpenPG(ctx, l.url)
	if err != nil {
		return nil, err
	}
	l.pg = pg
	return pg, nil
}

func (l *LazyPG) Top(ctx context.Context, n int) ([]Entry, error) {
	pg, err := l.get(ctx)
	if err != nil {
		return nil, err
	}
	return pg.Top(ctx, n)
}

func (l *LazyPG) Add(ctx context.Context, e Entry) (int, error) {
	pg, err := l.get(ctx)
	if err != nil {
		return 0, err
	}
	return pg.Add(ctx, e)
}

func (l *LazyPG) Allow(ctx context.Context, addr string, now time.Time) (bool, error) {
	pg, err := l.get(ctx)
	if err != nil {
		return false, err
	}
	return pg.Allow(ctx, addr, now)
}

// order is the board's order: the higher score, then the run made first,
// then the row made first.
const order = `ORDER BY score DESC, at, id`

func (p *PG) Top(ctx context.Context, n int) ([]Entry, error) {
	ctx, cancel := context.WithTimeout(ctx, pgTimeout)
	defer cancel()
	rows, err := p.pool.Query(ctx, `SELECT name, score, secs, aim, won, at FROM runs `+order+` LIMIT $1`, n)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Entry, error) {
		var e Entry
		err := r.Scan(&e.Name, &e.Score, &e.Secs, &e.Aim, &e.Won, &e.At)
		return e, err
	})
}

// Add inserts the run, counts the runs above it for its place, and drops
// whatever has fallen below the runs kept, in one transaction.
func (p *PG) Add(ctx context.Context, e Entry) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, pgTimeout)
	defer cancel()
	rank := 0
	err := pgx.BeginFunc(ctx, p.pool, func(tx pgx.Tx) error {
		var id int64
		if err := tx.QueryRow(ctx,
			`INSERT INTO runs (name, score, secs, aim, won, at) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
			e.Name, e.Score, e.Secs, e.Aim, e.Won, e.At).Scan(&id); err != nil {
			return err
		}
		if err := tx.QueryRow(ctx,
			`SELECT count(*) + 1 FROM runs WHERE score > $1 OR (score = $1 AND (at < $2 OR (at = $2 AND id < $3)))`,
			e.Score, e.At, id).Scan(&rank); err != nil {
			return err
		}
		if rank > Keep {
			rank = 0
		}
		_, err := tx.Exec(ctx, `DELETE FROM runs WHERE id IN (SELECT id FROM runs `+order+` OFFSET $1)`, Keep)
		return err
	})
	return rank, err
}

// Allow counts the runs addr sent in the minute before now, and records
// this one if there is room. Old records are cleared as it goes.
func (p *PG) Allow(ctx context.Context, addr string, now time.Time) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, pgTimeout)
	defer cancel()
	ok := false
	err := pgx.BeginFunc(ctx, p.pool, func(tx pgx.Tx) error {
		since := now.Add(-time.Minute)
		if _, err := tx.Exec(ctx, `DELETE FROM posts WHERE at <= $1`, since); err != nil {
			return err
		}
		// Two runs from one address at once must not both find room.
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, addr); err != nil {
			return err
		}
		var n int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM posts WHERE addr = $1 AND at > $2`, addr, since).Scan(&n); err != nil {
			return err
		}
		if n >= PostsPerMinute {
			return nil
		}
		ok = true
		_, err := tx.Exec(ctx, `INSERT INTO posts (addr, at) VALUES ($1, $2)`, addr, now)
		return err
	})
	return ok, err
}
