package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/utkudarilmaz/monctl/internal/monitor"
)

type inputEntry struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type currentInput struct {
	ID    string `json:"id,omitempty"`
	Label string `json:"label,omitempty"`
	Value uint16 `json:"value"`
}

type inputOutput struct {
	Display string       `json:"display"`
	Current currentInput `json:"current"`
	Inputs  []inputEntry `json:"inputs"`
}

type switchOutput struct {
	Display string `json:"display"`
	ID      string `json:"id,omitempty"`
	Label   string `json:"label,omitempty"`
	Value   *uint8 `json:"value,omitempty"`
}

func (a app) input(opts options) int {
	if len(opts.args) > 1 {
		return a.usageError("input takes one name")
	}
	if opts.raw && len(opts.args) == 0 {
		return a.usageError("-raw needs a value, for example: monctl input -raw 15")
	}
	var raw uint8
	if opts.raw {
		n, err := parseRaw(opts.args[0])
		if err != nil {
			return a.usageError("raw input value %q must be a number from 0 to 255", opts.args[0])
		}
		raw = uint8(n)
	}

	s, err := a.connect(opts.display)
	if err != nil {
		return a.fail(err)
	}
	defer s.close()

	switch {
	case opts.raw:
		if err := s.monitor.SetInputRaw(raw); err != nil {
			return a.fail(err)
		}
		if opts.json {
			return a.printJSON(switchOutput{Display: s.display.ID, Value: &raw})
		}
		_, _ = fmt.Fprintf(a.stdout, "Sent input value %d.\n", raw)
		return 0
	case len(opts.args) == 1:
		return a.switchInput(s, opts)
	default:
		return a.showInput(s, opts)
	}
}

func (a app) switchInput(s *session, opts options) int {
	in, err := s.monitor.SetInput(opts.args[0])
	switch {
	case errors.Is(err, monitor.ErrNoProfile):
		_, _ = fmt.Fprintln(a.stderr, "monctl: no profile for this monitor, so its inputs are not known.")
		_, _ = fmt.Fprintln(a.stderr, "Use \"monctl input -raw <value>\" to send a value yourself.")
		return 1
	case errors.Is(err, monitor.ErrUnknownInput):
		_, _ = fmt.Fprintf(a.stderr, "monctl: this monitor has no input %q. Run \"monctl input\" to see the list.\n", opts.args[0])
		return 1
	case err != nil:
		return a.fail(err)
	}
	if opts.json {
		return a.printJSON(switchOutput{Display: s.display.ID, ID: in.ID, Label: in.Label})
	}
	_, _ = fmt.Fprintf(a.stdout, "Switched to %s (%s).\n", in.Label, in.ID)
	return 0
}

func (a app) showInput(s *session, opts options) int {
	src, err := s.monitor.Source()
	if err != nil {
		return a.fail(err)
	}
	inputs := s.monitor.Inputs()

	if opts.json {
		out := inputOutput{
			Display: s.display.ID,
			Current: currentInput{ID: src.Input.ID, Label: src.Input.Label, Value: src.Value},
			Inputs:  make([]inputEntry, 0, len(inputs)),
		}
		for _, in := range inputs {
			out.Inputs = append(out.Inputs, inputEntry{ID: in.ID, Label: in.Label})
		}
		return a.printJSON(out)
	}

	if src.Known {
		_, _ = fmt.Fprintf(a.stdout, "Current: %s (%s)\n", src.Input.Label, src.Input.ID)
	} else {
		_, _ = fmt.Fprintf(a.stdout, "Current: unknown input, value %d\n", src.Value)
	}
	if !s.monitor.HasProfile() {
		_, _ = fmt.Fprintln(a.stdout, "No profile for this monitor. Use \"monctl input -raw <value>\" to switch.")
		return 0
	}
	_, _ = fmt.Fprintln(a.stdout, "Inputs:")
	w := tabwriter.NewWriter(a.stdout, 0, 0, 2, ' ', 0)
	for _, in := range inputs {
		_, _ = fmt.Fprintf(w, "  %s\t%s\n", in.ID, in.Label)
	}
	_ = w.Flush()
	return 0
}

// parseRaw reads decimal, or hex with a 0x prefix. A leading zero is not octal.
func parseRaw(s string) (uint64, error) {
	if hex, ok := strings.CutPrefix(strings.ToLower(s), "0x"); ok {
		return strconv.ParseUint(hex, 16, 8)
	}
	return strconv.ParseUint(s, 10, 8)
}
