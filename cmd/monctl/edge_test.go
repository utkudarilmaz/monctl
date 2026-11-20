package main

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/utkudarilmaz/monctl/internal/bus/bustest"
	"github.com/utkudarilmaz/monctl/internal/config"
	"github.com/utkudarilmaz/monctl/internal/display"
	"github.com/utkudarilmaz/monctl/internal/edid"
	"github.com/utkudarilmaz/monctl/internal/profile"
)

func TestParseOptions(t *testing.T) {
	tests := []struct {
		args []string
		want options
	}{
		{nil, options{}},
		{[]string{"-d", "a", "40"}, options{display: "a", args: []string{"40"}}},
		{[]string{"40", "-d", "a"}, options{display: "a", args: []string{"40"}}},
		{[]string{"-d=a", "-10"}, options{display: "a", args: []string{"-10"}}},
		{[]string{"-10", "-json"}, options{json: true, args: []string{"-10"}}},
		{[]string{"-json", "+0"}, options{json: true, args: []string{"+0"}}},
		{[]string{"-0"}, options{args: []string{"-0"}}},
		{[]string{""}, options{args: []string{""}}},
		{[]string{"-"}, options{args: []string{"-"}}},
		{[]string{"-json=false", "1"}, options{args: []string{"1"}}},
		{[]string{"15", "-raw", "-json"}, options{json: true, raw: true, args: []string{"15"}}},
		{[]string{"-d", "-5"}, options{display: "-5"}},
	}
	for _, tt := range tests {
		got, err := parseOptions("test", tt.args)
		if err != nil {
			t.Errorf("parseOptions(%q) error = %v", tt.args, err)
			continue
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("parseOptions(%q) = %+v, want %+v", tt.args, got, tt.want)
		}
	}
}

func TestParseOptionsErrors(t *testing.T) {
	for _, args := range [][]string{
		{"-d"},
		{"40", "-d"},
		{"-x"},
		{"-json=maybe"},
		{"-1.5"},
	} {
		if _, err := parseOptions("test", args); err == nil {
			t.Errorf("parseOptions(%q) gave no error", args)
		}
	}
}

func TestBrightnessZeroChangeKeepsTheValue(t *testing.T) {
	for _, arg := range []string{"+0", "-0"} {
		r := newRig(msi)
		r.run(t, 0, "brightness", arg)
		if got := r.out.String(); got != "50\n" || r.fake.VCP[0x10].Current != 50 {
			t.Errorf("brightness %s printed %q, monitor has %d, want 50", arg, got, r.fake.VCP[0x10].Current)
		}
	}

	r := newRig(msi)
	r.run(t, 0, "brightness", "0")
	if r.fake.VCP[0x10].Current != 0 {
		t.Fatalf("brightness 0 left %d", r.fake.VCP[0x10].Current)
	}
}

func TestBrightnessUsesTheMonitorMax(t *testing.T) {
	tests := []struct {
		args    []string
		current int
		max     int
	}{
		{[]string{"brightness"}, 25, 50},
		{[]string{"brightness", "80"}, 50, 50},
		{[]string{"brightness", "10"}, 10, 50},
		{[]string{"brightness", "+100"}, 50, 50},
		{[]string{"brightness", "-100"}, 0, 50},
	}
	for _, tt := range tests {
		r := newRig(msi)
		r.fake.VCP[0x10] = bustest.VCP{Current: 25, Max: 50}
		r.run(t, 0, append(tt.args, "-json")...)

		var got brightnessOutput
		r.decode(t, &got)
		if got.Brightness != tt.current || got.Max != tt.max || int(r.fake.VCP[0x10].Current) != tt.current {
			t.Errorf("monctl %v = %+v, monitor has %d, want %d of %d", tt.args, got, r.fake.VCP[0x10].Current, tt.current, tt.max)
		}
	}
}

func TestBrightnessUsageErrorsOpenNothing(t *testing.T) {
	for _, args := range [][]string{
		{"brightness", ""},
		{"brightness", "-"},
		{"brightness", "+"},
		{"brightness", "1.5"},
		{"brightness", "99999999999999999999"},
		{"brightness", "40", "-d"},
		{"brightness", "-d"},
		{"brightness", "-json=maybe"},
	} {
		r := newRig(msi)
		r.run(t, 2, args...)
		if len(r.opened) != 0 || r.out.Len() != 0 {
			t.Errorf("monctl %q opened %v and printed %q", args, r.opened, r.out)
		}
		if r.errOut.Len() == 0 {
			t.Errorf("monctl %q printed nothing on stderr", args)
		}
	}
}

