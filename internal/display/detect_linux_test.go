package display

import (
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/utkudarilmaz/monctl/internal/bus"
	"github.com/utkudarilmaz/monctl/internal/bus/bustest"
)

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

func panelEDID(t *testing.T) []byte {
	t.Helper()
	b := msiEDID(t)
	b[10], b[11] = 0x36, 0x20
	var sum byte
	for _, v := range b[:127] {
		sum += v
	}
	b[127] = -sum
	return b
}

type machine struct {
	t      *testing.T
	root   string
	edids  map[string][]byte
	denied map[string]bool
	opened []string
}

func newMachine(t *testing.T) *machine {
	t.Helper()
	m := &machine{t: t, root: t.TempDir(), edids: map[string][]byte{}, denied: map[string]bool{}}
	m.mkdir("sys/bus/i2c/devices")
	m.mkdir("sys/class/drm")
	return m
}

func (m *machine) mkdir(rel string) string {
	m.t.Helper()
	p := filepath.Join(m.root, rel)
	if err := os.MkdirAll(p, 0o755); err != nil {
		m.t.Fatal(err)
	}
	return p
}

func (m *machine) write(rel string, data []byte) {
	m.t.Helper()
	m.mkdir(filepath.Dir(rel))
	if err := os.WriteFile(filepath.Join(m.root, rel), data, 0o644); err != nil {
		m.t.Fatal(err)
	}
}

func (m *machine) link(target, rel string) {
	m.t.Helper()
	if err := os.Symlink(target, filepath.Join(m.root, rel)); err != nil {
		m.t.Fatal(err)
	}
}

func (m *machine) pciDevice(rel, class string) {
	m.write(filepath.Join("sys/devices", rel, "class"), []byte(class+"\n"))
}

func (m *machine) i2cBus(n int, parent string, edid []byte) {
	name := "i2c-" + strconv.Itoa(n)
	dir := m.mkdir(filepath.Join("sys/devices", parent, name))
	m.link(dir, filepath.Join("sys/bus/i2c/devices", name))
	if edid != nil {
		m.edids[filepath.Join(m.root, "dev", name)] = edid
	}
}

func (m *machine) connector(card, name, status string, edid []byte) string {
	rel := filepath.Join("sys/devices", card, "drm/card0", "card0-"+name)
	m.write(filepath.Join(rel, "status"), []byte(status+"\n"))
	m.write(filepath.Join(rel, "edid"), edid)
	m.link(filepath.Join(m.root, rel), filepath.Join("sys/class/drm", "card0-"+name))
	return rel
}

func (m *machine) detector() detector {
	return detector{
		sysfs: filepath.Join(m.root, "sys"),
		dev:   filepath.Join(m.root, "dev"),
		open: func(path string) (bus.Bus, error) {
			m.opened = append(m.opened, filepath.Base(path))
			if m.denied[filepath.Base(path)] {
				return nil, &fs.PathError{Op: "open", Path: path, Err: fs.ErrPermission}
			}
			f := bustest.NewFake()
			f.EDID = m.edids[path]
			return f, nil
		},
	}
}

const (
	nvidia = "pci0000:00/0000:00:01.1/0000:01:00.0"
	intel  = "pci0000:00/0000:00:02.0"
	smbus  = "pci0000:00/0000:00:1f.4"
)

