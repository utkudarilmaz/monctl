package main

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/utkudarilmaz/monctl/internal/ddc"
	"github.com/utkudarilmaz/monctl/internal/monitor"
	"github.com/utkudarilmaz/monctl/internal/remote"
)

type brightnessOutput struct {
	Display    string `json:"display,omitempty"`
	Brightness int    `json:"brightness"`
	Max        int    `json:"max"`
}

func (a app) brightness(opts options) int {
	if len(opts.args) > 1 {
		return a.usageError("brightness takes one value, for example 40, +10 or -10")
	}
	var value int
	var relative bool
	if len(opts.args) == 1 {
		arg := opts.args[0]
		n, err := strconv.Atoi(arg)
		if err != nil {
			return a.usageError("brightness value %q is not a number", arg)
		}
		value, relative = n, arg[0] == '+' || arg[0] == '-'
	}

	if len(opts.args) == 1 && opts.display == "" {
		command := "set"
		if relative {
			command = "step"
		}
		current, limit, err := a.remote(command, value)
		if err == nil {
			return a.printBrightness(opts, "", monitor.Level{Current: current, Max: limit})
		}
		if err.Error() == silentReply {
			return a.fail(ddc.ErrNoReply)
		}
		if !errors.Is(err, remote.ErrNoServer) {
			return a.fail(err)
		}
	}

	s, err := a.connect(opts.display)
	if err != nil {
		return a.fail(err)
	}
	defer s.close()

	var level monitor.Level
	switch {
	case len(opts.args) == 0:
		level, err = s.monitor.Brightness()
	case relative:
		level, err = s.monitor.AdjustBrightness(value)
	default:
		level, err = s.monitor.SetBrightness(value)
	}
	if err != nil {
		return a.fail(err)
	}

	return a.printBrightness(opts, s.display.ID, level)
}

func (a app) printBrightness(opts options, display string, level monitor.Level) int {
	if opts.json {
		return a.printJSON(brightnessOutput{Display: display, Brightness: level.Current, Max: level.Max})
	}
	_, _ = fmt.Fprintln(a.stdout, level.Current)
	return 0
}
