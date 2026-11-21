package main

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/utkudarilmaz/monctl/internal/config"
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
