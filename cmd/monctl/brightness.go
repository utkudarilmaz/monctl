package main

import (
	"fmt"
	"strconv"

	"github.com/utkudarilmaz/monctl/internal/monitor"
)

type brightnessOutput struct {
	Display    string `json:"display"`
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

	if opts.json {
		return a.printJSON(brightnessOutput{Display: s.display.ID, Brightness: level.Current, Max: level.Max})
	}
	_, _ = fmt.Fprintln(a.stdout, level.Current)
	return 0
}
