package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/thebanri/limoni/automation"
	"github.com/thebanri/limoni/core/accessibility"
)

// requestTimeout bounds one round trip to the application when the caller's
// context sets no deadline. A frame is milliseconds; an application that takes
// seconds to answer is stuck, and the agent should hear that rather than hang.
const requestTimeout = 10 * time.Second

// app is a connection to one Limoni application's automation socket.
//
// It speaks the automation wire protocol itself rather than wrapping
// automation.Client, because a tool call has to be cancellable: the client
// blocks on a read with no way to interrupt it.
type app struct {
	socket string
	// settle is how long an input tool waits for the frame its input caused.
	settle time.Duration

	mu     sync.Mutex
	conn   net.Conn
	reader *bufio.Reader
}

// errTransport marks failures of the connection itself, as opposed to the
// application refusing a request.
var errTransport = errors.New("transport")

// errAppGone reports that input was delivered and the application then closed
// its socket, most likely by exiting.
var errAppGone = errors.New("application gone")

// do sends one request. A read-only request is retried once on a fresh
// connection if the old one turns out to be dead — the application may have
// restarted. Input is never retried: if the connection dropped after the
// request was written, the application may already have acted on it, and
// sending it again would click twice.
func (a *app) do(ctx context.Context, req automation.Request, readOnly bool) (automation.Response, error) {
	resp, err := a.roundTrip(ctx, req)
	if err != nil && readOnly && errors.Is(err, errTransport) && ctx.Err() == nil {
		resp, err = a.roundTrip(ctx, req)
	}
	return resp, err
}

func (a *app) roundTrip(ctx context.Context, req automation.Request) (automation.Response, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.conn == nil {
		var d net.Dialer
		conn, err := d.DialContext(ctx, "unix", a.socket)
		if err != nil {
			return automation.Response{}, fmt.Errorf(
				"no Limoni application is listening at %s (%v). The application must be built with -tags limoni_debug and started with limoni.WithAutomation(%q, ...): %w",
				a.socket, err, a.socket, errTransport)
		}
		a.conn, a.reader = conn, bufio.NewReaderSize(conn, 256*1024)
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(requestTimeout)
	}
	_ = a.conn.SetDeadline(deadline)
	// Cancelling the tool call unblocks the read by expiring the deadline now.
	stop := context.AfterFunc(ctx, func() { _ = a.conn.SetDeadline(time.Unix(1, 0)) })
	defer stop()

	payload, err := json.Marshal(req)
	if err != nil {
		return automation.Response{}, err
	}
	if _, err := a.conn.Write(append(payload, '\n')); err != nil {
		a.dropLocked()
		return automation.Response{}, a.transportError(ctx, "send", err)
	}
	line, err := a.reader.ReadBytes('\n')
	if err != nil {
		a.dropLocked()
		return automation.Response{}, a.transportError(ctx, "read", err)
	}

	var resp automation.Response
	if err := json.Unmarshal(line, &resp); err != nil {
		a.dropLocked()
		return automation.Response{}, fmt.Errorf("decode response from the application: %v: %w", err, errTransport)
	}
	if !resp.OK {
		return resp, errors.New(resp.Err)
	}
	return resp, nil
}

func (a *app) transportError(ctx context.Context, what string, err error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return fmt.Errorf("%s to the application at %s failed (%v); it may have exited: %w", what, a.socket, err, errTransport)
}

func (a *app) dropLocked() {
	if a.conn != nil {
		_ = a.conn.Close()
		a.conn, a.reader = nil, nil
	}
}

func (a *app) close() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.dropLocked()
}

func (a *app) tree(ctx context.Context) ([]accessibility.Node, error) {
	resp, err := a.do(ctx, automation.Request{Op: automation.OpTree}, true)
	return resp.Nodes, err
}

// act sends an input request and waits for the frames it causes, returning the
// tree after them.
//
// Input is asynchronous: the socket acknowledges the event when it is queued,
// and the application draws its response a moment later — for typed text, a
// frame per character. Returning the tree straight away would show the agent
// the screen before its own click, the most confusing thing a tool could do.
// So this waits for the tree to change and then stay still for a quiet period,
// or for the settle time to pass. If nothing changed at all it says so,
// because "no visible effect" is itself information.
func (a *app) act(ctx context.Context, req automation.Request) (automation.Response, []accessibility.Node, bool, error) {
	before, err := a.tree(ctx)
	if err != nil {
		return automation.Response{}, nil, false, err
	}
	beforeJSON, _ := json.Marshal(before)

	resp, err := a.do(ctx, req, false)
	if err != nil {
		return resp, nil, false, err
	}

	const (
		poll  = 10 * time.Millisecond
		quiet = 50 * time.Millisecond
	)
	deadline := time.Now().Add(a.settle)
	latest, latestJSON := before, beforeJSON
	changed := false
	var stableSince time.Time
	for time.Now().Before(deadline) {
		if err := sleep(ctx, poll); err != nil {
			return resp, nil, false, err
		}
		nodes, err := a.tree(ctx)
		if errors.Is(err, errTransport) {
			// The input was delivered; the application went away after it.
			// That is an outcome to report — an Esc or a Quit button does
			// exactly this — not a failure of the call.
			return resp, nil, false, errAppGone
		}
		if err != nil {
			return resp, nil, false, err
		}
		current, _ := json.Marshal(nodes)
		if string(current) != string(latestJSON) {
			latest, latestJSON = nodes, current
			changed = changed || string(current) != string(beforeJSON)
			stableSince = time.Now()
			continue
		}
		if changed && time.Since(stableSince) >= quiet {
			break
		}
	}
	return resp, latest, string(latestJSON) != string(beforeJSON), nil
}

// waitFor polls until the selector matches, the timeout passes, or the call is
// cancelled.
func (a *app) waitFor(ctx context.Context, sel automation.Selector, timeout time.Duration) ([]accessibility.Node, error) {
	deadline := time.Now().Add(timeout)
	for {
		resp, err := a.do(ctx, automation.Request{Op: automation.OpFind, Selector: sel}, true)
		if err != nil && errors.Is(err, errTransport) {
			return nil, err
		}
		if err == nil && len(resp.Nodes) > 0 {
			return resp.Nodes, nil
		}
		if !time.Now().Before(deadline) {
			return nil, fmt.Errorf("nothing matched %s within %s", sel, timeout)
		}
		if err := sleep(ctx, 20*time.Millisecond); err != nil {
			return nil, err
		}
	}
}

func sleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
