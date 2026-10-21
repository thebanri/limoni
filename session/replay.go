package session

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/buffer"
	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/engine"
	"github.com/thebanri/limoni/core/terminal"
)

// ReplayOptions relaxes the checks a replay makes before it trusts a file.
// Each one exists because refusing is the safe default and a person may know
// better — never because the check is optional in general.
type ReplayOptions struct {
	// AllowIncomplete replays a recording with no trailer, which is what a
	// crashed process leaves behind. Its checksum cannot be verified.
	AllowIncomplete bool
	// AllowRevisionMismatch replays a recording made by a different build of
	// the application. The model may have changed since, so a divergence is
	// then as likely to be the change as the bug.
	AllowRevisionMismatch bool
}

// Divergence is the first frame whose replayed tree differs from the recorded
// one.
type Divergence struct {
	Step uint64
	// AfterRedactedInput is set when a redacted keystroke or paste came before
	// this frame. The replay typed 'x' where the user typed something else, so
	// the divergence may be the redaction rather than a bug.
	AfterRedactedInput bool
	// Want and Got are the recorded and replayed trees in screen-reader line
	// mode, which is readable in a test failure.
	Want, Got string
}

func (d *Divergence) String() string {
	note := ""
	if d.AfterRedactedInput {
		note = " (after redacted input — the difference may come from the redaction)"
	}
	return fmt.Sprintf("replay diverged at step %d%s\n--- recorded\n%s\n--- replayed\n%s", d.Step, note, d.Want, d.Got)
}

// Report describes a replay.
type Report struct {
	Header   Header
	Complete bool
	Steps    uint64
	Frames   uint64
	// RedactedSteps counts messages whose content the recording withheld.
	RedactedSteps uint64

	// Divergence is the first frame that did not match, or nil.
	Divergence *Divergence

	// PanicRecorded is set when the original run panicked, at PanicStep with
	// PanicValue; PanicReproduced reports whether the replay panicked at the
	// same step with the same value.
	PanicRecorded   bool
	PanicStep       uint64
	PanicValue      string
	PanicReproduced bool
}

// Verified reports whether the replay matched the recording end to end:
// every frame identical, and a recorded panic, if any, reproduced.
func (r Report) Verified() bool {
	return r.Divergence == nil && (!r.PanicRecorded || r.PanicReproduced)
}

// Replay reads a recording and replays it against a fresh model from newModel.
//
// It returns an error only when the recording cannot be trusted or replayed at
// all — bad checksum, wrong model, unregistered message type. A replay that
// ran and found a difference is not an error; that is the result, in
// Report.Divergence.
func Replay(path string, newModel func() engine.Model, opts ReplayOptions) (Report, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Report{}, fmt.Errorf("session: read recording: %w", err)
	}
	return ReplayBytes(data, newModel, opts)
}

