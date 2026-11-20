package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/utkudarilmaz/monctl/internal/bus"
	"github.com/utkudarilmaz/monctl/internal/bus/bustest"
	"github.com/utkudarilmaz/monctl/internal/config"
	"github.com/utkudarilmaz/monctl/internal/display"
	"github.com/utkudarilmaz/monctl/internal/edid"
)

var (
	msi = display.Display{
		ID:   "msi-3cf0",
		Info: edid.Info{Manufacturer: "MSI", ProductCode: 0x3cf0, Name: "MPG 274U E16M", Serial: "SECRET123"},
		Bus:  "/dev/i2c-13",
	}
	unknown = display.Display{
		ID:   "xyz-0001",
		Info: edid.Info{Manufacturer: "XYZ", ProductCode: 1, Name: "Mystery"},
	}
)

type rig struct {
	app    app
	out    *bytes.Buffer
	errOut *bytes.Buffer
	fake   *bustest.Fake
	opened []string
}

func newRig(displays ...display.Display) *rig {
	r := &rig{fake: bustest.NewFake()}
	r.fake.VCP[0x10] = bustest.VCP{Current: 50, Max: 100}
	r.fake.VCP[0x60] = bustest.VCP{Current: 7, Max: 14}
	r.fake.Capabilities = "(prot(monitor)vcp(10 60))"
	r.app, r.out, r.errOut = newApp(displays, nil)
	r.app.open = func(d display.Display) (bus.Bus, error) {
		r.opened = append(r.opened, d.ID)
		return r.fake, nil
	}
	return r
}

func (r *rig) run(t *testing.T, want int, args ...string) {
	t.Helper()
	r.out.Reset()
	r.errOut.Reset()
	if code := r.app.run(args); code != want {
		t.Fatalf("monctl %v: exit code = %d, want %d\nstdout: %s\nstderr: %s", args, code, want, r.out, r.errOut)
	}
}

func (r *rig) decode(t *testing.T, v any) {
	t.Helper()
	if err := json.Unmarshal(r.out.Bytes(), v); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, r.out)
	}
}

func TestBrightnessPrintsTheValue(t *testing.T) {
	r := newRig(msi)
	r.run(t, 0, "brightness")
	if got := r.out.String(); got != "50\n" {
		t.Fatalf("output = %q, want 50", got)
	}
	if !r.fake.Closed {
		t.Fatal("the bus was not closed")
	}
}

func TestBrightnessSetAndAdjust(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"brightness", "40"}, "40\n"},
		{[]string{"brightness", "+10"}, "60\n"},
		{[]string{"brightness", "-10"}, "40\n"},
		{[]string{"brightness", "-80"}, "0\n"},
		{[]string{"brightness", "500"}, "100\n"},
		{[]string{"brightness", "-10", "-d", "msi-3cf0"}, "40\n"},
		{[]string{"brightness", "-d", "msi-3cf0", "+5"}, "55\n"},
	}
	for _, tt := range tests {
		r := newRig(msi)
		r.run(t, 0, tt.args...)
		if got := r.out.String(); got != tt.want {
			t.Errorf("monctl %v printed %q, want %q", tt.args, got, tt.want)
		}
	}
}

func TestBrightnessJSON(t *testing.T) {
	r := newRig(msi)
	r.run(t, 0, "brightness", "-json", "30")

	var got map[string]any
	r.decode(t, &got)
	if got["display"] != "msi-3cf0" || got["brightness"] != float64(30) || got["max"] != float64(100) {
		t.Fatalf("unexpected JSON %v", got)
	}
}

func TestBrightnessRejectsBadValues(t *testing.T) {
	for _, args := range [][]string{
		{"brightness", "bright"},
		{"brightness", "10", "20"},
		{"brightness", "-nope"},
	} {
		r := newRig(msi)
		r.run(t, 2, args...)
		if len(r.opened) != 0 {
			t.Errorf("monctl %v opened the monitor", args)
		}
	}
}

func TestSilentMonitorExplainsWhy(t *testing.T) {
	r := newRig(msi)
	r.fake.Asleep = true
	r.app.open = func(display.Display) (bus.Bus, error) { return r.fake, nil }

	for _, args := range [][]string{{"brightness"}, {"brightness", "10"}, {"input"}, {"probe"}} {
		r.run(t, 1, args...)
		if !strings.Contains(r.errOut.String(), "another\ncomputer") {
			t.Errorf("monctl %v: stderr %q does not explain the silent monitor", args, r.errOut)
		}
	}
}

