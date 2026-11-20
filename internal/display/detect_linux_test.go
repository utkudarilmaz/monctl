package display

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"

	"github.com/utkudarilmaz/monctl/internal/bus"
	"github.com/utkudarilmaz/monctl/internal/bus/bustest"
)

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
	t       *testing.T
	root    string
	edids   map[string][]byte
	denied  map[string]bool
	minorOf map[string]uint32
	opened  []string
	writes  int
}

type fakeBus struct {
	*bustest.Fake
	m     *machine
	minor uint32
}

func (f fakeBus) Write(addr uint8, data []byte) error {
	f.m.writes++
	return f.Fake.Write(addr, data)
}

func (f fakeBus) DeviceNumber() (uint32, uint32, error) {
	return 89, f.minor, nil
}

func newMachine(t *testing.T) *machine {
	t.Helper()
	m := &machine{
		t: t, root: t.TempDir(),
		edids: map[string][]byte{}, denied: map[string]bool{}, minorOf: map[string]uint32{},
	}
	m.mkdir("sys/bus/i2c/devices")
	m.mkdir("sys/bus/pci")
	m.mkdir("sys/bus/sdio")
	m.mkdir("sys/dev/char")
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
	m.device(rel, class, "pci")
}

func (m *machine) device(rel, class, subsystem string) {
	m.write(filepath.Join("sys/devices", rel, "class"), []byte(class+"\n"))
	m.link(filepath.Join(m.root, "sys/bus", subsystem), filepath.Join("sys/devices", rel, "subsystem"))
}