// ReplayBytes is Replay over an in-memory recording.
func ReplayBytes(data []byte, newModel func() engine.Model, opts ReplayOptions) (Report, error) {
	var report Report
	if newModel == nil {
		return report, fmt.Errorf("session: newModel is required")
	}

	header, records, complete, err := parse(data)
	if err != nil {
		return report, err
	}
	report.Header = header
	report.Complete = complete
	if !complete && !opts.AllowIncomplete {
		return report, fmt.Errorf("session: recording is incomplete (no trailer, the process likely crashed); set AllowIncomplete to replay it")
	}

	model := newModel()
	if model == nil {
		return report, fmt.Errorf("session: newModel returned nil")
	}
	if got := modelName(model); got != header.Model {
		return report, fmt.Errorf("session: recording is for model %s, newModel built %s", header.Model, got)
	}
	if _, revision := buildInfo(); revision != header.Revision && !opts.AllowRevisionMismatch {
		return report, fmt.Errorf("session: recording was made by revision %q, this build is %q; set AllowRevisionMismatch to replay anyway", header.Revision, revision)
	}

	// The frame is kept across renders the way a live terminal keeps it, so
	// focus survives from one frame to the next exactly as it did originally.
	buf := buffer.NewBuffer(cell.NewRect(0, 0, header.Width, header.Height))
	focus := terminal.NewFocusManager()
	frame := terminal.NewFrame(buf, focus)

	replayPanic := func(step uint64, fn func()) {
		defer func() {
			if recovered := recover(); recovered != nil {
				if report.PanicRecorded && step == report.PanicStep && fmt.Sprint(recovered) == report.PanicValue {
					report.PanicReproduced = true
				}
				if report.PanicStep == 0 && !report.PanicRecorded {
					// A panic the original run did not have is itself a divergence.
					report.Divergence = &Divergence{Step: step, Want: "(no panic)", Got: fmt.Sprintf("panic: %v", recovered)}
				}
			}
		}()
		fn()
	}

	// Panics are known up front so a replayed panic can be matched to one.
	for _, rec := range records {
		if rec.Kind == kindPanic && !report.PanicRecorded {
			report.PanicRecorded = true
			report.PanicStep = rec.Step
			report.PanicValue = rec.Value
		}
	}

	// Init commands are discarded: their results are already in the stream as
	// the messages they produced.
	replayPanic(0, func() { _ = model.Init() })

	redactedSoFar := false
	for _, rec := range records {
		if report.Divergence != nil {
			break
		}
		switch rec.Kind {
		case kindMessage:
			report.Steps = rec.Step
			if rec.Redacted {
				report.RedactedSteps++
				redactedSoFar = true
			}
			if rec.Opaque {
				return report, fmt.Errorf("session: step %d is an unregistered message type %s whose content was not recorded; register it with session.Register and record again", rec.Step, rec.Type)
			}
			msg, err := decode(rec.Type, rec.Data)
			if err != nil {
				return report, fmt.Errorf("session: step %d: %w", rec.Step, err)
			}
			replayPanic(rec.Step, func() { _ = model.Update(msg) })

		case kindFrame:
			report.Frames++
			var got []accessibility.Node
			replayPanic(rec.Step, func() {
				buf.Clear()
				frame.Reset()
				focus.Clear()
				model.View(frame)
				got = accessibility.Redact(frame.AccessibilityTree(), header.Policy.ExposeInputValues)
			})
			if report.Divergence != nil {
				break
			}
			want, _ := json.Marshal(rec.Tree)
			have, _ := json.Marshal(got)
			if !bytes.Equal(want, have) {
				mode := accessibility.Mode{ScreenReader: true}
				report.Divergence = &Divergence{
					Step:               rec.Step,
					AfterRedactedInput: redactedSoFar,
					Want:               mode.LineMode(rec.Tree),
					Got:                mode.LineMode(got),
				}
			}
		}
	}
	return report, nil
}

// parse splits a recording into its header and records, and verifies the
// checksum when a trailer is present.
func parse(data []byte) (Header, []record, bool, error) {
	var header Header
	reader := bufio.NewReader(bytes.NewReader(data))
	sum := sha256.New()

	first, err := reader.ReadBytes('\n')
	if err != nil {
		return header, nil, false, fmt.Errorf("session: recording has no header")
	}
	if err := json.Unmarshal(first, &header); err != nil || header.Kind != kindHeader {
		return header, nil, false, fmt.Errorf("session: not a recording: first line is not a header")
	}
	if header.Format != FormatName {
		return header, nil, false, fmt.Errorf("session: not a recording: format %q", header.Format)
	}
	if header.Version != FormatVersion {
		return header, nil, false, fmt.Errorf("session: recording format version %d, this build reads version %d", header.Version, FormatVersion)
	}
	sum.Write(first)

	var records []record
	for {
		line, err := reader.ReadBytes('\n')
		if errors.Is(err, io.EOF) {
			if len(bytes.TrimSpace(line)) > 0 {
				// A partial final line: the process died mid-write. Everything
				// before it is intact; the fragment is dropped.
				return header, records, false, nil
			}
			return header, records, false, nil
		}
		if err != nil {
			return header, nil, false, fmt.Errorf("session: read recording: %w", err)
		}
		var rec record
		if err := json.Unmarshal(line, &rec); err != nil {
			return header, nil, false, fmt.Errorf("session: corrupt record after step %d: %w", lastStep(records), err)
		}
		if rec.Kind == kindTrailer {
			if got := hex.EncodeToString(sum.Sum(nil)); !strings.EqualFold(got, rec.SHA256) {
				return header, nil, false, fmt.Errorf("session: checksum mismatch, the recording was modified or damaged after it was written")
			}
			if rest, _ := io.ReadAll(reader); len(bytes.TrimSpace(rest)) > 0 {
				return header, nil, false, fmt.Errorf("session: data after the trailer, the recording was modified")
			}
			return header, records, true, nil
		}
		sum.Write(line)
		records = append(records, rec)
	}
}

func lastStep(records []record) uint64 {
	if len(records) == 0 {
		return 0
	}
	return records[len(records)-1].Step
}
