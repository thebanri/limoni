//go:build !js

package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// A free Render service sleeps after a quiet quarter of an hour and takes up
// to a minute to wake; the request waits that long, off the frame, while the
// screens say they are connecting.
var client = &http.Client{Timeout: 70 * time.Second}

func httpDo(method, url string, body []byte) ([]byte, error) {
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err == nil && resp.StatusCode/100 != 2 {
		err = fmt.Errorf("the scoreboard answered %s", resp.Status)
	}
	return data, err
}

// boardURL is the shared leaderboard's address: -board, or LEMONHUNT_BOARD,
// or the one the game ships with. "off" keeps the scores on this machine.
func boardURL(flag string) string {
	u := flag
	if u == "" {
		u = os.Getenv("LEMONHUNT_BOARD")
	}
	if u == "" {
		u = defaultBoard
	}
	if u == "off" {
		return ""
	}
	return u
}
