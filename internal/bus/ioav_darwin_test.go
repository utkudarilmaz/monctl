package bus

import (
	"errors"
	"testing"
)

func TestAVBusRefusesUseAfterClose(t *testing.T) {
	b := &avBus{}
	if err := b.Close(); err != nil {
		t.Fatal(err)
	}
	if err := b.Write(AddrDDC, []byte{0x51}); !errors.Is(err, errAVClosed) {
		t.Fatalf("Write error = %v, want closed", err)
	}
	if _, err := b.Read(AddrDDC, 11); !errors.Is(err, errAVClosed) {
		t.Fatalf("Read error = %v, want closed", err)
	}
}

func TestAVBusCloseTwice(t *testing.T) {
	b := &avBus{}
	for i := 0; i < 2; i++ {
		if err := b.Close(); err != nil {
			t.Fatalf("Close %d: %v", i+1, err)
		}
	}
}

func TestAVBusClosedWinsOverBadInput(t *testing.T) {
	b := &avBus{}
	if err := b.Write(AddrDDC, nil); !errors.Is(err, errAVClosed) {
		t.Fatalf("Write error = %v, want closed", err)
	}
	if _, err := b.Read(AddrDDC, 0); !errors.Is(err, errAVClosed) {
		t.Fatalf("Read error = %v, want closed", err)
	}
}

// The placeholder service is never passed to IOKit: every case returns
// before the I2C call, and the bus is never closed.
func TestAVBusChecksBeforeIOKit(t *testing.T) {
	b := &avBus{service: 1}

	if err := b.Write(AddrDDC, nil); err == nil || errors.Is(err, errAVClosed) {
		t.Fatalf("empty Write error = %v, want empty write", err)
	}
	if err := b.Write(AddrEDID, []byte{}); err == nil {
		t.Fatal("empty EDID Write was accepted")
	}
	for _, n := range []int{0, -1} {
		if _, err := b.Read(AddrDDC, n); err == nil || errors.Is(err, errAVClosed) {
			t.Fatalf("Read(%d) error = %v, want empty read", n, err)
		}
	}
	if err := b.Write(AddrEDID, []byte{0x80, 0x01}); err != nil || b.edidOffset != 0x80 {
		t.Fatalf("EDID Write = %v, offset %#x, want offset 0x80", err, b.edidOffset)
	}
	if err := b.Write(AddrEDID, []byte{0x00}); err != nil || b.edidOffset != 0x00 {
		t.Fatalf("EDID Write = %v, offset %#x, want offset 0x00", err, b.edidOffset)
	}
}
