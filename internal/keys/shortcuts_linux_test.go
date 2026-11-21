package keys

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeSettings struct {
	values map[string]string
	sets   []string
	broken bool
}

func (f *fakeSettings) run(args ...string) (string, error) {
	if f.broken {
		return "", errors.New("no such schema")
	}
	key := strings.Join(args[1:3], " ")
	if args[0] == "set" {
		f.values[key] = args[3]
		f.sets = append(f.sets, key+" = "+args[3])
		return "", nil
	}
	if v, ok := f.values[key]; ok {
		return v, nil
	}
	return "@as []", nil
}

const (
	guake    = customRoot + "custom0/"
	brighter = customRoot + "monctl-brighter/"
	dimmer   = customRoot + "monctl-dimmer/"
)

func gnome() *fakeSettings {
	return &fakeSettings{values: map[string]string{
		mediaKeys + " " + customList:                 "['" + guake + "']",
		customKey + guake + " binding":               "'Home'",
		mediaKeys + " screen-brightness-up-static":   "['XF86MonBrightnessUp']",
		mediaKeys + " screen-brightness-down-static": "['XF86MonBrightnessDown']",
	}}
}

func marker(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "state", "shortcuts")
}

func TestRegisterAddsBothShortcutsAndKeepsTheOthers(t *testing.T) {
	f, m := gnome(), marker(t)

	if err := register(f.run, "/usr/bin/monctl", m, false); err != nil {
		t.Fatal(err)
	}

	want := "['" + guake + "', '" + brighter + "', '" + dimmer + "']"
	if got := f.values[mediaKeys+" "+customList]; got != want {
		t.Fatalf("list = %s, want %s", got, want)
	}
	for key, want := range map[string]string{
		customKey + brighter + " command":            "'/usr/bin/monctl brightness +5'",
		customKey + brighter + " binding":            "'XF86MonBrightnessUp'",
		customKey + dimmer + " command":              "'/usr/bin/monctl brightness -5'",
		customKey + dimmer + " name":                 "'monctl dimmer'",
		mediaKeys + " screen-brightness-up-static":   "[]",
		mediaKeys + " screen-brightness-down-static": "[]",
	} {
		if f.values[key] != want {
			t.Errorf("%s = %q, want %q", key, f.values[key], want)
		}
	}
	if _, err := os.Stat(m); err != nil {
		t.Fatal("the marker file was not written")
	}
}

func TestRegisterRunsOnlyOnce(t *testing.T) {
	f, m := gnome(), marker(t)
	_ = register(f.run, "/usr/bin/monctl", m, false)
	f.values[mediaKeys+" "+customList] = "['" + guake + "']"
	f.sets = nil

	if err := register(f.run, "/usr/bin/monctl", m, false); err != nil {
		t.Fatal(err)
	}
	if len(f.sets) != 0 {
		t.Fatalf("a removed shortcut came back: %v", f.sets)
	}
}

func TestRegisterLeavesAComputerWithItsOwnBacklightAlone(t *testing.T) {
	f, m := gnome(), marker(t)

	if err := register(f.run, "/usr/bin/monctl", m, true); err != nil {
		t.Fatal(err)
	}
	if len(f.sets) != 0 {
		t.Fatalf("settings were changed: %v", f.sets)
	}
	if _, err := os.Stat(m); err == nil {
		t.Fatal("the marker was written, so a later check would be skipped")
	}
}

func TestRegisterLeavesAKeyTheUserBoundAlone(t *testing.T) {
	f, m := gnome(), marker(t)
	f.values[customKey+guake+" binding"] = "'XF86MonBrightnessUp'"

	if err := register(f.run, "/usr/bin/monctl", m, false); err != nil {
		t.Fatal(err)
	}

	want := "['" + guake + "', '" + dimmer + "']"
	if got := f.values[mediaKeys+" "+customList]; got != want {
		t.Fatalf("list = %s, want %s", got, want)
	}
	if got := f.values[mediaKeys+" screen-brightness-up-static"]; got != "['XF86MonBrightnessUp']" {
		t.Fatalf("the key GNOME holds was freed: %s", got)
	}
}

func TestRegisterKeepsShortcutsThatAreAlreadyThere(t *testing.T) {
	f, m := gnome(), marker(t)
	f.values[mediaKeys+" "+customList] = "['" + guake + "', '" + brighter + "', '" + dimmer + "']"
	f.values[customKey+brighter+" command"] = "'monctl brightness +10'"

	if err := register(f.run, "/usr/bin/monctl", m, false); err != nil {
		t.Fatal(err)
	}
	if len(f.sets) != 0 {
		t.Fatalf("existing shortcuts were changed: %v", f.sets)
	}
}

func TestRegisterDoesNothingWithoutGnomeSettings(t *testing.T) {
	f, m := &fakeSettings{broken: true}, marker(t)

	if err := register(f.run, "/usr/bin/monctl", m, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(m); err == nil {
		t.Fatal("the marker was written without GNOME settings")
	}
}

func TestRegisterLeavesAListItCannotReadAlone(t *testing.T) {
	f, m := gnome(), marker(t)
	f.values[mediaKeys+" "+customList] = `["/it's/", '` + guake + `']`

	if err := register(f.run, "/usr/bin/monctl", m, false); err != nil {
		t.Fatal(err)
	}
	if len(f.sets) != 0 {
		t.Fatalf("settings were changed: %v", f.sets)
	}
}

func TestWord(t *testing.T) {
	for path, want := range map[string]string{
		"/usr/bin/monctl":       "/usr/bin/monctl",
		"/opt/my apps/monctl":   "'/opt/my apps/monctl'",
		"/opt/it's 100%/monctl": `'/opt/it'\''s 100%%/monctl'`,
	} {
		if got := word(path); got != want {
			t.Errorf("word(%q) = %s, want %s", path, got, want)
		}
	}
}

func TestQuote(t *testing.T) {
	if got := quote(`/opt/it's/monctl`); got != `'/opt/it\'s/monctl'` {
		t.Fatalf("quote = %s", got)
	}
}

func TestList(t *testing.T) {
	for value, want := range map[string]int{"@as []": 0, "['a']": 1, "['a', 'b']": 2, "'Home'": 1, "": 0} {
		if got := len(list(value)); got != want {
			t.Errorf("list(%q) has %d items, want %d", value, got, want)
		}
	}
}

func TestRegisterKeepsOtherKeysGnomeHoldsForBrightness(t *testing.T) {
	f, m := gnome(), marker(t)
	f.values[mediaKeys+" screen-brightness-up-static"] = "['XF86MonBrightnessUp', 'F12']"

	if err := register(f.run, "/usr/bin/monctl", m, false); err != nil {
		t.Fatal(err)
	}
	if got := f.values[mediaKeys+" screen-brightness-up-static"]; got != "['F12']" {
		t.Fatalf("static keys = %s, want ['F12']", got)
	}
}
