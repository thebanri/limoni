// Package healthz is the scoreboard's /healthz on Vercel; vercel.json sends
// /healthz here. The playground calls it as the page opens, which also
// wakes the functions before a run ends. See package fn.
package healthz

import (
	"net/http"

	"github.com/thebanri/limoni/apps/scoreboard/fn"
)

// Handler is what Vercel calls.
func Handler(w http.ResponseWriter, r *http.Request) { fn.Health(w, r) }
