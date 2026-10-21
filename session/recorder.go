package session

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"os"
	"reflect"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/core/engine"
)

// Recorder writes a session recording. It implements engine.Observer; attach
// it with engine.WithObserver, and Close it when the program ends.
type Recorder struct {
	mu     sync.Mutex
	file   *os.File
	out    *bufio.Writer
	sum    hash.Hash
	policy Policy
	err    error
	closed bool

	steps  uint64
	frames uint64

	lastFrameStep uint64
	lastFrameJSON []byte
	haveFrame     bool

	// focusUnsafe is true while a sensitive field might have focus. It starts
	// true, because before the first frame nothing is known, and it is set
	// again by any input that can move focus, until a frame shows where focus
	// landed. While it is set, typed text is redacted even under RecordText.
	focusUnsafe bool
}

// Create starts a recording for model at the given viewport size.
//
// The file is created with 0600 permissions and must not already exist:
// silently overwriting an earlier recording would destroy the one artefact a
// bug report depends on.
func Create(path string, model engine.Model, width, height uint16, policy Policy) (*Recorder, error) {
	if model == nil {
		return nil, fmt.Errorf("session: model is required")
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return nil, fmt.Errorf("session: create recording: %w", err)
	}
	r := &Recorder{
		file:        file,
		out:         bufio.NewWriterSize(file, 64*1024),
		sum:         sha256.New(),
		policy:      policy,
		focusUnsafe: true,
	}
	limoniVersion, revision := buildInfo()
	r.writeLine(Header{
		Kind:      kindHeader,
		Format:    FormatName,
		Version:   FormatVersion,
		Limoni:    limoniVersion,
		Revision:  revision,
		GoVersion: runtime.Version(),
		Model:     modelName(model),
		Width:     width,
		Height:    height,
		Created:   time.Now().UTC(),
		Policy:    policy,
	})
	if r.err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return nil, r.err
	}
	return r, nil
}

// Message implements engine.Observer.
func (r *Recorder) Message(step uint64, msg engine.Msg) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return
	}
	r.steps = step

	rec := record{Kind: kindMessage, Step: step}
	if msg == nil {
		rec.Type = nilMessage
		r.writeLine(rec)
		return
	}

	msg, rec.Redacted = r.redactInput(msg)
	name := typeName(reflect.TypeOf(msg))
	rt, registered := lookup(name)
	rec.Type = name
	if !registered {
		rec.Opaque = true
		r.writeLine(rec)
		return
	}
	if rt.redact != nil {
		msg = rt.redact(msg)
		rec.Redacted = true
	}
	data, err := json.Marshal(msg)
	if err != nil {
		r.fail(fmt.Errorf("session: encode %s at step %d: %w", name, step, err))
		return
	}
	rec.Data = data
	r.writeLine(rec)
}

// Frame implements engine.Observer.
func (r *Recorder) Frame(step uint64, tree []accessibility.Node) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return
	}

	redacted := accessibility.Redact(tree, r.policy.ExposeInputValues)
	encoded, err := json.Marshal(redacted)
	if err != nil {
		r.fail(fmt.Errorf("session: encode frame at step %d: %w", step, err))
		return
	}

	// Focus is known again: unsafe only if the focused node is sensitive.
	r.focusUnsafe = focusedSensitive(tree)

	// A program redraws on a ticker whether or not anything changed. Only a
	// frame at a new step, or a tree that differs, is worth a line.
	if r.haveFrame && step == r.lastFrameStep && bytes.Equal(encoded, r.lastFrameJSON) {
		return
	}
	r.haveFrame = true
	r.lastFrameStep = step
	r.lastFrameJSON = encoded
	r.frames++
	r.writeLine(record{Kind: kindFrame, Step: step, Tree: redacted})
}

// Panic implements engine.Observer.
func (r *Recorder) Panic(step uint64, value any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return
	}
	r.writeLine(record{Kind: kindPanic, Step: step, Value: fmt.Sprint(value)})
}

