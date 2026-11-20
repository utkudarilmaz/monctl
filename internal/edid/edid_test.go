package edid

import (
	"encoding/hex"
	"errors"
	"testing"
)

const msiHex = "00ffffffffffff003669f03c00000000" +
	"07230104b53c2278fb1e65ad4f43af26" +
	"0c5054a54b0081809500b300d1c00101" +
	"01010101010108e80030f2705a80b058" +
	"4a0055502100001a000000fd0030a0ff" +
	"ff9d010a202020202020000000fc004d" +
	"50472032373455204531364d000000ff" +
	"00303030303030303030303030300230"

func msiBlock(t *testing.T) []byte {
	t.Helper()
	b, err := hex.DecodeString(msiHex)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestParseRealMonitor(t *testing.T) {
	got, err := Parse(msiBlock(t))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	want := Info{
		Manufacturer: "MSI",
		ProductCode:  0x3cf0,
		Name:         "MPG 274U E16M",
		Serial:       "0000000000000",
	}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestParseIgnoresExtensionBlocks(t *testing.T) {
	block := append(msiBlock(t), make([]byte, 256)...)

	if _, err := Parse(block); err != nil {
		t.Fatalf("Parse: %v", err)
	}
}

func TestParseFallsBackToNumericSerial(t *testing.T) {
	block := msiBlock(t)
	block[12], block[13], block[14], block[15] = 0x78, 0x56, 0x34, 0x12
	block[111] = 0x10
	fixChecksum(block)

	got, err := Parse(block)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got.Serial != "305419896" {
		t.Fatalf("Serial = %q, want %q", got.Serial, "305419896")
	}
}

func TestParseRejectsBadInput(t *testing.T) {
	short := msiBlock(t)[:100]

	badHeader := msiBlock(t)
	badHeader[0] = 0x01
	fixChecksum(badHeader)

	badSum := msiBlock(t)
	badSum[20] ^= 0xff

	cases := map[string]struct {
		in   []byte
		want error
	}{
		"too short":    {short, ErrTooShort},
		"bad header":   {badHeader, ErrBadHeader},
		"bad checksum": {badSum, ErrBadChecksum},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Parse(tc.in); !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
		})
	}
}

func fixChecksum(b []byte) {
	var sum byte
	for _, v := range b[:127] {
		sum += v
	}
	b[127] = -sum
}
