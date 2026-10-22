//go:build linux

package main

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
	"sync/atomic"
)

// An EI client: the receiving end of libei's protocol, which is how a
// compositor hands captured input to the application that asked for it
// (the InputCapture portal's ConnectToEIS, portal_linux.go). Only what the
// mouse needs is spoken: the handshake, a seat bound to its pointer and
// buttons, relative motion, buttons, and pings. Messages are a 16-byte
// header (object id u64, length u32 counting the header, opcode u32) and
// 4-byte-aligned arguments, in the host's byte order; see libei's
// proto/protocol.xml (1.6.0) for every opcode below.
type eiClient struct {
	rw    io.ReadWriter
	types map[uint64]string // object id → interface
	masks map[string]uint64 // seat capability masks by interface
	dx    atomic.Int64      // motion since the last drain, in 1/256 of a unit
	dy    atomic.Int64
	press chan pointerEvent
	out   []byte
}

const (
	btnLeft  = 0x110
	btnRight = 0x111
)

// The interfaces this client speaks, all at version 1.
var eiInterfaces = []string{"ei_connection", "ei_callback", "ei_pingpong", "ei_seat", "ei_device", "ei_pointer", "ei_button"}

func newEIClient(rw io.ReadWriter) *eiClient {
	return &eiClient{rw: rw, types: map[uint64]string{0: "ei_handshake"}, masks: map[string]uint64{}, press: make(chan pointerEvent, 64)}
}

// ── writing ──────────────────────────────────────────────────────────────

func (c *eiClient) begin(object uint64, opcode uint32) {
	c.out = binary.NativeEndian.AppendUint64(c.out[:0], object)
	c.out = binary.NativeEndian.AppendUint32(c.out, 0) // the length, set by send
	c.out = binary.NativeEndian.AppendUint32(c.out, opcode)
}

func (c *eiClient) u32(v uint32) { c.out = binary.NativeEndian.AppendUint32(c.out, v) }
func (c *eiClient) u64(v uint64) { c.out = binary.NativeEndian.AppendUint64(c.out, v) }

func (c *eiClient) str(s string) {
	c.u32(uint32(len(s) + 1))
	c.out = append(c.out, s...)
	c.out = append(c.out, 0)
	for len(c.out)%4 != 0 {
		c.out = append(c.out, 0)
	}
}

func (c *eiClient) send() error {
	binary.NativeEndian.PutUint32(c.out[8:], uint32(len(c.out)))
	_, err := c.rw.Write(c.out)
	return err
}

// ── reading ──────────────────────────────────────────────────────────────

// args reads a message's arguments in order.
type args struct {
	b   []byte
	bad bool
}

func (a *args) u32() uint32 {
	if len(a.b) < 4 {
		a.bad = true
		return 0
	}
	v := binary.NativeEndian.Uint32(a.b)
	a.b = a.b[4:]
	return v
}

func (a *args) u64() uint64 {
	if len(a.b) < 8 {
		a.bad = true
		return 0
	}
	v := binary.NativeEndian.Uint64(a.b)
	a.b = a.b[8:]
	return v
}

func (a *args) f32() float32 { return math.Float32frombits(a.u32()) }

func (a *args) str() string {
	n := int(a.u32())
	padded := (n + 3) &^ 3
	if n == 0 || len(a.b) < padded {
		a.bad = a.bad || n != 0
		return ""
	}
	s := string(a.b[:n-1])
	a.b = a.b[padded:]
	return s
}

// run reads and answers messages until the connection ends.
func (c *eiClient) run() error {
	head := make([]byte, 16)
	var body []byte
	for {
		if _, err := io.ReadFull(c.rw, head); err != nil {
			return err
		}
		object := binary.NativeEndian.Uint64(head)
		n := int(binary.NativeEndian.Uint32(head[8:]))
		opcode := binary.NativeEndian.Uint32(head[12:])
		if n < 16 || n > 1<<16 {
			return errors.New("ei: a message of impossible length")
		}
		if cap(body) < n-16 {
			body = make([]byte, n-16)
		}
		body = body[:n-16]
		if _, err := io.ReadFull(c.rw, body); err != nil {
			return err
		}
		if err := c.handle(object, opcode, &args{b: body}); err != nil {
			return err
		}
	}
}

func (c *eiClient) handle(object uint64, opcode uint32, a *args) error {
	switch c.types[object] {
	case "ei_handshake":
		switch opcode {
		case 0: // handshake_version: answer it, then say who this is
			c.begin(0, 0)
			c.u32(1)
			if err := c.send(); err != nil {
				return err
			}
			c.begin(0, 2) // context_type: receiver
			c.u32(1)
			if err := c.send(); err != nil {
				return err
			}
			c.begin(0, 3) // name
			c.str("Mark & Sweep")
			if err := c.send(); err != nil {
				return err
			}
			for _, name := range eiInterfaces {
				c.begin(0, 4) // interface_version
				c.str(name)
				c.u32(1)
				if err := c.send(); err != nil {
					return err
				}
			}
			c.begin(0, 1) // finish
			return c.send()
		case 2: // connection
			a.u32()
			c.types[a.u64()] = "ei_connection"
		}
	case "ei_connection":
		switch opcode {
		case 0: // disconnected
			return io.EOF
		case 1: // seat
			c.types[a.u64()] = "ei_seat"
		case 3: // ping: answer at once
			ping := a.u64()
			c.begin(ping, 0) // ei_pingpong.done
			c.u64(0)
			return c.send()
		}
	case "ei_seat":
		switch opcode {
		case 0: // destroyed
			delete(c.types, object)
		case 2: // capability
			mask := a.u64()
			c.masks[a.str()] = mask
		case 3: // done: bind to the pointer and its buttons
			c.begin(object, 1)
			c.u64(c.masks["ei_pointer"] | c.masks["ei_button"])
			return c.send()
		case 4: // device
			c.types[a.u64()] = "ei_device"
		}
	case "ei_device":
		switch opcode {
		case 0:
			delete(c.types, object)
		case 5: // interface: a new object of the named interface
			id := a.u64()
			c.types[id] = a.str()
		}
	case "ei_pointer":
		if opcode == 1 { // motion_relative
			x, y := a.f32(), a.f32()
			c.dx.Add(int64(x * 256))
			c.dy.Add(int64(y * 256))
		}
	case "ei_button":
		if opcode == 1 { // button
			code, state := a.u32(), a.u32()
			button := -1
			switch code {
			case btnLeft:
				button = 0
			case btnRight:
				button = 2
			}
			if button >= 0 {
				select {
				case c.press <- pointerEvent{button, state == 1}:
				default:
				}
			}
		}
	}
	if a.bad {
		return errors.New("ei: a message shorter than its arguments")
	}
	return nil
}

// motion takes the motion since the last call, in pixels.
func (c *eiClient) motion() (float64, float64) {
	return float64(c.dx.Swap(0)) / 256, float64(c.dy.Swap(0)) / 256
}
