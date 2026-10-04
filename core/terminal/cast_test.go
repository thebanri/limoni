package terminal_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"github.com/thebanri/limoni/core/cell"
	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/widgets"
)

// A recording is an asciicast v2 file: a header with the size, then the
// frames as output events, then a resize event when the window changes.
func TestRecordCastWritesAsciicast(t *testing.T) {
	t.Setenv("LIMONI_PROBE", "0")
	io := &resizableIO{w: 30, h: 4}
	b := driver.NewPortableBackend(io)
	if err := b.Setup(); err != nil {
		t.Fatal(err)
	}
	term, err := terminal.New(b)
	if err != nil {
		t.Fatal(err)
	}
	var cast bytes.Buffer
	if err := term.RecordCast(&cast); err != nil {
		t.Fatal(err)
	}
	draw := func(text string) {
		if err := term.Draw(func(f *terminal.Frame) {
			f.RenderWidget(widgets.NewLabel(text), cell.NewRect(0, 0, 20, 1))
		}); err != nil {
			t.Fatal(err)
		}
	}
	draw("first frame")
	draw("second frame")
	io.resize(40, 6) // Windows asks the IO for its size,
	b.SetSize(40, 6) // Unix the backend
	draw("after resize")
	if err := term.Close(); err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(cast.String()), "\n")
	var header struct{ Version, Width, Height int }
	if err := json.Unmarshal([]byte(lines[0]), &header); err != nil || header.Version != 2 || header.Width != 30 || header.Height != 4 {
		t.Fatalf("header %s (%v)", lines[0], err)
	}
	var output strings.Builder
	resized := false
	last := -1.0
	for _, line := range lines[1:] {
		var ev []any
		if err := json.Unmarshal([]byte(line), &ev); err != nil || len(ev) != 3 {
			t.Fatalf("event %s (%v)", line, err)
		}
		at := ev[0].(float64)
		if at < last {
			t.Errorf("time went backwards at %s", line)
		}
		last = at
		switch ev[1] {
		case "o":
			output.WriteString(ev[2].(string))
		case "r":
			resized = ev[2] == "40x6"
		}
	}
	for _, want := range []string{"first", "second frame", "after", "resize"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("the recording never shows %q: %q", want, output.String())
		}
	}
	if !resized {
		t.Errorf("no resize event to 40x6")
	}
}

// resizableIO is an in-memory terminal whose size the test changes.
type resizableIO struct {
	mu   sync.Mutex
	w, h uint16
	out  bytes.Buffer
}

func (r *resizableIO) Read([]byte) (int, error) { select {} }
func (r *resizableIO) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.out.Write(p)
}
func (r *resizableIO) Size() (uint16, uint16, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.w, r.h, nil
}
func (r *resizableIO) resize(w, h uint16) {
	r.mu.Lock()
	r.w, r.h = w, h
	r.mu.Unlock()
}
