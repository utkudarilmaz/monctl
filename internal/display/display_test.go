package display

import (
	"encoding/hex"
	"testing"

	"github.com/utkudarilmaz/monctl/internal/edid"
)

func TestAssignIDs(t *testing.T) {
	msi := edid.Info{Manufacturer: "MSI", ProductCode: 0x3cf0}
	withSerial := msi
	withSerial.Serial = "AB12"
	displays := []Display{{Info: msi}, {Info: withSerial}, {Info: msi}, {Info: withSerial}}

	assignIDs(displays)

	want := []string{"msi-3cf0", "msi-3cf0-ab12", "msi-3cf0-2", "msi-3cf0-ab12-2"}
	for i, d := range displays {
		if d.ID != want[i] {
			t.Errorf("displays[%d].ID = %q, want %q", i, d.ID, want[i])
		}
	}
}

func TestAssignIDsNeverRepeatsAnID(t *testing.T) {
	plain := edid.Info{Manufacturer: "MSI", ProductCode: 0x3cf0}
	numbered := plain
	numbered.Serial = "2"
	displays := []Display{{Info: plain}, {Info: plain}, {Info: numbered}, {Info: plain}}

	assignIDs(displays)

	seen := map[string]bool{}
	for _, d := range displays {
		if seen[d.ID] {
			t.Fatalf("ID %q is used twice in %+v", d.ID, displays)
		}
		seen[d.ID] = true
	}
}

const msiHex = "00ffffffffffff003669f03c00000000" +
	"07230104b53c2278fb1e65ad4f43af26" +
	"0c5054a54b0081809500b300d1c00101" +
	"01010101010108e80030f2705a80b058" +
	"4a0055502100001a000000fd0030a0ff" +
	"ff9d010a202020202020000000fc004d" +
	"50472032373455204531364d000000ff" +
	"00303030303030303030303030300230"

func msiEDID(t *testing.T) []byte {
	t.Helper()
	b, err := hex.DecodeString(msiHex)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
