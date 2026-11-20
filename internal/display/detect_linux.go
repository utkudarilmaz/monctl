package display

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/utkudarilmaz/monctl/internal/bus"
	"github.com/utkudarilmaz/monctl/internal/edid"
)

const displayClassPrefix = "0x03"

var builtInConnectors = []string{"eDP", "LVDS", "DSI"}

type detector struct {
	sysfs string
	dev   string
	open  func(path string) (bus.Bus, error)
}

type connector struct {
	name    string
	edid    []byte
	builtIn bool
	ddcBus  int
}

// Detect returns the external displays attached to this machine.
func Detect() ([]Display, error) {
	return detector{sysfs: "/sys", dev: "/dev", open: bus.OpenI2C}.detect()
}

func (d detector) detect() ([]Display, error) {
	connectors := d.connectors()

	var keys []string
	busesByEDID := map[string][]int{}
	denied := false
	for _, n := range d.graphicsBuses() {
		raw, err := d.readEDID(n)
		if errors.Is(err, fs.ErrPermission) {
			denied = true
		}
		if err != nil {
			continue
		}
		key := string(raw)
		if _, ok := busesByEDID[key]; !ok {
			keys = append(keys, key)
		}
		busesByEDID[key] = append(busesByEDID[key], n)
	}

	var out []Display
	for _, key := range keys {
		info, err := edid.Parse([]byte(key))
		if err != nil {
			continue
		}
		var matching []connector
		builtIn := false
		for _, c := range connectors {
			if !bytes.Equal(c.edid, []byte(key)) {
				continue
			}
			builtIn = builtIn || c.builtIn
			matching = append(matching, c)
		}
		if builtIn {
			continue
		}
		for _, p := range pair(busesByEDID[key], matching) {
			out = append(out, Display{
				Info:      info,
				Connector: p.connector,
				Bus:       d.busPath(p.bus),
			})
		}
	}
	if len(out) == 0 && denied {
		return nil, ErrAccess
	}
	assignIDs(out)
	return out, nil
}

type pairing struct {
	bus       int
	connector string
}

// pair decides which buses carry a display. Several buses can show the same
// EDID for one display, so the count of connected connectors sets how many
// displays there are, and a bus named by a connector wins over the others.
func pair(buses []int, connectors []connector) []pairing {
	want := max(1, len(connectors))
	var out []pairing
	used := map[int]bool{}
	var unlinked []string
	for _, c := range connectors {
		if c.ddcBus >= 0 && slices.Contains(buses, c.ddcBus) && !used[c.ddcBus] {
			used[c.ddcBus] = true
			out = append(out, pairing{bus: c.ddcBus, connector: c.name})
			continue
		}
		unlinked = append(unlinked, c.name)
	}
	for _, n := range buses {
		if len(out) >= want {
			break
		}
		if used[n] {
			continue
		}
		p := pairing{bus: n}
		if len(unlinked) > 0 {
			p.connector, unlinked = unlinked[0], unlinked[1:]
		}
		out = append(out, p)
	}
	return out
}

func (d detector) busPath(n int) string {
	return filepath.Join(d.dev, "i2c-"+strconv.Itoa(n))
}

func (d detector) readEDID(n int) ([]byte, error) {
	b, err := d.open(d.busPath(n))
	if err != nil {
		return nil, err
	}
	defer func() { _ = b.Close() }()

	if err := b.Write(bus.AddrEDID, []byte{0x00}); err != nil {
		return nil, err
	}
	raw, err := b.Read(bus.AddrEDID, edid.BlockSize)
	if err != nil {
		return nil, err
	}
	if _, err := edid.Parse(raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func (d detector) graphicsBuses() []int {
	entries, _ := filepath.Glob(filepath.Join(d.sysfs, "bus/i2c/devices/i2c-*"))
	var out []int
	for _, entry := range entries {
		n, ok := busNumber(filepath.Base(entry))
		if !ok {
			continue
		}
		resolved, err := filepath.EvalSymlinks(entry)
		if err != nil {
			continue
		}
		if d.belongsToGraphicsCard(resolved) {
			out = append(out, n)
		}
	}
	slices.Sort(out)
	return out
}

func (d detector) belongsToGraphicsCard(busDir string) bool {
	root, err := filepath.EvalSymlinks(d.sysfs)
	if err != nil {
		return false
	}
	for dir := filepath.Dir(busDir); strings.HasPrefix(dir, root+string(filepath.Separator)); dir = filepath.Dir(dir) {
		class, err := os.ReadFile(filepath.Join(dir, "class"))
		if err != nil {
			continue
		}
		return strings.HasPrefix(strings.TrimSpace(string(class)), displayClassPrefix)
	}
	return false
}

func (d detector) connectors() []connector {
	entries, _ := filepath.Glob(filepath.Join(d.sysfs, "class/drm/card*-*"))
	var out []connector
	for _, entry := range entries {
		status, err := os.ReadFile(filepath.Join(entry, "status"))
		if err != nil || strings.TrimSpace(string(status)) != "connected" {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(entry, "edid"))
		if err != nil || len(raw) < edid.BlockSize {
			continue
		}
		_, name, _ := strings.Cut(filepath.Base(entry), "-")
		c := connector{name: name, edid: raw[:edid.BlockSize], ddcBus: -1}
		for _, prefix := range builtInConnectors {
			c.builtIn = c.builtIn || strings.HasPrefix(name, prefix)
		}
		if target, err := os.Readlink(filepath.Join(entry, "ddc")); err == nil {
			if n, ok := busNumber(filepath.Base(target)); ok {
				c.ddcBus = n
			}
		}
		out = append(out, c)
	}
	return out
}

func busNumber(name string) (int, bool) {
	digits, ok := strings.CutPrefix(name, "i2c-")
	if !ok {
		return 0, false
	}
	n, err := strconv.Atoi(digits)
	return n, err == nil
}
