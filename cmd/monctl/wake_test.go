package main

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/utkudarilmaz/monctl/internal/config"
)

func before(r *rig, id string, argv ...string) *[][]string {
	var ran [][]string
	r.app.config = func() (config.Config, error) {
		return config.Config{Inputs: map[string]config.Input{id: {Before: argv}}}, nil
	}
	r.app.exec = func(argv []string) error {
		ran = append(ran, argv)
		return nil
	}
	return &ran
}

func TestInputRunsTheBeforeCommandThenSwitches(t *testing.T) {
	r := newRig(msi)
	ran := before(r, "usbc", "ssh", "macbook", "monctl", "wake")

	r.run(t, 0, "input", "USB-C")

	if want := [][]string{{"ssh", "macbook", "monctl", "wake"}}; !reflect.DeepEqual(*ran, want) {
		t.Fatalf("ran %v, want %v", *ran, want)
	}
	if r.fake.VCP[0x60].Current != 8 {
		t.Fatalf("monitor got input value %d, want 8", r.fake.VCP[0x60].Current)
	}
}

func TestInputDoesNotSwitchWhenTheBeforeCommandFails(t *testing.T) {
	r := newRig(msi)
	before(r, "usbc", "ssh", "macbook", "monctl", "wake")
	r.app.exec = func([]string) error { return errors.New("ssh: host is down") }

	r.run(t, 1, "input", "usbc")

	if r.fake.VCP[0x60].Current != 7 {
		t.Fatal("the monitor was switched although the before command failed")
	}
	for _, want := range []string{"not moved", "host is down"} {
		if !strings.Contains(r.errOut.String(), want) {
			t.Fatalf("stderr %q does not contain %q", r.errOut, want)
		}
	}
}

func TestBeforeCommandOnlyRunsForItsInput(t *testing.T) {
	r := newRig(msi)
	ran := before(r, "usbc", "true")

	r.run(t, 0, "input", "dp")
	r.run(t, 1, "input", "vga")
	r.run(t, 0, "input", "-raw", "8")
	r.run(t, 0, "input")

	if len(*ran) != 0 {
		t.Fatalf("ran %v, want nothing", *ran)
	}
}

func TestTraySwitchUsesTheBeforeCommand(t *testing.T) {
	r := newRig(msi)
	ran := before(r, "usbc", "wake-the-mac")
	ctl := trayController{app: r.app}

	if err := ctl.SetInput("usbc"); err != nil || len(*ran) != 1 || r.fake.VCP[0x60].Current != 8 {
		t.Fatalf("SetInput: %v, ran %v, input %d", err, *ran, r.fake.VCP[0x60].Current)
	}

	r.app.exec = func([]string) error { return errors.New("no route") }
	r.fake.VCP[0x60] = r.fake.VCP[0x10]
	ctl = trayController{app: r.app}
	if err := ctl.SetInput("usbc"); !errors.Is(err, errBefore) {
		t.Fatalf("SetInput error = %v, want errBefore", err)
	}
}

func TestWakeCommand(t *testing.T) {
	r := newRig()
	calls := 0
	r.app.wake = func() error { calls++; return nil }

	r.run(t, 0, "wake")
	if calls != 1 {
		t.Fatalf("wake ran %d times, want 1", calls)
	}
	for _, args := range [][]string{{"wake", "now"}, {"wake", "-json"}, {"wake", "-d", "x"}} {
		r.run(t, 2, args...)
	}

	r.app.wake = func() error { return errors.New("nobody is logged in") }
	r.run(t, 1, "wake")
	if !strings.Contains(r.errOut.String(), "nobody is logged in") {
		t.Fatalf("stderr = %q", r.errOut)
	}
}

func TestRunCommand(t *testing.T) {
	if err := runCommand([]string{"true"}); err != nil {
		t.Fatalf("true failed: %v", err)
	}
	err := runCommand([]string{"sh", "-c", "echo first; echo broken pipe >&2; exit 3"})
	if err == nil || !strings.Contains(err.Error(), "broken pipe") {
		t.Fatalf("error = %v, want the last output line", err)
	}
	if err := runCommand([]string{"sh", "-c", "exit 3"}); err == nil || !strings.Contains(err.Error(), "exit status 3") {
		t.Fatalf("error = %v, want the exit status", err)
	}
	if err := runCommand([]string{"monctl-no-such-program"}); err == nil {
		t.Fatal("a missing program must fail")
	}
}

func TestLastLine(t *testing.T) {
	for in, want := range map[string]string{"": "", "one\n": "one", "one\n  two  \n\n": "two"} {
		if got := lastLine(in); got != want {
			t.Errorf("lastLine(%q) = %q, want %q", in, got, want)
		}
	}
}
