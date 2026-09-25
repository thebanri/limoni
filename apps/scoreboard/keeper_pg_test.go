package main

import (
	"context"
	"os"
	"testing"
)

// The Postgres keeper is tested against a real database when
// TEST_DATABASE_URL names one; CI starts one (ci.yml, scoreboard-postgres).
func TestTheBoardSurvivesARestartInPostgres(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	first, err := openPG(url)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := first.pool.Exec(context.Background(), `TRUNCATE runs`); err != nil {
		t.Fatal(err)
	}
	testRestart(t, func() keeper {
		k, err := openPG(url)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(k.pool.Close)
		return k
	})
	var n int
	if err := first.pool.QueryRow(context.Background(), `SELECT count(*) FROM runs`).Scan(&n); err != nil || n != keep {
		t.Errorf("%d rows in the table (%v), want %d", n, err, keep)
	}
	first.pool.Close()
}
