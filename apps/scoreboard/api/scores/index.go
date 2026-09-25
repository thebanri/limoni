// Package scores is the scoreboard's /scores on Vercel; vercel.json sends
// /scores here. See package fn.
package scores

import (
	"net/http"

	"github.com/thebanri/limoni/apps/scoreboard/fn"
)

// Handler is what Vercel calls.
func Handler(w http.ResponseWriter, r *http.Request) { fn.Scores(w, r) }
