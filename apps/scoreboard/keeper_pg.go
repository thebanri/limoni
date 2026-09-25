package main

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// pgKeeper keeps the board in Postgres: on Render, whose free web services
// have no disk that outlives a restart, a free Neon database. One row a run,
// and never more than the runs kept.
type pgKeeper struct{ pool *pgxpool.Pool }

// A database that scales to zero, as Neon's free one does, takes a moment
// to wake; this is how long the server waits for it.
const pgTimeout = 30 * time.Second

func openPG(url string) (*pgKeeper, error) {
	ctx, cancel := context.WithTimeout(context.Background(), pgTimeout)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	_, err = pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS runs (
		id    bigserial PRIMARY KEY,
		name  text    NOT NULL,
		score integer NOT NULL,
		secs  integer NOT NULL,
		aim   integer NOT NULL,
		won   boolean NOT NULL,
		at    bigint  NOT NULL
	)`)
	if err != nil {
		pool.Close()
		return nil, err
	}
	return &pgKeeper{pool}, nil
}

func (k *pgKeeper) load() ([]entry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), pgTimeout)
	defer cancel()
	rows, err := k.pool.Query(ctx,
		`SELECT name, score, secs, aim, won, at FROM runs ORDER BY score DESC, at, id LIMIT $1`, keep)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (entry, error) {
		var e entry
		err := r.Scan(&e.Name, &e.Score, &e.Secs, &e.Aim, &e.Won, &e.At)
		return e, err
	})
}

// added inserts the run and drops whatever has fallen below the runs kept.
func (k *pgKeeper) added(e entry, _ []entry) error {
	ctx, cancel := context.WithTimeout(context.Background(), pgTimeout)
	defer cancel()
	batch := &pgx.Batch{}
	batch.Queue(`INSERT INTO runs (name, score, secs, aim, won, at) VALUES ($1, $2, $3, $4, $5, $6)`,
		e.Name, e.Score, e.Secs, e.Aim, e.Won, e.At)
	batch.Queue(`DELETE FROM runs WHERE id IN (SELECT id FROM runs ORDER BY score DESC, at, id OFFSET $1)`, keep)
	return k.pool.SendBatch(ctx, batch).Close()
}