// Close writes the trailer, which carries a checksum of everything before it,
// and closes the file. It returns the first error the recorder hit.
//
// A recording that never reaches Close — the process crashed — has no trailer.
// It is still readable, and replaying it requires ReplayOptions.AllowIncomplete.
func (r *Recorder) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return r.err
	}
	r.closed = true
	sum := hex.EncodeToString(r.sum.Sum(nil))
	// The trailer is not part of its own checksum.
	if r.err == nil {
		line, err := json.Marshal(record{Kind: kindTrailer, Steps: r.steps, Frames: r.frames, SHA256: sum})
		if err != nil {
			r.err = err
		} else if _, err := r.out.Write(append(line, '\n')); err != nil {
			r.err = err
		}
	}
	if err := r.out.Flush(); err != nil && r.err == nil {
		r.err = err
	}
	if err := r.file.Close(); err != nil && r.err == nil {
		r.err = err
	}
	return r.err
}

// redactInput applies the text policy to input messages.
func (r *Recorder) redactInput(msg engine.Msg) (engine.Msg, bool) {
	var (
		out      engine.Msg = msg
		redacted bool
		plain    bool // a character key with no modifier: typing, not navigation
	)
	switch m := msg.(type) {
	case engine.KeyPressMsg:
		plain = isPlainText(m.Key)
		if plain && r.hideText() {
			m.Key.Ch = 'x'
			out, redacted = m, true
		}
	case engine.KeyReleaseMsg:
		plain = isPlainText(m.Key)
		if plain && r.hideText() {
			m.Key.Ch = 'x'
			out, redacted = m, true
		}
	case engine.PasteMsg:
		if r.hideText() {
			m.Text = strings.Repeat("x", utf8.RuneCountInString(m.Text))
			out, redacted = m, true
		}
	}
	// Focus can only move while Update handles a message, and any message can
	// move it: a Tab, a click, or a command result that makes the application
	// focus a password field. Only typing into the field that already has focus
	// cannot. So everything except plain typing makes focus unknown until the
	// next frame shows where it is — and text typed in that window is redacted.
	// A list of "focus-moving keys" would miss the command result.
	if !plain {
		r.focusUnsafe = true
	}
	return out, redacted
}

// hideText reports whether typed characters must not be written.
func (r *Recorder) hideText() bool {
	return !r.policy.RecordText || r.focusUnsafe
}

// isPlainText reports whether a key is a character typed without Ctrl or Alt.
// Shortcuts are kept in the recording because they carry the interaction —
// a replay must still save on Ctrl+S.
func isPlainText(key driver.KeyEvent) bool {
	return key.Type == driver.KeyRune && !key.Ctrl && !key.Alt
}

func (r *Recorder) writeLine(v any) {
	if r.err != nil {
		return
	}
	line, err := json.Marshal(v)
	if err != nil {
		r.fail(err)
		return
	}
	line = append(line, '\n')
	if _, err := r.out.Write(line); err != nil {
		r.fail(err)
		return
	}
	r.sum.Write(line)
	// Flushed per line: a crash must leave everything up to the crash on disk,
	// because that is the part of the recording that matters.
	if err := r.out.Flush(); err != nil {
		r.fail(err)
	}
}

func (r *Recorder) fail(err error) {
	if r.err == nil {
		r.err = err
	}
}

func focusedSensitive(nodes []accessibility.Node) bool {
	for _, node := range nodes {
		if node.State&accessibility.StateFocused != 0 && node.State&accessibility.StateSensitive != 0 {
			return true
		}
		if focusedSensitive(node.Children) {
			return true
		}
	}
	return false
}

func modelName(model engine.Model) string {
	if name := typeName(reflect.TypeOf(model)); name != "" {
		return name
	}
	return fmt.Sprintf("%T", model)
}

// buildInfo reports the Limoni module version and the application's VCS
// revision, both read from the binary rather than typed by hand.
func buildInfo() (limoni, revision string) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown", ""
	}
	limoni = "(devel)"
	if info.Main.Path == "github.com/thebanri/limoni" {
		limoni = info.Main.Version
	}
	for _, dep := range info.Deps {
		if dep.Path == "github.com/thebanri/limoni" {
			limoni = dep.Version
		}
	}
	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" {
			revision = setting.Value
		}
	}
	return limoni, revision
}