func TestDetectFindsDisplayWithoutSysfsLink(t *testing.T) {
	m := newMachine(t)
	m.pciDevice(nvidia, "0x030000")
	m.i2cBus(12, nvidia, nil)
	m.i2cBus(13, nvidia, msiEDID(t))
	m.connector(nvidia, "DP-5", "connected", msiEDID(t))

	got, err := m.detector().detect()
	if err != nil {
		t.Fatalf("detect: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("found %d displays, want 1: %+v", len(got), got)
	}
	d := got[0]
	if d.Info.Name != "MPG 274U E16M" || d.Connector != "DP-5" || filepath.Base(d.Bus) != "i2c-13" {
		t.Fatalf("got %+v", d)
	}
	if d.ID != "msi-3cf0-0000000000000" {
		t.Fatalf("ID = %q", d.ID)
	}
}

func TestDetectNeverOpensNonGraphicsBuses(t *testing.T) {
	m := newMachine(t)
	m.pciDevice(nvidia, "0x030000")
	m.pciDevice(smbus, "0x0c0500")
	m.i2cBus(0, smbus, msiEDID(t))
	m.i2cBus(1, "platform/AMDI0010:00", msiEDID(t))
	m.i2cBus(13, nvidia, msiEDID(t))

	got, err := m.detector().detect()
	if err != nil {
		t.Fatalf("detect: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("found %d displays, want 1", len(got))
	}
	if len(m.opened) != 1 || m.opened[0] != "i2c-13" {
		t.Fatalf("opened %v, want only i2c-13", m.opened)
	}
}

func TestDetectSkipsBuiltInPanel(t *testing.T) {
	m := newMachine(t)
	m.pciDevice(intel, "0x030000")
	rel := m.connector(intel, "eDP-1", "connected", panelEDID(t))
	m.i2cBus(4, filepath.Join(intel, "drm/card0/card0-eDP-1"), panelEDID(t))
	m.link("i2c-4", filepath.Join(rel, "ddc"))

	got, err := m.detector().detect()
	if err != nil {
		t.Fatalf("detect: %v", err)
	}

	if len(got) != 0 {
		t.Fatalf("found %+v, want nothing", got)
	}
}

func TestDetectPrefersBusNamedBySysfsLink(t *testing.T) {
	m := newMachine(t)
	m.pciDevice(intel, "0x030000")
	rel := m.connector(intel, "DP-1", "connected", msiEDID(t))
	m.i2cBus(2, intel, msiEDID(t))
	m.i2cBus(5, filepath.Join(intel, "drm/card0/card0-DP-1"), msiEDID(t))
	m.link("i2c-5", filepath.Join(rel, "ddc"))

	got, err := m.detector().detect()
	if err != nil {
		t.Fatalf("detect: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("found %d displays, want 1: %+v", len(got), got)
	}
	if filepath.Base(got[0].Bus) != "i2c-5" {
		t.Fatalf("Bus = %q, want i2c-5", got[0].Bus)
	}
}

func TestDetectIgnoresDisconnectedConnectorEDID(t *testing.T) {
	m := newMachine(t)
	m.pciDevice(nvidia, "0x030000")
	m.i2cBus(13, nvidia, msiEDID(t))
	m.connector(nvidia, "DP-4", "disconnected", msiEDID(t))

	got, err := m.detector().detect()
	if err != nil {
		t.Fatalf("detect: %v", err)
	}

	if len(got) != 1 || got[0].Connector != "" {
		t.Fatalf("got %+v, want one display with no connector name", got)
	}
}

func TestDetectTwoIdenticalDisplaysGetDifferentIDs(t *testing.T) {
	m := newMachine(t)
	m.pciDevice(nvidia, "0x030000")
	m.i2cBus(12, nvidia, msiEDID(t))
	m.i2cBus(13, nvidia, msiEDID(t))
	m.connector(nvidia, "DP-4", "connected", msiEDID(t))
	m.connector(nvidia, "DP-5", "connected", msiEDID(t))

	got, err := m.detector().detect()
	if err != nil {
		t.Fatalf("detect: %v", err)
	}

	if len(got) != 2 || got[0].ID == got[1].ID {
		t.Fatalf("got %+v, want two displays with different IDs", got)
	}
}

func TestDetectReportsAccessDenied(t *testing.T) {
	m := newMachine(t)
	m.pciDevice(nvidia, "0x030000")
	m.i2cBus(13, nvidia, msiEDID(t))
	m.denied["i2c-13"] = true

	_, err := m.detector().detect()

	if !errors.Is(err, ErrAccess) {
		t.Fatalf("err = %v, want ErrAccess", err)
	}
}

func TestDetectNothingAttached(t *testing.T) {
	m := newMachine(t)
	m.pciDevice(nvidia, "0x030000")
	m.i2cBus(13, nvidia, nil)

	got, err := m.detector().detect()

	if err != nil || len(got) != 0 {
		t.Fatalf("got %+v, %v, want nothing and no error", got, err)
	}
}
