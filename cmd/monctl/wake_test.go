package main

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/utkudarilmaz/monctl/internal/bus/bustest"
	"github.com/utkudarilmaz/monctl/internal/config"
	"github.com/utkudarilmaz/monctl/internal/dimmer"
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

func TestInputSwitchesWithAWarningWhenTheBeforeCommandFails(t *testing.T) {
	r := newRig(msi)
	before(r, "usbc", "ssh", "macbook", "monctl", "wake")
	r.app.exec = func([]string) error { return errors.New("ssh: host is down") }

	r.run(t, 0, "input", "usbc")

	if r.fake.VCP[0x60].Current != 8 {
		t.Fatal("the monitor was not switched")
	}
	for _, want := range []string{"switching anyway", "host is down"} {
		if !strings.Contains(r.errOut.String(), want) {
			t.Fatalf("stderr %q does not contain %q", r.errOut, want)
		}
	}
}

// bounce makes the fake monitor return to its old input during the first n
// waits after a switch, as a monitor does when the new input has no picture.
func bounce(r *rig, n int) *int {
	waits := 0
	r.app.pause = func(time.Duration) {
		waits++
		if n > 0 && r.fake.VCP[0x60].Current != 7 {
			n--
			r.fake.VCP[0x60] = bustest.VCP{Current: 7, Max: 14}
		}
	}
	return &waits
}

func switches(r *rig) int {
	count := 0
	for _, w := range r.fake.Writes {
		if len(w) == 7 && w[2] == 0x03 && w[3] == 0x60 {
			count++
		}
	}
	return count
}

func TestSwitchIsTriedAgainWhenTheMonitorComesBack(t *testing.T) {
	r := newRig(msi)
	ran := before(r, "usbc", "wake-the-mac")
	bounce(r, 1)

	r.run(t, 0, "input", "usbc")

	if got := switches(r); got != 2 {
		t.Fatalf("the monitor was switched %d times, want 2", got)
	}
	if len(*ran) != 1 {
		t.Fatalf("the before command ran %d times, want 1", len(*ran))
	}
	if r.fake.VCP[0x60].Current != 8 {
		t.Fatalf("monitor ended on input value %d, want 8", r.fake.VCP[0x60].Current)
	}
}

func TestSwitchGivesUpWhenTheMonitorKeepsComingBack(t *testing.T) {
	r := newRig(msi)
	bounce(r, 100)

	r.run(t, 1, "input", "usbc")

	if got := switches(r); got != switchAttempts {
		t.Fatalf("the monitor was switched %d times, want %d", got, switchAttempts)
	}
	if !strings.Contains(r.errOut.String(), "came back") {
		t.Fatalf("stderr = %q", r.errOut)
	}
}

func TestSwitchWaitsWhileTheMonitorIsSilent(t *testing.T) {
	r := newRig(msi)
	waits := bounce(r, 0)
	r.app.pause = func(time.Duration) { *waits++; r.fake.Asleep = true }

	r.run(t, 0, "input", "usbc")

	if *waits != switchChecks || switches(r) != 1 {
		t.Fatalf("waited %d times and switched %d times, want %d and 1", *waits, switches(r), switchChecks)
	}
}

