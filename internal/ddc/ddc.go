// Package ddc speaks the DDC/CI protocol used to control display settings.
package ddc

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/utkudarilmaz/monctl/internal/bus"
)

// Errors returned by Client.
var (
	ErrNoReply     = errors.New("ddc: display did not answer, it may be asleep or showing another input")
	ErrChecksum    = errors.New("ddc: reply checksum mismatch")
	ErrBadReply    = errors.New("ddc: unexpected reply")
	ErrUnsupported = errors.New("ddc: display does not support this feature")
)

const (
	hostAddr    = 0x51
	displayAddr = 0x6e
	replySeed   = 0x50

	opGetVCP      = 0x01
	opGetVCPReply = 0x02
	opSetVCP      = 0x03
	opCaps        = 0xf3
	opCapsReply   = 0xe3

	getVCPReplySize  = 11
	capsFragmentMax  = 32
	capsReplySize    = capsFragmentMax + 6
	capsMaxLength    = 8192
	capsMaxFragments = 512
	waitBeforeRead   = 40 * time.Millisecond
	waitAfterWrite   = 50 * time.Millisecond
	waitBetweenTries = 100 * time.Millisecond
	maxAttempts      = 3
)

// Value is the state of one VCP feature.
type Value struct {
	Current uint16
	Max     uint16
}

// Client sends DDC/CI commands to one display.
type Client struct {
	bus   bus.Bus
	sleep func(time.Duration)
}

// Option changes how a Client behaves.
type Option func(*Client)

// WithSleep replaces the function used to wait between messages.
func WithSleep(sleep func(time.Duration)) Option {
	return func(c *Client) { c.sleep = sleep }
}

// New returns a Client for the display behind b.
func New(b bus.Bus, opts ...Option) *Client {
	c := &Client{bus: b, sleep: time.Sleep}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// GetVCP reads the current and maximum value of a feature.
func (c *Client) GetVCP(code byte) (Value, error) {
	payload, err := c.exchange([]byte{opGetVCP, code}, getVCPReplySize, func(p []byte) error {
		if len(p) != 8 || p[0] != opGetVCPReply || p[2] != code {
			return ErrBadReply
		}
		return nil
	})
	if err != nil {
		return Value{}, fmt.Errorf("get vcp %#02x: %w", code, err)
	}
	if payload[1] != 0 {
		return Value{}, fmt.Errorf("get vcp %#02x: %w", code, ErrUnsupported)
	}
	return Value{
		Max:     uint16(payload[4])<<8 | uint16(payload[5]),
		Current: uint16(payload[6])<<8 | uint16(payload[7]),
	}, nil
}

// SetVCP writes a new value for a feature. The display sends no reply.
func (c *Client) SetVCP(code byte, value uint16) error {
	request := encode([]byte{opSetVCP, code, byte(value >> 8), byte(value)})
	var err error
	for try := range maxAttempts {
		if try > 0 {
			c.sleep(waitBetweenTries)
		}
		if err = c.bus.Write(bus.AddrDDC, request); err == nil {
			c.sleep(waitAfterWrite)
			return nil
		}
	}
	return fmt.Errorf("set vcp %#02x: %w", code, err)
}

// Capabilities reads the capabilities string, which lists the features and values a display accepts.
func (c *Client) Capabilities() (string, error) {
	var out []byte
	for range capsMaxFragments {
		offset := len(out)
		hi, lo := byte(offset>>8), byte(offset)
		payload, err := c.exchange([]byte{opCaps, hi, lo}, capsReplySize, func(p []byte) error {
			if len(p) < 3 || p[0] != opCapsReply || p[1] != hi || p[2] != lo {
				return ErrBadReply
			}
			return nil
		})
		if err != nil {
			return "", fmt.Errorf("capabilities: %w", err)
		}
		fragment := payload[3:]
		if len(fragment) == 0 {
			return printable(out), nil
		}
		out = append(out, fragment...)
		if len(out) > capsMaxLength {
			break
		}
	}
	return "", fmt.Errorf("capabilities: display never finished sending: %w", ErrBadReply)
}

func printable(raw []byte) string {
	var out strings.Builder
	for _, b := range raw {
		if b >= 0x20 && b <= 0x7e {
			out.WriteByte(b)
		}
	}
	return out.String()
}

func (c *Client) exchange(payload []byte, replySize int, check func([]byte) error) ([]byte, error) {
	request := encode(payload)
	var lastErr error
	for try := range maxAttempts {
		if try > 0 {
			c.sleep(waitBetweenTries)
		}
		reply, err := c.once(request, replySize)
		if err == nil {
			err = check(reply)
		}
		if err == nil {
			return reply, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

func (c *Client) once(request []byte, replySize int) ([]byte, error) {
	if err := c.bus.Write(bus.AddrDDC, request); err != nil {
		return nil, err
	}
	c.sleep(waitBeforeRead)
	raw, err := c.bus.Read(bus.AddrDDC, replySize)
	if err != nil {
		return nil, err
	}
	return decode(raw)
}

func encode(payload []byte) []byte {
	out := make([]byte, 0, len(payload)+3)
	out = append(out, hostAddr, 0x80|byte(len(payload)))
	out = append(out, payload...)
	sum := byte(displayAddr)
	for _, b := range out {
		sum ^= b
	}
	return append(out, sum)
}

func decode(raw []byte) ([]byte, error) {
	if len(raw) < 3 || raw[0] != displayAddr {
		return nil, ErrNoReply
	}
	n := int(raw[1] & 0x7f)
	if n == 0 {
		return nil, ErrNoReply
	}
	if len(raw) < n+3 {
		return nil, ErrBadReply
	}
	sum := byte(replySeed)
	for _, b := range raw[:n+2] {
		sum ^= b
	}
	if sum != raw[n+2] {
		return nil, ErrChecksum
	}
	return raw[2 : n+2], nil
}
