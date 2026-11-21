package main

import (
	"fmt"
	"sync/atomic"

	"github.com/utkudarilmaz/monctl/internal/dimmer"
	hotkeys "github.com/utkudarilmaz/monctl/internal/keys"
	"github.com/utkudarilmaz/monctl/internal/remote"
)

// keyStep is how far one press of a brightness key moves, in percent.
const keyStep = 5

// trayMonitor lets the dimmer reach the monitor through the tray controller.
type trayMonitor struct{ trayController }

func (m trayMonitor) Level() (int, int, error) {
	var current, limit int
	err := m.with(func(s *session) error {
		level, err := s.monitor.Brightness()
		current, limit = level.Current, level.Max
		return err
	})
	return current, limit, err
}

func (m trayMonitor) Write(value int) error {
	return m.with(func(s *session) error { return s.monitor.WriteBrightness(value) })
}

// serveKeys makes the running tray the fast way to change the brightness: it
// answers `monctl brightness` commands and, on macOS, the brightness keys.
func serveKeys(ctl trayController, report func(error)) func() {
	d := dimmer.New(trayMonitor{ctl}, report)
	go d.Probe()

	stop, err := remote.Serve(remote.Path(), handleRemote(d))
	if err != nil {
		report(err)
		stop = func() {}
	}

	var probing atomic.Bool
	missed := func() {
		if probing.CompareAndSwap(false, true) {
			go func() {
				d.Probe()
				probing.Store(false)
			}()
		}
	}
	step := func(direction int) { go func() { _, _, _ = d.StepPercent(direction * keyStep) }() }
	if err := hotkeys.Start(step, d.Usable, missed); err != nil {
		report(err)
	}
	return stop
}

func handleRemote(d *dimmer.Dimmer) remote.Handler {
	return func(command string, value int) (int, int, error) {
		switch command {
		case "step":
			return d.Step(value)
		case "set":
			return d.Set(value)
		default:
			return 0, 0, fmt.Errorf("unknown command %q", command)
		}
	}
}