func TestBrightnessBusFailure(t *testing.T) {
	r := newRig(msi)
	r.fake.FailWrites = 3
	r.run(t, 1, "brightness", "10")
	if r.fake.VCP[0x10].Current != 50 {
		t.Fatalf("monitor has %d, want 50", r.fake.VCP[0x10].Current)
	}
}

func TestPickWithSeveralDisplays(t *testing.T) {
	third := display.Display{ID: "abc-0002", Info: edid.Info{Manufacturer: "ABC", ProductCode: 2}}
	for _, flag := range [][]string{{"-d", "xyz-0001"}, {"-d=xyz-0001"}} {
		r := newRig(msi, unknown, third)
		r.run(t, 0, append([]string{"brightness"}, flag...)...)
		if len(r.opened) != 1 || r.opened[0] != "xyz-0001" {
			t.Errorf("monctl brightness %q opened %v, want xyz-0001", flag, r.opened)
		}
	}

	r := newRig(msi, unknown, third)
	r.run(t, 1, "brightness", "-d", "MSI-3CF0")
	if len(r.opened) != 0 {
		t.Fatalf("an id in another case opened %v", r.opened)
	}
	r.run(t, 1, "input")
	if len(r.opened) != 0 || !strings.Contains(r.errOut.String(), "more than one") {
		t.Fatalf("opened %v, stderr %q", r.opened, r.errOut)
	}
}

func TestInputIDWinsOverAnotherInputsLabel(t *testing.T) {
	r := newRig(msi)
	r.app.config = func() (config.Config, error) {
		return config.Config{Inputs: map[string]config.Input{"usbc": {Label: "dp"}, "dp": {Label: "usbc"}}}, nil
	}
	for name, want := range map[string]uint16{"dp": 5, "DP": 5, "usbc": 8, "USBC": 8} {
		r.run(t, 0, "input", name)
		if r.fake.VCP[0x60].Current != want {
			t.Errorf("input %s wrote %d, want %d", name, r.fake.VCP[0x60].Current, want)
		}
	}
}

func TestInputConfigForMissingInputsIsIgnored(t *testing.T) {
	r := newRig(msi)
	r.app.config = func() (config.Config, error) {
		return config.Config{Inputs: map[string]config.Input{
			"hdmi9": {Label: "Console"},
			"dp":    {Hidden: true},
		}}, nil
	}

	r.run(t, 0, "input", "-json")
	var got inputOutput
	r.decode(t, &got)
	if got.Current.ID != "dp" || len(got.Inputs) != 1 || got.Inputs[0].ID != "usbc" {
		t.Fatalf("unexpected JSON %+v", got)
	}

	for _, name := range []string{"hdmi9", "console"} {
		r.run(t, 1, "input", name)
		if r.fake.VCP[0x60].Current != 7 {
			t.Fatalf("input %s wrote %d", name, r.fake.VCP[0x60].Current)
		}
	}
}

func TestInputEmptyNameIsUnknown(t *testing.T) {
	r := newRig(msi)
	r.run(t, 1, "input", "")
	if r.fake.VCP[0x60].Current != 7 || !strings.Contains(r.errOut.String(), `no input ""`) {
		t.Fatalf("value %d, stderr %q", r.fake.VCP[0x60].Current, r.errOut)
	}
}

func TestInputSwitchWriteFailure(t *testing.T) {
	r := newRig(msi)
	r.fake.FailWrites = 3
	r.run(t, 1, "input", "usbc")
	if r.fake.VCP[0x60].Current != 7 || !strings.Contains(r.errOut.String(), bustest.ErrInjected.Error()) {
		t.Fatalf("value %d, stderr %q", r.fake.VCP[0x60].Current, r.errOut)
	}

	r = newRig(unknown)
	r.fake.FailWrites = 3
	r.run(t, 1, "input", "-raw", "15")
	if r.fake.VCP[0x60].Current != 7 {
		t.Fatalf("value %d", r.fake.VCP[0x60].Current)
	}
}

