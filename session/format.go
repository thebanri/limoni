// Package session records a running declarative Limoni application and replays
// the recording against the same model, verifying every frame.
//
// A recording is every message the model received, in the order Update saw
// them, plus the semantic tree each rendered frame produced. Replaying feeds
// the messages back through a fresh model and renders its View at each recorded
// frame; the first frame whose tree differs is reported as a divergence, and a
// panic in the original run is reported as reproduced or not.
//
// # Why messages rather than input
//
// A recording taken from the terminal's input would not replay: timers fire at
// different moments and asynchronous commands finish in a different order.
// Recording at Update instead captures the outcome of those races, and a
// command's effect enters the recording as the message it produced — so a
// replay never re-runs an HTTP request or reads the clock, it receives what the
// original run received. The runtime already applies command results in the
// order they were scheduled, which is what makes that stream reproducible.
//
// # What it does not guarantee
//
// Update and View must be deterministic given their inputs. A model that reads
// time.Now or math/rand directly, instead of receiving them as messages
// (engine.NowCmd), will diverge on replay. The replay detects that and names the
// frame; it cannot correct it. The limonivet analyzer in tools/limonivet flags
// such calls before they get that far.
//
// # Privacy
//
// Recordings are written with 0600 permissions and are closed by default:
//
//   - Typed text is replaced with 'x' unless Policy.RecordText is set. Special
//     keys and shortcuts are kept, because they carry the interaction.
//   - Tree values of input fields are dropped unless Policy.ExposeInputValues is
//     set, and values of sensitive nodes are always dropped.
//   - Application message types are recorded by name only unless registered
//     with Register or RegisterRedacted.
//
// Even with RecordText set, characters are redacted whenever a sensitive field
// might have focus — including in the window after a focus-moving key before
// the next frame confirms where focus went.
package session

import (
	"encoding/json"
	"time"

	"github.com/thebanri/limoni/core/accessibility"
)

// FormatName and FormatVersion identify a recording file. A replay refuses a
// file whose name or version it does not recognise.
const (
	FormatName    = "limoni-session"
	FormatVersion = 1
)

// Policy decides what a recording keeps. Every field defaults to closed.
type Policy struct {
	// RecordText keeps the characters of typed text and pastes. Off by default,
	// in which case each character is recorded as 'x': the replay still sees
	// the same number of keystrokes, but not what they spelled.
	RecordText bool
	// ExposeInputValues keeps the values of input fields in recorded trees.
	// Values of sensitive nodes are never kept.
	ExposeInputValues bool
}

// Header is the first line of a recording.
type Header struct {
	Kind      string    `json:"kind"`
	Format    string    `json:"format"`
	Version   int       `json:"version"`
	Limoni    string    `json:"limoni"`
	Revision  string    `json:"revision"`
	GoVersion string    `json:"go"`
	Model     string    `json:"model"`
	Width     uint16    `json:"width"`
	Height    uint16    `json:"height"`
	Created   time.Time `json:"created"`
	Policy    Policy    `json:"policy"`
}

const (
	kindHeader  = "header"
	kindMessage = "msg"
	kindFrame   = "frame"
	kindPanic   = "panic"
	kindTrailer = "trailer"
)

// record is one line after the header.
type record struct {
	Kind string `json:"kind"`
	Step uint64 `json:"step"`

	// Message records.
	Type     string          `json:"type,omitempty"`
	Data     json.RawMessage `json:"data,omitempty"`
	Redacted bool            `json:"redacted,omitempty"`
	// Opaque marks an unregistered application message: its name is known,
	// its content was deliberately not recorded.
	Opaque bool `json:"opaque,omitempty"`

	// Frame records.
	Tree []accessibility.Node `json:"tree,omitempty"`

	// Panic records.
	Value string `json:"value,omitempty"`

	// Trailer.
	Steps  uint64 `json:"steps,omitempty"`
	Frames uint64 `json:"frames,omitempty"`
	SHA256 string `json:"sha256,omitempty"`
}
