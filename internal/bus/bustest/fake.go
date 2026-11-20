// Package bustest provides an in-memory display for tests.
package bustest

import (
	"errors"
	"fmt"
	"sync"

	"github.com/utkudarilmaz/monctl/internal/bus"
)

// ErrInjected is returned by reads and writes that were told to fail.
var ErrInjected = errors.New("bustest: injected failure")

// VCP is the state of one feature on the fake display.
type VCP struct {
	Current uint16
	Max     uint16
}

// Fake is a display that speaks DDC/CI. The zero value is not usable, call NewFake.
type Fake struct {
	mu sync.Mutex

	VCP          map[byte]VCP
	Capabilities string
	EDID         []byte

	Asleep         bool
	FailWrites     int
	FailReads      int
	CorruptReplies int

	Writes [][]byte
	Closed bool

	pending []byte
}

// NewFake returns a fake display with no features.
func NewFake() *Fake {
	return &Fake{VCP: map[byte]VCP{}}
}

// Write implements bus.Bus.
func (f *Fake) Write(addr uint8, data []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.FailWrites > 0 {
		f.FailWrites--
		return ErrInjected
	}
	if addr == bus.AddrEDID {
		return nil
	}
	if addr != bus.AddrDDC {
		return fmt.Errorf("bustest: nothing at address %#x", addr)
	}
	f.Writes = append(f.Writes, append([]byte(nil), data...))
	f.pending = nil

	if f.Asleep {
		return nil
	}
	if payload, ok := requestPayload(data); ok {
		f.handle(payload)
	}
	return nil
}

// Read implements bus.Bus.
func (f *Fake) Read(addr uint8, n int) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.FailReads > 0 {
		f.FailReads--
		return nil, ErrInjected
	}
	out := make([]byte, n)
	switch addr {
	case bus.AddrEDID:
		copy(out, f.EDID)
	case bus.AddrDDC:
		reply := f.pending
		f.pending = nil
		if reply == nil {
			reply = frame(nil)
		}
		if f.CorruptReplies > 0 {
			f.CorruptReplies--
			reply[len(reply)-1] ^= 0xff
		}
		copy(out, reply)
	default:
		return nil, fmt.Errorf("bustest: nothing at address %#x", addr)
	}
	return out, nil
}

// Close implements bus.Bus.
func (f *Fake) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Closed = true
	return nil
}

func (f *Fake) handle(payload []byte) {
	switch {
	case len(payload) == 2 && payload[0] == 0x01:
		code := payload[1]
		v, ok := f.VCP[code]
		result := byte(0x00)
		if !ok {
			result = 0x01
		}
		f.pending = frame([]byte{
			0x02, result, code, 0x00,
			byte(v.Max >> 8), byte(v.Max),
			byte(v.Current >> 8), byte(v.Current),
		})
	case len(payload) == 4 && payload[0] == 0x03:
		code := payload[1]
		if v, ok := f.VCP[code]; ok {
			v.Current = uint16(payload[2])<<8 | uint16(payload[3])
			f.VCP[code] = v
		}
	case len(payload) == 3 && payload[0] == 0xf3:
		offset := int(payload[1])<<8 | int(payload[2])
		var chunk []byte
		if offset < len(f.Capabilities) {
			chunk = []byte(f.Capabilities[offset:])
			if len(chunk) > 32 {
				chunk = chunk[:32]
			}
		}
		f.pending = frame(append([]byte{0xe3, payload[1], payload[2]}, chunk...))
	}
}

func requestPayload(data []byte) ([]byte, bool) {
	if len(data) < 3 || data[0] != 0x51 {
		return nil, false
	}
	n := int(data[1] & 0x7f)
	if len(data) != n+3 {
		return nil, false
	}
	sum := byte(0x6e)
	for _, b := range data[:len(data)-1] {
		sum ^= b
	}
	if sum != data[len(data)-1] {
		return nil, false
	}
	return data[2 : 2+n], true
}

func frame(payload []byte) []byte {
	out := append([]byte{0x6e, 0x80 | byte(len(payload))}, payload...)
	sum := byte(0x50)
	for _, b := range out {
		sum ^= b
	}
	return append(out, sum)
}