func TestInputRawValues(t *testing.T) {
	tests := []struct {
		value string
		want  uint16
	}{{"0", 0}, {"255", 255}, {"0x0f", 15}, {"0X11", 17}, {"015", 15}}
	for _, tt := range tests {
		r := newRig(unknown)
		r.run(t, 0, "input", "-raw", tt.value)
		if r.fake.VCP[0x60].Current != tt.want {
			t.Errorf("input -raw %s wrote %d, want %d", tt.value, r.fake.VCP[0x60].Current, tt.want)
		}
	}
	for _, value := range []string{"-1", "+5", "", "0x100"} {
		r := newRig(unknown)
		r.run(t, 2, "input", "-raw", value)
		if len(r.opened) != 0 {
			t.Errorf("input -raw %q opened the monitor", value)
		}
	}
}

func keys(t *testing.T, r *rig) []string {
	t.Helper()
	var m map[string]any
	r.decode(t, &m)
	return sortedKeys(m)
}

func sortedKeys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func TestJSONShapes(t *testing.T) {
	tests := []struct {
		display display.Display
		args    []string
		want    []string
	}{
		{msi, []string{"brightness", "-json"}, []string{"brightness", "display", "max"}},
		{msi, []string{"input", "-json"}, []string{"current", "display", "inputs"}},
		{msi, []string{"input", "-json", "usbc"}, []string{"display", "id", "label"}},
		{unknown, []string{"input", "-json", "-raw", "0"}, []string{"display", "value"}},
		{msi, []string{"probe", "-json"}, []string{"brightness", "capabilities", "inputValue", "manufacturer", "name", "productCode", "profile"}},
	}
	for _, tt := range tests {
		r := newRig(tt.display)
		r.run(t, 0, tt.args...)
		if got := keys(t, r); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("monctl %v keys = %v, want %v", tt.args, got, tt.want)
		}
	}

	r := newRig(unknown)
	r.run(t, 0, "input", "-json", "-raw", "0")
	var raw map[string]any
	r.decode(t, &raw)
	if raw["value"] != float64(0) {
		t.Fatalf("raw switch JSON %v, want value 0", raw)
	}

	r.run(t, 0, "list", "-json")
	var list []map[string]any
	r.decode(t, &list)
	want := []string{"bus", "connector", "id", "manufacturer", "name", "productCode", "profile"}
	if len(list) != 1 || !reflect.DeepEqual(sortedKeys(list[0]), want) {
		t.Fatalf("list JSON %v, want keys %v", list, want)
	}
}

func TestInputJSONWithoutProfile(t *testing.T) {
	r := newRig(unknown)
	r.run(t, 0, "input", "-json")

	var got map[string]any
	r.decode(t, &got)
	current, _ := got["current"].(map[string]any)
	if !reflect.DeepEqual(sortedKeys(current), []string{"value"}) || current["value"] != float64(7) {
		t.Fatalf("current = %v, want only value 7", got["current"])
	}
	if inputs, ok := got["inputs"].([]any); !ok || len(inputs) != 0 {
		t.Fatalf("inputs = %#v, want an empty list", got["inputs"])
	}
}

func TestProbeWhenSingleReadsFail(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*bustest.Fake)
		check func(probeOutput) bool
	}{
		{"brightness read fails", func(f *bustest.Fake) { f.FailReads = 3 }, func(p probeOutput) bool {
			return strings.HasPrefix(p.Brightness, "failed: ") && p.InputValue == "7 (max 14)" && p.Capabilities != "" && !strings.HasPrefix(p.Capabilities, "failed")
		}},
		{"one corrupt reply is retried", func(f *bustest.Fake) { f.CorruptReplies = 1 }, func(p probeOutput) bool {
			return p.Brightness == "50 (max 100)"
		}},
		{"corrupt replies use every try", func(f *bustest.Fake) { f.CorruptReplies = 3 }, func(p probeOutput) bool {
			return strings.HasPrefix(p.Brightness, "failed: ") && p.InputValue == "7 (max 14)"
		}},
		{"input select not supported", func(f *bustest.Fake) { delete(f.VCP, 0x60) }, func(p probeOutput) bool {
			return p.Brightness == "50 (max 100)" && strings.Contains(p.InputValue, "does not support")
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newRig(msi)
			tt.setup(r.fake)
			r.run(t, 0, "probe", "-json")
			var got probeOutput
			r.decode(t, &got)
			if !tt.check(got) {
				t.Fatalf("unexpected probe %+v", got)
			}
			if r.errOut.Len() != 0 {
				t.Fatalf("stderr = %q", r.errOut)
			}
		})
	}
}

