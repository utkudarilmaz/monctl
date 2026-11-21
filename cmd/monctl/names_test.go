package main

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/utkudarilmaz/monctl/internal/bus/bustest"
	"github.com/utkudarilmaz/monctl/internal/config"
	"github.com/utkudarilmaz/monctl/internal/tray"
)

func named(t *testing.T, host string) *rig {
	t.Helper()
	r := newRig(msi)
	r.app.state = filepath.Join(t.TempDir(), "names.json")
	r.app.host = func() string { return host }
	return r
}

func TestInputShowsThisComputerOnTheActiveInput(t *testing.T) {
	r := named(t, "desk")

	r.run(t, 0, "input")

	for _, want := range []string{"Current: desk (dp)", "USB-C"} {
		if !strings.Contains(r.out.String(), want) {
			t.Fatalf("output %q does not contain %q", r.out, want)
		}
	}
}

func TestALabelFromTheSettingsWinsOverALearnedName(t *testing.T) {
	r := named(t, "desk")
	r.app.config = func() (config.Config, error) {
		return config.Config{Inputs: map[string]config.Input{"dp": {Label: "Workstation"}}}, nil
	}

	r.run(t, 0, "input")

	if !strings.Contains(r.out.String(), "Current: Workstation (dp)") {
		t.Fatalf("output %q does not show the label from the settings", r.out)
	}
}

func TestSwitchingLearnsTheNameOfTheOtherComputer(t *testing.T) {
	r := named(t, "desk")
	before(r, "usbc", "wake-the-mac")
	r.app.exec = func([]string) (string, error) { return "computer: laptop\n", nil }

	r.run(t, 0, "input", "usbc")
	if !strings.Contains(r.out.String(), "Switched to laptop (usbc).") {
		t.Fatalf("output %q does not name the other computer", r.out)
	}

	other := named(t, "desk")
	other.app.state = r.app.state
	other.run(t, 0, "input")
	for _, want := range []string{"Current: desk (dp)", "laptop"} {
		if !strings.Contains(other.out.String(), want) {
			t.Fatalf("output %q does not contain %q", other.out, want)
		}
	}
}

func TestTheNameIsLearnedEvenWhenTheBeforeCommandFails(t *testing.T) {
	r := named(t, "desk")
	before(r, "usbc", "wake-the-mac")
	r.app.exec = func([]string) (string, error) { return "computer: laptop\nno desktop\n", errors.New("failed") }

	r.run(t, 0, "input", "usbc")

	if !strings.Contains(r.out.String(), "Switched to laptop (usbc).") {
		t.Fatalf("output %q does not name the other computer", r.out)
	}
}

func TestWakePrintsTheComputerName(t *testing.T) {
	r := named(t, "desk")
	r.app.wake = func() error { return nil }

	r.run(t, 0, "wake")

	if r.out.String() != "computer: desk\n" {
		t.Fatalf("output is %q", r.out)
	}
}

func TestALearnedNameCannotPassForAnotherInput(t *testing.T) {
	for _, name := range []string{"DP", "displayport"} {
		r := named(t, "desk")
		before(r, "usbc", "wake-the-mac")
		r.app.exec = func([]string) (string, error) { return "computer: " + name + "\n", nil }

		r.run(t, 0, "input", "usbc")

		if strings.Contains(r.out.String(), name) {
			t.Fatalf("output %q shows the name %q of another input", r.out, name)
		}
	}
}

func TestTheNameMovesWhenTheMonitorShowsAnotherInput(t *testing.T) {
	r := named(t, "desk")
	r.run(t, 0, "input")

	r.fake.VCP[0x60] = bustest.VCP{Current: 8, Max: 14}
	r.run(t, 0, "input")

	for _, want := range []string{"Current: desk (usbc)", "DisplayPort"} {
		if !strings.Contains(r.out.String(), want) {
			t.Fatalf("output %q does not contain %q", r.out, want)
		}
	}
}

func TestALearnedNameCannotPassForALabelFromTheSettings(t *testing.T) {
	r := named(t, "MacBook")
	r.app.config = func() (config.Config, error) {
		return config.Config{Inputs: map[string]config.Input{"usbc": {Label: "macbook"}}}, nil
	}

	r.run(t, 0, "input")

	if !strings.Contains(r.out.String(), "Current: DisplayPort (dp)") {
		t.Fatalf("output %q gives dp the label of another input", r.out)
	}
}

func TestTheTrayShowsLearnedNames(t *testing.T) {
	r := named(t, "desk")

	state, err := (trayController{app: r.app}).State()
	if err != nil {
		t.Fatal(err)
	}
	want := []tray.Input{{ID: "dp", Label: "desk"}, {ID: "usbc", Label: "USB-C"}}
	if state.Current != "dp" || !reflect.DeepEqual(state.Inputs, want) {
		t.Fatalf("State = %+v, want dp current with inputs %+v", state, want)
	}
}

func TestANameThatCannotBeSavedIsStillShown(t *testing.T) {
	r := named(t, "desk")
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	r.app.state = filepath.Join(blocker, "names.json")

	r.run(t, 0, "input")

	if !strings.Contains(r.out.String(), "Current: desk (dp)") {
		t.Fatalf("output %q does not show the name", r.out)
	}
	if !strings.Contains(r.errOut.String(), "cannot remember the computer names") {
		t.Fatalf("stderr %q does not report the save error", r.errOut)
	}
}
