package main

import (
	"errors"
	"fmt"

	"github.com/utkudarilmaz/monctl/internal/tray"
)

// trayController opens the monitor for each menu action and closes it again,
// so the tray holds nothing while it waits.
type trayController struct {
	app     app
	display string
}

func (c trayController) with(action func(*session) error) error {
	s, err := c.app.connect(c.display)
	if errors.Is(err, errNoMonitor) {
		return tray.ErrNoMonitor
	}
	if err != nil {
		return err
	}
	defer s.close()
	return action(s)
}

func (c trayController) State() (tray.State, error) {
	var state tray.State
	err := c.with(func(s *session) error {
		level, err := s.monitor.Brightness()
		if err != nil {
			return err
		}
		state.Brightness, state.Max = level.Current, level.Max
		for _, in := range s.monitor.Inputs() {
			state.Inputs = append(state.Inputs, tray.Input{ID: in.ID, Label: in.Label})
		}
		if src, err := s.monitor.Source(); err == nil && src.Known {
			state.Current = src.Input.ID
		}
		return nil
	})
	return state, err
}

func (c trayController) SetBrightness(percent int) error {
	return c.with(func(s *session) error {
		level, err := s.monitor.Brightness()
		if err != nil {
			return err
		}
		_, err = s.monitor.SetBrightness(tray.FromPercent(percent, level.Max))
		return err
	})
}

func (c trayController) AdjustBrightness(delta int) error {
	return c.with(func(s *session) error {
		_, err := s.monitor.AdjustBrightness(delta)
		return err
	})
}

func (c trayController) SetInput(id string) error {
	return c.with(func(s *session) error {
		_, err := c.app.switchTo(s, id)
		return err
	})
}

func (a app) tray(opts options) int {
	if len(opts.args) > 0 || opts.json || opts.raw {
		return a.usageError("tray takes only -d <id>")
	}
	report := func(err error) { _, _ = fmt.Fprintf(a.stderr, "monctl: %v\n", err) }
	if err := a.runTray(trayController{app: a, display: opts.display}, report); err != nil {
		return a.fail(err)
	}
	return 0
}