func TestSwitchToTheCurrentInputDoesNotWait(t *testing.T) {
	r := newRig(msi)
	ran := before(r, "dp", "wake-myself")
	waits := bounce(r, 0)

	r.run(t, 0, "input", "dp")

	if *waits != 0 || len(*ran) != 0 || switches(r) != 1 {
		t.Fatalf("waits %d, before ran %d, switches %d; want 0, 0, 1", *waits, len(*ran), switches(r))
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

	r = newRig(msi)
	bounce(r, 100)
	if err := (trayController{app: r.app}).SetInput("usbc"); !errors.Is(err, errCameBack) {
		t.Fatalf("SetInput error = %v, want errCameBack", err)
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
	if err := runCommand([]string{"true"}, beforeTimeout); err != nil {
		t.Fatalf("true failed: %v", err)
	}
	err := runCommand([]string{"sh", "-c", "echo first; echo broken pipe >&2; exit 3"}, beforeTimeout)
	if err == nil || !strings.Contains(err.Error(), "broken pipe") {
		t.Fatalf("error = %v, want the last output line", err)
	}
	err = runCommand([]string{"sh", "-c", "exit 3"}, beforeTimeout)
	if err == nil || !strings.Contains(err.Error(), "exit status 3") {
		t.Fatalf("error = %v, want the exit status", err)
	}
	if err := runCommand([]string{"monctl-no-such-program"}, beforeTimeout); err == nil {
		t.Fatal("a missing program must fail")
	}
}

func TestRunCommandStopsAtTheTimeout(t *testing.T) {
	start := time.Now()
	err := runCommand([]string{"sh", "-c", "sleep 5; true"}, 100*time.Millisecond)

	if err == nil || !strings.Contains(err.Error(), "did not finish") {
		t.Fatalf("error = %v, want a timeout", err)
	}
	if took := time.Since(start); took > 3*time.Second {
		t.Fatalf("returned after %s, a child that held the output kept it waiting", took)
	}
}

func TestRunCommandPrintsOnlySafeOutput(t *testing.T) {
	err := runCommand([]string{"sh", "-c", `head -c 100000 /dev/zero; printf '\nbad\033[2Jline\n'; exit 1`}, beforeTimeout)

	if err == nil || err.Error() != "sh: bad[2Jline" {
		t.Fatalf("error = %q, want the last line without the escape character", err)
	}
}

func TestTailKeepsOnlyTheEnd(t *testing.T) {
	var out tail
	for range 100 {
		_, _ = out.Write([]byte(strings.Repeat("x", 1000)))
	}
	_, _ = out.Write([]byte("end"))

	if len(out) != outputKept || cap(out) > 64*outputKept || !strings.HasSuffix(string(out), "xend") {
		t.Fatalf("kept %d bytes with room for %d", len(out), cap(out))
	}
}

func TestLastLine(t *testing.T) {
	for in, want := range map[string]string{
		"":                         "",
		"one\n":                    "one",
		"one\n  two  \n\n":         "two",
		"one\ntwo\x1b]0;title\a\r": "two]0;title",
		strings.Repeat("é", 500):   strings.Repeat("é", messageRunes),
	} {
		if got := lastLine(in); got != want {
			t.Errorf("lastLine(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestBrightnessChangeGoesToTheRunningTray(t *testing.T) {
	r := newRig(msi)
	var asked []string
	r.app.remote = func(command string, value int) (int, int, error) {
		asked = append(asked, fmt.Sprintf("%s %d", command, value))
		return 42, 100, nil
	}

	r.run(t, 0, "brightness", "+10")
	r.run(t, 0, "brightness", "-10")
	r.run(t, 0, "brightness", "30")
	if got := r.out.String(); got != "42\n" {
		t.Fatalf("output = %q, want the value the tray reported", got)
	}
	if want := []string{"step 10", "step -10", "set 30"}; !reflect.DeepEqual(asked, want) || len(r.opened) != 0 {
		t.Fatalf("asked %v, opened %v; want %v and no monitor opened", asked, r.opened, want)
	}

	r.run(t, 0, "brightness", "-json", "+5")
	var got map[string]any
	r.decode(t, &got)
	if got["brightness"] != float64(42) || got["max"] != float64(100) {
		t.Fatalf("JSON = %v", got)
	}
}

func TestBrightnessSkipsTheTrayForReadsAndPickedMonitors(t *testing.T) {
	r := newRig(msi)
	asked := 0
	r.app.remote = func(string, int) (int, int, error) { asked++; return 0, 0, nil }

	r.run(t, 0, "brightness")
	r.run(t, 0, "brightness", "-d", "msi-3cf0", "+10")
	if asked != 0 || len(r.opened) != 2 {
		t.Fatalf("asked the tray %d times, opened %v", asked, r.opened)
	}
}

func TestBrightnessTrayErrorIsShown(t *testing.T) {
	r := newRig(msi)
	r.app.remote = func(string, int) (int, int, error) { return 0, 0, errors.New("monitor is silent") }

	r.run(t, 1, "brightness", "+10")
	if !strings.Contains(r.errOut.String(), "monitor is silent") || len(r.opened) != 0 {
		t.Fatalf("stderr %q, opened %v", r.errOut, r.opened)
	}
}

func TestHandleRemote(t *testing.T) {
	r := newRig(msi)
	d := dimmer.New(trayMonitor{trayController{app: r.app}}, func(error) {})
	handle := handleRemote(d)

	if current, limit, err := handle("step", 10); err != nil || current != 60 || limit != 100 {
		t.Fatalf("step = %d, %d, %v", current, limit, err)
	}
	d.Wait()
	if current, _, err := handle("set", 20); err != nil || current != 20 {
		t.Fatalf("set = %d, %v", current, err)
	}
	d.Wait()
	if r.fake.VCP[0x10].Current != 20 {
		t.Fatalf("monitor at %d, want 20", r.fake.VCP[0x10].Current)
	}
	if _, _, err := handle("reboot", 1); err == nil {
		t.Fatal("an unknown command must fail")
	}
}
