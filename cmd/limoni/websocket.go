package main

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
)

// A WebSocket server connection (RFC 6455), just enough for a terminal: the
// browser sends keystrokes and resizes, the server sends the program's
// output. The standard library has no WebSocket, and the module takes no
// dependency for one command.

const (
	wsContinuation = 0x0
	wsText         = 0x1
	wsBinary       = 0x2
	wsClose        = 0x8
	wsPing         = 0x9
	wsPong         = 0xA

	// wsMaxMessage bounds what a client may send in one message. Keystrokes
	// and pastes are small; this keeps a hostile client from making the
	// server allocate without limit.
	wsMaxMessage = 1 << 20
)

const wsGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

var errWSClosed = errors.New("websocket closed")

type wsConn struct {
	conn net.Conn
	r    *bufio.Reader
	wmu  sync.Mutex // one writer at a time: output, pongs and the close
}

// wsAccept answers the opening handshake and takes over the connection.
func wsAccept(w http.ResponseWriter, r *http.Request) (*wsConn, error) {
	if r.Method != http.MethodGet ||
		!headerHasToken(r.Header, "Connection", "upgrade") ||
		!headerHasToken(r.Header, "Upgrade", "websocket") {
		http.Error(w, "expected a WebSocket upgrade", http.StatusBadRequest)
		return nil, errors.New("not a websocket upgrade")
	}
	if r.Header.Get("Sec-WebSocket-Version") != "13" {
		w.Header().Set("Sec-WebSocket-Version", "13")
		http.Error(w, "unsupported WebSocket version", http.StatusUpgradeRequired)
		return nil, errors.New("unsupported websocket version")
	}
	key := r.Header.Get("Sec-WebSocket-Key")
	if decoded, err := base64.StdEncoding.DecodeString(key); err != nil || len(decoded) != 16 {
		http.Error(w, "bad Sec-WebSocket-Key", http.StatusBadRequest)
		return nil, errors.New("bad websocket key")
	}
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "connection cannot be upgraded", http.StatusInternalServerError)
		return nil, errors.New("response writer cannot hijack")
	}
	conn, rw, err := hj.Hijack()
	if err != nil {
		return nil, err
	}
	sum := sha1.Sum([]byte(key + wsGUID))
	_, err = fmt.Fprintf(rw, "HTTP/1.1 101 Switching Protocols\r\n"+
		"Upgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: %s\r\n\r\n",
		base64.StdEncoding.EncodeToString(sum[:]))
	if err == nil {
		err = rw.Flush()
	}
	if err != nil {
		conn.Close()
		return nil, err
	}
	return &wsConn{conn: conn, r: rw.Reader}, nil
}

func headerHasToken(h http.Header, name, token string) bool {
	for _, v := range h.Values(name) {
		for _, part := range strings.Split(v, ",") {
			if strings.EqualFold(strings.TrimSpace(part), token) {
				return true
			}
		}
	}
	return false
}

// ReadMessage returns the next text or binary message, answering pings and
// joining fragments on the way. A close from the client is errWSClosed.
func (c *wsConn) ReadMessage() (opcode byte, payload []byte, err error) {
	var message []byte
	var messageOp byte
	for {
		fin, op, data, err := c.readFrame()
		if err != nil {
			return 0, nil, err
		}
		switch op {
		case wsPing:
			if err := c.writeFrame(wsPong, data); err != nil {
				return 0, nil, err
			}
			continue
		case wsPong:
			continue
		case wsClose:
			_ = c.writeFrame(wsClose, closePayload(data))
			return 0, nil, errWSClosed
		case wsText, wsBinary:
			if messageOp != 0 {
				return 0, nil, errors.New("websocket: new message inside a fragmented one")
			}
			messageOp = op
		case wsContinuation:
			if messageOp == 0 {
				return 0, nil, errors.New("websocket: continuation without a message")
			}
		default:
			return 0, nil, fmt.Errorf("websocket: unknown opcode %#x", op)
		}
		if len(message)+len(data) > wsMaxMessage {
			return 0, nil, errors.New("websocket: message too large")
		}
		message = append(message, data...)
		if fin {
			return messageOp, message, nil
		}
	}
}

// closePayload echoes the status code of a client's close, as the protocol
// asks, and nothing else.
func closePayload(data []byte) []byte {
	if len(data) >= 2 {
		return data[:2]
	}
	return nil
}

func (c *wsConn) readFrame() (fin bool, opcode byte, payload []byte, err error) {
	var head [2]byte
	if _, err = io.ReadFull(c.r, head[:]); err != nil {
		return
	}
	fin = head[0]&0x80 != 0
	if head[0]&0x70 != 0 {
		return false, 0, nil, errors.New("websocket: reserved bits set")
	}
	opcode = head[0] & 0x0F
	if head[1]&0x80 == 0 {
		return false, 0, nil, errors.New("websocket: client frame not masked")
	}
	length := uint64(head[1] & 0x7F)
	switch length {
	case 126:
		var ext [2]byte
		if _, err = io.ReadFull(c.r, ext[:]); err != nil {
			return
		}
		length = uint64(binary.BigEndian.Uint16(ext[:]))
	case 127:
		var ext [8]byte
		if _, err = io.ReadFull(c.r, ext[:]); err != nil {
			return
		}
		length = binary.BigEndian.Uint64(ext[:])
	}
	if opcode >= wsClose && (length > 125 || !fin) {
		return false, 0, nil, errors.New("websocket: bad control frame")
	}
	if length > wsMaxMessage {
		return false, 0, nil, errors.New("websocket: frame too large")
	}
	var mask [4]byte
	if _, err = io.ReadFull(c.r, mask[:]); err != nil {
		return
	}
	payload = make([]byte, length)
	if _, err = io.ReadFull(c.r, payload); err != nil {
		return
	}
	for i := range payload {
		payload[i] ^= mask[i%4]
	}
	return fin, opcode, payload, nil
}

// WriteBinary sends one binary message.
func (c *wsConn) WriteBinary(p []byte) error { return c.writeFrame(wsBinary, p) }

// Close sends a close frame with status code and closes the connection.
func (c *wsConn) Close(code uint16) error {
	var payload [2]byte
	binary.BigEndian.PutUint16(payload[:], code)
	_ = c.writeFrame(wsClose, payload[:])
	return c.conn.Close()
}

func (c *wsConn) writeFrame(opcode byte, payload []byte) error {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	var head [10]byte
	head[0] = 0x80 | opcode
	n := 2
	switch l := len(payload); {
	case l <= 125:
		head[1] = byte(l)
	case l <= 0xFFFF:
		head[1] = 126
		binary.BigEndian.PutUint16(head[2:], uint16(l))
		n = 4
	default:
		head[1] = 127
		binary.BigEndian.PutUint64(head[2:], uint64(l))
		n = 10
	}
	if _, err := c.conn.Write(head[:n]); err != nil {
		return err
	}
	_, err := c.conn.Write(payload)
	return err
}
