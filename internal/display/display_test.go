package display

import (
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
