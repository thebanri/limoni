// Package scores is the scoreboard's /drop/scores, Lemon Drop's board, on
// Vercel; vercel.json sends /drop/scores here. See package fn.
package scores

import (
	"net/http"

	"github.com/thebanri/limoni/apps/scoreboard/fn"
)

// Handler is what Vercel calls.
func Handler(w http.ResponseWriter, r *http.Request) { fn.DropScores(w, r) }
