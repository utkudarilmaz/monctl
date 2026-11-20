package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/utkudarilmaz/monctl/internal/display"
	"github.com/utkudarilmaz/monctl/internal/edid"
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
		stdout: &out,
		stderr: &errOut,
		detect: func() ([]display.Display, error) { return displays, err },
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
