package main

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/utkudarilmaz/monctl/internal/bus"
	"github.com/utkudarilmaz/monctl/internal/config"
	"github.com/utkudarilmaz/monctl/internal/ddc"
	"github.com/utkudarilmaz/monctl/internal/display"
	"github.com/utkudarilmaz/monctl/internal/edid"
	"github.com/utkudarilmaz/monctl/internal/profile"
	"github.com/utkudarilmaz/monctl/internal/remote"
	"github.com/utkudarilmaz/monctl/profiles"
)

func TestRunVersion(t *testing.T) {
	var out, errOut bytes.Buffer

	code := run([]string{"version"}, &out, &errOut)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if got, want := out.String(), "monctl dev\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestRunUnknownCommand(t *testing.T) {
	var out, errOut bytes.Buffer

	code := run([]string{"nope"}, &out, &errOut)

	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if errOut.Len() == 0 {
		t.Fatal("expected a message on stderr")
	}
}

func TestRunNoArgs(t *testing.T) {
	var out, errOut bytes.Buffer

	if code := run(nil, &out, &errOut); code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
}

func newApp(displays []display.Display, err error) (app, *bytes.Buffer, *bytes.Buffer) {
	var out, errOut bytes.Buffer
	a := app{
		stdout:   &out,
		stderr:   &errOut,
		detect:   func() ([]display.Display, error) { return displays, err },
		open:     func(display.Display) (bus.Bus, error) { return nil, errors.New("no bus in this test") },
		profiles: func() ([]profile.Profile, error) { return profile.Load(profiles.FS) },
		config:   func() (config.Config, error) { return config.Config{}, nil },
		ddc:      []ddc.Option{ddc.WithSleep(func(time.Duration) {})},
		pause:    func(time.Duration) {},
		lock:     func() func() { return func() {} },
		remote:   func(string, int) (int, int, error) { return 0, 0, remote.ErrNoServer },
		serve:    func(trayController, func(error)) func() { return func() {} },
	}
	return a, &out, &errOut
}

func TestListShowsDisplays(t *testing.T) {
	a, out, _ := newApp([]display.Display{{
		ID:        "msi-3cf0-0000000000000",
		Info:      edid.Info{Manufacturer: "MSI", ProductCode: 0x3cf0, Name: "MPG 274U E16M"},
		Connector: "DP-5",
		Bus:       "/dev/i2c-13",
	}}, nil)

	code := a.run([]string{"list"})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	for _, want := range []string{"msi-3cf0-0000000000000", "MPG 274U E16M", "DP-5", "/dev/i2c-13"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output %q does not contain %q", out.String(), want)
		}
	}
}

func TestListNothingAttached(t *testing.T) {
	a, out, _ := newApp(nil, nil)

	code := a.run([]string{"list"})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if got, want := out.String(), "No external monitor found.\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestListAccessDeniedExplainsTheFix(t *testing.T) {
	a, _, errOut := newApp(nil, display.ErrAccess)

	code := a.run([]string{"list"})

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(errOut.String(), "udev") {
		t.Fatalf("stderr %q does not explain the udev rule", errOut.String())
	}
}

func TestListOtherError(t *testing.T) {
	a, _, errOut := newApp(nil, errors.New("boom"))

	if code := a.run([]string{"list"}); code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(errOut.String(), "boom") {
		t.Fatalf("stderr %q does not show the error", errOut.String())
	}
}

func TestListShowsHeaderAndOneLinePerDisplay(t *testing.T) {
	a, out, _ := newApp([]display.Display{
		{ID: "msi-3cf0", Bus: "/dev/i2c-12"},
		{ID: "msi-3cf0-2", Connector: "DP-5", Bus: "/dev/i2c-13"},
	}, nil)

	if code := a.run([]string{"list"}); code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 3 || !strings.HasPrefix(lines[0], "ID") {
		t.Fatalf("output %q, want a header and two rows", out.String())
	}
	if !strings.Contains(lines[1], "/dev/i2c-12") || !strings.Contains(lines[2], "/dev/i2c-13") {
		t.Fatalf("rows out of order: %q", out.String())
	}
}

func TestRealAppHasEveryFunctionSet(t *testing.T) {
	v := reflect.ValueOf(realApp(io.Discard, io.Discard))
	for i := range v.NumField() {
		if f := v.Field(i); f.Kind() == reflect.Func && f.IsNil() {
			t.Errorf("app.%s is not set", v.Type().Field(i).Name)
		}
	}
}