func (m *machine) i2cBus(n int, parent string, edid []byte) {
	name := "i2c-" + strconv.Itoa(n)
	dir := m.mkdir(filepath.Join("sys/devices", parent, name))
	m.link(dir, filepath.Join("sys/bus/i2c/devices", name))
	char := m.mkdir(filepath.Join("sys/devices", parent, name, "i2c-dev", name))
	m.link(char, filepath.Join("sys/dev/char", "89:"+strconv.Itoa(n)))
	m.minorOf[name] = uint32(n)
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
			return fakeBus{Fake: f, m: m, minor: m.minorOf[filepath.Base(path)]}, nil
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

func TestDetectIgnoresClassOnNonPCIParent(t *testing.T) {
	m := newMachine(t)
	m.device("platform/mmc0/mmc0:0001", "0x030000", "sdio")
	m.i2cBus(3, "platform/mmc0/mmc0:0001", msiEDID(t))

	got, err := m.detector().detect()

	if err != nil || len(got) != 0 {
		t.Fatalf("got %+v, %v, want nothing", got, err)
	}
	if len(m.opened) != 0 {
		t.Fatalf("opened %v, want nothing opened", m.opened)
	}
}

func TestDetectRejectsMalformedClass(t *testing.T) {
	m := newMachine(t)
	m.pciDevice(nvidia, "0x03")
	m.i2cBus(13, nvidia, msiEDID(t))

	got, _ := m.detector().detect()

	if len(got) != 0 || len(m.opened) != 0 {
		t.Fatalf("got %+v, opened %v, want nothing", got, m.opened)
	}
}

func TestDetectDoesNotWriteWhenBusNumberWasReused(t *testing.T) {
	m := newMachine(t)
	m.pciDevice(nvidia, "0x030000")
	m.pciDevice(smbus, "0x0c0500")
	m.i2cBus(0, smbus, nil)
	m.i2cBus(13, nvidia, msiEDID(t))
	m.minorOf["i2c-13"] = 0

	got, err := m.detector().detect()

	if err != nil || len(got) != 0 {
		t.Fatalf("got %+v, %v, want nothing", got, err)
	}
	if m.writes != 0 {
		t.Fatalf("wrote %d times to a bus that is not a graphics bus", m.writes)
	}
}

func TestOpenChecksTheBusAgain(t *testing.T) {
	m := newMachine(t)
	m.pciDevice(nvidia, "0x030000")
	m.pciDevice(smbus, "0x0c0500")
	m.i2cBus(0, smbus, nil)
	m.i2cBus(13, nvidia, msiEDID(t))
	d := m.detector()

	if _, err := d.openVerified(d.busPath(0)); !errors.Is(err, ErrNotGraphicsBus) {
		t.Fatalf("err = %v, want ErrNotGraphicsBus", err)
	}
	b, err := d.openVerified(d.busPath(13))
	if err != nil {
		t.Fatalf("openVerified: %v", err)
	}
	_ = b.Close()
}

func TestBusNumber(t *testing.T) {
	cases := map[string]struct {
		n  int
		ok bool
	}{
		"i2c-13":  {13, true},
		"i2c-0":   {0, true},
		"i2c-+4":  {0, false},
		"i2c--1":  {0, false},
		"i2c-":    {0, false},
		"i2c-ITE": {0, false},
		"spi-1":   {0, false},
	}
	for name, want := range cases {
		n, ok := busNumber(name)
		if n != want.n || ok != want.ok {
			t.Errorf("busNumber(%q) = %d, %v, want %d, %v", name, n, ok, want.n, want.ok)
		}
	}
}

func TestDetectLinkedBusIsNotGivenToAnotherConnector(t *testing.T) {
	m := newMachine(t)
	m.pciDevice(nvidia, "0x030000")
	m.i2cBus(12, nvidia, msiEDID(t))
	m.i2cBus(13, nvidia, msiEDID(t))
	m.connector(nvidia, "DP-4", "connected", msiEDID(t))
	rel := m.connector(nvidia, "DP-5", "connected", msiEDID(t))
	m.link("i2c-12", filepath.Join(rel, "ddc"))

	got, err := m.detector().detect()
	if err != nil {
		t.Fatalf("detect: %v", err)
	}

	want := map[string]string{"DP-5": "i2c-12", "DP-4": "i2c-13"}
	if len(got) != 2 {
		t.Fatalf("found %d displays, want 2: %+v", len(got), got)
	}
	for _, d := range got {
		if want[d.Connector] != filepath.Base(d.Bus) {
			t.Fatalf("connector %q on %q, want %v", d.Connector, d.Bus, want)
		}
	}
}

func TestPair(t *testing.T) {
	cases := map[string]struct {
		buses      []int
		connectors []connector
		want       []pairing
	}{
		"no connector uses the first bus": {
			buses: []int{12, 13},
			want:  []pairing{{bus: 12}},
		},
		"more connectors than buses": {
			buses:      []int{13},
			connectors: []connector{{name: "DP-4", ddcBus: -1}, {name: "DP-5", ddcBus: -1}},
			want:       []pairing{{bus: 13, connector: "DP-4"}},
		},
		"ddc link outside the group is ignored": {
			buses:      []int{12, 13},
			connectors: []connector{{name: "DP-4", ddcBus: 5}},
			want:       []pairing{{bus: 12, connector: "DP-4"}},
		},
		"two connectors linked to one bus": {
			buses:      []int{12, 13},
			connectors: []connector{{name: "DP-4", ddcBus: 13}, {name: "DP-5", ddcBus: 13}},
			want:       []pairing{{bus: 13, connector: "DP-4"}, {bus: 12, connector: "DP-5"}},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := pair(tc.buses, tc.connectors)
			if !slices.Equal(got, tc.want) {
				t.Fatalf("pair = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestDetectIgnoresConnectorWithShortOrMissingEDID(t *testing.T) {
	m := newMachine(t)
	m.pciDevice(nvidia, "0x030000")
	m.i2cBus(13, nvidia, msiEDID(t))
	m.connector(nvidia, "DP-4", "connected", msiEDID(t)[:64])
	rel := m.connector(nvidia, "DP-5", "connected", msiEDID(t))
	if err := os.Remove(filepath.Join(m.root, rel, "edid")); err != nil {
		t.Fatal(err)
	}

	got, err := m.detector().detect()
	if err != nil {
		t.Fatalf("detect: %v", err)
	}

	if len(got) != 1 || got[0].Connector != "" {
		t.Fatalf("got %+v, want one display with no connector name", got)
	}
}

func TestDetectIgnoresConnectorShowingAnotherDisplay(t *testing.T) {
	m := newMachine(t)
	m.pciDevice(nvidia, "0x030000")
	m.i2cBus(13, nvidia, msiEDID(t))
	m.connector(nvidia, "DP-4", "connected", panelEDID(t))

	got, err := m.detector().detect()
	if err != nil {
		t.Fatalf("detect: %v", err)
	}

	if len(got) != 1 || got[0].Connector != "" {
		t.Fatalf("got %+v, want one display with no connector name", got)
	}
}

func TestDetectSkipsBrokenOrOddBusEntries(t *testing.T) {
	m := newMachine(t)
	m.pciDevice(nvidia, "0x030000")
	m.i2cBus(13, nvidia, msiEDID(t))
	m.link(filepath.Join(m.root, "sys/devices", nvidia, "gone"), "sys/bus/i2c/devices/i2c-14")
	m.link(filepath.Join(m.root, "sys/devices", nvidia, "i2c-13"), "sys/bus/i2c/devices/i2c-abc")

	got, err := m.detector().detect()
	if err != nil {
		t.Fatalf("detect: %v", err)
	}

	if len(got) != 1 || len(m.opened) != 1 || m.opened[0] != "i2c-13" {
		t.Fatalf("got %+v, opened %v, want one display from i2c-13 only", got, m.opened)
	}
}

func TestDetectMissingSysfsFindsNothing(t *testing.T) {
	d := detector{
		sysfs: filepath.Join(t.TempDir(), "missing"),
		open: func(string) (bus.Bus, error) {
			t.Fatal("opened a bus with no sysfs")
			return nil, nil
		},
	}

	got, err := d.detect()

	if err != nil || len(got) != 0 {
		t.Fatalf("got %+v, %v, want nothing", got, err)
	}
}

func TestDetectDoesNotWriteWhenCharDeviceLinkIsMissing(t *testing.T) {
	m := newMachine(t)
	m.pciDevice(nvidia, "0x030000")
	m.i2cBus(13, nvidia, msiEDID(t))
	if err := os.Remove(filepath.Join(m.root, "sys/dev/char/89:13")); err != nil {
		t.Fatal(err)
	}

	got, err := m.detector().detect()

	if err != nil || len(got) != 0 || m.writes != 0 {
		t.Fatalf("got %+v, %v, %d writes, want nothing", got, err, m.writes)
	}
}

func TestDetectKeepsWorkingDisplayWhenAnotherBusIsDenied(t *testing.T) {
	m := newMachine(t)
	m.pciDevice(nvidia, "0x030000")
	m.i2cBus(12, nvidia, panelEDID(t))
	m.i2cBus(13, nvidia, msiEDID(t))
	m.denied["i2c-12"] = true

	got, err := m.detector().detect()

	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if len(got) != 1 || filepath.Base(got[0].Bus) != "i2c-13" {
		t.Fatalf("got %+v, want the display on i2c-13", got)
	}
}

type plainBus struct{ *bustest.Fake }

type brokenNumberBus struct{ *bustest.Fake }

func (brokenNumberBus) DeviceNumber() (uint32, uint32, error) {
	return 89, 13, errors.New("no device number")
}

func TestOpenRejectsBusWithoutDeviceNumber(t *testing.T) {
	cases := map[string]func(*bustest.Fake) bus.Bus{
		"no DeviceNumber method": func(f *bustest.Fake) bus.Bus { return plainBus{f} },
		"DeviceNumber fails":     func(f *bustest.Fake) bus.Bus { return brokenNumberBus{f} },
	}
	for name, wrap := range cases {
		t.Run(name, func(t *testing.T) {
			m := newMachine(t)
			m.pciDevice(nvidia, "0x030000")
			m.i2cBus(13, nvidia, msiEDID(t))
			f := bustest.NewFake()
			d := m.detector()
			d.open = func(string) (bus.Bus, error) { return wrap(f), nil }

			b, err := d.openVerified(d.busPath(13))

			if err == nil || b != nil {
				t.Fatalf("openVerified = %v, %v, want an error", b, err)
			}
			if !f.Closed {
				t.Fatal("rejected bus was not closed")
			}
		})
	}
}

func TestDetectSkipsBusWhenEDIDWriteFails(t *testing.T) {
	m := newMachine(t)
	m.pciDevice(nvidia, "0x030000")
	m.i2cBus(13, nvidia, msiEDID(t))
	d := m.detector()
	open := d.open
	d.open = func(path string) (bus.Bus, error) {
		b, err := open(path)
		if err == nil {
			b.(fakeBus).FailWrites = 1
		}
		return b, err
	}

	got, err := d.detect()

	if err != nil || len(got) != 0 {
		t.Fatalf("got %+v, %v, want nothing", got, err)
	}
}
