package main

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/utkudarilmaz/monctl/internal/bus/bustest"
	"github.com/utkudarilmaz/monctl/internal/config"
	"github.com/utkudarilmaz/monctl/internal/ddc"
	"github.com/utkudarilmaz/monctl/internal/tray"
)

func TestTrayControllerState(t *testing.T) {
	r := newRig(msi)
	r.app.config = func() (config.Config, error) {
		return config.Config{Inputs: map[string]config.Input{"usbc": {Label: "MacBook"}}}, nil
	}
	ctl := trayController{app: r.app}

	state, err := ctl.State()
	if err != nil {
		t.Fatal(err)
	}
	want := tray.State{
		Brightness: 50, Max: 100, Current: "dp",
		Inputs: []tray.Input{{ID: "dp", Label: "DisplayPort"}, {ID: "usbc", Label: "MacBook"}},
	}
	if !reflect.DeepEqual(state, want) {
		t.Fatalf("State = %+v, want %+v", state, want)
	}
	if !r.fake.Closed {
		t.Fatal("the bus must be closed after each action")
	}
}

func TestTrayControllerActions(t *testing.T) {
	r := newRig(msi)
	ctl := trayController{app: r.app}

	if err := ctl.AdjustBrightness(tray.Step); err != nil || r.fake.VCP[0x10].Current != 60 {
		t.Fatalf("AdjustBrightness: %v, brightness %d", err, r.fake.VCP[0x10].Current)
	}
	if err := ctl.SetBrightness(25); err != nil || r.fake.VCP[0x10].Current != 25 {
		t.Fatalf("SetBrightness: %v, brightness %d", err, r.fake.VCP[0x10].Current)
	}
	if err := ctl.SetInput("usbc"); err != nil || r.fake.VCP[0x60].Current != 8 {
		t.Fatalf("SetInput: %v, input %d", err, r.fake.VCP[0x60].Current)
	}
	if err := ctl.SetInput("vga"); err == nil {
		t.Fatal("an unknown input must fail")
	}
}

func TestTrayControllerSetBrightnessUsesTheMonitorMaximum(t *testing.T) {
	r := newRig(msi)
	r.fake.VCP[0x10] = bustest.VCP{Current: 10, Max: 50}

	if err := (trayController{app: r.app}).SetBrightness(50); err != nil || r.fake.VCP[0x10].Current != 25 {
		t.Fatalf("SetBrightness(50%%) on max 50: %v, brightness %d, want 25", err, r.fake.VCP[0x10].Current)
	}
}

func TestTrayControllerErrors(t *testing.T) {
	r := newRig()
	ctl := trayController{app: r.app}
	if _, err := ctl.State(); !errors.Is(err, tray.ErrNoMonitor) {
		t.Fatalf("State with no monitor: %v, want ErrNoMonitor", err)
	}

	r = newRig(msi)
	r.fake.Asleep = true
	ctl = trayController{app: r.app}
	if _, err := ctl.State(); !errors.Is(err, ddc.ErrNoReply) {
		t.Fatalf("State with a silent monitor: %v, want ErrNoReply", err)
	}
	if err := ctl.SetBrightness(10); !errors.Is(err, ddc.ErrNoReply) {
		t.Fatalf("SetBrightness with a silent monitor: %v", err)
	}
	if err := ctl.AdjustBrightness(10); !errors.Is(err, ddc.ErrNoReply) {
		t.Fatalf("AdjustBrightness with a silent monitor: %v", err)
	}
}

func TestTrayControllerPicksTheDisplay(t *testing.T) {
	r := newRig(msi, unknown)
	if _, err := (trayController{app: r.app}).State(); err == nil {
		t.Fatal("two monitors without -d must fail")
	}
	if _, err := (trayController{app: r.app, display: "xyz-0001"}).State(); err != nil {
		t.Fatal(err)
	}
	if got := r.opened; len(got) != 1 || got[0] != "xyz-0001" {
		t.Fatalf("opened %v", got)
	}
}

func TestTrayCommand(t *testing.T) {
	r := newRig(msi)
	var got tray.Controller
	r.app.runTray = func(c tray.Controller, report func(error)) error {
		got = c
		report(errors.New("monitor went away"))
		return nil
	}

	r.run(t, 0, "tray", "-d", "msi-3cf0")
	if c, ok := got.(trayController); !ok || c.display != "msi-3cf0" {
		t.Fatalf("tray got controller %+v", got)
	}
	if !strings.Contains(r.errOut.String(), "monctl: monitor went away\n") {
		t.Fatalf("stderr = %q, want the reported error", r.errOut)
	}

	for _, args := range [][]string{{"tray", "now"}, {"tray", "-json"}, {"tray", "-raw"}} {
		r.run(t, 2, args...)
	}

	r.app.runTray = func(tray.Controller, func(error)) error { return errors.New("no tray here") }
	r.run(t, 1, "tray")
	if !strings.Contains(r.errOut.String(), "no tray here") {
		t.Fatalf("stderr = %q", r.errOut)
	}
}
