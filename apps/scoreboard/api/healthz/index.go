// Package healthz is the scoreboard's /healthz on Vercel. The playground
// calls it as the page opens, which also warms the functions up before a
// run ends.
package healthz

import (
	"net/http"

	"github.com/thebanri/limoni/apps/scoreboard/board"
)

var health = board.Health()

// Handler is what Vercel calls.
func Handler(w http.ResponseWriter, r *http.Request) { health.ServeHTTP(w, r) }