func TestPickingAMonitor(t *testing.T) {
	r := newRig()
	r.run(t, 1, "brightness")
	if !strings.Contains(r.errOut.String(), "no external monitor") {
		t.Fatalf("stderr = %q", r.errOut)
	}

	r = newRig(msi, unknown)
	r.run(t, 1, "brightness")
	if !strings.Contains(r.errOut.String(), "-d") {
		t.Fatalf("stderr %q does not point to -d", r.errOut)
	}

	r.run(t, 0, "brightness", "-d", "xyz-0001")
	if len(r.opened) != 1 || r.opened[0] != "xyz-0001" {
		t.Fatalf("opened %v, want xyz-0001", r.opened)
	}

	r.run(t, 1, "brightness", "-d", "missing")
	if !strings.Contains(r.errOut.String(), "missing") {
		t.Fatalf("stderr = %q", r.errOut)
	}
}

func TestErrorsFromDetectOpenProfilesAndConfig(t *testing.T) {
	r := newRig(msi)
	r.app.detect = func() ([]display.Display, error) { return nil, display.ErrAccess }
	r.run(t, 1, "brightness")
	if !strings.Contains(r.errOut.String(), "udev") {
		t.Fatalf("stderr %q does not explain the udev rule", r.errOut)
	}

	r = newRig(msi)
	r.app.open = func(display.Display) (bus.Bus, error) { return nil, errors.New("open failed") }
	r.run(t, 1, "input")
	if !strings.Contains(r.errOut.String(), "open failed") {
		t.Fatalf("stderr = %q", r.errOut)
	}

	r = newRig(msi)
	r.app.config = func() (config.Config, error) { return config.Config{}, errors.New("bad config") }
	r.run(t, 1, "input")
	if !strings.Contains(r.errOut.String(), "bad config") || len(r.opened) != 0 {
		t.Fatalf("stderr = %q, opened %v", r.errOut, r.opened)
	}
}

func TestInputShowsCurrentAndList(t *testing.T) {
	r := newRig(msi)
	r.app.config = func() (config.Config, error) {
		return config.Config{Inputs: map[string]config.Input{"usbc": {Label: "MacBook"}}}, nil
	}
	r.run(t, 0, "input")

	out := r.out.String()
	for _, want := range []string{"Current: DisplayPort (dp)", "usbc", "MacBook"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output %q does not contain %q", out, want)
		}
	}
}

func TestInputJSON(t *testing.T) {
	r := newRig(msi)
	r.run(t, 0, "input", "-json")

	var got inputOutput
	r.decode(t, &got)
	if got.Display != "msi-3cf0" || got.Current.ID != "dp" || got.Current.Value != 7 || len(got.Inputs) != 2 {
		t.Fatalf("unexpected JSON %+v", got)
	}
}

func TestInputSwitch(t *testing.T) {
	r := newRig(msi)
	r.run(t, 0, "input", "usbc")

	if r.fake.VCP[0x60].Current != 8 {
		t.Fatalf("monitor got input value %d, want 8", r.fake.VCP[0x60].Current)
	}
	if got := r.out.String(); got != "Switched to USB-C (usbc).\n" {
		t.Fatalf("output = %q", got)
	}

	r.run(t, 0, "input", "-json", "dp")
	var got map[string]any
	r.decode(t, &got)
	if got["id"] != "dp" || r.fake.VCP[0x60].Current != 5 {
		t.Fatalf("JSON %v, monitor value %d", got, r.fake.VCP[0x60].Current)
	}
}

func TestInputUnknownNameWritesNothing(t *testing.T) {
	r := newRig(msi)
	r.run(t, 1, "input", "vga")

	if r.fake.VCP[0x60].Current != 7 {
		t.Fatal("the input value changed")
	}
	if !strings.Contains(r.errOut.String(), "vga") {
		t.Fatalf("stderr = %q", r.errOut)
	}
}

