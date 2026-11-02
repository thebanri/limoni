package engine

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/thebanri/limoni/core/driver"
	"github.com/thebanri/limoni/core/terminal"
	"github.com/thebanri/limoni/widgets"
)

type commitModel struct{}

func (commitModel) Init() []Cmd { return nil }
func (commitModel) Update(msg Msg) UpdateResult {
	if n, ok := msg.(int); ok {
		// Two per Update, to check the order inside one as well as across.
		return UpdateResult{Commit: []widgets.Widget{
			widgets.NewLabel("msg" + strconv.Itoa(n) + "a."),
			widgets.NewLabel("msg" + strconv.Itoa(n) + "b."),
		}}
	}
	return UpdateResult{}
}
func (commitModel) View(*terminal.Frame) {}

// Every commit reaches the scrollback through RunTerminal, none is dropped,
// and they arrive in the order Update asked for them.
func TestUpdateCommitsReachTheTerminalInOrder(t *testing.T) {
	t.Setenv("LIMONI_PROBE", "0")
	io := driver.NewMemoryTerminalIO(nil, 20, 5)
	b := driver.NewPortableBackend(io)
	b.SetInline(2)
	term, err := terminal.New(b)
	if err != nil {
		t.Fatal(err)
	}
	term.SetInline(2)

	const n = 200
	p := New(WithModel(commitModel{}))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- p.RunTerminal(ctx, term, b) }()
	for i := range n {
		if err := p.Send(ctx, i); err != nil {
			t.Fatal(err)
		}
	}

	var want []string
	for i := range n {
		want = append(want, "msg"+strconv.Itoa(i)+"a.", "msg"+strconv.Itoa(i)+"b.")
	}
	inOrder := func() (int, bool) {
		out, at := string(io.Output()), 0
		for k, w := range want {
			i := strings.Index(out[at:], w)
			if i < 0 {
				return k, false
			}
			at += i + len(w)
		}
		return len(want), true
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		k, ok := inOrder()
		if ok {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("commit %d (%s) missing or out of order", k, want[k])
		}
		time.Sleep(5 * time.Millisecond)
	}
	cancel()
	<-done
}
