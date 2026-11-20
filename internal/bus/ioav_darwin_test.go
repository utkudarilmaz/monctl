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