func TestInputWithoutProfile(t *testing.T) {
	r := newRig(unknown)

	r.run(t, 0, "input")
	if out := r.out.String(); !strings.Contains(out, "value 7") || !strings.Contains(out, "-raw") {
		t.Fatalf("output = %q", out)
	}

	r.run(t, 1, "input", "dp")
	if !strings.Contains(r.errOut.String(), "-raw") || r.fake.VCP[0x60].Current != 7 {
		t.Fatalf("stderr = %q, value %d", r.errOut, r.fake.VCP[0x60].Current)
	}

	r.run(t, 0, "input", "-raw", "15")
	if r.fake.VCP[0x60].Current != 15 {
		t.Fatalf("monitor got %d, want 15", r.fake.VCP[0x60].Current)
	}

	r.run(t, 0, "input", "-raw", "0x11", "-json")
	if r.fake.VCP[0x60].Current != 0x11 {
		t.Fatalf("monitor got %d, want 17", r.fake.VCP[0x60].Current)
	}
}

func TestInputRejectsBadUse(t *testing.T) {
	for _, args := range [][]string{
		{"input", "a", "b"},
		{"input", "-raw"},
		{"input", "-raw", "dp"},
		{"input", "-raw", "256"},
	} {
		r := newRig(msi)
		r.run(t, 2, args...)
		if len(r.opened) != 0 {
			t.Errorf("monctl %v opened the monitor", args)
		}
	}
}

func TestProbeLeavesTheSerialOut(t *testing.T) {
	r := newRig(msi)
	r.run(t, 0, "probe")

	out := r.out.String()
	for _, want := range []string{"MPG 274U E16M", "MSI", "0x3cf0", "MSI MPG 274URDFW E16M", "50 (max 100)", "7 (max 14)", "(prot(monitor)vcp(10 60))"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output %q does not contain %q", out, want)
		}
	}
	if strings.Contains(out, "SECRET123") {
		t.Fatal("probe printed the serial number")
	}

	r.run(t, 0, "probe", "-json")
	var got probeOutput
	r.decode(t, &got)
	if got.Profile != "MSI MPG 274URDFW E16M" || strings.Contains(r.out.String(), "SECRET123") {
		t.Fatalf("unexpected JSON %+v", got)
	}
}

func TestProbeUnknownMonitor(t *testing.T) {
	r := newRig(unknown)
	r.run(t, 0, "probe")
	if !strings.Contains(r.out.String(), "Profile:       none") {
		t.Fatalf("output = %q", r.out)
	}
	r.run(t, 2, "probe", "extra")
}

func TestListJSONAndProfileColumn(t *testing.T) {
	r := newRig(msi, unknown)
	r.run(t, 0, "list")
	if out := r.out.String(); !strings.Contains(out, "MSI MPG 274URDFW E16M") || !strings.Contains(out, "PROFILE") {
		t.Fatalf("output = %q", out)
	}

	r.run(t, 0, "list", "-json")
	var got []listEntry
	r.decode(t, &got)
	if len(got) != 2 || got[0].ProductCode != "0x3cf0" || got[0].Profile == "" || got[1].Profile != "" {
		t.Fatalf("unexpected JSON %+v", got)
	}
	if strings.Contains(r.out.String(), "SECRET123") {
		t.Fatal("list -json printed the serial number outside the id")
	}

	r = newRig()
	r.run(t, 0, "list", "-json")
	if got := strings.TrimSpace(r.out.String()); got != "[]" {
		t.Fatalf("empty list JSON = %q, want []", got)
	}
	r.run(t, 2, "list", "extra")
}

func TestListPrintsDashForEmptyConnector(t *testing.T) {
	withConnector := msi
	withConnector.Connector = "DP-1"
	noConnector := unknown
	noConnector.Bus = "avservice:42"
	r := newRig(withConnector, noConnector)

	r.run(t, 0, "list")
	rows := map[string][]string{}
	for _, line := range strings.Split(strings.TrimSpace(r.out.String()), "\n")[1:] {
		fields := strings.Fields(line)
		rows[fields[0]] = fields
	}
	if got := rows["msi-3cf0"]; len(got) < 2 || !slices.Contains(got, "DP-1") {
		t.Fatalf("msi row = %q, want connector DP-1", got)
	}
	want := []string{"xyz-0001", "Mystery", "-", "avservice:42", "-"}
	if got := rows["xyz-0001"]; strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("row = %q, want %q", got, want)
	}

	r.run(t, 0, "list", "-json")
	var entries []listEntry
	r.decode(t, &entries)
	if len(entries) != 2 || entries[0].Connector != "DP-1" || entries[1].Connector != "" {
		t.Fatalf("JSON connectors = %+v, want DP-1 and empty", entries)
	}
}
