package display

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/utkudarilmaz/monctl/internal/bus"
	"github.com/utkudarilmaz/monctl/internal/edid"
)

const avPrefix = "avservice:"

type detector struct {
	list func() ([]uint64, error)
	open func(id uint64) (bus.Bus, error)
}

func defaultDetector() detector {
	return detector{list: bus.ListAV, open: bus.OpenAV}
}

// Detect returns the external displays attached to this machine. The built-in
// panel is never listed.
func Detect() ([]Display, error) {
	return defaultDetector().detect()
}

// Open opens the control channel of a display found by Detect.
func Open(disp Display) (bus.Bus, error) {
	return defaultDetector().openDisplay(disp)
}

func (d detector) detect() ([]Display, error) {
	ids, err := d.list()
	if err != nil {
		return nil, err
	}
	var out []Display
	for _, id := range ids {
		info, err := d.readEDID(id)
		if err != nil {
			continue
		}
		out = append(out, Display{Info: info, Bus: avPrefix + strconv.FormatUint(id, 10)})
	}
	assignIDs(out)
	return out, nil
}

func (d detector) readEDID(id uint64) (edid.Info, error) {
	b, err := d.open(id)
	if err != nil {
		return edid.Info{}, err
	}
	defer func() { _ = b.Close() }()

	if err := b.Write(bus.AddrEDID, []byte{0x00}); err != nil {
		return edid.Info{}, err
	}
	raw, err := b.Read(bus.AddrEDID, edid.BlockSize)
	if err != nil {
		return edid.Info{}, err
	}
	return edid.Parse(raw)
}

func (d detector) openDisplay(disp Display) (bus.Bus, error) {
	text, ok := strings.CutPrefix(disp.Bus, avPrefix)
	id, err := strconv.ParseUint(text, 10, 64)
	if !ok || err != nil {
		return nil, fmt.Errorf("display: %q is not a display service", disp.Bus)
	}
	b, err := d.open(id)
	if errors.Is(err, bus.ErrNoAVService) {
		return nil, fmt.Errorf("display: %s is gone, run \"monctl list\" again", disp.ID)
	}
	return b, err
}
