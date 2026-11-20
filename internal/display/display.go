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

// ErrNotGraphicsBus means a device is not an I2C bus of a graphics card, so monctl refuses to use it.
var ErrNotGraphicsBus = errors.New("display: not an I2C bus of a graphics card")

// Display is one external display and the I2C bus that reaches it.
type Display struct {
	ID        string
	Info      edid.Info
	Connector string
	Bus       string
}

func assignIDs(displays []Display) {
	used := map[string]bool{}
	for i := range displays {
		info := displays[i].Info
		base := fmt.Sprintf("%s-%04x", strings.ToLower(info.Manufacturer), info.ProductCode)
		if info.Serial != "" {
			base += "-" + strings.ToLower(info.Serial)
		}
		id := base
		for n := 2; used[id]; n++ {
			id = fmt.Sprintf("%s-%d", base, n)
		}
		used[id] = true
		displays[i].ID = id
	}
}
