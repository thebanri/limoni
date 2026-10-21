package automation

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/thebanri/limoni/core/accessibility"
)

// Client drives a Limoni application over its automation socket.
//
// It is safe for concurrent use: requests are serialised, because the protocol
// is one response per request on a single connection.
type Client struct {
	mu      sync.Mutex
	conn    net.Conn
	reader  *bufio.Reader
	encoder *json.Encoder
}

// Dial connects to an application's automation socket.
func Dial(socketPath string) (*Client, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("automation: dial %s: %w", socketPath, err)
	}
	return &Client{
		conn:    conn,
		reader:  bufio.NewReaderSize(conn, 64*1024),
		encoder: json.NewEncoder(conn),
	}, nil
}

// Close releases the connection.
func (c *Client) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

// Do sends one request and returns its response.
func (c *Client) Do(req Request) (Response, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.encoder.Encode(req); err != nil {
		return Response{}, fmt.Errorf("automation: send %s: %w", req.Op, err)
	}
	line, err := c.reader.ReadBytes('\n')
	if err != nil {
		return Response{}, fmt.Errorf("automation: read %s response: %w", req.Op, err)
	}
	var resp Response
	if err := json.Unmarshal(line, &resp); err != nil {
		return Response{}, fmt.Errorf("automation: decode %s response: %w", req.Op, err)
	}
	if !resp.OK {
		return resp, errors.New(resp.Err)
	}
	return resp, nil
}

// Hello returns the protocol version and current viewport size.
func (c *Client) Hello() (version int, width, height uint16, err error) {
	resp, err := c.Do(Request{Op: OpHello})
	if err != nil {
		return 0, 0, 0, err
	}
	return resp.Version, resp.Width, resp.Height, nil
}

// Tree returns the semantic tree of the most recent frame.
func (c *Client) Tree() ([]accessibility.Node, error) {
	resp, err := c.Do(Request{Op: OpTree})
	return resp.Nodes, err
}

// Screen returns the rendered text grid.
func (c *Client) Screen() (string, error) {
	resp, err := c.Do(Request{Op: OpSnapshot})
	return resp.Snapshot, err
}

// Focused returns the focused widget ID.
func (c *Client) Focused() (string, error) {
	resp, err := c.Do(Request{Op: OpFocused})
	return resp.Focused, err
}

// Find returns every node matching the selector.
func (c *Client) Find(sel Selector) ([]accessibility.Node, error) {
	resp, err := c.Do(Request{Op: OpFind, Selector: sel})
	return resp.Nodes, err
}

// Click resolves the selector and clicks the centre of the matching node.
func (c *Client) Click(sel Selector) error {
	_, err := c.Do(Request{Op: OpClick, Selector: sel})
	return err
}

// Key synthesises a key press. Name is a single character or one of the names
// listed on Request.Key.
func (c *Client) Key(name string) error {
	_, err := c.Do(Request{Op: OpKey, Key: name})
	return err
}

// Type synthesises a run of character key presses.
func (c *Client) Type(text string) error {
	_, err := c.Do(Request{Op: OpText, Text: text})
	return err
}

// WaitFor polls until the selector matches at least one node, or the timeout
// expires.
//
// A frame is rendered in response to input, so a command's effect is not
// visible the instant the command returns. Polling a selector is the honest
// way to wait for it; sleeping a fixed duration is what makes TUI tests flaky.
func (c *Client) WaitFor(sel Selector, timeout time.Duration) (accessibility.Node, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for {
		nodes, err := c.Find(sel)
		if err != nil {
			lastErr = err
		} else if len(nodes) > 0 {
			return nodes[0], nil
		}
		if time.Now().After(deadline) {
			if lastErr != nil {
				return accessibility.Node{}, fmt.Errorf(
					"automation: timed out after %s waiting for %s: %w", timeout, sel, lastErr)
			}
			return accessibility.Node{}, fmt.Errorf(
				"automation: timed out after %s waiting for %s", timeout, sel)
		}
		time.Sleep(5 * time.Millisecond)
	}
}
