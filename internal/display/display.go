// Package display finds the external displays attached to this machine.
package display

import (
	"errors"
	"fmt"
	"strings"

	"github.com/utkudarilmaz/monctl/internal/edid"
)

// ErrAccess means displays may be attached, but their control channel cannot be opened.
var ErrAccess = errors.New("display: not allowed to open the display control channel")

// Display is one external display and the I2C bus that reaches it.
type Display struct {
	ID        string
	Info      edid.Info
	Connector string
	Bus       string
}

func assignIDs(displays []Display) {
	seen := map[string]int{}
	for i := range displays {
		info := displays[i].Info
		id := fmt.Sprintf("%s-%04x", strings.ToLower(info.Manufacturer), info.ProductCode)
		if info.Serial != "" {
			id += "-" + strings.ToLower(info.Serial)
		}
		seen[id]++
		if n := seen[id]; n > 1 {
			id = fmt.Sprintf("%s-%d", id, n)
		}
		displays[i].ID = id
	}
}