func TestProbeSilentMonitorFailsInJSONToo(t *testing.T) {
	r := newRig(msi)
	r.fake.Asleep = true
	r.run(t, 1, "probe", "-json")

	var got probeOutput
	r.decode(t, &got)
	if !strings.HasPrefix(got.Brightness, "failed: ") || got.Profile != "MSI MPG 274URDFW E16M" {
		t.Fatalf("unexpected probe %+v", got)
	}
	if !strings.Contains(r.errOut.String(), "did not answer") {
		t.Fatalf("stderr %q does not explain the silent monitor", r.errOut)
	}
}

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errors.New("pipe closed") }

func TestJSONWriteFailure(t *testing.T) {
	r := newRig(msi)
	r.app.stdout = failWriter{}
	r.run(t, 1, "brightness", "-json")
	if !strings.Contains(r.errOut.String(), "pipe closed") {
		t.Fatalf("stderr = %q", r.errOut)
	}
}

func TestProfilesErrorStopsEveryCommand(t *testing.T) {
	for _, args := range [][]string{{"list"}, {"brightness"}, {"input"}, {"probe"}} {
		r := newRig(msi)
		r.app.profiles = func() ([]profile.Profile, error) { return nil, errors.New("bad profile") }
		r.run(t, 1, args...)
		if !strings.Contains(r.errOut.String(), "bad profile") || len(r.opened) != 0 {
			t.Errorf("monctl %v: stderr %q, opened %v", args, r.errOut, r.opened)
		}
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

var msiInfo = edid.Info{Manufacturer: "MSI", ProductCode: 0x3cf0}

func TestLoadFromAMissingConfigDir(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "nothing"))

	all, err := loadProfiles()
	if err != nil {
		t.Fatal(err)
	}
	if p, ok := profile.Find(all, msiInfo); !ok || p.Name != "MSI MPG 274URDFW E16M" {
		t.Fatalf("Find = %+v, %v, want the shipped profile", p, ok)
	}
	cfg, err := loadConfig()
	if err != nil || len(cfg.Inputs) != 0 {
		t.Fatalf("loadConfig = %+v, %v", cfg, err)
	}
}

func TestLoadWithoutAHome(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")

	all, err := loadProfiles()
	if err != nil || len(all) == 0 {
		t.Fatalf("loadProfiles = %d profiles, %v, want the shipped ones", len(all), err)
	}
	cfg, err := loadConfig()
	if err != nil || len(cfg.Inputs) != 0 {
		t.Fatalf("loadConfig = %+v, %v", cfg, err)
	}
}

func TestUserProfileOverridesTheShippedOne(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", base)
	writeFile(t, filepath.Join(base, "monctl", "profiles", "msi.json"), `{
		"name": "My MSI",
		"match": {"manufacturer": "MSI", "productCode": "0x3cf0"},
		"inputs": [{"id": "hdmi1", "label": "HDMI", "write": 17, "read": 17}]
	}`)
	writeFile(t, filepath.Join(base, "monctl", config.FileName), `{"inputs": {"hdmi1": {"label": "Console"}}}`)

	all, err := loadProfiles()
	if err != nil {
		t.Fatal(err)
	}
	if p, ok := profile.Find(all, msiInfo); !ok || p.Name != "My MSI" {
		t.Fatalf("Find = %+v, %v, want the user profile", p, ok)
	}
	cfg, err := loadConfig()
	if err != nil || cfg.Inputs["hdmi1"].Label != "Console" {
		t.Fatalf("loadConfig = %+v, %v", cfg, err)
	}

	r := newRig(msi)
	r.app.profiles, r.app.config = loadProfiles, loadConfig
	r.run(t, 0, "input", "console")
	if r.fake.VCP[0x60].Current != 17 {
		t.Fatalf("monitor got %d, want 17", r.fake.VCP[0x60].Current)
	}
}

func TestBrokenUserFilesAreReported(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", base)
	writeFile(t, filepath.Join(base, "monctl", "profiles", "bad.json"), `{`)
	writeFile(t, filepath.Join(base, "monctl", config.FileName), `{"inputs": 1}`)

	if _, err := loadProfiles(); err == nil || !strings.Contains(err.Error(), "bad.json") {
		t.Fatalf("loadProfiles error = %v, want it to name bad.json", err)
	}
	if _, err := loadConfig(); err == nil || !strings.Contains(err.Error(), config.FileName) {
		t.Fatalf("loadConfig error = %v, want it to name %s", err, config.FileName)
	}
}
